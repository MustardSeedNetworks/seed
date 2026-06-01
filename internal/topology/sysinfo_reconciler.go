// Package topology reconciles snmp_observations into the fat-Node
// + edge topology graph. One reconciler per observation kind:
// sys_info builds identity (A4.1, here), if_table folds interface
// state (A4.2), lldp/cdp/fdp build neighbor edges (A4.3), arp
// attaches IP↔MAC bindings (A4.4).
//
// Each reconciler is an [engine.Engine] that polls
// snmp_observations on a scheduler tick, processes new rows since
// a persisted high-water mark, and upserts the relevant slice of
// topology state. The high-water mark lives in the settings table
// so a restart resumes where it left off.
package topology

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/engine"
)

// SysInfoReconcilerName is the engine identifier.
const SysInfoReconcilerName = "topology-sysinfo-reconciler"

// sysInfoHighWaterKey is the settings key holding the latest
// ObservedAt timestamp this reconciler has already processed.
// A restart resumes from here so we don't reprocess every prior
// observation on every cold start.
const sysInfoHighWaterKey = "topology.sysinfo.high_water"

// Tunables. Production defaults — tests override via config.
const (
	defaultBatchLimit = 500

	// minPollInterval guards against a misconfigured tick that would
	// hammer snmp_observations every millisecond.
	minPollInterval = 100 * time.Millisecond

	// defaultPollInterval matches the typical snmp poll cadence so
	// reconciliation runs at the same heartbeat as ingestion.
	defaultPollInterval = 30 * time.Second
)

// observationsReader is the narrowed surface the reconciler needs
// from the SNMP observations repo. Tests inject a fake.
type observationsReader interface {
	List(ctx context.Context, opts database.ListOptions) ([]*database.SNMPObservation, error)
}

// nodeUpserter is the narrowed surface for topology_nodes writes.
// UpsertTargetNode records the (client, target) -> node mapping so
// downstream reconcilers (if_table, lldp, arp, fdb, routing, bgp4)
// can resolve their observations to the right node without re-
// decoding sysinfo on every pass.
type nodeUpserter interface {
	Upsert(ctx context.Context, node *database.TopologyNode) (*database.TopologyNode, error)
	UpsertTargetNode(
		ctx context.Context,
		clientID, targetID, nodeID string,
		lastSeen time.Time,
	) error
}

// settingsKV is the high-water-mark store.
type settingsKV interface {
	GetWithDefault(ctx context.Context, key, defaultValue string) (string, error)
	Set(ctx context.Context, key, value string) error
}

// SysInfoReconciler turns sys_info observations into topology_nodes.
// Each observation maps to one node, deduped by identity_hash.
type SysInfoReconciler struct {
	obs      observationsReader
	nodes    nodeUpserter
	settings settingsKV
	logger   *slog.Logger
	now      func() time.Time
	interval time.Duration
	tracker  *tickTracker

	mu      sync.Mutex
	started bool
	stopped bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// Config wires the dependencies.
type Config struct {
	Observations observationsReader
	Nodes        nodeUpserter
	Settings     settingsKV
	Logger       *slog.Logger
	Now          func() time.Time
	Interval     time.Duration
}

// NewSysInfoReconciler returns an unstarted reconciler.
func NewSysInfoReconciler(cfg Config) (*SysInfoReconciler, error) {
	if cfg.Observations == nil {
		return nil, errors.New("topology: Observations required")
	}
	if cfg.Nodes == nil {
		return nil, errors.New("topology: Nodes required")
	}
	if cfg.Settings == nil {
		return nil, errors.New("topology: Settings required")
	}
	d := applyDefaults(cfg.Logger, cfg.Now, cfg.Interval)
	return &SysInfoReconciler{
		obs:      cfg.Observations,
		nodes:    cfg.Nodes,
		settings: cfg.Settings,
		logger:   d.logger,
		now:      d.now,
		interval: d.interval,
		tracker:  newTickTracker(d.interval, d.now),
	}, nil
}

// Name implements [engine.Engine].
func (*SysInfoReconciler) Name() string { return SysInfoReconcilerName }

// Status implements [engine.Reporter].
func (r *SysInfoReconciler) Status() engine.Status {
	r.mu.Lock()
	stopped := r.stopped
	r.mu.Unlock()
	return r.tracker.status(stopped)
}

// Start kicks off the reconcile loop. Idempotent.
func (r *SysInfoReconciler) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return nil
	}
	loopCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.started = true
	r.wg.Add(1)
	go r.loop(loopCtx)
	r.logger.InfoContext(ctx, "sysinfo reconciler started", "interval", r.interval)
	return nil
}

// Stop terminates the reconcile loop. Honors ctx deadline.
func (r *SysInfoReconciler) Stop(ctx context.Context) error {
	r.mu.Lock()
	if !r.started {
		r.mu.Unlock()
		return nil
	}
	r.started = false
	r.stopped = true
	if r.cancel != nil {
		r.cancel()
	}
	r.mu.Unlock()

	doneCh := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(doneCh)
	}()
	select {
	case <-doneCh:
	case <-ctx.Done():
		return ctx.Err()
	}
	r.logger.InfoContext(ctx, "sysinfo reconciler stopped")
	return nil
}

// loop ticks every r.interval and runs one ReconcileOnce pass.
func (r *SysInfoReconciler) loop(ctx context.Context) {
	defer r.wg.Done()
	t := time.NewTicker(r.interval)
	defer t.Stop()
	// Tick immediately so the first pass doesn't wait a full
	// interval for the loop's initial reconcile.
	if err := r.ReconcileOnce(ctx); err != nil {
		r.logger.WarnContext(ctx, "sysinfo reconcile failed", "error", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := r.ReconcileOnce(ctx); err != nil {
				r.logger.WarnContext(ctx, "sysinfo reconcile failed", "error", err)
			}
		}
	}
}

// ReconcileOnce processes one batch of new sys_info observations.
// Exposed for tests and for the engine loop alike.
func (r *SysInfoReconciler) ReconcileOnce(ctx context.Context) error {
	err := r.reconcileOnceInner(ctx)
	r.tracker.recordTick(err)
	return err
}

func (r *SysInfoReconciler) reconcileOnceInner(ctx context.Context) error {
	since, err := r.loadHighWater(ctx)
	if err != nil {
		return fmt.Errorf("load high-water: %w", err)
	}
	observations, err := r.obs.List(ctx, database.ListOptions{
		Kind:  "sys_info",
		Since: since,
		Limit: defaultBatchLimit,
	})
	if err != nil {
		return fmt.Errorf("list sys_info observations: %w", err)
	}
	if len(observations) == 0 {
		return nil
	}

	var maxObservedAt time.Time
	upserted := 0
	for _, obs := range observations {
		// observations are returned newest-first; track the max
		// across the batch for the new high-water mark.
		if obs.ObservedAt.After(maxObservedAt) {
			maxObservedAt = obs.ObservedAt
		}
		node, buildErr := r.buildNode(obs)
		if buildErr != nil {
			r.logger.WarnContext(ctx, "skip sysinfo observation",
				"target_id", obs.TargetID, "error", buildErr)
			continue
		}
		stored, upsertErr := r.nodes.Upsert(ctx, node)
		if upsertErr != nil {
			r.logger.WarnContext(ctx, "upsert topology_node failed",
				"target_id", obs.TargetID, "identity", node.IdentityHash, "error", upsertErr)
			continue
		}
		// Persist (client, target) -> node so A4.2+ reconcilers can
		// look up the node by target_id without re-decoding sysinfo.
		mapErr := r.nodes.UpsertTargetNode(
			ctx, obs.ClientID, obs.TargetID, stored.ID, obs.ObservedAt,
		)
		if mapErr != nil {
			r.logger.WarnContext(ctx, "upsert target_node mapping failed",
				"target_id", obs.TargetID, "node_id", stored.ID, "error", mapErr)
		}
		upserted++
	}
	r.logger.DebugContext(ctx, "sysinfo reconcile pass",
		"batch", len(observations), "upserted", upserted, "max_observed_at", maxObservedAt)

	if !maxObservedAt.IsZero() {
		// Save high-water = newest observation's timestamp.
		// Next pass uses Since = high-water so we re-read it only
		// if a duplicate row arrives (same observedAt).
		if saveErr := r.saveHighWater(ctx, maxObservedAt); saveErr != nil {
			return fmt.Errorf("save high-water: %w", saveErr)
		}
	}
	return nil
}

// sysInfoPayload mirrors the relevant subset of the sysinfo
// collector's Observation JSON. Keep field names aligned with
// internal/polling/snmp/collectors/sysinfo.Observation.
type sysInfoPayload struct {
	ClientID    string `json:"ClientID"`
	TargetID    string `json:"TargetID"`
	SysDescr    string `json:"SysDescr"`
	SysObjectID string `json:"SysObjectID"`
	SysName     string `json:"SysName"`
	SysLocation string `json:"SysLocation"`
	SysContact  string `json:"SysContact"`
}

// buildNode decodes the observation payload into a TopologyNode
// ready for upsert.
func (r *SysInfoReconciler) buildNode(
	obs *database.SNMPObservation,
) (*database.TopologyNode, error) {
	var p sysInfoPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &p); err != nil {
		return nil, fmt.Errorf("unmarshal sysinfo payload: %w", err)
	}
	if p.SysName == "" && p.SysObjectID == "" {
		// Without any identifying scalar there's no value in a node row;
		// downstream reconcilers (if_table, lldp) still attach against
		// the same identity once it's populated on a future poll.
		return nil, errors.New("no SysName / SysObjectID — observation has no identity")
	}

	hash := identityHashFor(obs.ClientID, p.SysObjectID, p.SysName, obs.TargetID)
	displayName := p.SysName
	if displayName == "" {
		displayName = obs.TargetID
	}

	metadata, _ := json.Marshal(map[string]string{
		"sys_descr":    p.SysDescr,
		"sys_location": p.SysLocation,
		"sys_contact":  p.SysContact,
	})

	node := &database.TopologyNode{
		ID:           "node-" + hash[:16],
		ClientID:     obs.ClientID,
		IdentityHash: hash,
		DisplayName:  displayName,
		DeviceType:   deviceTypeFromObjectID(p.SysObjectID),
		SysName:      p.SysName,
		LastSeen:     obs.ObservedAt,
		MetadataJSON: string(metadata),
	}
	return node, nil
}

// identityHashFor computes the merge key for a node. Same (client,
// sysObjectID, sysName) = same physical device. We include client_id
// so two clients can have devices with identical sysName + OID
// without collapsing into one topology row.
//
// Falls back to client_id + target_id when sysObjectID/sysName are
// missing — preserves identity at least within a single Seed install.
func identityHashFor(clientID, sysObjectID, sysName, targetID string) string {
	h := sha256.New()
	h.Write([]byte(clientID))
	h.Write([]byte{0x00})
	if sysObjectID != "" || sysName != "" {
		h.Write([]byte(sysObjectID))
		h.Write([]byte{0x00})
		h.Write([]byte(sysName))
	} else {
		h.Write([]byte(targetID))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// deviceTypeFromObjectID maps a few common Cisco/Juniper/Arista
// enterprise OIDs to coarse device categories. V1.0 ships a small
// lookup; future iterations replace with a MIB-driven extension
// table.
func deviceTypeFromObjectID(oid string) string {
	switch {
	case oid == "":
		return ""
	case len(oid) > len(ciscoPrefix) && oid[:len(ciscoPrefix)] == ciscoPrefix:
		return "cisco"
	case len(oid) > len(juniperPrefix) && oid[:len(juniperPrefix)] == juniperPrefix:
		return "juniper"
	case len(oid) > len(aristaPrefix) && oid[:len(aristaPrefix)] == aristaPrefix:
		return "arista"
	case len(oid) > len(linuxPrefix) && oid[:len(linuxPrefix)] == linuxPrefix:
		return "linux"
	case len(oid) > len(mikrotikPrefix) && oid[:len(mikrotikPrefix)] == mikrotikPrefix:
		return "mikrotik"
	}
	return "unknown"
}

const (
	ciscoPrefix    = "1.3.6.1.4.1.9."
	juniperPrefix  = "1.3.6.1.4.1.2636."
	aristaPrefix   = "1.3.6.1.4.1.30065."
	linuxPrefix    = "1.3.6.1.4.1.8072.3.2.10"
	mikrotikPrefix = "1.3.6.1.4.1.14988."
)

// loadHighWater reads the persisted timestamp; returns zero time
// when no row has been processed yet (first-run case).
func (r *SysInfoReconciler) loadHighWater(ctx context.Context) (time.Time, error) {
	raw, err := r.settings.GetWithDefault(ctx, sysInfoHighWaterKey, "")
	if err != nil {
		return time.Time{}, err
	}
	if raw == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse high-water %q: %w", raw, err)
	}
	return parsed, nil
}

func (r *SysInfoReconciler) saveHighWater(ctx context.Context, t time.Time) error {
	return r.settings.Set(ctx, sysInfoHighWaterKey, t.UTC().Format(time.RFC3339Nano))
}
