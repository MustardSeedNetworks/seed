package app

// update.go wires the composition root to the update-lifecycle application
// (use-case) service (ADR-0020): the self-update surface — release checking,
// status, download, apply, rollback, and configuration. The adapter below
// implements the narrow port declared in internal/update/lifecycle over the
// concrete update service, so the API handlers depend on the use-case instead
// of reaching into the service container directly. The service is resolved
// through a lazy accessor per call so a later-set value (the api test harness)
// is honored, and a nil service degrades every method to ErrUnavailable rather
// than panicking.

import (
	"context"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/update"
	"github.com/MustardSeedNetworks/seed/internal/update/lifecycle"
)

// NewUpdateLifecycle builds the update-lifecycle use-case (ADR-0020) over a
// lazy accessor for the concrete update service. A nil accessor result makes
// every method degrade to lifecycle.ErrUnavailable (the 503 path).
func NewUpdateLifecycle(svc func() *update.Service) *lifecycle.Service {
	a := updater{svc: svc}
	return lifecycle.NewService(a, a)
}

// updater implements lifecycle.Updater and lifecycle.ConfigStore over
// *update.Service, resolving it lazily. Methods beyond Available are only
// invoked by the use-case once Available reports true, so they assume a
// non-nil service.
type updater struct {
	svc func() *update.Service
}

func (a updater) Available() bool { return a.svc() != nil }

func (a updater) Check(ctx context.Context) (*update.UpdateInfo, error) {
	return a.svc().CheckForUpdate(ctx)
}

func (a updater) Info() *update.UpdateInfo    { return a.svc().GetUpdateInfo() }
func (a updater) Status() update.UpdateStatus { return a.svc().GetStatus() }
func (a updater) LastCheck() time.Time        { return a.svc().GetLastCheckTime() }
func (a updater) Downloaded() bool            { return a.svc().IsUpdateDownloaded() }

func (a updater) Download(ctx context.Context) error {
	_, err := a.svc().DownloadUpdate(ctx, nil)
	return err
}

func (a updater) Apply(ctx context.Context) error { return a.svc().ApplyUpdate(ctx) }
func (a updater) Rollback() error                 { return a.svc().Rollback() }

func (a updater) Config() update.UpdateConfig       { return a.svc().GetConfig() }
func (a updater) SetConfig(cfg update.UpdateConfig) { a.svc().SetConfig(cfg) }
