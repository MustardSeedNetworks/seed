package api

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/promote"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// promoteDiscoveredDevices turns the devices the last sweep found answering
// SNMP into polling targets, so the topology is built from what discovery saw
// rather than from a list typed in by hand (seed#2692).
//
// It only ever adds: an address that already has a target is left alone,
// whether that target is the operator's, enabled or disabled. Like the
// learner it runs after a sweep, because the sweep's result is its whole
// input, and it is best-effort — a promotion that cannot write must not make
// a scan look failed.
func (s *Server) promoteDiscoveredDevices(
	ctx context.Context, discovered []*discovery.DiscoveredDevice,
) {
	if s.pollingTargets == nil || s.db() == nil {
		return
	}

	devices := promotionViews(discovered)
	if len(devices) == 0 {
		return
	}

	logger := logging.FromContext(ctx)

	clientID, err := discovery.SingleClientID(ctx, clientIDLister{repo: s.db().Clients()})
	if err != nil {
		logger.WarnContext(ctx, "Discovered devices not promoted to polling targets",
			"error", err)
		return
	}

	candidates := make([]*polling.Target, 0, len(devices))
	for _, target := range promote.Targets(devices) {
		candidates = append(candidates, &polling.Target{
			ClientID:      clientID,
			Name:          target.Name,
			IPAddress:     target.IP,
			SNMPVersion:   target.SNMPVersion,
			CredentialsID: target.CredentialID,
			Enabled:       true,
		})
	}

	created, err := s.pollingTargets.CreateMissing(ctx, clientID, candidates)
	if err != nil {
		logger.WarnContext(ctx, "Some discovered devices could not be promoted",
			"error", err)
	}
	if created > 0 {
		// The live poller reads its target list once and reloads when the
		// list changes, exactly as the create handler does. Without this the
		// promoted targets sit in the table until the next restart, and the
		// topology the promotion exists to fill stays empty.
		s.reloadSNMPPoller(ctx)
		logger.InfoContext(ctx, "Promoted SNMP-answering devices to polling targets",
			"created", created, "answered", len(devices))
	}
}

// promotionViews reduces the discovered devices to what a polling target
// needs, skipping every device no SNMP credential answered.
//
// Both SNMP paths are read because both run: the profiler's own probe fills
// Profile.SNMPInfo, and the MIB collection that follows it fills SNMPData.
// Reading only one would promote nothing on a sweep that took the other.
func promotionViews(devices []*discovery.DiscoveredDevice) []promote.Device {
	views := make([]promote.Device, 0, len(devices))
	for _, device := range devices {
		if device == nil || device.IP == "" {
			continue
		}
		cred := deviceCredential(device)
		if cred.ID == "" {
			continue
		}
		views = append(views, promote.Device{
			IP:         device.IP,
			Name:       deviceName(device),
			Credential: cred,
		})
	}
	return views
}

// deviceCredential returns the credential this device answered, from whichever
// SNMP exchange reached it.
func deviceCredential(device *discovery.DiscoveredDevice) snmp.CredentialRef {
	if device.SNMPData != nil && device.SNMPData.Credential.ID != "" {
		return device.SNMPData.Credential
	}
	if device.Profile != nil && device.Profile.SNMPInfo != nil {
		return device.Profile.SNMPInfo.Credential
	}
	return snmp.CredentialRef{}
}

// deviceName is the name the operator will recognise in the target list. A
// device SNMP answered has a sysName; the rest is what the UI already shows.
func deviceName(device *discovery.DiscoveredDevice) string {
	if device.SNMPData != nil && device.SNMPData.System != nil &&
		device.SNMPData.System.SysName != "" {
		return device.SNMPData.System.SysName
	}
	if device.Profile != nil && device.Profile.SNMPInfo != nil &&
		device.Profile.SNMPInfo.SysName != "" {
		return device.Profile.SNMPInfo.SysName
	}
	if device.DisplayName != "" {
		return device.DisplayName
	}
	return device.Hostname
}

// afterSweep is the one observer the discovery service calls when a sweep
// finishes. Both consumers read the same result, and the service holds a
// single observer.
func (s *Server) afterSweep(ctx context.Context, discovered []*discovery.DiscoveredDevice) {
	s.learnTargetNetworks(ctx, discovered)
	s.promoteDiscoveredDevices(ctx, discovered)
}
