// Package discovery_test provides OUI database tests.
package discovery_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestNewOUIDatabase(t *testing.T) {
	db := discovery.NewOUIDatabase()

	if db == nil {
		t.Fatal("NewOUIDatabase returned nil")
	}

	// Should have embedded common OUIs loaded
	count := db.Count()
	if count < 100 {
		t.Errorf("Expected at least 100 embedded OUI entries, got %d", count)
	}
}

func TestOUILookup(t *testing.T) {
	db := discovery.NewOUIDatabase()

	tests := []struct {
		mac      string
		expected string
	}{
		{"00:00:0C:12:34:56", "Cisco"},        // Cisco MAC
		{"00:03:93:AB:CD:EF", "Apple"},        // Apple MAC
		{"B8:27:EB:00:00:00", "Raspberry Pi"}, // Raspberry Pi
		{"00:50:56:12:34:56", "VMware"},       // VMware
		{"08:00:27:AB:CD:EF", "VirtualBox"},   // VirtualBox
		{"00:00:00:00:00:00", ""},             // Unknown - should return empty
		{"ff:ff:ff:ff:ff:ff", ""},             // Broadcast - no vendor
		{"00-00-0C-12-34-56", "Cisco"},        // Hyphen format
		{"00000C123456", ""},                  // Compact format (not supported for lookup)
	}

	for _, tt := range tests {
		t.Run(tt.mac, func(t *testing.T) {
			result := db.Lookup(tt.mac)
			if result != tt.expected {
				t.Errorf("Lookup(%q) = %q, want %q", tt.mac, result, tt.expected)
			}
		})
	}
}

func TestOUILookupWithDefault(t *testing.T) {
	db := discovery.NewOUIDatabase()

	// Known vendor
	result := db.LookupWithDefault("00:00:0C:12:34:56", "Unknown")
	if result != "Cisco" {
		t.Errorf("Expected Cisco, got %q", result)
	}

	// Unknown vendor - should return default
	result = db.LookupWithDefault("00:00:00:00:00:00", "Unknown")
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

	db := discovery.NewOUIDatabase()
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

	db := discovery.NewOUIDatabase()
	initialCount := db.Count()

	if err := db.LoadFromIEEEFormat(ouiFile); err != nil {
		t.Fatalf("LoadFromIEEEFormat failed: %v", err)
	}

	// Should have added entries from the file
	if db.Count() <= initialCount {
		t.Errorf("Expected more entries after loading IEEE format, got %d (was %d)", db.Count(), initialCount)
	}

	// Verify lookups work - IEEE format converts AA-BB-CC to AA:BB:CC
	if v := db.Lookup("AA:BB:CC:11:22:33"); v != "Test Company Inc" {
		t.Errorf("Expected 'Test Company Inc', got %q", v)
	}
}

func TestOUINeedsUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	ouiFile := filepath.Join(tmpDir, "oui.txt")

	db := discovery.NewOUIDatabase()

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

	tmpDir := t.TempDir()
	ouiFile := filepath.Join(tmpDir, "oui.txt")

	db := discovery.NewOUIDatabase()
	initialCount := db.Count()

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

	// Verify entries were loaded
	if db.Count() <= initialCount {
		t.Errorf("Expected more entries after download, got %d (was %d)", db.Count(), initialCount)
	}

	t.Logf("Downloaded OUI database: %d bytes, %d entries", info.Size(), db.Count())
}

func TestOUIUpdateIfNeeded(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	tmpDir := t.TempDir()
	ouiFile := filepath.Join(tmpDir, "oui.txt")

	db := discovery.NewOUIDatabase()

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
