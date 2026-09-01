//go:build linux

package dhcp

import (
	"os/exec"
	"testing"
)

// The selection order is the point: a host running systemd-networkd must be
// renewed with networkctl even when dhcpcd is also installed, because telling
// dhcpcd to renew an interface it does not manage does nothing useful. Ubuntu
// 26.04 is exactly that shape -- dhcpcd present, dhclient absent,
// systemd-networkd active.
func TestSelectRenewCommand_PrefersTheManagingClient(t *testing.T) {
	got, ok := selectRenewCommandFrom([]renewCommand{
		{
			name:      "sh",
			available: func() bool { return false },
			args:      func(string) []string { return []string{"unmanaged"} },
		},
		{
			name:      "sh",
			available: func() bool { return true },
			args:      func(string) []string { return []string{"managing"} },
		},
	})
	if !ok {
		t.Fatal("no command selected although one is available")
	}
	if got.args("eth0")[0] != "managing" {
		t.Errorf("selected the unmanaged client: %v", got.args("eth0"))
	}
}

func TestSelectRenewCommand_SkipsAbsentBinaries(t *testing.T) {
	candidates := []renewCommand{
		{
			name:      "definitely-not-a-real-binary-xyz",
			available: func() bool { return true },
			args:      func(string) []string { return nil },
		},
	}

	if _, ok := selectRenewCommandFrom(candidates); ok {
		t.Error("selected a command whose binary is not on PATH")
	}
}

func TestRenewLease_RefusesWithoutAnInterface(t *testing.T) {
	if err := RenewLease(t.Context(), ""); err == nil {
		t.Error("an empty interface name must be refused, not applied to everything")
	}
}

// The real host: whatever this machine runs, RenewSupported must agree with
// whether a managing client is actually present. Asserted against the machine
// rather than a fixture so a wrong detector cannot pass.
func TestRenewSupported_MatchesTheHost(t *testing.T) {
	_, networkctlErr := exec.LookPath("networkctl")
	_, dhcpcdErr := exec.LookPath("dhcpcd")
	_, dhclientErr := exec.LookPath("dhclient")
	_, nmcliErr := exec.LookPath("nmcli")

	anyPresent := networkctlErr == nil || dhcpcdErr == nil || dhclientErr == nil || nmcliErr == nil
	if got := RenewSupported(); got != anyPresent {
		t.Errorf("RenewSupported() = %v, but client presence on this host = %v", got, anyPresent)
	}
}
