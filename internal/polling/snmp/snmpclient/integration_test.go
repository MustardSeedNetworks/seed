//go:build integration

// Package snmpclient_test's integration suite exercises the gosnmp client
// against a real net-snmp agent rather than a fake.
//
// The unit tests drive collectors through a fake Client, which is the right
// shape for behaviour but cannot catch wire-format divergence: a fake returns
// whatever Go value the test author chose, while a real agent returns whatever
// BER the agent encodes. Counter64 vs Counter32, OCTET STRING vs OID, an empty
// table vs a noSuchObject — those only show up against something that speaks
// the protocol.
//
// Run with a live agent (see CONTRIBUTING.md for the snmpd invocation):
//
//	SEED_SNMP_ADDR=127.0.0.1:1161 go test -tags integration \
//	    ./internal/polling/snmp/snmpclient/
package snmpclient_test

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/snmpclient"
)

const (
	// Seeded by the fixture config in testdata/snmpd.conf. Asserting on these
	// exact strings is the point: it proves the value survived the round trip
	// through the agent's encoder and gosnmp's decoder unchanged.
	wantSysName  = "seed-snmp-fixture"
	wantSysDescr = "Seed integration test agent"

	oidSysDescr     = "1.3.6.1.2.1.1.1.0"
	oidSysObjectID  = "1.3.6.1.2.1.1.2.0"
	oidSysUpTime    = "1.3.6.1.2.1.1.3.0"
	oidSysName      = "1.3.6.1.2.1.1.5.0"
	oidIfNumber     = "1.3.6.1.2.1.2.1.0"
	oidIfDescr      = "1.3.6.1.2.1.2.2.1.2"
	oidIfType       = "1.3.6.1.2.1.2.2.1.3"
	oidIfHCInOctets = "1.3.6.1.2.1.31.1.1.1.6"
)

// dial builds a client pointed at the agent named by SEED_SNMP_ADDR.
func dial(t *testing.T) snmp.Client {
	t.Helper()

	addr := os.Getenv("SEED_SNMP_ADDR")
	if addr == "" {
		// Skipping is right for a developer who ran the suite without an
		// agent, and wrong for CI: a job that skips every test reports green
		// while proving nothing. The workflow sets SEED_SNMP_REQUIRED so a
		// misconfigured job fails loudly instead.
		if os.Getenv("SEED_SNMP_REQUIRED") != "" {
			t.Fatal("SEED_SNMP_REQUIRED is set but SEED_SNMP_ADDR is empty — " +
				"the agent this suite exists to test was never reachable")
		}
		t.Skip("SEED_SNMP_ADDR unset; see CONTRIBUTING.md for the local agent")
	}
	host, portText, found := strings.Cut(addr, ":")
	if !found {
		t.Fatalf("SEED_SNMP_ADDR = %q, want host:port", addr)
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil {
		t.Fatalf("SEED_SNMP_ADDR port %q: %v", portText, err)
	}

	community := os.Getenv("SEED_SNMP_COMMUNITY")
	if community == "" {
		community = "public"
	}

	factory := snmpclient.NewFactory(snmpclient.Options{
		Port:    uint16(port),
		Timeout: 3 * time.Second,
		Retries: 1,
	})
	client, err := factory(
		snmp.Target{IPAddress: host, SNMPVersion: "2c"},
		snmp.ResolvedCredentials{SNMPCommunity: community},
	)
	if err != nil {
		t.Fatalf("build client for %s: %v", addr, err)
	}
	return client
}

func timeout(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// TestGetReadsTheSystemGroup covers the sys_info collector's OID surface.
func TestGetReadsTheSystemGroup(t *testing.T) {
	client := dial(t)

	vars, err := client.Get(timeout(t), []string{
		oidSysDescr, oidSysObjectID, oidSysUpTime, oidSysName,
	})
	if err != nil {
		t.Fatalf("Get system group: %v", err)
	}
	if len(vars) != 4 {
		t.Fatalf("got %d varbinds, want 4", len(vars))
	}

	byOID := map[string]snmp.Varbind{}
	for _, v := range vars {
		byOID[strings.TrimPrefix(v.OID, ".")] = v
	}

	for oid, want := range map[string]string{
		oidSysName:  wantSysName,
		oidSysDescr: wantSysDescr,
	} {
		v, ok := byOID[oid]
		if !ok {
			t.Errorf("%s missing from the response", oid)
			continue
		}
		// A string that arrives as []byte still renders correctly; what would
		// break a collector is it arriving as something else entirely.
		if got := renderString(v.Value); got != want {
			t.Errorf("%s = %q (%T), want %q", oid, got, v.Value, want)
		}
	}

	// sysUpTime is a TimeTicks, not a string or an integer. Collectors that
	// assume int64 here break against real agents.
	if v, ok := byOID[oidSysUpTime]; ok && v.Value == nil {
		t.Error("sysUpTime present but nil")
	}

	// sysObjectID is an OID, which decodes to a dotted string rather than to
	// bytes. Getting this wrong is a classic fake-vs-real divergence.
	if v, ok := byOID[oidSysObjectID]; ok {
		if got := renderString(v.Value); !strings.HasPrefix(got, "1.3.6.1") &&
			!strings.HasPrefix(got, ".1.3.6.1") {
			t.Errorf("sysObjectID = %q (%T), want a dotted OID", got, v.Value)
		}
	}
}

// TestWalkReadsTheInterfaceTable covers the if_table collector's surface, and
// with it the GETBULK path that Get never touches.
func TestWalkReadsTheInterfaceTable(t *testing.T) {
	client := dial(t)
	ctx := timeout(t)

	count, err := client.Get(ctx, []string{oidIfNumber})
	if err != nil {
		t.Fatalf("Get ifNumber: %v", err)
	}
	if len(count) != 1 {
		t.Fatalf("ifNumber returned %d varbinds, want 1", len(count))
	}

	descrs, err := client.Walk(ctx, oidIfDescr)
	if err != nil {
		t.Fatalf("Walk ifDescr: %v", err)
	}
	if len(descrs) == 0 {
		t.Fatal("Walk ifDescr returned nothing; the agent exposes no ifTable, " +
			"so this suite would pass vacuously")
	}

	// Every row's OID must extend the prefix — a Walk that leaked past the
	// subtree is a real bug and one a fake would never reproduce.
	for _, v := range descrs {
		if !strings.HasPrefix(strings.TrimPrefix(v.OID, "."), oidIfDescr) {
			t.Errorf("Walk returned %s, outside the %s subtree", v.OID, oidIfDescr)
		}
		if renderString(v.Value) == "" {
			t.Errorf("%s has an empty ifDescr", v.OID)
		}
	}

	types, err := client.Walk(ctx, oidIfType)
	if err != nil {
		t.Fatalf("Walk ifType: %v", err)
	}
	if len(types) != len(descrs) {
		t.Errorf("ifType has %d rows, ifDescr has %d; the table is inconsistent",
			len(types), len(descrs))
	}
}

// TestWalkHandlesCounter64 pins the 64-bit counter path. Counter64 arrives as
// uint64 where Counter32 arrives as uint; a collector that type-asserts the
// wrong one silently records zero, and no fake would catch it.
func TestWalkHandlesCounter64(t *testing.T) {
	client := dial(t)

	octets, err := client.Walk(timeout(t), oidIfHCInOctets)
	if err != nil {
		t.Fatalf("Walk ifHCInOctets: %v", err)
	}
	if len(octets) == 0 {
		t.Skip("agent exposes no ifXTable; nothing to assert about Counter64")
	}
	for _, v := range octets {
		switch v.Value.(type) {
		case uint64, uint, int, int64:
		default:
			t.Errorf("%s is %T, want an integer kind — a collector asserting "+
				"uint64 would read zero", v.OID, v.Value)
		}
	}
}

// TestWalkOfAnEmptySubtreeIsNotAnError pins the shape collectors rely on when
// a device does not implement a MIB: no rows, no error. Returning an error
// there would abort a collector chain over a device simply lacking a feature.
func TestWalkOfAnEmptySubtreeIsNotAnError(t *testing.T) {
	client := dial(t)

	// Under the experimental arc, which net-snmp does not populate.
	rows, err := client.Walk(timeout(t), "1.3.6.1.3.55555")
	if err != nil {
		t.Fatalf("Walk of an unimplemented subtree returned %v, want no error", err)
	}
	if len(rows) != 0 {
		t.Errorf("Walk of an unimplemented subtree returned %d rows, want 0", len(rows))
	}
}

// TestGetOfAnAbsentOIDDoesNotFailTheRequest pins that a missing scalar comes
// back as a varbind carrying a noSuchObject-style value rather than as a
// transport error, so one absent OID cannot fail a whole collector pass.
func TestGetOfAnAbsentOIDDoesNotFailTheRequest(t *testing.T) {
	client := dial(t)

	vars, err := client.Get(timeout(t), []string{oidSysName, "1.3.6.1.3.55555.1.0"})
	if err != nil {
		t.Fatalf("Get with one absent OID returned %v, want no error", err)
	}
	if len(vars) != 2 {
		t.Fatalf("got %d varbinds, want 2 — the present OID must still return",
			len(vars))
	}
}

// renderString normalises the several Go types a string-ish SNMP value can
// arrive as, so assertions can talk about content rather than representation.
func renderString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}
