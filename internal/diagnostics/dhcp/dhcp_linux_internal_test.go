//go:build linux

package dhcp

import (
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

// writeLeaseFile writes a minimal dhclient lease that parseLeaseFile accepts,
// and returns its path.
func writeLeaseFile(t *testing.T, path string) string {
	t.Helper()

	lease := "lease {\n" +
		"  fixed-address 10.9.9.9;\n" +
		"  option subnet-mask 255.255.255.0;\n" +
		"  option routers 10.9.9.1;\n" +
		"}\n"

	if err := os.WriteFile(path, []byte(lease), 0o600); err != nil {
		t.Fatalf("write lease file %s: %v", path, err)
	}

	return path
}

// networkdLease is a lease as systemd 259 writes it on Ubuntu 26.04, copied
// from dev-srv-ubuntu's ens19 with the client identifier dropped.
const networkdLease = `# This is private data. Do not parse.
ADDRESS=10.44.20.142
NETMASK=255.255.255.0
ROUTER=10.44.20.1
SERVER_ADDRESS=10.44.20.1
NEXT_SERVER=10.44.20.1
T1=12h
T2=21h
LIFETIME=1d
DNS=10.44.40.10 10.44.40.11
DOMAINNAME=msn.lab
HOSTNAME=dev-srv-ubuntu
`

// hostLoopback returns the host's loopback interface. Enumeration needs no
// privileges, so a failure here is a real problem rather than a reason to skip.
func hostLoopback(t *testing.T) net.Interface {
	t.Helper()

	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("cannot enumerate interfaces: %v", err)
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			return iface
		}
	}

	t.Fatal("no loopback interface found; every host this suite runs on has one")

	return net.Interface{}
}

// TestFindAndParseLeaseFile_RejectsTraversal is the reason the interface name is
// resolved against the kernel's list before it is interpolated into a path.
//
// The layout mirrors the real one: the templates put the interface name inside a
// filename under a search directory. A name carrying a traversal segment, with a
// directory that completes the prefix, escapes that directory — here reaching a
// file the caller was never meant to read. The test asserts the escape does not
// happen, and that the refusal names the interface rather than the path.
func TestFindAndParseLeaseFile_RejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	templates := []string{filepath.Join(dir, "dhclient.%s.leases")}

	// The file the traversal aims at, outside the intended filename shape.
	secret := writeLeaseFile(t, filepath.Join(dir, "secret.leases"))

	// The directory that completes the "dhclient." prefix, so the traversal
	// resolves rather than dying on a missing path segment.
	if err := os.Mkdir(filepath.Join(dir, "dhclient.x"), 0o700); err != nil {
		t.Fatalf("mkdir prefix directory: %v", err)
	}

	traversal := "x/../secret"

	// Confirm the test's own premise: without the guard this path resolves to
	// the secret file. If this stops holding, the test is no longer proving
	// anything and must be fixed rather than trusted.
	resolved := strings.ReplaceAll(templates[0], "%s", traversal)
	if _, err := os.Stat(resolved); err != nil {
		t.Fatalf("premise broken: %s should resolve to %s, got %v", resolved, secret, err)
	}

	lease, err := findLeaseFileIn(dir, templates, traversal)
	if err == nil {
		t.Fatalf("findLeaseFileIn(%q) read %+v, want a refusal", traversal, lease)
	}
	if lease != nil {
		t.Errorf("lease = %+v, want nil alongside the error", lease)
	}
	if !strings.Contains(err.Error(), "interface not found") {
		t.Errorf("error = %v, want it to name the interface as the problem", err)
	}
}

// TestFindAndParseLeaseFile_ReadsRealInterfaceLease is the positive half: a name
// the host really has still finds and parses its lease, so the guard has not
// simply broken the feature.
func TestFindAndParseLeaseFile_ReadsRealInterfaceLease(t *testing.T) {
	dir := t.TempDir()
	templates := []string{filepath.Join(dir, "dhclient.%s.leases")}

	name := hostLoopback(t).Name
	writeLeaseFile(t, filepath.Join(dir, "dhclient."+name+".leases"))

	lease, err := findLeaseFileIn(t.TempDir(), templates, name)
	if err != nil {
		t.Fatalf("findLeaseFileIn(%q): %v", name, err)
	}

	if lease.IPAddress != "10.9.9.9" {
		t.Errorf("IPAddress = %q, want 10.9.9.9", lease.IPAddress)
	}
	if lease.Interface != name {
		t.Errorf("Interface = %q, want %q", lease.Interface, name)
	}
}

// TestFindLeaseFileIn_ReadsNetworkdLease is #3530: on a networkd host the lease
// lives under the interface's index, not its name, and every field the gateway
// and AutoTest stages read comes out of it.
func TestFindLeaseFileIn_ReadsNetworkdLease(t *testing.T) {
	networkdDir := t.TempDir()
	loopback := hostLoopback(t)

	path := filepath.Join(networkdDir, strconv.Itoa(loopback.Index))
	if err := os.WriteFile(path, []byte(networkdLease), 0o600); err != nil {
		t.Fatalf("write networkd lease: %v", err)
	}

	lease, err := findLeaseFileIn(networkdDir, nil, loopback.Name)
	if err != nil {
		t.Fatalf("findLeaseFileIn(%q): %v", loopback.Name, err)
	}

	want := LeaseInfo{
		Interface:    loopback.Name,
		IPAddress:    "10.44.20.142",
		SubnetMask:   "255.255.255.0",
		Gateway:      "10.44.20.1",
		ServerIP:     "10.44.20.1",
		DNSServers:   []string{"10.44.40.10", "10.44.40.11"},
		DomainName:   "msn.lab",
		LeaseTime:    24 * time.Hour,
		LeaseTimeSec: 86400,
	}
	if !reflect.DeepEqual(*lease, want) {
		t.Errorf("lease = %+v\nwant    %+v", *lease, want)
	}
}

// TestFindLeaseFileIn_PrefersNetworkd pins the search order: a dhclient file
// left under /var/lib by an earlier client must not shadow networkd's live one.
func TestFindLeaseFileIn_PrefersNetworkd(t *testing.T) {
	dir := t.TempDir()
	loopback := hostLoopback(t)

	templates := []string{filepath.Join(dir, "dhclient.%s.leases")}
	writeLeaseFile(t, filepath.Join(dir, "dhclient."+loopback.Name+".leases"))

	networkdDir := t.TempDir()
	path := filepath.Join(networkdDir, strconv.Itoa(loopback.Index))
	if err := os.WriteFile(path, []byte(networkdLease), 0o600); err != nil {
		t.Fatalf("write networkd lease: %v", err)
	}

	lease, err := findLeaseFileIn(networkdDir, templates, loopback.Name)
	if err != nil {
		t.Fatalf("findLeaseFileIn(%q): %v", loopback.Name, err)
	}
	if lease.IPAddress != "10.44.20.142" {
		t.Errorf("IPAddress = %q, want networkd's 10.44.20.142 over dhclient's 10.9.9.9", lease.IPAddress)
	}
}

func TestParseNetworkdLeaseFile_NoAddress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "1")
	if err := os.WriteFile(path, []byte("ROUTER=10.0.0.1\nDNS=10.0.0.53\n"), 0o600); err != nil {
		t.Fatalf("write lease: %v", err)
	}

	if lease, err := parseNetworkdLeaseFile(path, "eth0"); err == nil {
		t.Errorf("parseNetworkdLeaseFile = %+v, want an error for a lease without ADDRESS", lease)
	}
}

func TestParseNetworkdTimespan(t *testing.T) {
	tests := []struct {
		in     string
		want   time.Duration
		wantOK bool
	}{
		{in: "1d", want: 24 * time.Hour, wantOK: true},
		{in: "12h", want: 12 * time.Hour, wantOK: true},
		{in: "1d 2h 30min 5s", want: 26*time.Hour + 30*time.Minute + 5*time.Second, wantOK: true},
		{in: "1w", want: 7 * 24 * time.Hour, wantOK: true},
		{in: "86400", want: 24 * time.Hour, wantOK: true},
		{in: "infinity"},
		{in: ""},
		{in: "5fortnights"},
		{in: "h"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := parseNetworkdTimespan(tt.in)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("parseNetworkdTimespan(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestHostInterface covers the resolver directly: it returns the kernel's own
// entry for a real interface and refuses anything else, including the names
// that would matter for a path.
func TestHostInterface(t *testing.T) {
	loopback := hostLoopback(t)

	got, err := hostInterface(loopback.Name)
	if err != nil {
		t.Fatalf("hostInterface(%q): %v", loopback.Name, err)
	}
	if got.Name != loopback.Name || got.Index != loopback.Index {
		t.Errorf("hostInterface(%q) = %s/%d, want %s/%d",
			loopback.Name, got.Name, got.Index, loopback.Name, loopback.Index)
	}

	rejected := []string{
		"",
		"..",
		"../../etc/passwd",
		"/etc/passwd",
		loopback.Name + "/../../etc",
		strings.ToUpper(loopback.Name) + "X",
	}

	for _, name := range rejected {
		if _, nameErr := hostInterface(name); nameErr == nil {
			t.Errorf("hostInterface(%q) returned nil error, want a refusal", name)
		}
	}
}
