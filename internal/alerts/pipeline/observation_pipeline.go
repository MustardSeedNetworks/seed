package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
)

// degradedTickMultiplier is how many scan intervals can elapse
// before a pipeline's Status() reports degraded. Two ticks gives
// the loop one missed tick of grace before paging.
const degradedTickMultiplier = 2

// ObservationPipelineName is the engine identifier.
const ObservationPipelineName = "alert-observation-pipeline"

// observationHighWaterKey is the settings key holding the latest
// ObservedAt the observation pipeline has already processed.
const observationHighWaterKey = "alerts.observation.high_water"

// Storage-threshold defaults — operators may tune in a follow-up
// stage; V1.0 ships with sensible "alert before it bites" values.
const (
	storageHighPct = 85.0
	storageFullPct = 95.0
	// percentMultiplier converts a used/size ratio to a percentage.
	percentMultiplier = 100.0
)

// observationReader is the narrowed surface this pipeline needs.
type observationReader interface {
	List(ctx context.Context, opts observation.ListOptions) ([]*observation.SNMPObservation, error)
}

// ObservationPipeline consumes the snmp_observations stream, detects
// per-entity state transitions (interface oper up→down, BGP peer
// leaving established, filesystem crossing usage thresholds), and
// emits alerts via the existing alertWriter.
//
// State is tracked in memory per pipeline instance. Restarts lose
// the prior state, so the next observation after a restart re-fires
// an alert if the entity is still in the bad state — bounded by the
// shared suppression window from listener_pipeline.go.
type ObservationPipeline struct {
	obs         observationReader
	alerts      alertWriter
	settings    settingsKV
	logger      *slog.Logger
	now         func() time.Time
	interval    time.Duration
	suppression time.Duration
	replayDepth int

	mu       sync.Mutex
	started  bool
	stopped  bool
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	suppress suppressionStore

	// Per-scan status, updated under mu for engine.Reporter.
	lastTickAt time.Time
	lastError  string

	// Previous-state caches. Keyed by entity identifier
	// ("target/ifindex" or "target/peer-ip", etc.). Updated after
	// every observation; the diff drives the alert.
	ifaceState   map[string]int     // (target, ifindex) -> last ifOperStatus
	bgpState     map[string]int     // (target, peerAddr) -> last PeerState
	storageState map[string]float64 // (target, storage_index) -> last %
}

// defaultReplayDepth is how many recent observations per kind get
// pulled at startup to prime the state caches. Two = the previous
// state plus the current one, which is enough to detect the most
// recent transition without a deeper history walk.
const defaultReplayDepth = 2

// ObservationConfig wires the pipeline.
type ObservationConfig struct {
	Observations observationReader
	Alerts       alertWriter
	Settings     settingsKV
	Logger       *slog.Logger
	Now          func() time.Time
	Interval     time.Duration
	Suppression  time.Duration

	// Suppressions is the persistence backend; see ListenerConfig.
	Suppressions suppressionStore

	// ReplayDepth controls how many recent observations per kind
	// are read at Start to prime the state caches (#1381). Zero
	// uses defaultReplayDepth (2). Set to -1 to disable replay
	// (legacy in-memory behavior — fresh state every restart).
	ReplayDepth int
}

// NewObservationPipeline returns an unstarted pipeline.
func NewObservationPipeline(cfg ObservationConfig) (*ObservationPipeline, error) {
	if cfg.Observations == nil {
		return nil, errors.New("alerts: observation Observations required")
	}
	if cfg.Alerts == nil {
		return nil, errors.New("alerts: observation Alerts writer required")
	}
	if cfg.Settings == nil {
		return nil, errors.New("alerts: observation Settings required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	if cfg.Interval <= 0 {
		cfg.Interval = defaultInterval
	}
	if cfg.Interval < minInterval {
		cfg.Interval = minInterval
	}
	if cfg.Suppression <= 0 {
		cfg.Suppression = defaultSuppression
	}
	suppress := cfg.Suppressions
	if suppress == nil {
		suppress = newInMemorySuppressionStore()
	}
	replayDepth := cfg.ReplayDepth
	if replayDepth == 0 {
		replayDepth = defaultReplayDepth
	}
	return &ObservationPipeline{
		obs:          cfg.Observations,
		alerts:       cfg.Alerts,
		settings:     cfg.Settings,
		logger:       cfg.Logger,
		now:          cfg.Now,
		interval:     cfg.Interval,
		suppression:  cfg.Suppression,
		suppress:     suppress,
		replayDepth:  replayDepth,
		ifaceState:   make(map[string]int),
		bgpState:     make(map[string]int),
		storageState: make(map[string]float64),
	}, nil
}

// Name implements [engine.Engine].
func (*ObservationPipeline) Name() string { return ObservationPipelineName }

// Status implements [engine.Reporter]. State derives from how long
// it has been since the last scan completed: "ok" within
// degradedTickMultiplier*interval, "degraded" beyond that, and
// "stopped" after Stop has been called.
func (p *ObservationPipeline) Status() engine.Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	s := engine.Status{
		LastTickAt: p.lastTickAt,
		LastError:  p.lastError,
	}
	switch {
	case p.stopped:
		s.State = engine.StateStopped
	case p.lastTickAt.IsZero():
		s.State = engine.StateOK
	case p.now().Sub(p.lastTickAt) > degradedTickMultiplier*p.interval:
		s.State = engine.StateDegraded
	default:
		s.State = engine.StateOK
	}
	return s
}

// Start kicks off the scan loop. Idempotent.
//
// Before the loop kicks, Start primes the per-entity state caches
// from the most recent observations in the DB (#1381). Without this,
// a Seed restart would re-fire every "interface down" alert on the
// first scan because the state cache starts empty and any non-up
// reading looks like a transition. The replay is bounded by
// replayDepth so we don't pay for unbounded history at startup.
func (p *ObservationPipeline) Start(ctx context.Context) error {
	// Prime state outside the mu — primeFromHistory takes its own
	// locks on the per-kind state maps. Replay errors are non-fatal:
	// the worst case is a few spurious alerts on first scan, which
	// suppression then absorbs.
	if p.replayDepth > 0 {
		if err := p.primeFromHistory(ctx); err != nil {
			p.logger.WarnContext(ctx, "observation pipeline state replay failed",
				"error", err)
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started {
		return nil
	}
	loopCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.started = true
	p.wg.Add(1)
	go p.loop(loopCtx)
	p.logger.InfoContext(ctx, "observation alert pipeline started",
		"interval", p.interval, "replay_depth", p.replayDepth)
	return nil
}

// primeFromHistory reads recent observations and walks them through
// the same payload-decode logic the scan loop uses, but stops short
// of firing alerts. The point is to populate ifaceState / bgpState /
// storageState so the first real scan after restart compares against
// real previous values instead of zero.
func (p *ObservationPipeline) primeFromHistory(ctx context.Context) error {
	for _, kind := range observationKinds() {
		observations, err := p.obs.List(ctx, observation.ListOptions{
			Kind:  kind,
			Limit: p.replayDepth,
		})
		if err != nil {
			return fmt.Errorf("replay %s: %w", kind, err)
		}
		// Walk oldest-first so the most recent observation overwrites
		// older state — matches what the scan loop would have done.
		for _, obs := range slices.Backward(observations) {
			p.primeOne(kind, obs)
		}
	}
	return nil
}

// primeOne dispatches to the per-kind primer. Unknown kinds are
// silently skipped — they were filtered out by observationKinds()
// at the caller, but we double-check here for safety.
func (p *ObservationPipeline) primeOne(kind string, obs *observation.SNMPObservation) {
	switch kind {
	case "if_table":
		p.primeIfTable(obs)
	case "bgp4_mib":
		p.primeBGP(obs)
	case "host_resources":
		p.primeHostResources(obs)
	}
}

func (p *ObservationPipeline) primeIfTable(obs *observation.SNMPObservation) {
	var pay ifTableObsPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &pay); err != nil {
		return
	}
	for _, row := range pay.Rows {
		key := fmt.Sprintf("if/%s/%d", obs.TargetID, row.IfIndex)
		p.recordIface(key, row.IfOper)
	}
}

func (p *ObservationPipeline) primeBGP(obs *observation.SNMPObservation) {
	var pay bgpObsPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &pay); err != nil {
		return
	}
	for _, peer := range pay.Peers {
		key := fmt.Sprintf("bgp/%s/%s", obs.TargetID, peer.RemoteAddr)
		p.recordBGP(key, peer.State)
	}
}

func (p *ObservationPipeline) primeHostResources(obs *observation.SNMPObservation) {
	var pay hostResObsPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &pay); err != nil {
		return
	}
	for _, st := range pay.Storage {
		if st.SizeBytes == 0 {
			continue
		}
		pct := percentMultiplier * float64(st.UsedBytes) / float64(st.SizeBytes)
		key := fmt.Sprintf("storage/%s/%d", obs.TargetID, st.Index)
		p.recordStorage(key, pct)
	}
}

// Stop terminates the scan loop. Honors ctx deadline.
func (p *ObservationPipeline) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return nil
	}
	p.started = false
	p.stopped = true
	if p.cancel != nil {
		p.cancel()
	}
	p.mu.Unlock()

	doneCh := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(doneCh)
	}()
	select {
	case <-doneCh:
	case <-ctx.Done():
		return ctx.Err()
	}
	p.logger.InfoContext(ctx, "observation alert pipeline stopped")
	return nil
}

func (p *ObservationPipeline) loop(ctx context.Context) {
	defer p.wg.Done()
	t := time.NewTicker(p.interval)
	defer t.Stop()
	if err := p.ScanOnce(ctx); err != nil {
		p.logger.WarnContext(ctx, "observation alert scan failed", "error", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := p.ScanOnce(ctx); err != nil {
				p.logger.WarnContext(ctx, "observation alert scan failed", "error", err)
			}
		}
	}
}

// ScanOnce processes one batch across every observation kind that
// has a delta signal. iftable + bgp4_mib + host_resources are the
// V1.0 set; future kinds extend the per-kind dispatch.
func (p *ObservationPipeline) ScanOnce(ctx context.Context) error {
	err := p.scanOnceInner(ctx)
	p.recordScan(err)
	return err
}

// recordScan stamps lastTickAt + lastError for engine.Reporter.
// Called from ScanOnce on every pass; held briefly under mu.
func (p *ObservationPipeline) recordScan(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastTickAt = p.now()
	if err != nil {
		p.lastError = err.Error()
		return
	}
	p.lastError = ""
}

func (p *ObservationPipeline) scanOnceInner(ctx context.Context) error {
	since, err := p.loadHighWater(ctx)
	if err != nil {
		return fmt.Errorf("load high-water: %w", err)
	}

	var maxObservedAt time.Time
	alertsEmitted := 0
	for _, kind := range observationKinds() {
		count, kindMax, kindErr := p.scanKind(ctx, kind, since)
		if kindErr != nil {
			p.logger.WarnContext(ctx, "observation alert kind failed",
				"kind", kind, "error", kindErr)
			continue
		}
		alertsEmitted += count
		if kindMax.After(maxObservedAt) {
			maxObservedAt = kindMax
		}
	}
	p.logger.DebugContext(ctx, "observation alert pass",
		"alerts", alertsEmitted, "max_observed_at", maxObservedAt)

	if !maxObservedAt.IsZero() {
		if saveErr := p.saveHighWater(ctx, maxObservedAt); saveErr != nil {
			return fmt.Errorf("save high-water: %w", saveErr)
		}
	}
	return nil
}

// observationKinds enumerates the kinds with delta signals.
// Declared as a function (not a global) for lint compatibility +
// to keep adding kinds a one-line edit.
func observationKinds() []string {
	return []string{"if_table", "bgp4_mib", "host_resources"}
}

func (p *ObservationPipeline) scanKind(
	ctx context.Context, kind string, since time.Time,
) (int, time.Time, error) {
	observations, err := p.obs.List(ctx, observation.ListOptions{
		Kind: kind, Since: since, Limit: defaultBatch,
	})
	if err != nil {
		return 0, time.Time{}, err
	}
	var maxAt time.Time
	count := 0
	for _, obs := range observations {
		if obs.ObservedAt.After(maxAt) {
			maxAt = obs.ObservedAt
		}
		switch kind {
		case "if_table":
			count += p.evaluateIfTable(ctx, obs)
		case "bgp4_mib":
			count += p.evaluateBGP(ctx, obs)
		case "host_resources":
			count += p.evaluateHostResources(ctx, obs)
		}
	}
	return count, maxAt, nil
}

// ifTableObsPayload mirrors the iftable Observation. Subset of
// fields needed for the delta check.
type ifTableObsPayload struct {
	Rows []struct {
		IfIndex uint32 `json:"IfIndex"`
		IfName  string `json:"IfName"`
		IfAdmin int    `json:"IfAdmin"`
		IfOper  int    `json:"IfOper"`
	} `json:"Rows"`
}

const ifOperUp = 1

// evaluateIfTable detects oper status transitions. Fires "interface
// down" alerts when admin=up AND oper changes from up to anything
// else. Up→up and any change from initial unknown state do NOT fire
// (we need a "previous up" to define a transition).
func (p *ObservationPipeline) evaluateIfTable(
	ctx context.Context, obs *observation.SNMPObservation,
) int {
	var pay ifTableObsPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &pay); err != nil {
		return 0
	}
	count := 0
	for _, row := range pay.Rows {
		key := fmt.Sprintf("if/%s/%d", obs.TargetID, row.IfIndex)
		prev := p.lookupIface(key)
		p.recordIface(key, row.IfOper)

		// Transition: previously up, now not. Admin-down doesn't
		// alert (operator intentionally disabled the port).
		if prev != ifOperUp || row.IfOper == ifOperUp || row.IfAdmin != ifOperUp {
			continue
		}
		alert := &database.Alert{
			Type:     database.AlertTypeConnectivity,
			Severity: database.AlertSeverityWarning,
			Title:    fmt.Sprintf("Interface %s down on %s", row.IfName, obs.TargetID),
			Message: fmt.Sprintf("ifOperStatus transitioned from up to %d (ifIndex=%d, admin=up)",
				row.IfOper, row.IfIndex),
			Source:   obs.TargetID,
			Metadata: obs.PayloadJSON,
		}
		if p.fire(ctx, "iface.down", key, alert) {
			count++
		}
	}
	return count
}

// bgpObsPayload mirrors the bgp4 Observation peers subset.
type bgpObsPayload struct {
	Peers []struct {
		RemoteAddr string `json:"RemoteAddr"`
		State      int    `json:"State"`
		RemoteAS   uint32 `json:"RemoteAS"`
	} `json:"Peers"`
}

const bgpStateEstablished = 6

// evaluateBGP detects peers leaving the Established state.
func (p *ObservationPipeline) evaluateBGP(
	ctx context.Context, obs *observation.SNMPObservation,
) int {
	var pay bgpObsPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &pay); err != nil {
		return 0
	}
	count := 0
	for _, peer := range pay.Peers {
		key := fmt.Sprintf("bgp/%s/%s", obs.TargetID, peer.RemoteAddr)
		prev := p.lookupBGP(key)
		p.recordBGP(key, peer.State)

		// Transition: was Established, now isn't.
		if prev != bgpStateEstablished || peer.State == bgpStateEstablished {
			continue
		}
		alert := &database.Alert{
			Type:     database.AlertTypeConnectivity,
			Severity: database.AlertSeverityError,
			Title:    fmt.Sprintf("BGP peer %s left Established on %s", peer.RemoteAddr, obs.TargetID),
			Message: fmt.Sprintf("Peer state transitioned from 6 (Established) to %d, AS%d",
				peer.State, peer.RemoteAS),
			Source:   obs.TargetID,
			Metadata: obs.PayloadJSON,
		}
		if p.fire(ctx, "bgp.flap", key, alert) {
			count++
		}
	}
	return count
}

// hostResObsPayload mirrors the host_resources Observation storage
// subset. SizeBytes / UsedBytes come from the collector already
// pre-multiplied by allocation_units.
type hostResObsPayload struct {
	Storage []struct {
		Index       uint32 `json:"Index"`
		Description string `json:"Description"`
		SizeBytes   uint64 `json:"SizeBytes"`
		UsedBytes   uint64 `json:"UsedBytes"`
	} `json:"Storage"`
}

// evaluateHostResources fires when a filesystem crosses 85% (warning)
// or 95% (critical). The delta state tracks the last-seen %, so we
// fire only on the upward crossing (not every poll while above the
// threshold).
func (p *ObservationPipeline) evaluateHostResources(
	ctx context.Context, obs *observation.SNMPObservation,
) int {
	var pay hostResObsPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &pay); err != nil {
		return 0
	}
	count := 0
	for _, st := range pay.Storage {
		if st.SizeBytes == 0 {
			continue
		}
		pct := percentMultiplier * float64(st.UsedBytes) / float64(st.SizeBytes)
		key := fmt.Sprintf("storage/%s/%d", obs.TargetID, st.Index)
		prev := p.lookupStorage(key)
		p.recordStorage(key, pct)

		// Upward crossings: prev below threshold and now at-or-above.
		// Two thresholds means two possible alerts per observation.
		if prev < storageFullPct && pct >= storageFullPct {
			alert := &database.Alert{
				Type:     database.AlertTypeSystem,
				Severity: database.AlertSeverityCritical,
				Title:    fmt.Sprintf("Filesystem %s critical on %s", st.Description, obs.TargetID),
				Message:  fmt.Sprintf("Usage crossed %.0f%%: %.1f%% of %d bytes", storageFullPct, pct, st.SizeBytes),
				Source:   obs.TargetID,
				Metadata: obs.PayloadJSON,
			}
			if p.fire(ctx, "storage.critical", key, alert) {
				count++
			}
		} else if prev < storageHighPct && pct >= storageHighPct {
			alert := &database.Alert{
				Type:     database.AlertTypeSystem,
				Severity: database.AlertSeverityWarning,
				Title:    fmt.Sprintf("Filesystem %s high on %s", st.Description, obs.TargetID),
				Message:  fmt.Sprintf("Usage crossed %.0f%%: %.1f%% of %d bytes", storageHighPct, pct, st.SizeBytes),
				Source:   obs.TargetID,
				Metadata: obs.PayloadJSON,
			}
			if p.fire(ctx, "storage.high", key, alert) {
				count++
			}
		}
	}
	return count
}

// fire writes an alert if the (ruleID, entityKey) fingerprint isn't
// suppressed. Returns true when the alert was actually written.
func (p *ObservationPipeline) fire(
	ctx context.Context, ruleID, entityKey string, alert *database.Alert,
) bool {
	now := p.now()
	fingerprint := fingerprintFor(ruleID, entityKey, alert.Source)
	suppressed, suppErr := p.suppress.IsSuppressed(ctx, fingerprint, now)
	if suppErr != nil {
		p.logger.WarnContext(ctx, "suppression check failed",
			"rule", ruleID, "key", entityKey, "error", suppErr)
	}
	if suppressed {
		return false
	}
	if err := p.alerts.Create(ctx, alert); err != nil {
		p.logger.WarnContext(ctx, "alert create failed",
			"rule", ruleID, "key", entityKey, "error", err)
		return false
	}
	if markErr := p.suppress.Mark(ctx, fingerprint, ruleID, entityKey, now.Add(p.suppression)); markErr != nil {
		p.logger.WarnContext(ctx, "suppression mark failed",
			"rule", ruleID, "key", entityKey, "error", markErr)
	}
	return true
}

// state map accessors keep the lock surface narrow.
func (p *ObservationPipeline) lookupIface(key string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ifaceState[key]
}

func (p *ObservationPipeline) recordIface(key string, oper int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ifaceState[key] = oper
}

func (p *ObservationPipeline) lookupBGP(key string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.bgpState[key]
}

func (p *ObservationPipeline) recordBGP(key string, state int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.bgpState[key] = state
}

func (p *ObservationPipeline) lookupStorage(key string) float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.storageState[key]
}

func (p *ObservationPipeline) recordStorage(key string, pct float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.storageState[key] = pct
}

func (p *ObservationPipeline) loadHighWater(ctx context.Context) (time.Time, error) {
	raw, err := p.settings.GetWithDefault(ctx, observationHighWaterKey, "")
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

func (p *ObservationPipeline) saveHighWater(ctx context.Context, t time.Time) error {
	return p.settings.Set(ctx, observationHighWaterKey, t.UTC().Format(time.RFC3339Nano))
}
