package app_test

// The MIB store's lookups, run against the migrated schema. They live here
// because this package is the one place that opens the store (mibdb.go).

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/mibdb"
)

func builtinMIB(t *testing.T) *mibdb.DB {
	t.Helper()
	mib := app.NewMIBDatabase(app.NewPersistence(dbtest.Open(t)))
	if err := mib.LoadBuiltinOIDs(); err != nil {
		t.Fatalf("load builtin OIDs: %v", err)
	}
	return mib
}

func TestLookupNumeric(t *testing.T) {
	t.Parallel()
	mib := builtinMIB(t)
	tests := []struct {
		oid      string
		name     string
		instance string
	}{
		{oid: "1.3.6.1.2.1.1.1.0", name: "sysDescr", instance: "0"},
		{oid: ".1.3.6.1.2.1.2.2.1.2.3", name: "ifDescr", instance: "3"},
		{oid: "1.3.6.1.2.1.2.2.1.2", name: "ifDescr", instance: ""},
		// ifDescr is 1.3.6.1.2.1.2.2.1.2; its arc must not swallow ifType's 20.
		{oid: "1.3.6.1.2.1.2.2.1.20.1", name: "ifOutErrors", instance: "1"},
		{oid: "1.3.6.1.2.1.4.20.1.1.10.0.0.1", name: "ipAdEntAddr", instance: "10.0.0.1"},
		// A column the table does not list still names its row: ifEntry.
		{oid: "1.3.6.1.2.1.2.2.1.99.7", name: "ifEntry", instance: "99.7"},
		{oid: "1.3.6.1.2.1.2.2", name: "ifTable", instance: ""},
	}
	for _, tt := range tests {
		t.Run(tt.oid, func(t *testing.T) {
			t.Parallel()
			entry, instance, err := mib.LookupNumeric(t.Context(), tt.oid)
			if err != nil {
				t.Fatalf("LookupNumeric(%q): %v", tt.oid, err)
			}
			if entry.Name != tt.name || instance != tt.instance {
				t.Errorf("LookupNumeric(%q) = %s, %q; want %s, %q", tt.oid, entry.Name, instance, tt.name, tt.instance)
			}
		})
	}
}

func TestLookupNumericUnknown(t *testing.T) {
	t.Parallel()
	mib := builtinMIB(t)
	if _, _, err := mib.LookupNumeric(t.Context(), "2.999.1"); !errors.Is(err, mibdb.ErrNotFound) {
		t.Errorf("LookupNumeric(2.999.1) error = %v, want ErrNotFound", err)
	}
}

// A subtree can be named by the branch its full paths pass through, not only
// by a leaf object.
func TestResolveOIDNameBranches(t *testing.T) {
	t.Parallel()
	mib := builtinMIB(t)
	tests := map[string]string{
		"system":      "1.3.6.1.2.1.1",
		"mib-2":       "1.3.6.1.2.1",
		"ifTable":     "1.3.6.1.2.1.2.2",
		"ifEntry.2":   "1.3.6.1.2.1.2.2.1.2",
		"dot1dBridge": "1.3.6.1.2.1.17",
		"sysDescr.0":  "1.3.6.1.2.1.1.1.0",
	}
	for name, want := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := mib.ResolveOIDName(name)
			if err != nil || got != want {
				t.Errorf("ResolveOIDName(%q) = %q, %v; want %q", name, got, err, want)
			}
		})
	}
}
