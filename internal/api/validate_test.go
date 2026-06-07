package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
)

// testLocalizer returns a localizer that surfaces the message key as the
// translated string — sufficient for asserting validation-failure paths
// without depending on real translation bundles.
func testLocalizer(t *testing.T) *i18n.Localizer {
	t.Helper()
	return i18n.NewLocalizer("en")
}

// validateFixture is a DTO with rules that exercise each branch of
// validateStruct: required, len, oneof, gte/lte.
type validateFixture struct {
	Name string `json:"name" validate:"required"`
	Code string `json:"code" validate:"required,numeric,len=6"`
	Mode string `json:"mode" validate:"required,oneof=read write"`
	Port int    `json:"port" validate:"required,gte=1,lte=65535"`
}

func newValidateRequest() *http.Request {
	return httptest.NewRequest(http.MethodPost, "/", nil)
}

func TestValidateStruct_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	r := newValidateRequest()
	dto := validateFixture{Name: "alpha", Code: "123456", Mode: "read", Port: 8080}

	if !api.ExportValidateStruct(w, r, &dto, testLocalizer(t)) {
		t.Fatalf("expected validation to pass, got %d: %s", w.Code, w.Body.String())
	}
}

func TestValidateStruct_MissingRequired(t *testing.T) {
	w := httptest.NewRecorder()
	r := newValidateRequest()
	dto := validateFixture{Code: "123456", Mode: "read", Port: 8080} // missing Name

	if api.ExportValidateStruct(w, r, &dto, testLocalizer(t)) {
		t.Fatal("expected validation to fail for missing required field")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var env map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("error response not valid JSON: %v", err)
	}
	if env["code"] != api.ErrCodeValidation {
		t.Errorf("expected code=%s, got %v", api.ErrCodeValidation, env["code"])
	}
	if !strings.Contains(env["details"].(string), "name") {
		t.Errorf("details should mention `name`: %v", env["details"])
	}
}

func TestValidateStruct_InvalidEnum(t *testing.T) {
	w := httptest.NewRecorder()
	r := newValidateRequest()
	dto := validateFixture{Name: "alpha", Code: "123456", Mode: "delete", Port: 8080}

	if api.ExportValidateStruct(w, r, &dto, testLocalizer(t)) {
		t.Fatal("expected validation to fail for invalid enum value")
	}
	if !strings.Contains(w.Body.String(), "mode") {
		t.Errorf("response should mention `mode`: %s", w.Body.String())
	}
}

func TestValidateStruct_PortOutOfRange(t *testing.T) {
	w := httptest.NewRecorder()
	r := newValidateRequest()
	dto := validateFixture{Name: "alpha", Code: "123456", Mode: "read", Port: 99999}

	if api.ExportValidateStruct(w, r, &dto, testLocalizer(t)) {
		t.Fatal("expected validation to fail for port > 65535")
	}
}

func TestValidateStruct_NonNumericCode(t *testing.T) {
	w := httptest.NewRecorder()
	r := newValidateRequest()
	dto := validateFixture{Name: "alpha", Code: "abcdef", Mode: "read", Port: 8080}

	if api.ExportValidateStruct(w, r, &dto, testLocalizer(t)) {
		t.Fatal("expected validation to fail for non-numeric code")
	}
}

func TestValidateStruct_MultipleFailures_AllReported(t *testing.T) {
	w := httptest.NewRecorder()
	r := newValidateRequest()
	dto := validateFixture{} // every field invalid

	if api.ExportValidateStruct(w, r, &dto, testLocalizer(t)) {
		t.Fatal("expected validation to fail for empty DTO")
	}

	body := w.Body.String()
	for _, field := range []string{"name", "code", "mode", "port"} {
		if !strings.Contains(body, field) {
			t.Errorf("expected response to mention %q: %s", field, body)
		}
	}
}
