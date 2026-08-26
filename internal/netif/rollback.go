package netif

import (
	"errors"
	"fmt"
	"net/netip"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
)

// Applying a static IP is several independent changes in sequence — address,
// then gateway, then DNS — and the platform implementations return on the first
// error. That leaves the interface half-configured: on a failure at the gateway
// step the box sits on its new address with its old default route, which is
// precisely the "unreachable, needs physical access" case #50 describes.
//
// Capturing the previous configuration first, and putting it back when any step
// fails, removes that. It is not the connectivity-verify-and-auto-revert the
// issue also asks for — that needs a decision about what "reachable" means and
// a timer that survives losing the caller's own connection — but it removes the
// worst case and is testable without touching a real interface.

// configSnapshotter captures an interface's current configuration.
//
// A seam, so the revert path can be exercised against a fake. Reconfiguring a
// live NIC to test a rollback is how a test takes a machine off the network.
type configSnapshotter interface {
	Snapshot(iface string) (*StaticIPConfig, error)
}

// configApplier applies a configuration to an interface. The production
// implementation is the platform-specific configureStaticIPPlatform.
type configApplier interface {
	Apply(iface string, cfg *StaticIPConfig) error
}

// systemSnapshotter reads the live configuration by composing the read paths
// that already exist: the address from the interface itself, the default route
// from internal/diagnostics/gateway, and the resolvers from
// internal/diagnostics/dns. Nothing here is new platform code.
type systemSnapshotter struct{ manager *Manager }

// Snapshot captures what ConfigureStaticIP is about to overwrite.
func (s systemSnapshotter) Snapshot(iface string) (*StaticIPConfig, error) {
	info, err := s.manager.GetInterface(iface)
	if err != nil {
		return nil, fmt.Errorf("snapshot %s: %w", iface, err)
	}

	address, netmask := splitFirstCIDR(info.Addresses)
	if address == "" {
		// An interface with no address has nothing to restore, and reverting to
		// "no address" would be a change of its own rather than a revert.
		return nil, fmt.Errorf("snapshot %s: interface has no IPv4 address", iface)
	}

	return &StaticIPConfig{
		Address: address,
		Netmask: netmask,
		Gateway: defaultGatewayFor(iface),
		DNS:     dns.GetSystemDNS(),
	}, nil
}

// platformApplier is the production configApplier.
type platformApplier struct{}

// Apply calls the platform implementation.
func (platformApplier) Apply(iface string, cfg *StaticIPConfig) error {
	return configureStaticIPPlatform(iface, cfg)
}

// applyWithRollback applies cfg and restores prev if the apply fails.
//
// The apply error is what the caller wants; the rollback outcome is context. A
// failed rollback is joined onto it rather than replacing it, because "your
// change failed AND the interface is now in an unknown state" is a materially
// different situation from "your change failed" and the operator has to be able
// to tell them apart.
func applyWithRollback(
	applier configApplier, iface string, cfg, prev *StaticIPConfig,
) error {
	applyErr := applier.Apply(iface, cfg)
	if applyErr == nil {
		return nil
	}
	if prev == nil {
		return applyErr
	}

	if revertErr := applier.Apply(iface, prev); revertErr != nil {
		return errors.Join(applyErr, fmt.Errorf(
			"rollback to the previous configuration also failed, so %s is in an "+
				"indeterminate state: %w", iface, revertErr))
	}
	return errors.Join(applyErr, fmt.Errorf(
		"rolled %s back to its previous configuration", iface))
}

// splitFirstCIDR returns the address and dotted netmask of the first IPv4
// entry, which is the form the platform apply functions expect.
//
// InterfaceInfo.Addresses holds [net.Addr] strings — "192.168.1.10/24" — while
// StaticIPConfig wants the address and a dotted mask separately.
func splitFirstCIDR(addresses []string) (string, string) {
	for _, addr := range addresses {
		prefix, err := netip.ParsePrefix(addr)
		if err != nil || !prefix.Addr().Is4() {
			continue
		}
		return prefix.Addr().String(), cidrToNetmask(prefix.Bits())
	}
	return "", ""
}

// defaultGatewayFor returns the default route through iface, or "" if the
// interface has none. A missing gateway is not an error: an interface can
// legitimately have an address and no default route.
func defaultGatewayFor(iface string) string {
	routes, err := gateway.GetAllRoutes()
	if err != nil {
		return ""
	}
	for _, route := range routes {
		if route.Interface == iface && route.Gateway != "" {
			return route.Gateway
		}
	}
	return ""
}
