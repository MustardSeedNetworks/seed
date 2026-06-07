package api_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/api"
)

type decodeFixture struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

func newDecodeRequest(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
}

func TestDecodeJSONStrict_ValidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := newDecodeRequest(`{"name":"alpha","port":8080}`)
	var dst decodeFixture

	if !api.ExportDecodeJSONStrict(w, r, &dst, api.MaxBodySizeJSON) {
		t.Fatalf("expected success, got %d: %s", w.Code, w.Body.String())
	}
	if dst.Name != "alpha" || dst.Port != 8080 {
		t.Errorf("unexpected decode: %+v", dst)
	}
}

func TestDecodeJSONStrict_RejectsUnknownField(t *testing.T) {
	w := httptest.NewRecorder()
	r := newDecodeRequest(`{"name":"alpha","port":8080,"extra":"oops"}`)
	var dst decodeFixture

	if api.ExportDecodeJSONStrict(w, r, &dst, api.MaxBodySizeJSON) {
		t.Fatal("expected failure for unknown field")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Invalid JSON") {
		t.Errorf("response should mention invalid JSON: %s", w.Body.String())
	}
}

func TestDecodeJSONStrict_RejectsMalformedJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := newDecodeRequest(`{"name":}`)
	var dst decodeFixture

	if api.ExportDecodeJSONStrict(w, r, &dst, api.MaxBodySizeJSON) {
		t.Fatal("expected failure for malformed JSON")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDecodeJSONStrict_RejectsOversizedBody(t *testing.T) {
	// Build a body slightly larger than the tiny cap we'll pass in.
	big := strings.Repeat("x", 100)
	body := `{"name":"` + big + `","port":1}`
	w := httptest.NewRecorder()
	r := newDecodeRequest(body)
	var dst decodeFixture

	if api.ExportDecodeJSONStrict(w, r, &dst, 50) {
		t.Fatal("expected failure for oversized body")
	}
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "too large") {
		t.Errorf("response should mention size: %s", w.Body.String())
	}
}

func TestDecodeJSONStrict_RejectsTrailingData(t *testing.T) {
	// Two concatenated JSON objects — decoder.More() should reject this.
	w := httptest.NewRecorder()
	r := newDecodeRequest(`{"name":"a","port":1}{"name":"b","port":2}`)
	var dst decodeFixture

	if api.ExportDecodeJSONStrict(w, r, &dst, api.MaxBodySizeJSON) {
		t.Fatal("expected failure for trailing data")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "trailing") {
		t.Errorf("response should mention trailing data: %s", w.Body.String())
	}
}

func TestDecodeJSONStrict_RejectsEmptyBody(t *testing.T) {
	w := httptest.NewRecorder()
	r := newDecodeRequest(``)
	var dst decodeFixture

	if api.ExportDecodeJSONStrict(w, r, &dst, api.MaxBodySizeJSON) {
		t.Fatal("expected failure for empty body")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDecodeJSONOrFail_WrapsThroughHandlerContext(t *testing.T) {
	// Smoke test: the HandlerContext method should delegate to the same
	// implementation. Mirror one of the failure paths to prove it.
	w := httptest.NewRecorder()
	r := newDecodeRequest(`{"name":"alpha","port":8080,"extra":1}`)
	c := &api.HandlerContext{
		W:      w,
		R:      r,
		Logger: slog.New(slog.DiscardHandler),
	}
	var dst decodeFixture

	if c.DecodeJSONOrFail(&dst, api.MaxBodySizeJSON) {
		t.Fatal("expected failure for unknown field via HandlerContext")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDecodeJSONStrict_ResponseIsValidJSON(t *testing.T) {
	// Error responses themselves should be valid JSON the frontend can parse.
	w := httptest.NewRecorder()
	r := newDecodeRequest(`not json`)
	var dst decodeFixture

	_ = api.ExportDecodeJSONStrict(w, r, &dst, api.MaxBodySizeJSON)
	var envelope map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error response is not valid JSON: %v\n  body: %s", err, w.Body.String())
	}
	if envelope["code"] != api.ErrCodeValidation {
		t.Errorf("expected code=%s, got %v", api.ErrCodeValidation, envelope["code"])
	}
}
