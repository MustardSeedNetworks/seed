package deviceconfig_test

import (
	"bufio"
	"bytes"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

const (
	base10 = "urn:ietf:params:netconf:base:1.0"
	base11 = "urn:ietf:params:netconf:base:1.1"
	eom    = "]]>]]>"
)

// netconfScript is the device's side of one NETCONF session: it answers the
// client's hello, replies to the get-config and acknowledges close-session,
// recording every message the client sent.
type netconfScript struct {
	capabilities []string
	// helloSuffix is written after the hello's end-of-message marker.
	helloSuffix string
	// reply is the get-config rpc-reply, split over chunks chunks when the
	// session is chunked.
	reply  string
	chunks int
	// rawReply, when set, is written verbatim in place of the framed reply.
	rawReply string

	mu       sync.Mutex
	received []string
	// done closes when the session is over.
	done chan struct{}
}

func (sc *netconfScript) serve(ch ssh.Channel) {
	defer close(sc.done)
	r := bufio.NewReader(ch)
	hello, err := readUntil(r, eom)
	if err != nil {
		return
	}
	sc.record(hello)
	var caps strings.Builder
	for _, c := range sc.capabilities {
		caps.WriteString("<capability>" + c + "</capability>")
	}
	_, _ = fmt.Fprintf(ch, `<?xml version="1.0" encoding="UTF-8"?>`+
		`<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><capabilities>%s</capabilities>`+
		`<session-id>7</session-id></hello>%s%s`, caps.String(), eom, sc.helloSuffix)

	chunked := slices.Contains(sc.capabilities, base11)
	replies := []string{
		sc.reply,
		`<rpc-reply message-id="2" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><ok/></rpc-reply>`,
	}
	for i, reply := range replies {
		end := eom
		if chunked {
			end = "\n##\n"
		}
		msg, readErr := readUntil(r, end)
		if readErr != nil {
			return
		}
		sc.record(msg)
		if i == 0 && sc.rawReply != "" {
			_, _ = ch.Write([]byte(sc.rawReply))
			return
		}
		if !chunked {
			_, _ = ch.Write([]byte(reply + eom))
			continue
		}
		n := max(sc.chunks, 1)
		size := (len(reply) + n - 1) / n
		for part := range slices.Chunk([]byte(reply), size) {
			_, _ = fmt.Fprintf(ch, "\n#%d\n%s", len(part), part)
		}
		_, _ = ch.Write([]byte("\n##\n"))
	}
}

func (sc *netconfScript) record(msg string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.received = append(sc.received, msg)
}

func (sc *netconfScript) messages() []string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return slices.Clone(sc.received)
}

func readUntil(r *bufio.Reader, end string) (string, error) {
	var buf bytes.Buffer
	for !strings.HasSuffix(buf.String(), end) {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		buf.WriteByte(b)
	}
	return buf.String(), nil
}

func dataReply(data string) string {
	return `<rpc-reply message-id="1" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">` +
		`<data>` + data + `</data></rpc-reply>`
}

func fetchNETCONF(t *testing.T, sc *netconfScript) (deviceconfig.FetchResult, error) {
	t.Helper()
	sc.done = make(chan struct{})
	dev := startSSHDevice(t, deviceOpts{password: "s3cret", netconf: sc.serve})
	req := dev.request(t, "s3cret", "", dev.fingerprint)
	req.NETCONF = true
	res, err := deviceconfig.SSHFetcher{}.Fetch(fetchCtx(t), req)
	// The client closes the channel without awaiting the close-session
	// reply; wait for the device to finish recording what it was sent.
	select {
	case <-sc.done:
	case <-time.After(5 * time.Second):
		t.Fatal("the device session did not end")
	}
	return res, err
}

const nativeConfig = `<native xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-native"><hostname>core</hostname></native>`

func TestNETCONFFetchReadsTheRunningDatastore(t *testing.T) {
	t.Parallel()
	for name, sc := range map[string]*netconfScript{
		"base 1.1, reply in three chunks": {
			capabilities: []string{base10, base11}, reply: dataReply(nativeConfig), chunks: 3,
		},
		// A 1.0-only device keeps marker framing; this one also ends its
		// hello with a newline, as some do.
		"base 1.0 only": {
			capabilities: []string{base10}, helloSuffix: "\n", reply: dataReply(nativeConfig),
		},
		"base 1.1, newline after the hello": {
			capabilities: []string{base10, base11}, helloSuffix: "\n", reply: dataReply(nativeConfig),
		},
		"warning alongside the data": {
			capabilities: []string{base10, base11},
			reply: `<rpc-reply message-id="1" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">` +
				`<rpc-error><error-severity>warning</error-severity><error-message>slow</error-message></rpc-error>` +
				`<data>` + nativeConfig + `</data></rpc-reply>`,
		},
	} {
		res, err := fetchNETCONF(t, sc)
		require.NoError(t, err, name)
		require.Equal(t, nativeConfig+"\n", res.Output, name)

		msgs := sc.messages()
		require.Len(t, msgs, 3, name)
		require.Contains(t, msgs[0], base11, "%s: the client offers chunked framing", name)
		require.Contains(t, msgs[1], "<get-config><source><running/></source></get-config>", name)
		require.Contains(t, msgs[2], "<close-session/>", name)
	}
}

func TestNETCONFFetchClassifiesFailures(t *testing.T) {
	t.Parallel()
	both := []string{base10, base11}
	for name, tc := range map[string]struct {
		script *netconfScript
		want   string
	}{
		"rpc-error": {
			script: &netconfScript{capabilities: both, reply: `<rpc-reply message-id="1">` +
				`<rpc-error><error-severity>error</error-severity><error-tag>access-denied</error-tag>` +
				`<error-message>permission denied</error-message></rpc-error></rpc-reply>`},
			want: "permission denied",
		},
		"rpc-error with only a tag": {
			script: &netconfScript{capabilities: both, reply: `<rpc-reply message-id="1">` +
				`<rpc-error><error-severity>error</error-severity><error-tag>operation-not-supported</error-tag>` +
				`</rpc-error></rpc-reply>`},
			want: "operation-not-supported",
		},
		"reply without data": {
			script: &netconfScript{capabilities: both, reply: `<rpc-reply message-id="1"><ok/></rpc-reply>`},
			want:   "carries no data",
		},
		"no base capability": {
			script: &netconfScript{capabilities: []string{"urn:example:other"}},
			want:   "no NETCONF base capability",
		},
		"zero-length chunk": {
			script: &netconfScript{capabilities: both, rawReply: "\n#0\n"},
			want:   "chunk size",
		},
		"chunk larger than a configuration": {
			script: &netconfScript{capabilities: both, rawReply: "\n#99999999\n"},
			want:   "exceeds",
		},
		"junk instead of a chunk": {
			script: &netconfScript{capabilities: both, rawReply: "<rpc-reply/>]]>]]>"},
			want:   "before a chunk",
		},
	} {
		_, err := fetchNETCONF(t, tc.script)
		require.ErrorIs(t, err, deviceconfig.ErrCommand, name)
		require.ErrorContains(t, err, tc.want, name)
	}
}

func TestNETCONFFetchReportsADeviceWithoutNETCONF(t *testing.T) {
	t.Parallel()
	dev := startSSHDevice(t, deviceOpts{password: "s3cret"})
	req := dev.request(t, "s3cret", "", dev.fingerprint)
	req.NETCONF = true
	_, err := deviceconfig.SSHFetcher{}.Fetch(fetchCtx(t), req)
	require.ErrorIs(t, err, deviceconfig.ErrCommand)
	require.ErrorContains(t, err, "netconf subsystem")
}

func TestNETCONFFetchRefusesAnythingButAHello(t *testing.T) {
	t.Parallel()
	dev := startSSHDevice(t, deviceOpts{password: "s3cret", netconf: func(ch ssh.Channel) {
		_, _ = readUntil(bufio.NewReader(ch), eom)
		_, _ = ch.Write([]byte("% Invalid input detected" + eom))
	}})
	req := dev.request(t, "s3cret", "", dev.fingerprint)
	req.NETCONF = true
	_, err := deviceconfig.SSHFetcher{}.Fetch(fetchCtx(t), req)
	require.ErrorIs(t, err, deviceconfig.ErrCommand)
	require.ErrorContains(t, err, "hello is not NETCONF")
}
