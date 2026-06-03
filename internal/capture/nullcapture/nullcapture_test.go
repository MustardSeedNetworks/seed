package nullcapture_test

import (
	"errors"
	"testing"

	"github.com/krisarmstrong/seed/internal/capture"
	"github.com/krisarmstrong/seed/internal/capture/nullcapture"
)

// TestOpenLiveReturnsUnavailable verifies the stub never opens a handle and
// reports ErrUnavailable so callers degrade gracefully on CGO_ENABLED=0 builds.
func TestOpenLiveReturnsUnavailable(t *testing.T) {
	t.Parallel()

	var opener capture.Opener = nullcapture.New()

	handle, err := opener.OpenLive("eth0", 65535, true, capture.BlockForever)
	if handle != nil {
		t.Errorf("OpenLive returned a non-nil handle: %v", handle)
	}
	if !errors.Is(err, nullcapture.ErrUnavailable) {
		t.Errorf("OpenLive error = %v, want ErrUnavailable", err)
	}
}
