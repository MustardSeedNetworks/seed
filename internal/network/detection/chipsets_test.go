package detection_test

// Test suite for chipset database and identification.

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/network/detection"
)

func TestNewChipsetDatabase(t *testing.T) {
	db := detection.NewChipsetDatabase()
	if db == nil {
		t.Fatal("NewChipsetDatabase() returned nil")
	}

	if db.ChipsetsCount() == 0 {
		t.Error("chipsets should not be empty")
	}

	if db.OUIMapCount() == 0 {
		t.Error("ouiMap should not be empty")
	}
}

func TestIdentifyByMAC(t *testing.T) {
	db := detection.NewChipsetDatabase()

	tests := []struct {
		name       string
		mac        string
		wantVendor string
		wantNil    bool
	}{
		{
			name:       "Intel OUI",
			mac:        "00:1b:21:aa:bb:cc",
			wantVendor: "Intel",
		},
		{
			name:       "Realtek OUI",
			mac:        "00:e0:4c:11:22:33",
			wantVendor: "Realtek",
		},
		{
			name:       "Mellanox OUI",
			mac:        "00:02:c9:aa:bb:cc",
			wantVendor: "Mellanox",
		},
		{
			name:    "Unknown OUI",
			mac:     "ff:ff:ff:aa:bb:cc",
			wantNil: true,
		},
		{
			name:    "Invalid MAC",
			mac:     "invalid",
			wantNil: true,
		},
		{
			name:    "Short MAC",
			mac:     "00:1b",
			wantNil: true,
		},
		{
			name:       "MAC with dashes",
			mac:        "00-1b-21-aa-bb-cc",
			wantVendor: "Intel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := db.IdentifyByMAC(tt.mac)

			if tt.wantNil {
				if info != nil {
					t.Errorf("IdentifyByMAC(%q) = %v, want nil", tt.mac, info)
				}
				return
			}

			if info == nil {
				t.Fatalf("IdentifyByMAC(%q) = nil, want vendor %s", tt.mac, tt.wantVendor)
			}

			if info.Vendor != tt.wantVendor {
				t.Errorf("Vendor = %q, want %q", info.Vendor, tt.wantVendor)
			}
		})
	}
}

func TestIdentifyByKeyword(t *testing.T) {
	db := detection.NewChipsetDatabase()

	tests := []struct {
		name       string
		keyword    string
		wantVendor string
		wantModel  string
		wantNil    bool
	}{
		{
			name:       "igb driver",
			keyword:    "igb",
			wantVendor: "Intel",
		},
		{
			name:       "igc driver",
			keyword:    "igc",
			wantVendor: "Intel",
			wantModel:  "I225-V",
		},
		{
			name:       "r8169 driver",
			keyword:    "r8169",
			wantVendor: "Realtek",
		},
		{
			name:       "mlx5 driver",
			keyword:    "mlx5",
			wantVendor: "Mellanox",
		},
		{
			name:       "i40e driver",
			keyword:    "i40e",
			wantVendor: "Intel",
			wantModel:  "X710",
		},
		{
			name:    "unknown driver",
			keyword: "unknown_driver",
			wantNil: true,
		},
		{
			name:       "case insensitive",
			keyword:    "IGB",
			wantVendor: "Intel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := db.IdentifyByKeyword(tt.keyword)

			if tt.wantNil {
				if info != nil {
					t.Errorf("IdentifyByKeyword(%q) = %v, want nil", tt.keyword, info)
				}
				return
			}

			if info == nil {
				t.Fatalf("IdentifyByKeyword(%q) = nil, want vendor %s", tt.keyword, tt.wantVendor)
			}

			if info.Vendor != tt.wantVendor {
				t.Errorf("Vendor = %q, want %q", info.Vendor, tt.wantVendor)
			}

			if tt.wantModel != "" && info.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", info.Model, tt.wantModel)
			}
		})
	}
}

func TestChipsetData(t *testing.T) {
	db := detection.NewChipsetDatabase()
	chipsets := db.GetAll()

	if len(chipsets) == 0 {
		t.Fatal("GetAll() returned empty slice")
	}

	// Verify required fields are populated.
	for i, chip := range chipsets {
		if chip.Vendor == "" {
			t.Errorf("chipset[%d]: Vendor is empty", i)
		}
		if chip.Model == "" {
			t.Errorf("chipset[%d] (%s): Model is empty", i, chip.Vendor)
		}
		if chip.Speed == "" {
			t.Errorf("chipset[%d] (%s %s): Speed is empty", i, chip.Vendor, chip.Model)
		}
		if chip.Quality < 1 || chip.Quality > 100 {
			t.Errorf("chipset[%d] (%s %s): Quality %d out of range [1,100]",
				i, chip.Vendor, chip.Model, chip.Quality)
		}
	}

	// Verify known chipsets exist.
	expectedChipsets := []struct {
		vendor string
		model  string
	}{
		{"Intel", "I210"},
		{"Intel", "I225-V"},
		{"Intel", "X710"},
		{"Intel", "E810"},
		{"Realtek", "RTL8111"},
		{"Realtek", "RTL8125"},
		{"Broadcom", "BCM5720"},
		{"Mellanox", "ConnectX-5"},
		{"Aquantia", "AQC107"},
	}

	for _, expected := range expectedChipsets {
		found := false
		for _, chip := range chipsets {
			if chip.Vendor == expected.vendor && chip.Model == expected.model {
				found = true
				break
			}
		}
		if !found {
			t.Errorf(
				"Expected chipset %s %s not found in database",
				expected.vendor,
				expected.model,
			)
		}
	}
}

func TestChipsetTDRFlags(t *testing.T) {
	db := detection.NewChipsetDatabase()
	chipsets := db.GetAll()

	// Intel desktop NICs should have TDR.
	tdrExpected := map[string]bool{
		"I210":   true,
		"I211":   true,
		"I225-V": true,
		"I226-V": true,
		"X540":   true,
		"X550":   true,
	}

	for _, chip := range chipsets {
		if expected, ok := tdrExpected[chip.Model]; ok {
			if chip.HasTDR != expected {
				t.Errorf("%s %s: HasTDR = %v, want %v",
					chip.Vendor, chip.Model, chip.HasTDR, expected)
			}
		}
	}
}

func TestChipsetDOMFlags(t *testing.T) {
	db := detection.NewChipsetDatabase()
	chipsets := db.GetAll()

	// High-speed NICs with SFP+ should have DOM.
	domExpected := map[string]bool{
		"X540":       true,
		"X550":       true,
		"X710":       true,
		"XL710":      true,
		"E810":       true,
		"ConnectX-4": true,
		"ConnectX-5": true,
		"ConnectX-6": true,
	}

	for _, chip := range chipsets {
		if expected, ok := domExpected[chip.Model]; ok {
			if chip.HasDOM != expected {
				t.Errorf("%s %s: HasDOM = %v, want %v",
					chip.Vendor, chip.Model, chip.HasDOM, expected)
			}
		}
	}
}

func TestQualityRatings(t *testing.T) {
	db := detection.NewChipsetDatabase()
	chipsets := db.GetAll()

	// Intel enterprise NICs should have high quality.
	highQuality := []string{"I210", "I350", "E810"}
	for _, model := range highQuality {
		for _, chip := range chipsets {
			if chip.Model == model && chip.Quality < 90 {
				t.Errorf("%s %s: Quality %d should be >= 90",
					chip.Vendor, chip.Model, chip.Quality)
			}
		}
	}

	// Consumer Realtek should have lower quality.
	for _, chip := range chipsets {
		if chip.Vendor == "Realtek" && chip.Quality > 80 {
			t.Errorf("Realtek %s: Quality %d seems too high",
				chip.Model, chip.Quality)
		}
	}
}

func TestChipsetDatabaseCount(t *testing.T) {
	db := detection.NewChipsetDatabase()

	count := db.Count()
	if count == 0 {
		t.Error("Count() returned 0, expected > 0")
	}

	// Count should match GetAll length.
	allChipsets := db.GetAll()
	if count != len(allChipsets) {
		t.Errorf("Count() = %d, but GetAll() returned %d chipsets",
			count, len(allChipsets))
	}
}

func TestIdentifyByPCI(t *testing.T) {
	db := detection.NewChipsetDatabase()

	tests := []struct {
		name       string
		vendor     string
		device     string
		wantVendor string
		wantModel  string
		wantNil    bool
	}{
		{
			name:       "Intel I210",
			vendor:     "8086",
			device:     "1533",
			wantVendor: "Intel",
			wantModel:  "I210",
		},
		{
			name:       "Intel I225-V",
			vendor:     "8086",
			device:     "15f2",
			wantVendor: "Intel",
			wantModel:  "I225-V",
		},
		{
			name:       "Intel X710",
			vendor:     "8086",
			device:     "1572",
			wantVendor: "Intel",
			wantModel:  "X710",
		},
		{
			name:       "Realtek RTL8111",
			vendor:     "10ec",
			device:     "8168",
			wantVendor: "Realtek",
			wantModel:  "RTL8111",
		},
		{
			name:       "Mellanox ConnectX-5",
			vendor:     "15b3",
			device:     "1017",
			wantVendor: "Mellanox",
		},
		{
			name:       "case insensitive vendor",
			vendor:     "8086",
			device:     "1533",
			wantVendor: "Intel",
			wantModel:  "I210",
		},
		{
			name:       "case insensitive device",
			vendor:     "8086",
			device:     "15F2", // uppercase
			wantVendor: "Intel",
			wantModel:  "I225-V",
		},
		{
			name:    "unknown vendor",
			vendor:  "ffff",
			device:  "0000",
			wantNil: true,
		},
		{
			name:    "unknown device for known vendor",
			vendor:  "8086",
			device:  "0000",
			wantNil: true,
		},
		{
			name:    "empty vendor",
			vendor:  "",
			device:  "1533",
			wantNil: true,
		},
		{
			name:    "empty device",
			vendor:  "8086",
			device:  "",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := db.IdentifyByPCI(tt.vendor, tt.device)

			if tt.wantNil {
				if info != nil {
					t.Errorf("IdentifyByPCI(%q, %q) = %+v, want nil",
						tt.vendor, tt.device, info)
				}
				return
			}

			if info == nil {
				t.Fatalf("IdentifyByPCI(%q, %q) = nil, want %s %s",
					tt.vendor, tt.device, tt.wantVendor, tt.wantModel)
			}

			if info.Vendor != tt.wantVendor {
				t.Errorf("Vendor = %q, want %q", info.Vendor, tt.wantVendor)
			}

			if tt.wantModel != "" && info.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", info.Model, tt.wantModel)
			}
		})
	}
}

func TestNewChipsetDatabaseFromFile(t *testing.T) {
	// Test with existing chipsets.yaml file.
	db, err := detection.NewChipsetDatabaseFromFile("chipsets.yaml")
	if err != nil {
		t.Fatalf("NewChipsetDatabaseFromFile() error: %v", err)
	}

	if db == nil {
		t.Fatal("NewChipsetDatabaseFromFile() returned nil")
	}

	if db.Count() == 0 {
		t.Error("Database loaded from file has no chipsets")
	}

	// Test with non-existent file.
	_, err = detection.NewChipsetDatabaseFromFile("nonexistent_file.yaml")
	if err == nil {
		t.Error("NewChipsetDatabaseFromFile() should return error for missing file")
	}
}

func TestChipsetInfoFields(t *testing.T) {
	db := detection.NewChipsetDatabase()
	chipsets := db.GetAll()

	// Find a known chipset and verify all fields.
	var found *detection.ChipsetInfo
	for i := range chipsets {
		if chipsets[i].Vendor == "Intel" && chipsets[i].Model == "I225-V" {
			found = &chipsets[i]
			break
		}
	}

	if found == nil {
		t.Fatal("Could not find Intel I225-V in database")
	}

	// Verify all fields are populated correctly.
	if found.Speed != "2.5G" {
		t.Errorf("Speed = %q, want 2.5G", found.Speed)
	}

	if found.Quality < 80 || found.Quality > 100 {
		t.Errorf("Quality = %d, expected 80-100", found.Quality)
	}

	if !found.HasTDR {
		t.Error("HasTDR should be true for I225-V")
	}

	if found.HasDOM {
		t.Error("HasDOM should be false for I225-V")
	}

	if len(found.OUIPrefix) == 0 {
		t.Error("OUIPrefix should not be empty")
	}

	if found.PCIVendor != "8086" {
		t.Errorf("PCIVendor = %q, want 8086", found.PCIVendor)
	}

	if len(found.PCIDevice) == 0 {
		t.Error("PCIDevice should not be empty")
	}

	if len(found.Keywords) == 0 {
		t.Error("Keywords should not be empty")
	}
}

func TestIdentifyByInterfaceWithMAC(t *testing.T) {
	db := detection.NewChipsetDatabase()

	// Test with known Intel OUI.
	info := db.IdentifyByInterface("eth0", "00:1b:21:aa:bb:cc")
	if info == nil {
		t.Fatal("IdentifyByInterface() returned nil for known Intel OUI")
	}
	if info.Vendor != "Intel" {
		t.Errorf("Vendor = %q, want Intel", info.Vendor)
	}

	// Test with known Realtek OUI.
	info = db.IdentifyByInterface("eth1", "00:e0:4c:11:22:33")
	if info == nil {
		t.Fatal("IdentifyByInterface() returned nil for known Realtek OUI")
	}
	if info.Vendor != "Realtek" {
		t.Errorf("Vendor = %q, want Realtek", info.Vendor)
	}

	// Test with unknown MAC - should fall through to keyword matching.
	info = db.IdentifyByInterface("igb0", "ff:ff:ff:aa:bb:cc")
	// May or may not find via keyword, but should not panic.
	t.Logf("IdentifyByInterface with unknown MAC: %+v", info)
}
