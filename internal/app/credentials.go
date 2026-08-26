package app

// credentials.go wires the composition root to the device-credential CRUD
// use-case (ADR-0020). The adapter implements the credentials.Repository port
// over the device-credential repository, resolving the database lazily; a nil
// database yields credentials.ErrUnavailable so the handler degrades to 503
// rather than panicking.
//
// It also translates the repository's in-use sentinel. The use-case cannot
// import internal/database — that is the direction the port exists to
// prevent — so the mapping of "FK still references this row" onto
// credentials.ErrInUse belongs here, at the boundary.

import (
	"context"
	"errors"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/polling/credentials"
)

// NewDeviceCredentials builds the device-credential CRUD use-case over a lazy
// database accessor and the keyring that owns the DEK.
func NewDeviceCredentials(
	db func() *database.DB,
	encrypter credentials.Encrypter,
) (*credentials.Service, error) {
	return credentials.NewService(deviceCredentialRepo{db: db}, encrypter)
}

// deviceCredentialRepo implements credentials.Repository over the
// device-credential repository.
type deviceCredentialRepo struct {
	db func() *database.DB
}

func (a deviceCredentialRepo) repo() (*database.DeviceCredentialRepository, error) {
	db := a.db()
	if db == nil {
		return nil, credentials.ErrUnavailable
	}
	return db.DeviceCredentials(), nil
}

func (a deviceCredentialRepo) List(
	ctx context.Context, clientID string,
) ([]*polling.Credentials, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.List(ctx, clientID)
}

func (a deviceCredentialRepo) Get(
	ctx context.Context, id, clientID string,
) (*polling.Credentials, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.Get(ctx, id, clientID)
}

func (a deviceCredentialRepo) Upsert(ctx context.Context, c *polling.Credentials) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.Upsert(ctx, c)
}

func (a deviceCredentialRepo) Delete(ctx context.Context, clientID, id string) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	if delErr := repo.Delete(ctx, clientID, id); delErr != nil {
		if errors.Is(delErr, database.ErrCredentialInUse) {
			return fmt.Errorf("%w: %s", credentials.ErrInUse, id)
		}
		return delErr
	}
	return nil
}
