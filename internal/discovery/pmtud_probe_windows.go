//go:build windows

package discovery

// Windows sets the don't-fragment bit with IP_DONTFRAGMENT, but reaching it
// needs a raw ICMP socket, and Windows restricts those in ways the Unix
// implementation does not have to account for -- the Windows tracer already
// takes a different route for the same reason.
//
// Rather than ship a socket path that has not been run on the platform, path
// MTU discovery reports itself unsupported here and the capability matrix says
// so, which is the machinery #750 exists for. Tracked as a platform gap.

import (
	"context"
	"time"
)

// MeasurePathMTU reports that this platform cannot probe path MTU.
func MeasurePathMTU(_ context.Context, _ string, _ time.Duration) (*PMTUDResult, error) {
	return nil, ErrPMTUDUnsupported
}
