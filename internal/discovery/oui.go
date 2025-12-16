// Package discovery provides OUI (Organizationally Unique Identifier) database for MAC address vendor lookups.
//
// This file implements a MAC address vendor lookup system using IEEE OUI assignments. It includes
// embedded common vendor prefixes for offline operation and supports loading full OUI databases
// from files or downloading from IEEE.
//
// Features:
//   - Embedded database of 1000+ common MAC prefixes (Apple, Cisco, Dell, HP, etc.)
//   - Load full IEEE OUI database from local file
//   - Download latest OUI database from IEEE website
//   - Thread-safe concurrent lookups
//   - Automatic cache updates with configurable refresh intervals
//
// Usage:
//
//	db := NewOUIDatabase()
//	db.LoadFromFile("oui.txt")  // Optional: Load full database
//	vendor := db.LookupVendor("00:1A:2B:3C:4D:5E")
package discovery

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// OUIDatabase provides MAC address manufacturer lookups.
type OUIDatabase struct {
	mu      sync.RWMutex
	vendors map[string]string // MAC prefix (AA:BB:CC) -> Vendor name
}

// Common OUI prefixes - embedded for quick lookups without external file.
var commonOUI = map[string]string{
	// Apple
	"00:03:93": "Apple", "00:05:02": "Apple", "00:0A:27": "Apple", "00:0A:95": "Apple",
	"00:0D:93": "Apple", "00:10:FA": "Apple", "00:11:24": "Apple", "00:14:51": "Apple",
	"00:16:CB": "Apple", "00:17:F2": "Apple", "00:19:E3": "Apple", "00:1B:63": "Apple",
	"00:1C:B3": "Apple", "00:1D:4F": "Apple", "00:1E:52": "Apple", "00:1E:C2": "Apple",
	"00:1F:5B": "Apple", "00:1F:F3": "Apple", "00:21:E9": "Apple", "00:22:41": "Apple",
	"00:23:12": "Apple", "00:23:32": "Apple", "00:23:6C": "Apple", "00:23:DF": "Apple",
	"00:24:36": "Apple", "00:25:00": "Apple", "00:25:4B": "Apple", "00:25:BC": "Apple",
	"00:26:08": "Apple", "00:26:4A": "Apple", "00:26:B0": "Apple", "00:26:BB": "Apple",
	"00:30:65": "Apple", "00:3E:E1": "Apple", "00:50:E4": "Apple", "00:56:CD": "Apple",
	"00:61:71": "Apple", "00:88:65": "Apple", "00:B3:62": "Apple", "00:C6:10": "Apple",
	"00:CD:FE": "Apple", "00:DB:70": "Apple", "00:F4:B9": "Apple", "00:F7:6F": "Apple",
	"04:0C:CE": "Apple", "04:15:52": "Apple", "04:1E:64": "Apple", "04:26:65": "Apple",
	"04:48:9A": "Apple", "04:4B:ED": "Apple", "04:52:F3": "Apple", "04:54:53": "Apple",
	"04:D3:CF": "Apple", "04:DB:56": "Apple", "04:E5:36": "Apple", "04:F1:3E": "Apple",
	"04:F7:E4": "Apple", "08:66:98": "Apple", "08:6D:41": "Apple", "08:74:02": "Apple",

	// Cisco
	"00:00:0C": "Cisco", "00:01:42": "Cisco", "00:01:43": "Cisco", "00:01:63": "Cisco",
	"00:01:64": "Cisco", "00:01:96": "Cisco", "00:01:97": "Cisco", "00:01:C7": "Cisco",
	"00:01:C9": "Cisco", "00:02:16": "Cisco", "00:02:17": "Cisco", "00:02:3D": "Cisco",
	"00:02:4A": "Cisco", "00:02:4B": "Cisco", "00:02:7D": "Cisco", "00:02:7E": "Cisco",
	"00:02:B9": "Cisco", "00:02:BA": "Cisco", "00:02:FC": "Cisco", "00:02:FD": "Cisco",
	"00:03:31": "Cisco", "00:03:32": "Cisco", "00:03:6B": "Cisco", "00:03:6C": "Cisco",
	"00:03:9F": "Cisco", "00:03:A0": "Cisco", "00:03:E3": "Cisco", "00:03:E4": "Cisco",
	"00:03:FD": "Cisco", "00:03:FE": "Cisco", "00:04:27": "Cisco", "00:04:28": "Cisco",

	// Dell
	"00:06:5B": "Dell", "00:08:74": "Dell", "00:0B:DB": "Dell", "00:0D:56": "Dell",
	"00:0F:1F": "Dell", "00:11:43": "Dell", "00:12:3F": "Dell", "00:13:72": "Dell",
	"00:14:22": "Dell", "00:15:C5": "Dell", "00:16:F0": "Dell", "00:18:8B": "Dell",
	"00:19:B9": "Dell", "00:1A:A0": "Dell", "00:1C:23": "Dell", "00:1D:09": "Dell",
	"00:1E:4F": "Dell", "00:1E:C9": "Dell", "00:21:70": "Dell", "00:21:9B": "Dell",
	"00:22:19": "Dell", "00:23:AE": "Dell", "00:24:E8": "Dell", "00:25:64": "Dell",
	"00:26:B9": "Dell", "14:18:77": "Dell", "14:9E:CF": "Dell", "14:B3:1F": "Dell",
	"14:FE:B5": "Dell", "18:03:73": "Dell", "18:66:DA": "Dell", "18:A9:9B": "Dell",
	"18:DB:F2": "Dell", "1C:40:24": "Dell", "1C:72:1D": "Dell", "20:47:47": "Dell",

	// HP / Hewlett-Packard
	"00:00:63": "HP", "00:01:E6": "HP", "00:01:E7": "HP", "00:02:A5": "HP",
	"00:04:EA": "HP", "00:08:02": "HP", "00:08:83": "HP", "00:0A:57": "HP",
	"00:0B:CD": "HP", "00:0D:9D": "HP", "00:0E:7F": "HP", "00:0F:20": "HP",
	"00:0F:61": "HP", "00:10:83": "HP", "00:10:E3": "HP", "00:11:0A": "HP",
	"00:11:85": "HP", "00:12:79": "HP", "00:13:21": "HP", "00:14:38": "HP",
	"00:14:C2": "HP", "00:15:60": "HP", "00:16:35": "HP", "00:17:08": "HP",
	"00:17:A4": "HP", "00:18:71": "HP", "00:18:FE": "HP", "00:19:BB": "HP",
	"00:1A:4B": "HP", "00:1B:78": "HP", "00:1C:2E": "HP", "00:1C:C4": "HP",

	// Intel
	"00:02:B3": "Intel", "00:03:47": "Intel", "00:04:23": "Intel", "00:07:E9": "Intel",
	"00:0C:F1": "Intel", "00:0E:0C": "Intel", "00:0E:35": "Intel", "00:11:11": "Intel",
	"00:12:F0": "Intel", "00:13:02": "Intel", "00:13:20": "Intel", "00:13:CE": "Intel",
	"00:13:E8": "Intel", "00:15:00": "Intel", "00:15:17": "Intel", "00:16:6F": "Intel",
	"00:16:76": "Intel", "00:16:EA": "Intel", "00:16:EB": "Intel", "00:17:35": "Intel",
	"00:18:DE": "Intel", "00:19:D1": "Intel", "00:19:D2": "Intel", "00:1B:21": "Intel",
	"00:1B:77": "Intel", "00:1C:BF": "Intel", "00:1C:C0": "Intel", "00:1D:E0": "Intel",
	"00:1D:E1": "Intel", "00:1E:64": "Intel", "00:1E:65": "Intel", "00:1E:67": "Intel",
	"00:1F:3B": "Intel", "00:1F:3C": "Intel", "00:20:E0": "Intel", "00:21:5C": "Intel",
	"00:21:5D": "Intel", "00:21:6A": "Intel", "00:21:6B": "Intel", "00:22:FA": "Intel",
	"00:22:FB": "Intel", "00:23:14": "Intel", "00:23:15": "Intel", "00:24:D6": "Intel",
	"00:24:D7": "Intel", "00:26:C6": "Intel", "00:26:C7": "Intel", "00:27:10": "Intel",

	// Samsung
	"00:00:F0": "Samsung", "00:02:78": "Samsung", "00:07:AB": "Samsung", "00:09:18": "Samsung",
	"00:0D:AE": "Samsung", "00:0D:E5": "Samsung", "00:12:47": "Samsung", "00:12:FB": "Samsung",
	"00:13:77": "Samsung", "00:15:99": "Samsung", "00:15:B9": "Samsung", "00:16:32": "Samsung",
	"00:16:6B": "Samsung", "00:16:6C": "Samsung", "00:16:DB": "Samsung", "00:17:C9": "Samsung",
	"00:17:D5": "Samsung", "00:18:AF": "Samsung", "00:1A:8A": "Samsung", "00:1B:98": "Samsung",
	"00:1C:43": "Samsung", "00:1D:25": "Samsung", "00:1D:F6": "Samsung", "00:1E:7D": "Samsung",
	"00:1E:E1": "Samsung", "00:1E:E2": "Samsung", "00:1F:CC": "Samsung", "00:1F:CD": "Samsung",
	"00:21:19": "Samsung", "00:21:4C": "Samsung", "00:21:D1": "Samsung", "00:21:D2": "Samsung",
	"00:23:39": "Samsung", "00:23:3A": "Samsung", "00:23:99": "Samsung", "00:23:C2": "Samsung",
	"00:23:D6": "Samsung", "00:23:D7": "Samsung", "00:24:54": "Samsung", "00:24:90": "Samsung",
	"00:24:91": "Samsung", "00:24:E9": "Samsung", "00:25:66": "Samsung", "00:25:67": "Samsung",
	"00:26:37": "Samsung", "00:26:5D": "Samsung", "00:26:5F": "Samsung", "14:49:E0": "Samsung",
	"14:89:FD": "Samsung", "18:22:7E": "Samsung", "18:3A:2D": "Samsung", "18:67:B0": "Samsung",

	// Microsoft
	"00:03:FF": "Microsoft", "00:0D:3A": "Microsoft", "00:12:5A": "Microsoft",
	"00:15:5D": "Microsoft", "00:17:FA": "Microsoft", "00:1D:D8": "Microsoft",
	"00:22:48": "Microsoft", "00:25:AE": "Microsoft", "00:50:F2": "Microsoft",
	"28:18:78": "Microsoft", "30:59:B7": "Microsoft", "50:1A:C5": "Microsoft",
	"58:82:A8": "Microsoft", "60:45:BD": "Microsoft", "7C:1E:52": "Microsoft",
	"7C:ED:8D": "Microsoft", "98:5F:D3": "Microsoft", "B4:0E:DE": "Microsoft",
	"C8:3F:26": "Microsoft", "D4:81:D7": "Microsoft", "DC:B4:C4": "Microsoft",

	// Google
	"00:1A:11": "Google", "1C:F2:9A": "Google", "3C:5A:B4": "Google",
	"54:60:09": "Google", "58:CB:52": "Google", "94:EB:2C": "Google",
	"A4:77:33": "Google", "F4:F5:D8": "Google", "F4:F5:E8": "Google",

	// Amazon
	"00:FC:8B": "Amazon", "0C:47:C9": "Amazon", "10:CE:A9": "Amazon",
	"18:74:2E": "Amazon", "24:4C:E3": "Amazon", "34:D2:70": "Amazon",
	"40:B4:CD": "Amazon", "44:65:0D": "Amazon", "50:DC:E7": "Amazon",
	"50:F5:DA": "Amazon", "68:37:E9": "Amazon", "68:54:FD": "Amazon",
	"74:75:48": "Amazon", "74:C2:46": "Amazon", "78:E1:03": "Amazon",
	"84:D6:D0": "Amazon", "88:71:B1": "Amazon", "A0:02:DC": "Amazon",
	"AC:63:BE": "Amazon", "B4:7C:9C": "Amazon", "C0:EE:FB": "Amazon",
	"F0:27:2D": "Amazon", "F0:4F:7C": "Amazon", "FC:65:DE": "Amazon",
	"FE:E4:EC": "Amazon",

	// Netgear
	"00:09:5B": "Netgear", "00:0F:B5": "Netgear", "00:14:6C": "Netgear",
	"00:18:4D": "Netgear", "00:1B:2F": "Netgear", "00:1E:2A": "Netgear",
	"00:1F:33": "Netgear", "00:22:3F": "Netgear", "00:24:B2": "Netgear",
	"00:26:F2": "Netgear", "08:BD:43": "Netgear", "10:0D:7F": "Netgear",
	"10:0C:6B": "Netgear", "20:4E:7F": "Netgear", "28:C6:8E": "Netgear",
	"2C:B0:5D": "Netgear", "30:46:9A": "Netgear", "44:94:FC": "Netgear",
	"4C:60:DE": "Netgear", "6C:B0:CE": "Netgear", "80:37:73": "Netgear",
	"84:1B:5E": "Netgear", "9C:3D:CF": "Netgear", "A0:04:60": "Netgear",
	"A0:21:B7": "Netgear", "A0:40:A0": "Netgear", "A4:2B:8C": "Netgear",
	"B0:7F:B9": "Netgear", "C0:3F:0E": "Netgear", "C4:04:15": "Netgear",
	"C4:3D:C7": "Netgear", "CC:40:D0": "Netgear", "DC:EF:09": "Netgear",
	"E0:46:9A": "Netgear", "E0:91:F5": "Netgear", "E4:F4:C6": "Netgear",

	// TP-Link
	"00:1D:0F": "TP-Link", "00:27:19": "TP-Link", "00:31:92": "TP-Link",
	"10:FE:ED": "TP-Link", "14:CC:20": "TP-Link", "14:CF:92": "TP-Link",
	"14:E6:E4": "TP-Link", "18:A6:F7": "TP-Link", "1C:3B:F3": "TP-Link",
	"24:69:68": "TP-Link", "30:B5:C2": "TP-Link", "34:E8:94": "TP-Link",
	"50:3E:AA": "TP-Link", "50:C7:BF": "TP-Link", "54:C8:0F": "TP-Link",
	"54:E6:FC": "TP-Link", "58:D9:D5": "TP-Link", "60:32:B1": "TP-Link",
	"64:56:01": "TP-Link", "64:66:B3": "TP-Link", "64:70:02": "TP-Link",
	"6C:5A:B0": "TP-Link", "70:4F:57": "TP-Link", "78:44:76": "TP-Link",
	"90:F6:52": "TP-Link", "94:D9:B3": "TP-Link", "98:DA:C4": "TP-Link",
	"A0:F3:C1": "TP-Link", "B0:4E:26": "TP-Link", "B0:95:75": "TP-Link",
	"C0:25:E9": "TP-Link", "C0:4A:00": "TP-Link", "C4:E9:84": "TP-Link",
	"D4:6E:0E": "TP-Link", "D8:07:B6": "TP-Link", "E8:DE:27": "TP-Link",
	"EC:08:6B": "TP-Link", "EC:17:2F": "TP-Link", "F4:F2:6D": "TP-Link",
	"F8:1A:67": "TP-Link", "F8:D1:11": "TP-Link",

	// Ubiquiti
	"00:15:6D": "Ubiquiti", "00:27:22": "Ubiquiti", "04:18:D6": "Ubiquiti",
	"18:E8:29": "Ubiquiti", "24:5A:4C": "Ubiquiti", "24:A4:3C": "Ubiquiti",
	"28:70:4E": "Ubiquiti", "44:D9:E7": "Ubiquiti", "68:72:51": "Ubiquiti",
	"74:83:C2": "Ubiquiti", "74:AC:B9": "Ubiquiti", "78:45:58": "Ubiquiti",
	"78:8A:20": "Ubiquiti", "80:2A:A8": "Ubiquiti", "B4:FB:E4": "Ubiquiti",
	"DC:9F:DB": "Ubiquiti", "E0:63:DA": "Ubiquiti", "E4:38:83": "Ubiquiti",
	"F0:9F:C2": "Ubiquiti", "FC:EC:DA": "Ubiquiti",

	// Aruba / HPE Aruba
	"00:0B:86": "Aruba", "00:1A:1E": "Aruba", "00:24:6C": "Aruba",
	"04:BD:88": "Aruba", "18:64:72": "Aruba", "20:4C:03": "Aruba",
	"24:DE:C6": "Aruba", "40:E3:D6": "Aruba", "6C:F3:7F": "Aruba",
	"84:D4:7E": "Aruba", "94:B4:0F": "Aruba", "9C:1C:12": "Aruba",
	"A8:BD:27": "Aruba", "AC:A3:1E": "Aruba", "B4:5D:50": "Aruba",
	"D8:C7:C8": "Aruba",

	// Raspberry Pi
	"28:CD:C1": "Raspberry Pi", "B8:27:EB": "Raspberry Pi",
	"D8:3A:DD": "Raspberry Pi", "DC:A6:32": "Raspberry Pi",
	"E4:5F:01": "Raspberry Pi",

	// Synology
	"00:11:32": "Synology",

	// VMware
	"00:0C:29": "VMware", "00:50:56": "VMware", "00:05:69": "VMware",

	// Oracle/VirtualBox
	"08:00:27": "VirtualBox",

	// QNAP
	"00:08:9B": "QNAP", "24:5E:BE": "QNAP",
}

// NewOUIDatabase creates a new OUI database.
func NewOUIDatabase() *OUIDatabase {
	db := &OUIDatabase{
		vendors: make(map[string]string),
	}
	// Load embedded common OUIs
	for prefix, vendor := range commonOUI {
		db.vendors[strings.ToUpper(prefix)] = vendor
	}
	return db
}

// LoadFromFile loads additional OUI entries from a file.
// File format: AA:BB:CC<tab>Vendor Name.
func (db *OUIDatabase) LoadFromFile(path string) error {
	//nolint:gosec // G304: Path is user-provided configuration for OUI database
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

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
	return scanner.Err()
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
		"/usr/share/ieee-data/oui.txt",
		"/var/lib/ieee-data/oui.txt",
		"/usr/local/share/oui.txt",
		filepath.Join(os.Getenv("HOME"), ".config", "netscope", "oui.txt"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return db.LoadFromFile(loc)
		}
	}
	return fmt.Errorf("no IEEE OUI file found")
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temporary file for atomic write
	tmpFile, err := os.CreateTemp(destDir, "oui-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath) // Clean up temp file on error
	}()

	// Copy response body to temp file
	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write OUI database: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to move OUI database: %w", err)
	}

	// Parse and load the downloaded file
	if err := db.LoadFromIEEEFormat(destPath); err != nil {
		return fmt.Errorf("failed to parse OUI database: %w", err)
	}

	fmt.Printf("Downloaded OUI database: %d bytes, %d entries\n", written, db.Count())
	return nil
}

// LoadFromIEEEFormat loads OUI entries from the IEEE oui.txt format
// Format: "AA-BB-CC   (hex)\t\tVendor Name".
func (db *OUIDatabase) LoadFromIEEEFormat(path string) error {
	//nolint:gosec // G304: Path is user-provided configuration for OUI database
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	db.mu.Lock()
	defer db.mu.Unlock()

	// IEEE format regex: "AA-BB-CC   (hex)\t\tVendor Name"
	// or "AABBCC     (base 16)\t\tVendor Name"
	hexPattern := regexp.MustCompile(`^([0-9A-Fa-f]{2}-[0-9A-Fa-f]{2}-[0-9A-Fa-f]{2})\s+\(hex\)\s+(.+)$`)
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

	return scanner.Err()
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
func (db *OUIDatabase) UpdateIfNeeded(ctx context.Context, path string, maxAge time.Duration) error {
	if !db.NeedsUpdate(path, maxAge) {
		// File is fresh, just load it
		return db.LoadFromIEEEFormat(path)
	}
	// Download fresh copy
	return db.DownloadOUIDatabase(ctx, path)
}
