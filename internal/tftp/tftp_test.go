package tftp_test

import (
	"bytes"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
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
	inbox  string
	client *pintftp.Client
}

type setup struct {
	allowUpload bool
	idle        time.Duration
	limits      tftp.Limits
	allowFrom   []netip.Prefix
}

func start(t *testing.T, allowUpload bool, idle time.Duration) fixture {
	t.Helper()
	return startWith(t, setup{allowUpload: allowUpload, idle: idle})
}

func startWith(t *testing.T, su setup) fixture {
	t.Helper()
	parent := t.TempDir()
	dir := filepath.Join(parent, "tftp")
	inbox := filepath.Join(parent, "tftp-incoming")
	rec := &recorder{}
	mgr := tftp.NewManager(tftp.Config{
		Dir:         dir,
		UploadDir:   inbox,
		IdleTimeout: su.idle,
		Port:        freePort(t),
		Limits:      su.limits,
		OnTransfer:  rec.transfer,
		OnStop:      rec.stop,
		Logger:      slog.New(slog.DiscardHandler),
	})
	st, err := mgr.Start(tftp.Options{Interface: loopback(t), AllowUpload: su.allowUpload, AllowFrom: su.allowFrom})
	require.NoError(t, err)
	t.Cleanup(func() { _ = mgr.Stop() })
	require.True(t, st.Running)
	require.Equal(t, dir, st.Directory)
	require.Equal(t, inbox, st.UploadDirectory)

	// A file beside the served directory, for the escape cases.
	require.NoError(t, os.WriteFile(filepath.Join(parent, "secret.txt"), []byte("outside"), 0o600))

	client, err := pintftp.NewClient(net.JoinHostPort(st.Address, strconv.Itoa(st.Port)))
	require.NoError(t, err)
	client.SetTimeout(time.Second)
	client.SetRetries(1)
	return fixture{mgr: mgr, rec: rec, dir: dir, inbox: inbox, client: client}
}

func (f fixture) stops() []tftp.StopReason {
	f.rec.mu.Lock()
	defer f.rec.mu.Unlock()
	return append([]tftp.StopReason(nil), f.rec.stops...)
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
	symlinks := runtime.GOOS != "windows" // creating one there needs a privilege CI lacks
	if symlinks {
		require.NoError(t, os.Symlink(filepath.Join(filepath.Dir(f.dir), "secret.txt"), filepath.Join(f.dir, "link")))
	}

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
		{name: "symlink out of the directory", request: "link", wantErr: tftp.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.request == "link" && !symlinks {
				t.Skip("the premise of this case is POSIX symlink creation")
			}
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
	got, err := os.ReadFile(filepath.Join(f.inbox, "sw1-confg"))
	require.NoError(t, err)
	require.Equal(t, config, got)
	entries, err := os.ReadDir(f.inbox)
	require.NoError(t, err)
	require.Len(t, entries, 1, "the partial file must be gone once the upload is published")

	// An upload lands only in the upload directory, which is never served:
	// a device cannot plant a file another device will then fetch.
	require.NoFileExists(t, filepath.Join(f.dir, "sw1-confg"))
	_, err = f.get("sw1-confg")
	require.Error(t, err)
	require.ErrorIs(t, f.rec.nth(t, 1).Err, tftp.ErrNotFound)

	// A second upload of the same name never replaces the first.
	require.Error(t, f.put("sw1-confg", []byte("hostname evil\n")))
	require.ErrorIs(t, f.rec.nth(t, 2).Err, tftp.ErrExists)
	got, err = os.ReadFile(filepath.Join(f.inbox, "sw1-confg"))
	require.NoError(t, err)
	require.Equal(t, config, got)

	// Nor does an upload land outside the upload directory.
	require.Error(t, f.put("../planted.txt", config))
	require.NoFileExists(t, filepath.Join(filepath.Dir(f.inbox), "planted.txt"))
}

func TestUploadQuotas(t *testing.T) {
	tests := []struct {
		name    string
		limits  tftp.Limits
		uploads [][]byte
		wantErr error
	}{
		{
			name:    "upload count",
			limits:  tftp.Limits{MaxUploads: 2},
			uploads: [][]byte{[]byte("one"), []byte("two"), []byte("three")},
			wantErr: tftp.ErrQuota,
		},
		{
			name:    "session bytes",
			limits:  tftp.Limits{UploadQuota: 600},
			uploads: [][]byte{bytes.Repeat([]byte("a"), 400), bytes.Repeat([]byte("b"), 400)},
			wantErr: tftp.ErrQuota,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := startWith(t, setup{allowUpload: true, idle: time.Minute, limits: tt.limits})
			last := len(tt.uploads) - 1
			for i, body := range tt.uploads {
				name := "file" + strconv.Itoa(i)
				err := f.put(name, body)
				rec := f.rec.nth(t, i)
				if i < last {
					require.NoError(t, err)
					require.NoError(t, rec.Err)
					continue
				}
				require.Error(t, err)
				require.ErrorIs(t, rec.Err, tt.wantErr)
				require.NoFileExists(t, filepath.Join(f.inbox, name))
			}
			entries, err := os.ReadDir(f.inbox)
			require.NoError(t, err)
			require.Len(t, entries, last, "a refused upload must leave nothing behind")
		})
	}
}

func TestRefusesSourcesOutsideAllowFrom(t *testing.T) {
	f := startWith(t, setup{
		idle:      time.Minute,
		allowFrom: []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")},
	})
	require.NoError(t, os.WriteFile(filepath.Join(f.dir, "image.bin"), []byte("image"), 0o600))
	require.Equal(t, []string{"192.0.2.0/24"}, f.mgr.Status().AllowFrom)

	_, err := f.get("image.bin")
	require.Error(t, err)
	rec := f.rec.nth(t, 0)
	require.ErrorIs(t, rec.Err, tftp.ErrAccessDenied)
	require.Equal(t, "127.0.0.1", rec.Remote)
}

func TestAnswersTheInterfaceSubnetByDefault(t *testing.T) {
	f := start(t, false, time.Minute)
	for _, p := range f.mgr.Status().AllowFrom {
		if netip.MustParsePrefix(p).Contains(netip.MustParseAddr("127.0.0.1")) {
			return
		}
	}
	t.Fatalf("allowFrom = %v, want the loopback subnet", f.mgr.Status().AllowFrom)
}

func TestRefusesBeyondMaxTransfers(t *testing.T) {
	f := startWith(t, setup{idle: time.Minute, limits: tftp.Limits{MaxTransfers: 1}})
	image := bytes.Repeat([]byte("x"), 10<<10)
	require.NoError(t, os.WriteFile(filepath.Join(f.dir, "image.bin"), image, 0o600))

	// Receive returns on the first block; the server then waits for its ACK,
	// so this transfer is in flight until WriteTo drains it.
	held, err := f.client.Receive("image.bin", "octet")
	require.NoError(t, err)
	require.Equal(t, int64(1), f.mgr.Status().ActiveTransfers)

	_, err = f.get("image.bin")
	require.Error(t, err)
	require.ErrorIs(t, f.rec.nth(t, 0).Err, tftp.ErrBusy)

	var buf bytes.Buffer
	_, err = held.WriteTo(&buf)
	require.NoError(t, err)
	require.Equal(t, image, buf.Bytes())
	require.NoError(t, f.rec.nth(t, 1).Err)
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

// TestRefusedRequestsDoNotHoldTheSession: only a completed transfer is use,
// so a host polling for a missing file cannot keep the port open.
func TestRefusedRequestsDoNotHoldTheSession(t *testing.T) {
	f := start(t, false, 300*time.Millisecond)

	require.Eventually(t, func() bool {
		_, _ = f.get("missing.bin")
		return !f.mgr.Status().Running
	}, 5*time.Second, 20*time.Millisecond)
	require.Equal(t, []tftp.StopReason{tftp.StopIdle}, f.stops())
}

func TestStopsAtMaxLifetime(t *testing.T) {
	f := startWith(t, setup{idle: time.Minute, limits: tftp.Limits{MaxLifetime: 300 * time.Millisecond}})
	require.NoError(t, os.WriteFile(filepath.Join(f.dir, "image.bin"), []byte("image"), 0o600))

	// Completed transfers reset the idle timer, never the lifetime.
	require.Eventually(t, func() bool {
		_, _ = f.get("image.bin")
		return !f.mgr.Status().Running
	}, 5*time.Second, 20*time.Millisecond)
	require.Equal(t, []tftp.StopReason{tftp.StopExpired}, f.stops())
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
