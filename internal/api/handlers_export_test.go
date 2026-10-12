package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/MustardSeedNetworks/seed/internal/api"
)

// TestExportAnswersExportResponse strict-decodes the diagnostic export, so a
// card key outside the registered ExportResponse fails here, and checks that
// no field is null: the published schema makes none of them nullable.
func TestExportAnswersExportResponse(t *testing.T) {
	srv := newTestEndpointServer(t)
	defer srv.close()

	rec := httptest.NewRecorder()
	srv.server.HandleExport(rec, httptest.NewRequestWithContext(
		t.Context(), http.MethodGet, "/api/v1/reporting/export", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	if got := rec.Header().Get("Content-Disposition"); got != "attachment; filename=seed-export.json" {
		t.Errorf("Content-Disposition = %q", got)
	}
	body := rec.Body.Bytes()
	if bytes.Contains(body, []byte("null")) {
		t.Errorf("export carries a null:\n%s", body)
	}

	var doc api.ExportResponse
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("decode ExportResponse: %v\n%s", err, body)
	}
	if doc.Version == "" || doc.Timestamp == "" || doc.Device.IPMode == "" {
		t.Errorf("document header not populated: %+v", doc)
	}
}
