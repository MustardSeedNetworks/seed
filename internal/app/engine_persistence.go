package app

// engine_persistence.go builds the persistence the database-backed engines
// (anomaly platform, probe engine, telemetry, retention rollups, SNMP poller,
// topology reconcilers, alert pipelines) and SNMP discovery's credential
// source read and write through, so the api wires each against its own ports
// and never names a repository (ADR-0020).

import (
	"context"

	alertpipeline "github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	snmporchestrator "github.com/MustardSeedNetworks/seed/internal/polling/snmp/orchestrator"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/retention"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/telemetry"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// EnginePersistence is what each database-backed engine persists into.
type EnginePersistence struct {
	Anomalies anomaly.Store
	Probes    *ProbeStorage
	Telemetry telemetry.Store
	// Rollups are the retention engine's sources: probe_results, metrics and
	// flow_records.
	Rollups []retention.RollupSource
	// SNMPPoller carries the poller's stores; the caller adds the rest of the
	// orchestrator config.
	SNMPPoller snmporchestrator.Config
	// Topology and the alert pipelines carry each engine's stores; the caller
	// adds the logger and, for the pipelines, the alert writer.
	Topology          TopologyReconcilers
	ListenerAlerts    alertpipeline.ListenerConfig
	ObservationAlerts alertpipeline.ObservationConfig
}

// TopologyReconcilers is the config of each Stage A4 topology reconciler.
type TopologyReconcilers struct {
	SysInfo topology.Config
	IfTable topology.IfTableConfig
	Edge    topology.EdgeConfig
	ARP     topology.ARPConfig
}

// NewEnginePersistence binds the engine ports to db.
func NewEnginePersistence(db *database.DB) EnginePersistence {
	obs := db.SNMPObservations()
	topo := db.Topology()
	settings := db.Settings()
	suppressions := alertpipeline.NewDBSuppressionStore(db.AlertSuppressions())
	return EnginePersistence{
		Anomalies: db.Anomalies(),
		Probes:    NewProbeStorage(db),
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
		Topology: TopologyReconcilers{
			SysInfo: topology.Config{Observations: obs, Nodes: topo, Settings: settings},
			IfTable: topology.IfTableConfig{Observations: obs, Store: topo, Settings: settings},
			Edge:    topology.EdgeConfig{Observations: obs, Store: topo, Settings: settings},
			ARP:     topology.ARPConfig{Observations: obs, Store: topo, Settings: settings},
		},
		ListenerAlerts: alertpipeline.ListenerConfig{
			Events:       db.ListenerEvents(),
			Settings:     settings,
			AlertRules:   db.AlertRules(),
			Suppressions: suppressions,
		},
		ObservationAlerts: alertpipeline.ObservationConfig{
			Observations: obs,
			Settings:     settings,
			Suppressions: suppressions,
		},
	}
}

// lazyRepo resolves a repository from the lazily read database, nil while no
// database is wired (the api test harness), so the adapters over it degrade
// instead of panicking.
func lazyRepo[R any](db func() *database.DB, repo func(*database.DB) *R) func() *R {
	return func() *R {
		if d := db(); d != nil {
			return repo(d)
		}
		return nil
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
