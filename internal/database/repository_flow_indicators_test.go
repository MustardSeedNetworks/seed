package database_test

import (
	"context"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/indicators"
)

func TestFlowIndicatorsPersist(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	list, err := db.FlowRecords().FlowIndicators(ctx)
	require.NoError(t, err)
	require.Zero(t, list.Len(), "Seed ships no indicator list")

	set, err := indicators.New([]string{"198.51.100.7", "203.0.113.0/24"})
	require.NoError(t, err)
	require.NoError(t, db.FlowRecords().SetFlowIndicators(ctx, set))

	stored, err := db.Settings().GetValue(ctx, database.SettingKeyFlowIndicators)
	require.NoError(t, err)
	reread, err := indicators.Parse([]byte(stored))
	require.NoError(t, err)
	require.Equal(t, []string{"198.51.100.7", "203.0.113.0/24"}, reread.Entries())

	list, err = db.FlowRecords().FlowIndicators(ctx)
	require.NoError(t, err)
	_, ok := list.Match(netip.MustParseAddr("203.0.113.9"))
	require.True(t, ok)
}

// A stored list that no longer parses is an error, so the collector logs
// that matching stopped instead of matching nothing in silence.
func TestFlowIndicatorsRefuseAnUnreadableStoredList(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	require.NoError(t, db.Settings().Set(ctx, database.SettingKeyFlowIndicators, `{"indicators":["10.0.0.1"]}`))

	_, err := db.FlowRecords().FlowIndicators(ctx)
	require.ErrorContains(t, err, "stored threat indicators")
}
