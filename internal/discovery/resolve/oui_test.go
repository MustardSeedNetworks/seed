package resolve_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery/resolve"
)

func TestNewOUIDatabase(t *testing.T) {
	db := resolve.NewOUIDatabase()

	if db == nil {
		t.Fatal("NewOUIDatabase returned nil")
	}

	// NewOUIDatabase seeds from the embedded IEEE registry (~39k assignments).
	count := db.Count()
	if count < 30000 {
		t.Errorf("Expected the embedded IEEE registry (>=30k entries), got %d", count)
	}
}

func TestOUILookup(t *testing.T) {
	db := resolve.NewOUIDatabase()

	// Vendor names come from the embedded IEEE registry (e.g. "Apple, Inc."),
	// so assert the brand is contained rather than an exact short string — the
	// IEEE org text changes over time but the brand substring is stable.
	contains := []struct {
		mac   string
		brand string
	}{
		{"00:00:0C:12:34:56", "Cisco"},        // Cisco Systems, Inc
		{"00:03:93:AB:CD:EF", "Apple"},        // Apple, Inc.
		{"B8:27:EB:00:00:00", "Raspberry Pi"}, // Raspberry Pi Foundation
		{"00:50:56:12:34:56", "VMware"},       // VMware, Inc.
		{"00-00-0C-12-34-56", "Cisco"},        // Hyphen format normalises
	}
	for _, tt := range contains {
		t.Run(tt.mac, func(t *testing.T) {
			result := db.Lookup(tt.mac)
			if !strings.Contains(result, tt.brand) {
				t.Errorf("Lookup(%q) = %q, want a string containing %q", tt.mac, result, tt.brand)
			}
		})
	}

	empty := []string{
		"02:00:00:00:00:00", // locally-administered (never IEEE-assigned) -> empty
		"00000C123456",      // compact format (not supported for lookup) -> empty
	}
	for _, mac := range empty {
		t.Run("empty/"+mac, func(t *testing.T) {
			if result := db.Lookup(mac); result != "" {
				t.Errorf("Lookup(%q) = %q, want empty", mac, result)
			}
		})
	}
}

func TestOUILookupWithDefault(t *testing.T) {
	db := resolve.NewOUIDatabase()

	// Known vendor (embedded IEEE registry).
	result := db.LookupWithDefault("00:00:0C:12:34:56", "Unknown")
	if !strings.Contains(result, "Cisco") {
		t.Errorf("Expected a Cisco vendor string, got %q", result)
	}

	// Unknown vendor (locally-administered, never IEEE-assigned) - returns default
	result = db.LookupWithDefault("02:00:00:00:00:00", "Unknown")
	if result != "Unknown" {
		t.Errorf("Expected Unknown, got %q", result)
	}
}

func TestOUILoadFromFile(t *testing.T) {
	// Create a temp OUI file
	tmpDir := t.TempDir()
	ouiFile := filepath.Join(tmpDir, "oui.txt")

	content := "# Test OUI file\nAA:BB:CC\tTest Vendor\nDD:EE:FF\tAnother Vendor\n"
	if err := os.WriteFile(ouiFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write temp OUI file: %v", err)
	}

	db := resolve.NewOUIDatabase()
	initialCount := db.Count()

	if err := db.LoadFromFile(ouiFile); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Should have added 2 entries
	if db.Count() < initialCount+2 {
		t.Errorf("Expected at least %d entries, got %d", initialCount+2, db.Count())
	}

	// Verify lookups work
	if v := db.Lookup("AA:BB:CC:11:22:33"); v != "Test Vendor" {
		t.Errorf("Expected 'Test Vendor', got %q", v)
	}
	if v := db.Lookup("DD:EE:FF:44:55:66"); v != "Another Vendor" {
		t.Errorf("Expected 'Another Vendor', got %q", v)
	}
}

func TestOUILoadFromIEEEFormat(t *testing.T) {
	// Create a temp IEEE format OUI file
	tmpDir := t.TempDir()
	ouiFile := filepath.Join(tmpDir, "ieee-oui.txt")

	// IEEE format uses "(hex)" and "(base 16)" markers
	content := `OUI/MA-L				Organization
company_id		Organization
				Address

AA-BB-CC   (hex)		Test Company Inc
AABBCC     (base 16)		Test Company Inc
				123 Main St
				City, ST 12345
				US

DD-EE-FF   (hex)		Another Corp
DDEEFF     (base 16)		Another Corp
				456 Oak Ave
				Town, ST 67890
				US
`
	if err := os.WriteFile(ouiFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write temp IEEE OUI file: %v", err)
	}

	db := resolve.NewOUIDatabase()
	initialCount := db.Count()

	if err := db.LoadFromIEEEFormat(ouiFile); err != nil {
		t.Fatalf("LoadFromIEEEFormat failed: %v", err)
	}

	// Should have added entries from the file
	if db.Count() <= initialCount {
		t.Errorf(
			"Expected more entries after loading IEEE format, got %d (was %d)",
			db.Count(),
			initialCount,
		)
	}

	// Verify lookups work - IEEE format converts AA-BB-CC to AA:BB:CC
	if v := db.Lookup("AA:BB:CC:11:22:33"); v != "Test Company Inc" {
		t.Errorf("Expected 'Test Company Inc', got %q", v)
	}
}

func TestOUINeedsUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	ouiFile := filepath.Join(tmpDir, "oui.txt")

	db := resolve.NewOUIDatabase()

	// Non-existent file should need update
	if !db.NeedsUpdate(ouiFile, 24*time.Hour) {
		t.Error("Non-existent file should need update")
	}

	// Create the file
	if err := os.WriteFile(ouiFile, []byte("# empty\n"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Fresh file should not need update
	if db.NeedsUpdate(ouiFile, 24*time.Hour) {
		t.Error("Fresh file should not need update")
	}

	// File older than maxAge should need update
	oldTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(ouiFile, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to change file time: %v", err)
	}

	if !db.NeedsUpdate(ouiFile, 24*time.Hour) {
		t.Error("Old file should need update")
	}
}

func TestOUIDownloadDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}
	if os.Getenv("SKIP_NETWORK_TESTS") != "" {
		t.Skip("SKIP_NETWORK_TESTS set — skipping network-dependent test")
	}

	tmpDir := t.TempDir()
	ouiFile := filepath.Join(tmpDir, "oui.txt")

	db := resolve.NewOUIDatabase()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Test download from IEEE - this verifies the URL is correct and accessible
	err := db.DownloadOUIDatabase(ctx, ouiFile)
	if err != nil {
		t.Fatalf("DownloadOUIDatabase failed: %v", err)
	}

	// Verify file was created
	info, err := os.Stat(ouiFile)
	if err != nil {
		t.Fatalf("OUI file not created: %v", err)
	}

	// IEEE OUI file is about 6MB
	if info.Size() < 1000000 {
		t.Errorf("OUI file too small: %d bytes (expected > 1MB)", info.Size())
	}

	// The DB already carries the embedded IEEE baseline; a fresh download loads
	// the same registry on top, so assert a healthy full set rather than growth.
	if db.Count() < 30000 {
		t.Errorf("Expected the full IEEE registry after download, got %d", db.Count())
	}

	t.Logf("Downloaded OUI database: %d bytes, %d entries", info.Size(), db.Count())
}

func TestOUIUpdateIfNeeded(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}
	if os.Getenv("SKIP_NETWORK_TESTS") != "" {
		t.Skip("SKIP_NETWORK_TESTS set — skipping network-dependent test")
	}

	tmpDir := t.TempDir()
	ouiFile := filepath.Join(tmpDir, "oui.txt")

	db := resolve.NewOUIDatabase()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// First call should download since file doesn't exist
	err := db.UpdateIfNeeded(ctx, ouiFile, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("UpdateIfNeeded (first) failed: %v", err)
	}

	firstCount := db.Count()

	// Second call with fresh file should just load, not download
	err = db.UpdateIfNeeded(ctx, ouiFile, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("UpdateIfNeeded (second) failed: %v", err)
	}

	// Count should be similar (same data)
	if db.Count() != firstCount {
		t.Logf("Count changed: %d -> %d (expected same for fresh file)", firstCount, db.Count())
	}
}
