package database_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSNMPEngineKeepsIDAndCountsBoots: the first boot stores the offered
// engine ID, and every later boot keeps it and increments boots, so senders'
// localized keys stay valid and the boots counter never goes backwards.
func TestSNMPEngineKeepsIDAndCountsBoots(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.SNMPEngine()

	first := []byte("\x80\x01\x06\xb1\x05first-engine")
	id, boots, err := repo.BootSNMPEngine(ctx, first)
	require.NoError(t, err)
	require.Equal(t, first, id)
	require.Equal(t, uint32(1), boots)

	id, boots, err = repo.BootSNMPEngine(ctx, []byte("\x80\x01\x06\xb1\x05other-engine"))
	require.NoError(t, err)
	require.Equal(t, first, id, "a later boot must keep the stored engine ID")
	require.Equal(t, uint32(2), boots)
}

// TestSNMPEngineBootsLatch: boots stops at RFC 3414's maximum rather than
// wrapping or failing the CHECK constraint.
func TestSNMPEngineBootsLatch(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := db.SNMPEngine()

	_, _, err := repo.BootSNMPEngine(ctx, []byte("\x80\x01\x06\xb1\x05latched"))
	require.NoError(t, err)
	_, err = db.Exec(ctx, `UPDATE snmp_engine SET boots = 2147483647`)
	require.NoError(t, err)

	_, boots, err := repo.BootSNMPEngine(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, uint32(2147483647), boots)
}
