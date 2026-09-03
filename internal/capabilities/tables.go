package capabilities

// tables.go is the whole support matrix, for every platform, in one place.
//
// Deliberately not build-tagged. These are *declarations* about what each
// operating system can do, not code that calls it, so tagging them would buy no
// isolation and would cost the two things that matter: the generator could not
// render a column for a platform it is not running on, and a test could not
// check macOS's answers from Linux CI. A row only its own platform can read is
// a row that drifts.
//
// Runtime detection -- privileges, whether a binary exists -- belongs in the
// caller. `icmpAvailable` is that kind of check and stays where it is.

import "runtime"

// Platforms is the set the matrix covers, in column order.
func Platforms() []string {
	return []string{"linux", "darwin", "windows"}
}

// levelsByPlatform is the matrix. Every platform carries every capability --
// TestEveryPlatformCoversEveryCapability enforces it, so a new capability
// cannot be added for one OS and forgotten on the others.
func levelsByPlatform() map[string]map[Capability]Level {
	return map[string]map[Capability]Level{
		"linux": {
			InterfaceListing:  LevelFull,
			StaticIP:          LevelFull,
			DHCPConfig:        LevelFull,
			MTUConfig:         LevelFull,
			LinkMonitoring:    LevelFull,
			SpeedDuplex:       LevelFull,
			WiFiScanning:      LevelFull,
			WiFiConnection:    LevelFull,
			ARPTable:          LevelFull,
			NDPDiscovery:      LevelFull,
			BluetoothScanning: LevelFull,
			GatewayDetection:  LevelFull,
			DNSDetection:      LevelFull,
			DHCPLeaseInfo:     LevelFull,
			VLANDetection:     LevelFull,
			VLANManagement:    LevelFull,
			CableDiagnostics:  LevelPartial,
			PHYInfo:           LevelFull,
			OpticalMonitoring: LevelPartial,
		},
		"darwin": {
			InterfaceListing:  LevelFull,
			StaticIP:          LevelFull,
			DHCPConfig:        LevelFull,
			MTUConfig:         LevelFull,
			LinkMonitoring:    LevelFull,
			SpeedDuplex:       LevelPartial,
			WiFiScanning:      LevelFull,
			WiFiConnection:    LevelFull,
			ARPTable:          LevelFull,
			NDPDiscovery:      LevelFull,
			BluetoothScanning: LevelPartial,
			GatewayDetection:  LevelFull,
			DNSDetection:      LevelFull,
			DHCPLeaseInfo:     LevelFull,
			VLANDetection:     LevelPartial,
			VLANManagement:    LevelLimited,
			CableDiagnostics:  LevelNone,
			PHYInfo:           LevelPartial,
			OpticalMonitoring: LevelNone,
		},
		"windows": {
			InterfaceListing:  LevelFull,
			StaticIP:          LevelFull,
			DHCPConfig:        LevelFull,
			MTUConfig:         LevelFull,
			LinkMonitoring:    LevelFull,
			SpeedDuplex:       LevelFull,
			WiFiScanning:      LevelFull,
			WiFiConnection:    LevelFull,
			ARPTable:          LevelFull,
			NDPDiscovery:      LevelFull,
			BluetoothScanning: LevelLimited,
			GatewayDetection:  LevelFull,
			DNSDetection:      LevelFull,
			DHCPLeaseInfo:     LevelFull,
			VLANDetection:     LevelLimited,
			VLANManagement:    LevelNone,
			CableDiagnostics:  LevelNone,
			PHYInfo:           LevelPartial,
			OpticalMonitoring: LevelNone,
		},
	}
}

// notesByPlatform explains every level that is not Full. A degraded capability
// with no explanation tells an operator nothing they can act on.
//
// Two rows are corrections to what HARDWARE.md published, and both were found
// by running the product rather than reading the table:
//
//   - macOS ARP reading was published as Full while the reader asked the
//     routing socket for routes with no flags and got zero bytes. It is fixed
//     now and stands at Full on measurement (#2272) -- but it stood at Full
//     while returning nothing, which is the argument for generating this.
//   - macOS Wi-Fi scanning was Full while it shelled `airport`, which Apple
//     removed in macOS 27 (#2031). It is fixed now and stands at Full -- but it
//     stood at Full while broken, which is the argument for generating this.
//
// The two starred Linux rows lose their star and become Partial. TDR and
// optical monitoring need driver and transceiver support most hardware lacks;
// a Full with a footnote reads as Full.
func notesByPlatform() map[string]map[Capability]string {
	return map[string]map[Capability]string{
		"linux": {
			CableDiagnostics:  "Needs a NIC driver that implements ethtool's cable test.",
			OpticalMonitoring: "Needs an SFP/QSFP transceiver that reports diagnostics over ethtool.",
		},
		"darwin": {
			SpeedDuplex:       "Reports negotiated speed; duplex is not exposed.",
			BluetoothScanning: "Discovery only; no service enumeration.",
			VLANDetection:     "Detects tagged interfaces; does not enumerate the VLANs a trunk carries.",
			VLANManagement:    "Needs networksetup and an operator-created VLAN service.",
			PHYInfo:           "Link speed and media type only.",
			CableDiagnostics:  "No macOS API exposes TDR.",
			OpticalMonitoring: "No macOS API exposes transceiver diagnostics.",
		},
		"windows": {
			BluetoothScanning: "Needs a vendor stack; the built-in APIs do not expose discovery.",
			VLANDetection:     "Depends on the NIC vendor's driver exposing tagged interfaces.",
			VLANManagement:    "No supported API outside Hyper-V (#2104).",
			PHYInfo:           "Link speed only.",
			CableDiagnostics:  "No Windows API exposes TDR.",
			OpticalMonitoring: "No Windows API exposes transceiver diagnostics.",
		},
	}
}

// platformLevels is the running platform's table.
func platformLevels() map[Capability]Level { return LevelsFor(runtime.GOOS) }

// platformNotes is the running platform's notes.
func platformNotes() map[Capability]string { return NotesFor(runtime.GOOS) }

// LevelsFor returns one platform's table. An unknown GOOS reports nothing
// rather than an empty map, which a caller would read as "everything works".
func LevelsFor(goos string) map[Capability]Level {
	levels, ok := levelsByPlatform()[goos]
	if !ok {
		rows := order()
		unsupported := make(map[Capability]Level, len(rows))
		for _, capability := range rows {
			unsupported[capability] = LevelNone
		}

		return unsupported
	}

	return levels
}

// NotesFor returns one platform's notes.
func NotesFor(goos string) map[Capability]string { return notesByPlatform()[goos] }
