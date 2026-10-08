package app_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/listener"
)

// TestListenerEventsPersist pins that a passive-ingress event published to
// the shared sink lands in listener_events, where alert rules read it.
func TestListenerEventsPersist(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)

	require.NoError(t, app.NewListenerPersistence(db).Events.Publish(ctx, listener.Event{
		Kind:       "syslog",
		ClientID:   "default",
		SourceAddr: "192.0.2.7:514",
		Payload:    json.RawMessage(`{"msg":"link down"}`),
	}))

	got, err := db.ListenerEvents().List(ctx, listener.EventListOptions{Kind: "syslog"})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "192.0.2.7:514", got[0].SourceAddr)
}
