package wmi_test

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/wmi"
)

const (
	testUser   = `LAB\inventory`
	testPass   = "correct horse"
	testDomain = "LAB"
	testName   = "inventory"
)

// winrmHost stands in for a WinRM HTTPS listener. It asks for Negotiate
// (and offers Basic, as a listener with Basic enabled does), runs the NTLM
// exchange per connection, and answers WS-Management requests from the
// recorded responses in testdata.
//
// It checks the AUTHENTICATE message's account and that its NT response has
// the NTLMv2 shape. The proof inside it is HMAC-MD5 over the password hash;
// computing that is go-ntlmssp's job and is not re-implemented here, so a
// wrong password is modelled by a host that refuses the account.
type winrmHost struct {
	srv *httptest.Server

	refuse bool
	// malformedDisks answers the disk query with a size that is not a number.
	malformedDisks bool

	mu        sync.Mutex
	authed    map[string]bool
	requests  int
	basicSeen bool
	bodies    []string
}

func startHost(t *testing.T, refuse bool) *winrmHost {
	t.Helper()
	h := &winrmHost{refuse: refuse, authed: map[string]bool{}}
	h.srv = httptest.NewTLSServer(http.HandlerFunc(h.serve))
	t.Cleanup(h.srv.Close)
	return h
}

func (h *winrmHost) endpoint(pin string) wmi.Endpoint {
	host, port, _ := net.SplitHostPort(h.srv.Listener.Addr().String())
	p, _ := strconv.Atoi(port)
	return wmi.Endpoint{Host: host, Port: p, User: testUser, Password: testPass, PinnedCert: pin}
}

func (h *winrmHost) fingerprint() string { return wmi.Fingerprint(h.srv.Certificate().Raw) }

func (h *winrmHost) serve(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.requests++
	body, _ := io.ReadAll(r.Body)

	auth := r.Header.Get("Authorization")
	switch {
	case strings.HasPrefix(auth, "Basic "):
		h.basicSeen = true
		challenge(w, "")
		return
	case strings.HasPrefix(auth, "Negotiate "):
		msg, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Negotiate "))
		if err != nil || len(msg) < 12 {
			challenge(w, "")
			return
		}
		switch binary.LittleEndian.Uint32(msg[8:12]) {
		case 1:
			challenge(w, base64.StdEncoding.EncodeToString(challengeMessage()))
			return
		case 3:
			if h.refuse || !acceptable(msg) {
				challenge(w, "")
				return
			}
			h.authed[r.RemoteAddr] = true
		}
	}
	if !h.authed[r.RemoteAddr] {
		challenge(w, "")
		return
	}

	h.bodies = append(h.bodies, string(body))
	fixture, status := h.route(string(body))
	w.Header().Set("Content-Type", "application/soap+xml;charset=UTF-8")
	w.WriteHeader(status)
	data, _ := os.ReadFile("testdata/" + fixture)
	_, _ = w.Write(data)
}

func challenge(w http.ResponseWriter, token string) {
	if token == "" {
		w.Header().Add("WWW-Authenticate", "Negotiate")
		w.Header().Add("WWW-Authenticate", `Basic realm="WSMAN"`)
	} else {
		w.Header().Set("WWW-Authenticate", "Negotiate "+token)
	}
	w.WriteHeader(http.StatusUnauthorized)
}

func (h *winrmHost) route(body string) (string, int) {
	switch {
	case strings.Contains(body, "FROM Win32_LogicalDisk") && h.malformedDisks:
		return "enumerate_disk_malformed.xml", http.StatusOK
	case strings.Contains(body, "FROM Win32_LogicalDisk"):
		return "enumerate_disk.xml", http.StatusOK
	case strings.Contains(body, "FROM Win32_ComputerSystem"):
		return "enumerate_system.xml", http.StatusOK
	case strings.Contains(body, "FROM Win32_BIOS"):
		return "enumerate_bios.xml", http.StatusOK
	case strings.Contains(body, "FROM Win32_Processor"):
		return "enumerate_processor.xml", http.StatusOK
	case strings.Contains(body, "enumeration/Pull<"):
		return "pull_software.xml", http.StatusOK
	case strings.Contains(body, "FROM Win32_OperatingSystem"):
		return "enumerate_os.xml", http.StatusOK
	case strings.Contains(body, "FROM Win32_InstalledWin32Program"):
		return "enumerate_software.xml", http.StatusOK
	case strings.Contains(body, "FROM Win32_NetworkAdapterConfiguration"):
		return "enumerate_nic.xml", http.StatusOK
	}
	return "fault_invalid_query.xml", http.StatusInternalServerError
}

// challengeMessage is an NTLM CHALLENGE (MS-NLMP 2.2.1.2): Unicode, NTLM,
// extended session security and target info, with a target info list holding
// only MsvAvEOL.
func challengeMessage() []byte {
	const (
		flags      = 0x00000001 | 0x00000200 | 0x00080000 | 0x00800000
		headerSize = 48
	)
	msg := make([]byte, headerSize+4)
	copy(msg, "NTLMSSP\x00")
	binary.LittleEndian.PutUint32(msg[8:], 2)
	binary.LittleEndian.PutUint32(msg[16:], headerSize) // empty TargetName
	binary.LittleEndian.PutUint32(msg[20:], flags)
	copy(msg[24:32], "8bytes!!") // server challenge
	binary.LittleEndian.PutUint16(msg[40:], 4)
	binary.LittleEndian.PutUint16(msg[42:], 4)
	binary.LittleEndian.PutUint32(msg[44:], headerSize)
	return msg
}

// acceptable reads an AUTHENTICATE message (MS-NLMP 2.2.1.3) and accepts the
// test account with an NTLMv2 response: longer than the 24 bytes of NTLMv1.
func acceptable(msg []byte) bool {
	field := func(off int) []byte {
		n := int(binary.LittleEndian.Uint16(msg[off:]))
		at := int(binary.LittleEndian.Uint32(msg[off+4:]))
		if at+n > len(msg) {
			return nil
		}
		return msg[at : at+n]
	}
	return len(msg) >= 64 &&
		len(field(20)) > 24 &&
		decodeUTF16(field(28)) == testDomain &&
		decodeUTF16(field(36)) == testName
}

func decodeUTF16(b []byte) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[2*i:])
	}
	return string(utf16.Decode(u))
}

func TestQueryReadsInstancesAndProperties(t *testing.T) {
	h := startHost(t, false)
	c := wmi.NewClient(h.endpoint(h.fingerprint()))
	defer c.Close()

	got, err := c.Query(t.Context(),
		"SELECT Caption, Version, BuildNumber, OSArchitecture, InstallDate, LastBootUpTime FROM Win32_OperatingSystem")
	require.NoError(t, err)
	require.Len(t, got, 1)
	os := got[0]
	require.Equal(t, "Microsoft Windows 11 Pro", os.Value("Caption"))
	require.Equal(t, "26100", os.Value("BuildNumber"))
	require.Equal(t, "2025-03-01T10:15:30-08:00", os.Value("InstallDate"), "a cim:Datetime is unwrapped")
	_, present := os["OtherTypeDescription"]
	require.False(t, present, "an xsi:nil property is absent")

	nics, err := c.Query(t.Context(),
		"SELECT Description, MACAddress, IPAddress FROM Win32_NetworkAdapterConfiguration WHERE IPEnabled = TRUE")
	require.NoError(t, err)
	require.Len(t, nics, 1)
	require.Equal(t, []string{"10.44.30.70", "fe80::1c2b:3a4d:5e6f:7081"}, nics[0]["IPAddress"])

	require.False(t, h.basicSeen, "Basic credentials were offered to the host")
}

func TestQueryPullsUntilEndOfSequence(t *testing.T) {
	h := startHost(t, false)
	c := wmi.NewClient(h.endpoint(h.fingerprint()))
	defer c.Close()

	got, err := c.Query(t.Context(), "SELECT Name, Version, Vendor FROM Win32_InstalledWin32Program")
	require.NoError(t, err)
	names := make([]string, 0, len(got))
	for _, inst := range got {
		names = append(names, inst.Value("Name"))
	}
	require.Equal(t, []string{
		"Microsoft 365 Apps for enterprise - en-us",
		"Microsoft OneDrive",
		"Microsoft Visio - en-us",
	}, names)

	require.Len(t, h.bodies, 2)
	require.Contains(t, h.bodies[0], "<w:OptimizeEnumeration/>")
	require.Contains(t, h.bodies[1],
		"<n:EnumerationContext>uuid:9A8B7C6D-5E4F-4A3B-8C2D-1E0F9A8B7C6D</n:EnumerationContext>")
}

func TestQueryEscapesTheFilter(t *testing.T) {
	h := startHost(t, false)
	c := wmi.NewClient(h.endpoint(h.fingerprint()))
	defer c.Close()

	_, err := c.Query(t.Context(), "SELECT Name FROM Win32_Service WHERE Name = '<a&b>'")
	require.ErrorIs(t, err, wmi.ErrFault)
	require.Contains(t, h.bodies[0], "Name = &#39;&lt;a&amp;b&gt;&#39;")
}

func TestQueryReportsFault(t *testing.T) {
	h := startHost(t, false)
	c := wmi.NewClient(h.endpoint(h.fingerprint()))
	defer c.Close()

	_, err := c.Query(t.Context(), "SELECT Name FROM Win32_NoSuchClass")
	require.ErrorIs(t, err, wmi.ErrFault)
	require.ErrorContains(t, err, "2150859002")
	require.ErrorContains(t, err, "could not process the filter")
}

func TestQueryRefusesCertificates(t *testing.T) {
	h := startHost(t, false)

	tests := []struct {
		name  string
		pin   string
		check func(t *testing.T, err error)
	}{
		{
			name: "unpinned",
			pin:  "",
			check: func(t *testing.T, err error) {
				t.Helper()
				var unconfirmed *wmi.CertUnconfirmedError
				require.ErrorAs(t, err, &unconfirmed)
				require.Equal(t, h.fingerprint(), unconfirmed.Fingerprint)
			},
		},
		{
			name: "mismatch",
			pin:  strings.Repeat("AB", 32),
			check: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, wmi.ErrCertMismatch)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := wmi.NewClient(h.endpoint(tt.pin))
			defer c.Close()
			_, err := c.Query(t.Context(), "SELECT Caption FROM Win32_OperatingSystem")
			tt.check(t, err)
		})
	}
	require.Zero(t, h.requests, "a request reached a host whose certificate is not the pinned one")
}

func TestQueryReportsRefusedCredential(t *testing.T) {
	h := startHost(t, true)
	c := wmi.NewClient(h.endpoint(h.fingerprint()))
	defer c.Close()

	_, err := c.Query(t.Context(), "SELECT Caption FROM Win32_OperatingSystem")
	require.ErrorIs(t, err, wmi.ErrAuth)
	require.False(t, h.basicSeen, "Basic credentials were offered after NTLM was refused")
}

func TestQueryReportsUnreachable(t *testing.T) {
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().(*net.TCPAddr)
	require.NoError(t, ln.Close())

	c := wmi.NewClient(wmi.Endpoint{
		Host: "127.0.0.1", Port: addr.Port, User: testUser, Password: testPass,
		PinnedCert: strings.Repeat("AB", 32),
	})
	defer c.Close()
	_, err = c.Query(t.Context(), "SELECT Caption FROM Win32_OperatingSystem")
	require.ErrorIs(t, err, wmi.ErrUnreachable)
	require.NotErrorIs(t, err, wmi.ErrCertMismatch)
}
