package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/retention"
)

// handlers_history.go is #175's read surface over the tiered time-series store
// seed already keeps. Two series, each over a fixed window: a probe's trend
// and the per-day anomaly count.
//
// Deliberately not a query surface. There is no metric selector, no filter
// expression and no grouping parameter: seed stays handheld-class, competing
// with AirCheck/EtherScope rather than with Zabbix, and it exposes a fixed
// window over what it recorded (#175, "Why the line is here").
//
// The window a tier can answer is resolved once, in resolveHistoryWindow, and
// reported back on every response: a request past the tier's horizon is served
// short and says so rather than erroring.

// historyProbesPathPrefix is the collection prefix the per-probe route hangs
// the probe id off, matching the /polling-targets/ idiom.
const historyProbesPathPrefix = APIVersionPrefix + "/history/probes/"

// errInvalidHistoryRange is returned for a range outside (0, historyMaxRange].
var errInvalidHistoryRange = errors.New("range outside the retained horizons")

// historyMaxRange caps what a caller may ask for, before the tier horizon is
// applied. It is the longest horizon any tier retains, so it never overrides
// the tier's own answer — it only stops an absurd duration reaching the store.
const historyMaxRange = 730 * hoursPerDay * time.Hour

// defaultHistoryRange is the window served when the request names none.
const defaultHistoryRange = 24 * time.Hour

// HistoryWindowResponse describes the window a history series was served over.
// It is what lets a client draw an honest chart: Clamped says the tier's
// horizon is shorter than the request, and Source says whether the last bucket
// is still filling.
type HistoryWindowResponse struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
	// Days is the served window in whole days.
	Days int `json:"days"`
	// Resolution is the bucket width: "hourly" or "daily".
	Resolution string `json:"resolution"`
	// Source is "raw" when the series was aggregated from raw rows on read —
	// in which case the final bucket is still filling — or "rollup" when it
	// came from a pre-aggregated table, whose buckets are all closed.
	Source string `json:"source"`
	// Clamped is true when the licence tier retains less than was asked for.
	Clamped bool `json:"clamped"`
	// RequestedDays is what the caller asked for, so a clamped response can
	// say by how much without the client re-parsing its own request.
	RequestedDays int `json:"requestedDays"`
}

// ProbeHistoryResponse is one probe's trend over the resolved window.
type ProbeHistoryResponse struct {
	ProbeID string                     `json:"probeId"`
	Window  HistoryWindowResponse      `json:"window"`
	Points  []database.ProbeTrendPoint `json:"points"`
}

// AnomalyHistoryResponse is the per-day anomaly count over the resolved
// window. Days with no anomaly are present with a zero count, so a client can
// draw the series without filling gaps itself.
type AnomalyHistoryResponse struct {
	Window HistoryWindowResponse      `json:"window"`
	Days   []database.AnomalyDayCount `json:"days"`
}

// handleProbeHistory serves GET /api/v1/history/probes/{probeID}.
func (s *Server) handleProbeHistory(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	probeID := strings.TrimPrefix(r.URL.Path, historyProbesPathPrefix)
	if probeID == "" || strings.Contains(probeID, "/") {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.history.probeRequired"), "")
		return
	}
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	window, ok := s.historyWindow(w, r)
	if !ok {
		return
	}

	daily := window.Resolution == historyResolutionDaily
	history := s.db().History()

	var (
		points []database.ProbeTrendPoint
		err    error
	)
	if window.Source == historySourceRaw {
		points, err = history.ProbeTrendRaw(r.Context(), clientID, probeID,
			window.From, window.To, daily)
	} else {
		points, err = history.ProbeTrendRollup(r.Context(), clientID, probeID,
			window.From, window.To, daily)
	}
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to read probe history", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.history.readFailed"), "")
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, ProbeHistoryResponse{
		ProbeID: probeID,
		Window:  windowResponse(window),
		Points:  emptyIfNil(points),
	})
}

// handleAnomalyHistory serves GET /api/v1/history/anomalies.
//
// The daily census (ADR-0028) is retained under the DailyDays horizon, which
// is zero below Pro — so below Pro the same counts are derived from the live
// anomalies table over the days the raw horizon still covers, using the
// census's own "open on this day" predicate. The response shape does not
// change with the tier; only the window does.
func (s *Server) handleAnomalyHistory(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	window, ok := s.historyWindow(w, r)
	if !ok {
		return
	}

	history := s.db().History()
	// The window's last day is inclusive here: a day bucket is the whole day,
	// and the day in progress is one the caller wants to see.
	from, to := window.From, window.To

	var (
		days []database.AnomalyDayCount
		err  error
	)
	if window.Source == historySourceRaw {
		days, err = history.AnomalyCountsByDayLive(r.Context(), from, to)
	} else {
		days, err = history.AnomalyCountsByDayRollup(r.Context(), from, to)
	}
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to read anomaly history", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.history.readFailed"), "")
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, AnomalyHistoryResponse{
		Window: windowResponse(window),
		Days:   emptyIfNil(days),
	})
}

// historyWindow parses ?range= and resolves it against the licence tier's
// retention horizons, or writes the rejection and reports false.
func (s *Server) historyWindow(w http.ResponseWriter, r *http.Request) (historyWindow, bool) {
	requested := defaultHistoryRange
	if raw := r.URL.Query().Get("range"); raw != "" {
		parsed, err := parseHistoryRange(raw)
		if err != nil {
			sendErrorResponseWithDetails(w, logging.FromContext(r.Context()),
				http.StatusBadRequest, ErrCodeValidation,
				i18n.FromRequest(r).T("errors.history.invalidRange"), "")
			return historyWindow{}, false
		}
		requested = parsed
	}

	return resolveHistoryWindow(time.Now().UTC(), requested,
		retention.HorizonsFor(s.effectiveTier())), true
}

// parseHistoryRange accepts a Go duration ("90m", "24h") or a whole number of
// days ("7d", "90d"), which [time.ParseDuration] does not. Anything longer than
// the longest retained horizon is refused rather than silently clamped: a
// caller asking for five years has misunderstood the surface, and the tier
// clamp exists for the tier, not for typos.
func parseHistoryRange(raw string) (time.Duration, error) {
	var d time.Duration
	if days, ok := parseDaySuffix(raw); ok {
		d = time.Duration(days) * hoursPerDay * time.Hour
	} else {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return 0, err
		}
		d = parsed
	}
	if d <= 0 || d > historyMaxRange {
		return 0, errInvalidHistoryRange
	}
	return d, nil
}

// parseDaySuffix reads the "<n>d" form [time.ParseDuration] does not accept.
func parseDaySuffix(raw string) (int, bool) {
	if len(raw) < 2 || raw[len(raw)-1] != 'd' {
		return 0, false
	}
	days, err := strconv.Atoi(raw[:len(raw)-1])
	if err != nil {
		return 0, false
	}
	return days, true
}

// windowResponse maps the resolved window onto its wire shape.
func windowResponse(w historyWindow) HistoryWindowResponse {
	requestedDays := w.Days
	if w.Clamped {
		requestedDays = w.RequestedDays
	}
	return HistoryWindowResponse{
		From:          w.From,
		To:            w.To,
		Days:          w.Days,
		Resolution:    w.Resolution,
		Source:        w.Source,
		Clamped:       w.Clamped,
		RequestedDays: requestedDays,
	}
}

// emptyIfNil keeps an empty series `[]` on the wire rather than `null`, so a
// client never has to distinguish "no data" from "no field".
func emptyIfNil[T any](v []T) []T {
	if v == nil {
		return []T{}
	}
	return v
}
