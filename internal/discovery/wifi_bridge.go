package discovery

// wifi_bridge.go connects the existing canopy/wifi scanner to the unified discovery
// types. This allows WiFi scan results to be stored with extended metadata,
// authorization tracking, and correlation with discovered devices.

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/krisarmstrong/seed/internal/canopy/wifi"
	"github.com/krisarmstrong/seed/internal/logging"
)

// WiFiBridge connects the canopy/wifi scanner to unified discovery.
// It converts wifi.ScannedNetwork results into extended WiFi types.
type WiFiBridge struct {
	mu              sync.RWMutex
	scanner         *wifi.Scanner
	manager         *wifi.Manager
	oui             *OUIDatabase
	config          *WiFiBridgeConfig
	lastScan        *WiFiScanResult
	lastScanTime    time.Time
	networks        map[string]*WiFiNetwork     // key: SSID
	accessPoints    map[string]*WiFiAccessPoint // key: BSSID
	authorizedSSIDs map[string]bool
	authorizedMACs  map[string]bool
}

// WiFiBridgeConfig configures the WiFi bridge behavior.
type WiFiBridgeConfig struct {
	// MinSignalDBm filters out APs below this signal strength
	MinSignalDBm int `json:"min_signal_dbm" yaml:"min_signal_dbm"`

	// AuthorizedSSIDs lists SSIDs to mark as authorized
	AuthorizedSSIDs []string `json:"authorized_ssids" yaml:"authorized_ssids"`

	// AuthorizedBSSIDs lists BSSIDs to mark as authorized
	AuthorizedBSSIDs []string `json:"authorized_bssids" yaml:"authorized_bssids"`

	// TrackHistory enables historical tracking of networks
	TrackHistory bool `json:"track_history" yaml:"track_history"`
}

// DefaultWiFiBridgeConfig returns sensible defaults.
func DefaultWiFiBridgeConfig() *WiFiBridgeConfig {
	return &WiFiBridgeConfig{
		MinSignalDBm:     -90,
		AuthorizedSSIDs:  []string{},
		AuthorizedBSSIDs: []string{},
		TrackHistory:     true,
	}
}

// NewWiFiBridge creates a new WiFi bridge.
func NewWiFiBridge(
	scanner *wifi.Scanner,
	manager *wifi.Manager,
	oui *OUIDatabase,
	config *WiFiBridgeConfig,
) *WiFiBridge {
	if config == nil {
		config = DefaultWiFiBridgeConfig()
	}

	authorizedSSIDs := make(map[string]bool)
	for _, ssid := range config.AuthorizedSSIDs {
		authorizedSSIDs[ssid] = true
	}

	authorizedMACs := make(map[string]bool)
	for _, mac := range config.AuthorizedBSSIDs {
		authorizedMACs[normalizeMAC(mac)] = true
	}

	return &WiFiBridge{
		scanner:         scanner,
		manager:         manager,
		oui:             oui,
		config:          config,
		networks:        make(map[string]*WiFiNetwork),
		accessPoints:    make(map[string]*WiFiAccessPoint),
		authorizedSSIDs: authorizedSSIDs,
		authorizedMACs:  authorizedMACs,
	}
}

// SetScanner updates the underlying WiFi scanner.
func (b *WiFiBridge) SetScanner(scanner *wifi.Scanner) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.scanner = scanner
}

// SetManager updates the underlying WiFi manager.
func (b *WiFiBridge) SetManager(manager *wifi.Manager) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.manager = manager
}

// SetAuthorizedSSIDs sets the list of authorized SSIDs.
func (b *WiFiBridge) SetAuthorizedSSIDs(ssids []string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.authorizedSSIDs = make(map[string]bool)
	for _, ssid := range ssids {
		b.authorizedSSIDs[ssid] = true
	}
}

// SetAuthorizedBSSIDs sets the list of authorized BSSIDs.
func (b *WiFiBridge) SetAuthorizedBSSIDs(bssids []string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.authorizedMACs = make(map[string]bool)
	for _, mac := range bssids {
		b.authorizedMACs[normalizeMAC(mac)] = true
	}
}

// Scan performs a WiFi scan using the underlying scanner and converts results.
func (b *WiFiBridge) Scan(ctx context.Context) (*WiFiScanResult, error) {
	b.mu.RLock()
	scanner := b.scanner
	config := b.config
	b.mu.RUnlock()

	if scanner == nil {
		return nil, ErrScannerNotAvailable
	}

	logger := logging.GetLogger()
	logger.InfoContext(ctx, "Starting WiFi scan via bridge")

	start := time.Now()

	// Perform scan using existing canopy/wifi scanner
	scannedNetworks, err := scanner.Scan()
	if err != nil {
		return nil, err
	}

	// Convert to extended types
	aps := b.convertScannedNetworks(scannedNetworks, config.MinSignalDBm)

	// Group APs into networks
	networks := b.groupAPsToNetworks(aps)

	// Calculate channel utilization
	utilization := b.calculateChannelUtilization(aps)

	// Get current interface info
	interfaceName := ""
	if b.manager != nil {
		if info := b.manager.GetInfo(); info != nil {
			interfaceName = info.SSID // Best we can get without interface name
		}
	}

	result := &WiFiScanResult{
		Networks:    networks,
		APs:         aps,
		Clients:     []WiFiClient{}, // Clients require monitor mode
		Utilization: utilization,
		ScanTime:    start,
		Interface:   interfaceName,
	}

	// Update cached state
	b.mu.Lock()
	b.lastScan = result
	b.lastScanTime = start

	// Update persistent AP/network tracking
	for i := range aps {
		b.accessPoints[aps[i].BSSID] = &aps[i]
	}
	for i := range networks {
		b.networks[networks[i].SSID] = &networks[i]
	}
	b.mu.Unlock()

	logger.InfoContext(ctx, "WiFi scan via bridge complete",
		"networks", len(networks),
		"aps", len(aps),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return result, nil
}

// GetLastScan returns the most recent scan result.
func (b *WiFiBridge) GetLastScan() *WiFiScanResult {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastScan
}

// GetNetworks returns all known networks.
func (b *WiFiBridge) GetNetworks() []WiFiNetwork {
	b.mu.RLock()
	defer b.mu.RUnlock()

	networks := make([]WiFiNetwork, 0, len(b.networks))
	for _, n := range b.networks {
		networks = append(networks, *n)
	}
	return networks
}

// GetAccessPoints returns all known access points.
func (b *WiFiBridge) GetAccessPoints() []WiFiAccessPoint {
	b.mu.RLock()
	defer b.mu.RUnlock()

	aps := make([]WiFiAccessPoint, 0, len(b.accessPoints))
	for _, ap := range b.accessPoints {
		aps = append(aps, *ap)
	}
	return aps
}

// GetStats returns WiFi discovery statistics.
func (b *WiFiBridge) GetStats() *WiFiDiscoveryStats {
	b.mu.RLock()
	lastScan := b.lastScan
	lastTime := b.lastScanTime
	b.mu.RUnlock()

	if lastScan == nil {
		return &WiFiDiscoveryStats{
			ChannelsByBand:    make(map[string]int),
			SecurityBreakdown: make(map[string]int),
			VendorBreakdown:   make(map[string]int),
		}
	}

	stats := &WiFiDiscoveryStats{
		TotalNetworks:     len(lastScan.Networks),
		TotalAPs:          len(lastScan.APs),
		TotalClients:      len(lastScan.Clients),
		ChannelsByBand:    make(map[string]int),
		SecurityBreakdown: make(map[string]int),
		VendorBreakdown:   make(map[string]int),
		LastScanTime:      lastTime,
	}

	for _, n := range lastScan.Networks {
		if n.IsHidden {
			stats.HiddenNetworks++
		}
		stats.SecurityBreakdown[string(n.SecurityType)]++
	}

	for _, ap := range lastScan.APs {
		if ap.IsAuthorized {
			stats.AuthorizedAPs++
		} else {
			stats.UnauthorizedAPs++
		}
		stats.ChannelsByBand[string(ap.Band)]++
		if ap.Vendor != "" {
			stats.VendorBreakdown[ap.Vendor]++
		}
	}

	return stats
}

// convertScannedNetworks converts canopy/wifi results to extended types.
func (b *WiFiBridge) convertScannedNetworks(scanned []*wifi.ScannedNetwork, minSignal int) []WiFiAccessPoint {
	b.mu.RLock()
	authorizedMACs := b.authorizedMACs
	authorizedSSIDs := b.authorizedSSIDs
	b.mu.RUnlock()

	now := time.Now()
	aps := make([]WiFiAccessPoint, 0, len(scanned))

	for _, sn := range scanned {
		if sn.Signal < minSignal {
			continue
		}

		// Determine band from channel
		band := ChannelToBand(sn.Channel)

		// Determine WiFi standards
		standards := determineWiFiStandards(sn.HTMode, band)

		ap := WiFiAccessPoint{
			ID:           uuid.New().String(),
			BSSID:        sn.BSSID,
			SSIDName:     sn.SSID,
			Channel:      sn.Channel,
			ChannelWidth: sn.ChannelWidth,
			FrequencyMHz: sn.Frequency,
			Band:         band,
			SignalDBm:    sn.Signal,
			NoiseDBm:     sn.NoiseFloor,
			WiFiStandard: standards,
			FirstSeen:    now,
			LastSeen:     now,
			Metadata: map[string]any{
				"security": sn.Security,
				"ht_mode":  sn.HTMode,
				"is_dfs":   sn.IsDFS,
				"snr":      sn.SNR,
			},
		}

		// Vendor lookup
		if b.oui != nil {
			ap.Vendor = b.oui.Lookup(sn.BSSID)
		}

		// Check authorization
		normalized := normalizeMAC(sn.BSSID)
		if authorizedMACs[normalized] || authorizedSSIDs[sn.SSID] {
			ap.IsAuthorized = true
		}

		aps = append(aps, ap)
	}

	return aps
}

// groupAPsToNetworks groups access points by SSID into WiFiNetwork entries.
func (b *WiFiBridge) groupAPsToNetworks(aps []WiFiAccessPoint) []WiFiNetwork {
	b.mu.RLock()
	authorizedSSIDs := b.authorizedSSIDs
	b.mu.RUnlock()

	ssidMap := make(map[string]*WiFiNetwork)
	now := time.Now()

	for _, ap := range aps {
		ssid := ap.SSIDName
		if ssid == "" {
			ssid = "(Hidden)"
		}

		network, exists := ssidMap[ssid]
		if !exists {
			network = &WiFiNetwork{
				ID:        uuid.New().String(),
				SSID:      ssid,
				IsHidden:  ssid == "(Hidden)",
				FirstSeen: now,
				LastSeen:  now,
				Metadata:  make(map[string]any),
			}
			ssidMap[ssid] = network
		}

		// Update network stats
		network.APCount++
		if ap.SignalDBm > network.BestSignal || network.BestSignal == 0 {
			network.BestSignal = ap.SignalDBm
		}

		// Use security from strongest AP
		if network.SecurityType == "" {
			if secStr, ok := ap.Metadata["security"].(string); ok {
				network.SecurityType = mapSecurityString(secStr)
			}
		}

		// Set authorization status
		if authorizedSSIDs[ssid] {
			network.AuthorizationStatus = WiFiAuthAuthorized
		} else if network.AuthorizationStatus == "" {
			network.AuthorizationStatus = WiFiAuthUnknown
		}
	}

	// Convert to slice
	networks := make([]WiFiNetwork, 0, len(ssidMap))
	for _, n := range ssidMap {
		networks = append(networks, *n)
	}

	return networks
}

// calculateChannelUtilization calculates per-channel metrics from AP list.
func (b *WiFiBridge) calculateChannelUtilization(aps []WiFiAccessPoint) []ChannelUtilization {
	channelMap := make(map[string]*ChannelUtilization) // key: "band-channel"
	now := time.Now()

	for _, ap := range aps {
		key := string(ap.Band) + "-" + string(rune(ap.Channel+'0'))

		util, exists := channelMap[key]
		if !exists {
			util = &ChannelUtilization{
				ID:           uuid.New().String(),
				Channel:      ap.Channel,
				Band:         ap.Band,
				FrequencyMHz: ap.FrequencyMHz,
				RecordedAt:   now,
			}
			channelMap[key] = util
		}

		util.APCount++
		util.ClientCount += ap.ClientCount
	}

	// Estimate utilization based on AP count
	for _, util := range channelMap {
		util.UtilizationPercent = float64(util.APCount) * 10
		if util.UtilizationPercent > 100 {
			util.UtilizationPercent = 100
		}
	}

	// Convert to slice
	result := make([]ChannelUtilization, 0, len(channelMap))
	for _, u := range channelMap {
		result = append(result, *u)
	}

	return result
}

// determineWiFiStandards determines WiFi standards from HT mode and band.
func determineWiFiStandards(htMode string, band WiFiBand) []string {
	var standards []string

	// Parse HT mode (e.g., "HT20", "VHT80", "HE160", "EHT320")
	switch {
	case containsAny(htMode, "EHT", "320"):
		standards = append(standards, "be") // WiFi 7
	case containsAny(htMode, "HE", "ax"):
		standards = append(standards, "ax") // WiFi 6
	case containsAny(htMode, "VHT", "ac"):
		standards = append(standards, "ac") // WiFi 5
	case containsAny(htMode, "HT"):
		standards = append(standards, "n") // WiFi 4
	}

	// Add legacy standards based on band
	switch band {
	case WiFiBand24GHz:
		if len(standards) == 0 {
			standards = append(standards, "n")
		}
		standards = append(standards, "g", "b")
	case WiFiBand5GHz:
		if len(standards) == 0 {
			standards = append(standards, "ac", "n")
		}
		standards = append(standards, "a")
	case WiFiBand6GHz:
		if len(standards) == 0 {
			standards = append(standards, "ax")
		}
	}

	if len(standards) == 0 {
		standards = []string{"unknown"}
	}

	return standards
}

// containsAny checks if s contains any of the substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// mapSecurityString maps security protocol string to WiFiSecurityType.
func mapSecurityString(sec string) WiFiSecurityType {
	sec = toLower(sec)
	switch {
	case containsAny(sec, "wpa3", "sae"):
		return WiFiSecurityWPA3
	case containsAny(sec, "wpa2"):
		return WiFiSecurityWPA2
	case containsAny(sec, "wpa"):
		return WiFiSecurityWPA
	case containsAny(sec, "wep"):
		return WiFiSecurityWEP
	default:
		return WiFiSecurityOpen
	}
}

// toLower is a simple ASCII lowercase function.
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// ErrScannerNotAvailable indicates the WiFi scanner is not available.
var ErrScannerNotAvailable = &scannerError{"WiFi scanner not available"}

type scannerError struct {
	msg string
}

func (e *scannerError) Error() string {
	return e.msg
}
