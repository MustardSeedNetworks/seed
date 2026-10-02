package settings_test

import (
	"errors"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/discovery/learn"
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

func (f *fakeSink) SetTargetNetworks(cidrs []string) error { f.last = cidrs; return nil }

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
		ScanTimeout: 45 * time.Second,
		Timing:      config.DiscoveryTiming{RescanInterval: 5 * time.Minute},
		OUIFilePath: "/old/oui",
		Enabled:     true,
	})

	// Conditional fields with zero/empty input keep the existing value; bools
	// are set unconditionally.
	err := svc.Update(settings.Update{
		Enabled:       false,                                      // unconditional → flips to false
		ScanTimeoutMs: 0,                                          // keep 45s
		Timing:        settings.TimingUpdate{RescanIntervalMs: 0}, // keep 5m
		OUIFilePath:   "",                                         // keep /old/oui
		AutoScan:      true,                                       // unconditional
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
	if got.ScanTimeout != 45*time.Second {
		t.Errorf("ScanTimeout = %v, want 45s (kept)", got.ScanTimeout)
	}
	if got.Timing.RescanInterval != 5*time.Minute {
		t.Errorf("RescanInterval = %v, want 5m (kept)", got.Timing.RescanInterval)
	}
	if got.OUIFilePath != "/old/oui" {
		t.Errorf("OUIFilePath = %q, want kept", got.OUIFilePath)
	}
}

func TestUpdatePositiveValuesConvertMs(t *testing.T) {
	svc, st, _ := newService(config.NetworkDiscoveryConfig{})
	err := svc.Update(settings.Update{
		ScanTimeoutMs: 3000,
		Options: settings.OptionsUpdate{
			TCPProbe: settings.TCPProbeUpdate{TimeoutMs: 1500, Workers: 20},
		},
		Timing:   settings.TimingUpdate{RescanIntervalMs: 600000},
		Profiler: settings.ProfilerUpdate{TimeoutMs: 2000, MaxConcurrent: 5, QuickPorts: []int{22, 80}},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	c := st.cfg
	if c.ScanTimeout != 3*time.Second {
		t.Errorf("ms→duration conversion off: scan=%v", c.ScanTimeout)
	}
	if c.Options.TCPProbe.Timeout != 1500*time.Millisecond {
		t.Errorf("options durations off: %+v", c.Options)
	}
	if c.Timing.RescanInterval != 10*time.Minute {
		t.Errorf("timing off: %+v", c.Timing)
	}
	if c.Profiler.MaxConcurrent != 5 || len(c.Profiler.QuickPorts) != 2 {
		t.Errorf("profiler off: %+v", c.Profiler)
	}
}

// TestUpdateReloadsTheScannerOnlyWhenItsInputsChange pins #491's second half:
// a new rescan interval saved from the drawer must reach the running scanner,
// and the drawer's auto-save of an unchanged form must not restart it.
func TestUpdateReloadsTheScannerOnlyWhenItsInputsChange(t *testing.T) {
	base := config.NetworkDiscoveryConfig{
		ScanTimeout: 30 * time.Second,
		Timing:      config.DiscoveryTiming{RescanInterval: time.Minute},
		Options: config.DiscoveryOptions{
			ARPScan:  true,
			PortScan: config.PortScanConfig{Preset: config.PortPresetCommon},
		},
	}
	unchanged := settings.Update{
		ScanTimeoutMs: 30000,
		Timing:        settings.TimingUpdate{RescanIntervalMs: 60000},
		Options:       settings.OptionsUpdate{ARPScan: true},
	}
	for _, tc := range []struct {
		name        string
		edit        func(*settings.Update)
		wantReloads int
	}{
		{name: "nothing changed", edit: func(*settings.Update) {}, wantReloads: 0},
		{name: "scan time limit only", edit: func(u *settings.Update) { u.ScanTimeoutMs = 45000 }},
		{
			name:        "rescan interval",
			edit:        func(u *settings.Update) { u.Timing.RescanIntervalMs = 180000 },
			wantReloads: 1,
		},
		{
			name:        "a scan method",
			edit:        func(u *settings.Update) { u.Options.ICMPScan = true },
			wantReloads: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := &fakeStore{cfg: base}
			ap := &fakeApplier{}
			in := unchanged
			tc.edit(&in)
			if err := settings.NewService(st, &fakeSink{}, ap).Update(in); err != nil {
				t.Fatalf("Update: %v", err)
			}
			if ap.reloads != tc.wantReloads {
				t.Errorf("reloads = %d, want %d", ap.reloads, tc.wantReloads)
			}
			if st.saved != 1 {
				t.Errorf("saved = %d, want 1", st.saved)
			}
		})
	}
}

func TestUpdatePortPreset(t *testing.T) {
	for _, tc := range []struct {
		preset  string
		want    config.PortPreset
		wantErr error
	}{
		{preset: "secure", want: config.PortPresetSecure},
		{preset: "custom", want: config.PortPresetCustom},
		{preset: "", want: config.PortPresetInsecure},
		{preset: "everything", want: config.PortPresetInsecure, wantErr: settings.ErrInvalidPortPreset},
	} {
		t.Run(tc.preset, func(t *testing.T) {
			st := &fakeStore{cfg: config.NetworkDiscoveryConfig{
				Options: config.DiscoveryOptions{
					PortScan: config.PortScanConfig{Preset: config.PortPresetInsecure},
				},
			}}
			err := settings.NewService(st, &fakeSink{}, &fakeApplier{}).Update(settings.Update{
				Options: settings.OptionsUpdate{PortScan: settings.PortScanUpdate{Preset: tc.preset}},
			})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Update error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil && st.saved != 0 {
				t.Error("a refused preset was saved")
			}
			if got := st.cfg.Options.PortScan.Preset; got != tc.want {
				t.Errorf("preset = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAddSubnet(t *testing.T) {
	svc, st, sk := newService(config.NetworkDiscoveryConfig{})

	if err := svc.AddSubnet(config.SubnetConfig{CIDR: "10.0.0.0/24", Name: "lan", Enabled: true}); err != nil {
		t.Fatalf("AddSubnet: %v", err)
	}
	if len(st.cfg.TargetNetworks) != 1 {
		t.Fatalf("want 1 subnet, got %d", len(st.cfg.TargetNetworks))
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
		TargetNetworks: []config.SubnetConfig{{CIDR: "192.168.1.0/24", Name: "old", Enabled: true}},
	})

	if err := svc.UpdateSubnet(config.SubnetConfig{CIDR: "192.168.1.0/24", Name: "new", Enabled: false}); err != nil {
		t.Fatalf("UpdateSubnet: %v", err)
	}
	if st.cfg.TargetNetworks[0].Name != "new" || st.cfg.TargetNetworks[0].Enabled {
		t.Errorf("subnet not updated: %+v", st.cfg.TargetNetworks[0])
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
	if len(st.cfg.TargetNetworks) != 0 {
		t.Errorf("subnet not deleted: %+v", st.cfg.TargetNetworks)
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

// Learn is the write half of seed#2695: a network the sweep saw becomes a
// target the operator can switch on, without ever switching it on for them.
func TestLearnAddsDisabledEntries(t *testing.T) {
	svc, st, sink := newService(config.NetworkDiscoveryConfig{})

	added, err := svc.Learn([]learn.Candidate{
		{CIDR: "10.44.10.0/24", Source: learn.SourceRouteTable, Router: "10.44.40.1"},
		{CIDR: "10.44.20.0/24", Source: learn.SourceAddressTable, Router: "10.44.40.2"},
	})
	if err != nil {
		t.Fatalf("Learn() error = %v", err)
	}
	if added != 2 {
		t.Errorf("Learn() added %d, want 2", added)
	}

	got := st.cfg.TargetNetworks
	if len(got) != 2 {
		t.Fatalf("TargetNetworks = %+v, want 2 entries", got)
	}
	for _, subnet := range got {
		if subnet.Enabled {
			t.Errorf("%s is enabled; a learned network is the operator's call", subnet.CIDR)
		}
		if !subnet.Learned {
			t.Errorf("%s is not marked learned", subnet.CIDR)
		}
		if subnet.Name == "" {
			t.Errorf("%s has no name", subnet.CIDR)
		}
	}
	if len(sink.last) != 0 {
		t.Errorf("scanner was given %v; nothing learned is enabled", sink.last)
	}
}

// The learner runs after every sweep. Re-learning must not rewrite the config,
// and above all must not undo an operator who switched a learned network on.
func TestLearnIsIdempotentAndNeverUndoesTheOperator(t *testing.T) {
	svc, st, _ := newService(config.NetworkDiscoveryConfig{
		TargetNetworks: []config.SubnetConfig{
			{CIDR: "10.44.10.0/24", Name: "Ward network", Enabled: true, Learned: true},
			{CIDR: "10.44.20.0/24", Name: "Typed in by hand", Enabled: true},
		},
	})
	saves := st.saved

	added, err := svc.Learn([]learn.Candidate{
		{CIDR: "10.44.10.0/24", Source: learn.SourceRouteTable, Router: "10.44.40.1"},
		{CIDR: "10.44.20.0/24", Source: learn.SourceRouteTable, Router: "10.44.40.1"},
	})
	if err != nil {
		t.Fatalf("Learn() error = %v", err)
	}
	if added != 0 {
		t.Errorf("Learn() added %d, want 0", added)
	}
	if st.saved != saves {
		t.Errorf("config was saved %d extra times; nothing changed", st.saved-saves)
	}

	for _, subnet := range st.cfg.TargetNetworks {
		if !subnet.Enabled {
			t.Errorf("%s was switched off by re-learning", subnet.CIDR)
		}
	}
	if st.cfg.TargetNetworks[1].Learned {
		t.Errorf("an operator's own entry was relabelled as learned")
	}
	if st.cfg.TargetNetworks[0].Name != "Ward network" {
		t.Errorf("Name = %q; a rename by the operator was overwritten", st.cfg.TargetNetworks[0].Name)
	}
}

// A malformed candidate is dropped rather than persisted: everything downstream
// of TargetNetworks parses the CIDR.
func TestLearnRejectsAMalformedCandidate(t *testing.T) {
	svc, st, _ := newService(config.NetworkDiscoveryConfig{})

	added, err := svc.Learn([]learn.Candidate{{CIDR: "not-a-network", Source: learn.SourceRouteTable}})
	if err != nil {
		t.Fatalf("Learn() error = %v", err)
	}
	if added != 0 || len(st.cfg.TargetNetworks) != 0 {
		t.Errorf("Learn() added %d entries %+v, want none", added, st.cfg.TargetNetworks)
	}
}
