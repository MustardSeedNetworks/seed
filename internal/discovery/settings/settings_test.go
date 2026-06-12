package settings_test

import (
	"errors"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/discovery/settings"
)

// fakeStore is an in-memory settings.Store.
type fakeStore struct {
	cfg   config.NetworkDiscoveryConfig
	saved int
}

func (f *fakeStore) Discovery() config.NetworkDiscoveryConfig { return f.cfg }
func (f *fakeStore) SaveDiscovery(c config.NetworkDiscoveryConfig) error {
	f.cfg = c
	f.saved++
	return nil
}

// fakeSink records the last subnet set pushed to the scanner.
type fakeSink struct{ last []string }

func (f *fakeSink) SetAdditionalSubnets(cidrs []string) error { f.last = cidrs; return nil }

// fakeApplier records reload calls and can be made to fail.
type fakeApplier struct {
	reloads int
	err     error
}

func (f *fakeApplier) ReloadOptions() error { f.reloads++; return f.err }

func newService(cfg config.NetworkDiscoveryConfig) (*settings.Service, *fakeStore, *fakeSink) {
	st := &fakeStore{cfg: cfg}
	sk := &fakeSink{}
	return settings.NewService(st, sk, &fakeApplier{}), st, sk
}

func TestUpdateMergeRules(t *testing.T) {
	// Seed with non-zero values so "keep existing" is observable.
	svc, st, _ := newService(config.NetworkDiscoveryConfig{
		ARPScanWorkers: 7,
		PingTimeout:    500 * time.Millisecond,
		ScanInterval:   9 * time.Second,
		OUIFilePath:    "/old/oui",
		Enabled:        true,
	})

	// Conditional fields with zero/empty input keep the existing value; bools and
	// ScanInterval are set unconditionally.
	err := svc.Update(settings.Update{
		Enabled:        false, // unconditional → flips to false
		ARPScanWorkers: 0,     // keep 7
		PingTimeoutMs:  0,     // keep 500ms
		ScanIntervalMs: 0,     // unconditional → set to 0
		OUIFilePath:    "",    // keep /old/oui
		AutoScan:       true,  // unconditional
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	got := st.cfg
	if got.Enabled {
		t.Error("Enabled should be set unconditionally to false")
	}
	if !got.AutoScan {
		t.Error("AutoScan should be set unconditionally to true")
	}
	if got.ARPScanWorkers != 7 {
		t.Errorf("ARPScanWorkers = %d, want 7 (kept)", got.ARPScanWorkers)
	}
	if got.PingTimeout != 500*time.Millisecond {
		t.Errorf("PingTimeout = %v, want 500ms (kept)", got.PingTimeout)
	}
	if got.ScanInterval != 0 {
		t.Errorf("ScanInterval = %v, want 0 (set unconditionally)", got.ScanInterval)
	}
	if got.OUIFilePath != "/old/oui" {
		t.Errorf("OUIFilePath = %q, want kept", got.OUIFilePath)
	}
}

func TestUpdatePositiveValuesConvertMs(t *testing.T) {
	svc, st, _ := newService(config.NetworkDiscoveryConfig{})
	err := svc.Update(settings.Update{
		ARPScanWorkers: 4,
		PingTimeoutMs:  250,
		ScanTimeoutMs:  3000,
		Options: settings.OptionsUpdate{
			PortScan: settings.PortScanUpdate{BannerTimeoutMs: 2000},
			TCPProbe: settings.TCPProbeUpdate{TimeoutMs: 1500, Workers: 20},
		},
		Timing:   settings.TimingUpdate{ProbeIntervalMs: 75, RescanIntervalMs: 600000, Workers: 50},
		Profiler: settings.ProfilerUpdate{TimeoutMs: 2000, MaxConcurrent: 5, QuickPorts: []int{22, 80}},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	c := st.cfg
	if c.PingTimeout != 250*time.Millisecond || c.ScanTimeout != 3*time.Second {
		t.Errorf("ms→duration conversion off: ping=%v scan=%v", c.PingTimeout, c.ScanTimeout)
	}
	if c.Options.PortScan.BannerTimeout != 2*time.Second || c.Options.TCPProbe.Timeout != 1500*time.Millisecond {
		t.Errorf("options durations off: %+v", c.Options)
	}
	if c.Timing.RescanInterval != 10*time.Minute || c.Timing.Workers != 50 {
		t.Errorf("timing off: %+v", c.Timing)
	}
	if c.Profiler.MaxConcurrent != 5 || len(c.Profiler.QuickPorts) != 2 {
		t.Errorf("profiler off: %+v", c.Profiler)
	}
}

func TestAddSubnet(t *testing.T) {
	svc, st, sk := newService(config.NetworkDiscoveryConfig{})

	if err := svc.AddSubnet(config.SubnetConfig{CIDR: "10.0.0.0/24", Name: "lan", Enabled: true}); err != nil {
		t.Fatalf("AddSubnet: %v", err)
	}
	if len(st.cfg.AdditionalSubnets) != 1 {
		t.Fatalf("want 1 subnet, got %d", len(st.cfg.AdditionalSubnets))
	}
	if len(sk.last) != 1 || sk.last[0] != "10.0.0.0/24" {
		t.Errorf("enabled subnet not synced to scanner: %v", sk.last)
	}

	// Duplicate.
	if err := svc.AddSubnet(config.SubnetConfig{CIDR: "10.0.0.0/24"}); !errors.Is(err, settings.ErrSubnetExists) {
		t.Errorf("duplicate add: want ErrSubnetExists, got %v", err)
	}
	// Invalid CIDR.
	if err := svc.AddSubnet(config.SubnetConfig{CIDR: "not-a-cidr"}); !errors.Is(err, settings.ErrInvalidCIDR) {
		t.Errorf("invalid CIDR: want ErrInvalidCIDR, got %v", err)
	}
}

func TestUpdateAndDeleteSubnet(t *testing.T) {
	svc, st, sk := newService(config.NetworkDiscoveryConfig{
		AdditionalSubnets: []config.SubnetConfig{{CIDR: "192.168.1.0/24", Name: "old", Enabled: true}},
	})

	if err := svc.UpdateSubnet(config.SubnetConfig{CIDR: "192.168.1.0/24", Name: "new", Enabled: false}); err != nil {
		t.Fatalf("UpdateSubnet: %v", err)
	}
	if st.cfg.AdditionalSubnets[0].Name != "new" || st.cfg.AdditionalSubnets[0].Enabled {
		t.Errorf("subnet not updated: %+v", st.cfg.AdditionalSubnets[0])
	}
	// Disabled subnet is excluded from the scanner sync.
	if len(sk.last) != 0 {
		t.Errorf("disabled subnet should not sync: %v", sk.last)
	}

	if err := svc.UpdateSubnet(config.SubnetConfig{CIDR: "10.9.9.0/24"}); !errors.Is(err, settings.ErrSubnetNotFound) {
		t.Errorf("update missing: want ErrSubnetNotFound, got %v", err)
	}

	if err := svc.DeleteSubnet("192.168.1.0/24"); err != nil {
		t.Fatalf("DeleteSubnet: %v", err)
	}
	if len(st.cfg.AdditionalSubnets) != 0 {
		t.Errorf("subnet not deleted: %+v", st.cfg.AdditionalSubnets)
	}
	if err := svc.DeleteSubnet("192.168.1.0/24"); !errors.Is(err, settings.ErrSubnetNotFound) {
		t.Errorf("delete missing: want ErrSubnetNotFound, got %v", err)
	}
}

func TestSetOptionsPersistsThenApplies(t *testing.T) {
	st := &fakeStore{cfg: config.NetworkDiscoveryConfig{Enabled: true}}
	ap := &fakeApplier{}
	svc := settings.NewService(st, &fakeSink{}, ap)

	opts := config.DiscoveryOptions{PortScan: config.PortScanConfig{Enabled: true}}
	if err := svc.SetOptions(opts); err != nil {
		t.Fatalf("SetOptions: %v", err)
	}
	if !st.cfg.Options.PortScan.Enabled {
		t.Error("options not persisted")
	}
	if st.saved != 1 {
		t.Errorf("want 1 save, got %d", st.saved)
	}
	if ap.reloads != 1 {
		t.Errorf("want 1 reload, got %d", ap.reloads)
	}
}

func TestSetOptionsReturnsApplyError(t *testing.T) {
	st := &fakeStore{}
	ap := &fakeApplier{err: errors.New("reload failed")}
	svc := settings.NewService(st, &fakeSink{}, ap)

	if err := svc.SetOptions(config.DiscoveryOptions{}); err == nil {
		t.Error("SetOptions should surface the apply error")
	}
	// The save still committed before the apply was attempted.
	if st.saved != 1 {
		t.Errorf("want save committed before apply, got %d saves", st.saved)
	}
}
