// Package capabilities reports what the running platform can actually do.
//
// Seed maintained two hand-written capability tables that disagreed with each
// other and with the code: HARDWARE.md's Platform Support Matrix, and a second
// one in seed#749's original body. HARDWARE.md listed macOS Wi-Fi scanning as
// Full while it shelled a binary Apple had removed (#2031), and macOS ARP table
// reading as Full while it returns nothing at all (#2272). Nothing detected
// either drift, because a table in a Markdown file cannot be wrong in a way a
// build notices.
//
// So the matrix lives here, in Go, and the document is generated from it. A
// third hand-written table would not have helped.
//
// This is a *platform* question, deliberately distinct from two neighbours:
//
//   - licence tier — "your tier does not include this" (internal/license,
//     requireFeature, the UI's RequireFeature/TierGate)
//   - privilege — "this process cannot open a raw socket" (the existing
//     icmpAvailable check)
//
// Three different messages with three different remedies. A level here answers
// only "can this operating system do this at all".
package capabilities

// Level is how well a platform supports a capability.
type Level string

const (
	// LevelFull is complete support through standard OS APIs.
	LevelFull Level = "full"
	// LevelPartial is reduced functionality through the APIs that exist.
	LevelPartial Level = "partial"
	// LevelLimited needs vendor-specific tools or drivers to work at all.
	LevelLimited Level = "limited"
	// LevelNone is not available through any standard API on this platform.
	LevelNone Level = "none"
)

// Capability is one row of the support matrix — one thing an operator can ask
// Seed to do.
type Capability string

// The matrix rows.
const (
	InterfaceListing  Capability = "interface_listing"
	StaticIP          Capability = "static_ip"
	DHCPConfig        Capability = "dhcp_config"
	MTUConfig         Capability = "mtu_config"
	LinkMonitoring    Capability = "link_monitoring"
	SpeedDuplex       Capability = "speed_duplex"
	WiFiScanning      Capability = "wifi_scanning"
	WiFiConnection    Capability = "wifi_connection"
	ARPTable          Capability = "arp_table"
	NDPDiscovery      Capability = "ndp_discovery"
	BluetoothScanning Capability = "bluetooth_scanning"
	GatewayDetection  Capability = "gateway_detection"
	DNSDetection      Capability = "dns_detection"
	DHCPLeaseInfo     Capability = "dhcp_lease_info"
	VLANDetection     Capability = "vlan_detection"
	VLANManagement    Capability = "vlan_management"
	CableDiagnostics  Capability = "cable_diagnostics"
	PHYInfo           Capability = "phy_info"
	OpticalMonitoring Capability = "optical_monitoring"
	DriverStatistics  Capability = "driver_statistics"
)

// titles are the human-readable names, shared by the API response and the
// generated document so the two cannot drift apart either.
func titles() map[Capability]string {
	return map[Capability]string{
		InterfaceListing:  "Interface listing",
		StaticIP:          "Static IP configuration",
		DHCPConfig:        "DHCP configuration",
		MTUConfig:         "MTU configuration",
		LinkMonitoring:    "Link status monitoring",
		SpeedDuplex:       "Speed/duplex detection",
		WiFiScanning:      "Wi-Fi scanning",
		WiFiConnection:    "Wi-Fi connect/disconnect",
		ARPTable:          "ARP table reading",
		NDPDiscovery:      "IPv6 NDP discovery",
		BluetoothScanning: "Bluetooth scanning",
		GatewayDetection:  "Gateway detection",
		DNSDetection:      "DNS server detection",
		DHCPLeaseInfo:     "DHCP lease info",
		VLANDetection:     "VLAN detection",
		VLANManagement:    "VLAN creation/deletion",
		CableDiagnostics:  "Cable diagnostics (TDR)",
		PHYInfo:           "PHY layer info",
		OpticalMonitoring: "Digital Optical Monitoring",
		DriverStatistics:  "Driver error counters",
	}
}

// order is the matrix's row order, kept stable so a generated document has no
// spurious diffs and a reader finds rows where they were last time.
func order() []Capability {
	return []Capability{
		InterfaceListing, StaticIP, DHCPConfig, MTUConfig, LinkMonitoring,
		SpeedDuplex, WiFiScanning, WiFiConnection, ARPTable, NDPDiscovery,
		BluetoothScanning, GatewayDetection, DNSDetection, DHCPLeaseInfo,
		VLANDetection, VLANManagement, CableDiagnostics, PHYInfo,
		OpticalMonitoring, DriverStatistics,
	}
}

// Title returns the human-readable name of a capability.
func Title(c Capability) string {
	if title, ok := titles()[c]; ok {
		return title
	}

	return string(c)
}

// Order returns the matrix row order.
func Order() []Capability {
	return order()
}

// Entry is one capability as reported to a caller.
type Entry struct {
	Capability Capability `json:"capability"`
	Title      string     `json:"title"`
	Level      Level      `json:"level"`
	// Note explains a level that is not Full, so a caller can say why rather
	// than only that. Empty for Full.
	Note string `json:"note,omitempty"`
}

// Report returns this platform's capabilities in matrix order.
func Report() []Entry {
	levels := platformLevels()
	notes := platformNotes()
	rows := order()

	out := make([]Entry, 0, len(rows))
	for _, capability := range rows {
		level, ok := levels[capability]
		if !ok {
			// A capability with no entry for this platform is unsupported
			// rather than silently absent: a missing row would read as "fine".
			level = LevelNone
		}
		out = append(out, Entry{
			Capability: capability,
			Title:      Title(capability),
			Level:      level,
			Note:       notes[capability],
		})
	}

	return out
}

// Supported reports whether the capability works at all here.
func Supported(c Capability) bool {
	return LevelOf(c) != LevelNone
}

// LevelOf returns this platform's level for one capability.
func LevelOf(c Capability) Level {
	level, ok := platformLevels()[c]
	if !ok {
		return LevelNone
	}

	return level
}

// Degraded returns the capabilities that are not fully supported here, in
// matrix order — what a caller warns about.
func Degraded() []Entry {
	var out []Entry
	for _, entry := range Report() {
		if entry.Level != LevelFull {
			out = append(out, entry)
		}
	}

	return out
}
