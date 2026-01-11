//go:build linux

package discovery

// bluetooth_linux.go implements Bluetooth scanning on Linux using bluetoothctl
// and hcitool for device discovery.
//
// For BLE scanning, we use hcitool lescan with btmon for capturing advertisements.
// Full BLE scanning may require root privileges.

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Linux scanning constants.
const (
	linuxBTScanTimeoutSecs = 30
)

// scanPlatform performs Bluetooth scanning on Linux.
func (s *BluetoothScanner) scanPlatform(ctx context.Context, adapter string, config *BluetoothScanConfig) ([]BluetoothDevice, error) {
	ctx, cancel := context.WithTimeout(ctx, linuxBTScanTimeoutSecs*time.Second)
	defer cancel()

	var devices []BluetoothDevice

	// Get paired/connected devices via bluetoothctl
	btctlDevices, err := s.scanBluetoothctl(ctx)
	if err == nil {
		devices = append(devices, btctlDevices...)
	}

	// Try hcitool for classic discovery if enabled
	if config.IncludeClassic {
		hciDevices, err := s.scanHCITool(ctx, adapter, config)
		if err == nil {
			devices = mergeBluetoothDevices(devices, hciDevices)
		}
	}

	// Try BLE scanning if enabled
	if config.IncludeBLE {
		bleDevices, err := s.scanBLE(ctx, adapter, config)
		if err == nil {
			devices = mergeBluetoothDevices(devices, bleDevices)
		}
	}

	return devices, nil
}

// scanBluetoothctl uses bluetoothctl to get paired/connected devices.
func (s *BluetoothScanner) scanBluetoothctl(ctx context.Context) ([]BluetoothDevice, error) {
	// Get list of devices
	cmd := exec.CommandContext(ctx, "bluetoothctl", "devices")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bluetoothctl devices failed: %w", err)
	}

	var devices []BluetoothDevice
	now := time.Now()

	// Parse output: "Device AA:BB:CC:DD:EE:FF DeviceName"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Device ") {
			continue
		}

		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 2 {
			continue
		}

		address := parts[1]
		name := ""
		if len(parts) > 2 {
			name = parts[2]
		}

		// Get detailed info for this device
		dev := BluetoothDevice{
			Address:     address,
			Name:        name,
			Type:        BluetoothTypeClassic,
			DeviceClass: BluetoothClassUncategorized,
			FirstSeen:   now,
			LastSeen:    now,
			Metadata:    make(map[string]any),
		}

		// Get device info
		info, err := s.getBluetoothctlDeviceInfo(ctx, address)
		if err == nil {
			dev = mergeDeviceInfo(dev, info)
		}

		devices = append(devices, dev)
	}

	return devices, nil
}

// getBluetoothctlDeviceInfo gets detailed info for a specific device.
func (s *BluetoothScanner) getBluetoothctlDeviceInfo(ctx context.Context, address string) (BluetoothDevice, error) {
	cmd := exec.CommandContext(ctx, "bluetoothctl", "info", address)
	output, err := cmd.Output()
	if err != nil {
		return BluetoothDevice{}, err
	}

	return parseBluetoothctlInfo(string(output), address)
}

func parseBluetoothctlInfo(output, address string) (BluetoothDevice, error) {
	dev := BluetoothDevice{
		Address:  address,
		Metadata: make(map[string]any),
	}
	now := time.Now()

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Name:") {
			dev.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		} else if strings.HasPrefix(line, "Alias:") {
			dev.Alias = strings.TrimSpace(strings.TrimPrefix(line, "Alias:"))
		} else if strings.HasPrefix(line, "Class:") {
			// Parse hex class of device
			codStr := strings.TrimSpace(strings.TrimPrefix(line, "Class:"))
			codStr = strings.TrimPrefix(codStr, "0x")
			if cod, err := strconv.ParseUint(codStr, 16, 32); err == nil {
				dev.ClassOfDev = uint32(cod)
				dev.DeviceClass = ClassOfDeviceToClass(dev.ClassOfDev)
			}
		} else if strings.HasPrefix(line, "Paired:") {
			dev.IsPaired = strings.Contains(line, "yes")
		} else if strings.HasPrefix(line, "Trusted:") {
			dev.IsTrusted = strings.Contains(line, "yes")
		} else if strings.HasPrefix(line, "Blocked:") {
			dev.IsBlocked = strings.Contains(line, "yes")
		} else if strings.HasPrefix(line, "Connected:") {
			dev.IsConnected = strings.Contains(line, "yes")
		} else if strings.HasPrefix(line, "RSSI:") {
			rssiStr := strings.TrimSpace(strings.TrimPrefix(line, "RSSI:"))
			if rssi, err := strconv.Atoi(rssiStr); err == nil {
				dev.RSSI = rssi
			}
		} else if strings.HasPrefix(line, "TxPower:") {
			txStr := strings.TrimSpace(strings.TrimPrefix(line, "TxPower:"))
			if tx, err := strconv.Atoi(txStr); err == nil {
				dev.TxPower = tx
			}
		} else if strings.HasPrefix(line, "UUID:") {
			// Parse service UUID
			parts := strings.SplitN(line, "(", 2)
			if len(parts) > 1 {
				uuid := strings.TrimSuffix(strings.TrimSpace(parts[1]), ")")
				dev.ServiceUUIDs = append(dev.ServiceUUIDs, uuid)
			}
		} else if strings.HasPrefix(line, "ManufacturerData Key:") {
			// Parse manufacturer ID
			keyStr := strings.TrimSpace(strings.TrimPrefix(line, "ManufacturerData Key:"))
			keyStr = strings.TrimPrefix(keyStr, "0x")
			if key, err := strconv.ParseUint(keyStr, 16, 16); err == nil {
				dev.ManufacturerID = uint16(key)
			}
		}
	}

	dev.FirstSeen = now
	dev.LastSeen = now

	// Determine type based on available info
	if len(dev.ServiceUUIDs) > 0 || dev.TxPower != 0 {
		if dev.ClassOfDev != 0 {
			dev.Type = BluetoothTypeDual
		} else {
			dev.Type = BluetoothTypeBLE
		}
	}

	return dev, nil
}

// scanHCITool uses hcitool for classic Bluetooth discovery.
func (s *BluetoothScanner) scanHCITool(ctx context.Context, adapter string, config *BluetoothScanConfig) ([]BluetoothDevice, error) {
	// hcitool inq for inquiry scan
	args := []string{"inq"}
	if adapter != "" {
		args = []string{"-i", adapter, "inq"}
	}

	cmd := exec.CommandContext(ctx, "hcitool", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("hcitool inq failed: %w", err)
	}

	return parseHCIToolOutput(string(output))
}

func parseHCIToolOutput(output string) ([]BluetoothDevice, error) {
	var devices []BluetoothDevice
	now := time.Now()

	// hcitool inq output format:
	// Inquiring ...
	// AA:BB:CC:DD:EE:FF       clock offset: 0x1234    class: 0x240404

	addrRegex := regexp.MustCompile(`([0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2})`)
	classRegex := regexp.MustCompile(`class:\s*0x([0-9A-Fa-f]+)`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		addrMatch := addrRegex.FindStringSubmatch(line)
		if len(addrMatch) < 2 {
			continue
		}

		dev := BluetoothDevice{
			Address:     addrMatch[1],
			Type:        BluetoothTypeClassic,
			DeviceClass: BluetoothClassUncategorized,
			FirstSeen:   now,
			LastSeen:    now,
			Metadata:    make(map[string]any),
		}

		// Parse class of device
		if classMatch := classRegex.FindStringSubmatch(line); len(classMatch) > 1 {
			if cod, err := strconv.ParseUint(classMatch[1], 16, 32); err == nil {
				dev.ClassOfDev = uint32(cod)
				dev.DeviceClass = ClassOfDeviceToClass(dev.ClassOfDev)
			}
		}

		devices = append(devices, dev)
	}

	return devices, nil
}

// scanBLE uses hcitool lescan for BLE discovery.
func (s *BluetoothScanner) scanBLE(ctx context.Context, adapter string, config *BluetoothScanConfig) ([]BluetoothDevice, error) {
	// hcitool lescan requires root, timeout after configured duration
	scanDuration := time.Duration(config.ScanDurationSec) * time.Second
	if scanDuration == 0 {
		scanDuration = 5 * time.Second
	}

	scanCtx, cancel := context.WithTimeout(ctx, scanDuration)
	defer cancel()

	args := []string{"lescan", "--duplicates"}
	if adapter != "" {
		args = []string{"-i", adapter, "lescan", "--duplicates"}
	}

	cmd := exec.CommandContext(scanCtx, "hcitool", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("hcitool lescan failed: %w", err)
	}

	// Collect devices while scanning
	deviceMap := make(map[string]*BluetoothDevice)
	scanner := bufio.NewScanner(stdout)
	now := time.Now()

	// hcitool lescan output format:
	// LE Scan ...
	// AA:BB:CC:DD:EE:FF DeviceName
	// AA:BB:CC:DD:EE:FF (unknown)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "LE Scan") {
			continue
		}

		parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
		if len(parts) < 1 {
			continue
		}

		address := parts[0]
		if !isValidMACAddress(address) {
			continue
		}

		name := ""
		if len(parts) > 1 {
			name = parts[1]
			if name == "(unknown)" {
				name = ""
			}
		}

		if existing, ok := deviceMap[address]; ok {
			// Update with better name if we got one
			if name != "" && existing.Name == "" {
				existing.Name = name
			}
			existing.LastSeen = time.Now()
		} else {
			deviceMap[address] = &BluetoothDevice{
				Address:       address,
				Name:          name,
				Type:          BluetoothTypeBLE,
				DeviceClass:   BluetoothClassUncategorized,
				IsConnectable: true,
				FirstSeen:     now,
				LastSeen:      now,
				Metadata:      make(map[string]any),
			}
		}
	}

	// Kill the scan (it runs until killed or timeout)
	_ = cmd.Process.Kill()
	_ = cmd.Wait()

	// Convert map to slice
	devices := make([]BluetoothDevice, 0, len(deviceMap))
	for _, dev := range deviceMap {
		devices = append(devices, *dev)
	}

	return devices, nil
}

// isValidMACAddress checks if a string looks like a MAC address.
func isValidMACAddress(s string) bool {
	// Simple check: 6 hex pairs separated by colons
	parts := strings.Split(s, ":")
	if len(parts) != 6 {
		return false
	}
	for _, part := range parts {
		if len(part) != 2 {
			return false
		}
		if _, err := strconv.ParseUint(part, 16, 8); err != nil {
			return false
		}
	}
	return true
}

// mergeDeviceInfo merges detailed info into a device.
func mergeDeviceInfo(base, info BluetoothDevice) BluetoothDevice {
	if info.Name != "" {
		base.Name = info.Name
	}
	if info.Alias != "" {
		base.Alias = info.Alias
	}
	if info.ClassOfDev != 0 {
		base.ClassOfDev = info.ClassOfDev
		base.DeviceClass = info.DeviceClass
	}
	if info.RSSI != 0 {
		base.RSSI = info.RSSI
	}
	if info.TxPower != 0 {
		base.TxPower = info.TxPower
	}
	if len(info.ServiceUUIDs) > 0 {
		base.ServiceUUIDs = info.ServiceUUIDs
	}
	if info.ManufacturerID != 0 {
		base.ManufacturerID = info.ManufacturerID
	}
	base.IsPaired = info.IsPaired
	base.IsTrusted = info.IsTrusted
	base.IsBlocked = info.IsBlocked
	base.IsConnected = info.IsConnected
	base.Type = info.Type

	return base
}

// mergeBluetoothDevices merges two device lists.
func mergeBluetoothDevices(primary, secondary []BluetoothDevice) []BluetoothDevice {
	seen := make(map[string]int) // address -> index in result

	for i, dev := range primary {
		if dev.Address != "" {
			seen[normalizeMAC(dev.Address)] = i
		}
	}

	for _, dev := range secondary {
		if dev.Address == "" {
			continue
		}
		normalized := normalizeMAC(dev.Address)
		if idx, exists := seen[normalized]; exists {
			// Merge additional info
			primary[idx] = mergeDeviceInfo(primary[idx], dev)
		} else {
			primary = append(primary, dev)
			seen[normalized] = len(primary) - 1
		}
	}

	return primary
}
