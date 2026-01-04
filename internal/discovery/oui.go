// Package discovery provides OUI (Organizationally Unique Identifier) database for MAC address vendor lookups.
//
// This file implements a MAC address vendor lookup system using IEEE OUI assignments.
// The database is populated from the IEEE OUI file which can be downloaded automatically.
//
// Features:
//   - Load full IEEE OUI database from local file
//   - Download latest OUI database from IEEE website
//   - Thread-safe concurrent lookups
//   - Automatic cache updates with configurable refresh intervals
//
// Usage:
//
//	db := NewOUIDatabase()
//	db.LoadFromFile("oui.txt")
//	vendor := db.Lookup("00:1A:2B:3C:4D:5E")
package discovery

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

// OUIDatabase provides MAC address manufacturer lookups.
type OUIDatabase struct {
	mu      sync.RWMutex
	vendors map[string]string // MAC prefix (AA:BB:CC) -> Vendor name
}

// NewOUIDatabase creates a new OUI database.
// The database starts empty and should be populated from the IEEE OUI file
// using LoadFromFile or LoadFromIEEEFormat.
func NewOUIDatabase() *OUIDatabase {
	return &OUIDatabase{
		vendors: make(map[string]string),
	}
}

// LoadFromFile loads additional OUI entries from a file.
// File format: AA:BB:CC<tab>Vendor Name.
func (db *OUIDatabase) LoadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open OUI file: %w", err)
	}
	defer func() { _ = file.Close() }()

	db.mu.Lock()
	defer db.mu.Unlock()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Skip comments and empty lines
		if line == "" || line[0] == '#' {
			continue
		}
		// Parse "AA:BB:CC\tVendor" or "AA-BB-CC\tVendor" format
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		prefix := strings.ToUpper(strings.TrimSpace(parts[0]))
		prefix = strings.ReplaceAll(prefix, "-", ":")
		vendor := strings.TrimSpace(parts[1])
		if len(prefix) >= 8 && vendor != "" {
			db.vendors[prefix[:8]] = vendor
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return fmt.Errorf("scan OUI file: %w", scanErr)
	}
	return nil
}

// Lookup returns the manufacturer for a MAC address.
// Returns empty string if not found.
func (db *OUIDatabase) Lookup(mac string) string {
	if len(mac) < 8 {
		return ""
	}
	// Normalize MAC format
	mac = strings.ToUpper(mac)
	mac = strings.ReplaceAll(mac, "-", ":")
	prefix := mac[:8]

	db.mu.RLock()
	defer db.mu.RUnlock()

	if vendor, ok := db.vendors[prefix]; ok {
		return vendor
	}
	return ""
}

// LookupWithDefault returns the manufacturer or a default if not found.
func (db *OUIDatabase) LookupWithDefault(mac, defaultVal string) string {
	if vendor := db.Lookup(mac); vendor != "" {
		return vendor
	}
	return defaultVal
}

// Count returns the number of OUI entries loaded.
func (db *OUIDatabase) Count() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.vendors)
}

// TryLoadIEEEFile attempts to load the IEEE OUI file from common locations.
func (db *OUIDatabase) TryLoadIEEEFile() error {
	locations := []string{
		"data/oui.txt", // Project data directory
		"/usr/share/ieee-data/oui.txt",
		"/var/lib/ieee-data/oui.txt",
		"/usr/local/share/oui.txt",
		filepath.Join(os.Getenv("HOME"), ".config", "seed", "oui.txt"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return db.LoadFromFile(loc)
		}
	}
	return errors.New("no IEEE OUI file found")
}

// IEEE OUI database URLs.
const (
	// IEEEOUIURL is the official IEEE OUI database URL.
	IEEEOUIURL = "https://standards-oui.ieee.org/oui/oui.txt"
	// IEEEOUICsvURL is the CSV format URL.
	IEEEOUICsvURL = "https://standards-oui.ieee.org/oui/oui.csv"
)

// DownloadOUIDatabase downloads the IEEE OUI database from the official source.
// It saves the file to the specified path and loads it into the database.
func (db *OUIDatabase) DownloadOUIDatabase(ctx context.Context, destPath string) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, IEEEOUIURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent to avoid being blocked
	req.Header.Set("User-Agent", "The Seed/1.0 (Network Discovery Tool)")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download OUI database: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if mkdirErr := os.MkdirAll(destDir, 0o750); mkdirErr != nil {
		return fmt.Errorf("failed to create directory: %w", mkdirErr)
	}

	// Create temporary file for atomic write
	tmpFile, err := os.CreateTemp(destDir, "oui-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath) // Clean up temp file on error
	}()

	// Copy response body to temp file
	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write OUI database: %w", err)
	}

	if closeErr := tmpFile.Close(); closeErr != nil {
		return fmt.Errorf("failed to close temp file: %w", closeErr)
	}

	// Atomic rename
	if renameErr := os.Rename(tmpPath, destPath); renameErr != nil {
		return fmt.Errorf("failed to move OUI database: %w", renameErr)
	}

	// Parse and load the downloaded file
	if loadErr := db.LoadFromIEEEFormat(destPath); loadErr != nil {
		return fmt.Errorf("failed to parse OUI database: %w", loadErr)
	}

	logging.GetLogger().InfoContext(ctx, "Downloaded OUI database", "bytes", written, "entries", db.Count())
	return nil
}

// LoadFromIEEEFormat loads OUI entries from the IEEE oui.txt format
// Format: "AA-BB-CC   (hex)\t\tVendor Name".
func (db *OUIDatabase) LoadFromIEEEFormat(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open IEEE OUI file: %w", err)
	}
	defer func() { _ = file.Close() }()

	db.mu.Lock()
	defer db.mu.Unlock()

	// IEEE format regex: "AA-BB-CC   (hex)\t\tVendor Name"
	// or "AABBCC     (base 16)\t\tVendor Name"
	hexPattern := regexp.MustCompile(
		`^([0-9A-Fa-f]{2}-[0-9A-Fa-f]{2}-[0-9A-Fa-f]{2})\s+\(hex\)\s+(.+)$`,
	)
	base16Pattern := regexp.MustCompile(`^([0-9A-Fa-f]{6})\s+\(base 16\)\s+(.+)$`)

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		line := scanner.Text()

		// Try hex format first (AA-BB-CC)
		if matches := hexPattern.FindStringSubmatch(line); len(matches) == 3 {
			prefix := strings.ToUpper(strings.ReplaceAll(matches[1], "-", ":"))
			vendor := strings.TrimSpace(matches[2])
			if vendor != "" {
				db.vendors[prefix] = vendor
				count++
			}
			continue
		}

		// Try base 16 format (AABBCC)
		if matches := base16Pattern.FindStringSubmatch(line); len(matches) == 3 {
			mac := strings.ToUpper(matches[1])
			prefix := fmt.Sprintf("%s:%s:%s", mac[0:2], mac[2:4], mac[4:6])
			vendor := strings.TrimSpace(matches[2])
			if vendor != "" {
				db.vendors[prefix] = vendor
				count++
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return fmt.Errorf("scan IEEE OUI file: %w", scanErr)
	}
	return nil
}

// NeedsUpdate checks if the OUI database file needs updating.
// Returns true if file doesn't exist or is older than maxAge.
func (db *OUIDatabase) NeedsUpdate(path string, maxAge time.Duration) bool {
	info, err := os.Stat(path)
	if err != nil {
		return true // File doesn't exist
	}
	return time.Since(info.ModTime()) > maxAge
}

// UpdateIfNeeded downloads a fresh OUI database if the existing one is stale.
// maxAge specifies how old the file can be before updating (e.g., 30*24*time.Hour for monthly).
func (db *OUIDatabase) UpdateIfNeeded(
	ctx context.Context,
	path string,
	maxAge time.Duration,
) error {
	if !db.NeedsUpdate(path, maxAge) {
		// File is fresh, just load it
		return db.LoadFromIEEEFormat(path)
	}
	// Download fresh copy
	return db.DownloadOUIDatabase(ctx, path)
}
