package i18n_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/nicksnyder/go-i18n/v2/i18n/template"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
)

// keyCall matches a literal key passed to localizer.T, to TWithData, or as a
// validation MessageKey (rendered through TWithData). Group 1 is the form, so
// a key that is given data is not judged by a lookup without it. Keys built at
// run time are out of reach of a static check and are not what this is for.
//
// The `\s*` is load-bearing: gofmt splits a long TWithData call so the key
// starts the next line, and a matcher without it never saw
// errors.service.notAvailable, which reached the client verbatim.
func keyCall() *regexp.Regexp {
	return regexp.MustCompile(`(\.T\(|\.TWithData\(|MessageKey:)\s*"([a-zA-Z0-9_.]+)"`)
}

// TestEveryLiteralKeyResolves is the backend counterpart of the UI's
// missing-key detector.
//
// Localizer.T returns the key itself when lookup fails, so an unresolvable key
// is not an error — it is the raw string "errors.netif.invalidMode" arriving at
// the client as the user-facing message. Every call site in handlers_network.go
// did exactly that: the namespace is `errors.network.`, not `errors.netif.`,
// and nothing failed (#50).
func TestEveryLiteralKeyResolves(t *testing.T) {
	t.Parallel()

	localizer := i18n.NewLocalizer("en")
	seen := literalKeys(t)
	if len(seen) == 0 {
		t.Fatal("found no translation calls at all; the matcher is broken")
	}

	var stillBroken []string
	for key, use := range seen {
		if resolves(localizer, key, use.withData) {
			continue
		}
		stillBroken = append(stillBroken, use.path+": "+key)
	}
	slices.Sort(stillBroken)
	for _, entry := range stillBroken {
		t.Errorf("%s does not resolve — it would reach the client verbatim", entry)
	}
}

// resolves reports whether key has a message. A key given data only has to
// exist here, since its placeholders are filled at the call; a key looked up
// with T must also render without data.
func resolves(localizer *i18n.Localizer, key string, withData bool) bool {
	if !withData {
		return localizer.T(key) != key
	}
	_, err := localizer.Localize(&goi18n.LocalizeConfig{
		MessageID:      key,
		TemplateParser: template.IdentityParser{},
	})

	return err == nil
}

type keyUse struct {
	path     string
	withData bool
}

// literalKeys collects every literal translation key in the tree with the
// first file it was seen in. A key is withData when any call gives it data.
func literalKeys(t *testing.T) map[string]keyUse {
	t.Helper()

	found := map[string]keyUse{}
	root := filepath.Join("..", "..", "internal")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range keyCall().FindAllStringSubmatch(string(source), -1) {
			use, ok := found[match[2]]
			if !ok {
				use.path = path
			}
			use.withData = use.withData || match[1] != ".T("
			found[match[2]] = use
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}

	return found
}
