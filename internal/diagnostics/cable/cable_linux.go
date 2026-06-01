//go:build linux

// Package cable provides TDR cable testing functionality.
// Linux implementation uses ethtool ioctl interface to perform Time Domain Reflectometry (TDR)
// testing on network interfaces, detecting cable faults and cable length.
package cable

import (
	"bufio"
	"bytes"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// SIOCETHTOOL ioctl command.
const siocethtool = 0x8946

// Ethtool command codes.
const (
	ethtoolGDrvInfo   = 0x00000003 // Get driver info.
	ethtoolGLinkState = 0x0000000a // Get link state.
)

// ethtoolDrvInfo structure for driver information.
type ethtoolDrvInfo struct {
	cmd         uint32
	driver      [32]byte
	version     [32]byte
	fwVersion   [32]byte
	busInfo     [32]byte
	eromVersion [32]byte
	reserved2   [12]byte
	nPrivFlags  uint32
	nStats      uint32
	testInfoLen uint32
	eedumpLen   uint32
	regdumpLen  uint32
}

// ethtoolValue structure for simple queries.
type ethtoolValue struct {
	cmd  uint32
	data uint32
}

// ifreq structure for ioctl.
type ifreq struct {
	name [16]byte
	data uintptr
}

// isSupportedPlatform checks if the NIC supports TDR on Linux.
// TDR support is driver-dependent and rare. Most NICs don't support it.
func isSupportedPlatform(iface string) bool {
	// Check if interface exists and get driver info
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return false
	}
	defer syscall.Close(fd)

	var req ifreq
	copy(req.name[:], iface)

	// Get driver info
	drvInfo := ethtoolDrvInfo{cmd: ethtoolGDrvInfo}
	req.data = uintptr(unsafe.Pointer(&drvInfo))

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(siocethtool),
		uintptr(unsafe.Pointer(&req)),
	)

	if errno != 0 {
		return false
	}

	// Convert driver name to string
	driverEnd := 0
	for i, b := range drvInfo.driver {
		if b == 0 {
			driverEnd = i
			break
		}
	}
	driver := string(drvInfo.driver[:driverEnd])

	// Only certain drivers support cable test
	// Common ones: e1000e, igb, ixgbe, marvell (some models)
	// Most virtual and common consumer NICs don't support it
	supportedDrivers := []string{
		"e1000e",
		"igb",
		"ixgbe",
		"i40e",
		"ice",
		"mlx5_core",
		"marvell",
	}

	for _, supported := range supportedDrivers {
		if driver == supported {
			return true
		}
	}

	return false
}

// testPlatform performs a cable test on Linux.
// Uses ethtool command-line tool for TDR testing when available.
func testPlatform(iface string) *TestResult {
	result := &TestResult{
		Supported: false,
		Status:    StatusUnknown,
		Faults:    make([]string, 0),
		WiringStd: Wiring568B, // Default to 568B (most common)
		Pinout:    Get568BPinout(),
	}

	// Get driver name for reference
	result.DriverName = getDriverName(iface)

	// Check if interface exists
	netIface, err := net.InterfaceByName(iface)
	if err != nil {
		result.Faults = append(result.Faults, "Interface not found")
		return result
	}

	// Try ethtool --cable-test first (requires supported driver and root)
	if tryEthtoolCableTest(iface, result) {
		return result
	}

	// Fallback to basic link state detection
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return result
	}
	defer syscall.Close(fd)

	var req ifreq
	copy(req.name[:], iface)

	// Get link state via ethtool ioctl
	linkState := ethtoolValue{cmd: ethtoolGLinkState}
	req.data = uintptr(unsafe.Pointer(&linkState))

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(siocethtool),
		uintptr(unsafe.Pointer(&req)),
	)

	if errno != 0 {
		// Fallback to interface flags
		if netIface.Flags&net.FlagUp != 0 && netIface.Flags&net.FlagRunning != 0 {
			result.Status = StatusOK
			result.Pairs = createOKPairs()
		} else if netIface.Flags&net.FlagUp != 0 {
			result.Status = StatusOpen
			result.Faults = append(result.Faults, "No carrier detected - cable may be disconnected or open")
		}
		return result
	}

	result.Supported = true

	if linkState.data != 0 {
		result.Status = StatusOK
		result.Pairs = createOKPairs()
	} else {
		// Link is down - could be cable issue
		result.Status = StatusOpen
		result.Faults = append(result.Faults, "No link detected - cable may be disconnected")
	}

	return result
}

// getDriverName returns the NIC driver name via ethtool.
func getDriverName(iface string) string {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return ""
	}
	defer syscall.Close(fd)

	var req ifreq
	copy(req.name[:], iface)

	drvInfo := ethtoolDrvInfo{cmd: ethtoolGDrvInfo}
	req.data = uintptr(unsafe.Pointer(&drvInfo))

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(siocethtool),
		uintptr(unsafe.Pointer(&req)),
	)

	if errno != 0 {
		return ""
	}

	driverEnd := 0
	for i, b := range drvInfo.driver {
		if b == 0 {
			driverEnd = i
			break
		}
	}
	return string(drvInfo.driver[:driverEnd])
}

// tryEthtoolCableTest attempts to run ethtool --cable-test and parse results.
// Returns true if successful, false if not supported or failed.
func tryEthtoolCableTest(iface string, result *TestResult) bool {
	// Check if ethtool supports cable-test for this interface
	cmd := exec.Command("ethtool", "--cable-test", iface)
	output, err := cmd.Output()
	if err != nil {
		// Cable test not supported or not root
		return false
	}

	result.Supported = true
	parseEthtoolCableTestOutput(output, result)
	return true
}

// parseEthtoolCableTestOutput parses ethtool --cable-test output.
// Example output:
//
//	Cable test completed:
//	    Pair A, pin 1/2:
//	        Status: OK
//	        Length: 5 meters
//	    Pair B, pin 3/6:
//	        Status: Open
//	        Length: 2.5 meters (fault)
func parseEthtoolCableTestOutput(output []byte, result *TestResult) {
	scanner := bufio.NewScanner(bytes.NewReader(output))

	// Regex patterns
	pairRe := regexp.MustCompile(`Pair\s+([A-D]),?\s*(?:pin\s+)?(\d+[/-]\d+)?`)
	statusRe := regexp.MustCompile(`Status:\s*(\w+)`)
	lengthRe := regexp.MustCompile(`Length:\s*([\d.]+)\s*(?:meters?|m)`)

	var currentPair *PairResult
	pairMap := map[string]*PairResult{
		"A": {Pair: "1-2", PairLetter: "A", Status: StatusUnknown},
		"B": {Pair: "3-6", PairLetter: "B", Status: StatusUnknown},
		"C": {Pair: "4-5", PairLetter: "C", Status: StatusUnknown},
		"D": {Pair: "7-8", PairLetter: "D", Status: StatusUnknown},
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Check for pair header
		if m := pairRe.FindStringSubmatch(line); m != nil {
			pairLetter := m[1]
			currentPair = pairMap[pairLetter]
		}

		// Check for status
		if m := statusRe.FindStringSubmatch(line); m != nil && currentPair != nil {
			statusStr := strings.ToLower(m[1])
			switch statusStr {
			case "ok":
				currentPair.Status = StatusOK
			case "open":
				currentPair.Status = StatusOpen
			case "short":
				currentPair.Status = StatusShort
			case "impedance_mismatch", "mismatch":
				currentPair.Status = StatusImpedanceMismatch
			default:
				currentPair.Status = StatusUnknown
			}
		}

		// Check for length
		if m := lengthRe.FindStringSubmatch(line); m != nil && currentPair != nil {
			if length, err := strconv.ParseFloat(m[1], 64); err == nil {
				currentPair.LengthM = &length
				ft := MetersToFeet(length)
				currentPair.LengthFt = &ft
			}
		}
	}

	// Compile pairs into result
	result.Pairs = make([]PairResult, 0, 4)
	overallStatus := StatusOK
	var minLength *float64

	for _, letter := range []string{"A", "B", "C", "D"} {
		pair := pairMap[letter]
		result.Pairs = append(result.Pairs, *pair)

		// Determine overall status (worst case)
		if pair.Status != StatusOK {
			overallStatus = pair.Status
			result.Faults = append(
				result.Faults,
				"Pair "+pair.PairLetter+" ("+pair.Pair+"): "+string(pair.Status),
			)
		}

		// Track minimum length (to fault)
		if pair.LengthM != nil && (minLength == nil || *pair.LengthM < *minLength) {
			minLength = pair.LengthM
		}
	}

	result.Status = overallStatus
	if minLength != nil {
		result.Length = minLength
		ft := MetersToFeet(*minLength)
		result.LengthFt = &ft
	}
}

// createOKPairs creates default OK status for all 4 pairs when link is up.
func createOKPairs() []PairResult {
	return []PairResult{
		{Pair: "1-2", PairLetter: "A", Status: StatusOK},
		{Pair: "3-6", PairLetter: "B", Status: StatusOK},
		{Pair: "4-5", PairLetter: "C", Status: StatusOK},
		{Pair: "7-8", PairLetter: "D", Status: StatusOK},
	}
}
