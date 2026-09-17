package gateway_test

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
)

// The gateway shown on the Network page must belong to the interface the
// operator selected (#2690). It used to come from the system default route
// whatever the active interface was, so a probe pinned to a virtual link
// reported — and pinged — the Mac's Wi-Fi gateway at 100 % loss.

func routeOn(iface string) func() (string, error) {
	return func() (string, error) { return iface, nil }
}

// The one gateway every case in this file reads back, so the assertions name
// the same address the stub hands out.
const testGateway = "192.168.20.1"

func gatewayIs() func() (string, error) {
	return func() (string, error) { return testGateway, nil }
}

func TestGatewayForInterfaceScoping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		iface string
		want  string
	}{
		{
			name:  "the interface carrying the default route gets its gateway",
			iface: "en0",
			want:  testGateway,
		},
		{
			name:  "another interface has no gateway of its own",
			iface: "feth0",
			want:  "",
		},
		{
			name:  "no selection keeps the system-wide answer",
			iface: "",
			want:  testGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := gateway.GatewayForInterfaceWithReads(
				tt.iface, routeOn("en0"), gatewayIs(),
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGatewayForInterfaceReportsARoutingTableError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("fetch RIB: permission denied")
	_, err := gateway.GatewayForInterfaceWithReads(
		"en0",
		func() (string, error) { return "", wantErr },
		gatewayIs(),
	)
	if !errors.Is(err, wantErr) {
		t.Errorf("got %v, want the routing-table error, not a silent fallback", err)
	}
}

func TestTestScopesDetectionToTheSelectedInterface(t *testing.T) {
	t.Parallel()

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetRoutingForTesting(routeOn("en0"), gatewayIs())
	tester.SetInterface("feth0")
	tester.TesterSetPingCount(1)

	stats := tester.Test()
	if stats.Gateway != "" {
		t.Errorf("got gateway %q, want none: the tester must not ping another interface's gateway", stats.Gateway)
	}
	if stats.Sent != 0 {
		t.Errorf("got %d packets sent, want 0: there is nothing on this interface to ping", stats.Sent)
	}
}

func TestTestPingsTheGatewayOfTheSelectedInterface(t *testing.T) {
	t.Parallel()

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetRoutingForTesting(routeOn("en0"), gatewayIs())
	tester.SetInterface("en0")
	tester.TesterSetPingCount(1)

	if stats := tester.Test(); stats.Gateway != testGateway {
		t.Errorf("got gateway %q, want 192.168.20.1: en0 carries the default route", stats.Gateway)
	}
}

func TestSetInterfaceDropsAGatewayDetectedForTheOldInterface(t *testing.T) {
	t.Parallel()

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetGateway(testGateway)

	tester.SetInterface("feth0")

	if got := tester.GetGateway(); got != "" {
		t.Errorf("got %q, want the cached gateway cleared when the interface changes", got)
	}
}

func TestSetInterfaceKeepsTheGatewayWhenTheInterfaceIsUnchanged(t *testing.T) {
	t.Parallel()

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.SetInterface("en0")
	tester.SetGateway(testGateway)

	tester.SetInterface("en0")

	if got := tester.GetGateway(); got != testGateway {
		t.Errorf("got %q, want 192.168.20.1: a repeated selection is not a change", got)
	}
}

func TestSetInterfaceDropsStatsMeasuredForTheOldInterface(t *testing.T) {
	t.Parallel()

	tester := gateway.NewTester(gateway.DefaultThresholds())
	tester.TesterSetStats(&gateway.PingStats{
		Gateway:     testGateway,
		Sent:        3,
		Received:    0,
		LossPercent: 100,
		Status:      gateway.StatusError,
	})

	tester.SetInterface("feth0")

	stats := tester.GetStats()
	if stats.Gateway != "" {
		t.Errorf("got gateway %q in the cached stats, want them cleared with the interface", stats.Gateway)
	}
	if stats.LossPercent != 0 {
		t.Errorf("got %v %% loss carried over, want 0: it was measured against another link", stats.LossPercent)
	}
}
