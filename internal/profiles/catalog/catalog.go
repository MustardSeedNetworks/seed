// Package catalog holds the profiles application (use-case) layer (ADR-0020).
// It owns the CRUD, active-profile resolution, duplicate, and import/export
// orchestration that previously lived in the api.Server profile handlers,
// behind a narrow consumer-defined Store port — so handlers depend on a
// use-case instead of reaching into the database repositories. Handlers keep
// transport concerns (decode/encode, localized error mapping), license
// feature-gating, and the config-apply/SSE side effects.
package catalog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors. The use-case returns these so handlers can map each to the
// exact HTTP status + localized message the pre-strangle code used, without the
// app layer importing the database or i18n packages.
var (
	ErrNotFound      = errors.New("profile not found")
	ErrNameExists    = errors.New("profile name already exists")
	ErrNameRequired  = errors.New("profile name is required")
	ErrIDRequired    = errors.New("profile id is required")
	ErrDeleteDefault = errors.New("cannot delete the default profile")
	ErrDeleteActive  = errors.New("cannot delete the active profile")
	// ErrConflict is an optimistic-concurrency conflict: the caller's If-Match
	// ETag no longer matches the stored profile (a concurrent writer won).
	ErrConflict = errors.New("profile was modified by another writer")

	// ErrActiveLookup and the ErrNoActiveOrDefault/ErrDefaultLookup/
	// ErrActiveNotFound cases below each map to a distinct active-profile
	// handler message, preserving the pre-strangle responses.
	ErrActiveLookup      = errors.New("failed to read the active profile id")
	ErrNoActiveOrDefault = errors.New("no active or default profile")
	ErrDefaultLookup     = errors.New("failed to read the default profile")
	ErrActiveNotFound    = errors.New("active profile missing and no default")

	// ErrConfigApply wraps a fatal failure to persist the running config after an
	// activated profile's settings were applied. The transport layer maps it to a
	// 500 "failed to save" — the active-id is already set, only the save failed.
	ErrConfigApply = errors.New("failed to persist config after profile activation")
)

// Profile is the use-case profile model. The adapter maps it to/from
// database.Profile so the app layer stays free of persistence types.
type Profile struct {
	ID          string
	Name        string
	Description string
	ConfigJSON  string
	IsDefault   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	// RowVersion is the optimistic-concurrency token surfaced as the ETag. It is
	// opaque to the use-case (the ifMatch string is passed through to the Store).
	RowVersion int64
}

// NewProfile is the input for Create.
type NewProfile struct {
	Name        string
	Description string
	ConfigJSON  string
	IsDefault   bool
}

// ProfileUpdate is the input for Update. Name "" keeps the existing name and a
// nil ConfigJSON keeps the existing config; Description and IsDefault are always
// applied — mirroring the pre-strangle handler's partial-update semantics.
type ProfileUpdate struct {
	Name        string
	Description string
	ConfigJSON  *string
	IsDefault   bool
}

// ImportItem is one entry in an Import request.
type ImportItem struct {
	Name        string
	Description string
	ConfigJSON  string
}

// ImportResult tallies an Import run.
type ImportResult struct {
	Created int
	Updated int
	Skipped int
	Errors  []string
}

// Store is the persistence surface the profiles use-case needs, defined at the
// consumer (ADR-0016). The adapter satisfies it over the database Profiles and
// Settings repositories, mapping their ErrProfileNotFound/ErrProfileNameExists
// to ErrNotFound/ErrNameExists.
type Store interface {
	// Available reports whether the backing persistence is wired. A false result
	// means the profiles subsystem cannot serve any request (the 503 path).
	Available() bool
	List(ctx context.Context) ([]Profile, error)
	Get(ctx context.Context, id string) (Profile, error)
	// GetByName returns ok=false (and a nil error) when no profile has the name.
	GetByName(ctx context.Context, name string) (Profile, bool, error)
	GetDefault(ctx context.Context) (Profile, error)
	Create(ctx context.Context, p Profile) error
	// Update writes p. An empty ifMatch is unconditional; a non-empty ifMatch
	// makes the write optimistic-concurrency-checked against that version
	// (ETag), returning ErrConflict on mismatch.
	Update(ctx context.Context, p Profile, ifMatch string) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
	ActiveID(ctx context.Context) (string, error)
	SetActiveID(ctx context.Context, id string) error
}

// LiveConfig applies an activated profile's saved settings to the running config
// and persists them. A nil-backed applier is a no-op (no live config wired, e.g.
// tests). Apply returns ErrConfigApply only for a fatal persistence failure; a
// profile whose JSON cannot be parsed is logged and skipped by the adapter
// (best-effort), so a malformed saved config never fails activation.
type LiveConfig interface {
	Apply(ctx context.Context, profileJSON string) error
}

// Service is the profiles use-case.
type Service struct {
	store Store
	live  LiveConfig
}

// NewService builds the use-case over its Store port and an optional LiveConfig
// applier (nil = activation does not touch the running config).
func NewService(store Store, live LiveConfig) *Service {
	return &Service{store: store, live: live}
}

// Available reports whether the profiles subsystem can serve requests.
func (s *Service) Available() bool {
	return s.store.Available()
}

// List returns all profiles.
func (s *Service) List(ctx context.Context) ([]Profile, error) {
	return s.store.List(ctx)
}

// Count returns the number of stored profiles (used by the multi_client gate).
func (s *Service) Count(ctx context.Context) (int, error) {
	return s.store.Count(ctx)
}

// Get returns a profile by id (ErrNotFound when absent).
func (s *Service) Get(ctx context.Context, id string) (Profile, error) {
	return s.store.Get(ctx, id)
}

// Create validates and stores a new profile, returning the created record.
func (s *Service) Create(ctx context.Context, in NewProfile) (Profile, error) {
	if in.Name == "" {
		return Profile{}, ErrNameRequired
	}
	p := Profile{
		ID:          uuid.New().String(),
		Name:        in.Name,
		Description: in.Description,
		ConfigJSON:  in.ConfigJSON,
		IsDefault:   in.IsDefault,
	}
	if err := s.store.Create(ctx, p); err != nil {
		return Profile{}, err
	}
	return p, nil
}

// Update applies a partial update to an existing profile. When ifMatch is
// non-empty the write is optimistic-concurrency-checked against that ETag
// (ErrConflict on mismatch); an empty ifMatch performs an unconditional write,
// preserving the prior behavior. It re-reads the row so the returned profile
// carries the fresh updated_at (the new ETag).
func (s *Service) Update(ctx context.Context, id string, u ProfileUpdate, ifMatch string) (Profile, error) {
	p, err := s.store.Get(ctx, id)
	if err != nil {
		return Profile{}, err
	}
	if u.Name != "" {
		p.Name = u.Name
	}
	p.Description = u.Description
	if u.ConfigJSON != nil {
		p.ConfigJSON = *u.ConfigJSON
	}
	p.IsDefault = u.IsDefault

	if err = s.store.Update(ctx, p, ifMatch); err != nil {
		return Profile{}, err
	}

	// Re-read so the caller gets the persisted updated_at for the next ETag.
	if current, getErr := s.store.Get(ctx, id); getErr == nil {
		return current, nil
	}
	return p, nil
}

// Delete removes a profile, refusing to delete the default or active profile.
func (s *Service) Delete(ctx context.Context, id string) error {
	p, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if p.IsDefault {
		return ErrDeleteDefault
	}
	// A failed active-id read must not block deletion (best-effort, as before).
	if activeID, activeErr := s.store.ActiveID(ctx); activeErr == nil && activeID == id {
		return ErrDeleteActive
	}
	return s.store.Delete(ctx, id)
}

// ActiveProfile resolves the active profile, falling back to the default and
// self-healing a stale active-id pointer, mirroring the pre-strangle handler.
func (s *Service) ActiveProfile(ctx context.Context) (Profile, error) {
	activeID, err := s.store.ActiveID(ctx)
	if err != nil {
		return Profile{}, ErrActiveLookup
	}

	if activeID == "" {
		def, defErr := s.store.GetDefault(ctx)
		if defErr != nil {
			if errors.Is(defErr, ErrNotFound) {
				return Profile{}, ErrNoActiveOrDefault
			}
			return Profile{}, ErrDefaultLookup
		}
		return def, nil
	}

	p, err := s.store.Get(ctx, activeID)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return Profile{}, err
		}
		// Active profile was deleted — fall back to default and repoint.
		def, defErr := s.store.GetDefault(ctx)
		if defErr != nil {
			return Profile{}, ErrActiveNotFound
		}
		_ = s.store.SetActiveID(ctx, def.ID)
		return def, nil
	}
	return p, nil
}

// SetActiveProfile verifies the target exists, persists it as active, and
// returns it (the handler then applies its config and broadcasts the change).
func (s *Service) SetActiveProfile(ctx context.Context, id string) (Profile, error) {
	if id == "" {
		return Profile{}, ErrIDRequired
	}
	p, err := s.store.Get(ctx, id)
	if err != nil {
		return Profile{}, err
	}
	if setErr := s.store.SetActiveID(ctx, id); setErr != nil {
		return Profile{}, setErr
	}

	// Activating a profile applies its saved settings to the running config. The
	// applier is best-effort on a malformed saved config (logged and skipped) and
	// returns ErrConfigApply only when the post-apply persist fails — in which case
	// the profile is already active but the config save did not land.
	if p.ConfigJSON != "" && s.live != nil {
		if applyErr := s.live.Apply(ctx, p.ConfigJSON); applyErr != nil {
			return p, applyErr
		}
	}
	return p, nil
}

// Duplicate copies a source profile under a new name (defaulting to
// "<name> (Copy)"), retrying once with a timestamped name on a name collision.
func (s *Service) Duplicate(ctx context.Context, sourceID, requestedName string) (Profile, error) {
	source, err := s.store.Get(ctx, sourceID)
	if err != nil {
		return Profile{}, err
	}

	name := requestedName
	if name == "" {
		name = source.Name + " (Copy)"
	}
	dup := Profile{
		ID:          uuid.New().String(),
		Name:        name,
		Description: source.Description,
		ConfigJSON:  source.ConfigJSON,
		IsDefault:   false, // duplicates are never default
	}

	err = s.store.Create(ctx, dup)
	if err == nil {
		return dup, nil
	}
	if !errors.Is(err, ErrNameExists) {
		return Profile{}, err
	}
	// Retry with a timestamp suffix.
	dup.Name = fmt.Sprintf("%s (%s)", source.Name, time.Now().Format("2006-01-02 15:04"))
	if retryErr := s.store.Create(ctx, dup); retryErr != nil {
		return Profile{}, ErrNameExists
	}
	return dup, nil
}

// Import creates or (when overwrite is set) updates each profile, never marking
// imports as default. It collects per-item errors rather than failing the batch.
func (s *Service) Import(ctx context.Context, items []ImportItem, overwrite bool) ImportResult {
	result := ImportResult{Errors: make([]string, 0)}

	for i, item := range items {
		if item.Name == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Profile %d: name is required", i+1))
			result.Skipped++
			continue
		}

		// Matches the pre-strangle handler, which ignored a GetByName error and
		// branched only on presence.
		existing, found, _ := s.store.GetByName(ctx, item.Name)
		if found {
			s.importExisting(ctx, existing, item, overwrite, &result)
			continue
		}

		p := Profile{
			ID:          uuid.New().String(),
			Name:        item.Name,
			Description: item.Description,
			ConfigJSON:  item.ConfigJSON,
			IsDefault:   false,
		}
		if err := s.store.Create(ctx, p); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Profile '%s': failed to create - %v", item.Name, err))
			result.Skipped++
			continue
		}
		result.Created++
	}

	return result
}

func (s *Service) importExisting(
	ctx context.Context,
	existing Profile,
	item ImportItem,
	overwrite bool,
	result *ImportResult,
) {
	if !overwrite {
		result.Errors = append(
			result.Errors,
			fmt.Sprintf("Profile '%s': already exists (use overwrite=true to update)", item.Name),
		)
		result.Skipped++
		return
	}

	existing.Description = item.Description
	existing.ConfigJSON = item.ConfigJSON
	if err := s.store.Update(ctx, existing, ""); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Profile '%s': failed to update - %v", item.Name, err))
		result.Skipped++
		return
	}
	result.Updated++
}
