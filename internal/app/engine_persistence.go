package app

// engine_persistence.go builds the persistence the database-backed engines
// (anomaly platform, telemetry, retention rollups, SNMP poller) and SNMP
// discovery's credential source read and write through, so the api wires each
// against its own ports and never names a repository (ADR-0020).

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	snmporchestrator "github.com/MustardSeedNetworks/seed/internal/polling/snmp/orchestrator"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/retention"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/telemetry"
)

// EnginePersistence is what each database-backed engine persists into.
type EnginePersistence struct {
	Anomalies anomaly.Store
	Telemetry telemetry.Store
	// Rollups are the retention engine's sources: probe_results, metrics and
	// flow_records.
	Rollups []retention.RollupSource
	// SNMPPoller carries the poller's stores; the caller adds the rest of the
	// orchestrator config.
	SNMPPoller snmporchestrator.Config
}

// NewEnginePersistence binds the engine ports to db.
func NewEnginePersistence(db *database.DB) EnginePersistence {
	return EnginePersistence{
		Anomalies: db.Anomalies(),
		Telemetry: db.Metrics(),
		Rollups: []retention.RollupSource{
			database.NewProbeRollupSource(db),
			database.NewMetricsRollupSource(db),
			database.NewFlowRollupSource(db),
		},
		SNMPPoller: snmporchestrator.Config{
			Targets:      db.PollingTargets(),
			Observations: db.SNMPObservations(),
			Rates:        db.Metrics(),
			Credentials:  db.DeviceCredentials(),
		},
	}
}

// NewClientIDLister lists the deployment's client ids from db, the only part
// of a client record discovery needs to decide whose credentials it may use.
func NewClientIDLister(db *database.DB) discovery.ClientLister {
	return clientIDLister{repo: db.Clients()}
}

type clientIDLister struct {
	repo *database.ClientRepository
}

func (l clientIDLister) ListClientIDs(ctx context.Context) ([]string, error) {
	clients, err := l.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(clients))
	for _, c := range clients {
		ids = append(ids, c.ID)
	}
	return ids, nil
}

// NewDiscoverySNMPCredentials builds discovery's vault-backed SNMP credential
// source (#2118) over db's clients and device credentials.
func NewDiscoverySNMPCredentials(
	db *database.DB,
	keyring discovery.SecretDecrypter,
	transport *config.SNMPConfig,
) (*discovery.VaultSNMPCredentials, error) {
	return discovery.NewVaultSNMPCredentials(NewClientIDLister(db), db.DeviceCredentials(), keyring, transport)
}
