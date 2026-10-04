package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/license/licensetest"
	"github.com/MustardSeedNetworks/seed/internal/polling"
)

// licenseManagerAt is a licence manager at tier: Free has no licence at all.
func licenseManagerAt(t *testing.T, tier license.Tier) *license.Manager {
	t.Helper()
	if tier != license.TierFree {
		return licensetest.PaidManager(t, tier, time.Now().Add(24*time.Hour), thisDevice(t))
	}
	mgr, err := license.NewManagerWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("license manager: %v", err)
	}
	return mgr
}

// pollingServerAt is the polling-targets test server under a licence of tier.
func pollingServerAt(t *testing.T, tier license.Tier) *Server {
	t.Helper()
	s := newPollingTargetsTestServer(t)
	s.licenseMgr = licenseManagerAt(t, tier)
	return s
}

// postTarget creates one polling target through the handler.
func postTarget(t *testing.T, s *Server, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	s.handlePollingTargets(w, postJSON(t, body))
	return w
}

// requireFeatureGate asserts a 402 naming feature, the shape the UI keys on.
func requireFeatureGate(t *testing.T, w *httptest.ResponseRecorder, feature string) {
	t.Helper()
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402; body=%s", w.Code, w.Body.String())
	}
	var gate FeatureGateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &gate); err != nil {
		t.Fatalf("decode 402 body: %v", err)
	}
	if gate.Code != errCodeTierTooLow || gate.RequiredFeature != feature {
		t.Errorf("402 body = %+v, want %s naming %q", gate, errCodeTierTooLow, feature)
	}
}

// estate_polling is a count (seed#2327): each tier holds up to its limit and
// the create past it is refused with the upgrade hint, while Pro is unlimited.
func TestPollingTargetCreateCappedByTier(t *testing.T) {
	for _, tt := range []struct {
		tier  license.Tier
		limit int // 0 = unlimited; probed past starterPollingTargets
	}{
		{license.TierFree, freePollingTargets},
		{license.TierStarter, starterPollingTargets},
		{license.TierPro, 0},
	} {
		t.Run(tt.tier.String(), func(t *testing.T) {
			s := pollingServerAt(t, tt.tier)
			creates := tt.limit
			if creates == 0 {
				creates = starterPollingTargets + 1
			}
			for i := range creates {
				w := postTarget(t, s, map[string]any{
					"name": fmt.Sprintf("dev-%02d", i), "ipAddress": fmt.Sprintf("10.0.0.%d", i+1),
				})
				if w.Code != http.StatusOK {
					t.Fatalf("create %d of %d: status %d, body=%s", i+1, creates, w.Code, w.Body.String())
				}
			}
			if tt.limit == 0 {
				return
			}
			w := postTarget(t, s, map[string]any{"name": "one-too-many", "ipAddress": "10.0.1.1"})
			requireFeatureGate(t, w, "estate_polling")
			list, err := s.db().PollingTargets().ListAll(context.Background(), testClientID)
			if err != nil || len(list) != tt.limit {
				t.Errorf("stored %d targets (err %v), want the limit %d", len(list), err, tt.limit)
			}
		})
	}
}

// server_monitoring and bgp_monitoring are the host_resources and bgp4_mib
// collectors: below Pro, a chain naming either is refused on create and on
// update, and the default chain is not.
func TestPollingTargetProCollectorsGated(t *testing.T) {
	for _, tt := range []struct {
		collector string
		feature   string
	}{
		{"host_resources", "server_monitoring"},
		{"bgp4_mib", "bgp_monitoring"},
	} {
		t.Run(tt.collector, func(t *testing.T) {
			chain := []string{"sys_info", tt.collector}

			starter := pollingServerAt(t, license.TierStarter)
			requireFeatureGate(t, postTarget(t, starter, map[string]any{
				"name": "server-1", "ipAddress": "10.0.0.1", "collectorChain": chain,
			}), tt.feature)

			existing := seedTarget(t, starter.db(), "router-1")
			req := postJSON(t, map[string]any{
				"name": "router-1", "ipAddress": "10.0.0.1", "collectorChain": chain,
			})
			req.Method = http.MethodPut
			req.URL.Path = pollingTargetsPathPrefix + existing.ID
			w := httptest.NewRecorder()
			starter.handlePollingTargetByID(w, req)
			requireFeatureGate(t, w, tt.feature)

			defaultChain := postTarget(t, starter, map[string]any{"name": "router-2", "ipAddress": "10.0.0.2"})
			if defaultChain.Code != http.StatusOK {
				t.Errorf("default chain at Starter: status %d, body=%s",
					defaultChain.Code, defaultChain.Body.String())
			}

			pro := pollingServerAt(t, license.TierPro)
			atPro := postTarget(t, pro, map[string]any{
				"name": "server-1", "ipAddress": "10.0.0.1", "collectorChain": chain,
			})
			if atPro.Code != http.StatusOK {
				t.Errorf("Pro: status %d, body=%s", atPro.Code, atPro.Body.String())
			}
			if !pro.collectorLicensed(tt.collector) || starter.collectorLicensed(tt.collector) {
				t.Errorf("poller registration: Pro licensed %v, Starter licensed %v; want true, false",
					pro.collectorLicensed(tt.collector), starter.collectorLicensed(tt.collector))
			}
		})
	}
}

type fixedPollerTargets struct{ targets []*polling.Target }

func (f fixedPollerTargets) ListEnabled(context.Context) ([]*polling.Target, error) {
	return f.targets, nil
}

func (fixedPollerTargets) UpdateLastPoll(context.Context, string, string, string) error { return nil }

// A licence that shrinks under existing targets (a Pro trial with promoted
// devices, then a Starter key) stops polling the ones past the limit.
func TestLicensedPollerTargetsStopAtLimit(t *testing.T) {
	all := make([]*polling.Target, starterPollingTargets+5)
	for i := range all {
		all[i] = &polling.Target{ID: fmt.Sprintf("t-%02d", i)}
	}
	for _, tt := range []struct {
		tier license.Tier
		want int
	}{
		{license.TierStarter, starterPollingTargets},
		{license.TierPro, len(all)},
	} {
		t.Run(tt.tier.String(), func(t *testing.T) {
			s := pollingServerAt(t, tt.tier)
			got, err := licensedPollerTargets{
				PollerStorage: fixedPollerTargets{targets: all},
				limit:         s.pollingTargetLimit,
			}.ListEnabled(context.Background())
			if err != nil || len(got) != tt.want {
				t.Fatalf("poller reads %d targets (err %v), want %d", len(got), err, tt.want)
			}
			if got[0].ID != "t-00" {
				t.Errorf("first polled target = %s, want the repository's first", got[0].ID)
			}
		})
	}
}
