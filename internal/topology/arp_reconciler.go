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

// ARPReconcilerName is the engine identifier.
const ARPReconcilerName = "topology-arp-reconciler"

// arpHighWaterKey is the settings key holding the latest ObservedAt
// the ARP reconciler has already processed.
const arpHighWaterKey = "topology.arp.high_water"

// arpStore is the narrowed repo surface the ARP reconciler uses.
// In addition to writing bindings, the reconciler joins MAC values
// against topology_nodes.primary_mac to backfill primary_ip on
// nodes whose chassis MAC matches the binding — the bridge between
// L2 and L3 identity.
type arpStore interface {
	NodeIDForTarget(ctx context.Context, clientID, targetID string) (string, error)
	NodeIDForMAC(ctx context.Context, clientID, mac string) (string, error)
	UpsertARPBinding(ctx context.Context, b *database.TopologyARPBinding) error
	SetNodePrimaryIP(ctx context.Context, nodeID, ip string) error
}

// ARPReconciler turns arp observations into topology_arp_bindings
// rows and backfills node.primary_ip when a binding's MAC matches
// a known node. Two operations per binding — keep them in one
// reconciler so the high-water mark stays coherent.
type ARPReconciler struct {
	obs      observationsReader
	store    arpStore
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

// ARPConfig wires the reconciler.
type ARPConfig struct {
	Observations observationsReader
	Store        arpStore
	Settings     settingsKV
	Logger       *slog.Logger
	Now          func() time.Time
	Interval     time.Duration
}

// NewARPReconciler returns an unstarted reconciler.
func NewARPReconciler(cfg ARPConfig) (*ARPReconciler, error) {
	if cfg.Observations == nil {
		return nil, errors.New("topology: arp Observations required")
	}
	if cfg.Store == nil {
		return nil, errors.New("topology: arp Store required")
	}
	if cfg.Settings == nil {
		return nil, errors.New("topology: arp Settings required")
	}
	d := applyDefaults(cfg.Logger, cfg.Now, cfg.Interval)
	return &ARPReconciler{
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
func (*ARPReconciler) Name() string { return ARPReconcilerName }

// Status implements [engine.Reporter].
func (r *ARPReconciler) Status() engine.Status {
	r.mu.Lock()
	stopped := r.stopped
	r.mu.Unlock()
	return r.tracker.status(stopped)
}

// Start kicks off the reconcile loop. Idempotent.
func (r *ARPReconciler) Start(ctx context.Context) error {
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
	r.logger.InfoContext(ctx, "arp reconciler started", "interval", r.interval)
	return nil
}

// Stop terminates the reconcile loop. Honors ctx deadline.
func (r *ARPReconciler) Stop(ctx context.Context) error {
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
	r.logger.InfoContext(ctx, "arp reconciler stopped")
	return nil
}

func (r *ARPReconciler) loop(ctx context.Context) {
	defer r.wg.Done()
	t := time.NewTicker(r.interval)
	defer t.Stop()
	if err := r.ReconcileOnce(ctx); err != nil {
		r.logger.WarnContext(ctx, "arp reconcile failed", "error", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := r.ReconcileOnce(ctx); err != nil {
				r.logger.WarnContext(ctx, "arp reconcile failed", "error", err)
			}
		}
	}
}

// ReconcileOnce processes one batch of new arp observations.
func (r *ARPReconciler) ReconcileOnce(ctx context.Context) error {
	err := r.reconcileOnceInner(ctx)
	r.tracker.recordTick(err)
	return err
}

func (r *ARPReconciler) reconcileOnceInner(ctx context.Context) error {
	since, err := r.loadHighWater(ctx)
	if err != nil {
		return fmt.Errorf("load high-water: %w", err)
	}
	observations, err := r.obs.List(ctx, database.ListOptions{
		Kind:  "arp",
		Since: since,
		Limit: defaultBatchLimit,
	})
	if err != nil {
		return fmt.Errorf("list arp observations: %w", err)
	}
	if len(observations) == 0 {
		return nil
	}

	var maxObservedAt time.Time
	bindings := 0
	backfills := 0
	for _, obs := range observations {
		if obs.ObservedAt.After(maxObservedAt) {
			maxObservedAt = obs.ObservedAt
		}
		sourceNodeID, lookupErr := r.store.NodeIDForTarget(ctx, obs.ClientID, obs.TargetID)
		if lookupErr != nil {
			// Source not yet known — A4.1 sysinfo reconciler will
			// fill in the mapping shortly; skip for now.
			continue
		}
		b, bf := r.applyObservation(ctx, obs, sourceNodeID)
		bindings += b
		backfills += bf
	}
	r.logger.DebugContext(ctx, "arp reconcile pass",
		"batch", len(observations), "bindings", bindings, "backfills", backfills,
		"max_observed_at", maxObservedAt)

	if !maxObservedAt.IsZero() {
		if saveErr := r.saveHighWater(ctx, maxObservedAt); saveErr != nil {
			return fmt.Errorf("save high-water: %w", saveErr)
		}
	}
	return nil
}

// arpPayload mirrors the arp.Observation JSON.
type arpPayload struct {
	Entries []arpEntry `json:"Entries"`
}

type arpEntry struct {
	IfIndex    uint32 `json:"IfIndex"`
	IPAddress  string `json:"IPAddress"`
	MACAddress string `json:"MACAddress"`
	MediaType  int    `json:"MediaType"`
}

// applyObservation decodes the arp payload, upserts one binding per
// entry, and backfills primary_ip on any node whose primary_mac
// matches an entry's MAC. Returns (bindingCount, backfillCount).
func (r *ARPReconciler) applyObservation(
	ctx context.Context,
	obs *database.SNMPObservation,
	sourceNodeID string,
) (int, int) {
	var p arpPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &p); err != nil {
		r.logger.WarnContext(ctx, "arp: unmarshal payload failed",
			"target_id", obs.TargetID, "error", err)
		return 0, 0
	}
	bindings, backfills := 0, 0
	for _, e := range p.Entries {
		if e.IPAddress == "" || e.MACAddress == "" {
			continue
		}
		binding := &database.TopologyARPBinding{
			ClientID:     obs.ClientID,
			SourceNodeID: sourceNodeID,
			IfIndex:      e.IfIndex,
			IPAddress:    e.IPAddress,
			MACAddress:   e.MACAddress,
			MediaType:    e.MediaType,
			LastSeen:     obs.ObservedAt,
		}
		if upsertErr := r.store.UpsertARPBinding(ctx, binding); upsertErr != nil {
			r.logger.WarnContext(ctx, "arp: binding upsert failed",
				"target_id", obs.TargetID, "ip", e.IPAddress, "error", upsertErr)
			continue
		}
		bindings++

		// L2 -> L3 identity bridge: if this binding's MAC matches a
		// node's primary_mac, that node's primary_ip is this binding's
		// IP. The fat-Node finally has a network-layer identity.
		matchedNodeID, lookupErr := r.store.NodeIDForMAC(ctx, obs.ClientID, e.MACAddress)
		if lookupErr != nil {
			continue
		}
		if setErr := r.store.SetNodePrimaryIP(ctx, matchedNodeID, e.IPAddress); setErr != nil {
			r.logger.WarnContext(ctx, "arp: primary_ip backfill failed",
				"node_id", matchedNodeID, "ip", e.IPAddress, "error", setErr)
			continue
		}
		backfills++
	}
	return bindings, backfills
}

func (r *ARPReconciler) loadHighWater(ctx context.Context) (time.Time, error) {
	raw, err := r.settings.GetWithDefault(ctx, arpHighWaterKey, "")
	if err != nil {
		return time.Time{}, err
	}
	if raw == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse arp high-water %q: %w", raw, err)
	}
	return parsed, nil
}

func (r *ARPReconciler) saveHighWater(ctx context.Context, t time.Time) error {
	return r.settings.Set(ctx, arpHighWaterKey, t.UTC().Format(time.RFC3339Nano))
}
