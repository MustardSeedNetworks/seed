package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/indicators"
)

func mustIndicators(t *testing.T, entries ...string) *indicators.List {
	t.Helper()
	list, err := indicators.New(entries)
	require.NoError(t, err)
	return list
}

func serveFlowIndicators(t *testing.T, s *Server, method, body string) (int, FlowIndicatorsResponse) {
	t.Helper()
	w := httptest.NewRecorder()
	s.handleFlowIndicators(w, httptest.NewRequest(method, "/api/v1/flows/threat-indicators", strings.NewReader(body)))
	var resp FlowIndicatorsResponse
	if w.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	}
	return w.Code, resp
}

func TestFlowIndicatorsReplaceAndClear(t *testing.T) {
	t.Parallel()

	s := newHistoryServer(t)
	code, resp := serveFlowIndicators(t, s, http.MethodGet, "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, []string{}, resp.Indicators)

	code, resp = serveFlowIndicators(t, s, http.MethodPut, `{"indicators":["198.51.100.7/32","2001:DB8:BAD::/48"]}`)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, []string{"198.51.100.7", "2001:db8:bad::/48"}, resp.Indicators)

	code, resp = serveFlowIndicators(t, s, http.MethodGet, "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, []string{"198.51.100.7", "2001:db8:bad::/48"}, resp.Indicators)

	code, resp = serveFlowIndicators(t, s, http.MethodPut, `{"indicators":[]}`)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, []string{}, resp.Indicators)
}

func TestFlowIndicatorsRejectAnInvalidList(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{
		"not an address": `{"indicators":["evil.example"]}`,
		"private":        `{"indicators":["192.168.1.1"]}`,
		"unknown field":  `{"indicator":["198.51.100.7"]}`,
		"not json":       `indicators`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := newHistoryServer(t)
			require.NoError(t, s.db().FlowRecords().SetFlowIndicators(t.Context(), mustIndicators(t, "198.51.100.7")))

			w := httptest.NewRecorder()
			s.handleFlowIndicators(
				w,
				httptest.NewRequest(http.MethodPut, "/api/v1/flows/threat-indicators", strings.NewReader(body)),
			)
			require.Equal(t, http.StatusBadRequest, w.Code)
			require.NotContains(t, w.Body.String(), "evil.example")

			list, err := s.db().FlowRecords().FlowIndicators(t.Context())
			require.NoError(t, err)
			require.Equal(
				t,
				[]string{"198.51.100.7"},
				list.Entries(),
				"a rejected list must not replace the stored one",
			)
		})
	}
}
