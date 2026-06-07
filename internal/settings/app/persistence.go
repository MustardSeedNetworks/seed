// Package settingsapp holds the settings application (use-case) layer
// (ADR-0016 strangle phase 3). It owns the orchestration that previously lived
// in the api.Server settings handlers — resolving the active profile and
// persisting the current configuration to it — behind narrow, consumer-defined
// ports, so handlers depend on a use-case instead of reaching into the database
// repositories and the config directly.
package settingsapp

import (
	"context"
	"errors"
	"fmt"
)

// ErrNoProfile signals that no active or default profile exists to persist to.
// The use-case treats it as a no-op: settings still live in the config file, so
// the absence of a profile row is not an error.
var ErrNoProfile = errors.New("no profile to persist settings to")

// ProfileStore is the narrow persistence surface the settings use-case needs.
// It is defined here at the consumer (ADR-0016 interface-segregation) and
// satisfied by an adapter that owns the database.Profile type, the database
// repositories, and the row read-modify-write.
type ProfileStore interface {
	// ActiveProfileID returns the configured active profile id, or "" when none
	// is set. A non-nil error routes the use-case to the default profile.
	ActiveProfileID(ctx context.Context) (string, error)
	// DefaultProfileID returns the default profile id, or ErrNoProfile when no
	// profile exists.
	DefaultProfileID(ctx context.Context) (string, error)
	// SaveProfileConfig persists configJSON onto the profile identified by id.
	SaveProfileConfig(ctx context.Context, id, configJSON string) error
}

// ConfigSource serializes the live configuration to a profile JSON blob.
type ConfigSource interface {
	ProfileJSON() (string, error)
}

// Persistence is the settings-persistence use-case: it writes the current
// configuration to the active profile. Handlers depend on it instead of
// reaching into the database repositories.
type Persistence struct {
	store ProfileStore
	cfg   ConfigSource
}

// NewPersistence builds the use-case over its narrow dependencies.
func NewPersistence(store ProfileStore, cfg ConfigSource) *Persistence {
	return &Persistence{store: store, cfg: cfg}
}

// SaveToActiveProfile persists current settings to the active (or default)
// profile. It is an idempotent no-op when no store is wired (no database) or no
// profile exists (ErrNoProfile) — settings still live in the config file. Any
// other failure (serialization, the row write) propagates, so a genuinely
// broken save surfaces as an error instead of being silently dropped.
func (p *Persistence) SaveToActiveProfile(ctx context.Context) error {
	if p == nil || p.store == nil {
		return nil
	}

	id, err := p.store.ActiveProfileID(ctx)
	if err != nil || id == "" {
		// No active profile configured (or its lookup failed) — fall back to the
		// default profile. A missing default is the legitimate "nothing to persist
		// to" case; any other error is real and propagates.
		id, err = p.store.DefaultProfileID(ctx)
		if err != nil {
			if errors.Is(err, ErrNoProfile) {
				return nil
			}
			return err
		}
	}

	configJSON, err := p.cfg.ProfileJSON()
	if err != nil {
		return fmt.Errorf("serialize settings for profile: %w", err)
	}

	return p.store.SaveProfileConfig(ctx, id, configJSON)
}
