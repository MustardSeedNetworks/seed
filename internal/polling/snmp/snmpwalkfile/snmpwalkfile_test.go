package snmpwalkfile_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/snmpwalkfile"
)

const (
	oidSysDescr = "1.3.6.1.2.1.1.1.0"
	oidSysName  = "1.3.6.1.2.1.1.5.0"
	oidIfDescr  = "1.3.6.1.2.1.2.2.1.2"
	oidIfType   = "1.3.6.1.2.1.2.2.1.3"
)

// fixtures returns every recorded walk, so a new vendor added to testdata is
// covered without editing this file.
func fixtures(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("testdata", "*.walk"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no walk fixtures found; these tests would assert nothing")
	}
	return paths
}

func TestEveryFixtureParses(t *testing.T) {
	for _, path := range fixtures(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			walk, err := snmpwalkfile.Open(path)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if walk.Len() == 0 {
				t.Fatal("parsed zero varbinds")
			}
			// A handful of unparseable lines is expected in real captures —
			// continuation lines from multi-line strings, mostly. A large
			// number means the parser is not reading the format.
			if walk.Unparsed() > walk.Len()/10 {
				t.Errorf("%d of %d lines unparsed; the parser is not reading this format",
					walk.Unparsed(), walk.Len()+walk.Unparsed())
			}
		})
	}
}

// TestEveryFixtureAnswersTheSystemGroup pins that the fixtures carry the data
// the sysinfo collector needs. A fixture set that parses but holds nothing
// useful would let every replay test pass while asserting nothing.
func TestEveryFixtureAnswersTheSystemGroup(t *testing.T) {
	for _, path := range fixtures(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			walk, err := snmpwalkfile.Open(path)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			vars, err := walk.Client().Get(context.Background(),
				[]string{oidSysDescr, oidSysName})
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			for _, vb := range vars {
				text, ok := vb.Value.(string)
				if !ok || text == "" {
					t.Errorf("%s = %v (%T), want a non-empty string",
						vb.OID, vb.Value, vb.Value)
				}
			}
		})
	}
}

// TestWalkReturnsWholeTables covers the GETBULK-equivalent path: every fixture
// should yield an interface table whose columns agree on row count.
func TestWalkReturnsWholeTables(t *testing.T) {
	for _, path := range fixtures(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			walk, err := snmpwalkfile.Open(path)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			client := walk.Client()

			descrs, err := client.Walk(context.Background(), oidIfDescr)
			if err != nil {
				t.Fatalf("Walk ifDescr: %v", err)
			}
			if len(descrs) == 0 {
				t.Skip("fixture carries no ifTable")
			}
			assertAllInSubtree(t, descrs, oidIfDescr)

			types, typesErr := client.Walk(context.Background(), oidIfType)
			if typesErr != nil {
				t.Fatalf("Walk ifType: %v", typesErr)
			}
			if len(types) != len(descrs) {
				t.Errorf("ifType has %d rows, ifDescr has %d — the table is inconsistent",
					len(types), len(descrs))
			}
		})
	}
}

// assertAllInSubtree fails if any varbind escaped the requested prefix.
func assertAllInSubtree(t *testing.T, vars []snmp.Varbind, prefix string) {
	t.Helper()
	for _, vb := range vars {
		if !strings.HasPrefix(vb.OID, prefix+".") {
			t.Errorf("Walk returned %s, outside the %s subtree", vb.OID, prefix)
		}
	}
}

// TestWalkOrdersNumerically pins that rows come back in OID order rather than
// lexical order, which would put index 10 before index 2 and hide the ordering
// bugs these fixtures exist to catch.
func TestWalkOrdersNumerically(t *testing.T) {
	walk, err := snmpwalkfile.Parse(strings.NewReader(`
.1.3.6.1.2.1.2.2.1.2.10 = STRING: ten
.1.3.6.1.2.1.2.2.1.2.2 = STRING: two
.1.3.6.1.2.1.2.2.1.2.1 = STRING: one
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rows, err := walk.Client().Walk(context.Background(), oidIfDescr)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	want := []string{"one", "two", "ten"}
	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}
	for i, vb := range rows {
		if vb.Value != want[i] {
			t.Errorf("row %d = %v, want %q — rows are not in numeric OID order",
				i, vb.Value, want[i])
		}
	}
}

// TestTypesMatchGosnmp pins the decoded Go types. A fixture that hands a
// collector a string where a live agent sends a Gauge32 lets the collector pass
// here and fail on a type assertion against real gear — which is the exact
// class of bug these fixtures exist to catch.
func TestTypesMatchGosnmp(t *testing.T) {
	walk, err := snmpwalkfile.Parse(strings.NewReader(`
.1.3.6.1.2.1.1.1.0 = STRING: Test Switch
.1.3.6.1.2.1.1.2.0 = OID: .1.3.6.1.4.1.9.1.1719
.1.3.6.1.2.1.1.3.0 = Timeticks: (123456789) 14 days, 6:56:07.89
.1.3.6.1.2.1.2.2.1.1.1 = INTEGER: 1
.1.3.6.1.2.1.2.2.1.5.1 = Gauge32: 1000000000
.1.3.6.1.2.1.2.2.1.10.1 = Counter32: 42
.1.3.6.1.2.1.31.1.1.1.6.1 = Counter64: 9999999999
.1.3.6.1.2.1.2.2.1.6.1 = Hex-STRING: 00 1A 2B 3C 4D 5E
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	client := walk.Client()

	for _, tc := range []struct {
		oid  string
		want any
	}{
		{"1.3.6.1.2.1.1.1.0", "Test Switch"},
		{"1.3.6.1.2.1.1.2.0", "1.3.6.1.4.1.9.1.1719"},
		{"1.3.6.1.2.1.1.3.0", uint(123456789)},
		{"1.3.6.1.2.1.2.2.1.1.1", 1},
		{"1.3.6.1.2.1.2.2.1.5.1", uint(1000000000)},
		{"1.3.6.1.2.1.2.2.1.10.1", uint(42)},
		{"1.3.6.1.2.1.31.1.1.1.6.1", uint64(9999999999)},
	} {
		vars, getErr := client.Get(context.Background(), []string{tc.oid})
		if getErr != nil {
			t.Fatalf("Get %s: %v", tc.oid, getErr)
		}
		if vars[0].Value != tc.want {
			t.Errorf("%s = %v (%T), want %v (%T)",
				tc.oid, vars[0].Value, vars[0].Value, tc.want, tc.want)
		}
	}

	mac, macErr := client.Get(context.Background(), []string{"1.3.6.1.2.1.2.2.1.6.1"})
	if macErr != nil {
		t.Fatalf("Get physAddress: %v", macErr)
	}
	got, ok := mac[0].Value.([]byte)
	if !ok {
		t.Fatalf("physAddress is %T, want []byte", mac[0].Value)
	}
	if want := []byte{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E}; string(got) != string(want) {
		t.Errorf("physAddress = % X, want % X", got, want)
	}
}

// TestAbsentOIDYieldsNilValue pins the contract Client.Get documents: a missing
// OID is a varbind with a nil Value, never an error, so one absent scalar
// cannot fail a whole collector pass.
func TestAbsentOIDYieldsNilValue(t *testing.T) {
	walk, err := snmpwalkfile.Open(fixtures(t)[0])
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	vars, err := walk.Client().Get(context.Background(),
		[]string{oidSysName, "1.3.6.1.3.55555.1.0"})
	if err != nil {
		t.Fatalf("Get with an absent OID returned %v, want no error", err)
	}
	if len(vars) != 2 {
		t.Fatalf("got %d varbinds, want 2", len(vars))
	}
	if vars[1].Value != nil {
		t.Errorf("absent OID has value %v, want nil", vars[1].Value)
	}
}

// TestWalkOfUnimplementedSubtreeIsEmpty pins that a device merely lacking a MIB
// yields no rows and no error, rather than aborting a collector chain.
func TestWalkOfUnimplementedSubtreeIsEmpty(t *testing.T) {
	walk, err := snmpwalkfile.Open(fixtures(t)[0])
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	rows, err := walk.Client().Walk(context.Background(), "1.3.6.1.3.55555")
	if err != nil {
		t.Fatalf("Walk of an unimplemented subtree: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("got %d rows, want 0", len(rows))
	}
}

func TestCancelledContextIsHonoured(t *testing.T) {
	walk, err := snmpwalkfile.Open(fixtures(t)[0])
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, getErr := walk.Client().Get(ctx, []string{oidSysName}); getErr == nil {
		t.Error("Get on a cancelled context returned no error")
	}
	if _, walkErr := walk.Client().Walk(ctx, oidIfDescr); walkErr == nil {
		t.Error("Walk on a cancelled context returned no error")
	}
}

func TestOpenRejectsAMissingFile(t *testing.T) {
	if _, err := snmpwalkfile.Open(filepath.Join(t.TempDir(), "no-such.walk")); err == nil {
		t.Error("Open of a missing file returned no error")
	}
}
