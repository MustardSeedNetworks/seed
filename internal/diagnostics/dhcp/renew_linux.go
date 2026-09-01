//go:build linux

package dhcp

import (
	"context"
	"errors"
	"fmt"
	"net"
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

// renewCommands returns the candidates, tried in order.
//
// There is no single answer on Linux. Ubuntu 26.04 ships neither dhclient nor
// NetworkManager by default -- interfaces are managed by systemd-networkd, and
// `networkctl renew` is the supported way to ask for a new lease. Older hosts
// and containers may have any of the others, so each is detected rather than
// assumed.
func renewCommands() []renewCommand {
	return []renewCommand{
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
}

// errUnknownInterface is returned when the requested interface does not exist
// on this host. Separate from ErrRenewUnsupported: the platform can renew, the
// caller named something that is not there.
var errUnknownInterface = errors.New("dhcp: unknown interface")

// systemdNetworkdActive reports whether systemd-networkd is managing this
// host's interfaces. Its runtime directory exists only while it is running.
func systemdNetworkdActive() bool {
	_, err := os.Stat("/run/systemd/netif")

	return err == nil
}

// selectRenewCommand returns the first usable renewal command, or false.
func selectRenewCommand() (renewCommand, bool) {
	return selectRenewCommandFrom(renewCommands())
}

// selectRenewCommandFrom is the testable half: candidates are passed in, so a
// test never has to swap package state to exercise the ordering.
func selectRenewCommandFrom(candidates []renewCommand) (renewCommand, bool) {
	for _, candidate := range candidates {
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

	// The name arrives from an HTTP request body, so it is checked against the
	// host's actual interfaces before it reaches a command. Passing it as argv
	// already rules out shell injection; this rules out asking the DHCP client
	// to act on something that is not an interface at all (gosec G204).
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return fmt.Errorf("%w: no interface named %q", errUnknownInterface, interfaceName)
	}

	command, ok := selectRenewCommand()
	if !ok {
		return ErrRenewUnsupported
	}

	// iface.Name rather than the caller's string: the value handed to the
	// command comes back from the kernel, so it is an interface that exists on
	// this host by construction rather than by check.
	args := command.args(iface.Name)

	out, cmdErr := exec.CommandContext(ctx, command.name, args...).CombinedOutput()
	if cmdErr != nil {
		return fmt.Errorf("%s %s: %w: %s", command.name,
			strings.Join(args, " "), cmdErr, strings.TrimSpace(string(out)))
	}

	return nil
}
