// Package flows is the use-case over the flows the collector
// (internal/listener/flow) stored (P-C1..P-C5, ADR-0020): the top talkers,
// conversations and applications over a window, the signature table that
// names each flow's application, and the threat indicator list each flow is
// checked against. Persistence is reached through the Store port, implemented
// by internal/database's FlowRecordsRepository and wired in internal/app.
package flows

import (
	"context"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/appid"
	"github.com/MustardSeedNetworks/seed/internal/indicators"
)

// Tier names the table a top-N read aggregates. The caller picks it from the
// licence horizons, as it does for history: raw for any window the raw
// horizon covers, a rollup table past it.
type Tier int

// Flow tiers, finest first.
const (
	// TierRaw aggregates flow_records, including the hour in progress.
	TierRaw Tier = iota
	// TierHourly reads flow_conversations_hourly; closed hours only.
	TierHourly
	// TierDaily reads flow_conversations_daily; closed days only.
	TierDaily
)

// Rank is the counter a top-N list is ordered by.
type Rank string

// Flow rankings, as the wire spells them.
const (
	RankBytes   Rank = "bytes"
	RankPackets Rank = "packets"
)

// Query is one top-N read: the tenant's flows in [From, To) on Tier, the
// Limit largest by By.
type Query struct {
	ClientID string
	Tier     Tier
	From, To time.Time
	By       Rank
	Limit    int
}

// Talker is one host's traffic over the window, sent and received.
type Talker struct {
	Addr    string `json:"addr"`
	Bytes   int64  `json:"bytes"`
	Packets int64  `json:"packets"`
}

// Conversation is the traffic between two hosts over one protocol, both
// directions summed. AddrA and AddrB are ordered as text so each pair has one
// row; the order carries no meaning.
type Conversation struct {
	AddrA    string `json:"addrA"`
	AddrB    string `json:"addrB"`
	Protocol int    `json:"protocol"`
	Bytes    int64  `json:"bytes"`
	Packets  int64  `json:"packets"`
}

// Application is one application's traffic over a window.
type Application struct {
	Name    string `json:"name"`
	Bytes   int64  `json:"bytes"`
	Packets int64  `json:"packets"`
}

// AppSignatures is the signature table in effect and whether the operator
// replaced the builtin one.
type AppSignatures struct {
	Table  *appid.Table
	Custom bool
}

// Store is the persistence port behind the Service.
type Store interface {
	TopTalkers(ctx context.Context, q Query) ([]Talker, error)
	TopConversations(ctx context.Context, q Query) ([]Conversation, error)
	TopApplications(ctx context.Context, q Query) ([]Application, error)
	AppSignatures(ctx context.Context) (*AppSignatures, error)
	SetAppSignatures(ctx context.Context, table *appid.Table) error
	ResetAppSignatures(ctx context.Context) error
	FlowIndicators(ctx context.Context) (*indicators.List, error)
	SetFlowIndicators(ctx context.Context, list *indicators.List) error
}

// Service answers the flow reads and settings from a Store.
type Service struct {
	store Store
}

// NewService builds the flow use-case over store.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// TopTalkers returns the hosts that sent and received the most. A flow counts
// towards both of its ends, once each.
func (s *Service) TopTalkers(ctx context.Context, q Query) ([]Talker, error) {
	return s.store.TopTalkers(ctx, q)
}

// TopConversations returns the host pairs, per protocol, with the most
// traffic in both directions.
func (s *Service) TopConversations(ctx context.Context, q Query) ([]Conversation, error) {
	return s.store.TopConversations(ctx, q)
}

// TopApplications returns the applications with the most traffic, as named
// when each flow was stored.
func (s *Service) TopApplications(ctx context.Context, q Query) ([]Application, error) {
	return s.store.TopApplications(ctx, q)
}

// AppSignatures returns the signature table in effect.
func (s *Service) AppSignatures(ctx context.Context) (*AppSignatures, error) {
	return s.store.AppSignatures(ctx)
}

// SetAppSignatures replaces the signature table; flows stored from now on
// are named by it, stored flows keep their names.
func (s *Service) SetAppSignatures(ctx context.Context, table *appid.Table) error {
	return s.store.SetAppSignatures(ctx, table)
}

// ResetAppSignatures returns to the builtin signature table.
func (s *Service) ResetAppSignatures(ctx context.Context) error {
	return s.store.ResetAppSignatures(ctx)
}

// Indicators returns the threat indicator list in effect.
func (s *Service) Indicators(ctx context.Context) (*indicators.List, error) {
	return s.store.FlowIndicators(ctx)
}

// SetIndicators replaces the threat indicator list; flows collected from now
// on are checked against it.
func (s *Service) SetIndicators(ctx context.Context, list *indicators.List) error {
	return s.store.SetFlowIndicators(ctx, list)
}
