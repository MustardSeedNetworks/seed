// Package i18n provides internationalization support for The Seed backend.
//
// It uses go-i18n for loading and managing translations from JSON locale files.
// Translations are embedded in the binary using go:embed for easy deployment.
//
// Supported languages:
//   - en: English (default)
//   - es: Spanish
//
// Usage:
//
//	localizer := i18n.NewLocalizer("es")
//	msg := localizer.MustLocalize("errors.auth.invalidCredentials")
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales
var localesFS embed.FS

// Bundle holds all loaded translations.
var (
	bundle     *i18n.Bundle
	bundleOnce sync.Once
)

// SupportedLanguages lists all available language codes.
var SupportedLanguages = []string{"en", "es"}

// DefaultLanguage is the fallback language.
const DefaultLanguage = "en"

// namespaces lists all translation file namespaces.
var namespaces = []string{"common", "cards", "settings", "errors", "validation", "api", "help"}

// GetBundle returns the singleton i18n bundle with all translations loaded.
func GetBundle() *i18n.Bundle {
	bundleOnce.Do(func() {
		bundle = i18n.NewBundle(language.English)
		bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

		// Load all locale files for each language
		for _, lang := range SupportedLanguages {
			for _, ns := range namespaces {
				path := fmt.Sprintf("locales/%s/%s.json", lang, ns)
				data, err := localesFS.ReadFile(path)
				if err != nil {
					// Log warning but continue - file may not exist for all namespaces
					continue
				}

				// Parse the JSON and add messages with namespaced keys
				var messages map[string]any
				if err := json.Unmarshal(data, &messages); err != nil {
					continue
				}

				// Flatten nested structure and add to bundle.
				flatMessages := flattenMessages(messages, ns)
				for key, value := range flatMessages {
					if strValue, ok := value.(string); ok {
						//nolint:errcheck // Errors here indicate invalid message format, not runtime issues.
						bundle.AddMessages(language.Make(lang), &i18n.Message{
							ID:    key,
							Other: strValue,
						})
					}
				}
			}
		}
	})

	return bundle
}

// flattenMessages converts nested JSON structure to flat keys.
// Example: {"auth": {"login": "Login"}} -> {"auth.login": "Login"}.
func flattenMessages(messages map[string]any, prefix string) map[string]any {
	result := make(map[string]any)

	for key, value := range messages {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]any:
			// Recursively flatten nested maps
			for k, val := range flattenMessages(v, fullKey) {
				result[k] = val
			}
		case string:
			result[fullKey] = v
		}
	}

	return result
}

// Localizer wraps i18n.Localizer with convenience methods.
type Localizer struct {
	*i18n.Localizer
	lang string
}

// NewLocalizer creates a new localizer for the given language.
// Falls back to English if the language is not supported.
func NewLocalizer(lang string) *Localizer {
	// Normalize language code (e.g., "en-US" -> "en")
	lang = normalizeLanguage(lang)

	return &Localizer{
		Localizer: i18n.NewLocalizer(GetBundle(), lang, DefaultLanguage),
		lang:      lang,
	}
}

// normalizeLanguage extracts the primary language code.
func normalizeLanguage(lang string) string {
	// Handle Accept-Language format: "en-US,en;q=0.9,es;q=0.8"
	if idx := strings.Index(lang, ","); idx != -1 {
		lang = lang[:idx]
	}
	// Handle region codes: "en-US" -> "en"
	if idx := strings.Index(lang, "-"); idx != -1 {
		lang = lang[:idx]
	}
	// Handle quality values: "en;q=0.9" -> "en"
	if idx := strings.Index(lang, ";"); idx != -1 {
		lang = lang[:idx]
	}

	lang = strings.ToLower(strings.TrimSpace(lang))

	// Validate supported language
	for _, supported := range SupportedLanguages {
		if lang == supported {
			return lang
		}
	}

	return DefaultLanguage
}

// T translates a message by key.
// Returns the key itself if translation is not found.
func (l *Localizer) T(key string) string {
	msg, err := l.Localize(&i18n.LocalizeConfig{
		MessageID: key,
	})
	if err != nil {
		// Return key as fallback
		return key
	}
	return msg
}

// TWithData translates a message with template data.
// Example: l.TWithData("validation.port.invalidRange", map[string]any{"value": 70000}).
func (l *Localizer) TWithData(key string, data map[string]any) string {
	msg, err := l.Localize(&i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: data,
	})
	if err != nil {
		// Return key as fallback
		return key
	}
	return msg
}

// Language returns the current language code.
func (l *Localizer) Language() string {
	return l.lang
}

// IsSupported checks if a language code is supported.
func IsSupported(lang string) bool {
	lang = normalizeLanguage(lang)
	for _, supported := range SupportedLanguages {
		if lang == supported {
			return true
		}
	}
	return false
}
