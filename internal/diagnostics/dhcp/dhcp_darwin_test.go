//go:build darwin

package dhcp_test

import (
	"slices"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dhcp"
)

// The table dhcp_lease writes for en0 on a Mac holding a lease from
// 10.44.20.1, with the store's Option_N bytes as configd keeps them.
const storeTable = "address\t10.44.20.165\n" +
	"start\t1791782477\n" +
	"expiry\t1791868877\n" +
	"option\t1\tffffff00\n" +
	"option\t15\t6d736e2e6c6162\n" +
	"option\t3\t0a2c1401\n" +
	"option\t51\t00015180\n" +
	"option\t53\t05\n" +
	"option\t54\t0a2c1401\n" +
	"option\t6\t0a2c280a0a2c280b\n"

func TestStoreLeaseReportsTheLease(t *testing.T) {
	result, start, expiry, ok := dhcp.ExportStoreLease(storeTable)
	if !ok {
		t.Fatal("a table with a lease reported none")
	}
	want := dhcp.TestResult{
		OfferedIP:    "10.44.20.165",
		ServerIP:     "10.44.20.1",
		SubnetMask:   "255.255.255.0",
		Gateway:      "10.44.20.1",
		DNSServers:   []string{"10.44.40.10", "10.44.40.11"},
		DomainName:   "msn.lab",
		LeaseTime:    24 * time.Hour,
		LeaseTimeSec: 86400,
	}
	if result.OfferedIP != want.OfferedIP || result.ServerIP != want.ServerIP ||
		result.SubnetMask != want.SubnetMask || result.Gateway != want.Gateway ||
		!slices.Equal(result.DNSServers, want.DNSServers) || result.DomainName != want.DomainName ||
		result.LeaseTime != want.LeaseTime || result.LeaseTimeSec != want.LeaseTimeSec {
		t.Errorf("lease = %+v, want %+v", *result, want)
	}
	if !start.Equal(time.Unix(1791782477, 0)) || !expiry.Equal(time.Unix(1791868877, 0)) {
		t.Errorf("start, expiry = %v, %v", start, expiry)
	}
}

// The server is the server identifier, never the BOOTP siaddr field, which
// names a boot server and reads 0.0.0.0 on most networks.
func TestStoreLeaseServerIsTheServerIdentifier(t *testing.T) {
	result, _, _, _ := dhcp.ExportStoreLease("address\t192.0.2.10\noption\t54\tc0000201\n")
	if result.ServerIP != "192.0.2.1" {
		t.Errorf("ServerIP = %q, want 192.0.2.1", result.ServerIP)
	}
}

func TestStoreLeaseEmptyTableIsNoLease(t *testing.T) {
	if _, _, _, ok := dhcp.ExportStoreLease(""); ok {
		t.Error("an empty table reported a lease")
	}
}

// A router listing several addresses names the first; a server may NUL-pad
// the domain name; malformed or short option bytes are not decoded.
func TestStoreLeaseDecodesOptionsDefensively(t *testing.T) {
	tests := []struct {
		name  string
		table string
		check func(*dhcp.TestResult) bool
	}{
		{
			name:  "first of several routers",
			table: "option\t3\tc0000201c0000202\n",
			check: func(r *dhcp.TestResult) bool { return r.Gateway == "192.0.2.1" },
		},
		{
			name:  "NUL-padded domain",
			table: "option\t15\t6c616200\n",
			check: func(r *dhcp.TestResult) bool { return r.DomainName == "lab" },
		},
		{
			name:  "short router",
			table: "option\t3\tc00002\n",
			check: func(r *dhcp.TestResult) bool { return r.Gateway == "" },
		},
		{
			name:  "bad hex",
			table: "option\t1\tzz\n",
			check: func(r *dhcp.TestResult) bool { return r.SubnetMask == "" },
		},
		{
			name:  "short lease time",
			table: "option\t51\t0151\n",
			check: func(r *dhcp.TestResult) bool { return r.LeaseTime == 0 },
		},
		{
			name:  "bad option code",
			table: "option\tx\tffffff00\n",
			check: func(r *dhcp.TestResult) bool { return r.SubnetMask == "" },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, _, ok := dhcp.ExportStoreLease(tt.table)
			if !ok || !tt.check(result) {
				t.Errorf("ExportStoreLease(%q) = %+v", tt.table, result)
			}
		})
	}
}
