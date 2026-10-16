package paloalto_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/outbound"
	"github.com/MustardSeedNetworks/seed/internal/paloalto"
)

const testKey = "LUFRPT1Bc2VlZFRlc3RLZXk9PQ=="

const firewallInfo = `<response status="success"><result><system>
<hostname>fw-edge</hostname><ip-address>10.0.0.1</ip-address><model>PA-440</model>
<family>400</family><serial>021201000001</serial><sw-version>11.1.4</sw-version>
<operational-mode>normal</operational-mode>
</system></result></response>`

const panoramaInfo = `<response status="success"><result><system>
<hostname>pano</hostname><ip-address>10.0.0.10</ip-address><model>Panorama</model>
<family>pc</family><serial>000710000001</serial><sw-version>11.1.4</sw-version>
<system-mode>panorama</system-mode>
</system></result></response>`

const panoramaDevices = `<response status="success"><result><devices>
<entry name="021201000003"><serial>021201000003</serial><connected>no</connected>
<hostname>fw-branch</hostname><ip-address>10.2.0.1</ip-address><model>PA-410</model>
<family>400</family><sw-version>10.2.9</sw-version></entry>
<entry name="021201000002"><serial>021201000002</serial><connected>yes</connected>
<hostname>fw-dc-a</hostname><ip-address>10.1.0.1</ip-address><model>PA-3410</model>
<family>3400</family><sw-version>11.1.4</sw-version><ha><state>active</state></ha></entry>
</devices></result></response>`

// fakePANOS serves the XML API as a firewall, or as Panorama when panorama is
// set. An unknown key is answered as PAN-OS does: 403 with an error envelope.
type fakePANOS struct {
	requests atomic.Int64
	panorama bool
}

func newFakePANOS(t *testing.T, panorama bool) (*fakePANOS, *httptest.Server) {
	t.Helper()
	api := &fakePANOS{panorama: panorama}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.requests.Add(1)
		w.Header().Set("Content-Type", "application/xml")
		if r.Header.Get("X-Pan-Key") != testKey {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`<response status="error" code="403"><result><msg>Invalid Credential ` +
				r.Header.Get("X-Pan-Key") + `</msg></result></response>`))
			return
		}
		if r.URL.Path != "/api/" || r.URL.Query().Get("type") != "op" {
			http.NotFound(w, r)
			return
		}
		switch cmd := r.URL.Query().Get("cmd"); {
		case cmd == "<show><system><info></info></system></show>" && api.panorama:
			_, _ = w.Write([]byte(panoramaInfo))
		case cmd == "<show><system><info></info></system></show>":
			_, _ = w.Write([]byte(firewallInfo))
		case cmd == "<show><devices><all></all></devices></show>" && api.panorama:
			_, _ = w.Write([]byte(panoramaDevices))
		default:
			_, _ = w.Write(
				[]byte(`<response status="error" code="17"><msg><line>unknown command</line></msg></response>`),
			)
		}
	}))
	t.Cleanup(srv.Close)
	return api, srv
}

func TestClientDevices(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		panorama bool
		want     []paloalto.Device
	}{
		{
			name: "firewall reports itself",
			want: []paloalto.Device{{
				Serial: "021201000001", Hostname: "fw-edge", Model: "PA-440", Family: "400",
				SWVersion: "11.1.4", IPAddress: "10.0.0.1", Status: paloalto.StatusConnected,
			}},
		},
		{
			name:     "panorama reports itself and its firewalls in serial order",
			panorama: true,
			want: []paloalto.Device{
				{
					Serial: "000710000001", Hostname: "pano", Model: "Panorama", Family: "pc",
					SWVersion: "11.1.4", IPAddress: "10.0.0.10", Status: paloalto.StatusConnected,
				},
				{
					Serial: "021201000002", Hostname: "fw-dc-a", Model: "PA-3410", Family: "3400",
					SWVersion: "11.1.4", IPAddress: "10.1.0.1", ManagedBy: "000710000001",
					Status: paloalto.StatusConnected, HAState: "active",
				},
				{
					Serial: "021201000003", Hostname: "fw-branch", Model: "PA-410", Family: "400",
					SWVersion: "10.2.9", IPAddress: "10.2.0.1", ManagedBy: "000710000001",
					Status: paloalto.StatusDisconnected,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, srv := newFakePANOS(t, tt.panorama)
			devices, err := paloalto.NewClient(srv.Client()).Devices(context.Background(),
				outbound.Credentials{Endpoint: srv.URL + "/", APIKey: testKey})
			require.NoError(t, err)
			assert.Equal(t, tt.want, devices)
		})
	}
}

func TestClientErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		endpoint string
		key      string
		wantErr  string
	}{
		{name: "wrong key", key: "wrong-key", wantErr: "show system info: HTTP 403"},
		{name: "not the API", endpoint: "/elsewhere", key: testKey, wantErr: "HTTP 404"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, srv := newFakePANOS(t, false)
			_, err := paloalto.NewClient(srv.Client()).Devices(context.Background(),
				outbound.Credentials{Endpoint: srv.URL + tt.endpoint, APIKey: tt.key})
			require.ErrorContains(t, err, tt.wantErr)
			assert.NotContains(t, err.Error(), "wrong-key", "the body echoes the key and is never quoted")
		})
	}
}

func TestClientAPIErrorNamesCode(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<response status="error" code="22"><msg>session timed out</msg></response>`))
	}))
	t.Cleanup(srv.Close)
	_, err := paloalto.NewClient(srv.Client()).Devices(context.Background(),
		outbound.Credentials{Endpoint: srv.URL, APIKey: testKey})
	require.ErrorContains(t, err, "API error code 22")
	assert.NotContains(t, err.Error(), "session timed out")
}

func TestClientFollowsNoRedirect(t *testing.T) {
	t.Parallel()
	var elsewhere atomic.Int64
	other := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		elsewhere.Add(1)
	}))
	t.Cleanup(other.Close)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+r.URL.RequestURI(), http.StatusFound)
	}))
	t.Cleanup(srv.Close)

	hc := outbound.NewClient(time.Second)
	hc.Transport = srv.Client().Transport
	_, err := paloalto.NewClient(hc).Devices(context.Background(),
		outbound.Credentials{Endpoint: srv.URL, APIKey: testKey})
	require.ErrorContains(t, err, "HTTP 302")
	assert.Zero(t, elsewhere.Load(), "the key never follows a redirect")
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

// TestReaderOffMakesNoRequest is the acceptance's zero-outbound half: with the
// integration unanswered or off, start and wakes reach nothing. Turned on, the
// next wake reads device state; turned off again, it is dropped.
func TestReaderOffMakesNoRequest(t *testing.T) {
	t.Parallel()
	api, srv := newFakePANOS(t, true)
	source := &switchSource{endpoint: srv.URL}
	reader := outbound.NewReader("paloalto", source, paloalto.NewClient(srv.Client()).Devices, time.Hour,
		slog.New(slog.DiscardHandler))

	require.NoError(t, reader.Start(context.Background()))
	t.Cleanup(func() { require.NoError(t, reader.Stop(context.Background())) })
	for source.asked.Load() < 3 {
		reader.Wake()
		runtime.Gosched()
	}
	assert.Zero(t, api.requests.Load(), "an integration that is not on makes no request")
	assert.True(t, reader.Status().LastAttempt.IsZero())

	source.on.Store(true)
	reader.Wake()
	require.Eventually(t, func() bool { return !reader.Status().LastSuccess.IsZero() },
		5*time.Second, 10*time.Millisecond)
	assert.Empty(t, reader.Status().LastError)
	assert.Len(t, reader.Items(), 3)

	source.on.Store(false)
	asked := source.asked.Load()
	reader.Wake()
	require.Eventually(t, func() bool { return source.asked.Load() > asked && reader.Items() == nil },
		5*time.Second, 10*time.Millisecond, "turning it off drops the device state")
}
