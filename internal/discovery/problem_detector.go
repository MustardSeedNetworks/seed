package discovery

// problem_detector.go implements automated network problem detection.
// It scans the discovery database for issues like duplicate IPs, duplex mismatches,
// high error rates, and WiFi problems.

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/krisarmstrong/seed/internal/logging"
)

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
func (d *ProblemDetector) ScanWiFi(ctx context.Context, scanResult *WiFiScanResult, authorizedSSIDs, authorizedBSSIDs []string) []WiFiProblem {
	if scanResult == nil {
		return nil
	}

	d.mu.RLock()
	thresholds := d.thresholds
	d.mu.RUnlock()

	var problems []WiFiProblem

	// Build authorized lookups
	authSSIDs := make(map[string]bool)
	for _, ssid := range authorizedSSIDs {
		authSSIDs[ssid] = true
	}
	authBSSIDs := make(map[string]bool)
	for _, bssid := range authorizedBSSIDs {
		authBSSIDs[strings.ToUpper(bssid)] = true
	}

	now := time.Now()

	// Check for rogue/unauthorized APs
	for _, ap := range scanResult.APs {
		normalizedBSSID := strings.ToUpper(ap.BSSID)

		// Check if AP is unauthorized
		if !authBSSIDs[normalizedBSSID] && !authSSIDs[ap.SSIDName] {
			// Only flag if the SSID matches an authorized one but BSSID doesn't
			// This could be a rogue AP impersonating a legit network
			if authSSIDs[ap.SSIDName] {
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
		}

		// Check for weak signal
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

	// Check for channel interference
	channelAPCounts := make(map[string]int) // key: "band-channel"
	for _, ap := range scanResult.APs {
		key := fmt.Sprintf("%s-%d", ap.Band, ap.Channel)
		channelAPCounts[key]++
	}

	for key, count := range channelAPCounts {
		if count > thresholds.MaxCoChannelAPs {
			parts := strings.SplitN(key, "-", 2)
			if len(parts) != 2 {
				continue
			}
			band := WiFiBand(parts[0])
			var channel int
			fmt.Sscanf(parts[1], "%d", &channel)

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

	// Check channel utilization
	for _, util := range scanResult.Utilization {
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
func (d *ProblemDetector) detectIPConflicts(ctx context.Context, devices []*DiscoveredDevice, result *ProblemDetectionResult) {
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
func (d *ProblemDetector) detectDuplexMismatches(ctx context.Context, devices []*DiscoveredDevice, result *ProblemDetectionResult, thresholds ProblemThresholds) {
	// Currently SNMP interfaces don't have duplex info in the standard MIBs
	// This would require vendor-specific MIBs or LLDP/CDP data
	// For now, we detect issues via high error rates in detectInterfaceErrors
}

// detectResourceThresholds checks device resources against thresholds.
// Note: SNMPFullData doesn't currently have CPU/Memory fields - this is a placeholder
// for when those MIBs are implemented in the SNMP collector.
func (d *ProblemDetector) detectResourceThresholds(ctx context.Context, devices []*DiscoveredDevice, result *ProblemDetectionResult, thresholds ProblemThresholds) {
	// Resource thresholds will be checked when CPU/Memory SNMP MIBs are collected
	// (HOST-RESOURCES-MIB, UCD-SNMP-MIB, etc.)
}

// detectInterfaceErrors checks for high error rates on interfaces.
func (d *ProblemDetector) detectInterfaceErrors(ctx context.Context, devices []*DiscoveredDevice, result *ProblemDetectionResult, thresholds ProblemThresholds) {
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
				InputErrors:   int64(iface.InErrors),
				OutputErrors:  int64(iface.OutErrors),
				RecordedAt:    now,
			}

			if int64(iface.InErrors) > thresholds.InputErrorsPerMin {
				hasErrors = true
			}
			if int64(iface.OutErrors) > thresholds.OutputErrorsPerMin {
				hasErrors = true
			}

			if hasErrors {
				result.InterfaceErrors = append(result.InterfaceErrors, stats)
			}
		}
	}
}

// aggregateProblems converts specific issues into generic NetworkProblem entries.
func (d *ProblemDetector) aggregateProblems(result *ProblemDetectionResult) {
	now := time.Now()

	// IP Conflicts
	for _, conflict := range result.IPConflicts {
		result.Problems = append(result.Problems, NetworkProblem{
			ID:              uuid.New().String(),
			Category:        ProblemCategoryIPConflict,
			Type:            "duplicate_ip",
			Severity:        ProblemSeverityCritical,
			Status:          ProblemStatusActive,
			Title:           fmt.Sprintf("Duplicate IP: %s", conflict.IPAddress),
			Description:     fmt.Sprintf("IP address %s is claimed by %d devices: %s", conflict.IPAddress, len(conflict.MACs), strings.Join(conflict.MACs, ", ")),
			IPAddress:       conflict.IPAddress,
			AffectedMACs:    strings.Join(conflict.MACs, ","),
			FirstSeen:       conflict.FirstSeen,
			LastSeen:        conflict.LastSeen,
			OccurrenceCount: 1,
		})
	}

	// Duplex Mismatches
	for _, mismatch := range result.DuplexMismatches {
		result.Problems = append(result.Problems, NetworkProblem{
			ID:              uuid.New().String(),
			Category:        ProblemCategoryDuplexMismatch,
			Type:            "duplex_mismatch",
			Severity:        ProblemSeverityWarning,
			Status:          ProblemStatusActive,
			Title:           fmt.Sprintf("Duplex Mismatch: %s", mismatch.InterfaceName),
			Description:     fmt.Sprintf("Interface %s is running in %s duplex mode with %d collisions", mismatch.InterfaceName, mismatch.LocalDuplex, mismatch.CollisionCount),
			DeviceID:        mismatch.DeviceID,
			InterfaceName:   mismatch.InterfaceName,
			FirstSeen:       mismatch.FirstSeen,
			LastSeen:        mismatch.LastSeen,
			OccurrenceCount: 1,
		})
	}

	// Resource Alerts
	for _, alert := range result.ResourceAlerts {
		result.Problems = append(result.Problems, NetworkProblem{
			ID:              uuid.New().String(),
			Category:        ProblemCategoryResourceUsage,
			Type:            fmt.Sprintf("high_%s", alert.ResourceType),
			Severity:        SeverityForResourceUsage(alert.CurrentValue, alert.Threshold),
			Status:          ProblemStatusActive,
			Title:           fmt.Sprintf("High %s Usage", strings.Title(alert.ResourceType)),
			Description:     fmt.Sprintf("%s usage at %.1f%% (threshold: %.1f%%)", strings.Title(alert.ResourceType), alert.CurrentValue, alert.Threshold),
			DeviceID:        alert.DeviceID,
			CurrentValue:    alert.CurrentValue,
			ThresholdValue:  alert.Threshold,
			Unit:            alert.Unit,
			FirstSeen:       now,
			LastSeen:        now,
			OccurrenceCount: 1,
		})
	}

	// Interface Errors
	for _, errStats := range result.InterfaceErrors {
		result.Problems = append(result.Problems, NetworkProblem{
			ID:              uuid.New().String(),
			Category:        ProblemCategoryInterfaceErrors,
			Type:            "interface_errors",
			Severity:        ProblemSeverityWarning,
			Status:          ProblemStatusActive,
			Title:           fmt.Sprintf("Interface Errors: %s", errStats.InterfaceName),
			Description:     fmt.Sprintf("Interface %s has input errors: %d, output errors: %d, collisions: %d", errStats.InterfaceName, errStats.InputErrors, errStats.OutputErrors, errStats.Collisions),
			DeviceID:        errStats.DeviceID,
			InterfaceName:   errStats.InterfaceName,
			FirstSeen:       now,
			LastSeen:        now,
			OccurrenceCount: 1,
		})
	}

	// WiFi Problems
	for _, wifiProb := range result.WiFiProblems {
		severity := ProblemSeverityInfo
		if wifiProb.IsRogue {
			severity = ProblemSeverityCritical
		} else if wifiProb.ProblemType == "weak_signal" {
			severity = ProblemSeverityWarning
		}

		result.Problems = append(result.Problems, NetworkProblem{
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
		})
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
