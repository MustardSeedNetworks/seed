package dhcp

import (
	"context"
	"errors"
	"time"
)

// ErrRenewUnsupported means this platform cannot force a lease renewal.
// Callers should decline the request rather than attempt it and report a
// runtime failure, the way vlan.ErrUnsupported is handled.
var ErrRenewUnsupported = errors.New("dhcp: forced lease renewal is not supported on this platform")

// renewTimeout bounds the renewal. A DHCP exchange that has not completed by
// then is not going to, and the operator is waiting on an HTTP response.
const renewTimeout = 30 * time.Second

// RenewSupported reports whether this platform can force a lease renewal.
//
// Exposed so a caller can refuse up front. An endpoint that offers renewal on
// a platform that cannot do it can only ever fail, and failing at the point of
// the request is more useful than failing at the point of the attempt.
func RenewSupported() bool {
	return renewSupportedPlatform()
}

// RenewLease forces the interface to renew its DHCP lease.
//
// This is a state-changing operation on the host's own networking: a renewal
// can return a different address, and on a misconfigured network it can leave
// the interface without one. Callers must gate it accordingly -- the HTTP
// surface puts it behind operator+ and CSRF.
//
// Returns [ErrRenewUnsupported] where the platform cannot do it.
func RenewLease(ctx context.Context, interfaceName string) error {
	if !RenewSupported() {
		return ErrRenewUnsupported
	}

	ctx, cancel := context.WithTimeout(ctx, renewTimeout)
	defer cancel()

	return renewLeasePlatform(ctx, interfaceName)
}
