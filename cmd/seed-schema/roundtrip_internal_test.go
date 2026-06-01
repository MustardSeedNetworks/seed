package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// TestCoveredDTOsRoundTripAgainstSchema is the Phase 2 round-trip guardrail
// (docs/architecture/RE_ARCHITECTURE_BLUEPRINT.md).
//
// The schema generator reflects the Go struct, which can diverge from the
// JSON the handler actually puts on the wire: a custom MarshalJSON, a
// computed field, an embedded struct, or an `omitzero`/`omitempty` quirk can
// all make the published schema "confidently wrong". For every DTO we publish
// a schema for, this test builds a fully populated sample, marshals it the way
// the handler would, and asserts the bytes validate against the committed
// schema file. A failure means the struct and the wire contract have drifted —
// catch it here before clients bake the wrong type in.
//
// It validates against the on-disk docs/schemas/api/*.json (what clients
// consume), so it also transitively guards the generator: the schema-drift CI
// gate keeps those files equal to the generator's output, and this test keeps
// the generator's output honest about the wire.
func TestCoveredDTOsRoundTripAgainstSchema(t *testing.T) {
	t.Parallel()

	schemaDir := schemaDirForTest(t)

	for _, target := range schemaTargets() {
		t.Run(target.title, func(t *testing.T) {
			t.Parallel()

			schema := compileSchemaFile(t, filepath.Join(schemaDir, target.filename))

			// target.value is a pointer to a zero-value DTO; reflect the
			// pointed-to struct type and fill every field so the sample
			// exercises required and optional members alike.
			sample := fillValue(t, reflect.TypeOf(target.value).Elem())

			data, err := json.Marshal(sample.Interface())
			if err != nil {
				t.Fatalf("marshal sample for %s: %v", target.title, err)
			}

			var doc any
			if unmarshalErr := json.Unmarshal(data, &doc); unmarshalErr != nil {
				t.Fatalf("re-parse sample JSON for %s: %v", target.title, unmarshalErr)
			}

			if validateErr := schema.Validate(doc); validateErr != nil {
				t.Fatalf(
					"%s sample does not validate against %s — struct and wire have drifted:\n%v\nsample JSON: %s",
					target.title, target.filename, validateErr, data,
				)
			}
		})
	}
}

// schemaDirForTest resolves docs/schemas/api relative to this source file, so
// the test is independent of the working directory `go test` happens to use.
func schemaDirForTest(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller: could not locate test source file")
	}

	// thisFile is <root>/cmd/seed-schema/roundtrip_internal_test.go.
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "docs", "schemas", "api")
}

// compileSchemaFile reads and compiles a committed schema document. It compiles
// against the document's own $id when present so internal "#/$defs/..."
// references resolve the same way a client's validator would resolve them.
func compileSchemaFile(t *testing.T, path string) *jsonschema.Schema {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema %s: %v", path, err)
	}

	var doc any
	if unmarshalErr := json.Unmarshal(raw, &doc); unmarshalErr != nil {
		t.Fatalf("parse schema %s: %v", path, unmarshalErr)
	}

	loc := filepath.Base(path)
	if obj, ok := doc.(map[string]any); ok {
		if id, hasID := obj["$id"].(string); hasID && id != "" {
			loc = id
		}
	}

	c := jsonschema.NewCompiler()
	if addErr := c.AddResource(loc, doc); addErr != nil {
		t.Fatalf("add schema resource %s: %v", path, addErr)
	}

	schema, err := c.Compile(loc)
	if err != nil {
		t.Fatalf("compile schema %s: %v", path, err)
	}

	return schema
}

// fillValue builds a fully populated value of typ via reflection: every
// pointer is allocated, every slice and map gets one element, every scalar a
// representative non-zero value. The point is that no field is left at its zero
// value, so the marshaled sample carries every property the schema declares —
// including the non-omitempty ones the schema marks `required`.
func fillValue(t *testing.T, typ reflect.Type) reflect.Value {
	t.Helper()

	// time.Time is a struct, but it must not be walked field-by-field: it
	// marshals to an RFC 3339 string via its own MarshalJSON, which is what
	// the schema's "format": "date-time" expects. A fixed instant keeps the
	// test independent of the wall clock.
	if typ == reflect.TypeFor[time.Time]() {
		return reflect.ValueOf(time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC))
	}

	// Every reflect.Kind is listed (no default) so a future field type the
	// filler can't represent fails the exhaustiveness check at compile-review
	// time rather than silently producing a bogus sample.
	switch typ.Kind() {
	case reflect.Pointer:
		ptr := reflect.New(typ.Elem())
		ptr.Elem().Set(fillValue(t, typ.Elem()))
		return ptr
	case reflect.Struct:
		return fillStruct(t, typ)
	case reflect.Slice:
		slice := reflect.MakeSlice(typ, 1, 1)
		slice.Index(0).Set(fillValue(t, typ.Elem()))
		return slice
	case reflect.Array:
		arr := reflect.New(typ).Elem()
		for i := range typ.Len() {
			arr.Index(i).Set(fillValue(t, typ.Elem()))
		}
		return arr
	case reflect.Map:
		m := reflect.MakeMap(typ)
		m.SetMapIndex(fillValue(t, typ.Key()), fillValue(t, typ.Elem()))
		return m
	case reflect.String:
		return reflect.ValueOf("sample").Convert(typ)
	case reflect.Bool:
		return reflect.ValueOf(true).Convert(typ)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.ValueOf(int64(1)).Convert(typ)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return reflect.ValueOf(uint64(1)).Convert(typ)
	case reflect.Float32, reflect.Float64:
		return reflect.ValueOf(1.5).Convert(typ)
	case reflect.Interface:
		// Bare interface fields (e.g. map[string]any values) reflect to an
		// open schema; any concrete JSON value satisfies it.
		return reflect.ValueOf("sample")
	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.UnsafePointer:
		t.Fatalf("fillValue: unsupported kind %s for type %s — extend the filler", typ.Kind(), typ)
	}

	return reflect.Value{}
}

// fillStruct populates every exported, JSON-serialized field of a struct,
// honoring the same `json:"-"` skip the encoder applies.
func fillStruct(t *testing.T, typ reflect.Type) reflect.Value {
	t.Helper()

	out := reflect.New(typ).Elem()
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue // unexported: invisible to encoding/json and the schema
		}
		if field.Tag.Get("json") == "-" {
			continue // explicitly excluded from the wire
		}
		out.Field(i).Set(fillValue(t, field.Type))
	}

	return out
}
