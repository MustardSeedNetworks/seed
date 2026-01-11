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
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"

	"github.com/krisarmstrong/seed/internal/logging"
)

//go:embed locales
var localesFS embed.FS

// Bundle accessor functions use closure-encapsulated state for thread-safe singleton access.
// getBundle returns the global i18n bundle instance.
// setBundle sets the global i18n bundle instance.
// getBundleOnce returns the sync.Once for lazy bundle initialization.
//
//nolint:gochecknoglobals // Intentional thread-safe singleton using closure pattern
var (
	getBundle, setBundle, getBundleOnce = func() (
		func() *i18n.Bundle,
		func(*i18n.Bundle),
		func() *sync.Once,
	) {
		var (
			mu         sync.RWMutex
			bundle     *i18n.Bundle
			bundleOnce sync.Once
		)

		return func() *i18n.Bundle {
				mu.RLock()
				defer mu.RUnlock()
				return bundle
			}, func(b *i18n.Bundle) {
				mu.Lock()
				defer mu.Unlock()
				bundle = b
			}, func() *sync.Once {
				return &bundleOnce
			}
	}()
)

// DefaultLanguage is the fallback language.
const DefaultLanguage = "en"

// GetSupportedLanguages returns all available language codes.
func GetSupportedLanguages() []string {
	return []string{"en", "es"}
}

// getNamespaces returns all translation file namespaces.
func getNamespaces() []string {
	return []string{"common", "cards", "settings", "errors", "validation", "api", "help"}
}

// loadLocaleFile reads and parses a locale JSON file from the embedded filesystem.
// Returns the parsed messages map and true if successful, nil and false otherwise.
func loadLocaleFile(lang, ns string) (map[string]any, bool) {
	path := fmt.Sprintf("locales/%s/%s.json", lang, ns)
	data, err := localesFS.ReadFile(path)
	if err != nil {
		// File may not exist for all namespaces - this is expected
		return nil, false
	}

	var messages map[string]any
	if unmarshalErr := json.Unmarshal(data, &messages); unmarshalErr != nil {
		return nil, false
	}

	return messages, true
}

// addMessagesToBundle adds flattened messages to the bundle for a specific language.
func addMessagesToBundle(bundle *i18n.Bundle, lang string, flatMessages map[string]any) {
	langTag := language.Make(lang)
	for key, value := range flatMessages {
		strValue, ok := value.(string)
		if !ok {
			continue
		}
		if addErr := bundle.AddMessages(langTag, &i18n.Message{
			ID:    key,
			Other: strValue,
		}); addErr != nil {
			logging.GetLogger().WarnContext(
				context.Background(),
				"failed to add i18n message",
				"key", key,
				"lang", lang,
				"error", addErr,
			)
		}
	}
}

// loadNamespaceForLanguage loads a single namespace for a language into the bundle.
func loadNamespaceForLanguage(bundle *i18n.Bundle, lang, ns string) {
	messages, ok := loadLocaleFile(lang, ns)
	if !ok {
		return
	}

	flatMessages := flattenMessages(messages, ns)
	addMessagesToBundle(bundle, lang, flatMessages)
}

// loadAllTranslations loads all translations for all supported languages into the bundle.
func loadAllTranslations(bundle *i18n.Bundle) {
	for _, lang := range GetSupportedLanguages() {
		for _, ns := range getNamespaces() {
			loadNamespaceForLanguage(bundle, lang, ns)
		}
	}
}

// GetBundle returns the singleton i18n bundle with all translations loaded.
func GetBundle() *i18n.Bundle {
	getBundleOnce().Do(func() {
		newBundle := i18n.NewBundle(language.English)
		newBundle.RegisterUnmarshalFunc("json", json.Unmarshal)
		loadAllTranslations(newBundle)
		setBundle(newBundle)
	})

	return getBundle()
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
			maps.Copy(result, flattenMessages(v, fullKey))
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
	if slices.Contains(GetSupportedLanguages(), lang) {
		return lang
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
	return slices.Contains(GetSupportedLanguages(), lang)
}

// TranslationEntry represents a single translation key with all language values.
// Used for exporting translations for translator review.
type TranslationEntry struct {
	Key     string            `json:"key"`
	Values  map[string]string `json:"values"`
	Missing []string          `json:"missing,omitempty"`
}

// getAllFlatMessages loads and flattens all messages for a language.
func getAllFlatMessages(lang string) map[string]string {
	result := make(map[string]string)
	for _, ns := range getNamespaces() {
		messages, ok := loadLocaleFile(lang, ns)
		if !ok {
			continue
		}
		flatMessages := flattenMessages(messages, ns)
		for k, v := range flatMessages {
			if strVal, isString := v.(string); isString {
				result[k] = strVal
			}
		}
	}
	return result
}

// ExportTranslations returns all translations in a format suitable for review.
// This is useful for handing off to translators to review/improve translations.
func ExportTranslations() []TranslationEntry {
	// Load all messages for each language
	allLangMessages := make(map[string]map[string]string)
	for _, lang := range GetSupportedLanguages() {
		allLangMessages[lang] = getAllFlatMessages(lang)
	}

	// Collect all unique keys across all languages
	allKeys := make(map[string]bool)
	for _, msgs := range allLangMessages {
		for key := range msgs {
			allKeys[key] = true
		}
	}

	// Build entries
	entries := make([]TranslationEntry, 0, len(allKeys))
	for key := range allKeys {
		entry := TranslationEntry{
			Key:    key,
			Values: make(map[string]string),
		}

		for _, lang := range GetSupportedLanguages() {
			if val, ok := allLangMessages[lang][key]; ok {
				entry.Values[lang] = val
			} else {
				entry.Missing = append(entry.Missing, lang)
			}
		}

		entries = append(entries, entry)
	}

	return entries
}

// FindMissingTranslations returns keys that are missing in a specific language.
// Compares against English (the base language).
func FindMissingTranslations(targetLang string) []string {
	targetLang = normalizeLanguage(targetLang)

	englishMsgs := getAllFlatMessages(DefaultLanguage)
	targetMsgs := getAllFlatMessages(targetLang)

	missing := make([]string, 0)
	for key := range englishMsgs {
		if _, ok := targetMsgs[key]; !ok {
			missing = append(missing, key)
		}
	}

	return missing
}

// CompareTranslations returns side-by-side comparison for translator review.
// Each entry contains: key, english value, target value, and status.
// Status is one of: "translated", "needs_review" (identical to English), or "missing".
func CompareTranslations(targetLang string) []map[string]string {
	targetLang = normalizeLanguage(targetLang)

	englishMsgs := getAllFlatMessages(DefaultLanguage)
	targetMsgs := getAllFlatMessages(targetLang)

	result := make([]map[string]string, 0, len(englishMsgs))
	for key, enVal := range englishMsgs {
		entry := map[string]string{
			"key":     key,
			"english": enVal,
		}

		if targetVal, ok := targetMsgs[key]; ok {
			entry["target"] = targetVal
			// Flag if English and target are identical (might need translation)
			if enVal == targetVal {
				entry["status"] = "needs_review"
			} else {
				entry["status"] = "translated"
			}
		} else {
			entry["target"] = ""
			entry["status"] = "missing"
		}

		result = append(result, entry)
	}

	return result
}
