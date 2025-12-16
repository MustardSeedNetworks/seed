// Package detection provides intelligent network interface auto-detection.
// Chipset database module contains a curated list of network interface chipsets
// with quality ratings, capability flags (TDR, DOM), and MAC OUI prefixes.
//
// The database can be loaded from an external YAML file for easy updates,
// or falls back to an embedded default if no file is present.
package detection

import (
	_ "embed"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed chipsets.yaml
var embeddedChipsetData []byte

// chipsetYAML represents the YAML structure for chipset data.
type chipsetYAML struct {
	Chipsets []chipsetEntry `yaml:"chipsets"`
}

// chipsetEntry represents a single chipset in the YAML file.
type chipsetEntry struct {
	Vendor    string   `yaml:"vendor"`
	Model     string   `yaml:"model"`
	Speed     string   `yaml:"speed"`
	Quality   int      `yaml:"quality"`
	HasTDR    bool     `yaml:"has_tdr"`
	HasDOM    bool     `yaml:"has_dom"`
	OUIPrefix []string `yaml:"oui_prefix"`
	PCIVendor string   `yaml:"pci_vendor"`
	PCIDevice []string `yaml:"pci_device"`
	Keywords  []string `yaml:"keywords"`
}

// ChipsetInfo contains metadata about a network interface chipset.
type ChipsetInfo struct {
	Vendor    string   // Manufacturer name
	Model     string   // Chipset model
	Speed     string   // Max speed category: "1G", "2.5G", "5G", "10G", "25G", "40G", "100G"
	Quality   int      // Quality rating 1-100 (reliability, driver maturity, feature support)
	HasTDR    bool     // Supports Time Domain Reflectometry for cable testing
	HasDOM    bool     // Supports Digital Optical Monitoring (SFP+/QSFP)
	OUIPrefix []string // MAC address OUI prefixes for identification
	PCIVendor string   // PCI Vendor ID (for sysfs identification)
	PCIDevice []string // PCI Device IDs
	Keywords  []string // Keywords found in driver/device names
}

// ChipsetDatabase provides chipset identification.
type ChipsetDatabase struct {
	chipsets []ChipsetInfo
	ouiMap   map[string]*ChipsetInfo
}

// NewChipsetDatabase creates a populated chipset database.
// Attempts to load from external file first, falls back to embedded data.
func NewChipsetDatabase() *ChipsetDatabase {
	db := &ChipsetDatabase{
		ouiMap: make(map[string]*ChipsetInfo),
	}

	// Try to load from external file first (allows updates without rebuild)
	if chipsets, err := loadChipsetsFromFile(); err == nil {
		db.chipsets = chipsets
	} else {
		// Fall back to embedded data
		db.chipsets = loadChipsetsFromEmbedded()
	}

	// Build OUI lookup map
	for i := range db.chipsets {
		for _, oui := range db.chipsets[i].OUIPrefix {
			db.ouiMap[strings.ToLower(oui)] = &db.chipsets[i]
		}
	}

	return db
}

// NewChipsetDatabaseFromFile creates a database from a specific YAML file.
// Returns error if file cannot be loaded.
func NewChipsetDatabaseFromFile(path string) (*ChipsetDatabase, error) {
	//nolint:gosec // G304: path is provided by caller/config, not user input
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	chipsets, err := parseChipsetYAML(data)
	if err != nil {
		return nil, err
	}

	db := &ChipsetDatabase{
		chipsets: chipsets,
		ouiMap:   make(map[string]*ChipsetInfo),
	}

	for i := range db.chipsets {
		for _, oui := range db.chipsets[i].OUIPrefix {
			db.ouiMap[strings.ToLower(oui)] = &db.chipsets[i]
		}
	}

	return db, nil
}

// loadChipsetsFromFile attempts to load chipsets from external YAML file.
// Looks in common locations: ./chipsets.yaml, /etc/luminetiq/chipsets.yaml.
func loadChipsetsFromFile() ([]ChipsetInfo, error) {
	paths := []string{
		"chipsets.yaml",
		"/etc/luminetiq/chipsets.yaml",
		"/usr/local/share/luminetiq/chipsets.yaml",
	}

	for _, path := range paths {
		//nolint:gosec // G304: paths are hardcoded trusted locations
		if data, err := os.ReadFile(path); err == nil {
			chipsets, parseErr := parseChipsetYAML(data)
			if parseErr == nil {
				log.Printf("Loaded chipset database from %s (%d entries)", path, len(chipsets))
				return chipsets, nil
			}
		}
	}

	return nil, os.ErrNotExist
}

// loadChipsetsFromEmbedded loads chipsets from the embedded YAML data.
func loadChipsetsFromEmbedded() []ChipsetInfo {
	chipsets, err := parseChipsetYAML(embeddedChipsetData)
	if err != nil {
		log.Printf("Warning: failed to parse embedded chipset data: %v", err)
		return nil
	}
	return chipsets
}

// parseChipsetYAML parses YAML data into ChipsetInfo slice.
func parseChipsetYAML(data []byte) ([]ChipsetInfo, error) {
	var yamlData chipsetYAML
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return nil, err
	}

	chipsets := make([]ChipsetInfo, len(yamlData.Chipsets))
	for i := range yamlData.Chipsets {
		entry := &yamlData.Chipsets[i]
		chipsets[i] = ChipsetInfo{
			Vendor:    entry.Vendor,
			Model:     entry.Model,
			Speed:     entry.Speed,
			Quality:   entry.Quality,
			HasTDR:    entry.HasTDR,
			HasDOM:    entry.HasDOM,
			OUIPrefix: entry.OUIPrefix,
			PCIVendor: entry.PCIVendor,
			PCIDevice: entry.PCIDevice,
			Keywords:  entry.Keywords,
		}
	}

	return chipsets, nil
}

// Count returns the number of chipsets in the database.
func (db *ChipsetDatabase) Count() int {
	return len(db.chipsets)
}

// GetAll returns all chipsets in the database.
func (db *ChipsetDatabase) GetAll() []ChipsetInfo {
	result := make([]ChipsetInfo, len(db.chipsets))
	copy(result, db.chipsets)
	return result
}

// IdentifyByMAC attempts to identify chipset by MAC address OUI.
func (db *ChipsetDatabase) IdentifyByMAC(mac string) *ChipsetInfo {
	// Normalize MAC - take first 3 octets (OUI)
	mac = strings.ToLower(strings.ReplaceAll(mac, "-", ":"))
	parts := strings.Split(mac, ":")
	if len(parts) < 3 {
		return nil
	}
	oui := strings.Join(parts[:3], ":")

	return db.ouiMap[oui]
}

// IdentifyByKeyword attempts to identify chipset by driver/device keywords.
func (db *ChipsetDatabase) IdentifyByKeyword(text string) *ChipsetInfo {
	text = strings.ToLower(text)

	for i := range db.chipsets {
		for _, keyword := range db.chipsets[i].Keywords {
			if strings.Contains(text, strings.ToLower(keyword)) {
				return &db.chipsets[i]
			}
		}
	}

	return nil
}

// IdentifyByPCI attempts to identify chipset by PCI vendor and device IDs.
func (db *ChipsetDatabase) IdentifyByPCI(vendor, device string) *ChipsetInfo {
	vendor = strings.ToLower(vendor)
	device = strings.ToLower(device)

	for i := range db.chipsets {
		if strings.EqualFold(db.chipsets[i].PCIVendor, vendor) {
			for _, pciDev := range db.chipsets[i].PCIDevice {
				if strings.EqualFold(pciDev, device) {
					return &db.chipsets[i]
				}
			}
		}
	}

	return nil
}

// IdentifyByInterface attempts to identify chipset using multiple methods.
func (db *ChipsetDatabase) IdentifyByInterface(name, mac string) *ChipsetInfo {
	// Try MAC OUI first
	if info := db.IdentifyByMAC(mac); info != nil {
		return info
	}

	// Try platform-specific identification (PCI IDs, sysfs, etc.)
	if info := db.identifyByPlatform(name); info != nil {
		return info
	}

	// Try interface name patterns
	return db.IdentifyByKeyword(name)
}
