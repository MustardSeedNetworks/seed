//go:build integration

package snmp_test

import (
	"context"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// TestV3CredentialsAreSentAtTheirOwnSecurityLevel runs against the v3 users
// in internal/polling/snmp/snmpclient/testdata/snmpd.conf, one per security
// level. Every v3 exchange used to be sent at authPriv, so an authNoPriv or
// noAuthNoPriv credential failed every read (seed#3053). Each read path that
// builds its own v3 client is exercised: a single GET, a multi-OID GET and
// both walks.
func TestV3CredentialsAreSentAtTheirOwnSecurityLevel(t *testing.T) {
	host, port := fixtureAgent(t)

	creds := []snmp.V3Credential{
		{Name: "noAuthNoPriv", Username: "seednoauth", SecurityLevel: "noAuthNoPriv"},
		{
			Name: "authNoPriv", Username: "seedauth", SecurityLevel: "authNoPriv",
			AuthProtocol: "SHA256", AuthPassword: "seed-auth-fixture",
		},
		{
			Name: "authPriv", Username: "seedpriv", SecurityLevel: "authPriv",
			AuthProtocol: "SHA256", AuthPassword: "seed-auth-fixture",
			PrivProtocol: "AES", PrivPassword: "seed-priv-fixture",
		},
	}
	for _, cred := range creds {
		t.Run(cred.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			session := &snmp.Session{
				Port:          port,
				Timeout:       3 * time.Second,
				Retries:       1,
				V3Credentials: []snmp.V3Credential{cred},
			}

			name, err := snmp.Query(ctx, host, snmp.OIDSysName, session)
			if err != nil || name != "seed-snmp-fixture" {
				t.Errorf("Query sysName = %q, %v", name, err)
			}
			info, _, err := snmp.GetSystemInfo(ctx, host, session)
			switch {
			case err != nil:
				t.Errorf("GetSystemInfo: %v", err)
			case info.SysName != "seed-snmp-fixture":
				t.Errorf("GetSystemInfo sysName = %q", info.SysName)
			}
			ifaces, err := snmp.GetAllInterfaces(ctx, host, session)
			if err != nil || len(ifaces) == 0 {
				t.Errorf("GetAllInterfaces read %d interfaces: %v", len(ifaces), err)
			}
			addrs, err := snmp.GetIPAddresses(ctx, host, session)
			if err != nil || len(addrs) == 0 {
				t.Errorf("GetIPAddresses read %d addresses: %v", len(addrs), err)
			}
		})
	}
}

// fixtureAgent returns the net-snmp fixture's address from SEED_SNMP_ADDR,
// skipping the test when it is unset unless SEED_SNMP_REQUIRED is.
func fixtureAgent(t *testing.T) (string, int) {
	t.Helper()
	addr := os.Getenv("SEED_SNMP_ADDR")
	if addr == "" {
		if os.Getenv("SEED_SNMP_REQUIRED") != "" {
			t.Fatal("SEED_SNMP_REQUIRED is set but SEED_SNMP_ADDR is empty")
		}
		t.Skip("SEED_SNMP_ADDR unset; see CONTRIBUTING.md for the local agent")
	}
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("SEED_SNMP_ADDR = %q: %v", addr, err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("SEED_SNMP_ADDR port %q: %v", portText, err)
	}
	return host, port
}
