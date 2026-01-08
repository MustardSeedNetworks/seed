package i18n_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/krisarmstrong/seed/internal/i18n"
)

func TestGetSupportedLanguages(t *testing.T) {
	languages := i18n.GetSupportedLanguages()

	if len(languages) == 0 {
		t.Error("expected at least one supported language")
	}

	// English must be supported
	if !slices.Contains(languages, "en") {
		t.Error("expected English (en) to be supported")
	}
}

func TestGetBundle(t *testing.T) {
	bundle := i18n.GetBundle()

	if bundle == nil {
		t.Fatal("GetBundle returned nil")
	}

	// Calling GetBundle again should return the same instance (singleton)
	bundle2 := i18n.GetBundle()
	if bundle != bundle2 {
		t.Error("expected GetBundle to return singleton instance")
	}
}

func TestNewLocalizer(t *testing.T) {
	tests := []struct {
		name         string
		lang         string
		expectedLang string
	}{
		{
			name:         "english",
			lang:         "en",
			expectedLang: "en",
		},
		{
			name:         "spanish",
			lang:         "es",
			expectedLang: "es",
		},
		{
			name:         "unsupported language falls back to english",
			lang:         "fr",
			expectedLang: "en",
		},
		{
			name:         "empty language falls back to english",
			lang:         "",
			expectedLang: "en",
		},
		{
			name:         "language with region code",
			lang:         "en-US",
			expectedLang: "en",
		},
		{
			name:         "spanish with region code",
			lang:         "es-MX",
			expectedLang: "es",
		},
		{
			name:         "accept-language format",
			lang:         "es,en;q=0.9",
			expectedLang: "es",
		},
		{
			name:         "accept-language with quality",
			lang:         "en;q=0.9",
			expectedLang: "en",
		},
		{
			name:         "uppercase language",
			lang:         "EN",
			expectedLang: "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localizer := i18n.NewLocalizer(tt.lang)

			if localizer == nil {
				t.Fatal("NewLocalizer returned nil")
			}

			if localizer.Language() != tt.expectedLang {
				t.Errorf("expected language %s, got %s", tt.expectedLang, localizer.Language())
			}
		})
	}
}

func TestLocalizerT(t *testing.T) {
	localizer := i18n.NewLocalizer("en")

	// Test that missing keys return the key itself
	missingKey := "nonexistent.key.for.testing"
	result := localizer.T(missingKey)
	if result != missingKey {
		t.Errorf("expected missing key %q to be returned as-is, got %q", missingKey, result)
	}

	// Test a key that should exist (common translations)
	// Note: The actual key depends on what's in the locale files
	// If no translations exist, it should return the key
	key := "app.title"
	result = localizer.T(key)
	// Result should be either the translation or the key itself
	if result == "" {
		t.Error("expected T() to return non-empty string")
	}
}

func TestLocalizerTWithData(t *testing.T) {
	localizer := i18n.NewLocalizer("en")

	// Test with missing key (should return key)
	missingKey := "test.missing.key.with.data"
	result := localizer.TWithData(missingKey, map[string]any{"value": 42})
	if result != missingKey {
		t.Errorf("expected missing key %q to be returned, got %q", missingKey, result)
	}
}

func TestIsSupported(t *testing.T) {
	// Note: IsSupported normalizes languages first, so unsupported languages
	// normalize to the default "en" which is supported, returning true.
	tests := []struct {
		lang     string
		expected bool
	}{
		{"en", true},
		{"es", true},
		{"fr", true},      // Normalizes to "en" (default), which is supported
		{"de", true},      // Normalizes to "en" (default), which is supported
		{"en-US", true},   // Normalizes to "en"
		{"es-MX", true},   // Normalizes to "es"
		{"", true},        // Normalizes to "en" (default)
		{"invalid", true}, // Normalizes to "en" (default)
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			if got := i18n.IsSupported(tt.lang); got != tt.expected {
				t.Errorf("IsSupported(%q) = %v, want %v", tt.lang, got, tt.expected)
			}
		})
	}
}

func TestMiddleware(t *testing.T) {
	middleware := i18n.Middleware()

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		localizer := i18n.FromRequest(r)
		if localizer == nil {
			t.Error("expected localizer to be set in context")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name           string
		acceptLanguage string
		expectedLang   string
	}{
		{
			name:           "no accept-language header",
			acceptLanguage: "",
			expectedLang:   "en",
		},
		{
			name:           "english accept-language",
			acceptLanguage: "en",
			expectedLang:   "en",
		},
		{
			name:           "spanish accept-language",
			acceptLanguage: "es",
			expectedLang:   "es",
		},
		{
			name:           "complex accept-language",
			acceptLanguage: "es-MX,es;q=0.9,en;q=0.8",
			expectedLang:   "es",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rec.Code)
			}
		})
	}
}

func TestFromContext(t *testing.T) {
	t.Run("with localizer in context", func(t *testing.T) {
		localizer := i18n.NewLocalizer("es")
		ctx := context.WithValue(context.Background(), i18n.LocalizerKey, localizer)

		result := i18n.FromContext(ctx)
		if result == nil {
			t.Fatal("expected localizer from context")
		}
		if result.Language() != "es" {
			t.Errorf("expected language es, got %s", result.Language())
		}
	})

	t.Run("without localizer in context", func(t *testing.T) {
		result := i18n.FromContext(context.Background())
		if result == nil {
			t.Fatal("expected default localizer")
		}
		// Should return default English localizer
		if result.Language() != "en" {
			t.Errorf("expected default language en, got %s", result.Language())
		}
	})
}

func TestFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Without middleware, should return default localizer
	localizer := i18n.FromRequest(req)
	if localizer == nil {
		t.Fatal("expected default localizer")
	}
	if localizer.Language() != "en" {
		t.Errorf("expected default language en, got %s", localizer.Language())
	}
}

func TestDefaultLanguageConstant(t *testing.T) {
	if i18n.DefaultLanguage != "en" {
		t.Errorf("expected DefaultLanguage to be 'en', got %q", i18n.DefaultLanguage)
	}
}
