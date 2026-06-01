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

// EdgeReconcilerName is the engine identifier.
const EdgeReconcilerName = "topology-edge-reconciler"

// edgeHighWaterKey is the settings key holding the latest ObservedAt
// the edge reconciler has already processed across all neighbor kinds.
const edgeHighWaterKey = "topology.edge.high_water"

// neighborKinds enumerates every observation kind the edge
// reconciler consumes. lldp/cdp/fdp are wire-distinct discovery
// protocols that all converge on the same edge graph; reconciling
// across all three in one engine keeps the high-water mark coherent.
// Declared as a function (rather than a package var) so the linter
// doesn't flag a global; the caller pattern is `for _, k := range neighborKinds()`.
func neighborKinds() []string { return []string{"lldp", "cdp", "fdp"} }

// edgeStore is the narrowed repo surface the edge reconciler uses.
type edgeStore interface {
	NodeIDForTarget(ctx context.Context, clientID, targetID string) (string, error)
	NodeIDForSysName(ctx context.Context, clientID, sysName string) (string, error)
	UpsertLink(ctx context.Context, link *database.TopologyLink) error
}

// EdgeReconciler turns lldp/cdp/fdp observations into
// topology_links. Only emits edges when both endpoints map to a
// known topology node — orphan neighbors (where the remote isn't
// polled) are skipped with a debug log. Stage B may introduce
// "ghost" nodes for orphan neighbors once the operator UI needs
// them.
type EdgeReconciler struct {
	obs      observationsReader
	store    edgeStore
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

// EdgeConfig wires the edge reconciler.
type EdgeConfig struct {
	Observations observationsReader
	Store        edgeStore
	Settings     settingsKV
	Logger       *slog.Logger
	Now          func() time.Time
	Interval     time.Duration
}

// NewEdgeReconciler returns an unstarted reconciler.
func NewEdgeReconciler(cfg EdgeConfig) (*EdgeReconciler, error) {
	if cfg.Observations == nil {
		return nil, errors.New("topology: edge Observations required")
	}
	if cfg.Store == nil {
		return nil, errors.New("topology: edge Store required")
	}
	if cfg.Settings == nil {
		return nil, errors.New("topology: edge Settings required")
	}
	d := applyDefaults(cfg.Logger, cfg.Now, cfg.Interval)
	return &EdgeReconciler{
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
func (*EdgeReconciler) Name() string { return EdgeReconcilerName }

// Status implements [engine.Reporter].
func (r *EdgeReconciler) Status() engine.Status {
	r.mu.Lock()
	stopped := r.stopped
	r.mu.Unlock()
	return r.tracker.status(stopped)
}

// Start kicks off the reconcile loop. Idempotent.
func (r *EdgeReconciler) Start(ctx context.Context) error {
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
	r.logger.InfoContext(ctx, "edge reconciler started", "interval", r.interval)
	return nil
}

// Stop terminates the reconcile loop. Honors ctx deadline.
func (r *EdgeReconciler) Stop(ctx context.Context) error {
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
	r.logger.InfoContext(ctx, "edge reconciler stopped")
	return nil
}

func (r *EdgeReconciler) loop(ctx context.Context) {
	defer r.wg.Done()
	t := time.NewTicker(r.interval)
	defer t.Stop()
	if err := r.ReconcileOnce(ctx); err != nil {
		r.logger.WarnContext(ctx, "edge reconcile failed", "error", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := r.ReconcileOnce(ctx); err != nil {
				r.logger.WarnContext(ctx, "edge reconcile failed", "error", err)
			}
		}
	}
}

// ReconcileOnce processes one batch across all neighbor kinds.
// Iterating per-kind keeps the SQL filters indexed (snmp_observations
// has an idx on (client_id, kind, observed_at)).
func (r *EdgeReconciler) ReconcileOnce(ctx context.Context) error {
	err := r.reconcileOnceInner(ctx)
	r.tracker.recordTick(err)
	return err
}

func (r *EdgeReconciler) reconcileOnceInner(ctx context.Context) error {
	since, err := r.loadHighWater(ctx)
	if err != nil {
		return fmt.Errorf("load high-water: %w", err)
	}

	var maxObservedAt time.Time
	totalLinks := 0
	for _, kind := range neighborKinds() {
		count, kindMax, kindErr := r.reconcileKind(ctx, kind, since)
		if kindErr != nil {
			r.logger.WarnContext(ctx, "edge reconcile kind failed",
				"kind", kind, "error", kindErr)
			continue
		}
		totalLinks += count
		if kindMax.After(maxObservedAt) {
			maxObservedAt = kindMax
		}
	}
	r.logger.DebugContext(ctx, "edge reconcile pass",
		"links", totalLinks, "max_observed_at", maxObservedAt)

	if !maxObservedAt.IsZero() {
		if saveErr := r.saveHighWater(ctx, maxObservedAt); saveErr != nil {
			return fmt.Errorf("save high-water: %w", saveErr)
		}
	}
	return nil
}

// reconcileKind processes one observation kind. Returns the link
// count and the newest ObservedAt across the batch.
func (r *EdgeReconciler) reconcileKind(
	ctx context.Context,
	kind string,
	since time.Time,
) (int, time.Time, error) {
	observations, err := r.obs.List(ctx, database.ListOptions{
		Kind:  kind,
		Since: since,
		Limit: defaultBatchLimit,
	})
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("list %s observations: %w", kind, err)
	}

	var maxAt time.Time
	count := 0
	for _, obs := range observations {
		if obs.ObservedAt.After(maxAt) {
			maxAt = obs.ObservedAt
		}
		sourceNodeID, lookupErr := r.store.NodeIDForTarget(ctx, obs.ClientID, obs.TargetID)
		if lookupErr != nil {
			if errors.Is(lookupErr, database.ErrTopologyNodeNotFound) {
				continue
			}
			r.logger.WarnContext(ctx, "edge: source node lookup failed",
				"kind", kind, "target_id", obs.TargetID, "error", lookupErr)
			continue
		}
		count += r.applyObservation(ctx, obs, kind, sourceNodeID)
	}
	return count, maxAt, nil
}

// neighborRecord captures the fields needed to materialize one edge.
// Common to lldp + cdp + fdp; the per-kind decoder fills it in.
type neighborRecord struct {
	localInterface  string
	remoteName      string
	remoteInterface string
}

// applyObservation decodes the observation payload, resolves each
// neighbor's node, and upserts an edge. Returns the count of edges
// that landed.
func (r *EdgeReconciler) applyObservation(
	ctx context.Context,
	obs *database.SNMPObservation,
	kind string,
	sourceNodeID string,
) int {
	neighbors, decodeErr := decodeNeighbors(kind, obs.PayloadJSON)
	if decodeErr != nil {
		r.logger.WarnContext(ctx, "edge: decode payload failed",
			"kind", kind, "target_id", obs.TargetID, "error", decodeErr)
		return 0
	}
	count := 0
	for _, n := range neighbors {
		if n.remoteName == "" {
			continue
		}
		remoteNodeID, lookupErr := r.store.NodeIDForSysName(ctx, obs.ClientID, n.remoteName)
		if lookupErr != nil {
			// Skip orphan neighbors. A future stage may introduce
			// ghost nodes; V1.0 needs both endpoints already known.
			continue
		}
		linkID := linkIDFor(sourceNodeID, n.localInterface, remoteNodeID, n.remoteInterface)
		evidence, _ := json.Marshal(map[string]string{
			"kind":          kind,
			"source_target": obs.TargetID,
			"remote_name":   n.remoteName,
		})
		link := &database.TopologyLink{
			ID:              linkID,
			SourceNodeID:    sourceNodeID,
			TargetNodeID:    remoteNodeID,
			SourceInterface: n.localInterface,
			TargetInterface: n.remoteInterface,
			LinkType:        kind,
			Status:          "up",
			LastSeen:        obs.ObservedAt,
			EvidenceJSON:    string(evidence),
		}
		if upsertErr := r.store.UpsertLink(ctx, link); upsertErr != nil {
			r.logger.WarnContext(ctx, "edge: link upsert failed",
				"kind", kind, "link_id", linkID, "error", upsertErr)
			continue
		}
		count++
	}
	return count
}

// linkIDFor derives a stable ID from the sorted node-pair so that
// observations from either side of the cable upsert to the same
// row. Interfaces are intentionally NOT part of the ID — LLDP/CDP
// report local-port labels (ifIndex, descriptor, alias) differently
// across vendors, and two observations of the same physical cable
// often disagree on which label to use. Keying by node-pair only
// has the side-effect that parallel cables between the same two
// nodes collapse into one row; refinement to per-port granularity
// is tracked for Stage A4-bis once iftable-derived port identity
// is reliable across vendors.
func linkIDFor(sourceNodeID, _, remoteNodeID, _ string) string {
	a, b := sourceNodeID, remoteNodeID
	if a > b {
		a, b = b, a
	}
	h := sha256.New()
	h.Write([]byte(a))
	h.Write([]byte{0x00})
	h.Write([]byte(b))
	return "link-" + hex.EncodeToString(h.Sum(nil))[:16]
}

// lldpPayload mirrors the lldp.Observation JSON shape.
type lldpPayload struct {
	Neighbors []struct {
		LocalPortNum    uint32 `json:"LocalPortNum"`
		PortID          string `json:"PortID"`
		PortDescription string `json:"PortDescription"`
		SysName         string `json:"SysName"`
	} `json:"Neighbors"`
}

// cdpPayload mirrors the cdp.Observation JSON shape (also used for fdp).
type cdpPayload struct {
	Neighbors []struct {
		LocalIfIndex uint32 `json:"LocalIfIndex"`
		DeviceID     string `json:"DeviceID"`
		DevicePort   string `json:"DevicePort"`
	} `json:"Neighbors"`
}

// decodeNeighbors extracts the per-kind payload and returns a
// uniform slice of neighborRecord. Unknown kinds yield an error
// because the reconciler should only see kinds it advertised.
func decodeNeighbors(kind, raw string) ([]neighborRecord, error) {
	switch kind {
	case "lldp":
		var p lldpPayload
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			return nil, err
		}
		out := make([]neighborRecord, 0, len(p.Neighbors))
		for _, n := range p.Neighbors {
			out = append(out, neighborRecord{
				localInterface:  fmt.Sprintf("ifIndex-%d", n.LocalPortNum),
				remoteName:      n.SysName,
				remoteInterface: firstNonEmpty(n.PortDescription, n.PortID),
			})
		}
		return out, nil
	case "cdp", "fdp":
		var p cdpPayload
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			return nil, err
		}
		out := make([]neighborRecord, 0, len(p.Neighbors))
		for _, n := range p.Neighbors {
			out = append(out, neighborRecord{
				localInterface:  fmt.Sprintf("ifIndex-%d", n.LocalIfIndex),
				remoteName:      n.DeviceID,
				remoteInterface: n.DevicePort,
			})
		}
		return out, nil
	default:
		return nil, fmt.Errorf("edge reconciler: unknown kind %q", kind)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (r *EdgeReconciler) loadHighWater(ctx context.Context) (time.Time, error) {
	raw, err := r.settings.GetWithDefault(ctx, edgeHighWaterKey, "")
	if err != nil {
		return time.Time{}, err
	}
	if raw == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse edge high-water %q: %w", raw, err)
	}
	return parsed, nil
}

func (r *EdgeReconciler) saveHighWater(ctx context.Context, t time.Time) error {
	return r.settings.Set(ctx, edgeHighWaterKey, t.UTC().Format(time.RFC3339Nano))
}
