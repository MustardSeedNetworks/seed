package backups_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/config/backups"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

func newSvc(t *testing.T) (*backups.Service, *config.Config) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := testutil.NewConfigBuilder().WithPort(8080).Build()
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return backups.NewService(cfg, path, nil), cfg
}

func TestCreateListDeleteRoundTrip(t *testing.T) {
	svc, _ := newSvc(t)

	backup, err := svc.Create()
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	list, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Name != backup.Name {
		t.Fatalf("List did not return the created backup: %+v", list)
	}
	if err = svc.Delete(backup.Name); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	list, err = svc.List()
	if err != nil {
		t.Fatalf("List after delete: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("backup not deleted: %+v", list)
	}
}

func TestRestoreMissingBackupReturnsRestoreFailed(t *testing.T) {
	svc, _ := newSvc(t)
	if err := svc.Restore("does-not-exist"); !errors.Is(err, backups.ErrRestoreFailed) {
		t.Errorf("want ErrRestoreFailed, got %v", err)
	}
}

func TestVersionReportsCurrentAndLatest(t *testing.T) {
	svc, cfg := newSvc(t)
	current, latest := svc.Version()
	if current != cfg.Version {
		t.Errorf("current = %d, want %d", current, cfg.Version)
	}
	if latest != config.ConfigVersion {
		t.Errorf("latest = %d, want %d", latest, config.ConfigVersion)
	}
}

// alertProbe records the webhook URL the live config holds when the restore
// asks for the alert receiver to be re-pointed.
type alertProbe struct {
	cfg  *config.Config
	seen []string
}

func (p *alertProbe) ReconfigureAlerts() {
	p.cfg.RLock()
	defer p.cfg.RUnlock()
	p.seen = append(p.seen, p.cfg.Alerts.Webhook.URL)
}

// A restore applies every section of the backup, including the three that
// CopyFieldsFrom used to drop, and re-points the alert receiver at the
// restored one (#2928).
func TestRestoreAppliesAlertsLinkAndCableTest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := testutil.NewConfigBuilder().Build()
	cfg.Alerts.Webhook.URL = "https://restored.example/hook"
	cfg.Link.Mode = "100/full"
	cfg.CableTest.Enabled = true
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save config: %v", err)
	}
	probe := &alertProbe{cfg: cfg}
	svc := backups.NewService(cfg, path, probe)
	backup, err := svc.Create()
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	cfg.Alerts.Webhook.URL = "https://live.example/hook"
	cfg.Link.Mode = "auto"
	cfg.CableTest.Enabled = false
	if err = cfg.Save(path); err != nil {
		t.Fatalf("save live config: %v", err)
	}

	if err = svc.Restore(backup.Name); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if got := cfg.Alerts.Webhook.URL; got != "https://restored.example/hook" {
		t.Errorf("alert receiver = %q, want the restored one", got)
	}
	if got := cfg.Link.Mode; got != "100/full" {
		t.Errorf("link mode = %q, want 100/full", got)
	}
	if !cfg.CableTest.Enabled {
		t.Error("cable test not restored to enabled")
	}
	if len(probe.seen) != 1 || probe.seen[0] != "https://restored.example/hook" {
		t.Errorf("alert receiver re-pointed with %q, want once with the restored URL", probe.seen)
	}
}
