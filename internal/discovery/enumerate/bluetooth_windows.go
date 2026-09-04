//go:build windows

package enumerate

// Windows-specific Bluetooth discovery implementation.
// Provides the platform-specific scanPlatform method for the BluetoothScanner.
//
// Platform limitations:
//   - Requires Windows Bluetooth stack (not third-party)
//   - Limited compared to Linux BlueZ capabilities
//   - May require administrator privileges for some operations
//   - Full BLE scanning requires Windows Runtime API (WinRT)

import (
	"context"
	"errors"
)

// scanPlatform performs Bluetooth device discovery on Windows.
// Note: Full Bluetooth scanning on Windows requires either:
//   - Windows Runtime API (WinRT) via COM
//   - Windows Bluetooth SDK with CGO
//   - Third-party library like go-bluetooth
//
// For now, return a platform limitation error with guidance.
func (s *BluetoothScanner) scanPlatform(
	ctx context.Context,
	_ string,
	_ *BluetoothScanConfig,
) ([]BluetoothDevice, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Windows Bluetooth scanning requires WinRT or WMI which is complex in pure Go.
	// Return informative error directing user to platform requirements.
	return nil, errors.New("bluetooth scanning on Windows requires additional setup: " +
		"the Windows Bluetooth APIs require Windows Runtime (WinRT) integration. " +
		"See HARDWARE.md for platform-specific Bluetooth requirements and alternatives")
}
