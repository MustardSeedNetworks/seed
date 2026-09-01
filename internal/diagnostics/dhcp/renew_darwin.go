//go:build darwin

package dhcp

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// renewSupportedPlatform reports macOS support. `ipconfig set <if> DHCP`
// restarts the DHCP client on that interface, which is the documented way to
// force a renewal.
func renewSupportedPlatform() bool { return true }

// renewLeasePlatform forces a renewal via ipconfig.
//
// The interface name is required here, unlike Windows: `ipconfig set` takes no
// wildcard, and renewing "every interface" is not something to infer.
func renewLeasePlatform(ctx context.Context, interfaceName string) error {
	if interfaceName == "" {
		return fmt.Errorf("%w: an interface name is required on macOS", ErrRenewUnsupported)
	}

	out, err := exec.CommandContext(ctx, "ipconfig", "set", interfaceName, "DHCP").CombinedOutput()
	if err != nil {
		return fmt.Errorf("ipconfig set %s DHCP: %w: %s",
			interfaceName, err, strings.TrimSpace(string(out)))
	}

	return nil
}
