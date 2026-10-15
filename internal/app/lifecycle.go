package app

// lifecycle.go wires the composition root to the imported end-of-life data
// (P-D9, ADR-0020). The table lives in settings; an absent key means nothing
// has been imported, and Seed never fetches one.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/lifecycle"
)

// settingKeyLifecycleEoX holds the operator's imported EoX table as JSON.
const settingKeyLifecycleEoX = "lifecycle_eox"

// NewLifecycleStore builds the EoX table store over a lazy database accessor.
func NewLifecycleStore(persist func() *Persistence) lifecycle.Store {
	return lifecycleStore{db: lazyHandle(persist)}
}

type lifecycleStore struct {
	db func() *database.DB
}

// EoX returns the imported table. A stored table that no longer parses is an
// error rather than an empty table, so lost data is reported, not hidden.
func (a lifecycleStore) EoX(ctx context.Context) (*lifecycle.Table, error) {
	stored, err := a.db().Settings().GetValue(ctx, settingKeyLifecycleEoX)
	if err != nil {
		return nil, fmt.Errorf("read EoX table: %w", err)
	}
	if stored == "" {
		return lifecycle.New(nil)
	}
	return lifecycle.Parse([]byte(stored))
}

func (a lifecycleStore) SetEoX(ctx context.Context, table *lifecycle.Table) error {
	data, err := json.Marshal(table)
	if err != nil {
		return fmt.Errorf("encode EoX table: %w", err)
	}
	return a.db().Settings().Set(ctx, settingKeyLifecycleEoX, string(data))
}
