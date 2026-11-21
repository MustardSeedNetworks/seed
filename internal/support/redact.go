// Package support holds the operator-facing diagnostics and recovery
// archives: the redacted support bundle, and backup and restore of an
// install's state.
package support

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Placeholder replaces every credential a bundle would otherwise carry. One
// fixed string, not a length-preserving mask, so a reader can see a value was
// removed rather than merely obscured.
const Placeholder = "[REDACTED]"

// minScrubLen is the shortest secret value scrubbed from free text by value.
// Shorter values ("on", "1") would rewrite unrelated log words; they are still
// removed from the structured config by key.
const minScrubLen = 4

const (
	sensitiveKeyPattern = `(?i)(password|passwd|passphrase|secret|token|api[_-]?key|consumer[_-]?key|` +
		`community|license|credential|private|bearer|hash)`
	pathLikeKeyPattern = `(?i)(_file|_path|_dir|_url)$`
)

var (
	sensitiveKey = regexp.MustCompile(sensitiveKeyPattern)
	pathLikeKey  = regexp.MustCompile(pathLikeKeyPattern)

	privateKeyBlock = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`)
	bearerPattern   = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/=-]+`)
	jwtPattern      = regexp.MustCompile(`eyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*`)
	// credentialAssignment catches a key=value or "key": "value" pair whose key
	// names a credential, for secrets this package was never told about.
	credentialAssignment = regexp.MustCompile(
		`(?i)\b(token|community|password|passphrase|secret|api[_-]?key|license[_-]?key)\b` +
			`(["']?\s*[:=]\s*)"?[^"\s,}]+"?`)
)

// IsSensitiveKey reports whether a config key names a credential. File, path,
// directory and URL keys are never credentials even when their name contains
// one ("token_url", "key_file"): they name where a secret lives, not the secret.
func IsSensitiveKey(key string) bool {
	return sensitiveKey.MatchString(key) && !pathLikeKey.MatchString(key)
}

// RedactJSON replaces every string under a credential-named key with
// Placeholder and returns the document plus the removed values, so free text
// elsewhere (log lines) can be scrubbed of the same strings.
//
// Redaction is structural over the parsed document, not textual: it covers a
// field added tomorrow as long as its key names a credential, and it never
// rewrites an unrelated occurrence of a word. An unset value stays unset: a
// placeholder in a field the operator never configured would claim a
// credential exists where none does.
func RedactJSON(data []byte) ([]byte, []string, error) {
	var doc any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&doc); err != nil {
		return nil, nil, fmt.Errorf("parse config: %w", err)
	}
	var secrets []string
	doc = redactValue(doc, false, &secrets)
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(doc); err != nil {
		return nil, nil, fmt.Errorf("render config: %w", err)
	}
	return out.Bytes(), secrets, nil
}

func redactValue(value any, sensitive bool, secrets *[]string) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			typed[key] = redactValue(child, IsSensitiveKey(key), secrets)
		}
		return typed
	case []any:
		for i, child := range typed {
			typed[i] = redactValue(child, sensitive, secrets)
		}
		return typed
	case string:
		if !sensitive || typed == "" {
			return typed
		}
		if len(typed) >= minScrubLen {
			*secrets = append(*secrets, typed)
		}
		return Placeholder
	default:
		return value
	}
}

// ScrubText removes known secret values from free text, then masks anything
// still shaped like a credential.
//
// Known values go first because they are exact: a community string that
// happens to read like a word is removed by value, not by pattern. The
// patterns are the backstop for a secret nobody listed.
func ScrubText(text string, secrets []string) string {
	for _, secret := range secrets {
		if len(secret) >= minScrubLen {
			text = strings.ReplaceAll(text, secret, Placeholder)
		}
	}
	text = privateKeyBlock.ReplaceAllString(text, Placeholder)
	text = bearerPattern.ReplaceAllString(text, "Bearer "+Placeholder)
	text = jwtPattern.ReplaceAllString(text, Placeholder)
	return credentialAssignment.ReplaceAllString(text, "$1$2"+Placeholder)
}
