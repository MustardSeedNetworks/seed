// Package support holds the operator-facing diagnostics and recovery
// archives: the redacted support bundle, and backup and restore of an
// install's state.
package support

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
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
	// opaqueURLKey names settings whose URL is itself the credential: a
	// webhook or receiver endpoint carries its token in the path or query.
	opaqueURLKey = regexp.MustCompile(`(?i)(webhook|receiver|notification)`)
	// secretQueryParam names the query parameters that carry a credential.
	secretQueryParam = regexp.MustCompile(`(?i)(token|secret|password|passwd|api[_-]?key|key|sig|signature|auth|code)`)
	urlUserinfo      = regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.-]*://[^/\s:@]*:)[^@\s/]+@`)

	privateKeyBlock = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`)
	bearerPattern   = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/=-]+`)
	jwtPattern      = regexp.MustCompile(`eyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*`)
	// credentialAssignment catches a key=value or "key": "value" pair whose key
	// names a credential, for secrets this package was never told about. The
	// key may carry a snake_case, kebab-case or camelCase prefix
	// (smtp_password, snmp-community, apiToken); the keyword must end the key,
	// so max_tokens and tokenizer are left alone.
	credentialAssignment = regexp.MustCompile(
		`(?i)((?:^|[^A-Za-z0-9])[A-Za-z0-9_-]*` +
			`(?:token|secret|password|passphrase|community|api[_-]?key|license[_-]?key))` +
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
	doc = redactValue(doc, "", &secrets)
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(doc); err != nil {
		return nil, nil, fmt.Errorf("render config: %w", err)
	}
	return out.Bytes(), secrets, nil
}

func redactValue(value any, key string, secrets *[]string) any {
	switch typed := value.(type) {
	case map[string]any:
		for childKey, child := range typed {
			typed[childKey] = redactValue(child, childKey, secrets)
		}
		return typed
	case []any:
		for i, child := range typed {
			typed[i] = redactValue(child, key, secrets)
		}
		return typed
	case string:
		return redactString(typed, key, secrets)
	default:
		return value
	}
}

func redactString(value, key string, secrets *[]string) string {
	if value == "" {
		return value
	}
	if IsSensitiveKey(key) {
		remember(secrets, value)
		return Placeholder
	}
	parsed, ok := parseURL(value)
	if !ok {
		return value
	}
	if opaqueURLKey.MatchString(key) {
		remember(secrets, value)
		remember(secrets, lastSegment(parsed.Path))
		for _, values := range parsed.Query() {
			for _, v := range values {
				remember(secrets, v)
			}
		}
		return Placeholder
	}
	return redactURL(value, parsed, secrets)
}

func remember(secrets *[]string, value string) {
	if len(value) >= minScrubLen {
		*secrets = append(*secrets, value)
	}
}

func parseURL(value string) (*url.URL, bool) {
	if !strings.Contains(value, "://") {
		return nil, false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, false
	}
	return parsed, true
}

func lastSegment(p string) string {
	return p[strings.LastIndex(p, "/")+1:]
}

// redactURL removes the credential parts of a URL that is not credential-named
// as a whole: a userinfo password (or a bare userinfo token) and the values of
// credential-named query parameters. The rest of the URL stays readable.
func redactURL(raw string, parsed *url.URL, secrets *[]string) string {
	changed := false
	if parsed.User != nil {
		if password, has := parsed.User.Password(); has {
			remember(secrets, password)
			parsed.User = url.UserPassword(parsed.User.Username(), Placeholder)
		} else {
			remember(secrets, parsed.User.Username())
			parsed.User = url.User(Placeholder)
		}
		changed = true
	}
	query := parsed.Query()
	for name, values := range query {
		if !secretQueryParam.MatchString(name) {
			continue
		}
		for i, v := range values {
			remember(secrets, v)
			values[i] = Placeholder
			changed = true
		}
	}
	if !changed {
		return raw
	}
	parsed.RawQuery = query.Encode()
	return strings.ReplaceAll(parsed.String(), url.QueryEscape(Placeholder), Placeholder)
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
	text = urlUserinfo.ReplaceAllString(text, "${1}"+Placeholder+"@")
	text = bearerPattern.ReplaceAllString(text, "Bearer "+Placeholder)
	text = jwtPattern.ReplaceAllString(text, Placeholder)
	return credentialAssignment.ReplaceAllString(text, "$1$2"+Placeholder)
}
