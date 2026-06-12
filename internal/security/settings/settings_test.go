package settings_test

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/security/settings"
)

type fakeStore struct{ cfg *config.Config }

func (f *fakeStore) Read(fn func(*config.Config))              { fn(f.cfg) }
func (f *fakeStore) Write(fn func(*config.Config) error) error { return fn(f.cfg) }

type fakeDetector struct {
	cfg          settings.DetectorConfig
	lastKnownSet []string
}

func (d *fakeDetector) Config() settings.DetectorConfig     { return d.cfg }
func (d *fakeDetector) UpdateKnownServers(servers []string) { d.lastKnownSet = servers }

func newService(t *testing.T) (*settings.Service, *config.Config, *fakeDetector) {
	t.Helper()
	cfg := &config.Config{}
	if err := cfg.InitCredentialKeyring(t.TempDir()); err != nil {
		t.Fatalf("InitCredentialKeyring: %v", err)
	}
	det := &fakeDetector{}
	return settings.NewService(&fakeStore{cfg: cfg}, det), cfg, det
}

func TestSNMPMasksStoredPasswords(t *testing.T) {
	svc, cfg, _ := newService(t)
	cfg.SNMP.V3Credentials = []config.SNMPv3Credential{
		{Name: "c1", Username: "u", AuthPassword: "stored-enc", PrivPassword: ""},
	}
	view := svc.SNMP()
	if len(view.V3Credentials) != 1 {
		t.Fatalf("want 1 credential, got %d", len(view.V3Credentials))
	}
	if view.V3Credentials[0].AuthPassword != settings.PasswordPlaceholder {
		t.Errorf("stored auth password should be masked, got %q", view.V3Credentials[0].AuthPassword)
	}
	if view.V3Credentials[0].PrivPassword != "" {
		t.Errorf("absent priv password should stay empty, got %q", view.V3Credentials[0].PrivPassword)
	}
}

func TestUpdateSNMPEncryptsNewPassword(t *testing.T) {
	svc, cfg, _ := newService(t)
	err := svc.UpdateSNMP(settings.SNMPUpdate{
		Communities:   []string{"public"},
		V3Credentials: []settings.SNMPv3Credential{{Name: "c1", Username: "u", AuthPassword: "secret"}},
		TimeoutMs:     2000,
		Retries:       3,
		Port:          161,
	})
	if err != nil {
		t.Fatalf("UpdateSNMP: %v", err)
	}
	stored := cfg.SNMP.V3Credentials[0].AuthPassword
	if stored == "" || stored == "secret" {
		t.Errorf("auth password should be encrypted, got %q", stored)
	}
	if cfg.SNMP.Timeout != 2*time.Second || cfg.SNMP.Port != 161 {
		t.Errorf("scalar fields not applied: %+v", cfg.SNMP)
	}
}

func TestUpdateSNMPPlaceholderPreservesStored(t *testing.T) {
	svc, cfg, _ := newService(t)
	cfg.SNMP.V3Credentials = []config.SNMPv3Credential{{Name: "c1", AuthPassword: "already-enc"}}

	err := svc.UpdateSNMP(settings.SNMPUpdate{
		V3Credentials: []settings.SNMPv3Credential{{Name: "c1", AuthPassword: settings.PasswordPlaceholder}},
	})
	if err != nil {
		t.Fatalf("UpdateSNMP: %v", err)
	}
	if cfg.SNMP.V3Credentials[0].AuthPassword != "already-enc" {
		t.Errorf("placeholder should preserve stored password, got %q", cfg.SNMP.V3Credentials[0].AuthPassword)
	}
}

func TestRogueDHCPReadComposesConfigAndDetector(t *testing.T) {
	svc, cfg, det := newService(t)
	cfg.DHCP.RogueDetection.Enabled = true
	det.cfg = settings.DetectorConfig{
		Interface: "eth0", KnownServers: []string{"10.0.0.1"}, AlertOnDetection: true,
	}
	view := svc.RogueDHCP()
	if !view.Enabled || view.Interface != "eth0" || !view.AlertOnDetection ||
		len(view.KnownServers) != 1 {
		t.Errorf("RogueDHCP did not compose config + detector: %+v", view)
	}
}

func TestUpdateRogueDHCPAppliesAndSyncsDetector(t *testing.T) {
	svc, cfg, det := newService(t)
	enabled := true
	err := svc.UpdateRogueDHCP(settings.RogueUpdate{
		Enabled:      &enabled,
		KnownServers: []string{"10.0.0.1", "10.0.0.2"},
	})
	if err != nil {
		t.Fatalf("UpdateRogueDHCP: %v", err)
	}
	if !cfg.DHCP.RogueDetection.Enabled || len(cfg.DHCP.RogueDetection.KnownServers) != 2 {
		t.Errorf("config not applied: %+v", cfg.DHCP.RogueDetection)
	}
	if len(det.lastKnownSet) != 2 {
		t.Errorf("detector known-servers not synced: %v", det.lastKnownSet)
	}

	// A nil-KnownServers update must not re-sync the detector.
	det.lastKnownSet = nil
	if err = svc.UpdateRogueDHCP(settings.RogueUpdate{Enabled: &enabled}); err != nil {
		t.Fatalf("UpdateRogueDHCP (nil servers): %v", err)
	}
	if det.lastKnownSet != nil {
		t.Errorf("detector should not sync when KnownServers is nil, got %v", det.lastKnownSet)
	}
}
