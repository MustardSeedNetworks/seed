//go:build darwin

package discovery

// bluetooth_darwin.go implements Bluetooth scanning on macOS using system_profiler
// and blueutil (if available) for basic device discovery.
//
// Note: Full BLE scanning on macOS requires CoreBluetooth framework access,
// which needs a signed app with appropriate entitlements. This implementation
// provides basic classic Bluetooth discovery via system tools.

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Darwin scanning constants.
const (
	darwinBTScanTimeoutSecs = 30
	systemProfilerPath      = "/usr/sbin/system_profiler"
)

// scanPlatform performs Bluetooth scanning on macOS.
func (s *BluetoothScanner) scanPlatform(
	ctx context.Context,
	_ string,
	_ *BluetoothScanConfig,
) ([]BluetoothDevice, error) {
	ctx, cancel := context.WithTimeout(ctx, darwinBTScanTimeoutSecs*time.Second)
	defer cancel()

	var devices []BluetoothDevice

	// Get paired/connected devices via system_profiler
	sysDevices, err := s.scanSystemProfiler(ctx)
	if err == nil {
		devices = append(devices, sysDevices...)
	}

	// Try blueutil for additional info (if installed via homebrew)
	blueDevices, err := s.scanBlueutil(ctx)
	if err == nil {
		devices = mergeBluetoothDevices(devices, blueDevices)
	}

	return devices, nil
}

// scanSystemProfiler uses system_profiler to get Bluetooth device info.
func (s *BluetoothScanner) scanSystemProfiler(ctx context.Context) ([]BluetoothDevice, error) {
	cmd := exec.CommandContext(ctx, systemProfilerPath, "SPBluetoothDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("system_profiler failed: %w", err)
	}

	return parseSystemProfilerOutput(output)
}

// systemProfilerBluetooth represents the JSON structure from system_profiler.
type systemProfilerBluetooth struct {
	SPBluetoothDataType []struct {
		ControllerProperties struct {
			Address string `json:"controller_address"`
			State   string `json:"controller_state"`
		} `json:"controller_properties"`
		DevicesConnected []map[string]struct {
			Address      string `json:"device_address"`
			MinorType    string `json:"device_minorType"`
			MajorType    string `json:"device_majorType"`
			Connected    string `json:"device_isconnected"`
			Paired       string `json:"device_ispaired"`
			RSSI         any    `json:"device_rssi"`
			Services     string `json:"device_services"`
			Manufacturer string `json:"device_manufacturer"`
		} `json:"device_connected,omitempty"`
		DevicesPaired []map[string]struct {
			Address      string `json:"device_address"`
			MinorType    string `json:"device_minorType"`
			MajorType    string `json:"device_majorType"`
			Connected    string `json:"device_isconnected"`
			Paired       string `json:"device_ispaired"`
			RSSI         any    `json:"device_rssi"`
			Services     string `json:"device_services"`
			Manufacturer string `json:"device_manufacturer"`
		} `json:"devices_not_connected,omitempty"`
	} `json:"SPBluetoothDataType"`
}

func parseSystemProfilerOutput(output []byte) ([]BluetoothDevice, error) {
	var data systemProfilerBluetooth
	if err := json.Unmarshal(output, &data); err != nil {
		// Fall back to text parsing if JSON fails
		return parseSystemProfilerText(string(output))
	}

	var devices []BluetoothDevice
	now := time.Now()

	for _, bt := range data.SPBluetoothDataType {
		// Parse connected devices
		for _, devMap := range bt.DevicesConnected {
			for name, info := range devMap {
				dev := BluetoothDevice{
					Name:        name,
					Address:     info.Address,
					IsConnected: info.Connected == "Yes" || info.Connected == "attrib_Yes",
					IsPaired:    info.Paired == "Yes" || info.Paired == "attrib_Yes",
					Type:        BluetoothTypeClassic,
					DeviceClass: mapMajorTypeToClass(info.MajorType),
					FirstSeen:   now,
					LastSeen:    now,
					Metadata:    make(map[string]any),
				}

				if info.Manufacturer != "" {
					dev.Vendor = info.Manufacturer
				}

				// Parse RSSI if available
				switch v := info.RSSI.(type) {
				case float64:
					dev.RSSI = int(v)
				case string:
					if rssi, err := strconv.Atoi(v); err == nil {
						dev.RSSI = rssi
					}
				}

				devices = append(devices, dev)
			}
		}

		// Parse paired (not connected) devices
		for _, devMap := range bt.DevicesPaired {
			for name, info := range devMap {
				dev := BluetoothDevice{
					Name:        name,
					Address:     info.Address,
					IsConnected: false,
					IsPaired:    true,
					Type:        BluetoothTypeClassic,
					DeviceClass: mapMajorTypeToClass(info.MajorType),
					FirstSeen:   now,
					LastSeen:    now,
					Metadata:    make(map[string]any),
				}

				if info.Manufacturer != "" {
					dev.Vendor = info.Manufacturer
				}

				devices = append(devices, dev)
			}
		}
	}

	return devices, nil
}

// parseSystemProfilerText parses text output when JSON parsing fails.
func parseSystemProfilerText(output string) ([]BluetoothDevice, error) {
	var devices []BluetoothDevice
	var currentDevice *BluetoothDevice
	now := time.Now()

	addrRegex := regexp.MustCompile(`Address:\s*([0-9A-Fa-f:-]+)`)
	rssiRegex := regexp.MustCompile(`RSSI:\s*(-?\d+)`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// New device entry (indented device name followed by colon)
		if strings.HasSuffix(line, ":") && !strings.Contains(line, "Bluetooth") {
			if currentDevice != nil && currentDevice.Address != "" {
				devices = append(devices, *currentDevice)
			}
			name := strings.TrimSuffix(line, ":")
			currentDevice = &BluetoothDevice{
				Name:        name,
				Type:        BluetoothTypeClassic,
				DeviceClass: BluetoothClassUncategorized,
				FirstSeen:   now,
				LastSeen:    now,
				Metadata:    make(map[string]any),
			}
			continue
		}

		if currentDevice == nil {
			continue
		}

		// Parse address
		if matches := addrRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentDevice.Address = matches[1]
		}

		// Parse RSSI
		if matches := rssiRegex.FindStringSubmatch(line); len(matches) > 1 {
			if rssi, err := strconv.Atoi(matches[1]); err == nil {
				currentDevice.RSSI = rssi
			}
		}

		// Parse connected status
		if strings.Contains(line, "Connected: Yes") {
			currentDevice.IsConnected = true
		}

		// Parse paired status
		if strings.Contains(line, "Paired: Yes") {
			currentDevice.IsPaired = true
		}

		// Parse major type
		if strings.Contains(line, "Major Type:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				currentDevice.DeviceClass = mapMajorTypeToClass(strings.TrimSpace(parts[1]))
			}
		}
	}

	// Don't forget the last device
	if currentDevice != nil && currentDevice.Address != "" {
		devices = append(devices, *currentDevice)
	}

	return devices, nil
}

// scanBlueutil uses blueutil (homebrew) for additional device info.
func (s *BluetoothScanner) scanBlueutil(ctx context.Context) ([]BluetoothDevice, error) {
	// Check if blueutil is available
	cmd := exec.CommandContext(ctx, "which", "blueutil")
	if err := cmd.Run(); err != nil {
		return nil, errors.New("blueutil not installed")
	}

	// Get paired devices
	cmd = exec.CommandContext(ctx, "blueutil", "--paired")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("blueutil --paired failed: %w", err)
	}

	return parseBleutilOutput(string(output))
}

func parseBleutilOutput(output string) ([]BluetoothDevice, error) {
	var devices []BluetoothDevice
	now := time.Now()

	// blueutil output format: address, name, ...
	lines := strings.SplitSeq(output, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ",", 2)
		if len(parts) < 1 {
			continue
		}

		address := strings.TrimSpace(parts[0])
		name := ""
		if len(parts) > 1 {
			name = strings.TrimSpace(parts[1])
		}

		if address != "" {
			dev := BluetoothDevice{
				Address:     address,
				Name:        name,
				Type:        BluetoothTypeClassic,
				DeviceClass: BluetoothClassUncategorized,
				IsPaired:    true,
				FirstSeen:   now,
				LastSeen:    now,
				Metadata:    make(map[string]any),
			}
			devices = append(devices, dev)
		}
	}

	return devices, nil
}

// mapMajorTypeToClass maps macOS major type strings to our class enum.
func mapMajorTypeToClass(majorType string) BluetoothDeviceClass {
	majorType = strings.ToLower(majorType)
	switch {
	case strings.Contains(majorType, "computer"):
		return BluetoothClassComputer
	case strings.Contains(majorType, "phone"):
		return BluetoothClassPhone
	case strings.Contains(majorType, "audio"):
		return BluetoothClassAudioVideo
	case strings.Contains(majorType, "peripheral"):
		return BluetoothClassPeripheral
	case strings.Contains(majorType, "imaging"):
		return BluetoothClassImaging
	case strings.Contains(majorType, "wearable"):
		return BluetoothClassWearable
	case strings.Contains(majorType, "health"):
		return BluetoothClassHealth
	default:
		return BluetoothClassUncategorized
	}
}

// mergeBluetoothDevices merges two device lists, preferring entries with more info.
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
			if primary[idx].Name == "" && dev.Name != "" {
				primary[idx].Name = dev.Name
			}
			if primary[idx].Vendor == "" && dev.Vendor != "" {
				primary[idx].Vendor = dev.Vendor
			}
		} else {
			primary = append(primary, dev)
			seen[normalized] = len(primary) - 1
		}
	}

	return primary
}
