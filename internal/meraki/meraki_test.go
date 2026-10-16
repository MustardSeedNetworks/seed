package meraki_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/meraki"
	"github.com/MustardSeedNetworks/seed/internal/outbound"
)

const testKey = "0123456789abcdef0123456789abcdef01234567"

// fakeDashboard serves two organizations; org 1's devices come in two pages.
type fakeDashboard struct {
	requests atomic.Int64
	// nextHost, when set, makes the first devices page link to another host.
	nextHost string
}

func newFakeDashboard(t *testing.T) (*fakeDashboard, *httptest.Server) {
	t.Helper()
	api := &fakeDashboard{}
	mux := http.NewServeMux()
	var srv *httptest.Server
	write := func(w http.ResponseWriter, body any) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(body))
	}
	mux.HandleFunc("GET /api/v1/organizations", func(w http.ResponseWriter, _ *http.Request) {
		write(w, []map[string]string{{"id": "1", "name": "Clinic"}, {"id": "2", "name": "Lab"}})
	})
	mux.HandleFunc("GET /api/v1/organizations/1/devices", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "1000", r.URL.Query().Get("perPage"))
		if r.URL.Query().Get("startingAfter") == "" {
			host := srv.URL
			if api.nextHost != "" {
				host = api.nextHost
			}
			w.Header().Add("Link", "<"+srv.URL+"/api/v1/organizations/1/devices?perPage=1000>; rel=first, <"+
				host+"/api/v1/organizations/1/devices?perPage=1000&startingAfter=Q2-B>; rel=next")
			write(w, []map[string]string{{
				"serial": "Q2-B", "name": "ap-lobby", "model": "MR46", "productType": "wireless",
				"mac": "0c:8d:db:00:00:02", "lanIp": "10.1.0.12", "firmware": "wireless-29-7", "networkId": "N_1",
			}})
			return
		}
		write(w, []map[string]string{{
			"serial": "Q2-A", "name": "sw-core", "model": "MS250-48", "productType": "switch",
			"mac": "0c:8d:db:00:00:01", "lanIp": "10.1.0.2", "firmware": "switch-16-8", "networkId": "N_1",
		}})
	})
	mux.HandleFunc("GET /api/v1/organizations/1/devices/availabilities", func(w http.ResponseWriter, _ *http.Request) {
		write(w, []map[string]string{{"serial": "Q2-A", "status": "online"}, {"serial": "Q2-B", "status": "alerting"}})
	})
	mux.HandleFunc("GET /api/v1/organizations/2/devices", func(w http.ResponseWriter, _ *http.Request) {
		write(w, []map[string]string{{"serial": "Q2-C", "model": "MX68", "productType": "appliance"}})
	})
	mux.HandleFunc("GET /api/v1/organizations/2/devices/availabilities", func(w http.ResponseWriter, _ *http.Request) {
		write(w, []map[string]string{})
	})
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.requests.Add(1)
		if r.Header.Get("Authorization") != "Bearer "+testKey {
			http.Error(w, `{"errors":["Invalid API key `+r.Header.Get("Authorization")+`"]}`, http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return api, srv
}

func TestClientDevices(t *testing.T) {
	t.Parallel()
	_, srv := newFakeDashboard(t)
	client := meraki.NewClient(srv.Client())

	devices, err := client.Devices(context.Background(),
		outbound.Credentials{Endpoint: srv.URL + "/api/v1/", APIKey: testKey})
	require.NoError(t, err)
	assert.Equal(t, []meraki.Device{
		{
			OrganizationID: "1", OrganizationName: "Clinic", NetworkID: "N_1", Serial: "Q2-A", Name: "sw-core",
			Model: "MS250-48", ProductType: "switch", MAC: "0c:8d:db:00:00:01", LANIP: "10.1.0.2",
			Firmware: "switch-16-8", Status: "online",
		},
		{
			OrganizationID: "1", OrganizationName: "Clinic", NetworkID: "N_1", Serial: "Q2-B", Name: "ap-lobby",
			Model: "MR46", ProductType: "wireless", MAC: "0c:8d:db:00:00:02", LANIP: "10.1.0.12",
			Firmware: "wireless-29-7", Status: "alerting",
		},
		{OrganizationID: "2", OrganizationName: "Lab", Serial: "Q2-C", Model: "MX68", ProductType: "appliance"},
	}, devices, "both pages are read, availability joins by serial, a device with none has no status")
}

func TestClientRefusesForeignNextLink(t *testing.T) {
	t.Parallel()
	api, srv := newFakeDashboard(t)
	api.nextHost = "https://collector.example.net"

	_, err := meraki.NewClient(srv.Client()).Devices(context.Background(),
		outbound.Credentials{Endpoint: srv.URL + "/api/v1", APIKey: testKey})
	require.ErrorContains(t, err, "next page link leaves the configured endpoint")
}

func TestClientErrorOmitsBody(t *testing.T) {
	t.Parallel()
	_, srv := newFakeDashboard(t)

	_, err := meraki.NewClient(srv.Client()).Devices(context.Background(),
		outbound.Credentials{Endpoint: srv.URL + "/api/v1", APIKey: "wrong-key"})
	require.ErrorContains(t, err, "HTTP 401")
	assert.NotContains(t, err.Error(), "wrong-key", "the body echoes the key and is never quoted")
}

type switchSource struct {
	on       atomic.Bool
	asked    atomic.Int64
	endpoint string
}

func (s *switchSource) Credentials(context.Context) (outbound.Credentials, bool, error) {
	s.asked.Add(1)
	return outbound.Credentials{Endpoint: s.endpoint, APIKey: testKey}, s.on.Load(), nil
}

func newReader(source outbound.Source, client *meraki.Client) *outbound.Reader[meraki.Device] {
	return outbound.NewReader("meraki", source, client.Devices, time.Hour, slog.New(slog.DiscardHandler))
}

// TestRefresherOffMakesNoRequest is the acceptance's zero-outbound half: with
// the integration unanswered or off, start and wakes reach nothing. Turned
// on, the next wake reads device state; turned off again, it is dropped.
func TestRefresherOffMakesNoRequest(t *testing.T) {
	t.Parallel()
	api, srv := newFakeDashboard(t)
	source := &switchSource{endpoint: srv.URL + "/api/v1"}
	refresher := newReader(source, meraki.NewClient(srv.Client()))

	require.NoError(t, refresher.Start(context.Background()))
	t.Cleanup(func() { require.NoError(t, refresher.Stop(context.Background())) })
	// The loop consults the source at start and again after a wake; three
	// consultations mean two full off-passes ran after the first.
	for source.asked.Load() < 3 {
		refresher.Wake()
		runtime.Gosched()
	}
	assert.Zero(t, api.requests.Load(), "an integration that is not on makes no request")
	assert.True(t, refresher.Status().LastAttempt.IsZero())

	source.on.Store(true)
	refresher.Wake()
	require.Eventually(t, func() bool { return !refresher.Status().LastSuccess.IsZero() },
		5*time.Second, 10*time.Millisecond)
	assert.Empty(t, refresher.Status().LastError)
	assert.Len(t, refresher.Items(), 3)

	source.on.Store(false)
	asked := source.asked.Load()
	refresher.Wake()
	require.Eventually(t, func() bool { return source.asked.Load() > asked && refresher.Items() == nil },
		5*time.Second, 10*time.Millisecond, "turning it off drops the device state")
}

func TestRefresherRecordsFailure(t *testing.T) {
	t.Parallel()
	_, srv := newFakeDashboard(t)
	source := &switchSource{endpoint: srv.URL + "/api/v1/missing"}
	source.on.Store(true)
	refresher := newReader(source, meraki.NewClient(srv.Client()))

	require.NoError(t, refresher.Start(context.Background()))
	t.Cleanup(func() { require.NoError(t, refresher.Stop(context.Background())) })
	require.Eventually(t, func() bool { return !refresher.Status().LastAttempt.IsZero() },
		5*time.Second, 10*time.Millisecond)
	status := refresher.Status()
	assert.Contains(t, status.LastError, "HTTP 404")
	assert.True(t, status.LastSuccess.IsZero())
}
