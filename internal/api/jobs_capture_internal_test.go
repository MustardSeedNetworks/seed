package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/capture"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/packetcapture"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// captureFrames are what the fake interface carries before it goes quiet.
func captureFrames() [][]byte {
	return [][]byte{bytes.Repeat([]byte{0xaa}, 60), bytes.Repeat([]byte{0xbb}, 1514)}
}

// quietHandle delivers captureFrames, then times out until closed, as a live
// handle opened with a read timeout does on an idle link.
type quietHandle struct {
	mu     sync.Mutex
	frames [][]byte
}

func (h *quietHandle) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.frames) == 0 {
		time.Sleep(time.Millisecond)
		return nil, gopacket.CaptureInfo{}, capture.ErrTimeout
	}
	data := h.frames[0]
	h.frames = h.frames[1:]
	return data, gopacket.CaptureInfo{Timestamp: time.Now(), CaptureLength: len(data), Length: len(data)}, nil
}

func (*quietHandle) SetBPFFilter(string) error    { return nil }
func (*quietHandle) LinkType() layers.LinkType    { return layers.LinkTypeEthernet }
func (*quietHandle) WritePacketData([]byte) error { return nil }
func (*quietHandle) Close()                       {}

type quietOpener struct{ iface string }

func (o *quietOpener) OpenLive(iface string, _ int32, _ bool, _ time.Duration) (capture.Handle, error) {
	o.iface = iface
	return &quietHandle{frames: captureFrames()}, nil
}

// registerTestCaptureKind registers the kind over a fake interface and a
// store in a temp directory, the way registerDefaultPacketCaptureKind does
// over the capture port and the data directory. It returns the directory.
func registerTestCaptureKind(t *testing.T, srv *Server, opener *quietOpener) string {
	t.Helper()
	dir := t.TempDir()
	srv.captures = packetcapture.NewStore(dir)
	srv.registerPacketCaptureKind(func(
		ctx context.Context,
		req packetcapture.Request,
		report func(float64),
	) (*packetcapture.Result, error) {
		return packetcapture.Run(ctx, opener, srv.captures, req, report)
	})
	return dir
}

// withDefaultInterface gives srv a config whose default interface is iface.
func withDefaultInterface(srv *Server, iface string) {
	srv.config = config.DefaultConfig()
	srv.config.Interface.Default = iface
}

func download(t *testing.T, srv *Server, id string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, capturesPathPrefix+id, http.NoBody)
	w := httptest.NewRecorder()
	srv.handleCaptureDownload(w, req)
	return w
}

// A capture is started, read while running, stopped and downloaded through
// the runner's own lifecycle: the stop is DELETE /jobs/{id}, and the stopped
// job succeeds with the frames it recorded.
func TestPacketCaptureStopKeepsTheFileAndDownloadsIt(t *testing.T) {
	t.Parallel()
	srv, runner := newJobsTestServer(t, jobs.Config{})
	withDefaultInterface(srv, "eth7")
	opener := &quietOpener{}
	dir := registerTestCaptureKind(t, srv, opener)

	id, err := runner.Submit(packetCaptureJobKind, json.RawMessage(`{"durationSeconds":3600}`))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// While it runs the file exists but is incomplete, so it is refused.
	var running string
	waitFor(t, "capture file", func() bool {
		matches, _ := filepath.Glob(filepath.Join(dir, "*.pcap"))
		if len(matches) == 1 {
			running = strings.TrimSuffix(filepath.Base(matches[0]), ".pcap")
		}
		return running != ""
	})
	if w := download(t, srv, running); w.Code != http.StatusConflict {
		t.Fatalf("download while running = %d, want 409", w.Code)
	}

	if cancelErr := runner.Cancel(id); cancelErr != nil {
		t.Fatalf("Cancel: %v", cancelErr)
	}
	j := waitForState(t, runner, id, jobs.StateSucceeded)
	res, ok := j.Result.(*packetcapture.Result)
	if !ok {
		t.Fatalf("Result = %T, want *packetcapture.Result", j.Result)
	}
	if res.StopReason != packetcapture.StopStopped || res.Packets != len(captureFrames()) || res.ID != running {
		t.Fatalf("Result = %+v, want stopped with %d packets in %s", res, len(captureFrames()), running)
	}
	if opener.iface != "eth7" {
		t.Fatalf("captured on %q, want the configured default interface", opener.iface)
	}

	w := download(t, srv, res.ID)
	if w.Code != http.StatusOK {
		t.Fatalf("download = %d, want 200: %s", w.Code, w.Body)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/vnd.tcpdump.pcap" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if cd := w.Header().Get("Content-Disposition"); cd != `attachment; filename="seed-capture-`+res.ID+`.pcap"` {
		t.Fatalf("Content-Disposition = %q", cd)
	}
	if int64(w.Body.Len()) != res.Bytes {
		t.Fatalf("served %d bytes, result says %d", w.Body.Len(), res.Bytes)
	}
	r, err := pcapgo.NewReader(w.Body)
	if err != nil {
		t.Fatalf("served file is not a pcap: %v", err)
	}
	for i, want := range captureFrames() {
		got, _, readErr := r.ReadPacketData()
		if readErr != nil {
			t.Fatalf("frame %d: %v", i, readErr)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("frame %d differs", i)
		}
	}
	if _, _, tailErr := r.ReadPacketData(); !errors.Is(tailErr, io.EOF) {
		t.Fatalf("want exactly %d frames, next read err = %v", len(captureFrames()), tailErr)
	}
}

func TestPacketCaptureKindRejectsBadParams(t *testing.T) {
	t.Parallel()
	for name, params := range map[string]string{
		"not json":      `{`,
		"unknown field": `{"interface":"eth0","snaplen":96}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv, runner := newJobsTestServer(t, jobs.Config{})
			registerTestCaptureKind(t, srv, &quietOpener{})
			id, err := runner.Submit(packetCaptureJobKind, json.RawMessage(params))
			if err != nil {
				t.Fatalf("Submit: %v", err)
			}
			if j := waitForState(t, runner, id, jobs.StateFailed); !strings.Contains(j.Err, "packet-capture") {
				t.Fatalf("Err = %q, want it to name the kind", j.Err)
			}
		})
	}
}

// The production registration, not a test double: a request the package
// refuses before opening any handle proves the kind reaches packetcapture.Run.
func TestServerRegistersThePacketCaptureKind(t *testing.T) {
	t.Parallel()
	srv, runner := newJobsTestServer(t, jobs.Config{})
	withDefaultInterface(srv, "eth0")
	srv.registerJobKinds()
	if srv.captures == nil {
		t.Fatal("registerJobKinds left the download route without a store")
	}
	id, err := runner.Submit(packetCaptureJobKind, json.RawMessage(`{"durationSeconds":3601}`))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if j := waitForState(t, runner, id, jobs.StateFailed); !strings.Contains(j.Err, "durationSeconds") {
		t.Fatalf("Err = %q, want the duration refusal", j.Err)
	}
}

func TestCaptureDownloadRefusals(t *testing.T) {
	t.Parallel()
	srv, _ := usersTestSetup(t)
	srv.captures = packetcapture.NewStore(t.TempDir())

	tests := []struct {
		name  string
		id    string
		scope string
		want  int
	}{
		{name: "unknown id", id: "AAAAAAAAAAAAAAAAAAAAAAAAAA", want: http.StatusNotFound},
		{name: "traversal", id: "..%2F..%2Fetc%2Fpasswd", want: http.StatusNotFound},
		{name: "empty", id: "", want: http.StatusNotFound},
		{name: "viewer", id: "AAAAAAAAAAAAAAAAAAAAAAAAAA", scope: database.RoleViewer, want: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := newAuthedRequest(http.MethodGet, capturesPathPrefix+tt.id, nil, "admin")
			if tt.scope != "" {
				req = req.WithContext(auth.WithTokenScope(req.Context(), tt.scope))
			}
			w := httptest.NewRecorder()
			srv.handleCaptureDownload(w, req)
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d: %s", w.Code, tt.want, w.Body)
			}
		})
	}
}
