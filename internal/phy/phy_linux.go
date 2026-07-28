//go:build linux

package phy

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const ethtoolModuleTimeout = 10 * time.Second

// getPoEStatus returns an empty status because Linux NIC drivers do not expose
// reliable PoE telemetry through the interfaces Seed currently supports.
func getPoEStatus(_ string) *PoEStatus {
	return &PoEStatus{}
}

// getSFPInfo reads SFP module info and DDM via ethtool.
func getSFPInfo(iface string) *SFPInfo {
	info := &SFPInfo{
		Present:    false,
		DDMSupport: false,
	}

	// Run ethtool -m (module info) to get SFP/QSFP details
	ctx, cancel := context.WithTimeout(context.Background(), ethtoolModuleTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ethtool", "-m", iface)
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
	patterns := newModulePatterns()
	var ddm *DDMInfo
	for scanner.Scan() {
		line := scanner.Text()
		parseModuleIdentity(line, info, patterns)
		ddm = parseModuleDDM(line, info, ddm, patterns)
	}
	info.DDM = ddm
}

type modulePatterns struct {
	vendor, part, serial, connector, wavelength, distance *regexp.Regexp
	temperature, voltage, txPower, rxPower, bias          *regexp.Regexp
}

func newModulePatterns() modulePatterns {
	return modulePatterns{
		vendor:      regexp.MustCompile(`Vendor name\s*:\s*(.+)`),
		part:        regexp.MustCompile(`Vendor PN\s*:\s*(.+)`),
		serial:      regexp.MustCompile(`Vendor SN\s*:\s*(.+)`),
		connector:   regexp.MustCompile(`Connector\s*:\s*(.+)`),
		wavelength:  regexp.MustCompile(`Laser wavelength\s*:\s*(\d+)`),
		distance:    regexp.MustCompile(`Link length.*:\s*(\d+)`),
		temperature: regexp.MustCompile(`Module temperature\s*:\s*([\d.+-]+)`),
		voltage:     regexp.MustCompile(`Module voltage\s*:\s*([\d.]+)`),
		txPower:     regexp.MustCompile(`Laser output power\s*:\s*([\d.]+)\s*mW\s*/\s*([\d.-]+)\s*dBm`),
		rxPower:     regexp.MustCompile(`Receiver signal.*power\s*:\s*([\d.]+)\s*mW\s*/\s*([\d.-]+)\s*dBm`),
		bias:        regexp.MustCompile(`Laser bias current\s*:\s*([\d.]+)`),
	}
}

func parseModuleIdentity(line string, info *SFPInfo, patterns modulePatterns) {
	if match := patterns.vendor.FindStringSubmatch(line); match != nil {
		info.Vendor = strings.TrimSpace(match[1])
	}
	if match := patterns.part.FindStringSubmatch(line); match != nil {
		info.PartNumber = strings.TrimSpace(match[1])
	}
	if match := patterns.serial.FindStringSubmatch(line); match != nil {
		info.Serial = strings.TrimSpace(match[1])
	}
	if match := patterns.connector.FindStringSubmatch(line); match != nil {
		info.Connector = strings.TrimSpace(match[1])
	}
	if match := patterns.wavelength.FindStringSubmatch(line); match != nil {
		setModuleWavelength(match[1], info)
	}
	if match := patterns.distance.FindStringSubmatch(line); match != nil {
		if distance, err := strconv.Atoi(match[1]); err == nil {
			info.Distance = distance
		}
	}
}

func setModuleWavelength(value string, info *SFPInfo) {
	wavelength, err := strconv.Atoi(value)
	if err != nil {
		return
	}
	info.Wavelength = wavelength
	switch {
	case wavelength >= 840 && wavelength <= 860:
		info.Type = "SR"
	case wavelength >= 1300 && wavelength <= 1320:
		info.Type = "LR"
	case wavelength >= 1540 && wavelength <= 1560:
		info.Type = "ER"
	}
}

func parseModuleDDM(line string, info *SFPInfo, ddm *DDMInfo, patterns modulePatterns) *DDMInfo {
	if match := patterns.temperature.FindStringSubmatch(line); match != nil {
		ddm = ensureDDM(info, ddm)
		ddm.Temperature, _ = strconv.ParseFloat(match[1], 64)
	}
	if match := patterns.voltage.FindStringSubmatch(line); match != nil {
		ddm = ensureDDM(info, ddm)
		ddm.Voltage, _ = strconv.ParseFloat(match[1], 64)
	}
	if match := patterns.txPower.FindStringSubmatch(line); match != nil {
		ddm = ensureDDM(info, ddm)
		ddm.TxPowerMw, _ = strconv.ParseFloat(match[1], 64)
		ddm.TxPowerDbm, _ = strconv.ParseFloat(match[2], 64)
	}
	if match := patterns.rxPower.FindStringSubmatch(line); match != nil {
		ddm = ensureDDM(info, ddm)
		ddm.RxPowerMw, _ = strconv.ParseFloat(match[1], 64)
		ddm.RxPowerDbm, _ = strconv.ParseFloat(match[2], 64)
	}
	if match := patterns.bias.FindStringSubmatch(line); match != nil {
		ddm = ensureDDM(info, ddm)
		ddm.LaserBiasMa, _ = strconv.ParseFloat(match[1], 64)
	}
	if strings.Contains(line, "Alarm") && strings.Contains(line, "high") {
		ddm = ensureDDM(info, ddm)
		ddm.Alarms = append(ddm.Alarms, strings.TrimSpace(line))
	}
	if strings.Contains(line, "Warning") {
		ddm = ensureDDM(info, ddm)
		ddm.Warnings = append(ddm.Warnings, strings.TrimSpace(line))
	}
	return ddm
}

func ensureDDM(info *SFPInfo, ddm *DDMInfo) *DDMInfo {
	info.DDMSupport = true
	if ddm == nil {
		return &DDMInfo{}
	}
	return ddm
}
