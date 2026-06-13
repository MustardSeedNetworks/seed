package app

// profiles.go wires the composition root to the profiles catalog application
// (use-case) service (ADR-0020). The adapter implements the narrow catalog.Store
// port over the database Profiles + Settings repositories, mapping the
// repository's domain errors to the use-case's sentinels and the
// database.Profile row to the use-case model, so the profile handlers depend on
// a use-case instead of reaching into the database repositories directly.

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/profiles/catalog"
)

// profilesStore implements catalog.Store over the database repositories. The
// database is resolved lazily; Available reports whether it is wired so the
// use-case can degrade to the service-unavailable path.
type profilesStore struct {
	db func() *database.DB
}

func (s profilesStore) Available() bool { return s.db() != nil }

// mapProfileErr translates database sentinels to use-case sentinels.
func mapProfileErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, database.ErrProfileNotFound):
		return catalog.ErrNotFound
	case errors.Is(err, database.ErrProfileNameExists):
		return catalog.ErrNameExists
	case errors.Is(err, database.ErrProfileConflict):
		return catalog.ErrConflict
	default:
		return err
	}
}

func dbToAppProfile(p *database.Profile) catalog.Profile {
	return catalog.Profile{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		ConfigJSON:  p.ConfigJSON,
		IsDefault:   p.IsDefault,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		RowVersion:  p.RowVersion,
	}
}

func appToDBProfile(p catalog.Profile) *database.Profile {
	return &database.Profile{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		ConfigJSON:  p.ConfigJSON,
		IsDefault:   p.IsDefault,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		RowVersion:  p.RowVersion,
	}
}

func (s profilesStore) List(ctx context.Context) ([]catalog.Profile, error) {
	rows, err := s.db().Profiles().List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]catalog.Profile, 0, len(rows))
	for _, p := range rows {
		out = append(out, dbToAppProfile(p))
	}
	return out, nil
}

func (s profilesStore) Get(ctx context.Context, id string) (catalog.Profile, error) {
	p, err := s.db().Profiles().Get(ctx, id)
	if err != nil {
		return catalog.Profile{}, mapProfileErr(err)
	}
	return dbToAppProfile(p), nil
}

func (s profilesStore) GetByName(ctx context.Context, name string) (catalog.Profile, bool, error) {
	p, err := s.db().Profiles().GetByName(ctx, name)
	if err != nil {
		if errors.Is(err, database.ErrProfileNotFound) {
			return catalog.Profile{}, false, nil
		}
		return catalog.Profile{}, false, err
	}
	if p == nil {
		return catalog.Profile{}, false, nil
	}
	return dbToAppProfile(p), true, nil
}

func (s profilesStore) GetDefault(ctx context.Context) (catalog.Profile, error) {
	p, err := s.db().Profiles().GetDefault(ctx)
	if err != nil {
		return catalog.Profile{}, mapProfileErr(err)
	}
	return dbToAppProfile(p), nil
}

func (s profilesStore) Create(ctx context.Context, p catalog.Profile) error {
	return mapProfileErr(s.db().Profiles().Create(ctx, appToDBProfile(p)))
}

func (s profilesStore) Update(ctx context.Context, p catalog.Profile, ifMatch string) error {
	if ifMatch == "" {
		return mapProfileErr(s.db().Profiles().Update(ctx, appToDBProfile(p)))
	}
	// The ETag is the row_version as a decimal string. A token that does not parse
	// to an int64 cannot match any row's version, so it is a precondition failure
	// (412), not a 500 — treat it as a conflict without touching the row.
	expectedVersion, err := strconv.ParseInt(ifMatch, 10, 64)
	if err != nil {
		return catalog.ErrConflict
	}
	return mapProfileErr(s.db().Profiles().UpdateIfMatch(ctx, appToDBProfile(p), expectedVersion))
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

// profilesLiveConfig implements catalog.LiveConfig over the live config: it
// applies an activated profile's saved settings and persists them. A malformed
// saved config is logged and skipped (best-effort, never fatal); only a failed
// persist is returned, wrapped as catalog.ErrConfigApply.
type profilesLiveConfig struct {
	cfg  *config.Config
	path string
}

func (a profilesLiveConfig) Apply(ctx context.Context, profileJSON string) error {
	if err := a.cfg.ApplyProfileJSON(profileJSON); err != nil {
		logging.FromContext(ctx).WarnContext(ctx,
			"profile activate: failed to apply saved config, keeping current", "error", err)
		return nil
	}
	if err := a.cfg.Save(a.path); err != nil {
		return fmt.Errorf("%w: %w", catalog.ErrConfigApply, err)
	}
	return nil
}

// NewProfiles builds the profiles catalog use-case (ADR-0020) over the lazy db
// accessor and the live config. The catalog.Store adapter spans the database
// Profiles + Settings repositories; the catalog.LiveConfig adapter applies an
// activated profile's settings to cfg and persists to path. The database is
// resolved through db on each call so the api test harness's later-set DB is
// honored; the use-case reports Available()==false when no DB is wired.
func NewProfiles(db func() *database.DB, cfg *config.Config, path string) *catalog.Service {
	return catalog.NewService(profilesStore{db: db}, profilesLiveConfig{cfg: cfg, path: path})
}
