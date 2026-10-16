package api

// AutoTest record endpoints (P-E2):
//
//   GET  /api/v1/autotest/results                         list, newest first
//   POST /api/v1/autotest/results/import                  re-import a JSON export
//   GET  /api/v1/autotest/results/{resultId}              fetch one
//   GET  /api/v1/autotest/results/{resultId}/export       ?format=json|csv|pdf
//
// Every AutoTest job keeps its run as a record. The JSON export is the record
// itself, so importing it restores the same record, id and labels included, on
// this Seed or another. Export and import are data extraction and sit behind
// export_csv_json (Starter) like the other exports; PDF is audit_pdf (Pro), as
// it is for reports.

import (
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/autotest"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

const autotestResultsPath = APIVersionPrefix + "/autotest/results"

// AutotestResultListResponse is the GET /autotest/results envelope.
type AutotestResultListResponse struct {
	Count   int               `json:"count"`
	Results []autotest.Report `json:"results"`
}

func (s *Server) autotestResultRoutes() []route.Route {
	return []route.Route{
		{
			Path:    autotestResultsPath,
			Handler: s.handleAutotestResults,
			Methods: []string{http.MethodGet},
			Auth:    true,
		},
		{
			Path:    autotestResultsPath + "/import",
			Handler: s.handleAutotestResultImport,
			Methods: []string{http.MethodPost},
			Feature: "export_csv_json",
			Scope:   roles.Operator,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    autotestResultsPath + "/{resultId}",
			Handler: s.handleAutotestResult,
			Methods: []string{http.MethodGet},
			Auth:    true,
		},
		{
			Path:    autotestResultsPath + "/{resultId}/export",
			Handler: s.handleAutotestResultExport,
			Methods: []string{http.MethodGet},
			Feature: "export_csv_json",
			Auth:    true,
		},
	}
}

func (s *Server) handleAutotestResults(w http.ResponseWriter, r *http.Request) {
	results, err := s.autotestResults.List(r.Context())
	if err != nil {
		writeAutotestResultError(w, r, err)
		return
	}
	if results == nil {
		results = []autotest.Report{}
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK,
		AutotestResultListResponse{Count: len(results), Results: results})
}

func (s *Server) handleAutotestResult(w http.ResponseWriter, r *http.Request) {
	rep, ok := s.autotestResult(w, r)
	if !ok {
		return
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, rep)
}

func (s *Server) handleAutotestResultExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	var contentType string
	switch format {
	case "json":
		contentType = "application/json"
	case "csv":
		contentType = "text/csv; charset=utf-8"
	case "pdf":
		// The route gate is export_csv_json; a PDF is sold as Pro, and format
		// is a query value the route gate cannot see.
		if !s.hasFeature("audit_pdf") {
			s.sendFeatureGate(w, r, "audit_pdf")
			return
		}
		contentType = "application/pdf"
	default:
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, "format must be json, csv or pdf")
		return
	}
	rep, ok := s.autotestResult(w, r)
	if !ok {
		return
	}
	var (
		body []byte
		err  error
	)
	switch format {
	case "csv":
		body, err = autotest.CSV(rep)
	case "pdf":
		body, err = autotest.PDF(rep)
	}
	if err != nil {
		writeAutotestResultError(w, r, err)
		return
	}
	// The filename is the record's UUID, validated on the way in, never
	// operator text.
	w.Header().Set("Content-Disposition", `attachment; filename="autotest-`+rep.ID+`.`+format+`"`)
	if format == "json" {
		sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, rep)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if _, writeErr := w.Write(body); writeErr != nil {
		logging.FromContext(r.Context()).WarnContext(r.Context(), "autotest export truncated", "error", writeErr)
	}
}

func (s *Server) handleAutotestResultImport(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBodySizeJSON))
	if err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			writeError(w, r, http.StatusRequestEntityTooLarge, ErrCodeValidation, "Request body too large")
			return
		}
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, "Request body could not be read")
		return
	}
	rep, err := autotest.ParseRecord(data)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, "Not an AutoTest JSON export: "+err.Error())
		return
	}
	if err = s.autotestResults.Insert(r.Context(), rep); err != nil {
		writeAutotestResultError(w, r, err)
		return
	}
	w.Header().Set("Location", autotestResultsPath+"/"+rep.ID)
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusCreated, rep)
}

// autotestResult reads the record the path names, writing the error and
// reporting false when there is none.
func (s *Server) autotestResult(w http.ResponseWriter, r *http.Request) (*autotest.Report, bool) {
	id := r.PathValue("resultId")
	if uuid.Validate(id) != nil {
		writeAutotestResultError(w, r, autotest.ErrNotFound)
		return nil, false
	}
	rep, err := s.autotestResults.Get(r.Context(), id)
	if err != nil {
		writeAutotestResultError(w, r, err)
		return nil, false
	}
	return rep, true
}

func writeAutotestResultError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, autotest.ErrNotFound):
		writeError(w, r, http.StatusNotFound, ErrCodeNotFound, "AutoTest result not found")
	case errors.Is(err, autotest.ErrExists):
		writeError(w, r, http.StatusConflict, ErrCodeConflict, "This AutoTest result is already stored")
	case errors.Is(err, autotest.ErrUnavailable):
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "Database not initialized")
	default:
		logging.FromContext(r.Context()).ErrorContext(r.Context(), "autotest result request failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, ErrCodeInternal, "AutoTest result request failed")
	}
}
