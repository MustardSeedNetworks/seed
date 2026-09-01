//go:build linux || darwin

package dns_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
)

func TestParseResolvConf(t *testing.T) {
	// Create a temp resolv.conf file.
	tempDir := t.TempDir()
	resolvPath := filepath.Join(tempDir, "resolv.conf")

	content := `# This is a comment
nameserver 8.8.8.8
nameserver 8.8.4.4
# Another comment
nameserver 1.1.1.1
`
	err := os.WriteFile(resolvPath, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	servers := dns.ExportParseResolvConf(resolvPath)
	if len(servers) != 3 {
		t.Errorf("expected 3 servers, got %d", len(servers))
	}
	if servers[0] != "8.8.8.8" {
		t.Errorf("expected first server '8.8.8.8', got %q", servers[0])
	}
	if servers[1] != "8.8.4.4" {
		t.Errorf("expected second server '8.8.4.4', got %q", servers[1])
	}
	if servers[2] != "1.1.1.1" {
		t.Errorf("expected third server '1.1.1.1', got %q", servers[2])
	}
}

func TestParseResolvConfNonexistent(t *testing.T) {
	servers := dns.ExportParseResolvConf("/nonexistent/path/resolv.conf")
	if servers == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(servers) != 0 {
		t.Errorf("expected 0 servers for nonexistent file, got %d", len(servers))
	}
}

func TestParseResolvConfEmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	resolvPath := filepath.Join(tempDir, "resolv.conf")

	err := os.WriteFile(resolvPath, []byte(""), 0o644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	servers := dns.ExportParseResolvConf(resolvPath)
	if len(servers) != 0 {
		t.Errorf("expected 0 servers for empty file, got %d", len(servers))
	}
}

func TestParseResolvConfCommentsOnly(t *testing.T) {
	tempDir := t.TempDir()
	resolvPath := filepath.Join(tempDir, "resolv.conf")

	content := `# Just comments
# No nameservers
# Another comment
`
	err := os.WriteFile(resolvPath, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	servers := dns.ExportParseResolvConf(resolvPath)
	if len(servers) != 0 {
		t.Errorf("expected 0 servers for comments-only file, got %d", len(servers))
	}
}

func TestParseResolvConfMalformedLine(t *testing.T) {
	tempDir := t.TempDir()
	resolvPath := filepath.Join(tempDir, "resolv.conf")

	content := `nameserver
nameserver 8.8.8.8
nameserver
search example.com
`
	err := os.WriteFile(resolvPath, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	servers := dns.ExportParseResolvConf(resolvPath)
	// Should only parse the valid nameserver line.
	if len(servers) != 1 {
		t.Errorf("expected 1 server for malformed file, got %d", len(servers))
	}
	if len(servers) > 0 && servers[0] != "8.8.8.8" {
		t.Errorf("expected server '8.8.8.8', got %q", servers[0])
	}
}

func TestParseResolvConfWhitespace(t *testing.T) {
	tempDir := t.TempDir()
	resolvPath := filepath.Join(tempDir, "resolv.conf")

	content := `   nameserver 8.8.8.8
	nameserver 8.8.4.4
nameserver		1.1.1.1
`
	err := os.WriteFile(resolvPath, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	servers := dns.ExportParseResolvConf(resolvPath)
	if len(servers) != 3 {
		t.Errorf("expected 3 servers with whitespace, got %d", len(servers))
	}
}
