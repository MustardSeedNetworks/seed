package api

import (
	"cmp"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/reporting"
)

// reportSchedulesPathPrefix is the schedule collection path with its trailing
// slash, so handleReportScheduleByID can recover the id.
const reportSchedulesPathPrefix = APIVersionPrefix + "/reports/schedules/"

// ReportScheduleRequest is the body of POST /api/v1/reports/schedules and of
// PUT /api/v1/reports/schedules/{id}. A PUT replaces the whole schedule.
//
// Enabled is a pointer so that leaving it out is an error rather than a
// silently disabled schedule.
type ReportScheduleRequest struct {
	Name     string             `json:"name"     validate:"required"`
	Template string             `json:"template" validate:"required"`
	Format   string             `json:"format"   validate:"required"`
	Schedule reporting.Schedule `json:"schedule"`
	Enabled  *bool              `json:"enabled"  validate:"required"`
}

// ReportScheduleInfo is a scheduled report as the API presents it. Times are
// RFC3339; nextRun is when the scheduler will next generate it.
type ReportScheduleInfo struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Template  string             `json:"template"`
	Format    string             `json:"format"`
	Schedule  reporting.Schedule `json:"schedule"`
	Enabled   bool               `json:"enabled"`
	LastRun   string             `json:"lastRun,omitempty"`
	NextRun   string             `json:"nextRun,omitempty"`
	CreatedAt string             `json:"createdAt"`
	UpdatedAt string             `json:"updatedAt"`
}

// ReportSchedulesResponse is the payload of GET /api/v1/reports/schedules.
type ReportSchedulesResponse struct {
	Schedules []ReportScheduleInfo `json:"schedules"`
}

func toReportScheduleInfo(sr *reporting.ScheduledReport) ReportScheduleInfo {
	info := ReportScheduleInfo{
		ID:        sr.ID,
		Name:      sr.Name,
		Template:  sr.Template,
		Format:    string(sr.Format),
		Schedule:  sr.Schedule,
		Enabled:   sr.Enabled,
		LastRun:   "",
		NextRun:   "",
		CreatedAt: sr.CreatedAt.Format(time.RFC3339),
		UpdatedAt: sr.UpdatedAt.Format(time.RFC3339),
	}
	if sr.LastRun != nil {
		info.LastRun = sr.LastRun.Format(time.RFC3339)
	}
	if sr.NextRun != nil {
		info.NextRun = sr.NextRun.Format(time.RFC3339)
	}
	return info
}

// reportScheduler resolves the scheduler, or writes the unavailable response
// and reports false, for the same reason reportGenerator does.
func (s *Server) reportScheduler(
	w http.ResponseWriter,
	r *http.Request,
) (*reporting.SchedulerService, bool) {
	if s.background == nil || s.background.Reporting == nil {
		sendErrorResponseWithDetails(w, logging.FromContext(r.Context()),
			http.StatusServiceUnavailable, ErrCodeServiceUnavail,
			i18n.FromRequest(r).T("errors.reports.unavailable"), "")
		return nil, false
	}
	return s.background.Reporting.Scheduler(), true
}

// handleReportSchedules serves GET and POST on /api/v1/reports/schedules.
func (s *Server) handleReportSchedules(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if r.Method == http.MethodPost {
		var req ReportScheduleRequest
		if !decodeScheduleRequest(w, r, &req) {
			return
		}
		sched, ok := s.reportScheduler(w, r)
		if !ok {
			return
		}
		sr := req.toScheduledReport("")
		if err := sched.Create(r.Context(), sr); err != nil {
			s.sendScheduleError(w, r, err)
			return
		}
		logger.InfoContext(r.Context(), "report schedule created",
			"event", "report.schedule.created", "schedule_id", sr.ID)
		sendJSONResponse(w, logger, http.StatusCreated, toReportScheduleInfo(sr))
		return
	}

	sched, ok := s.reportScheduler(w, r)
	if !ok {
		return
	}
	schedules, err := sched.List(r.Context())
	if err != nil {
		s.sendScheduleError(w, r, err)
		return
	}
	// The scheduler holds a map; order by creation so the list is stable.
	slices.SortFunc(schedules, func(a, b reporting.ScheduledReport) int {
		return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), strings.Compare(a.ID, b.ID))
	})
	resp := ReportSchedulesResponse{Schedules: make([]ReportScheduleInfo, 0, len(schedules))}
	for i := range schedules {
		resp.Schedules = append(resp.Schedules, toReportScheduleInfo(&schedules[i]))
	}
	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// handleReportScheduleByID serves GET, PUT and DELETE on
// /api/v1/reports/schedules/{id}.
func (s *Server) handleReportScheduleByID(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	id := strings.TrimPrefix(r.URL.Path, reportSchedulesPathPrefix)
	if !reportIDPattern.MatchString(id) {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, i18n.FromRequest(r).T("errors.reports.invalidScheduleID"), "")
		return
	}

	var req ReportScheduleRequest
	if r.Method == http.MethodPut && !decodeScheduleRequest(w, r, &req) {
		return
	}

	sched, ok := s.reportScheduler(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodPut:
		sr := req.toScheduledReport(id)
		if err := sched.Update(r.Context(), sr); err != nil {
			s.sendScheduleError(w, r, err)
			return
		}
		logger.InfoContext(r.Context(), "report schedule updated", "event", "report.schedule.updated")
		sendJSONResponse(w, logger, http.StatusOK, toReportScheduleInfo(sr))
	case http.MethodDelete:
		if err := sched.Delete(r.Context(), id); err != nil {
			s.sendScheduleError(w, r, err)
			return
		}
		logger.InfoContext(r.Context(), "report schedule deleted", "event", "report.schedule.deleted")
		w.WriteHeader(http.StatusNoContent)
	default:
		sr, err := sched.Get(r.Context(), id)
		if err != nil {
			s.sendScheduleError(w, r, err)
			return
		}
		sendJSONResponse(w, logger, http.StatusOK, toReportScheduleInfo(sr))
	}
}

// decodeScheduleRequest decodes strictly and then enforces the struct tags; the
// scheduler validates the schedule itself.
func decodeScheduleRequest(w http.ResponseWriter, r *http.Request, req *ReportScheduleRequest) bool {
	localizer := i18n.FromRequest(r)
	return decodeJSONStrictLocalized(w, r, req, MaxBodySizeJSON, logging.FromContext(r.Context()), localizer) &&
		validateStruct(w, r, req, localizer)
}

func (req *ReportScheduleRequest) toScheduledReport(id string) *reporting.ScheduledReport {
	return &reporting.ScheduledReport{
		ID:       id,
		Name:     req.Name,
		Template: req.Template,
		Format:   reporting.ExportFormat(req.Format),
		Schedule: req.Schedule,
		Enabled:  *req.Enabled,
	}
}

// sendScheduleError maps a scheduler error onto the response. A validation
// error carries its reason in details: it is our own text naming the field,
// and the editor needs it to say what to fix.
func (s *Server) sendScheduleError(w http.ResponseWriter, r *http.Request, err error) {
	logger := logging.FromContext(r.Context())
	t := i18n.FromRequest(r)

	switch {
	case errors.Is(err, reporting.ErrInvalidSchedule):
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, t.T("errors.reports.invalidSchedule"), err.Error())
	case errors.Is(err, reporting.ErrScheduleNotFound):
		sendErrorResponseWithDetails(w, logger, http.StatusNotFound,
			ErrCodeNotFound, t.T("errors.reports.scheduleNotFound"), "")
	default:
		logger.ErrorContext(r.Context(), "saving report schedule failed",
			"event", "report.schedule.failed", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, t.T("errors.reports.scheduleSaveFailed"), "")
	}
}
