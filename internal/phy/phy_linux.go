//go:build linux

package phy

import (
	"bufio"
	"bytes"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// getPoEStatus detects PoE power status on Linux.
// PoE detection on Linux is limited - most NICs don't expose this.
// We check sysfs and ethtool for any PoE indicators.
func getPoEStatus(iface string) *PoEStatus {
	status := &PoEStatus{
		Detected: false,
	}

	// Try ethtool --show-eee (not PoE but can indicate power features)
	// Most consumer NICs don't expose PoE status directly
	// Enterprise NICs (Intel i210, i350 with PoE) may expose via driver

	// Check for PoE indicators in driver info
	cmd := exec.Command("ethtool", "-i", iface)
	output, err := cmd.Output()
	if err != nil {
		return status
	}

	// Look for PoE-capable drivers
	outputStr := string(output)
	poeDrivers := []string{"igb", "i40e", "ice"}
	for _, driver := range poeDrivers {
		if strings.Contains(outputStr, "driver: "+driver) {
			// These drivers may support PoE on certain hardware
			// Check for PoE-specific sysfs entries
			// TODO: Add sysfs PoE detection when we have hardware to test
			break
		}
	}

	return status
}

// getSFPInfo reads SFP module info and DDM via ethtool.
func getSFPInfo(iface string) *SFPInfo {
	info := &SFPInfo{
		Present:    false,
		DDMSupport: false,
	}

	// Run ethtool -m (module info) to get SFP/QSFP details
	cmd := exec.Command("ethtool", "-m", iface)
	output, err := cmd.Output()
	if err != nil {
		// No SFP or not supported
		return info
	}

	info.Present = true
	parseEthtoolModuleInfo(output, info)

	return info
}

// parseEthtoolModuleInfo parses ethtool -m output.
func parseEthtoolModuleInfo(output []byte, info *SFPInfo) {
	scanner := bufio.NewScanner(bytes.NewReader(output))

	// Regex patterns for parsing
	vendorRe := regexp.MustCompile(`Vendor name\s*:\s*(.+)`)
	partRe := regexp.MustCompile(`Vendor PN\s*:\s*(.+)`)
	serialRe := regexp.MustCompile(`Vendor SN\s*:\s*(.+)`)
	connectorRe := regexp.MustCompile(`Connector\s*:\s*(.+)`)
	wavelengthRe := regexp.MustCompile(`Laser wavelength\s*:\s*(\d+)`)
	distanceRe := regexp.MustCompile(`Link length.*:\s*(\d+)`)

	// DDM patterns
	tempRe := regexp.MustCompile(`Module temperature\s*:\s*([\d.+-]+)`)
	voltageRe := regexp.MustCompile(`Module voltage\s*:\s*([\d.]+)`)
	txPowerRe := regexp.MustCompile(`Laser output power\s*:\s*([\d.]+)\s*mW\s*/\s*([\d.-]+)\s*dBm`)
	rxPowerRe := regexp.MustCompile(
		`Receiver signal.*power\s*:\s*([\d.]+)\s*mW\s*/\s*([\d.-]+)\s*dBm`,
	)
	biasRe := regexp.MustCompile(`Laser bias current\s*:\s*([\d.]+)`)

	var ddm *DDMInfo

	for scanner.Scan() {
		line := scanner.Text()

		// Parse vendor info
		if m := vendorRe.FindStringSubmatch(line); m != nil {
			info.Vendor = strings.TrimSpace(m[1])
		}
		if m := partRe.FindStringSubmatch(line); m != nil {
			info.PartNumber = strings.TrimSpace(m[1])
		}
		if m := serialRe.FindStringSubmatch(line); m != nil {
			info.Serial = strings.TrimSpace(m[1])
		}
		if m := connectorRe.FindStringSubmatch(line); m != nil {
			info.Connector = strings.TrimSpace(m[1])
		}
		if m := wavelengthRe.FindStringSubmatch(line); m != nil {
			if wl, err := strconv.Atoi(m[1]); err == nil {
				info.Wavelength = wl
				// Infer type from wavelength
				switch {
				case wl >= 840 && wl <= 860:
					info.Type = "SR" // Short Range (multimode)
				case wl >= 1300 && wl <= 1320:
					info.Type = "LR" // Long Range (singlemode)
				case wl >= 1540 && wl <= 1560:
					info.Type = "ER" // Extended Range
				}
			}
		}
		if m := distanceRe.FindStringSubmatch(line); m != nil {
			if dist, err := strconv.Atoi(m[1]); err == nil {
				info.Distance = dist
			}
		}

		// Parse DDM values
		if m := tempRe.FindStringSubmatch(line); m != nil {
			info.DDMSupport = true
			if ddm == nil {
				ddm = &DDMInfo{}
			}
			if temp, err := strconv.ParseFloat(m[1], 64); err == nil {
				ddm.Temperature = temp
			}
		}
		if m := voltageRe.FindStringSubmatch(line); m != nil {
			info.DDMSupport = true
			if ddm == nil {
				ddm = &DDMInfo{}
			}
			if v, err := strconv.ParseFloat(m[1], 64); err == nil {
				ddm.Voltage = v
			}
		}
		if m := txPowerRe.FindStringSubmatch(line); m != nil {
			info.DDMSupport = true
			if ddm == nil {
				ddm = &DDMInfo{}
			}
			if mw, err := strconv.ParseFloat(m[1], 64); err == nil {
				ddm.TxPowerMw = mw
			}
			if dbm, err := strconv.ParseFloat(m[2], 64); err == nil {
				ddm.TxPowerDbm = dbm
			}
		}
		if m := rxPowerRe.FindStringSubmatch(line); m != nil {
			info.DDMSupport = true
			if ddm == nil {
				ddm = &DDMInfo{}
			}
			if mw, err := strconv.ParseFloat(m[1], 64); err == nil {
				ddm.RxPowerMw = mw
			}
			if dbm, err := strconv.ParseFloat(m[2], 64); err == nil {
				ddm.RxPowerDbm = dbm
			}
		}
		if m := biasRe.FindStringSubmatch(line); m != nil {
			info.DDMSupport = true
			if ddm == nil {
				ddm = &DDMInfo{}
			}
			if bias, err := strconv.ParseFloat(m[1], 64); err == nil {
				ddm.LaserBiasMa = bias
			}
		}

		// Check for alarms/warnings in the output
		if strings.Contains(line, "Alarm") && strings.Contains(line, "high") {
			if ddm == nil {
				ddm = &DDMInfo{}
			}
			ddm.Alarms = append(ddm.Alarms, strings.TrimSpace(line))
		}
		if strings.Contains(line, "Warning") {
			if ddm == nil {
				ddm = &DDMInfo{}
			}
			ddm.Warnings = append(ddm.Warnings, strings.TrimSpace(line))
		}
	}

	info.DDM = ddm
}
