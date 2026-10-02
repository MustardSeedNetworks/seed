package i18n

import (
	"fmt"
	"regexp"

	"github.com/nicksnyder/go-i18n/v2/i18n/template"
)

// placeholder matches the i18next interpolation the locale files are written
// in. The files are shared with the UI, so `{{service}}` is the syntax of
// record; go-i18n's default text/template parser reads it as a call to an
// undefined function, the lookup fails, and the client receives the raw key.
var placeholder = regexp.MustCompile(`\{\{\s*(\w+)\s*\}\}`)

// interpolation is a go-i18n template.Parser for that syntax.
type interpolation struct{}

func (interpolation) Cacheable() bool { return true }

func (interpolation) Parse(src, _, _ string) (template.ParsedTemplate, error) {
	return interpolated(src), nil
}

type interpolated string

// Execute substitutes each placeholder from data, a map[string]any. A
// placeholder with no value is an error, so the caller's fallback applies
// instead of a message that shows the operator `{{service}}`.
func (s interpolated) Execute(data any) (string, error) {
	values, _ := data.(map[string]any)
	var missing error
	out := placeholder.ReplaceAllStringFunc(string(s), func(match string) string {
		name := placeholder.FindStringSubmatch(match)[1]
		value, ok := values[name]
		if !ok {
			missing = fmt.Errorf("no value for placeholder %q", name)

			return match
		}

		return fmt.Sprint(value)
	})
	if missing != nil {
		return "", missing
	}

	return out, nil
}
