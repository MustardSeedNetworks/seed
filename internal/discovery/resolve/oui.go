package resolve

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

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// embeddedOUI is the IEEE OUI registry (standards-oui.ieee.org/oui/oui.txt)
// baked into the binary at build time. It is the single source of truth and the
// always-present baseline every OUIDatabase starts from — no hand-maintained
// vendor table, no runtime file required (air-gapped-safe, no phone-home).
// Operators can still layer a fresher on-disk/downloaded copy on top at runtime
// via LoadFromFile / LoadFromIEEEFormat / UpdateIfNeeded. Refresh by replacing
// this file from the IEEE source and rebuilding.
//
//go:embed oui.txt
var embeddedOUI []byte

// embeddedOUIKey is the ieeeOUICache key under which the parsed embedded
// registry is memoised (parsing the ~6.5MB file is done once per process).
const embeddedOUIKey = "<embedded-ieee-oui>"

// ouiFieldSplitLimit is the number of parts to split the OUI file line into.
const ouiFieldSplitLimit = 2

// OUI lookup constants.
const (
	macPrefixMinLen  = 8  // Minimum MAC address length for lookup (AA:BB:CC)
	regexMatchCount  = 3  // Expected number of regex matches (full match + 2 groups)
	ouiDownloadTimeS = 60 // HTTP timeout in seconds for OUI database download
)

// OUIDatabase provides MAC address manufacturer lookups.
type OUIDatabase struct {
	mu      sync.RWMutex
	vendors map[string]string // MAC prefix (AA:BB:CC) -> Vendor name
}

// NewOUIDatabase creates an OUI database seeded from the embedded IEEE registry
// (the single source of truth, baked into the binary). The full ~39k assignments
// are available immediately with no runtime file or network. Callers may layer a
// fresher on-disk/downloaded copy on top via LoadFromFile / LoadFromIEEEFormat /
// UpdateIfNeeded. The embedded file is parsed once per process and memoised.
func NewOUIDatabase() *OUIDatabase {
	db := &OUIDatabase{vendors: make(map[string]string, estimatedOUIEntries)}

	base, err := cachedVendors(embeddedOUIKey, func() (map[string]string, error) {
		return parseIEEEOUIReader(bytes.NewReader(embeddedOUI))
	})
	if err != nil {
		logging.GetLogger().Warn("Failed to parse embedded OUI database", "error", err)
		return db
	}
	maps.Copy(db.vendors, base)
	return db
}

// LoadFromFile loads additional OUI entries from a file.
// File format: AA:BB:CC<tab>Vendor Name.
//
// path is [filepath.Clean]'d to normalize traversal sequences. This is an
// administrative load operation invoked at boot or by privileged code
// paths — not user-controlled at runtime — so the cleaning is purely
// defensive (and to satisfy gosec's taint analysis).
func (db *OUIDatabase) LoadFromFile(path string) error {
	cleanPath := filepath.Clean(path)
	file, err := os.Open(cleanPath)
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
		parts := strings.SplitN(line, "\t", ouiFieldSplitLimit)
		if len(parts) != ouiFieldSplitLimit {
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
	if len(mac) < macPrefixMinLen {
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
		//nolint:gosec // G703: locations is a literal hardcoded list of well-known OUI database paths
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
		Timeout: ouiDownloadTimeS * time.Second,
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

// ieeeOUICache caches parsed IEEE OUI files keyed by absolute path. The
// 6.5MB+ oui.txt takes ~hundreds of ms to parse and was being parsed once
// per DeviceDiscovery construction — tests creating many DeviceDiscovery
// instances paid that cost N times. The cache makes repeat loads ~free.
//
//nolint:gochecknoglobals // Intentional thread-safe cache: same singleton pattern as i18n, config, messages.
var ieeeOUICache sync.Map // map[string]*ieeeOUIEntry

type ieeeOUIEntry struct {
	once    sync.Once
	vendors map[string]string
	err     error
}

// estimatedOUIEntries is the initial capacity hint for the vendors map.
// IEEE oui.txt has ~50k assignments today; over-allocating slightly is
// cheaper than rehashing during the parse.
const estimatedOUIEntries = 50000

// cachedVendors memoises a parsed vendor map under key in ieeeOUICache, running
// parse at most once per key for the process lifetime. Both the embedded
// registry (keyed by embeddedOUIKey) and on-disk files (keyed by absolute path)
// share this cache so each ~6.5MB parse happens once.
func cachedVendors(key string, parse func() (map[string]string, error)) (map[string]string, error) {
	val, _ := ieeeOUICache.LoadOrStore(key, &ieeeOUIEntry{})
	entry, ok := val.(*ieeeOUIEntry)
	if !ok {
		return nil, fmt.Errorf("ouiCache: unexpected entry type %T for key %q", val, key)
	}
	entry.once.Do(func() {
		entry.vendors, entry.err = parse()
	})
	return entry.vendors, entry.err
}

// LoadFromIEEEFormat loads OUI entries from the IEEE oui.txt format
// Format: "AA-BB-CC   (hex)\t\tVendor Name".
func (db *OUIDatabase) LoadFromIEEEFormat(path string) error {
	abs, absErr := filepath.Abs(path)
	if absErr != nil {
		abs = path
	}
	vendors, err := cachedVendors(abs, func() (map[string]string, error) {
		return parseIEEEOUIFile(path)
	})
	if err != nil {
		return err
	}

	db.mu.Lock()
	defer db.mu.Unlock()
	maps.Copy(db.vendors, vendors)
	logging.GetLogger().Debug("Loaded OUI database from cache", "path", abs, "entries", len(vendors))
	return nil
}

// parseIEEEOUIFile parses an IEEE oui.txt file into a vendor map. Called
// once per path via the ieeeOUICache.
func parseIEEEOUIFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open IEEE OUI file: %w", err)
	}
	defer func() { _ = file.Close() }()
	return parseIEEEOUIReader(file)
}

// parseIEEEOUIReader parses IEEE oui.txt content into a vendor map. Shared by the
// embedded-registry loader (bytes reader) and the on-disk loader (file reader).
func parseIEEEOUIReader(r io.Reader) (map[string]string, error) {
	// IEEE format regex: "AA-BB-CC   (hex)\t\tVendor Name"
	// or "AABBCC     (base 16)\t\tVendor Name"
	hexPattern := regexp.MustCompile(
		`^([0-9A-Fa-f]{2}-[0-9A-Fa-f]{2}-[0-9A-Fa-f]{2})\s+\(hex\)\s+(.+)$`,
	)
	base16Pattern := regexp.MustCompile(`^([0-9A-Fa-f]{6})\s+\(base 16\)\s+(.+)$`)

	vendors := make(map[string]string, estimatedOUIEntries)
	scanner := bufio.NewScanner(r)
	count := 0
	for scanner.Scan() {
		line := scanner.Text()

		// Try hex format first (AA-BB-CC)
		if matches := hexPattern.FindStringSubmatch(line); len(matches) == regexMatchCount {
			prefix := strings.ToUpper(strings.ReplaceAll(matches[1], "-", ":"))
			vendor := strings.TrimSpace(matches[2])
			if vendor != "" {
				vendors[prefix] = vendor
				count++
			}
			continue
		}

		// Try base 16 format (AABBCC)
		if matches := base16Pattern.FindStringSubmatch(line); len(matches) == regexMatchCount {
			mac := strings.ToUpper(matches[1])
			prefix := fmt.Sprintf("%s:%s:%s", mac[0:2], mac[2:4], mac[4:6])
			vendor := strings.TrimSpace(matches[2])
			if vendor != "" {
				vendors[prefix] = vendor
				count++
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("scan IEEE OUI file: %w", scanErr)
	}
	logging.GetLogger().Debug("Parsed IEEE OUI data", "entries", count)
	return vendors, nil
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
