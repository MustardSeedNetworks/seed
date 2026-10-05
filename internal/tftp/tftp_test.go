package tftp_test

import (
	"bytes"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	pintftp "github.com/pin/tftp/v3"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/tftp"
)

type recorder struct {
	mu        sync.Mutex
	transfers []tftp.Transfer
	stops     []tftp.StopReason
}

func (r *recorder) transfer(t tftp.Transfer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transfers = append(r.transfers, t)
}

func (r *recorder) stop(_ tftp.Status, reason tftp.StopReason) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stops = append(r.stops, reason)
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.transfers)
}

// nth waits for the transfer recorded at index n. The handler records after
// the client has its last packet, so the client returning is not enough.
func (r *recorder) nth(t *testing.T, n int) tftp.Transfer {
	t.Helper()
	var got tftp.Transfer
	require.Eventually(t, func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		if len(r.transfers) <= n {
			return false
		}
		got = r.transfers[n]
		return true
	}, 5*time.Second, 10*time.Millisecond)
	return got
}

// loopback names the host's loopback interface: lo on Linux, lo0 on macOS.
func loopback(t *testing.T) string {
	t.Helper()
	ifaces, err := net.Interfaces()
	require.NoError(t, err)
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback == 0 {
			continue
		}
		addrs, addrErr := iface.Addrs()
		require.NoError(t, addrErr)
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.To4() != nil {
				return iface.Name
			}
		}
	}
	t.Skip("no IPv4 loopback interface")
	return ""
}

// freePort finds a UDP port to stand in for 69, which tests cannot bind.
func freePort(t *testing.T) int {
	t.Helper()
	c, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err)
	port := c.LocalAddr().(*net.UDPAddr).Port
	require.NoError(t, c.Close())
	return port
}

type fixture struct {
	mgr    *tftp.Manager
	rec    *recorder
	dir    string
	client *pintftp.Client
}

func start(t *testing.T, allowUpload bool, idle time.Duration) fixture {
	t.Helper()
	parent := t.TempDir()
	dir := filepath.Join(parent, "tftp")
	rec := &recorder{}
	mgr := tftp.NewManager(tftp.Config{
		Dir:         dir,
		IdleTimeout: idle,
		Port:        freePort(t),
		OnTransfer:  rec.transfer,
		OnStop:      rec.stop,
		Logger:      slog.New(slog.DiscardHandler),
	})
	st, err := mgr.Start(tftp.Options{Interface: loopback(t), AllowUpload: allowUpload})
	require.NoError(t, err)
	t.Cleanup(func() { _ = mgr.Stop() })
	require.True(t, st.Running)
	require.Equal(t, dir, st.Directory)

	// A file beside the served directory, for the escape cases.
	require.NoError(t, os.WriteFile(filepath.Join(parent, "secret.txt"), []byte("outside"), 0o600))

	client, err := pintftp.NewClient(net.JoinHostPort(st.Address, strconv.Itoa(st.Port)))
	require.NoError(t, err)
	client.SetTimeout(time.Second)
	client.SetRetries(1)
	return fixture{mgr: mgr, rec: rec, dir: dir, client: client}
}

func (f fixture) get(name string) ([]byte, error) {
	wt, err := f.client.Receive(name, "octet")
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	_, err = wt.WriteTo(&buf)
	return buf.Bytes(), err
}

func (f fixture) put(name string, body []byte) error {
	rf, err := f.client.Send(name, "octet")
	if err != nil {
		return err
	}
	_, err = rf.ReadFrom(bytes.NewReader(body))
	return err
}

func TestDownload(t *testing.T) {
	f := start(t, false, time.Minute)
	image := bytes.Repeat([]byte("ios-xe image "), 1000)
	require.NoError(t, os.WriteFile(filepath.Join(f.dir, "image.bin"), image, 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(f.dir, "sub"), 0o750))

	tests := []struct {
		name    string
		request string
		want    []byte
		wantErr error
	}{
		{name: "served file", request: "image.bin", want: image},
		{name: "leading slash means the served directory", request: "/image.bin", want: image},
		{name: "missing file", request: "nope.bin", wantErr: tftp.ErrNotFound},
		{name: "parent escape", request: "../secret.txt", wantErr: tftp.ErrNotFound},
		{name: "escape behind a slash", request: "/../secret.txt", wantErr: tftp.ErrNotFound},
		{name: "directory", request: "sub", wantErr: tftp.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := f.rec.count()
			got, err := f.get(tt.request)
			rec := f.rec.nth(t, n)
			require.Equal(t, tftp.Download, rec.Direction)
			require.Equal(t, tt.request, rec.Filename)
			require.Equal(t, "127.0.0.1", rec.Remote)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, rec.Err, tt.wantErr)
				require.Contains(t, err.Error(), tt.wantErr.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NoError(t, rec.Err)
			require.Equal(t, int64(len(image)), rec.Bytes)
		})
	}
}

func TestUploadRefusedUnlessAllowed(t *testing.T) {
	f := start(t, false, time.Minute)

	err := f.put("running-config", []byte("hostname sw1\n"))
	require.Error(t, err)
	require.ErrorIs(t, f.rec.nth(t, 0).Err, tftp.ErrUploadsDisabled)
	require.NoFileExists(t, filepath.Join(f.dir, "running-config"))
}

func TestUploadWhenAllowed(t *testing.T) {
	f := start(t, true, time.Minute)
	config := []byte("hostname sw1\ninterface Gi1/0/1\n")

	require.NoError(t, f.put("sw1-confg", config))
	rec := f.rec.nth(t, 0)
	require.NoError(t, rec.Err)
	require.Equal(t, tftp.Upload, rec.Direction)
	got, err := os.ReadFile(filepath.Join(f.dir, "sw1-confg"))
	require.NoError(t, err)
	require.Equal(t, config, got)

	// A second upload of the same name never replaces the first.
	require.Error(t, f.put("sw1-confg", []byte("hostname evil\n")))
	require.ErrorIs(t, f.rec.nth(t, 1).Err, tftp.ErrExists)
	got, err = os.ReadFile(filepath.Join(f.dir, "sw1-confg"))
	require.NoError(t, err)
	require.Equal(t, config, got)

	// Nor does an upload land outside the served directory.
	require.Error(t, f.put("../planted.txt", config))
	require.NoFileExists(t, filepath.Join(filepath.Dir(f.dir), "planted.txt"))
}

func TestStopsWhenIdle(t *testing.T) {
	f := start(t, false, 200*time.Millisecond)

	require.Eventually(t, func() bool { return !f.mgr.Status().Running }, 5*time.Second, 20*time.Millisecond)
	f.rec.mu.Lock()
	require.Equal(t, []tftp.StopReason{tftp.StopIdle}, f.rec.stops)
	f.rec.mu.Unlock()

	_, err := f.get("anything")
	require.Error(t, err, "nothing may answer once the session has stopped")
	require.ErrorIs(t, f.mgr.Stop(), tftp.ErrNotRunning)
}

func TestSessionLifecycle(t *testing.T) {
	f := start(t, false, time.Minute)

	_, err := f.mgr.Start(tftp.Options{Interface: loopback(t)})
	require.ErrorIs(t, err, tftp.ErrRunning)

	require.NoError(t, f.mgr.Stop())
	require.False(t, f.mgr.Status().Running)
	f.rec.mu.Lock()
	require.Equal(t, []tftp.StopReason{tftp.StopOperator}, f.rec.stops)
	f.rec.mu.Unlock()
	require.ErrorIs(t, f.mgr.Stop(), tftp.ErrNotRunning)
}

func TestStartRejectsUnknownInterface(t *testing.T) {
	mgr := tftp.NewManager(tftp.Config{Dir: t.TempDir(), Logger: slog.New(slog.DiscardHandler)})

	_, err := mgr.Start(tftp.Options{Interface: "no-such-if0"})
	require.ErrorIs(t, err, tftp.ErrNoInterface)
	require.False(t, mgr.Status().Running)
}
