package mibdb

import (
	"slices"
	"testing"
)

func TestBranchOIDs(t *testing.T) {
	t.Parallel()
	entries := []OIDEntry{
		{Name: "ifDescr", OID: "1.3.2.2", FullPath: "iso(1).org(3).ifTable(2).ifDescr(2)"},
		// stack names 1.3.4 here and 1.3.5 below: ambiguous, so left out.
		{Name: "a", OID: "1.3.4.1", FullPath: "iso(1).org(3).stack(4).a(1)"},
		{Name: "b", OID: "1.3.5.1", FullPath: "iso(1).org(3).stack(5).b(1)"},
		// A branch named like a leaf is left out; the leaf keeps the name.
		{Name: "c", OID: "1.3.6.1", FullPath: "iso(1).org(3).ifDescr(6).c(1)"},
		// The path disagrees with the OID at bad(8), so only iso and org count.
		{Name: "d", OID: "1.3.7.1", FullPath: "iso(1).org(3).bad(8).d(1)"},
	}
	var got []string
	for _, b := range branchOIDs(entries) {
		got = append(got, b.Name+"="+b.OID)
	}
	slices.Sort(got)
	want := []string{"ifTable=1.3.2", "iso=1", "org=1.3"}
	if !slices.Equal(got, want) {
		t.Errorf("branchOIDs = %v, want %v", got, want)
	}
}
