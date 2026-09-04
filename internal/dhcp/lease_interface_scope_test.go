package dhcp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/dhcp"
)

// sharedLeases is the shape of /var/lib/dhcp/dhclient.leases: one file, every
// interface on the host, newest block last.
const sharedLeases = `lease {
  interface "eth0";
  fixed-address 10.0.0.5;
  option dhcp-server-identifier 10.0.0.1;
  option routers 10.0.0.1;
  option domain-name-servers 10.0.0.53;
  option dhcp-lease-time 3600;
}
lease {
  interface "eth1";
  fixed-address 192.0.2.5;
  option dhcp-server-identifier 192.0.2.1;
  option routers 192.0.2.1;
  option domain-name-servers 192.0.2.53;
  option dhcp-lease-time 7200;
}
`

// TestDHClientLeaseIsScopedToItsInterface pins the defect the CI runner exposed:
// the shared lease file was parsed without regard to which interface each block
// belonged to, so the last block won. Asking about eth0 on a host that also has
// eth1 returned eth1's gateway, DHCP server and DNS — and /api/v1/ipconfig and
// the diagnostics export reported them as eth0's.
func TestDHClientLeaseIsScopedToItsInterface(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dhclient.leases")
	if err := os.WriteFile(path, []byte(sharedLeases), 0o600); err != nil {
		t.Fatalf("write lease file: %v", err)
	}

	for _, tc := range []struct {
		iface       string
		wantGateway string
		wantServer  string
		wantDNS     string
	}{
		{iface: "eth0", wantGateway: "10.0.0.1", wantServer: "10.0.0.1", wantDNS: "10.0.0.53"},
		{iface: "eth1", wantGateway: "192.0.2.1", wantServer: "192.0.2.1", wantDNS: "192.0.2.53"},
	} {
		t.Run(tc.iface, func(t *testing.T) {
			got := dhcp.ParseDHClientLeaseFile(path, tc.iface)
			if got == nil {
				t.Fatalf("ParseDHClientLeaseFile(%q) = nil, want the block for that interface", tc.iface)
			}
			if got.Gateway != tc.wantGateway {
				t.Errorf("Gateway = %q, want %q", got.Gateway, tc.wantGateway)
			}
			if got.DHCPServer != tc.wantServer {
				t.Errorf("DHCPServer = %q, want %q", got.DHCPServer, tc.wantServer)
			}
			if len(got.DNS) != 1 || got.DNS[0] != tc.wantDNS {
				t.Errorf("DNS = %v, want [%s]", got.DNS, tc.wantDNS)
			}
		})
	}

	// An interface with no block in the file has no lease, rather than
	// inheriting somebody else's. Every block here names an interface; an
	// unattributed block would still match, since the per-interface lease files
	// rely on their filename instead of the line.
	if got := dhcp.ParseDHClientLeaseFile(path, "eth9"); got != nil {
		t.Errorf("ParseDHClientLeaseFile(%q) = %+v, want nil", "eth9", got)
	}
}
