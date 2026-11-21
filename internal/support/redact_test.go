package support_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/support"
)

func TestRedactJSON(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		wantKept    []string
		wantGone    []string
		wantSecrets []string
	}{
		{
			name:        "nested credential keys",
			in:          `{"auth":{"jwt_secret":"jwt-value-123","default_password_hash":"$2a$hash"},"server":{"port":8443}}`,
			wantKept:    []string{`"port": 8443`},
			wantGone:    []string{"jwt-value-123", "$2a$hash"},
			wantSecrets: []string{"jwt-value-123", "$2a$hash"},
		},
		{
			name:        "arrays of objects",
			in:          `{"sso":{"providers":[{"name":"okta","client_secret":"client-secret-1"},{"name":"entra","client_secret":"client-secret-2"}]}}`,
			wantKept:    []string{`"name": "okta"`, `"name": "entra"`},
			wantGone:    []string{"client-secret-1", "client-secret-2"},
			wantSecrets: []string{"client-secret-1", "client-secret-2"},
		},
		{
			name:        "array of strings under a credential key",
			in:          `{"api_tokens":["token-aaaa","token-bbbb"]}`,
			wantGone:    []string{"token-aaaa", "token-bbbb"},
			wantSecrets: []string{"token-aaaa", "token-bbbb"},
		},
		{
			name:        "webhook url is sensitive entirely",
			in:          `{"alerts":{"webhook_url":"https://hooks.example.com/services/T000/B000/XXXXSECRETXXXX"}}`,
			wantGone:    []string{"hooks.example.com", "XXXXSECRETXXXX"},
			wantSecrets: []string{"https://hooks.example.com/services/T000/B000/XXXXSECRETXXXX", "XXXXSECRETXXXX"},
		},
		{
			name:     "receiver and notification urls too",
			in:       `{"a":{"receiver":"https://r.example/in/tok-aaaa1111"},"b":{"notification_endpoint":"https://n.example/x?y=tok-bbbb2222"}}`,
			wantGone: []string{"tok-aaaa1111", "tok-bbbb2222"},
			wantSecrets: []string{
				"https://r.example/in/tok-aaaa1111",
				"tok-aaaa1111",
				"https://n.example/x?y=tok-bbbb2222",
				"tok-bbbb2222",
			},
		},
		{
			name:        "userinfo password in an ordinary url",
			in:          `{"integrations":{"base":"https://admin:hunter2000@meraki.example.com/api/v1"}}`,
			wantKept:    []string{"admin", "meraki.example.com/api/v1"},
			wantGone:    []string{"hunter2000"},
			wantSecrets: []string{"hunter2000"},
		},
		{
			name:        "userinfo token with no password",
			in:          `{"feed":"https://tok-cccc3333@feed.example.com/x"}`,
			wantKept:    []string{"feed.example.com/x"},
			wantGone:    []string{"tok-cccc3333"},
			wantSecrets: []string{"tok-cccc3333"},
		},
		{
			name:        "credential query parameters",
			in:          `{"feed":"https://feed.example.com/x?format=json&api_key=key-dddd4444&sig=sig-eeee5555"}`,
			wantKept:    []string{"feed.example.com/x", "format=json"},
			wantGone:    []string{"key-dddd4444", "sig-eeee5555"},
			wantSecrets: []string{"key-dddd4444", "sig-eeee5555"},
		},
		{
			name:     "plain urls and non-urls are untouched",
			in:       `{"a":"https://example.com/docs?page=2","b":"not://","c":"host:8443"}`,
			wantKept: []string{"https://example.com/docs?page=2", "not://", "host:8443"},
		},
		{
			name:     "unset credential stays unset",
			in:       `{"auth":{"jwt_secret":""}}`,
			wantKept: []string{`"jwt_secret": ""`},
		},
		{
			name:     "paths and urls naming a secret are not secrets",
			in:       `{"server":{"key_file":"/etc/seed/server.key","token_url":"https://idp.example/token"}}`,
			wantKept: []string{"/etc/seed/server.key", "https://idp.example/token"},
		},
		{
			name:     "non-string values under a credential key are kept",
			in:       `{"token_ttl":3600,"license_enforced":true}`,
			wantKept: []string{`"token_ttl": 3600`, `"license_enforced": true`},
		},
		{
			name:        "short secrets leave the structure but not the scrub list",
			in:          `{"snmp":{"community":"ab"}}`,
			wantGone:    []string{`"ab"`},
			wantSecrets: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, secrets, err := support.RedactJSON([]byte(tc.in))
			if err != nil {
				t.Fatalf("RedactJSON: %v", err)
			}
			text := string(out)
			checkText(t, text, tc.wantKept, tc.wantGone)
			if len(tc.wantGone) > 0 && !strings.Contains(text, support.Placeholder) {
				t.Errorf("output has no placeholder:\n%s", text)
			}
			if !sameSet(secrets, tc.wantSecrets) {
				t.Errorf("secrets = %v, want %v", secrets, tc.wantSecrets)
			}
		})
	}
}

func checkText(t *testing.T, text string, kept, gone []string) {
	t.Helper()
	for _, want := range kept {
		if !strings.Contains(text, want) {
			t.Errorf("output lost %q:\n%s", want, text)
		}
	}
	for _, unwanted := range gone {
		if strings.Contains(text, unwanted) {
			t.Errorf("output still carries %q:\n%s", unwanted, text)
		}
	}
}

func TestRedactJSONRejectsMalformedInput(t *testing.T) {
	if _, _, err := support.RedactJSON([]byte(`{"auth":`)); err == nil {
		t.Fatal("RedactJSON accepted truncated JSON")
	}
}

func sameSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]int)
	for _, value := range got {
		seen[value]++
	}
	for _, value := range want {
		seen[value]--
	}
	for _, count := range seen {
		if count != 0 {
			return false
		}
	}
	return true
}

type leaf struct {
	key      string
	sentinel string
}

// TestRedactJSONCoversEveryConfigField fills every string field of
// config.Config with a unique sentinel and proves that none of the ones whose
// name looks like a credential survives redaction. A credential field added to
// the config later fails here until the redactor learns its name, rather than
// shipping in a support bundle.
func TestRedactJSONCoversEveryConfigField(t *testing.T) {
	var leaves []leaf
	doc := shape(reflect.TypeFor[config.Config](), "", &leaves)

	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal populated config: %v", err)
	}
	out, _, err := support.RedactJSON(raw)
	if err != nil {
		t.Fatalf("RedactJSON: %v", err)
	}
	text := string(out)

	secretish := 0
	for _, l := range leaves {
		if !looksLikeCredential(l.key) {
			continue
		}
		secretish++
		if strings.Contains(text, l.sentinel) {
			t.Errorf("credential-looking config key %q survived redaction (%s)", l.key, l.sentinel)
		}
	}
	const minimumCredentialFields = 10
	if secretish < minimumCredentialFields {
		t.Fatalf("found only %d credential-looking config fields; the walk is not reaching them", secretish)
	}
}

// looksLikeCredential is deliberately broader than the redactor's own pattern:
// it is the independent check that the redactor's pattern is wide enough.
var opaqueURLName = regexp.MustCompile(`(?i)(webhook|receiver|notification)`)

func looksLikeCredential(key string) bool {
	if opaqueURLName.MatchString(key) {
		return true
	}
	if support.IsSensitiveKey(key) {
		return true
	}
	lower := strings.ToLower(key)
	for _, suffix := range []string{"_file", "_path", "_dir", "_url"} {
		if strings.HasSuffix(lower, suffix) {
			return false
		}
	}
	for _, word := range []string{"password", "secret", "token", "key", "community", "hash", "credential", "license"} {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}

// shape returns the JSON document a fully populated value of t's type would
// be, with every string leaf set to a unique sentinel. It builds the document
// from the type rather than marshalling a value, so a custom MarshalJSON that
// hides a field cannot hide it from this test too.
func shape(t reflect.Type, key string, leaves *[]leaf) any {
	if t.Kind() == reflect.Pointer {
		return shape(t.Elem(), key, leaves)
	}
	if t.Kind() == reflect.String {
		sentinel := fmt.Sprintf("SENTINEL_%s_%d", key, len(*leaves))
		if opaqueURLName.MatchString(key) {
			sentinel = "https://SENTINEL-" + strings.ReplaceAll(sentinel, "_", "-") + ".example/hook/abc"
		}
		*leaves = append(*leaves, leaf{key: key, sentinel: sentinel})
		return sentinel
	}
	if t.Kind() == reflect.Slice {
		return []any{shape(t.Elem(), key, leaves)}
	}
	if t.Kind() == reflect.Map {
		return map[string]any{"entry": shape(t.Elem(), key, leaves)}
	}
	if t.Kind() != reflect.Struct || t.PkgPath() == "time" {
		return nil
	}
	doc := make(map[string]any)
	for field := range t.Fields() {
		if !field.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = field.Name
		}
		doc[name] = shape(field.Type, name, leaves)
	}
	return doc
}

func TestScrubText(t *testing.T) {
	const pemKey = "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASC\n-----END PRIVATE KEY-----"
	jwt := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dBjftJeZ4CVPmB92K27uhbUJU1p1r"
	tests := []struct {
		name    string
		in      string
		secrets []string
		gone    []string
		kept    []string
	}{
		{
			name:    "known value anywhere in the line",
			in:      "poller started community=hunter2000 for 10.0.0.1 and hunter2000 again",
			secrets: []string{"hunter2000"},
			gone:    []string{"hunter2000"},
			kept:    []string{"10.0.0.1"},
		},
		{
			name: "bearer header",
			in:   "Authorization: Bearer abc.DEF-123_xyz== rejected",
			gone: []string{"abc.DEF-123_xyz=="},
			kept: []string{"rejected"},
		},
		{
			name: "jwt",
			in:   "session " + jwt + " expired",
			gone: []string{jwt},
			kept: []string{"expired"},
		},
		{
			name: "private key block",
			in:   "loaded\n" + pemKey + "\nnext line",
			gone: []string{"MIIEvQIBADANBgkqhkiG9w0BAQEFAASC", "BEGIN PRIVATE KEY"},
			kept: []string{"next line"},
		},
		{
			name: "credential assignments in both styles",
			in:   `login password=s3cretvalue ok; {"api_key": "k-12345-abc", "level": "info"}`,
			gone: []string{"s3cretvalue", "k-12345-abc"},
			kept: []string{`"level": "info"`},
		},
		{
			name: "snake_case, kebab-case and camelCase key prefixes",
			in: "smtp_password=pw-aaaa1111 snmp-community: com-bbbb2222 apiToken=tok-cccc3333 " +
				`{"db_secret":"sec-dddd4444"} x-api-key=key-eeee5555 SMTP_PASSWORD=pw-ffff6666`,
			gone: []string{
				"pw-aaaa1111",
				"com-bbbb2222",
				"tok-cccc3333",
				"sec-dddd4444",
				"key-eeee5555",
				"pw-ffff6666",
			},
			kept: []string{"smtp_password", "snmp-community", "apiToken", "db_secret"},
		},
		{
			name: "keywords that do not end the key are not credentials",
			in:   "max_tokens=100 tokenizer=fast password_policy=strict secretary=jo communityname=lab",
			kept: []string{
				"max_tokens=100",
				"tokenizer=fast",
				"password_policy=strict",
				"secretary=jo",
				"communityname=lab",
			},
		},
		{
			name: "url userinfo password in a log line",
			in:   "fetch https://admin:hunter2000@meraki.example.com/api failed",
			gone: []string{"hunter2000"},
			kept: []string{"admin", "meraki.example.com/api"},
		},
		{
			name:    "short known values are left to the patterns",
			in:      "link up on eth0",
			secrets: []string{"up"},
			kept:    []string{"link up on eth0"},
		},
		{
			name: "text without credentials is unchanged",
			in:   "database initialized path=data/seed.db",
			kept: []string{"database initialized path=data/seed.db"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checkText(t, support.ScrubText(tc.in, tc.secrets), tc.kept, tc.gone)
		})
	}
}
