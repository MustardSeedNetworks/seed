//go:build windows

package dhcp

import "context"

// renewSupportedPlatform reports Windows support. `ipconfig /renew` is
// available on every supported Windows version.
func renewSupportedPlatform() bool { return true }

// renewLeasePlatform forces a renewal via ipconfig.
//
// RenewDHCP already existed here and had no callers at all (#2275); this is
// the caller it was missing. It applies its own timeout, so ctx bounds the
// call only insofar as the caller cancels.
func renewLeasePlatform(_ context.Context, interfaceName string) error {
	return RenewDHCP(interfaceName)
}
