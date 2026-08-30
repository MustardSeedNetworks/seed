//go:build linux

package dhcp

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

// hostLoopbackName returns the host's loopback interface name. Enumeration
// needs no privileges, so a failure here is a real problem rather than a reason
// to skip.
func hostLoopbackName(t *testing.T) string {
	t.Helper()

	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("cannot enumerate interfaces: %v", err)
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			return iface.Name
		}
	}

	t.Fatal("no loopback interface found; every host this suite runs on has one")

	return ""
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

	lease, err := findLeaseFileIn(templates, traversal)
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

	name := hostLoopbackName(t)
	writeLeaseFile(t, filepath.Join(dir, "dhclient."+name+".leases"))

	lease, err := findLeaseFileIn(templates, name)
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

// TestHostInterfaceName covers the resolver directly: it returns the kernel's
// own spelling for a real interface and refuses anything else, including the
// names that would matter for a path.
func TestHostInterfaceName(t *testing.T) {
	loopback := hostLoopbackName(t)

	got, err := hostInterfaceName(loopback)
	if err != nil {
		t.Fatalf("hostInterfaceName(%q): %v", loopback, err)
	}
	if got != loopback {
		t.Errorf("hostInterfaceName(%q) = %q, want the same name back", loopback, got)
	}

	rejected := []string{
		"",
		"..",
		"../../etc/passwd",
		"/etc/passwd",
		loopback + "/../../etc",
		strings.ToUpper(loopback) + "X",
	}

	for _, name := range rejected {
		if _, nameErr := hostInterfaceName(name); nameErr == nil {
			t.Errorf("hostInterfaceName(%q) returned nil error, want a refusal", name)
		}
	}
}
