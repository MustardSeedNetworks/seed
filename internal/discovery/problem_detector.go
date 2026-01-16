package discovery

// problem_detector.go implements automated network problem detection.
// It scans the discovery database for issues like duplicate IPs, duplex mismatches,
// high error rates, and WiFi problems.

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/krisarmstrong/seed/internal/logging"
)

// keyValueParts is the number of parts when splitting key-value strings.
const keyValueParts = 2

// safeUint64ToInt64 safely converts uint64 to int64, clamping at MaxInt64.
func safeUint64ToInt64(v uint64) int64 {
	if v > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(v)
}

// titleCase converts a string to title case (first letter uppercase).
// Replacement for deprecated [strings.Title].
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// ProblemDetector scans for network problems and issues.
type ProblemDetector struct {
	mu          sync.RWMutex
	thresholds  ProblemThresholds
	lastScan    *ProblemDetectionResult
	lastScanAt  time.Time
	listeners   []ProblemListener
	knownIssues map[string]*NetworkProblem // ID -> Problem
}

// ProblemListener is notified when problems are detected or resolved.
type ProblemListener interface {
	OnProblemDetected(problem *NetworkProblem)
	OnProblemResolved(problem *NetworkProblem)
}

// NewProblemDetector creates a new problem detector with default thresholds.
func NewProblemDetector() *ProblemDetector {
	return &ProblemDetector{
		thresholds:  DefaultProblemThresholds(),
		knownIssues: make(map[string]*NetworkProblem),
	}
}

// SetThresholds updates the problem detection thresholds.
func (d *ProblemDetector) SetThresholds(t ProblemThresholds) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.thresholds = t
}

// GetThresholds returns the current thresholds.
func (d *ProblemDetector) GetThresholds() ProblemThresholds {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.thresholds
}

// AddListener adds a problem event listener.
func (d *ProblemDetector) AddListener(l ProblemListener) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listeners = append(d.listeners, l)
}

// Scan runs all problem detection checks and returns results.
func (d *ProblemDetector) Scan(ctx context.Context, devices []*DiscoveredDevice) (*ProblemDetectionResult, error) {
	start := time.Now()
	logger := logging.GetLogger()
	logger.InfoContext(ctx, "Starting problem detection scan", "devices", len(devices))

	d.mu.RLock()
	thresholds := d.thresholds
	d.mu.RUnlock()

	result := &ProblemDetectionResult{
		Problems:         []NetworkProblem{},
		IPConflicts:      []IPConflict{},
		DuplexMismatches: []DuplexMismatch{},
		STPEvents:        []STPEvent{},
		ResourceAlerts:   []ResourceThreshold{},
		InterfaceErrors:  []InterfaceErrorStats{},
		WiFiProblems:     []WiFiProblem{},
		ScanTime:         start,
	}

	// Run detection checks
	d.detectIPConflicts(ctx, devices, result)
	d.detectDuplexMismatches(ctx, devices, result, thresholds)
	d.detectResourceThresholds(ctx, devices, result, thresholds)
	d.detectInterfaceErrors(ctx, devices, result, thresholds)

	// Aggregate all issues into Problems list
	d.aggregateProblems(result)

	result.ScanDurationMS = time.Since(start).Milliseconds()

	// Update cached result
	d.mu.Lock()
	d.lastScan = result
	d.lastScanAt = start
	d.mu.Unlock()

	// Notify listeners
	d.notifyListeners(result)

	logger.InfoContext(ctx, "Problem detection complete",
		"problems", len(result.Problems),
		"ip_conflicts", len(result.IPConflicts),
		"duration_ms", result.ScanDurationMS,
	)

	return result, nil
}

// ScanWiFi runs WiFi-specific problem detection.
func (d *ProblemDetector) ScanWiFi(
	_ context.Context,
	scanResult *WiFiScanResult,
	authorizedSSIDs, authorizedBSSIDs []string,
) []WiFiProblem {
	if scanResult == nil {
		return nil
	}

	d.mu.RLock()
	thresholds := d.thresholds
	d.mu.RUnlock()

	authSSIDs := buildSSIDLookup(authorizedSSIDs)
	authBSSIDs := buildBSSIDLookup(authorizedBSSIDs)
	now := time.Now()

	var problems []WiFiProblem
	problems = append(problems, d.checkRogueAPs(scanResult.APs, authSSIDs, authBSSIDs, thresholds, now)...)
	problems = append(problems, d.checkChannelInterference(scanResult.APs, thresholds, now)...)
	problems = append(problems, d.checkUtilization(scanResult.Utilization, thresholds, now)...)

	return problems
}

func buildSSIDLookup(ssids []string) map[string]bool {
	lookup := make(map[string]bool)
	for _, ssid := range ssids {
		lookup[ssid] = true
	}
	return lookup
}

func buildBSSIDLookup(bssids []string) map[string]bool {
	lookup := make(map[string]bool)
	for _, bssid := range bssids {
		lookup[strings.ToUpper(bssid)] = true
	}
	return lookup
}

func (d *ProblemDetector) checkRogueAPs(
	aps []WiFiAccessPoint,
	authSSIDs, authBSSIDs map[string]bool,
	thresholds ProblemThresholds,
	now time.Time,
) []WiFiProblem {
	var problems []WiFiProblem
	for _, ap := range aps {
		normalizedBSSID := strings.ToUpper(ap.BSSID)

		if !authBSSIDs[normalizedBSSID] && !authSSIDs[ap.SSIDName] && authSSIDs[ap.SSIDName] {
			problems = append(problems, WiFiProblem{
				ProblemType:    "rogue_ap",
				SSID:           ap.SSIDName,
				BSSID:          ap.BSSID,
				Channel:        ap.Channel,
				Band:           ap.Band,
				SignalDBm:      ap.SignalDBm,
				IsRogue:        true,
				IsUnauthorized: true,
				FirstSeen:      now,
				LastSeen:       now,
			})
		}

		if ap.SignalDBm < thresholds.MinSignalDBm {
			problems = append(problems, WiFiProblem{
				ProblemType: "weak_signal",
				SSID:        ap.SSIDName,
				BSSID:       ap.BSSID,
				Channel:     ap.Channel,
				Band:        ap.Band,
				SignalDBm:   ap.SignalDBm,
				FirstSeen:   now,
				LastSeen:    now,
			})
		}
	}
	return problems
}

func (d *ProblemDetector) checkChannelInterference(
	aps []WiFiAccessPoint,
	thresholds ProblemThresholds,
	now time.Time,
) []WiFiProblem {
	channelAPCounts := make(map[string]int)
	for _, ap := range aps {
		key := fmt.Sprintf("%s-%d", ap.Band, ap.Channel)
		channelAPCounts[key]++
	}

	var problems []WiFiProblem
	for key, count := range channelAPCounts {
		if count > thresholds.MaxCoChannelAPs {
			parts := strings.SplitN(key, "-", keyValueParts)
			if len(parts) != keyValueParts {
				continue
			}
			band := WiFiBand(parts[0])
			var channel int
			if _, err := fmt.Sscanf(parts[1], "%d", &channel); err != nil {
				continue
			}

			problems = append(problems, WiFiProblem{
				ProblemType:  "channel_interference",
				Channel:      channel,
				Band:         band,
				CoChannelAPs: count,
				FirstSeen:    now,
				LastSeen:     now,
			})
		}
	}
	return problems
}

func (d *ProblemDetector) checkUtilization(
	utilization []ChannelUtilization,
	thresholds ProblemThresholds,
	now time.Time,
) []WiFiProblem {
	var problems []WiFiProblem
	for _, util := range utilization {
		if util.UtilizationPercent > thresholds.MaxChannelUtil {
			problems = append(problems, WiFiProblem{
				ProblemType:        "high_utilization",
				Channel:            util.Channel,
				Band:               util.Band,
				UtilizationPercent: util.UtilizationPercent,
				FirstSeen:          now,
				LastSeen:           now,
			})
		}
	}
	return problems
}

// detectIPConflicts finds duplicate IP addresses.
// Uses the HasDuplicateIP and DuplicateMACs fields already populated by discovery.
func (d *ProblemDetector) detectIPConflicts(
	_ context.Context,
	devices []*DiscoveredDevice,
	result *ProblemDetectionResult,
) {
	now := time.Now()

	for _, dev := range devices {
		if !dev.HasDuplicateIP || len(dev.DuplicateMACs) == 0 {
			continue
		}

		// Build MAC list including the device's own MAC
		macs := append([]string{dev.MAC}, dev.DuplicateMACs...)

		conflict := IPConflict{
			IPAddress: dev.IP,
			MACs:      macs,
			DeviceIDs: []string{}, // DiscoveredDevice doesn't have ID
			FirstSeen: now,
			LastSeen:  now,
		}
		result.IPConflicts = append(result.IPConflicts, conflict)
	}
}

// detectDuplexMismatches finds speed/duplex negotiation issues.
// Uses SNMP interface data when available.
func (d *ProblemDetector) detectDuplexMismatches(
	_ context.Context,
	_ []*DiscoveredDevice,
	_ *ProblemDetectionResult,
	_ ProblemThresholds,
) {
	// Currently SNMP interfaces don't have duplex info in the standard MIBs
	// This would require vendor-specific MIBs or LLDP/CDP data
	// For now, we detect issues via high error rates in detectInterfaceErrors
}

// detectResourceThresholds checks device resources against thresholds.
// Note: SNMPFullData doesn't currently have CPU/Memory fields - this is a placeholder
// for when those MIBs are implemented in the SNMP collector.
func (d *ProblemDetector) detectResourceThresholds(
	_ context.Context,
	_ []*DiscoveredDevice,
	_ *ProblemDetectionResult,
	_ ProblemThresholds,
) {
	// Resource thresholds will be checked when CPU/Memory SNMP MIBs are collected
	// (HOST-RESOURCES-MIB, UCD-SNMP-MIB, etc.)
}

// detectInterfaceErrors checks for high error rates on interfaces.
func (d *ProblemDetector) detectInterfaceErrors(
	_ context.Context,
	devices []*DiscoveredDevice,
	result *ProblemDetectionResult,
	thresholds ProblemThresholds,
) {
	now := time.Now()

	for _, dev := range devices {
		if dev.SNMPData == nil {
			continue
		}

		for _, iface := range dev.SNMPData.Interfaces {
			hasErrors := false
			stats := InterfaceErrorStats{
				DeviceID:      dev.MAC, // Use MAC as device identifier
				InterfaceName: iface.Name,
				InputErrors:   safeUint64ToInt64(iface.InErrors),
				OutputErrors:  safeUint64ToInt64(iface.OutErrors),
				RecordedAt:    now,
			}

			if safeUint64ToInt64(iface.InErrors) > thresholds.InputErrorsPerMin {
				hasErrors = true
			}
			if safeUint64ToInt64(iface.OutErrors) > thresholds.OutputErrorsPerMin {
				hasErrors = true
			}

			if hasErrors {
				result.InterfaceErrors = append(result.InterfaceErrors, stats)
			}
		}
	}
}

// createIPConflictProblem creates a NetworkProblem from an IPConflict.
func createIPConflictProblem(conflict IPConflict) NetworkProblem {
	return NetworkProblem{
		ID:       uuid.New().String(),
		Category: ProblemCategoryIPConflict,
		Type:     "duplicate_ip",
		Severity: ProblemSeverityCritical,
		Status:   ProblemStatusActive,
		Title:    fmt.Sprintf("Duplicate IP: %s", conflict.IPAddress),
		Description: fmt.Sprintf("IP address %s is claimed by %d devices: %s",
			conflict.IPAddress, len(conflict.MACs), strings.Join(conflict.MACs, ", ")),
		IPAddress:       conflict.IPAddress,
		AffectedMACs:    strings.Join(conflict.MACs, ","),
		FirstSeen:       conflict.FirstSeen,
		LastSeen:        conflict.LastSeen,
		OccurrenceCount: 1,
	}
}

// createDuplexMismatchProblem creates a NetworkProblem from a DuplexMismatch.
func createDuplexMismatchProblem(mismatch DuplexMismatch) NetworkProblem {
	return NetworkProblem{
		ID:       uuid.New().String(),
		Category: ProblemCategoryDuplexMismatch,
		Type:     "duplex_mismatch",
		Severity: ProblemSeverityWarning,
		Status:   ProblemStatusActive,
		Title:    fmt.Sprintf("Duplex Mismatch: %s", mismatch.InterfaceName),
		Description: fmt.Sprintf("Interface %s is running in %s duplex mode with %d collisions",
			mismatch.InterfaceName, mismatch.LocalDuplex, mismatch.CollisionCount),
		DeviceID:        mismatch.DeviceID,
		InterfaceName:   mismatch.InterfaceName,
		FirstSeen:       mismatch.FirstSeen,
		LastSeen:        mismatch.LastSeen,
		OccurrenceCount: 1,
	}
}

// createResourceAlertProblem creates a NetworkProblem from a ResourceThreshold.
func createResourceAlertProblem(alert ResourceThreshold, now time.Time) NetworkProblem {
	return NetworkProblem{
		ID:       uuid.New().String(),
		Category: ProblemCategoryResourceUsage,
		Type:     fmt.Sprintf("high_%s", alert.ResourceType),
		Severity: SeverityForResourceUsage(alert.CurrentValue, alert.Threshold),
		Status:   ProblemStatusActive,
		Title:    fmt.Sprintf("High %s Usage", titleCase(alert.ResourceType)),
		Description: fmt.Sprintf("%s usage at %.1f%% (threshold: %.1f%%)",
			titleCase(alert.ResourceType), alert.CurrentValue, alert.Threshold),
		DeviceID:        alert.DeviceID,
		CurrentValue:    alert.CurrentValue,
		ThresholdValue:  alert.Threshold,
		Unit:            alert.Unit,
		FirstSeen:       now,
		LastSeen:        now,
		OccurrenceCount: 1,
	}
}

// createInterfaceErrorProblem creates a NetworkProblem from InterfaceErrorStats.
func createInterfaceErrorProblem(errStats InterfaceErrorStats, now time.Time) NetworkProblem {
	return NetworkProblem{
		ID:       uuid.New().String(),
		Category: ProblemCategoryInterfaceErrors,
		Type:     "interface_errors",
		Severity: ProblemSeverityWarning,
		Status:   ProblemStatusActive,
		Title:    fmt.Sprintf("Interface Errors: %s", errStats.InterfaceName),
		Description: fmt.Sprintf("Interface %s has input errors: %d, output errors: %d, collisions: %d",
			errStats.InterfaceName, errStats.InputErrors, errStats.OutputErrors, errStats.Collisions),
		DeviceID:        errStats.DeviceID,
		InterfaceName:   errStats.InterfaceName,
		FirstSeen:       now,
		LastSeen:        now,
		OccurrenceCount: 1,
	}
}

// createWiFiProblem creates a NetworkProblem from a WiFiProblem.
func createWiFiProblem(wifiProb WiFiProblem) NetworkProblem {
	severity := ProblemSeverityInfo
	if wifiProb.IsRogue {
		severity = ProblemSeverityCritical
	} else if wifiProb.ProblemType == "weak_signal" {
		severity = ProblemSeverityWarning
	}
	return NetworkProblem{
		ID:              uuid.New().String(),
		Category:        ProblemCategoryWiFi,
		Type:            wifiProb.ProblemType,
		Severity:        severity,
		Status:          ProblemStatusActive,
		Title:           fmt.Sprintf("WiFi Issue: %s", wifiProb.ProblemType),
		Description:     formatWiFiProblemDescription(wifiProb),
		SSID:            wifiProb.SSID,
		BSSID:           wifiProb.BSSID,
		Channel:         wifiProb.Channel,
		CurrentValue:    float64(wifiProb.SignalDBm),
		FirstSeen:       wifiProb.FirstSeen,
		LastSeen:        wifiProb.LastSeen,
		OccurrenceCount: 1,
	}
}

// aggregateProblems converts specific issues into generic NetworkProblem entries.
func (d *ProblemDetector) aggregateProblems(result *ProblemDetectionResult) {
	now := time.Now()

	for _, conflict := range result.IPConflicts {
		result.Problems = append(result.Problems, createIPConflictProblem(conflict))
	}
	for _, mismatch := range result.DuplexMismatches {
		result.Problems = append(result.Problems, createDuplexMismatchProblem(mismatch))
	}
	for _, alert := range result.ResourceAlerts {
		result.Problems = append(result.Problems, createResourceAlertProblem(alert, now))
	}
	for _, errStats := range result.InterfaceErrors {
		result.Problems = append(result.Problems, createInterfaceErrorProblem(errStats, now))
	}
	for _, wifiProb := range result.WiFiProblems {
		result.Problems = append(result.Problems, createWiFiProblem(wifiProb))
	}
}

// formatWiFiProblemDescription creates a human-readable description.
func formatWiFiProblemDescription(p WiFiProblem) string {
	switch p.ProblemType {
	case "rogue_ap":
		return fmt.Sprintf("Unauthorized AP detected: SSID=%s, BSSID=%s on channel %d", p.SSID, p.BSSID, p.Channel)
	case "weak_signal":
		return fmt.Sprintf("Weak signal from %s (%s): %d dBm on channel %d", p.SSID, p.BSSID, p.SignalDBm, p.Channel)
	case "channel_interference":
		return fmt.Sprintf("Channel %d (%s) has %d overlapping APs", p.Channel, p.Band, p.CoChannelAPs)
	case "high_utilization":
		return fmt.Sprintf("Channel %d (%s) utilization at %.1f%%", p.Channel, p.Band, p.UtilizationPercent)
	default:
		return fmt.Sprintf("WiFi issue on channel %d: %s", p.Channel, p.ProblemType)
	}
}

// notifyListeners sends problem notifications to registered listeners.
func (d *ProblemDetector) notifyListeners(result *ProblemDetectionResult) {
	d.mu.RLock()
	listeners := make([]ProblemListener, len(d.listeners))
	copy(listeners, d.listeners)
	d.mu.RUnlock()

	for _, problem := range result.Problems {
		p := problem // Copy for closure
		for _, l := range listeners {
			l.OnProblemDetected(&p)
		}
	}
}

// GetLastScan returns the most recent scan result.
func (d *ProblemDetector) GetLastScan() *ProblemDetectionResult {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.lastScan
}

// GetActiveProblems returns currently active problems.
func (d *ProblemDetector) GetActiveProblems() []NetworkProblem {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.lastScan == nil {
		return nil
	}

	active := make([]NetworkProblem, 0)
	for _, p := range d.lastScan.Problems {
		if p.Status == ProblemStatusActive {
			active = append(active, p)
		}
	}
	return active
}

// GetSummary returns a summary of detected problems.
func (d *ProblemDetector) GetSummary() *ProblemSummary {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.lastScan == nil {
		return &ProblemSummary{
			BySeverity: make(map[string]int),
			ByCategory: make(map[string]int),
		}
	}

	summary := &ProblemSummary{
		BySeverity:   make(map[string]int),
		ByCategory:   make(map[string]int),
		LastScanTime: d.lastScanAt,
	}

	for _, p := range d.lastScan.Problems {
		if p.Status == ProblemStatusActive {
			summary.TotalActive++
			summary.BySeverity[string(p.Severity)]++
			summary.ByCategory[string(p.Category)]++
		}
	}

	return summary
}
