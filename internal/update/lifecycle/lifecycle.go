// Package lifecycle holds the software-update application (use-case) layer
// (ADR-0020). It owns the self-update lifecycle that previously lived in the
// api.Server update handlers — checking for a new release, reporting updater
// status and the latest check result, gating download on an available update
// and apply on a completed download, rolling back, and applying partial
// configuration changes — behind a narrow consumer-defined port over the
// concrete update service. Handlers keep transport concerns: request decode,
// response shaping, and error-to-status mapping. The adapter satisfying the
// port lives in the composition root (internal/app) and resolves the service
// lazily, so a nil service degrades every method to ErrUnavailable (the
// pre-strangle 503) rather than panicking.
//
// The parent internal/update package is the domain core (checker, downloader,
// applier); this subpackage is the application layer that orchestrates it for
// the API, keeping the two cleanly separated.
package lifecycle

import (
	"context"
	"errors"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/update"
)

// Sentinel errors, mapped by handlers to the pre-strangle HTTP responses.
var (
	// ErrUnavailable signals the updater is not wired (handlers map it to
	// 503, the pre-strangle degraded behavior).
	ErrUnavailable = errors.New("update service not available")
	// ErrNoUpdate signals there is no available update to download (handlers
	// map it to 400).
	ErrNoUpdate = errors.New("no update available")
	// ErrNotDownloaded signals no update has been downloaded to apply
	// (handlers map it to 400).
	ErrNotDownloaded = errors.New("no update downloaded")
	// ErrInvalidInterval signals an unparsable or out-of-range check interval
	// in a configuration patch (handlers map it to 400).
	ErrInvalidInterval = errors.New("invalid check interval")
)

// minCheckInterval is the floor for the configurable update-check interval; a
// shorter interval would hammer the release host for no operational benefit.
const minCheckInterval = time.Minute

// Updater is the self-update surface the use-case drives, defined at the
// consumer (ADR-0020) and satisfied by an adapter over *update.Service in
// internal/app. Available reports whether the updater is wired; the remaining
// methods are only invoked once availability is confirmed.
type Updater interface {
	Available() bool
	Check(ctx context.Context) (*update.UpdateInfo, error)
	Info() *update.UpdateInfo
	Status() update.UpdateStatus
	LastCheck() time.Time
	Downloaded() bool
	Download(ctx context.Context) error
	Apply(ctx context.Context) error
	Rollback() error
}

// ConfigStore is the update-configuration surface the use-case reads and
// writes, segregated from Updater so configuration management stands apart
// from the update state machine. In internal/app one adapter satisfies both.
type ConfigStore interface {
	Available() bool
	Config() update.UpdateConfig
	SetConfig(cfg update.UpdateConfig)
}

// Service is the update-lifecycle use-case.
type Service struct {
	updater Updater
	config  ConfigStore
}

// NewService builds the use-case over its narrow dependencies.
func NewService(updater Updater, config ConfigStore) *Service {
	return &Service{updater: updater, config: config}
}

// Check queries the release host for an available update.
func (s *Service) Check(ctx context.Context) (*update.UpdateInfo, error) {
	if !s.updater.Available() {
		return nil, ErrUnavailable
	}
	return s.updater.Check(ctx)
}

// Status is the use-case read model for the updater's current state: the
// underlying operation status, when the last check ran (zero before the first
// check), whether a downloaded update is ready to apply, and whether operator
// action is required (an update is available but not yet downloaded).
type Status struct {
	Status         update.UpdateStatus
	LastCheck      time.Time
	Ready          bool
	RequiresAction bool
}

// Status reports the updater's current state.
func (s *Service) Status() (Status, error) {
	if !s.updater.Available() {
		return Status{}, ErrUnavailable
	}
	info := s.updater.Info()
	downloaded := s.updater.Downloaded()
	return Status{
		Status:         s.updater.Status(),
		LastCheck:      s.updater.LastCheck(),
		Ready:          downloaded,
		RequiresAction: info != nil && info.Available && !downloaded,
	}, nil
}

// Info returns the most recent update information, defaulting to a
// no-update-available zero value before the first check has run.
func (s *Service) Info() (*update.UpdateInfo, error) {
	if !s.updater.Available() {
		return nil, ErrUnavailable
	}
	if info := s.updater.Info(); info != nil {
		return info, nil
	}
	return &update.UpdateInfo{}, nil
}

// Download fetches the available update, returning ErrNoUpdate when no update
// is available to download.
func (s *Service) Download(ctx context.Context) error {
	if !s.updater.Available() {
		return ErrUnavailable
	}
	if info := s.updater.Info(); info == nil || !info.Available {
		return ErrNoUpdate
	}
	return s.updater.Download(ctx)
}

// Apply installs the downloaded update, returning ErrNotDownloaded when no
// update has been downloaded yet.
func (s *Service) Apply(ctx context.Context) error {
	if !s.updater.Available() {
		return ErrUnavailable
	}
	if !s.updater.Downloaded() {
		return ErrNotDownloaded
	}
	return s.updater.Apply(ctx)
}

// Rollback reverts to the previous version.
func (s *Service) Rollback() error {
	if !s.updater.Available() {
		return ErrUnavailable
	}
	return s.updater.Rollback()
}

// Config returns the current update configuration.
func (s *Service) Config() (update.UpdateConfig, error) {
	if !s.config.Available() {
		return update.UpdateConfig{}, ErrUnavailable
	}
	return s.config.Config(), nil
}

// ConfigPatch is a partial update to the update configuration; nil fields are
// left unchanged.
type ConfigPatch struct {
	Enabled           *bool
	CheckInterval     *string
	AutoDownload      *bool
	AutoApply         *bool
	IncludePrerelease *bool
}

// Configure applies a partial configuration patch and returns the resulting
// configuration. A CheckInterval that does not parse as a Go duration or is
// shorter than one minute yields ErrInvalidInterval and leaves the
// configuration unchanged.
func (s *Service) Configure(patch ConfigPatch) (update.UpdateConfig, error) {
	if !s.config.Available() {
		return update.UpdateConfig{}, ErrUnavailable
	}
	cfg := s.config.Config()
	if patch.Enabled != nil {
		cfg.Enabled = *patch.Enabled
	}
	if patch.CheckInterval != nil {
		interval, err := time.ParseDuration(*patch.CheckInterval)
		if err != nil || interval < minCheckInterval {
			return update.UpdateConfig{}, ErrInvalidInterval
		}
		cfg.CheckInterval = interval
	}
	if patch.AutoDownload != nil {
		cfg.AutoDownload = *patch.AutoDownload
	}
	if patch.AutoApply != nil {
		cfg.AutoApply = *patch.AutoApply
	}
	if patch.IncludePrerelease != nil {
		cfg.IncludePrerelease = *patch.IncludePrerelease
	}
	s.config.SetConfig(cfg)
	return cfg, nil
}
