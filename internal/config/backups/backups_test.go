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
	return backups.NewService(cfg, path), cfg
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
