package api

// Vulnerability triage (S6-2, #899): the persisted findings, an operator's
// status decision on one, and the finding's remediation history.
//
//   GET  /api/v1/security/vulnerabilities/findings              list
//   POST /api/v1/security/vulnerabilities/findings/{id}/status  triage
//   GET  /api/v1/security/vulnerabilities/findings/{id}/history trail
//
// /results is the live view of the last scan pass, enriched from the CVE
// feed and held in memory; these routes serve device_vulnerabilities, the
// record reports and the export read, which is where a decision has to land.
// Triage is an operator write, gated like the scanner settings: by role, not
// by tier, so findings from an earlier Pro scan can still be handled.

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

const (
	vulnFindingsPath       = APIVersionPrefix + "/security/vulnerabilities/findings"
	vulnFindingsPathPrefix = vulnFindingsPath + "/"

	vulnFindingsDefaultLimit = 100
	vulnFindingsMaxLimit     = 1000

	// vulnReasonMaxLen bounds the free-text reason kept in the history.
	vulnReasonMaxLen = 500
)

// VulnFindingResponse is one persisted finding.
type VulnFindingResponse struct {
	ID                int64      `json:"id"`
	DeviceID          string     `json:"deviceId"`
	DeviceIP          string     `json:"deviceIp"`
	Hostname          string     `json:"hostname,omitempty"`
	CVEID             string     `json:"cveId"`
	Severity          string     `json:"severity"`
	Score             float64    `json:"score"`
	Description       string     `json:"description,omitempty"`
	AffectedComponent string     `json:"affectedComponent,omitempty"`
	AffectedVersion   string     `json:"affectedVersion,omitempty"`
	Status            string     `json:"status"`
	DetectedAt        time.Time  `json:"detectedAt"`
	ResolvedAt        *time.Time `json:"resolvedAt,omitempty"`
}

// VulnStatusChangeResponse is one entry of a finding's history. An empty
// actor is the scanner.
type VulnStatusChangeResponse struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Actor     string    `json:"actor,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	ChangedAt time.Time `json:"changedAt"`
}

// vulnStatusRequest is the triage body.
type vulnStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// handleVulnFindings serves GET /api/v1/security/vulnerabilities/findings.
// Filters: status, device_id, limit, offset.
func (s *Server) handleVulnFindings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	if s.db() == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.vulnerability.storeUnavailable"), "")
		return
	}

	q := r.URL.Query()
	opts := database.VulnListOptions{
		DeviceID: q.Get("device_id"),
		Limit:    vulnFindingsDefaultLimit,
	}
	if status := q.Get("status"); status != "" {
		if !validVulnStatus(status) {
			sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
				ErrCodeValidation, localizer.T("errors.vulnerability.invalidStatus"), status)
			return
		}
		opts.Status = database.VulnStatus(status)
	}
	var ok bool
	if opts.Limit, ok = vulnQueryInt(q.Get("limit"), vulnFindingsDefaultLimit, 1, vulnFindingsMaxLimit); !ok {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.vulnerability.invalidPaging"), "limit")
		return
	}
	if opts.Offset, ok = vulnQueryInt(q.Get("offset"), 0, 0, -1); !ok {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.vulnerability.invalidPaging"), "offset")
		return
	}

	findings, err := s.db().Vulnerabilities().ListFindings(r.Context(), opts)
	if err != nil {
		logger.ErrorContext(r.Context(), "list vulnerability findings failed", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.vulnerability.readFailed"), "")
		return
	}
	out := make([]VulnFindingResponse, 0, len(findings))
	for i := range findings {
		f := &findings[i]
		out = append(out, VulnFindingResponse{
			ID: f.ID, DeviceID: f.DeviceID, DeviceIP: f.DeviceIP, Hostname: f.Hostname,
			CVEID: f.CVEID, Severity: f.Severity, Score: f.CVSSScore,
			Description: f.Description, AffectedComponent: f.AffectedComponent,
			AffectedVersion: f.AffectedVersion, Status: string(f.Status),
			DetectedAt: f.DetectedAt, ResolvedAt: f.ResolvedAt,
		})
	}
	sendJSONResponse(w, logger, http.StatusOK, map[string]any{jsonKeyCount: len(out), "findings": out})
}

// handleVulnFindingAction routes /findings/{id}/status (POST) and
// /findings/{id}/history (GET).
func (s *Server) handleVulnFindingAction(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	idText, action, found := strings.Cut(strings.TrimPrefix(r.URL.Path, vulnFindingsPathPrefix), "/")
	id, err := strconv.ParseInt(idText, 10, 64)
	if !found || err != nil || id <= 0 {
		sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
			ErrCodeNotFound, localizer.T("errors.vulnerability.findingNotFound"), "")
		return
	}
	if s.db() == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.vulnerability.storeUnavailable"), "")
		return
	}

	switch {
	case action == "status" && r.Method == http.MethodPost:
		s.setVulnFindingStatus(w, r, id)
	case action == "history" && r.Method == http.MethodGet:
		s.writeVulnFindingHistory(w, r, id)
	case action == "status" || action == "history":
		allow := http.MethodPost
		if action == "history" {
			allow = http.MethodGet
		}
		w.Header().Set("Allow", allow)
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed, "Method not allowed", "")
	default:
		sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
			ErrCodeNotFound, localizer.T("errors.vulnerability.findingNotFound"), "")
	}
}

func (s *Server) setVulnFindingStatus(w http.ResponseWriter, r *http.Request, id int64) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	var req vulnStatusRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if !validVulnStatus(req.Status) {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.vulnerability.invalidStatus"), req.Status)
		return
	}
	if len(req.Reason) > vulnReasonMaxLen {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.vulnerability.reasonTooLong"), "")
		return
	}

	actor := s.usernameFromRequest(r)
	err := s.db().Vulnerabilities().SetStatus(r.Context(), id,
		database.VulnStatus(req.Status), actor, req.Reason, time.Now())
	switch {
	case err == nil:
		logger.InfoContext(r.Context(), "vulnerability finding triaged",
			"event", "vuln.status", "id", id, "status", req.Status, "actor", actor)
		sendJSONResponse(w, logger, http.StatusOK, map[string]any{"id": id, "status": req.Status})
	case errors.Is(err, database.ErrVulnFindingNotFound):
		sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
			ErrCodeNotFound, localizer.T("errors.vulnerability.findingNotFound"), "")
	case errors.Is(err, database.ErrVulnReasonRequired):
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.vulnerability.reasonRequired"), "")
	case errors.Is(err, database.ErrVulnTransition):
		sendErrorResponseWithDetails(w, logger, http.StatusConflict,
			ErrCodeConflict, localizer.T("errors.vulnerability.transitionRefused"), req.Status)
	default:
		logger.ErrorContext(r.Context(), "vulnerability triage failed", "id", id, "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.vulnerability.writeFailed"), "")
	}
}

func (s *Server) writeVulnFindingHistory(w http.ResponseWriter, r *http.Request, id int64) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	changes, err := s.db().Vulnerabilities().History(r.Context(), id)
	if errors.Is(err, database.ErrVulnFindingNotFound) {
		sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
			ErrCodeNotFound, localizer.T("errors.vulnerability.findingNotFound"), "")
		return
	}
	if err != nil {
		logger.ErrorContext(r.Context(), "vulnerability history failed", "id", id, "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.vulnerability.readFailed"), "")
		return
	}
	out := make([]VulnStatusChangeResponse, 0, len(changes))
	for _, c := range changes {
		out = append(out, VulnStatusChangeResponse{
			From: string(c.From), To: string(c.To), Actor: c.Actor,
			Reason: c.Reason, ChangedAt: c.ChangedAt,
		})
	}
	sendJSONResponse(w, logger, http.StatusOK, map[string]any{"id": id, "history": out})
}

// validVulnStatus reports whether s names one of the four finding states.
func validVulnStatus(s string) bool {
	switch database.VulnStatus(s) {
	case database.VulnStatusNew, database.VulnStatusAcknowledged,
		database.VulnStatusIgnored, database.VulnStatusResolved:
		return true
	}
	return false
}

// vulnQueryInt parses an optional integer query value within [lo, hi]; a
// negative hi means no upper bound.
func vulnQueryInt(raw string, def, lo, hi int) (int, bool) {
	if raw == "" {
		return def, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < lo || (hi >= 0 && n > hi) {
		return 0, false
	}
	return n, true
}
