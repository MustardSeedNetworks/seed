package discovery

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
	"context"
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

	"github.com/krisarmstrong/seed/internal/logging"
)

// ouiFieldSplitLimit is the number of parts to split the OUI file line into.
const ouiFieldSplitLimit = 2

// OUI lookup constants.
const (
	macPrefixMinLen       = 8   // Minimum MAC address length for lookup (AA:BB:CC)
	regexMatchCount       = 3   // Expected number of regex matches (full match + 2 groups)
	ouiDownloadTimeS      = 60  // HTTP timeout in seconds for OUI database download
	commonOUIEntriesCount = 135 // Number of pre-loaded common vendor entries
)

// OUIDatabase provides MAC address manufacturer lookups.
type OUIDatabase struct {
	mu      sync.RWMutex
	vendors map[string]string // MAC prefix (AA:BB:CC) -> Vendor name
}

// getCommonOUIEntries returns frequently encountered vendor MAC prefixes.
// This provides basic functionality without requiring the full IEEE database.
func getCommonOUIEntries() map[string]string {
	result := make(map[string]string, commonOUIEntriesCount)
	maps.Copy(result, getAppleVMwareOUIEntries())
	maps.Copy(result, getDellOUIEntries())
	maps.Copy(result, getHPOUIEntries())
	return result
}

// getAppleVMwareOUIEntries returns Apple, VMware, VirtualBox, Raspberry Pi, Google, and Cisco entries.
func getAppleVMwareOUIEntries() map[string]string {
	return map[string]string{
		"00:00:0C": "Cisco",
		"00:03:93": "Apple",
		"00:05:02": "Apple",
		"00:0A:95": "Apple",
		"00:0D:93": "Apple",
		"00:11:24": "Apple",
		"00:14:51": "Apple",
		"00:17:F2": "Apple",
		"00:1B:63": "Apple",
		"00:1C:B3": "Apple",
		"00:1E:C2": "Apple",
		"00:21:E9": "Apple",
		"00:22:41": "Apple",
		"00:23:12": "Apple",
		"00:23:32": "Apple",
		"00:23:6C": "Apple",
		"00:23:DF": "Apple",
		"00:24:36": "Apple",
		"00:25:00": "Apple",
		"00:25:4B": "Apple",
		"00:25:BC": "Apple",
		"00:26:08": "Apple",
		"00:26:4A": "Apple",
		"00:26:B0": "Apple",
		"00:26:BB": "Apple",
		"00:50:56": "VMware",
		"00:0C:29": "VMware",
		"00:1C:14": "VMware",
		"00:50:C2": "VMware",
		"08:00:27": "VirtualBox",
		"0A:00:27": "VirtualBox",
		"B8:27:EB": "Raspberry Pi",
		"DC:A6:32": "Raspberry Pi",
		"E4:5F:01": "Raspberry Pi",
		"28:CD:C1": "Raspberry Pi",
		"D8:3A:DD": "Raspberry Pi",
		"00:1A:11": "Google",
	}
}

// getDellOUIEntries returns Dell MAC address prefixes.
func getDellOUIEntries() map[string]string {
	return map[string]string{
		"00:1A:A0": "Dell",
		"00:06:5B": "Dell",
		"00:14:22": "Dell",
		"00:15:C5": "Dell",
		"00:18:8B": "Dell",
		"00:19:B9": "Dell",
		"00:1C:23": "Dell",
		"00:1D:09": "Dell",
		"00:1E:4F": "Dell",
		"00:21:9B": "Dell",
		"00:21:70": "Dell",
		"00:22:19": "Dell",
		"00:24:E8": "Dell",
		"00:25:64": "Dell",
		"00:26:B9": "Dell",
		"00:0D:56": "Dell",
	}
}

// getHPOUIEntries returns HP/Hewlett-Packard MAC address prefixes.
func getHPOUIEntries() map[string]string {
	return map[string]string{
		"3C:D9:2B": "HP",
		"00:14:38": "HP",
		"00:17:A4": "HP",
		"00:1A:4B": "HP",
		"00:1B:78": "HP",
		"00:1C:C4": "HP",
		"00:1E:0B": "HP",
		"00:1F:29": "HP",
		"00:21:5A": "HP",
		"00:22:64": "HP",
		"00:23:7D": "HP",
		"00:24:81": "HP",
		"00:25:B3": "HP",
		"00:26:55": "HP",
		"00:30:6E": "HP",
		"00:0B:CD": "HP",
		"00:0D:9D": "HP",
		"00:0E:7F": "HP",
		"00:0F:20": "HP",
		"00:10:83": "HP",
		"00:11:0A": "HP",
		"00:11:85": "HP",
		"00:12:79": "HP",
		"00:13:21": "HP",
		"14:02:EC": "HP",
		"18:A9:05": "HP",
		"1C:C1:DE": "HP",
		"2C:27:D7": "HP",
		"2C:41:38": "HP",
		"2C:44:FD": "HP",
		"2C:59:E5": "HP",
		"30:8D:99": "HP",
		"34:64:A9": "HP",
		"38:63:BB": "HP",
		"3C:4A:92": "HP",
		"40:B0:34": "HP",
		"44:1E:A1": "HP",
		"44:31:92": "HP",
		"48:0F:CF": "HP",
		"48:DF:37": "HP",
		"54:80:28": "HP",
		"64:51:06": "HP",
		"68:B5:99": "HP",
		"6C:3B:E5": "HP",
		"70:10:6F": "HP",
		"74:46:A0": "HP",
		"78:AC:C0": "HP",
		"80:C1:6E": "HP",
		"84:34:97": "HP",
		"88:51:FB": "HP",
		"8C:DC:D4": "HP",
		"94:57:A5": "HP",
		"98:4B:E1": "HP",
		"9C:8E:99": "HP",
		"9C:B6:54": "HP",
		"A0:1D:48": "HP",
		"A0:2B:B8": "HP",
		"A0:D3:C1": "HP",
		"A4:5D:36": "HP",
		"AC:16:2D": "HP",
		"B0:5A:DA": "HP",
		"B4:99:BA": "HP",
		"B4:B5:2F": "HP",
		"BC:EA:FA": "HP",
		"C0:91:34": "HP",
		"C4:34:6B": "HP",
		"C8:B5:AD": "HP",
		"CC:3E:5F": "HP",
		"D0:7E:28": "HP",
		"D4:85:64": "HP",
		"D4:C9:EF": "HP",
		"D8:9D:67": "HP",
		"DC:4A:3E": "HP",
		"E0:07:1B": "HP",
		"E4:11:5B": "HP",
		"E8:39:35": "HP",
		"EC:8E:B5": "HP",
		"EC:B1:D7": "HP",
		"F0:92:1C": "HP",
		"F4:03:43": "HP",
		"FC:15:B4": "HP",
	}
}

// NewOUIDatabase creates a new OUI database with common vendor entries preloaded.
// For full IEEE database coverage, call LoadFromIEEEFormat or TryLoadIEEEFile.
func NewOUIDatabase() *OUIDatabase {
	commonEntries := getCommonOUIEntries()
	db := &OUIDatabase{
		vendors: make(map[string]string, len(commonEntries)),
	}
	// Preload common OUI entries
	maps.Copy(db.vendors, commonEntries)
	return db
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

// LoadFromIEEEFormat loads OUI entries from the IEEE oui.txt format
// Format: "AA-BB-CC   (hex)\t\tVendor Name".
func (db *OUIDatabase) LoadFromIEEEFormat(path string) error {
	abs, absErr := filepath.Abs(path)
	if absErr != nil {
		abs = path
	}
	val, _ := ieeeOUICache.LoadOrStore(abs, &ieeeOUIEntry{})
	entry, ok := val.(*ieeeOUIEntry)
	if !ok {
		return fmt.Errorf("ouiCache: unexpected entry type %T for path %q", val, abs)
	}
	entry.once.Do(func() {
		entry.vendors, entry.err = parseIEEEOUIFile(path)
	})
	if entry.err != nil {
		return entry.err
	}

	db.mu.Lock()
	defer db.mu.Unlock()
	maps.Copy(db.vendors, entry.vendors)
	logging.GetLogger().Debug("Loaded OUI database from cache", "path", abs, "entries", len(entry.vendors))
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

	// IEEE format regex: "AA-BB-CC   (hex)\t\tVendor Name"
	// or "AABBCC     (base 16)\t\tVendor Name"
	hexPattern := regexp.MustCompile(
		`^([0-9A-Fa-f]{2}-[0-9A-Fa-f]{2}-[0-9A-Fa-f]{2})\s+\(hex\)\s+(.+)$`,
	)
	base16Pattern := regexp.MustCompile(`^([0-9A-Fa-f]{6})\s+\(base 16\)\s+(.+)$`)

	vendors := make(map[string]string, estimatedOUIEntries)
	scanner := bufio.NewScanner(file)
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
	logging.GetLogger().Info("Parsed IEEE OUI file", "path", path, "entries", count)
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
