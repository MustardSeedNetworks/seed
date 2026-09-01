//go:build linux

package dhcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// renewCommand is one way of forcing a renewal, and the check that says
// whether this machine can use it.
type renewCommand struct {
	// name is the binary, looked up on PATH.
	name string
	// available reports whether this machine is actually driven by it.
	// Presence of the binary is not enough: dhcpcd ships on hosts whose
	// interfaces are managed by systemd-networkd, and telling dhcpcd to renew
	// an interface it does not manage does nothing useful.
	available func() bool
	// args builds the argv for the interface.
	args func(iface string) []string
}

// renewCommands are tried in order.
//
// There is no single answer on Linux. Ubuntu 26.04 ships neither dhclient nor
// NetworkManager by default -- interfaces are managed by systemd-networkd, and
// `networkctl renew` is the supported way to ask for a new lease. Older hosts
// and containers may have any of the others, so each is detected rather than
// assumed.
var renewCommands = []renewCommand{
	{
		name:      "networkctl",
		available: systemdNetworkdActive,
		args:      func(iface string) []string { return []string{"renew", iface} },
	},
	{
		name:      "nmcli",
		available: func() bool { return true },
		args:      func(iface string) []string { return []string{"device", "reapply", iface} },
	},
	{
		name:      "dhcpcd",
		available: func() bool { return true },
		args:      func(iface string) []string { return []string{"-n", iface} },
	},
	{
		name:      "dhclient",
		available: func() bool { return true },
		args:      func(iface string) []string { return []string{"-v", iface} },
	},
}

// systemdNetworkdActive reports whether systemd-networkd is managing this
// host's interfaces. Its runtime directory exists only while it is running.
func systemdNetworkdActive() bool {
	_, err := os.Stat("/run/systemd/netif")

	return err == nil
}

// selectRenewCommand returns the first usable renewal command, or false.
func selectRenewCommand() (renewCommand, bool) {
	for _, candidate := range renewCommands {
		if !candidate.available() {
			continue
		}
		if _, err := exec.LookPath(candidate.name); err != nil {
			continue
		}

		return candidate, true
	}

	return renewCommand{}, false
}

// renewSupportedPlatform reports whether any known DHCP client is present and
// managing this host.
func renewSupportedPlatform() bool {
	_, ok := selectRenewCommand()

	return ok
}

// renewLeasePlatform forces a renewal with whichever client manages the host.
func renewLeasePlatform(ctx context.Context, interfaceName string) error {
	if interfaceName == "" {
		return fmt.Errorf("%w: an interface name is required on Linux", ErrRenewUnsupported)
	}

	command, ok := selectRenewCommand()
	if !ok {
		return ErrRenewUnsupported
	}

	out, err := exec.CommandContext(ctx, command.name, command.args(interfaceName)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", command.name,
			strings.Join(command.args(interfaceName), " "), err, strings.TrimSpace(string(out)))
	}

	return nil
}
