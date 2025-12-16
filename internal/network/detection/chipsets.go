// Package detection provides intelligent network interface auto-detection.
// Chipset database module contains a curated list of network interface chipsets
// with quality ratings, capability flags (TDR, DOM), and MAC OUI prefixes.
package detection

import (
	"strings"
)

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
func NewChipsetDatabase() *ChipsetDatabase {
	db := &ChipsetDatabase{
		chipsets: getChipsetData(),
		ouiMap:   make(map[string]*ChipsetInfo),
	}

	// Build OUI lookup map
	for i := range db.chipsets {
		for _, oui := range db.chipsets[i].OUIPrefix {
			db.ouiMap[strings.ToLower(oui)] = &db.chipsets[i]
		}
	}

	return db
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
	if info := db.IdentifyByKeyword(name); info != nil {
		return info
	}

	return nil
}

// getChipsetData returns the curated chipset database.
// Quality ratings based on: driver stability, feature completeness, community support.
func getChipsetData() []ChipsetInfo {
	return []ChipsetInfo{
		// Intel 1G Ethernet - Enterprise grade, excellent TDR support
		{
			Vendor:    "Intel",
			Model:     "I210",
			Speed:     "1G",
			Quality:   95,
			HasTDR:    true,
			HasDOM:    false,
			OUIPrefix: []string{"00:1b:21", "a0:36:9f", "68:05:ca"},
			PCIVendor: "8086",
			PCIDevice: []string{"1533", "1536", "1537", "1538"},
			Keywords:  []string{"i210", "igb"},
		},
		{
			Vendor:    "Intel",
			Model:     "I211",
			Speed:     "1G",
			Quality:   90,
			HasTDR:    true,
			HasDOM:    false,
			OUIPrefix: []string{"00:1b:21", "a0:36:9f"},
			PCIVendor: "8086",
			PCIDevice: []string{"1539"},
			Keywords:  []string{"i211"},
		},
		{
			Vendor:    "Intel",
			Model:     "I350",
			Speed:     "1G",
			Quality:   95,
			HasTDR:    true,
			HasDOM:    false,
			OUIPrefix: []string{"00:1b:21", "a0:36:9f"},
			PCIVendor: "8086",
			PCIDevice: []string{"1521", "1522", "1523", "1524"},
			Keywords:  []string{"i350"},
		},

		// Intel 2.5G Ethernet - Modern desktop/NUC grade
		{
			Vendor:    "Intel",
			Model:     "I225-V",
			Speed:     "2.5G",
			Quality:   85,
			HasTDR:    true,
			HasDOM:    false,
			OUIPrefix: []string{"00:1b:21", "a0:36:9f", "04:42:1a"},
			PCIVendor: "8086",
			PCIDevice: []string{"15f2", "15f3"},
			Keywords:  []string{"i225", "igc"},
		},
		{
			Vendor:    "Intel",
			Model:     "I226-V",
			Speed:     "2.5G",
			Quality:   88,
			HasTDR:    true,
			HasDOM:    false,
			OUIPrefix: []string{"00:1b:21", "a0:36:9f"},
			PCIVendor: "8086",
			PCIDevice: []string{"125b", "125c"},
			Keywords:  []string{"i226"},
		},

		// Intel 10G Ethernet - Server/Workstation grade
		{
			Vendor:    "Intel",
			Model:     "X540",
			Speed:     "10G",
			Quality:   92,
			HasTDR:    true,
			HasDOM:    true,
			OUIPrefix: []string{"00:1b:21", "90:e2:ba"},
			PCIVendor: "8086",
			PCIDevice: []string{"1528", "1529"},
			Keywords:  []string{"x540", "ixgbe"},
		},
		{
			Vendor:    "Intel",
			Model:     "X550",
			Speed:     "10G",
			Quality:   93,
			HasTDR:    true,
			HasDOM:    true,
			OUIPrefix: []string{"00:1b:21", "90:e2:ba", "a0:36:9f"},
			PCIVendor: "8086",
			PCIDevice: []string{"1563", "15ab", "15ad"},
			Keywords:  []string{"x550"},
		},
		{
			Vendor:    "Intel",
			Model:     "X710",
			Speed:     "10G",
			Quality:   95,
			HasTDR:    false,
			HasDOM:    true,
			OUIPrefix: []string{"00:1b:21", "3c:fd:fe", "68:05:ca"},
			PCIVendor: "8086",
			PCIDevice: []string{"1572", "1574", "1580", "1581"},
			Keywords:  []string{"x710", "i40e"},
		},

		// Intel 25G/40G/100G - Data Center grade
		{
			Vendor:    "Intel",
			Model:     "XXV710",
			Speed:     "25G",
			Quality:   94,
			HasTDR:    false,
			HasDOM:    true,
			OUIPrefix: []string{"00:1b:21", "3c:fd:fe"},
			PCIVendor: "8086",
			PCIDevice: []string{"158a", "158b"},
			Keywords:  []string{"xxv710"},
		},
		{
			Vendor:    "Intel",
			Model:     "XL710",
			Speed:     "40G",
			Quality:   94,
			HasTDR:    false,
			HasDOM:    true,
			OUIPrefix: []string{"00:1b:21", "3c:fd:fe"},
			PCIVendor: "8086",
			PCIDevice: []string{"1583", "1584", "1585"},
			Keywords:  []string{"xl710"},
		},
		{
			Vendor:    "Intel",
			Model:     "E810",
			Speed:     "100G",
			Quality:   96,
			HasTDR:    false,
			HasDOM:    true,
			OUIPrefix: []string{"00:1b:21", "b4:96:91"},
			PCIVendor: "8086",
			PCIDevice: []string{"1591", "1592", "1593"},
			Keywords:  []string{"e810", "ice"},
		},

		// Broadcom 1G Ethernet - Server grade
		{
			Vendor:    "Broadcom",
			Model:     "BCM5720",
			Speed:     "1G",
			Quality:   85,
			HasTDR:    true,
			HasDOM:    false,
			OUIPrefix: []string{"14:fe:b5", "d4:ae:52"},
			PCIVendor: "14e4",
			PCIDevice: []string{"165f"},
			Keywords:  []string{"bcm5720", "tg3"},
		},
		{
			Vendor:    "Broadcom",
			Model:     "BCM5719",
			Speed:     "1G",
			Quality:   85,
			HasTDR:    true,
			HasDOM:    false,
			OUIPrefix: []string{"14:fe:b5"},
			PCIVendor: "14e4",
			PCIDevice: []string{"1657"},
			Keywords:  []string{"bcm5719"},
		},

		// Broadcom 10G/25G - Server grade
		{
			Vendor:    "Broadcom",
			Model:     "BCM57810",
			Speed:     "10G",
			Quality:   82,
			HasTDR:    false,
			HasDOM:    true,
			OUIPrefix: []string{"00:10:18"},
			PCIVendor: "14e4",
			PCIDevice: []string{"168e"},
			Keywords:  []string{"bcm57810", "bnx2x"},
		},

		// Realtek 1G - Consumer grade
		{
			Vendor:    "Realtek",
			Model:     "RTL8111",
			Speed:     "1G",
			Quality:   60,
			HasTDR:    false,
			HasDOM:    false,
			OUIPrefix: []string{"00:e0:4c", "52:54:00", "00:14:2a"},
			PCIVendor: "10ec",
			PCIDevice: []string{"8168", "8167"},
			Keywords:  []string{"rtl8111", "r8169", "r8168"},
		},
		{
			Vendor:    "Realtek",
			Model:     "RTL8169",
			Speed:     "1G",
			Quality:   55,
			HasTDR:    false,
			HasDOM:    false,
			OUIPrefix: []string{"00:e0:4c"},
			PCIVendor: "10ec",
			PCIDevice: []string{"8169"},
			Keywords:  []string{"rtl8169"},
		},

		// Realtek 2.5G - Consumer/Prosumer grade
		{
			Vendor:    "Realtek",
			Model:     "RTL8125",
			Speed:     "2.5G",
			Quality:   65,
			HasTDR:    false,
			HasDOM:    false,
			OUIPrefix: []string{"00:e0:4c"},
			PCIVendor: "10ec",
			PCIDevice: []string{"8125", "3000"},
			Keywords:  []string{"rtl8125", "r8125"},
		},

		// Realtek 5G - Newer consumer
		{
			Vendor:    "Realtek",
			Model:     "RTL8126",
			Speed:     "5G",
			Quality:   68,
			HasTDR:    false,
			HasDOM:    false,
			OUIPrefix: []string{"00:e0:4c"},
			PCIVendor: "10ec",
			PCIDevice: []string{"8126"},
			Keywords:  []string{"rtl8126"},
		},

		// Aquantia/Marvell Multi-Gig
		{
			Vendor:    "Aquantia",
			Model:     "AQC107",
			Speed:     "10G",
			Quality:   78,
			HasTDR:    true,
			HasDOM:    false,
			OUIPrefix: []string{"00:d0:c5", "3c:7c:3f"},
			PCIVendor: "1d6a",
			PCIDevice: []string{"07b1", "d107"},
			Keywords:  []string{"aqc107", "atlantic"},
		},
		{
			Vendor:    "Aquantia",
			Model:     "AQC113",
			Speed:     "10G",
			Quality:   80,
			HasTDR:    true,
			HasDOM:    false,
			OUIPrefix: []string{"00:d0:c5"},
			PCIVendor: "1d6a",
			PCIDevice: []string{"94c0", "00c0"},
			Keywords:  []string{"aqc113"},
		},

		// Mellanox/NVIDIA ConnectX - Data center grade
		{
			Vendor:    "Mellanox",
			Model:     "ConnectX-3",
			Speed:     "40G",
			Quality:   90,
			HasTDR:    false,
			HasDOM:    true,
			OUIPrefix: []string{"00:02:c9", "e4:1d:2d"},
			PCIVendor: "15b3",
			PCIDevice: []string{"1003", "1007"},
			Keywords:  []string{"connectx-3", "mlx4"},
		},
		{
			Vendor:    "Mellanox",
			Model:     "ConnectX-4",
			Speed:     "100G",
			Quality:   93,
			HasTDR:    false,
			HasDOM:    true,
			OUIPrefix: []string{"00:02:c9", "e4:1d:2d", "b8:ce:f6"},
			PCIVendor: "15b3",
			PCIDevice: []string{"1013", "1015", "1017"},
			Keywords:  []string{"connectx-4", "mlx5"},
		},
		{
			Vendor:    "Mellanox",
			Model:     "ConnectX-5",
			Speed:     "100G",
			Quality:   95,
			HasTDR:    false,
			HasDOM:    true,
			OUIPrefix: []string{"00:02:c9", "e4:1d:2d", "b8:ce:f6"},
			PCIVendor: "15b3",
			PCIDevice: []string{"1017", "1019"},
			Keywords:  []string{"connectx-5"},
		},
		{
			Vendor:    "Mellanox",
			Model:     "ConnectX-6",
			Speed:     "100G",
			Quality:   97,
			HasTDR:    false,
			HasDOM:    true,
			OUIPrefix: []string{"00:02:c9", "b8:ce:f6"},
			PCIVendor: "15b3",
			PCIDevice: []string{"101b", "101d"},
			Keywords:  []string{"connectx-6"},
		},

		// Intel WiFi - Modern WiFi 6/6E/7
		{
			Vendor:    "Intel",
			Model:     "AX200",
			Speed:     "WiFi6",
			Quality:   88,
			HasTDR:    false,
			HasDOM:    false,
			OUIPrefix: []string{"00:1b:21", "7c:b2:7d"},
			PCIVendor: "8086",
			PCIDevice: []string{"2723"},
			Keywords:  []string{"ax200", "iwlwifi"},
		},
		{
			Vendor:    "Intel",
			Model:     "AX210",
			Speed:     "WiFi6E",
			Quality:   90,
			HasTDR:    false,
			HasDOM:    false,
			OUIPrefix: []string{"00:1b:21", "7c:b2:7d"},
			PCIVendor: "8086",
			PCIDevice: []string{"2725"},
			Keywords:  []string{"ax210"},
		},
		{
			Vendor:    "Intel",
			Model:     "AX211",
			Speed:     "WiFi6E",
			Quality:   90,
			HasTDR:    false,
			HasDOM:    false,
			OUIPrefix: []string{"00:1b:21"},
			PCIVendor: "8086",
			PCIDevice: []string{"51f0", "51f1"},
			Keywords:  []string{"ax211"},
		},
		{
			Vendor:    "Intel",
			Model:     "BE200",
			Speed:     "WiFi7",
			Quality:   92,
			HasTDR:    false,
			HasDOM:    false,
			OUIPrefix: []string{"00:1b:21"},
			PCIVendor: "8086",
			PCIDevice: []string{"272e"},
			Keywords:  []string{"be200"},
		},

		// Qualcomm/Atheros WiFi
		{
			Vendor:    "Qualcomm",
			Model:     "Atheros QCA6174",
			Speed:     "WiFi5",
			Quality:   75,
			HasTDR:    false,
			HasDOM:    false,
			OUIPrefix: []string{"00:03:7f", "98:de:d0"},
			PCIVendor: "168c",
			PCIDevice: []string{"003e"},
			Keywords:  []string{"qca6174", "ath10k"},
		},

		// MediaTek WiFi
		{
			Vendor:    "MediaTek",
			Model:     "MT7921",
			Speed:     "WiFi6",
			Quality:   72,
			HasTDR:    false,
			HasDOM:    false,
			OUIPrefix: []string{"a8:93:4a"},
			PCIVendor: "14c3",
			PCIDevice: []string{"7961"},
			Keywords:  []string{"mt7921"},
		},
	}
}
