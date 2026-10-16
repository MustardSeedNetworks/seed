package app

// wmi.go wires the composition root to the Windows inventory use-case
// (P-D11). The adapters resolve the database lazily, as the configuration
// backup's do, so a nil database degrades the handlers to 503 rather than
// panicking, and translate the repository's host-and-port conflict into the
// use-case's validation error.

import (
	"context"
	"errors"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/wmi"
)

// NewWMIInventory builds the Windows inventory use-case over a lazy database
// accessor, the keyring that decrypts vault passwords, and the collector that
// signs in to hosts (wmi.WSManCollector in production).
func NewWMIInventory(
	persist func() *Persistence, decrypt wmi.Decrypter, collect wmi.Collector,
) (*wmi.Service, error) {
	db := lazyHandle(persist)
	return wmi.NewService(wmiRepo{db: db}, wmiCreds{db: db}, decrypt, collect)
}

type wmiCreds struct {
	db func() *database.DB
}

func (a wmiCreds) Get(ctx context.Context, id, clientID string) (*polling.Credentials, error) {
	db := a.db()
	if db == nil {
		return nil, wmi.ErrUnavailable
	}
	return db.DeviceCredentials().Get(ctx, id, clientID)
}

// wmiRepo implements wmi.Repository over the database.
type wmiRepo struct {
	db func() *database.DB
}

func (a wmiRepo) repo() (*database.WMIRepository, error) {
	db := a.db()
	if db == nil {
		return nil, wmi.ErrUnavailable
	}
	return db.WMI(), nil
}

func (a wmiRepo) ListTargets(ctx context.Context, clientID string) ([]wmi.Target, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.ListTargets(ctx, clientID)
}

func (a wmiRepo) GetTarget(ctx context.Context, clientID, id string) (wmi.Target, error) {
	repo, err := a.repo()
	if err != nil {
		return wmi.Target{}, err
	}
	return repo.GetTarget(ctx, clientID, id)
}

func (a wmiRepo) SaveTarget(ctx context.Context, t *wmi.Target) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	if saveErr := repo.SaveTarget(ctx, t); errors.Is(saveErr, database.ErrWMITargetConflict) {
		return wmi.ValidationError{Msg: "another inventory target already uses this host and port"}
	} else if saveErr != nil {
		return saveErr
	}
	return nil
}

func (a wmiRepo) DeleteTarget(ctx context.Context, clientID, id string) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.DeleteTarget(ctx, clientID, id)
}

func (a wmiRepo) SetPendingCert(ctx context.Context, clientID, id, fingerprint string) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.SetPendingCert(ctx, clientID, id, fingerprint)
}

func (a wmiRepo) ConfirmCert(ctx context.Context, clientID, id, fingerprint string) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.ConfirmCert(ctx, clientID, id, fingerprint)
}

func (a wmiRepo) ClearCert(ctx context.Context, clientID, id string) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.ClearCert(ctx, clientID, id)
}

func (a wmiRepo) RecordAttempt(ctx context.Context, at wmi.Attempt) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.RecordAttempt(ctx, at)
}

func (a wmiRepo) GetInventory(ctx context.Context, clientID, targetID string) (wmi.StoredInventory, error) {
	repo, err := a.repo()
	if err != nil {
		return wmi.StoredInventory{}, err
	}
	return repo.GetInventory(ctx, clientID, targetID)
}
