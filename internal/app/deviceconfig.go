package app

// deviceconfig.go wires the composition root to the device configuration
// backup use-case (P-D1). The adapters resolve the database lazily, as the
// credential vault's does, so a nil database degrades the handlers to 503
// rather than panicking. They also translate the repository's host-and-port
// conflict into the use-case's validation error, since the use-case cannot
// import internal/database.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// NewDeviceConfigBackups builds the configuration backup use-case over a lazy
// database accessor, the keyring that decrypts vault passwords and encrypts
// stored configurations, the fetcher that logs in to devices
// (deviceconfig.SSHFetcher in production), and a lazy accessor for the alert
// write chain a configuration change is raised through.
func NewDeviceConfigBackups(
	persist func() *Persistence,
	cipher deviceconfig.Cipher,
	fetcher deviceconfig.Fetcher,
	alertWriter func() deviceconfig.AlertWriter,
) (*deviceconfig.Service, error) {
	db := lazyHandle(persist)
	return deviceconfig.NewService(deviceConfigRepo{db: db}, deviceConfigCreds{db: db}, cipher, fetcher,
		deviceConfigAlerts{writer: alertWriter})
}

// deviceConfigAlerts resolves the alert write chain when a change is raised:
// it is built with the alert pipelines, after the use-cases.
type deviceConfigAlerts struct {
	writer func() deviceconfig.AlertWriter
}

func (a deviceConfigAlerts) Create(ctx context.Context, alert *alerts.Alert) error {
	w := a.writer()
	if w == nil {
		return deviceconfig.ErrUnavailable
	}
	return w.Create(ctx, alert)
}

type deviceConfigCreds struct {
	db func() *database.DB
}

func (a deviceConfigCreds) Get(ctx context.Context, id, clientID string) (*polling.Credentials, error) {
	db := a.db()
	if db == nil {
		return nil, deviceconfig.ErrUnavailable
	}
	return db.DeviceCredentials().Get(ctx, id, clientID)
}

// deviceConfigRepo implements deviceconfig.Repository over the database.
type deviceConfigRepo struct {
	db func() *database.DB
}

func (a deviceConfigRepo) repo() (*database.DeviceConfigRepository, error) {
	db := a.db()
	if db == nil {
		return nil, deviceconfig.ErrUnavailable
	}
	return db.DeviceConfigs(), nil
}

func (a deviceConfigRepo) ListTargets(ctx context.Context, clientID string) ([]deviceconfig.Target, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.ListTargets(ctx, clientID)
}

func (a deviceConfigRepo) GetTarget(ctx context.Context, clientID, id string) (deviceconfig.Target, error) {
	repo, err := a.repo()
	if err != nil {
		return deviceconfig.Target{}, err
	}
	return repo.GetTarget(ctx, clientID, id)
}

func (a deviceConfigRepo) SaveTarget(ctx context.Context, t *deviceconfig.Target) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	if saveErr := repo.SaveTarget(ctx, t); errors.Is(saveErr, database.ErrDeviceConfigTargetConflict) {
		return deviceconfig.ValidationError{Msg: "another backup target already uses this host and port"}
	} else if saveErr != nil {
		return saveErr
	}
	return nil
}

func (a deviceConfigRepo) DeleteTarget(ctx context.Context, clientID, id string) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.DeleteTarget(ctx, clientID, id)
}

func (a deviceConfigRepo) SetPendingHostKey(ctx context.Context, clientID, id, fingerprint string) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.SetPendingHostKey(ctx, clientID, id, fingerprint)
}

func (a deviceConfigRepo) ConfirmHostKey(ctx context.Context, clientID, id, fingerprint string) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.ConfirmHostKey(ctx, clientID, id, fingerprint)
}

func (a deviceConfigRepo) ClearHostKey(ctx context.Context, clientID, id string) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.ClearHostKey(ctx, clientID, id)
}

func (a deviceConfigRepo) SetPushCredentials(ctx context.Context, clientID, id, credentialsID string) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.SetPushCredentials(ctx, clientID, id, credentialsID)
}

func (a deviceConfigRepo) InsertBackup(ctx context.Context, b *deviceconfig.Backup) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.InsertBackup(ctx, b)
}

func (a deviceConfigRepo) PruneBackups(ctx context.Context, clientID, targetID string, keep int) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.PruneBackups(ctx, clientID, targetID, keep)
}

func (a deviceConfigRepo) ListBackups(
	ctx context.Context, clientID, targetID string, limit int,
) ([]deviceconfig.Backup, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.ListBackups(ctx, clientID, targetID, limit)
}

func (a deviceConfigRepo) GetBackup(ctx context.Context, clientID, id string) (deviceconfig.Backup, error) {
	repo, err := a.repo()
	if err != nil {
		return deviceconfig.Backup{}, err
	}
	return repo.GetBackup(ctx, clientID, id)
}

func (a deviceConfigRepo) LatestConfig(
	ctx context.Context, clientID, targetID string, before time.Time,
) (deviceconfig.Backup, error) {
	repo, err := a.repo()
	if err != nil {
		return deviceconfig.Backup{}, err
	}
	return repo.LatestConfig(ctx, clientID, targetID, before)
}

// NewDeviceConfigPushes builds the configuration push use-case (P-D3) over the
// backup use-case, the database's plan store, and the dry runner
// (deviceconfig.SSHFetcher in production). Push events go to the database
// audit log; with no database wired the use-case's stores answer 503 first.
func NewDeviceConfigPushes(
	persist func() *Persistence, backups *deviceconfig.Service, dryRun deviceconfig.DryRunner,
) (*deviceconfig.Pushes, error) {
	db := lazyHandle(persist)
	return deviceconfig.NewPushes(backups, deviceConfigRepo{db: db}, dryRun,
		func(ctx context.Context, e deviceconfig.PushAuditEvent) error {
			d := db()
			if d == nil {
				return deviceconfig.ErrUnavailable
			}
			detail, err := json.Marshal(e.Detail)
			if err != nil {
				return fmt.Errorf("encode push audit detail: %w", err)
			}
			return d.RecordAuditLog(ctx, &database.AuditLogEntry{
				Action:       e.Action,
				User:         e.User,
				ResourceType: deviceconfig.PushAuditResource,
				ResourceID:   e.TargetID,
				NewValueJSON: string(detail),
				IPAddress:    e.Remote,
				UserAgent:    e.UserAgent,
			})
		})
}

func (a deviceConfigRepo) InsertPlan(ctx context.Context, p *deviceconfig.Plan) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.InsertPlan(ctx, p)
}

func (a deviceConfigRepo) GetPlan(ctx context.Context, clientID, id string) (deviceconfig.Plan, error) {
	repo, err := a.repo()
	if err != nil {
		return deviceconfig.Plan{}, err
	}
	return repo.GetPlan(ctx, clientID, id)
}

func (a deviceConfigRepo) ListPlans(
	ctx context.Context, clientID, targetID string, limit int,
) ([]deviceconfig.Plan, error) {
	repo, err := a.repo()
	if err != nil {
		return nil, err
	}
	return repo.ListPlans(ctx, clientID, targetID, limit)
}

func (a deviceConfigRepo) PrunePlans(ctx context.Context, clientID, targetID string, keep int) error {
	repo, err := a.repo()
	if err != nil {
		return err
	}
	return repo.PrunePlans(ctx, clientID, targetID, keep)
}
