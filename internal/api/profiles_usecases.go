package api

// profiles_usecases.go wires the API layer to the profiles application
// (use-case) service (ADR-0016 strangle phase 3). The adapter implements the
// narrow profilesapp.Store port over the database Profiles + Settings
// repositories, mapping the repository's domain errors to the use-case's
// sentinels and the database.Profile row to the use-case model.

import (
	"context"
	"errors"

	"github.com/MustardSeedNetworks/seed/internal/database"
	profilesapp "github.com/MustardSeedNetworks/seed/internal/profiles/app"
)

// profilesStore implements profilesapp.Store over the database repositories.
// The database is resolved lazily; callers (handlers) gate on s.db() != nil
// before invoking the use-case, so a nil db here is not expected.
type profilesStore struct {
	db func() *database.DB
}

// mapProfileErr translates database sentinels to use-case sentinels.
func mapProfileErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, database.ErrProfileNotFound):
		return profilesapp.ErrNotFound
	case errors.Is(err, database.ErrProfileNameExists):
		return profilesapp.ErrNameExists
	case errors.Is(err, database.ErrProfileConflict):
		return profilesapp.ErrConflict
	default:
		return err
	}
}

func dbToAppProfile(p *database.Profile) profilesapp.Profile {
	return profilesapp.Profile{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		ConfigJSON:  p.ConfigJSON,
		IsDefault:   p.IsDefault,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func appToDBProfile(p profilesapp.Profile) *database.Profile {
	return &database.Profile{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		ConfigJSON:  p.ConfigJSON,
		IsDefault:   p.IsDefault,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func (s profilesStore) List(ctx context.Context) ([]profilesapp.Profile, error) {
	rows, err := s.db().Profiles().List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]profilesapp.Profile, 0, len(rows))
	for _, p := range rows {
		out = append(out, dbToAppProfile(p))
	}
	return out, nil
}

func (s profilesStore) Get(ctx context.Context, id string) (profilesapp.Profile, error) {
	p, err := s.db().Profiles().Get(ctx, id)
	if err != nil {
		return profilesapp.Profile{}, mapProfileErr(err)
	}
	return dbToAppProfile(p), nil
}

func (s profilesStore) GetByName(ctx context.Context, name string) (profilesapp.Profile, bool, error) {
	p, err := s.db().Profiles().GetByName(ctx, name)
	if err != nil {
		if errors.Is(err, database.ErrProfileNotFound) {
			return profilesapp.Profile{}, false, nil
		}
		return profilesapp.Profile{}, false, err
	}
	if p == nil {
		return profilesapp.Profile{}, false, nil
	}
	return dbToAppProfile(p), true, nil
}

func (s profilesStore) GetDefault(ctx context.Context) (profilesapp.Profile, error) {
	p, err := s.db().Profiles().GetDefault(ctx)
	if err != nil {
		return profilesapp.Profile{}, mapProfileErr(err)
	}
	return dbToAppProfile(p), nil
}

func (s profilesStore) Create(ctx context.Context, p profilesapp.Profile) error {
	return mapProfileErr(s.db().Profiles().Create(ctx, appToDBProfile(p)))
}

func (s profilesStore) Update(ctx context.Context, p profilesapp.Profile, ifMatch string) error {
	if ifMatch == "" {
		return mapProfileErr(s.db().Profiles().Update(ctx, appToDBProfile(p)))
	}
	return mapProfileErr(s.db().Profiles().UpdateIfMatch(ctx, appToDBProfile(p), ifMatch))
}

func (s profilesStore) Delete(ctx context.Context, id string) error {
	return s.db().Profiles().Delete(ctx, id)
}

func (s profilesStore) Count(ctx context.Context) (int, error) {
	n, err := s.db().Profiles().Count(ctx)
	return int(n), err
}

func (s profilesStore) ActiveID(ctx context.Context) (string, error) {
	return s.db().Settings().GetValue(ctx, database.SettingKeyActiveProfile)
}

func (s profilesStore) SetActiveID(ctx context.Context, id string) error {
	return s.db().Settings().Set(ctx, database.SettingKeyActiveProfile, id)
}

// initProfilesUseCase builds the profiles use-case over the lazy database
// accessor (ADR-0016 phase 3).
func (s *Server) initProfilesUseCase() {
	s.profiles = profilesapp.NewService(profilesStore{db: s.db})
}
