package enumerate_test

import (
	"testing"
	"time"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture"
	"github.com/MustardSeedNetworks/seed/internal/capture/capturetest"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
)

// runningCapture is a started protocol capture: how to stop it and how many
// neighbours it has recorded.
type runningCapture struct {
	stop      func()
	neighbors func() int
}

// TestProtocolCaptureStopsOnQuietInterface pins seed#2862. Each capture hears
// one advertisement after a read timeout, then the interface goes quiet. The
// frame proves the reader keeps going past capture.ErrTimeout; Stop proves a
// quiet handle can still be closed. Opened without a read timeout, libpcap on
// Linux keeps the handle inside a read that never returns, and Stop hangs.
func TestProtocolCaptureStopsOnQuietInterface(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		frame []byte
		start func(*testing.T, capture.Opener) runningCapture
	}{
		{
			name:  "LLDP",
			frame: lldpUntagged,
			start: func(t *testing.T, o capture.Opener) runningCapture {
				t.Helper()
				c := enumerate.NewLLDPCapture(o, "eth0")
				if err := c.Start(); err != nil {
					t.Fatalf("Start: %v", err)
				}
				return runningCapture{stop: c.Stop, neighbors: func() int { return len(c.GetNeighbors()) }}
			},
		},
		{
			name:  "CDP",
			frame: cdpTaggedVLAN200,
			start: func(t *testing.T, o capture.Opener) runningCapture {
				t.Helper()
				c := enumerate.NewCDPCapture(o, "eth0")
				if err := c.Start(); err != nil {
					t.Fatalf("Start: %v", err)
				}
				return runningCapture{stop: c.Stop, neighbors: func() int { return len(c.GetNeighbors()) }}
			},
		},
		{
			name:  "EDP",
			frame: edpLLCSNAPFrame,
			start: func(t *testing.T, o capture.Opener) runningCapture {
				t.Helper()
				c := enumerate.NewEDPCapture(o, "eth0")
				if err := c.Start(); err != nil {
					t.Fatalf("Start: %v", err)
				}
				return runningCapture{stop: c.Stop, neighbors: func() int { return len(c.GetNeighbors()) }}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opener := &capturetest.QuietOpener{LinkType: layers.LinkTypeEthernet, Frames: [][]byte{tt.frame}}
			running := tt.start(t, opener)

			deadline := time.Now().Add(5 * time.Second)
			for running.neighbors() == 0 {
				if time.Now().After(deadline) {
					t.Fatal("no neighbour decoded from the frame after the read timeout")
				}
				time.Sleep(10 * time.Millisecond)
			}

			stopped := make(chan struct{})
			go func() {
				running.stop()
				close(stopped)
			}()
			select {
			case <-stopped:
			case <-time.After(time.Second):
				t.Fatal("Stop did not return within one second on a quiet interface")
			}
		})
	}
}
