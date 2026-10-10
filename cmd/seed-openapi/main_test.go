package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/seed/internal/api"
)

// generated renders the committed source against the live registry, which is
// what `make openapi` writes.
func generated(t *testing.T) map[string]map[string]any {
	t.Helper()
	src, err := os.ReadFile("../../docs/openapi-source.yaml")
	if err != nil {
		t.Fatalf("reading source: %v", err)
	}
	out, err := generate(src, api.RouteManifest())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	var doc struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if unmarshalErr := yaml.Unmarshal(out, &doc); unmarshalErr != nil {
		t.Fatalf("generated document is not valid YAML: %v", unmarshalErr)
	}
	return doc.Paths
}

// TestEveryRegistryRouteIsDocumented: no registered route may be absent, and
// each carries exactly the methods the registry gates. A route that declares
// no methods dispatches for itself and is documented with the full set.
func TestEveryRegistryRouteIsDocumented(t *testing.T) {
	t.Parallel()
	items := byRegistryPath(generated(t))

	for _, rt := range api.RouteManifest() {
		if rt.Hidden {
			continue // the embedded UI's catch-alls
		}
		item, ok := items[rt.Path]
		if !ok {
			t.Errorf("registered route %s is missing from the generated document", rt.Path)
			continue
		}
		methods := rt.Methods
		if len(methods) == 0 {
			methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
		}
		for _, m := range methods {
			if _, documented := item[strings.ToLower(m)]; !documented {
				t.Errorf("%s: method %s is registered but not documented", rt.Path, m)
			}
		}
		documented := len(item)
		if _, hasParams := item["parameters"]; hasParams {
			documented--
		}
		if documented != len(methods) {
			t.Errorf("%s documents %d methods, the registry gates %v", rt.Path, documented, methods)
		}
	}
}

// byRegistryPath keys each documented path item by the registry path it came
// from: a templated prefix route /x/{id} back to /x/.
func byRegistryPath(paths map[string]map[string]any) map[string]map[string]any {
	items := make(map[string]map[string]any, len(paths))
	for docPath, item := range paths {
		key := docPath
		if i := strings.Index(docPath, "/{"); i >= 0 {
			key = docPath[:i+1]
		}
		items[key] = item
	}
	return items
}

// TestPolicyIsDocumented checks the half a hand-written spec cannot keep
// true: a CSRF-protected, operator-scoped write names the CSRF scheme and its
// 403, its read does not, and a pre-session route asks for no credential.
func TestPolicyIsDocumented(t *testing.T) {
	t.Parallel()
	paths := generated(t)
	op := func(path, method string) map[string]any {
		t.Helper()
		o, ok := paths[path][method].(map[string]any)
		if !ok {
			t.Fatalf("%s %s is not documented", method, path)
		}
		return o
	}

	write := op("/api/v1/settings", "put")
	if _, has403 := write["responses"].(map[string]any)["403"]; !has403 {
		t.Error("PUT /api/v1/settings documents no scope/CSRF 403")
	}
	security, _ := write["security"].([]any)
	if len(security) != 1 || security[0].(map[string]any)["CsrfToken"] == nil {
		t.Errorf("PUT /api/v1/settings security = %v, want BearerAuth + CsrfToken", write["security"])
	}
	if !strings.Contains(write["description"].(string), "operator-scoped") {
		t.Errorf("PUT /api/v1/settings description = %q, want the operator scope", write["description"])
	}

	read := op("/api/v1/settings", "get")
	if _, has403 := read["responses"].(map[string]any)["403"]; has403 {
		t.Error("GET /api/v1/settings documents a 403, but safe methods pass the scope and CSRF gates")
	}

	login := op("/api/v1/auth/login", "post")
	if loginSecurity, _ := login["security"].([]any); loginSecurity == nil || len(loginSecurity) != 0 {
		t.Errorf("POST /api/v1/auth/login security = %v, want [] (pre-session)", login["security"])
	}
}

// TestRunWritesTheCommittedDocument: what `make openapi` writes is what is
// committed, so a route change that skips `make openapi` fails `go test` as
// well as the CI drift step.
func TestRunWritesTheCommittedDocument(t *testing.T) {
	t.Parallel()
	out := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := run("../../docs/openapi-source.yaml", out); err != nil {
		t.Fatalf("run: %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	committed, err := os.ReadFile("../../docs/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(committed) {
		t.Error("docs/openapi.yaml is stale; run `make openapi`")
	}
}
