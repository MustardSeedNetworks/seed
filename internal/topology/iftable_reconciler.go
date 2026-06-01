package topology

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/engine"
)

// IfTableReconcilerName is the engine identifier.
const IfTableReconcilerName = "topology-iftable-reconciler"

// ifTableHighWaterKey is the settings key holding the latest
// ObservedAt this reconciler has already processed.
const ifTableHighWaterKey = "topology.iftable.high_water"

// nodeIfaceUpserter is the narrowed surface the iftable reconciler
// needs. The sysinfo reconciler already populated the
// (client, target) -> node_id mapping; here we look it up + write
// one row per interface.
type nodeIfaceUpserter interface {
	NodeIDForTarget(ctx context.Context, clientID, targetID string) (string, error)
	UpsertInterface(ctx context.Context, iface *database.TopologyInterface) error
}

// IfTableReconciler turns if_table observations into
// topology_interfaces rows attached to their parent node. Skips
// observations whose target_id has no node mapping yet — those land
// on the next pass after sysinfo reconciliation catches up.
type IfTableReconciler struct {
	obs      observationsReader
	store    nodeIfaceUpserter
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

// IfTableConfig wires the iftable reconciler. Mirrors Config for
// consistency across reconcilers.
type IfTableConfig struct {
	Observations observationsReader
	Store        nodeIfaceUpserter
	Settings     settingsKV
	Logger       *slog.Logger
	Now          func() time.Time
	Interval     time.Duration
}

// NewIfTableReconciler returns an unstarted iftable reconciler.
func NewIfTableReconciler(cfg IfTableConfig) (*IfTableReconciler, error) {
	if cfg.Observations == nil {
		return nil, errors.New("topology: iftable Observations required")
	}
	if cfg.Store == nil {
		return nil, errors.New("topology: iftable Store required")
	}
	if cfg.Settings == nil {
		return nil, errors.New("topology: iftable Settings required")
	}
	d := applyDefaults(cfg.Logger, cfg.Now, cfg.Interval)
	return &IfTableReconciler{
		obs:      cfg.Observations,
		store:    cfg.Store,
		settings: cfg.Settings,
		logger:   d.logger,
		now:      d.now,
		interval: d.interval,
		tracker:  newTickTracker(d.interval, d.now),
	}, nil
}

// Name implements [engine.Engine].
func (*IfTableReconciler) Name() string { return IfTableReconcilerName }

// Status implements [engine.Reporter].
func (r *IfTableReconciler) Status() engine.Status {
	r.mu.Lock()
	stopped := r.stopped
	r.mu.Unlock()
	return r.tracker.status(stopped)
}

// Start kicks off the reconcile loop. Idempotent.
func (r *IfTableReconciler) Start(ctx context.Context) error {
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
	r.logger.InfoContext(ctx, "iftable reconciler started", "interval", r.interval)
	return nil
}

// Stop terminates the reconcile loop. Honors ctx deadline.
func (r *IfTableReconciler) Stop(ctx context.Context) error {
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
	r.logger.InfoContext(ctx, "iftable reconciler stopped")
	return nil
}

func (r *IfTableReconciler) loop(ctx context.Context) {
	defer r.wg.Done()
	t := time.NewTicker(r.interval)
	defer t.Stop()
	if err := r.ReconcileOnce(ctx); err != nil {
		r.logger.WarnContext(ctx, "iftable reconcile failed", "error", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := r.ReconcileOnce(ctx); err != nil {
				r.logger.WarnContext(ctx, "iftable reconcile failed", "error", err)
			}
		}
	}
}

// ReconcileOnce processes one batch of new if_table observations.
func (r *IfTableReconciler) ReconcileOnce(ctx context.Context) error {
	err := r.reconcileOnceInner(ctx)
	r.tracker.recordTick(err)
	return err
}

func (r *IfTableReconciler) reconcileOnceInner(ctx context.Context) error {
	since, err := r.loadHighWater(ctx)
	if err != nil {
		return fmt.Errorf("load high-water: %w", err)
	}
	observations, err := r.obs.List(ctx, database.ListOptions{
		Kind:  "if_table",
		Since: since,
		Limit: defaultBatchLimit,
	})
	if err != nil {
		return fmt.Errorf("list if_table observations: %w", err)
	}
	if len(observations) == 0 {
		return nil
	}

	var maxObservedAt time.Time
	rowsUpserted := 0
	for _, obs := range observations {
		if obs.ObservedAt.After(maxObservedAt) {
			maxObservedAt = obs.ObservedAt
		}
		nodeID, lookupErr := r.store.NodeIDForTarget(ctx, obs.ClientID, obs.TargetID)
		if lookupErr != nil {
			if errors.Is(lookupErr, database.ErrTopologyNodeNotFound) {
				r.logger.DebugContext(ctx, "iftable: target has no node mapping yet, skipping",
					"target_id", obs.TargetID)
				continue
			}
			r.logger.WarnContext(ctx, "iftable: target -> node lookup failed",
				"target_id", obs.TargetID, "error", lookupErr)
			continue
		}
		rowsUpserted += r.applyObservation(ctx, obs, nodeID)
	}
	r.logger.DebugContext(ctx, "iftable reconcile pass",
		"batch", len(observations), "rows", rowsUpserted, "max_observed_at", maxObservedAt)

	if !maxObservedAt.IsZero() {
		if saveErr := r.saveHighWater(ctx, maxObservedAt); saveErr != nil {
			return fmt.Errorf("save high-water: %w", saveErr)
		}
	}
	return nil
}

// ifTablePayload mirrors the iftable.Observation JSON shape.
type ifTablePayload struct {
	Rows []ifRow `json:"Rows"`
}

// ifRow mirrors the iftable.Row JSON shape — keep field names
// aligned with internal/polling/snmp/collectors/iftable.Row.
type ifRow struct {
	IfIndex    uint32 `json:"IfIndex"`
	IfDescr    string `json:"IfDescr"`
	IfName     string `json:"IfName"`
	IfAlias    string `json:"IfAlias"`
	IfType     uint32 `json:"IfType"`
	IfAdmin    int    `json:"IfAdmin"`
	IfOper     int    `json:"IfOper"`
	IfPhysAddr string `json:"IfPhysAddr"`
	SpeedBps   uint64 `json:"SpeedBps"`
}

// applyObservation decodes the iftable payload and upserts one row
// per interface. Returns the count of upserts that succeeded.
func (r *IfTableReconciler) applyObservation(
	ctx context.Context,
	obs *database.SNMPObservation,
	nodeID string,
) int {
	var p ifTablePayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &p); err != nil {
		r.logger.WarnContext(ctx, "iftable: unmarshal payload failed",
			"target_id", obs.TargetID, "error", err)
		return 0
	}
	count := 0
	for _, row := range p.Rows {
		if row.IfIndex == 0 {
			continue
		}
		iface := &database.TopologyInterface{
			NodeID:        nodeID,
			IfIndex:       row.IfIndex,
			IfName:        row.IfName,
			IfDescr:       row.IfDescr,
			IfAlias:       row.IfAlias,
			IfType:        row.IfType,
			IfAdminStatus: row.IfAdmin,
			IfOperStatus:  row.IfOper,
			IfPhysAddr:    row.IfPhysAddr,
			SpeedBps:      row.SpeedBps,
			LastSeen:      obs.ObservedAt,
		}
		if upsertErr := r.store.UpsertInterface(ctx, iface); upsertErr != nil {
			r.logger.WarnContext(ctx, "iftable: interface upsert failed",
				"target_id", obs.TargetID, "if_index", row.IfIndex, "error", upsertErr)
			continue
		}
		count++
	}
	return count
}

func (r *IfTableReconciler) loadHighWater(ctx context.Context) (time.Time, error) {
	raw, err := r.settings.GetWithDefault(ctx, ifTableHighWaterKey, "")
	if err != nil {
		return time.Time{}, err
	}
	if raw == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse iftable high-water %q: %w", raw, err)
	}
	return parsed, nil
}

func (r *IfTableReconciler) saveHighWater(ctx context.Context, t time.Time) error {
	return r.settings.Set(ctx, ifTableHighWaterKey, t.UTC().Format(time.RFC3339Nano))
}
