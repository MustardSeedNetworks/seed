package packetcapture

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/capture"
)

// readStep is how far the fake clock moves on every read.
const readStep = 10 * time.Millisecond

var errLinkDown = errors.New("link down")

// fakeHandle replays frames, then times out; every read moves the clock.
type fakeHandle struct {
	clock   *fakeClock
	frames  [][]byte
	readErr error // returned once the frames run out, instead of a timeout
	endless bool  // a frame on every read, however many were scripted
	filter  string
	closed  bool
	// onRead runs before each read, so a test can stop the capture mid-way.
	onRead func(reads int)
	reads  int
}

func (h *fakeHandle) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	h.reads++
	if h.onRead != nil {
		h.onRead(h.reads)
	}
	h.clock.t = h.clock.t.Add(readStep)
	if h.endless {
		data := frame(60)
		return data, gopacket.CaptureInfo{Timestamp: h.clock.t, CaptureLength: len(data), Length: len(data)}, nil
	}
	if len(h.frames) == 0 {
		if h.readErr != nil {
			return nil, gopacket.CaptureInfo{}, h.readErr
		}
		return nil, gopacket.CaptureInfo{}, capture.ErrTimeout
	}
	data := h.frames[0]
	h.frames = h.frames[1:]
	ci := gopacket.CaptureInfo{Timestamp: h.clock.t, CaptureLength: len(data), Length: len(data)}
	return data, ci, nil
}

func (h *fakeHandle) SetBPFFilter(filter string) error {
	if filter == "not a filter" {
		return errors.New("syntax error")
	}
	h.filter = filter
	return nil
}

func (*fakeHandle) LinkType() layers.LinkType    { return layers.LinkTypeEthernet }
func (*fakeHandle) WritePacketData([]byte) error { return nil }
func (h *fakeHandle) Close()                     { h.closed = true }

type fakeOpener struct {
	handle  *fakeHandle
	openErr error
	iface   string
}

func (o *fakeOpener) OpenLive(iface string, _ int32, _ bool, _ time.Duration) (capture.Handle, error) {
	o.iface = iface
	if o.openErr != nil {
		return nil, o.openErr
	}
	return o.handle, nil
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

func frame(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}

func newSession(t *testing.T, h *fakeHandle, maxBytes int64) (session, *fakeOpener) {
	t.Helper()
	o := &fakeOpener{handle: h}
	return session{opener: o, store: NewStore(t.TempDir()), now: h.clock.now, maxBytes: maxBytes}, o
}

// readCapture opens a finished capture and returns its frames.
func readCapture(t *testing.T, store *Store, id string) (layers.LinkType, [][]byte) {
	t.Helper()
	f, err := store.Open(id)
	require.NoError(t, err)
	defer f.Close()
	r, err := pcapgo.NewReader(f)
	require.NoError(t, err)
	var got [][]byte
	for {
		data, _, readErr := r.ReadPacketData()
		if errors.Is(readErr, io.EOF) {
			break
		}
		require.NoError(t, readErr)
		got = append(got, data)
	}
	return r.LinkType(), got
}

func TestRunStopsAtEachBound(t *testing.T) {
	tests := []struct {
		name       string
		frames     [][]byte
		endless    bool // the link never goes quiet
		duration   int
		maxBytes   int64
		stopAfter  int // cancel the context before this read; 0 never
		wantReason StopReason
		wantFrames int
	}{
		{
			name:       "duration",
			frames:     [][]byte{frame(60), frame(1500), frame(64)},
			duration:   1,
			maxBytes:   MaxFileBytes,
			wantReason: StopDuration,
			wantFrames: 3,
		},
		{
			name:     "size ceiling",
			frames:   [][]byte{frame(100), frame(100), frame(100)},
			duration: 1,
			// Room for the header and two records of 16+100, not a third.
			maxBytes:   fileHeaderBytes + 2*(recordHeaderBytes+100) + 50,
			wantReason: StopSize,
			wantFrames: 2,
		},
		{
			name:     "stopped by the operator drains what the kernel buffered",
			frames:   [][]byte{frame(60), frame(61), frame(62), frame(63), frame(64)},
			duration: 60,
			maxBytes: MaxFileBytes,
			// The stop lands during the third read. The two frames after it
			// were already buffered, so they are read before the first quiet
			// read ends the capture.
			stopAfter:  3,
			wantReason: StopStopped,
			wantFrames: 5,
		},
		{
			name:     "a link that never goes quiet stops after the drain window",
			endless:  true,
			duration: 60,
			maxBytes: MaxFileBytes,
			// Three frames up to the stop, then one per read for drainWindow.
			stopAfter:  3,
			wantReason: StopStopped,
			wantFrames: 3 + int(drainWindow/readStep),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			h := &fakeHandle{
				clock:   &fakeClock{t: time.Unix(1_700_000_000, 0)},
				frames:  tt.frames,
				endless: tt.endless,
			}
			if tt.stopAfter > 0 {
				h.onRead = func(reads int) {
					if reads == tt.stopAfter {
						cancel()
					}
				}
			}
			s, o := newSession(t, h, tt.maxBytes)
			req := Request{Interface: "eth1", Filter: "port 443", DurationSeconds: tt.duration}

			res, err := s.run(ctx, req, func(float64) {})
			require.NoError(t, err)

			assert.Equal(t, tt.wantReason, res.StopReason)
			assert.Equal(t, tt.wantFrames, res.Packets)
			assert.Equal(t, "eth1", o.iface)
			assert.Equal(t, "port 443", h.filter)
			assert.True(t, h.closed, "handle must be closed")

			link, got := readCapture(t, s.store, res.ID)
			assert.Equal(t, layers.LinkTypeEthernet, link)
			if tt.endless {
				assert.Len(t, got, tt.wantFrames)
			} else {
				assert.Equal(t, tt.frames[:tt.wantFrames], got)
			}
			info, err := os.Stat(s.store.path(res.ID))
			require.NoError(t, err)
			assert.Equal(t, info.Size(), res.Bytes, "reported bytes must be the file size")
			assert.LessOrEqual(t, res.Bytes, tt.maxBytes)
		})
	}
}

func TestRunDurationEndsOnAQuietInterface(t *testing.T) {
	h := &fakeHandle{clock: &fakeClock{t: time.Unix(1_700_000_000, 0)}}
	s, _ := newSession(t, h, MaxFileBytes)
	var progress []float64

	res, err := s.run(t.Context(), Request{Interface: "eth0", DurationSeconds: 2}, func(p float64) {
		progress = append(progress, p)
	})
	require.NoError(t, err)

	assert.Equal(t, StopDuration, res.StopReason)
	assert.Zero(t, res.Packets)
	assert.Equal(t, int64(2000), res.DurationMs)
	require.NotEmpty(t, progress)
	for _, p := range progress {
		assert.True(t, p > 0 && p < 1, "progress %v outside (0,1)", p)
	}
	_, got := readCapture(t, s.store, res.ID)
	assert.Empty(t, got)
}

func TestRunFailsWithoutLeavingAFile(t *testing.T) {
	tests := []struct {
		name    string
		req     Request
		openErr error
		readErr error
		wantErr error
	}{
		{name: "no interface", req: Request{}, wantErr: ErrInterface},
		{name: "negative duration", req: Request{Interface: "eth0", DurationSeconds: -1}},
		{name: "longer than an hour", req: Request{Interface: "eth0", DurationSeconds: 3601}},
		{
			name:    "interface will not open",
			req:     Request{Interface: "eth9"},
			openErr: capture.ErrTimeout,
			wantErr: capture.ErrTimeout,
		},
		{name: "filter does not compile", req: Request{Interface: "eth0", Filter: "not a filter"}},
		{name: "read fails mid-capture", req: Request{Interface: "eth0"}, readErr: errLinkDown, wantErr: errLinkDown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &fakeHandle{
				clock:   &fakeClock{t: time.Unix(1_700_000_000, 0)},
				frames:  [][]byte{frame(60)},
				readErr: tt.readErr,
			}
			s, o := newSession(t, h, MaxFileBytes)
			o.openErr = tt.openErr

			res, err := s.run(t.Context(), tt.req, func(float64) {})
			require.Error(t, err)
			assert.Nil(t, res)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}

			entries, dirErr := os.ReadDir(s.store.dir)
			if !errors.Is(dirErr, os.ErrNotExist) {
				require.NoError(t, dirErr)
				assert.Empty(t, entries, "a failed capture must leave no file")
			}
			assert.Empty(t, s.store.active)
		})
	}
}
