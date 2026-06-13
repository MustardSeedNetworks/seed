// Package sink wires the eleven Stage A3 collector Publisher
// interfaces (sysinfo, iftable, lldp, cdp, fdp, arp, fdb, routing,
// hostresources, bgp4) to a single concrete sink that persists
// every observation into the snmp_observations table.
//
// One Sink instance implements every Publisher interface — that
// keeps the orchestrator wire-up to one line per collector
// (RegisterCollector(sysinfo.New(factory, sink, nil)), etc.)
// instead of N glue types.
//
// Sink serializes the typed Observation as JSON. Downstream
// consumers (Stage A4 topology reconciler, the listener pipeline,
// the admin UI) decode the payload back into the kind-specific
// shape. Single JSON column beats per-kind columns because adding a
// new collector kind doesn't touch the migration list.
package sink

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/arp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/bgp4"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/cdp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/fdb"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/hostresources"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/iftable"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/lldp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/routing"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/sysinfo"
)

// Kind constants mirror each collector's Name() so downstream
// consumers (Stage A4, listener pipeline) can filter snmp_observations
// rows without importing every collector package.
const (
	KindSysInfo       = "sys_info"
	KindIfTable       = "if_table"
	KindLLDP          = "lldp"
	KindCDP           = "cdp"
	KindFDP           = "fdp"
	KindARP           = "arp"
	KindFDB           = "fdb"
	KindRouting       = "routing"
	KindHostResources = "host_resources"
	KindBGP4          = "bgp4_mib"
)

// observationsStore is the narrowed surface the Sink needs from the
// database layer. Tests inject a fake.
type observationsStore interface {
	Insert(ctx context.Context, obs *observation.SNMPObservation) error
}

// Sink persists every kind of collector Observation into the
// snmp_observations table. One Sink instance implements every
// Publisher interface so the orchestrator wire-up stays trivial.
type Sink struct {
	store  observationsStore
	logger *slog.Logger
	now    func() time.Time
}

// New returns a Sink bound to store. Pass nil logger to use
// [slog.Default]; pass nil now to use [time.Now] in UTC.
func New(store observationsStore, logger *slog.Logger, now func() time.Time) *Sink {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Sink{store: store, logger: logger, now: now}
}

// persist is the common path every PublishX method funnels into.
// Serializing here (rather than per-kind) keeps the shape stable
// across all observation kinds — payload_json is always the typed
// Observation struct serialized verbatim, no envelope.
func (s *Sink) persist(
	ctx context.Context,
	kind, clientID, targetID string,
	observedAt time.Time,
	payload any,
) error {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		s.logger.ErrorContext(ctx, "snmp sink: marshal failed",
			"kind", kind, "target_id", targetID, "error", err)
		return fmt.Errorf("snmp sink: marshal %s: %w", kind, err)
	}
	obs := &observation.SNMPObservation{
		ClientID:    clientID,
		TargetID:    targetID,
		Kind:        kind,
		ObservedAt:  observedAt,
		PayloadJSON: string(jsonBytes),
		IngestedAt:  s.now(),
	}
	if storeErr := s.store.Insert(ctx, obs); storeErr != nil {
		s.logger.WarnContext(ctx, "snmp sink: insert failed",
			"kind", kind, "target_id", targetID, "error", storeErr)
		return fmt.Errorf("snmp sink: insert %s: %w", kind, storeErr)
	}
	return nil
}

// PublishSysInfo implements [sysinfo.Publisher].
func (s *Sink) PublishSysInfo(ctx context.Context, obs sysinfo.Observation) error {
	return s.persist(ctx, KindSysInfo, obs.ClientID, obs.TargetID, obs.ObservedAt, obs)
}

// PublishIfTable implements [iftable.Publisher].
func (s *Sink) PublishIfTable(ctx context.Context, obs iftable.Observation) error {
	return s.persist(ctx, KindIfTable, obs.ClientID, obs.TargetID, obs.ObservedAt, obs)
}

// PublishLLDP implements [lldp.Publisher].
func (s *Sink) PublishLLDP(ctx context.Context, obs lldp.Observation) error {
	return s.persist(ctx, KindLLDP, obs.ClientID, obs.TargetID, obs.ObservedAt, obs)
}

// PublishCDP implements [cdp.Publisher]. CDP and FDP both land here
// because the observation shape is identical; obs.TablePrefix
// distinguishes the two on read.
func (s *Sink) PublishCDP(ctx context.Context, obs cdp.Observation) error {
	kind := KindCDP
	if obs.TablePrefix != "" && obs.TablePrefix != cdp.DefaultTablePrefix {
		kind = KindFDP
	}
	return s.persist(ctx, kind, obs.ClientID, obs.TargetID, obs.ObservedAt, obs)
}

// PublishARP implements [arp.Publisher].
func (s *Sink) PublishARP(ctx context.Context, obs arp.Observation) error {
	return s.persist(ctx, KindARP, obs.ClientID, obs.TargetID, obs.ObservedAt, obs)
}

// PublishFDB implements [fdb.Publisher].
func (s *Sink) PublishFDB(ctx context.Context, obs fdb.Observation) error {
	return s.persist(ctx, KindFDB, obs.ClientID, obs.TargetID, obs.ObservedAt, obs)
}

// PublishRouting implements [routing.Publisher].
func (s *Sink) PublishRouting(ctx context.Context, obs routing.Observation) error {
	return s.persist(ctx, KindRouting, obs.ClientID, obs.TargetID, obs.ObservedAt, obs)
}

// PublishHostResources implements [hostresources.Publisher].
func (s *Sink) PublishHostResources(ctx context.Context, obs hostresources.Observation) error {
	return s.persist(ctx, KindHostResources, obs.ClientID, obs.TargetID, obs.ObservedAt, obs)
}

// PublishBGP4 implements [bgp4.Publisher].
func (s *Sink) PublishBGP4(ctx context.Context, obs bgp4.Observation) error {
	return s.persist(ctx, KindBGP4, obs.ClientID, obs.TargetID, obs.ObservedAt, obs)
}
