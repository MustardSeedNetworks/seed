// SPDX-License-Identifier: BUSL-1.1

package license

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"iter"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Fingerprint constants.
const (
	defaultMAC        = "00:00:00:00:00:00"
	hashLength        = 16 // First 16 hex characters (64 bits of entropy).
	commandTimeout    = 5  // Seconds for external command timeout.
	splitPartsCount   = 2  // Expected parts when splitting on : or =.
	defaultLinuxCPU   = "LINUX-DEFAULT"
	defaultDarwinCPU  = "DARWIN-DEFAULT"
	defaultLinuxDisk  = "LINUX-DISK"
	defaultDarwinDisk = "DARWIN-DISK"
	unknownSerial     = "UNKNOWN"
)

// DeviceFingerprint contains hardware identifiers for device binding.
type DeviceFingerprint struct {
	MACAddress string `json:"mac"`
	CPUSerial  string `json:"cpu"`
	DiskSerial string `json:"disk"`
	Hostname   string `json:"hostname"`
	Platform   string `json:"platform"`
}

// generateFingerprintOnce memoizes the device fingerprint for the process
// lifetime. The fingerprint is a stable machine property, but computing it
// shells out to OS tools (getCPUSerial/getDiskSerial, 5s timeout) and reads
// the host interfaces, all of which can transiently fail, time out, or reorder
// under load. Recomputing per call therefore yielded a NON-DETERMINISTIC
// Hash() within a single process — which spuriously invalidated a valid
// license on reload (state.DeviceHash != fingerprint.Hash()) on a busy host,
// and flaked TestActivationLifecycle under parallel test load. Computing it
// once and reusing it is both correct (the device does not change while the
// process runs) and what the original doc comment always promised.
var generateFingerprintOnce = sync.OnceValues(computeFingerprint)

// GenerateFingerprint returns the process-wide device fingerprint, computing
// it on first call and caching it thereafter (hardware IDs don't change).
func GenerateFingerprint() (*DeviceFingerprint, error) {
	return generateFingerprintOnce()
}

// computeFingerprint gathers the raw device identifiers. Called at most once
// per process via generateFingerprintOnce.
func computeFingerprint() (*DeviceFingerprint, error) {
	fp := &DeviceFingerprint{
		MACAddress: "",
		CPUSerial:  "",
		DiskSerial: "",
		Hostname:   "",
		Platform:   runtime.GOOS,
	}

	hostname, err := os.Hostname()
	if err == nil {
		fp.Hostname = hostname
	}

	fp.MACAddress = getPrimaryMAC()
	fp.CPUSerial = getCPUSerial()
	fp.DiskSerial = getDiskSerial()

	return fp, nil
}

// Hash returns a 16-character hash of the fingerprint.
func (fp *DeviceFingerprint) Hash() string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s",
		fp.MACAddress,
		fp.CPUSerial,
		fp.DiskSerial,
		fp.Hostname,
		fp.Platform,
	)

	hash := sha256.Sum256([]byte(data))
	hexHash := hex.EncodeToString(hash[:])

	// Return first 16 characters (64 bits of entropy).
	return strings.ToUpper(hexHash[:hashLength])
}

// String returns a human-readable representation.
func (fp *DeviceFingerprint) String() string {
	const maskShowChars = 8
	const cpuDiskMaskChars = 4
	return fmt.Sprintf("MAC=%s CPU=%s DISK=%s HOST=%s",
		maskString(fp.MACAddress, maskShowChars),
		maskString(fp.CPUSerial, cpuDiskMaskChars),
		maskString(fp.DiskSerial, cpuDiskMaskChars),
		fp.Hostname,
	)
}

// getPrimaryMAC returns the MAC address of the first non-loopback interface.
func getPrimaryMAC() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return defaultMAC
	}

	for _, iface := range interfaces {
		// Skip loopback and interfaces without MAC.
		if iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) == 0 {
			continue
		}
		// Skip virtual interfaces (common patterns).
		name := strings.ToLower(iface.Name)
		if strings.HasPrefix(name, "veth") ||
			strings.HasPrefix(name, "docker") ||
			strings.HasPrefix(name, "br-") ||
			strings.HasPrefix(name, "virbr") {
			continue
		}
		return iface.HardwareAddr.String()
	}

	return defaultMAC
}

// getCPUSerial returns CPU serial number (platform-specific).
func getCPUSerial() string {
	switch runtime.GOOS {
	case "linux":
		return getLinuxCPUSerial()
	case "darwin":
		return getDarwinCPUSerial()
	default:
		return unknownSerial
	}
}

// splitLines returns an iterator over lines in a string.
func splitLines(s string) iter.Seq[string] {
	return strings.SplitSeq(s, "\n")
}

// getLinuxCPUSerial reads CPU serial from /proc/cpuinfo or dmidecode.
func getLinuxCPUSerial() string {
	// Try /proc/cpuinfo first (works on ARM).
	data, err := os.ReadFile("/proc/cpuinfo")
	if err == nil {
		for line := range splitLines(string(data)) {
			if strings.HasPrefix(line, "Serial") {
				parts := strings.Split(line, ":")
				if len(parts) == splitPartsCount {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}

	// Try dmidecode (requires root).
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "dmidecode", "-s", "processor-id").Output()
	if err == nil {
		serial := strings.TrimSpace(string(out))
		if serial != "" {
			return serial
		}
	}

	// Try /sys/class/dmi/id/product_serial.
	data, err = os.ReadFile("/sys/class/dmi/id/product_serial")
	if err == nil {
		return strings.TrimSpace(string(data))
	}

	return defaultLinuxCPU
}

// getDarwinCPUSerial reads hardware UUID on macOS.
func getDarwinCPUSerial() string {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return defaultDarwinCPU
	}

	for line := range splitLines(string(out)) {
		if strings.Contains(line, "IOPlatformUUID") {
			parts := strings.Split(line, "=")
			if len(parts) == splitPartsCount {
				uuid := strings.TrimSpace(parts[1])
				uuid = strings.Trim(uuid, "\"")
				return uuid
			}
		}
	}

	return defaultDarwinCPU
}

// getDiskSerial returns root disk serial number.
func getDiskSerial() string {
	switch runtime.GOOS {
	case "linux":
		return getLinuxDiskSerial()
	case "darwin":
		return getDarwinDiskSerial()
	default:
		return unknownSerial
	}
}

// getLinuxDiskSerial reads disk serial from /sys or udevadm.
func getLinuxDiskSerial() string {
	// Try common disk paths.
	diskPaths := []string{
		"/sys/block/sda/device/serial",
		"/sys/block/nvme0n1/device/serial",
		"/sys/block/vda/serial",
	}

	for _, path := range diskPaths {
		// Read from known system paths.
		f, err := os.Open(filepath.Clean(path))
		if err != nil {
			continue
		}
		data, err := io.ReadAll(f)
		_ = f.Close()
		if err == nil {
			serial := strings.TrimSpace(string(data))
			if serial != "" {
				return serial
			}
		}
	}

	// Try udevadm.
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "udevadm", "info", "--query=property", "--name=/dev/sda").Output()
	if err == nil {
		for line := range splitLines(string(out)) {
			if serial, found := strings.CutPrefix(line, "ID_SERIAL="); found {
				return serial
			}
		}
	}

	return defaultLinuxDisk
}

// getDarwinDiskSerial reads disk serial on macOS.
func getDarwinDiskSerial() string {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "system_profiler", "SPSerialATADataType").Output()
	if err == nil {
		for line := range splitLines(string(out)) {
			if strings.Contains(line, "Serial Number") {
				parts := strings.Split(line, ":")
				if len(parts) == splitPartsCount {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}

	// Try NVMe.
	ctx2, cancel2 := context.WithTimeout(context.Background(), commandTimeout*time.Second)
	defer cancel2()
	out, err = exec.CommandContext(ctx2, "system_profiler", "SPNVMeDataType").Output()
	if err == nil {
		for line := range splitLines(string(out)) {
			if strings.Contains(line, "Serial Number") {
				parts := strings.Split(line, ":")
				if len(parts) == splitPartsCount {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}

	return defaultDarwinDisk
}

// maskString masks a string for display, showing only first N chars.
func maskString(s string, show int) string {
	if len(s) <= show {
		return s
	}
	return s[:show] + "****"
}
