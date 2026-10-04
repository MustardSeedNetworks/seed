package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/indicators"
)

// repository_flow_indicators.go is the threat indicator list the flow
// collector checks flows against (P-C5). The list lives in settings; an
// absent key means no list, and Seed never fetches one.

// SettingKeyFlowIndicators holds the operator's indicator list as JSON.
const SettingKeyFlowIndicators = "flow_threat_indicators"

// FlowIndicators returns the indicator list in effect, empty when the
// operator has supplied none. A stored list that no longer parses is an
// error rather than an empty list: matching would otherwise stop without
// anyone saying so.
func (r *FlowRecordsRepository) FlowIndicators(ctx context.Context) (*indicators.List, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.indicators != nil {
		return r.indicators, nil
	}
	stored, err := r.db.Settings().GetValue(ctx, SettingKeyFlowIndicators)
	if err != nil {
		return nil, fmt.Errorf("read threat indicators: %w", err)
	}
	if stored == "" {
		r.indicators, err = indicators.New(nil)
		return r.indicators, err
	}
	list, err := indicators.Parse([]byte(stored))
	if err != nil {
		return nil, fmt.Errorf("stored threat indicators: %w", err)
	}
	r.indicators = list
	return list, nil
}

// SetFlowIndicators stores the operator's list; flows collected from now on
// are checked against it.
func (r *FlowRecordsRepository) SetFlowIndicators(ctx context.Context, list *indicators.List) error {
	data, err := json.Marshal(list)
	if err != nil {
		return fmt.Errorf("encode threat indicators: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if setErr := r.db.Settings().Set(ctx, SettingKeyFlowIndicators, string(data)); setErr != nil {
		return setErr
	}
	r.indicators = list
	return nil
}
