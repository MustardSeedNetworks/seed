package netif

import (
	"errors"
	"strings"
	"testing"
)

// fakeApplier records what it was asked to apply and fails on demand, so the
// rollback path can be driven without touching a network interface.
type fakeApplier struct {
	applied  []StaticIPConfig
	failOn   int // 1-based call number to fail, 0 for never
	failWith error
	calls    int
}

func (f *fakeApplier) Apply(_ string, cfg *StaticIPConfig) error {
	f.calls++
	if f.failOn == f.calls {
		return f.failWith
	}
	f.applied = append(f.applied, *cfg)
	return nil
}

type fakeSnapshotter struct {
	cfg *StaticIPConfig
	err error
}

func (f fakeSnapshotter) Snapshot(string) (*StaticIPConfig, error) {
	return f.cfg, f.err
}

func previousConfig() *StaticIPConfig {
	return &StaticIPConfig{
		Address: "192.168.1.10",
		Netmask: "255.255.255.0",
		Gateway: "192.168.1.1",
		DNS:     []string{"192.168.1.1"},
	}
}

func wantedConfig() *StaticIPConfig {
	return &StaticIPConfig{
		Address: "10.0.0.5",
		Netmask: "255.255.255.0",
		Gateway: "10.0.0.1",
		DNS:     []string{"10.0.0.1"},
	}
}

// TestFailedApplyIsRolledBack is the defect in #50: each platform applies
// address, then gateway, then DNS, returning on the first error. Without a
// rollback the interface keeps whatever was applied before the failure.
func TestFailedApplyIsRolledBack(t *testing.T) {
	applier := &fakeApplier{failOn: 1, failWith: errors.New("gateway unreachable")}
	m := &Manager{
		snapshotter: fakeSnapshotter{cfg: previousConfig()},
		applier:     applier,
	}

	err := m.ConfigureStaticIP("eth0", wantedConfig())
	if err == nil {
		t.Fatal("ConfigureStaticIP reported success after the apply failed")
	}
	if !strings.Contains(err.Error(), "gateway unreachable") {
		t.Errorf("error %q does not carry the original failure", err)
	}
	if !strings.Contains(err.Error(), "rolled eth0 back") {
		t.Errorf("error %q does not say the interface was rolled back", err)
	}

	if len(applier.applied) != 1 {
		t.Fatalf("applied %d configurations, want 1 (the rollback)", len(applier.applied))
	}
	if got := applier.applied[0]; got.Address != "192.168.1.10" ||
		got.Gateway != "192.168.1.1" {
		t.Errorf("rolled back to %+v, want the previous configuration %+v",
			got, *previousConfig())
	}
}

// TestSuccessfulApplyDoesNotRollBack pins that the happy path is untouched: one
// apply, no revert, no error.
func TestSuccessfulApplyDoesNotRollBack(t *testing.T) {
	applier := &fakeApplier{}
	m := &Manager{
		snapshotter: fakeSnapshotter{cfg: previousConfig()},
		applier:     applier,
	}

	if err := m.ConfigureStaticIP("eth0", wantedConfig()); err != nil {
		t.Fatalf("ConfigureStaticIP: %v", err)
	}
	if applier.calls != 1 {
		t.Errorf("applier called %d times, want 1 — a successful apply must not "+
			"revert", applier.calls)
	}
	if got := applier.applied[0]; got.Address != "10.0.0.5" {
		t.Errorf("applied %q, want the requested address", got.Address)
	}
}

// TestFailedRollbackIsReportedDistinctly pins that "your change failed" and
// "your change failed and the interface is now in an unknown state" are
// distinguishable. An operator has to be able to tell those apart.
func TestFailedRollbackIsReportedDistinctly(t *testing.T) {
	// Every call fails, so the rollback fails too.
	m := &Manager{
		snapshotter: fakeSnapshotter{cfg: previousConfig()},
		applier:     alwaysFailingApplier{err: errors.New("apply failed")},
	}

	err := m.ConfigureStaticIP("eth0", wantedConfig())
	if err == nil {
		t.Fatal("ConfigureStaticIP reported success")
	}
	if !strings.Contains(err.Error(), "indeterminate state") {
		t.Errorf("error %q does not warn that the interface is in an unknown "+
			"state after a failed rollback", err)
	}
}

type alwaysFailingApplier struct{ err error }

func (a alwaysFailingApplier) Apply(string, *StaticIPConfig) error { return a.err }

// TestApplyProceedsWithoutASnapshot pins that an interface with nothing to
// restore is still configurable. Refusing to configure an unconfigured
// interface would break the ordinary first-time case.
func TestApplyProceedsWithoutASnapshot(t *testing.T) {
	applier := &fakeApplier{}
	m := &Manager{
		snapshotter: fakeSnapshotter{err: errors.New("interface has no IPv4 address")},
		applier:     applier,
	}

	if err := m.ConfigureStaticIP("eth0", wantedConfig()); err != nil {
		t.Fatalf("ConfigureStaticIP: %v", err)
	}
	if applier.calls != 1 {
		t.Errorf("applier called %d times, want 1", applier.calls)
	}
}

// TestFailedApplyWithoutASnapshotReturnsTheApplyError pins that when there is
// nothing to roll back to, the caller still gets the real failure rather than a
// rollback message about a rollback that never happened.
func TestFailedApplyWithoutASnapshotReturnsTheApplyError(t *testing.T) {
	m := &Manager{
		snapshotter: fakeSnapshotter{err: errors.New("no address")},
		applier:     alwaysFailingApplier{err: errors.New("netlink refused")},
	}

	err := m.ConfigureStaticIP("eth0", wantedConfig())
	if err == nil {
		t.Fatal("ConfigureStaticIP reported success")
	}
	if !strings.Contains(err.Error(), "netlink refused") {
		t.Errorf("error %q does not carry the apply failure", err)
	}
	if strings.Contains(err.Error(), "rolled") {
		t.Errorf("error %q claims a rollback happened with nothing to roll back to", err)
	}
}

func TestSplitFirstCIDR(t *testing.T) {
	for _, tc := range []struct {
		name        string
		addresses   []string
		wantAddr    string
		wantNetmask string
	}{
		{"a plain IPv4 address", []string{"192.168.1.10/24"}, "192.168.1.10", "255.255.255.0"},
		{"a /16", []string{"10.1.2.3/16"}, "10.1.2.3", "255.255.0.0"},
		{
			// An interface commonly carries a link-local IPv6 address first.
			name:        "IPv6 is skipped in favour of IPv4",
			addresses:   []string{"fe80::1/64", "192.168.1.10/24"},
			wantAddr:    "192.168.1.10",
			wantNetmask: "255.255.255.0",
		},
		{"no IPv4 present", []string{"fe80::1/64"}, "", ""},
		{"no addresses at all", nil, "", ""},
		{"an unparseable entry", []string{"not-an-address"}, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			addr, netmask := splitFirstCIDR(tc.addresses)
			if addr != tc.wantAddr || netmask != tc.wantNetmask {
				t.Errorf("got (%q, %q), want (%q, %q)",
					addr, netmask, tc.wantAddr, tc.wantNetmask)
			}
		})
	}
}
