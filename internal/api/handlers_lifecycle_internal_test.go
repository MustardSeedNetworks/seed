package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/lifecycle"
)

func newLifecycleServer(t *testing.T) *Server {
	t.Helper()
	s := &Server{}
	s.dbConn = newTestDB(t)
	s.lifecycleEoX = app.NewLifecycleStore(s.db)
	return s
}

func serveLifecycleEoX(
	t *testing.T,
	s *Server,
	method, body string,
) (*httptest.ResponseRecorder, LifecycleEoXResponse) {
	t.Helper()
	w := httptest.NewRecorder()
	s.handleLifecycleEoX(w, httptest.NewRequest(method, APIVersionPrefix+"/lifecycle/eox", strings.NewReader(body)))
	var resp LifecycleEoXResponse
	if w.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	}
	return w, resp
}

// The acceptance of P-D9, first half: lifecycle data loads from a local
// source. The operator's downloaded EoX response is imported, kept, and
// cleared by importing a response with no records.
func TestLifecycleEoXImportAndClear(t *testing.T) {
	t.Parallel()
	s := newLifecycleServer(t)

	w, resp := serveLifecycleEoX(t, s, http.MethodGet, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []lifecycle.EoXRecord{}, resp.Records)

	fixture, err := os.ReadFile("../lifecycle/testdata/eox-response.json")
	require.NoError(t, err)
	w, resp = serveLifecycleEoX(t, s, http.MethodPut, string(fixture))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Len(t, resp.Records, 2)
	require.Equal(t, "C9300-48P", resp.Records[0].ProductID)
	require.Equal(t, "WS-C3750X-48P-S", resp.Records[1].ProductID)

	w, resp = serveLifecycleEoX(t, s, http.MethodGet, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, resp.Records, 2, "the import survives a fresh read from settings")

	w, resp = serveLifecycleEoX(t, s, http.MethodPut, `{"EOXRecord":[]}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []lifecycle.EoXRecord{}, resp.Records)
}

func TestLifecycleEoXRejectsAndKeepsTheTable(t *testing.T) {
	t.Parallel()
	s := newLifecycleServer(t)
	w, _ := serveLifecycleEoX(t, s, http.MethodPut,
		`{"EOXRecord":[{"EOLProductID":"WS-C3750X-48P-S","EndOfSaleDate":{"value":"2016-10-30"}}]}`)
	require.Equal(t, http.StatusOK, w.Code)

	for name, body := range map[string]string{
		"csv":      "EOLProductID,EndOfSaleDate\nWS-C3750X-48P-S,2016-10-30",
		"not eox":  `{"records":[]}`,
		"bad date": `{"EOXRecord":[{"EOLProductID":"A-1","EndOfSaleDate":{"value":"30/10/2016"}}]}`,
	} {
		rejected, _ := serveLifecycleEoX(t, s, http.MethodPut, body)
		require.Equal(t, http.StatusBadRequest, rejected.Code, name)
	}
	_, resp := serveLifecycleEoX(t, s, http.MethodGet, "")
	require.Len(t, resp.Records, 1)
}

func TestLifecycleDevices(t *testing.T) {
	table, err := lifecycle.New([]lifecycle.EoXRecord{
		{ProductID: "WS-C3750X-48P-S", EndOfSale: "2016-10-30", LastDateOfSupport: "2021-10-31"},
		{ProductID: "C3KX-PWR-715WAC", EndOfSale: "2026-12-01"},
	})
	require.NoError(t, err)
	devices := []*discovery.DiscoveredDevice{
		{IP: "10.44.10.20", DisplayName: "access-2", SNMPData: &discovery.SNMPFullData{
			Inventory: []discovery.SNMPEntity{
				{Index: 1001, Class: "powerSupply", Name: "Switch 1 - Power Supply A", ModelName: "C3KX-PWR-715WAC"},
				{Index: 1, Class: "chassis", Name: "1", ModelName: " WS-C3750X-48P-S ", SerialNum: "FDO1234X0AB"},
				{Index: 1010, Class: "port", Name: "Gi1/0/1"},
				{Index: 1100, Class: "module", Name: "GigabitEthernet1/1/1", ModelName: "GLC-TE"},
			},
		}},
		{IP: "10.44.10.3", SNMPData: &discovery.SNMPFullData{
			Inventory: []discovery.SNMPEntity{{Index: 1, Class: "chassis"}},
		}},
		{IP: "10.44.10.2"},
	}

	got := lifecycleDevices(table, devices, time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))

	chassis, psu := table.Records()[1], table.Records()[0]
	require.Equal(t, []LifecycleDevice{{
		IP:   "10.44.10.20",
		Name: "access-2",
		Components: []LifecycleComponent{
			{
				ProductID: "WS-C3750X-48P-S", Name: "1", Class: "chassis", SerialNum: "FDO1234X0AB",
				Status: lifecycle.StatusEndOfSupport, Record: &chassis,
			},
			{
				ProductID: "C3KX-PWR-715WAC", Name: "Switch 1 - Power Supply A", Class: "powerSupply",
				Status: lifecycle.StatusAnnounced, Record: &psu,
			},
			{
				ProductID: "GLC-TE", Name: "GigabitEthernet1/1/1", Class: "module",
				Status: lifecycle.StatusNotListed,
			},
		},
	}}, got)
}

func TestHandleLifecycleDevicesWithoutDiscovery(t *testing.T) {
	t.Parallel()
	s := newLifecycleServer(t)
	w := httptest.NewRecorder()
	s.handleLifecycleDevices(w, httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/lifecycle/devices", http.NoBody))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"records":0,"devices":[]}`, w.Body.String())
}
