package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestPublishedSchemasAreAcyclic enforces the locked non-recursive transport
// policy (RE_ARCHITECTURE_BLUEPRINT.md Phase 2): a published schema's $defs
// must form a directed ACYCLIC graph. Accidental recursion — e.g. a
// GatewayResponse whose `ipv6` field reuses GatewayResponse — produces a $ref
// cycle, which clients' code generators handle inconsistently and which signals
// an internal shape leaking onto the wire instead of a flat transport DTO.
//
// This complements the round-trip guardrail: round-trip proves the wire bytes
// match the schema; this proves the schema's structure stays flat. A recursive
// but otherwise valid schema would pass round-trip and only fail here.
func TestPublishedSchemasAreAcyclic(t *testing.T) {
	t.Parallel()

	dir := schemaDirForTest(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read schema dir %s: %v", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".schema.json") {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			t.Parallel()

			defs := schemaDefs(t, filepath.Join(dir, entry.Name()))
			if cycle := findRefCycle(defs); cycle != "" {
				t.Fatalf(
					"%s has a $ref cycle among its $defs (%s) — transport DTOs must be non-recursive; "+
						"give the recursive field a flat sub-type",
					entry.Name(), cycle,
				)
			}
		})
	}
}

// schemaDefs returns the $defs object of a schema file as a name->node map.
func schemaDefs(t *testing.T, path string) map[string]any {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema %s: %v", path, err)
	}

	var doc map[string]any
	if unmarshalErr := json.Unmarshal(raw, &doc); unmarshalErr != nil {
		t.Fatalf("parse schema %s: %v", path, unmarshalErr)
	}

	defs, _ := doc["$defs"].(map[string]any)
	return defs
}

// findRefCycle returns a "A -> B -> A" description of the first $ref cycle
// reachable in the $defs graph, or "" if the graph is acyclic.
func findRefCycle(defs map[string]any) string {
	// Stable iteration order so the reported cycle is deterministic.
	names := make([]string, 0, len(defs))
	for name := range defs {
		names = append(names, name)
	}
	sort.Strings(names)

	const (
		unvisited = 0
		onStack   = 1
		done      = 2
	)
	state := make(map[string]int, len(defs))

	var visit func(name string, path []string) string
	visit = func(name string, path []string) string {
		state[name] = onStack
		path = append(path, name)
		for _, ref := range localRefs(defs[name]) {
			if _, ok := defs[ref]; !ok {
				continue
			}
			switch state[ref] {
			case onStack:
				return strings.Join(append(path, ref), " -> ")
			case unvisited:
				if cycle := visit(ref, path); cycle != "" {
					return cycle
				}
			}
		}
		state[name] = done
		return ""
	}

	for _, name := range names {
		if state[name] == unvisited {
			if cycle := visit(name, nil); cycle != "" {
				return cycle
			}
		}
	}
	return ""
}

// localRefs collects every "#/$defs/<name>" target referenced anywhere within a
// schema node (recursively through nested objects and arrays).
func localRefs(node any) []string {
	var out []string
	switch typed := node.(type) {
	case map[string]any:
		for key, val := range typed {
			if key == "$ref" {
				if ref, ok := val.(string); ok {
					if name, found := strings.CutPrefix(ref, "#/$defs/"); found {
						out = append(out, name)
					}
				}
				continue
			}
			out = append(out, localRefs(val)...)
		}
	case []any:
		for _, val := range typed {
			out = append(out, localRefs(val)...)
		}
	}
	return out
}
