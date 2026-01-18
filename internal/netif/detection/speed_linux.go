//go:build linux

// Package detection provides intelligent network interface auto-detection.
// Linux-specific speed detection module uses sysfs and the safchain/ethtool
// Go package for interface speed detection, PCI device identification, and driver info.
package detection

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/safchain/ethtool"
)

// getInterfaceSpeed returns the interface speed in bits per second.
func getInterfaceSpeed(name string) int64 {
	// Try sysfs first (most reliable)
	speedPath := filepath.Join("/sys/class/net", name, "speed")
	if data, err := os.ReadFile(speedPath); err == nil {
		speedStr := strings.TrimSpace(string(data))
		if speed, err := strconv.ParseInt(speedStr, 10, 64); err == nil && speed > 0 {
			// sysfs reports speed in Mb/s, convert to bits per second
			return speed * 1_000_000
		}
	}

	// Fallback to ethtool Go package
	e, err := ethtool.NewEthtool()
	if err != nil {
		return 0
	}
	defer e.Close()

	cmd, err := e.CmdGetMapped(name)
	if err != nil {
		return 0
	}

	if speed, ok := cmd["speed"]; ok && speed > 0 && speed != 0xFFFFFFFF {
		return int64(speed) * 1_000_000
	}

	return 0
}

// identifyByPlatform attempts platform-specific chipset identification.
func (db *ChipsetDatabase) identifyByPlatform(name string) *ChipsetInfo {
	// Try to read device info from sysfs
	devicePath := filepath.Join("/sys/class/net", name, "device")

	// Get PCI vendor ID
	vendorPath := filepath.Join(devicePath, "vendor")
	vendorData, err := os.ReadFile(vendorPath)
	if err != nil {
		return nil
	}
	vendor := strings.TrimSpace(strings.TrimPrefix(string(vendorData), "0x"))

	// Get PCI device ID
	deviceIDPath := filepath.Join(devicePath, "device")
	deviceData, err := os.ReadFile(deviceIDPath)
	if err != nil {
		return nil
	}
	deviceID := strings.TrimSpace(strings.TrimPrefix(string(deviceData), "0x"))

	// Match against database
	for i := range db.chipsets {
		if strings.EqualFold(db.chipsets[i].PCIVendor, vendor) {
			for _, pciDev := range db.chipsets[i].PCIDevice {
				if strings.EqualFold(pciDev, deviceID) {
					return &db.chipsets[i]
				}
			}
		}
	}

	// Try driver name via ethtool Go package
	e, err := ethtool.NewEthtool()
	if err == nil {
		defer e.Close()
		if info, err := e.DriverInfo(name); err == nil {
			return db.IdentifyByKeyword(info.Driver)
		}
	}

	// Fallback to sysfs driver symlink
	driverPath := filepath.Join(devicePath, "driver")
	if target, err := os.Readlink(driverPath); err == nil {
		driverName := filepath.Base(target)
		return db.IdentifyByKeyword(driverName)
	}

	return nil
}

// hasTDRCapability checks if the interface supports Time Domain Reflectometry.
func hasTDRCapability(name string) bool {
	e, err := ethtool.NewEthtool()
	if err != nil {
		return false
	}
	defer e.Close()

	// Check driver info to determine TDR support
	info, err := e.DriverInfo(name)
	if err != nil {
		return false
	}

	// Intel igb/igc and some Broadcom drivers support cable diagnostics
	supportedDrivers := []string{"igb", "igc", "e1000e", "tg3", "bnx2"}
	for _, drv := range supportedDrivers {
		if strings.Contains(strings.ToLower(info.Driver), drv) {
			return true
		}
	}

	return false
}

// hasDOMCapability checks if the interface supports Digital Optical Monitoring.
func hasDOMCapability(name string) bool {
	// Check for SFP/QSFP module presence via sysfs
	eepromPath := filepath.Join("/sys/class/net", name, "device", "sfp_eeprom")
	if _, err := os.Stat(eepromPath); err == nil {
		return true
	}

	// Alternative path for some drivers
	domPath := filepath.Join("/sys/class/net", name, "device", "dom")
	if _, err := os.Stat(domPath); err == nil {
		return true
	}

	// Check driver for known SFP-capable drivers via ethtool
	e, err := ethtool.NewEthtool()
	if err != nil {
		return false
	}
	defer e.Close()

	info, err := e.DriverInfo(name)
	if err != nil {
		return false
	}

	// Drivers known to support SFP+ modules
	sfpDrivers := []string{"ixgbe", "i40e", "ice", "mlx4", "mlx5"}
	for _, drv := range sfpDrivers {
		if strings.Contains(strings.ToLower(info.Driver), drv) {
			return true
		}
	}

	return false
}
