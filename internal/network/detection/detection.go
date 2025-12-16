// Package detection provides intelligent network interface auto-detection with scoring.
// Analyzes interfaces based on link status, speed, chipset quality, and capabilities
// to automatically select the optimal interface for network diagnostics.
package detection

import (
	"log"
	"net"
	"sort"
	"strings"
)

// Interface type constants.
const (
	ifTypeEthernet = "ethernet"
	ifTypeWifi     = "wifi"
	ifTypeFiber    = "fiber"
	ifTypeVirtual  = "virtual"
	ifTypeOther    = "other"
)

// InterfaceScore represents a scored network interface with metadata.
type InterfaceScore struct {
	Name           string   `json:"name"`           // System interface name (e.g., "enp3s0")
	FriendlyName   string   `json:"friendlyName"`   // Human-readable name (e.g., "Intel I225-V 2.5GbE")
	Description    string   `json:"description"`    // Brief description (e.g., "2.5 Gigabit Ethernet")
	Score          int      `json:"score"`          // Computed score for ranking
	LinkStatus     bool     `json:"linkStatus"`     // Physical link detected
	Speed          int64    `json:"speed"`          // Speed in bits per second
	SpeedDisplay   string   `json:"speedDisplay"`   // Human-readable speed (e.g., "2.5 Gbps")
	ChipsetVendor  string   `json:"chipsetVendor"`  // NIC vendor (e.g., "Intel")
	ChipsetModel   string   `json:"chipsetModel"`   // NIC model (e.g., "I225-V")
	ChipsetQuality int      `json:"chipsetQuality"` // Quality score 1-100
	HasTDR         bool     `json:"hasTDR"`         // Time Domain Reflectometry support
	HasDOM         bool     `json:"hasDOM"`         // Digital Optical Monitoring (SFP+)
	Type           string   `json:"type"`           // "ethernet", "wifi", "fiber", "virtual"
	HasIP          bool     `json:"hasIP"`          // Has routable IP address
	Addresses      []string `json:"addresses"`      // IP addresses assigned
}

// Detector provides interface detection and scoring functionality.
type Detector struct {
	chipsetDB *ChipsetDatabase
}

// NewDetector creates a new interface detector.
func NewDetector() *Detector {
	return &Detector{
		chipsetDB: NewChipsetDatabase(),
	}
}

// DetectAll discovers and scores all network interfaces.
// Returns interfaces sorted by score (highest first).
func (d *Detector) DetectAll() ([]InterfaceScore, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	scores := make([]InterfaceScore, 0, len(ifaces))
	for _, iface := range ifaces {
		// Skip loopback
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		score := d.ScoreInterface(iface)
		scores = append(scores, score)
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	return scores, nil
}

// DetectBest returns the highest-scoring interface.
// Returns (nil, nil) when no interfaces are found - this is not an error condition.
func (d *Detector) DetectBest() (*InterfaceScore, error) {
	scores, err := d.DetectAll()
	if err != nil {
		return nil, err
	}

	if len(scores) == 0 {
		//nolint:nilnil // No interface found is a valid state, not an error
		return nil, nil
	}

	return &scores[0], nil
}

// ScoreInterface computes a score for a single interface.
func (d *Detector) ScoreInterface(iface net.Interface) InterfaceScore {
	score := InterfaceScore{
		Name: iface.Name,
		Type: detectType(iface.Name),
	}

	// Determine link status and addresses
	score.LinkStatus = iface.Flags&net.FlagRunning != 0
	addrs, err := iface.Addrs()
	if err != nil {
		log.Printf("Warning: failed to get addresses for %s: %v", iface.Name, err)
	}
	for _, addr := range addrs {
		score.Addresses = append(score.Addresses, addr.String())
	}
	score.HasIP = hasRoutableAddress(score.Addresses)

	// Get speed (platform-specific)
	score.Speed = getInterfaceSpeed(iface.Name)
	score.SpeedDisplay = formatSpeed(score.Speed)

	// Identify chipset
	chipset := d.chipsetDB.IdentifyByInterface(iface.Name, iface.HardwareAddr.String())
	if chipset != nil {
		score.ChipsetVendor = chipset.Vendor
		score.ChipsetModel = chipset.Model
		score.ChipsetQuality = chipset.Quality
		score.HasTDR = chipset.HasTDR
		score.HasDOM = chipset.HasDOM
	}

	// Generate friendly name and description
	score.FriendlyName = d.generateFriendlyName(&score)
	score.Description = d.generateDescription(&score)

	// Calculate final score
	score.Score = d.calculateScore(&score)

	return score
}

// speedThreshold defines a speed threshold and its associated score bonus.
type speedThreshold struct {
	minSpeed int64
	bonus    int
}

// speedBonuses maps interface speeds to score bonuses (sorted highest first).
var speedBonuses = []speedThreshold{
	{100_000_000_000, 500}, // 100G
	{40_000_000_000, 450},  // 40G
	{25_000_000_000, 425},  // 25G
	{10_000_000_000, 400},  // 10G
	{5_000_000_000, 350},   // 5G
	{2_500_000_000, 300},   // 2.5G
	{1_000_000_000, 200},   // 1G
	{100_000_000, 100},     // 100M
}

// calculateSpeedBonus returns the score bonus for a given interface speed.
func calculateSpeedBonus(speed int64) int {
	for _, t := range speedBonuses {
		if speed >= t.minSpeed {
			return t.bonus
		}
	}
	return 0
}

// calculateScore computes the ranking score for an interface.
func (d *Detector) calculateScore(s *InterfaceScore) int {
	if s.Type == ifTypeVirtual {
		return -1000
	}

	score := 0
	if s.LinkStatus {
		score += 1000
	}
	if s.HasIP {
		score += 500
	}
	if s.HasTDR {
		score += 1000
	}
	if s.HasDOM {
		score += 500
	}

	score += calculateSpeedBonus(s.Speed)
	score += s.ChipsetQuality

	switch s.Type {
	case ifTypeEthernet:
		score += 100
	case ifTypeWifi:
		score += 50
	case ifTypeFiber:
		score += 150
	}

	return score
}

// generateFriendlyName creates a human-readable interface name.
func (d *Detector) generateFriendlyName(s *InterfaceScore) string {
	if s.ChipsetVendor != "" && s.ChipsetModel != "" {
		return s.ChipsetVendor + " " + s.ChipsetModel
	}

	// Fallback to generic name based on type and speed
	switch s.Type {
	case ifTypeEthernet:
		if s.SpeedDisplay != "" {
			return s.SpeedDisplay + " Ethernet"
		}
		return "Ethernet Adapter"
	case ifTypeWifi:
		return "WiFi Adapter"
	case ifTypeFiber:
		return "Fiber Adapter"
	default:
		return s.Name
	}
}

// generateDescription creates a brief description of the interface.
func (d *Detector) generateDescription(s *InterfaceScore) string {
	parts := []string{}

	if s.SpeedDisplay != "" {
		parts = append(parts, s.SpeedDisplay)
	}

	switch s.Type {
	case ifTypeEthernet:
		parts = append(parts, "Ethernet")
	case ifTypeWifi:
		parts = append(parts, "WiFi")
	case ifTypeFiber:
		parts = append(parts, "Fiber")
	}

	if s.HasTDR {
		parts = append(parts, "with TDR")
	}

	if len(parts) == 0 {
		return "Network Interface"
	}

	return strings.Join(parts, " ")
}

// detectType determines interface type from name patterns.
func detectType(name string) string {
	// Virtual interfaces
	virtualPrefixes := []string{"docker", "br-", "veth", "virbr", "tun", "tap", "vnet", "vmnet", "vboxnet", "utun"}
	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(name, prefix) {
			return ifTypeVirtual
		}
	}

	// WiFi interfaces
	wifiPrefixes := []string{"wlan", "wlp", "wifi", "ath", "ra", "wl"}
	for _, prefix := range wifiPrefixes {
		if strings.HasPrefix(name, prefix) {
			return ifTypeWifi
		}
	}

	// Fiber patterns (often have sfp or xfp in name, or high-speed prefixes)
	if strings.Contains(name, "sfp") || strings.Contains(name, "xfp") {
		return ifTypeFiber
	}

	// Default to ethernet for physical interfaces
	ethPrefixes := []string{"eth", "enp", "ens", "eno", "em", "en"}
	for _, prefix := range ethPrefixes {
		if strings.HasPrefix(name, prefix) {
			return ifTypeEthernet
		}
	}

	return ifTypeOther
}

// hasRoutableAddress checks if any address is routable.
func hasRoutableAddress(addresses []string) bool {
	for _, addr := range addresses {
		// Parse CIDR notation
		ipStr := addr
		if idx := strings.Index(addr, "/"); idx != -1 {
			ipStr = addr[:idx]
		}

		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		// Skip loopback and link-local
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			continue
		}

		return true
	}
	return false
}

// formatSpeed converts bits per second to human-readable format.
func formatSpeed(bps int64) string {
	switch {
	case bps >= 100_000_000_000:
		return "100 Gbps"
	case bps >= 40_000_000_000:
		return "40 Gbps"
	case bps >= 25_000_000_000:
		return "25 Gbps"
	case bps >= 10_000_000_000:
		return "10 Gbps"
	case bps >= 5_000_000_000:
		return "5 Gbps"
	case bps >= 2_500_000_000:
		return "2.5 Gbps"
	case bps >= 1_000_000_000:
		return "1 Gbps"
	case bps >= 100_000_000:
		return "100 Mbps"
	case bps >= 10_000_000:
		return "10 Mbps"
	case bps > 0:
		return "< 10 Mbps"
	default:
		return ""
	}
}
