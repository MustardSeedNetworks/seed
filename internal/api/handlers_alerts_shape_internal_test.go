package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	alertcorrelation "github.com/MustardSeedNetworks/seed/internal/alerts/correlation"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
)

// TestAlertListResponseShape pins every key the inbox serves: the fixed keys
// on every alert, and each optional key present only when its value is set.
func TestAlertListResponseShape(t *testing.T) {
	s := newAlertsTestServer(t)
	ctx := context.Background()
	db := s.db()
	seedLinkFault(t, db, alertcorrelation.WrapWriter(db.Alerts(), alertcorrelation.Config{}))

	device := "02:00:00:00:00:07"
	require.NoError(
		t,
		db.Devices().Create(ctx, &database.Device{ID: device, IPAddress: "192.0.2.9", MACAddress: device}),
	)
	handled := &alerts.Alert{
		Type: alerts.TypeSecurity, Severity: alerts.SeverityCritical, Title: "Rogue DHCP server",
		Message: "192.0.2.9 answered DISCOVER", Source: "dhcp", DeviceID: &device,
		Metadata: `{"server":"192.0.2.9"}`,
	}
	require.NoError(t, db.Alerts().Create(ctx, handled))
	at := time.Date(2026, 10, 11, 21, 0, 0, 0, time.UTC)
	require.NoError(
		t,
		db.Alerts().RecordDelivery(ctx, handled.ID, alerts.ChannelWebhook, alerts.DeliveryDelivered, at, ""),
	)
	require.NoError(t, db.Alerts().Acknowledge(ctx, handled.ID, "alice"))
	require.NoError(t, db.Alerts().Resolve(ctx, handled.ID))

	w := httptest.NewRecorder()
	i18n.Middleware()(http.HandlerFunc(s.handleAlerts)).
		ServeHTTP(w, httptest.NewRequest(http.MethodGet, alertsPath, http.NoBody))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, []string{"alerts", "count"}, topLevelKeys(t, w.Body.Bytes()))
	var resp struct {
		Count  int               `json:"count"`
		Alerts []json.RawMessage `json:"alerts"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 3, resp.Count)

	fixed := []string{
		"acknowledged", "createdAt", "id", "message", "metadata", "resolved", "severity", "source", "title", "type",
	}
	byRule := map[string][]string{}
	for _, raw := range resp.Alerts {
		var row struct {
			ID   int64  `json:"id"`
			Rule string `json:"rule"`
		}
		require.NoError(t, json.Unmarshal(raw, &row))
		if row.ID == handled.ID {
			require.Equal(t, sortedKeys(fixed,
				"acknowledgedAt", "acknowledgedBy", "deliveries", "deviceId", "resolvedAt"),
				topLevelKeys(t, raw))
			var got map[string]any
			require.NoError(t, json.Unmarshal(raw, &got))
			require.Equal(t, map[string]any{"server": "192.0.2.9"}, got["metadata"])
			require.Equal(t, []any{map[string]any{
				"channel": "webhook", "status": "delivered", "attemptedAt": "2026-10-11T21:00:00Z",
			}}, got["deliveries"])
			continue
		}
		byRule[row.Rule] = topLevelKeys(t, raw)
	}
	require.Equal(t, sortedKeys(fixed, "narrative", "rule"), byRule["iface.down"])
	require.Equal(t, sortedKeys(fixed, "rootCauseId", "rule"), byRule["bgp.flap"])
}

func sortedKeys(base []string, extra ...string) []string {
	out := append(append([]string{}, base...), extra...)
	slices.Sort(out)
	return out
}

func TestAlertActionResponseShapes(t *testing.T) {
	s := newAlertsTestServer(t)
	id := seedAlert(t, s.db(), alerts.SeverityWarning)
	idPath := alertsPath + "/" + strconv.FormatInt(id, 10)

	req := httptest.NewRequest(http.MethodPost, idPath+"/acknowledge", http.NoBody)
	req = req.WithContext(auth.WithUsername(req.Context(), "alice"))
	w := httptest.NewRecorder()
	routed(alertAcknowledgePath, s.handleAlertAcknowledge)(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(
		t,
		`{"id":`+strconv.FormatInt(id, 10)+`,"acknowledged":true,"acknowledgedBy":"alice"}`,
		w.Body.String(),
	)

	w = httptest.NewRecorder()
	routed(
		alertResolvePath,
		s.handleAlertResolve,
	)(
		w,
		httptest.NewRequest(http.MethodPost, idPath+"/resolve", http.NoBody),
	)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"id":`+strconv.FormatInt(id, 10)+`,"resolved":true}`, w.Body.String())
}
