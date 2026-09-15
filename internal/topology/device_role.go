package topology

import "strings"

// Device roles. The spellings are deliberately identical to the ones
// internal/discovery's profiler produces, because the two subsystems
// classify the same devices from different evidence (SNMP here, open
// ports and banners there) and the UI branches on the string —
// ui/src/hooks/useDiscoveredDevices.ts matches "router", "switch",
// "server", "printer". TestDeviceRoleVocabularyMatchesDiscovery pins
// them equal. The two roles discovery has no notion of are new here.
const (
	RoleSwitch             = "switch"
	RoleRouter             = "router"
	RoleAccessPoint        = "access-point"
	RoleWirelessController = "wireless-controller"
	RoleFirewall           = "firewall"
	RolePrinter            = "printer"
	RoleServer             = "server"
	RoleUnknown            = "unknown"
)

// sysServices layer bits (RFC 1213 §6.4.7). The value is the sum of
// 2^(layer-1) for each OSI layer the agent implements.
const (
	svcDatalink     = 0x02 // layer 2 — bridging
	svcInternet     = 0x04 // layer 3 — routing
	svcEndToEnd     = 0x08 // layer 4
	svcApplications = 0x40 // layer 7
)

// descrRole is one prose rule: every needle must appear, lowercased.
type descrRole struct {
	role    string
	needles []string
}

// descrRoles is scanned in order, so the specific wins over the
// general: a wireless controller's sysDescr often also says "switch".
// Kept deliberately short — a keyword list is a guess about prose,
// and the bits below are the part that is specified. Declared as a
// function rather than a package var, the same way [neighborKinds] is.
func descrRoles() []descrRole {
	return []descrRole{
		{RoleWirelessController, []string{"wireless", "controller"}},
		{RoleWirelessController, []string{"wlan controller"}},
		{RoleAccessPoint, []string{"access point"}},
		{RoleFirewall, []string{"firewall"}},
		{RolePrinter, []string{"printer"}},
		{RoleRouter, []string{"router"}},
		{RoleSwitch, []string{"switch"}},
	}
}

// deviceRoleFrom classifies a node by what it does, not by who made
// it (seed#2456). Two pieces of evidence, in this order:
//
//  1. sysDescr prose, for the roles MIB-II cannot express. A wireless
//     controller, an access point and a firewall all report the same
//     service bits as the switch next to them.
//  2. sysServices bits, which are specified rather than guessed:
//     layer 2 implemented means the device bridges, so switch; layer 3
//     without layer 2 means it routes and does not bridge, so router;
//     layers 4 or 7 alone are a host running an agent, so server.
//
// The bits say which layers an agent *implements*, not which role it
// fills, which is why they cannot lead: the recorded corpus has a pure
// L2 access switch at 0x4E (L2+L3+L4+app) and an L3-capable one at
// 0x02. Reading "bridges at all" as switch is the conservative call —
// a device that bridges is drawn on the access layer, and a router
// that also bridges is doing both.
//
// Vendor is no longer a role and no longer lives in this field; see
// [vendorFromObjectID], whose answer goes to the node's metadata.
func deviceRoleFrom(sysDescr string, sysServices uint32) string {
	lower := strings.ToLower(sysDescr)
	for _, candidate := range descrRoles() {
		if containsAll(lower, candidate.needles) {
			return candidate.role
		}
	}
	switch {
	case sysServices&svcDatalink != 0:
		return RoleSwitch
	case sysServices&svcInternet != 0:
		return RoleRouter
	case sysServices&(svcEndToEnd|svcApplications) != 0:
		return RoleServer
	}
	if sysDescr == "" && sysServices == 0 {
		// The agent answered neither scalar. An empty role reads as
		// "not yet known" in the UI, where "unknown" reads as a
		// verdict seed never reached.
		return ""
	}
	return RoleUnknown
}

func containsAll(haystack string, needles []string) bool {
	for _, n := range needles {
		if !strings.Contains(haystack, n) {
			return false
		}
	}
	return true
}
