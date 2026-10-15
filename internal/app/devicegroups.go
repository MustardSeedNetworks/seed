package app

// devicegroups.go wires the composition root to the device-group use-case
// (P-D10, ADR-0020). A nil database makes every call devicegroups.ErrUnavailable,
// so the handler degrades to 503 rather than panicking.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/devicegroups"
)

// NewDeviceGroups builds the device-group use-case over a lazy database accessor.
func NewDeviceGroups(persist func() *Persistence) *devicegroups.Service {
	return devicegroups.NewService(deviceGroupStore{db: lazyHandle(persist)})
}

type deviceGroupStore struct {
	db func() *database.DB
}

func (a deviceGroupStore) repo() (*database.DeviceGroupRepository, error) {
	db := a.db()
	if db == nil {
		return nil, devicegroups.ErrUnavailable
	}
	return db.DeviceGroups(), nil
}

func (a deviceGroupStore) List(ctx context.Context, clientID string) ([]devicegroups.Group, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.List(ctx, clientID)
}

func (a deviceGroupStore) Get(ctx context.Context, clientID, id string) (devicegroups.Group, error) {
	repo, err := a.repo()
	if err != nil {
		return devicegroups.Group{}, err
	}
	return repo.Get(ctx, clientID, id)
}

func (a deviceGroupStore) Save(ctx context.Context, g *devicegroups.Group) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.Save(ctx, g)
}

func (a deviceGroupStore) Delete(ctx context.Context, clientID, id string) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.Delete(ctx, clientID, id)
}
