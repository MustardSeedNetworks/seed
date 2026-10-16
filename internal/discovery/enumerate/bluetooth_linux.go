//go:build linux

package enumerate

// bluetooth_linux.go implements Bluetooth scanning on Linux. Discovery runs
// over the kernel management interface (bluetooth_mgmt_linux.go). When
// bluetoothd is running, bluetoothctl adds what only it knows: pairing, trust
// and the devices connected to this host, which stop advertising once
// connected.

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Linux scanning constants.
const (
	bluetoothctlTimeout = 30 * time.Second
	deviceLineParts     = 3
	minimumDeviceParts  = 2
	uuidLineParts       = 2
)

// scanPlatform performs Bluetooth scanning on Linux.
func (s *BluetoothScanner) scanPlatform(
	ctx context.Context,
	adapter string,
	config *BluetoothScanConfig,
) ([]BluetoothDevice, error) {
	scanTypes, err := mgmtScanTypes(config)
	if err != nil {
		return nil, err
	}
	duration := time.Duration(config.ScanDurationSec) * time.Second
	if duration <= 0 {
		duration = defaultScanDurationSec * time.Second
	}
	devices, err := scanMgmt(ctx, adapter, scanTypes, duration)
	if err != nil {
		return nil, err
	}

	ctlCtx, cancel := context.WithTimeout(ctx, bluetoothctlTimeout)
	defer cancel()
	if known, ctlErr := s.scanBluetoothctl(ctlCtx); ctlErr == nil {
		devices = applyBluetoothctlState(devices, known)
	}
	return devices, nil
}

func mgmtScanTypes(config *BluetoothScanConfig) (byte, error) {
	var types byte
	if config.IncludeClassic {
		types |= mgmtScanBREDR
	}
	if config.IncludeBLE {
		types |= mgmtScanLE
	}
	if types == 0 {
		return 0, errors.New("bluetooth scan includes neither classic nor BLE devices")
	}
	return types, nil
}

// applyBluetoothctlState copies bluetoothd's pairing state onto the devices
// the scan heard, and adds connected devices it did not hear. bluetoothd's
// cache of devices seen in the past is not evidence a device is in range now.
func applyBluetoothctlState(heard, known []BluetoothDevice) []BluetoothDevice {
	index := make(map[string]int, len(heard))
	for i, dev := range heard {
		index[normalizeMAC(dev.Address)] = i
	}
	for _, k := range known {
		i, ok := index[normalizeMAC(k.Address)]
		if !ok {
			if k.IsConnected {
				heard = append(heard, k)
			}
			continue
		}
		dev := &heard[i]
		dev.IsPaired, dev.IsTrusted, dev.IsBlocked, dev.IsConnected = k.IsPaired, k.IsTrusted, k.IsBlocked, k.IsConnected
		dev.Alias = k.Alias
		if dev.Name == "" {
			dev.Name = k.Name
		}
	}
	return heard
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
	for line := range strings.SplitSeq(string(output), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Device ") {
			continue
		}

		parts := strings.SplitN(line, " ", deviceLineParts)
		if len(parts) < minimumDeviceParts {
			continue
		}

		address := parts[1]
		name := ""
		if len(parts) >= deviceLineParts {
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
		info, infoErr := s.getBluetoothctlDeviceInfo(ctx, address)
		if infoErr == nil {
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

	for line := range strings.SplitSeq(output, "\n") {
		parseBluetoothctlLine(strings.TrimSpace(line), &dev)
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

func parseBluetoothctlLine(line string, dev *BluetoothDevice) {
	key, value, found := strings.Cut(line, ":")
	if !found {
		return
	}
	value = strings.TrimSpace(value)
	switch key {
	case "Name":
		dev.Name = value
	case "Alias":
		dev.Alias = value
	case "Class":
		setBluetoothClass(dev, value)
	case "Paired":
		dev.IsPaired = strings.Contains(value, "yes")
	case "Trusted":
		dev.IsTrusted = strings.Contains(value, "yes")
	case "Blocked":
		dev.IsBlocked = strings.Contains(value, "yes")
	case "Connected":
		dev.IsConnected = strings.Contains(value, "yes")
	case "RSSI":
		dev.RSSI, _ = strconv.Atoi(value)
	case "TxPower":
		dev.TxPower, _ = strconv.Atoi(value)
	case "UUID":
		appendBluetoothUUID(dev, line)
	case "ManufacturerData Key":
		setBluetoothManufacturer(dev, value)
	}
}

func setBluetoothClass(dev *BluetoothDevice, value string) {
	class, err := strconv.ParseUint(strings.TrimPrefix(value, "0x"), 16, 32)
	if err == nil {
		dev.ClassOfDev = uint32(class)
		dev.DeviceClass = ClassOfDeviceToClass(dev.ClassOfDev)
	}
}

func appendBluetoothUUID(dev *BluetoothDevice, line string) {
	parts := strings.SplitN(line, "(", uuidLineParts)
	if len(parts) == uuidLineParts {
		dev.ServiceUUIDs = append(dev.ServiceUUIDs, strings.TrimSuffix(strings.TrimSpace(parts[1]), ")"))
	}
}

func setBluetoothManufacturer(dev *BluetoothDevice, value string) {
	manufacturer, err := strconv.ParseUint(strings.TrimPrefix(value, "0x"), 16, 16)
	if err == nil {
		dev.ManufacturerID = uint16(manufacturer)
	}
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
