package app

// flows.go wires the composition root to the flow use-case (P-C1..P-C5,
// ADR-0020). The adapter implements flows.Store over the database flow
// repository, resolving the handle lazily so a later-set value (the api test
// harness) is honored.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/appid"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/flows"
	"github.com/MustardSeedNetworks/seed/internal/indicators"
)

// NewFlows builds the flow use-case over a lazy database accessor.
func NewFlows(db func() *database.DB) *flows.Service {
	return flows.NewService(flowStore{db: db})
}

type flowStore struct {
	db func() *database.DB
}

func (a flowStore) TopTalkers(ctx context.Context, q flows.Query) ([]flows.Talker, error) {
	return a.db().FlowRecords().TopTalkers(ctx, q)
}

func (a flowStore) TopConversations(ctx context.Context, q flows.Query) ([]flows.Conversation, error) {
	return a.db().FlowRecords().TopConversations(ctx, q)
}

func (a flowStore) TopApplications(ctx context.Context, q flows.Query) ([]flows.Application, error) {
	return a.db().FlowRecords().TopApplications(ctx, q)
}

func (a flowStore) AppSignatures(ctx context.Context) (*flows.AppSignatures, error) {
	return a.db().FlowRecords().AppSignatures(ctx)
}

func (a flowStore) SetAppSignatures(ctx context.Context, table *appid.Table) error {
	return a.db().FlowRecords().SetAppSignatures(ctx, table)
}

func (a flowStore) ResetAppSignatures(ctx context.Context) error {
	return a.db().FlowRecords().ResetAppSignatures(ctx)
}

func (a flowStore) FlowIndicators(ctx context.Context) (*indicators.List, error) {
	return a.db().FlowRecords().FlowIndicators(ctx)
}

func (a flowStore) SetFlowIndicators(ctx context.Context, list *indicators.List) error {
	return a.db().FlowRecords().SetFlowIndicators(ctx, list)
}
