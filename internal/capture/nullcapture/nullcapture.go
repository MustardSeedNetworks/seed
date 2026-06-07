// Package nullcapture is the CGO-free capture.Opener used when live packet
// capture is not compiled in: CGO_ENABLED=0 builds (e.g. Windows, where capture
// is a no-op) and pure-Go test runs that must not link libpcap.
//
// OpenLive always fails with ErrUnavailable so callers degrade gracefully
// ("capture unavailable") instead of panicking or linking libpcap.
// See docs/architecture/CGO_BUILD_STRATEGY.md.
package nullcapture

import (
	"errors"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/capture"
)

// ErrUnavailable is returned by OpenLive when the binary was built without live
// capture support (CGO/libpcap).
var ErrUnavailable = errors.New(
	"capture: live packet capture unavailable (built without CGO/libpcap)",
)

// Compile-time guarantee: this stub implements the port.
var _ capture.Opener = Opener{}

// Opener is the no-op capture.Opener.
type Opener struct{}

// New returns a no-op capture.Opener.
func New() Opener { return Opener{} }

// OpenLive always returns ErrUnavailable.
func (Opener) OpenLive(_ string, _ int32, _ bool, _ time.Duration) (capture.Handle, error) {
	return nil, ErrUnavailable
}
