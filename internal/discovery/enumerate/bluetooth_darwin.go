//go:build darwin

package enumerate

// bluetooth_darwin.go implements Bluetooth scanning on macOS. Paired and
// connected devices come from system_profiler and blueutil (if installed).
// Advertising LE devices come from CoreBluetooth, which answers only a signed
// bundle the user granted Bluetooth to in their login session — never the root
// daemon — so the Seed Wi-Fi Helper runs that scan and returns what it heard.

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

	"github.com/MustardSeedNetworks/seed/internal/platform/procrun"
)

// Darwin scanning constants.
const (
	darwinBTScanTimeoutSecs = 30
	systemProfilerPath      = "/usr/sbin/system_profiler"
	// keyValueSplitParts is the number of parts for key:value splits.
	keyValueSplitParts = 2
)

// errNoBLEHelper means no Seed Wi-Fi Helper is connected, so no LE scan can
// run. It is an error rather than an empty result: an operator must start the
// helper, not conclude the room is quiet.
var errNoBLEHelper = errors.New(
	"bluetooth LE scan needs the Seed Wi-Fi Helper running in a logged-in user session")

// scanPlatform performs Bluetooth scanning on macOS.
func (s *BluetoothScanner) scanPlatform(
	ctx context.Context,
	_ string,
	config *BluetoothScanConfig,
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

	if !config.IncludeBLE {
		return devices, nil
	}

	// CoreBluetooth hides addresses, so a paired LE device heard advertising
	// cannot be matched to its system_profiler entry and is listed under each.
	le, err := s.scanBLE(config)
	if err != nil {
		return nil, err
	}
	return append(devices, le...), nil
}

// scanBLE has the helper listen for LE advertisements. A missing grant or a
// powered-off radio comes back as the helper's error.
func (s *BluetoothScanner) scanBLE(config *BluetoothScanConfig) ([]BluetoothDevice, error) {
	s.mu.RLock()
	helper := s.bleHelper
	s.mu.RUnlock()

	if helper == nil {
		return nil, errNoBLEHelper
	}

	found, err := helper.ScanBLE(bleScanDuration(config))
	if err != nil {
		return nil, fmt.Errorf("bluetooth LE scan via helper: %w", err)
	}
	return devicesFromPeripherals(found, time.Now()), nil
}

// scanSystemProfiler uses system_profiler to get Bluetooth device info.
func (s *BluetoothScanner) scanSystemProfiler(ctx context.Context) ([]BluetoothDevice, error) {
	output, err := procrun.Output(exec.CommandContext(ctx, systemProfilerPath, "SPBluetoothDataType", "-json"))
	if err != nil {
		return nil, fmt.Errorf("system_profiler failed: %w", err)
	}

	return parseSystemProfilerOutput(output)
}

// json-wire:external-adapter — these structs json.Unmarshal macOS
// `system_profiler SPBluetoothDataType -json` output verbatim (Apple's snake_case
// contract), then map to the camelCase BluetoothDevice domain type via
// createBluetoothDevice before any API emission. The snake_case tags here are
// the OS tool's format, never our wire — so this file is exempt from the
// json-casing gate by structural boundary (ADR-0010), not a per-key allow-list.

// btDeviceInfo represents a Bluetooth device from system_profiler JSON.
type btDeviceInfo struct {
	Address      string `json:"device_address"`
	MinorType    string `json:"device_minorType"`
	MajorType    string `json:"device_majorType"`
	Connected    string `json:"device_isconnected"`
	Paired       string `json:"device_ispaired"`
	RSSI         any    `json:"device_rssi"`
	Services     string `json:"device_services"`
	Manufacturer string `json:"device_manufacturer"`
}

// systemProfilerBluetooth represents the JSON structure from system_profiler.
type systemProfilerBluetooth struct {
	SPBluetoothDataType []struct {
		ControllerProperties struct {
			Address string `json:"controller_address"`
			State   string `json:"controller_state"`
		} `json:"controller_properties"`
		DevicesConnected []map[string]*btDeviceInfo `json:"device_connected,omitempty"`
		DevicesPaired    []map[string]*btDeviceInfo `json:"devices_not_connected,omitempty"`
	} `json:"SPBluetoothDataType"`
}

// parseRSSI extracts RSSI value from various types.
func parseRSSI(rssi any) int {
	switch v := rssi.(type) {
	case float64:
		return int(v)
	case string:
		if val, err := strconv.Atoi(v); err == nil {
			return val
		}
	}
	return 0
}

// createBluetoothDevice creates a BluetoothDevice from system_profiler device info.
// The forceNotConnected flag is used for paired-but-not-connected device lists.
func createBluetoothDevice(name string, info *btDeviceInfo, now time.Time, forceNotConnected bool) BluetoothDevice {
	isConnected := !forceNotConnected && (info.Connected == "Yes" || info.Connected == "attrib_Yes")
	dev := BluetoothDevice{
		Name:        name,
		Address:     info.Address,
		IsConnected: isConnected,
		IsPaired:    info.Paired == "Yes" || info.Paired == "attrib_Yes" || forceNotConnected,
		Type:        BluetoothTypeClassic,
		DeviceClass: mapMajorTypeToClass(info.MajorType),
		FirstSeen:   now,
		LastSeen:    now,
		Metadata:    make(map[string]any),
		RSSI:        parseRSSI(info.RSSI),
	}
	if info.Manufacturer != "" {
		dev.Vendor = info.Manufacturer
	}
	return dev
}

func parseSystemProfilerOutput(output []byte) ([]BluetoothDevice, error) {
	var data systemProfilerBluetooth
	if err := json.Unmarshal(output, &data); err != nil {
		return parseSystemProfilerText(string(output))
	}

	var devices []BluetoothDevice
	now := time.Now()

	for _, bt := range data.SPBluetoothDataType {
		for _, devMap := range bt.DevicesConnected {
			for name, info := range devMap {
				devices = append(devices, createBluetoothDevice(name, info, now, false))
			}
		}
		for _, devMap := range bt.DevicesPaired {
			for name, info := range devMap {
				devices = append(devices, createBluetoothDevice(name, info, now, true))
			}
		}
	}

	return devices, nil
}

// Regex patterns for text parsing (compiled once).
var (
	btAddrRegex = regexp.MustCompile(`Address:\s*([0-9A-Fa-f:-]+)`)
	btRSSIRegex = regexp.MustCompile(`RSSI:\s*(-?\d+)`)
)

// parseDeviceLine parses a line from system_profiler text output and updates the device.
func parseDeviceLine(line string, dev *BluetoothDevice) {
	if matches := btAddrRegex.FindStringSubmatch(line); len(matches) > 1 {
		dev.Address = matches[1]
		return
	}
	if matches := btRSSIRegex.FindStringSubmatch(line); len(matches) > 1 {
		if rssi, err := strconv.Atoi(matches[1]); err == nil {
			dev.RSSI = rssi
		}
		return
	}
	if strings.Contains(line, "Connected: Yes") {
		dev.IsConnected = true
		return
	}
	if strings.Contains(line, "Paired: Yes") {
		dev.IsPaired = true
		return
	}
	if strings.Contains(line, "Major Type:") {
		parts := strings.SplitN(line, ":", keyValueSplitParts)
		if len(parts) > 1 {
			dev.DeviceClass = mapMajorTypeToClass(strings.TrimSpace(parts[1]))
		}
	}
}

// isNewDeviceEntry checks if a line starts a new device entry.
func isNewDeviceEntry(line string) bool {
	return strings.HasSuffix(line, ":") && !strings.Contains(line, "Bluetooth")
}

// parseSystemProfilerText parses text output when JSON parsing fails.
func parseSystemProfilerText(output string) ([]BluetoothDevice, error) {
	var devices []BluetoothDevice
	var currentDevice *BluetoothDevice
	now := time.Now()

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if isNewDeviceEntry(line) {
			if currentDevice != nil && currentDevice.Address != "" {
				devices = append(devices, *currentDevice)
			}
			currentDevice = &BluetoothDevice{
				Name:        strings.TrimSuffix(line, ":"),
				Type:        BluetoothTypeClassic,
				DeviceClass: BluetoothClassUncategorized,
				FirstSeen:   now,
				LastSeen:    now,
				Metadata:    make(map[string]any),
			}
			continue
		}

		if currentDevice != nil {
			parseDeviceLine(line, currentDevice)
		}
	}

	if currentDevice != nil && currentDevice.Address != "" {
		devices = append(devices, *currentDevice)
	}

	return devices, nil
}

// scanBlueutil uses blueutil (homebrew) for additional device info.
func (s *BluetoothScanner) scanBlueutil(ctx context.Context) ([]BluetoothDevice, error) {
	path, err := exec.LookPath("blueutil")
	if err != nil {
		return nil, errors.New("blueutil not installed")
	}

	output, err := procrun.Output(exec.CommandContext(ctx, path, "--paired"))
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

		parts := strings.SplitN(line, ",", keyValueSplitParts)
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
