// Package promote turns the devices a sweep found answering SNMP into the
// polling targets the topology is built from (seed#2692).
//
// Before this, discovery and polling shared no path: discovery found 87
// devices and the topology said "0 nodes — add a polling target" until 74
// targets had been POSTed by hand. Discovery already knows the two facts a
// target needs — the address, and which stored credential the device answered
// — so the operator typing them back in is the whole defect.
//
// The rules are deliberately one-directional: promotion only ever ADDS. It
// never edits or deletes a target, and an address that already has one is
// skipped, enabled or not. An operator who does not want a device polled disables
// its target, which is the control that already exists; deleting a target for
// a device that still answers SNMP gets it back on the next sweep, the same
// way a handheld tester re-lists a device it can still see.
package promote

import "github.com/MustardSeedNetworks/seed/internal/protocols/snmp"

// Device is one discovered device, narrowed to what a polling target needs.
type Device struct {
	// IP is the address to poll.
	IP string
	// Name is the device's best display name, empty when nothing named it.
	Name string
	// Credential names the stored credential this device answered.
	Credential snmp.CredentialRef
}

// Target is one polling target to create. It carries no id or client: those
// belong to the caller that persists it.
type Target struct {
	IP           string
	Name         string
	CredentialID string
	SNMPVersion  string
}

// Targets returns a target for every device that answered SNMP, in the order
// the devices were discovered.
//
// A device reached twice in one sweep yields one target: polling_targets has
// no unique index on the address, so a duplicate here is a duplicate row that
// polls the device twice. Whether an address already HAS a target is not
// decided here — that is a question about the table, and it is answered where
// the rows are written, under the lock that makes the answer hold.
func Targets(devices []Device) []Target {
	seen := make(map[string]struct{}, len(devices))

	targets := make([]Target, 0, len(devices))
	for _, device := range devices {
		if device.IP == "" || device.Credential.ID == "" || device.Credential.Version == "" {
			continue
		}
		if _, known := seen[device.IP]; known {
			continue
		}
		seen[device.IP] = struct{}{}

		name := device.Name
		if name == "" {
			name = device.IP
		}
		targets = append(targets, Target{
			IP:           device.IP,
			Name:         name,
			CredentialID: device.Credential.ID,
			SNMPVersion:  device.Credential.Version,
		})
	}
	return targets
}
