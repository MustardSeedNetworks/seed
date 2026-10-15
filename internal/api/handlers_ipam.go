package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	discoverysettings "github.com/MustardSeedNetworks/seed/internal/discovery/settings"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/ipam"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

// arpInUseWindow is how recently a router must have held an ARP entry for its
// address to count as in use. A day spans the common DHCP lease and keeps a
// host that sleeps overnight counted, while an address released last week
// stops holding a slot.
const arpInUseWindow = 24 * time.Hour

// IPAMSubnet is one network's address utilization.
type IPAMSubnet struct {
	CIDR   string      `json:"cidr"`
	Name   string      `json:"name,omitempty"`
	Source ipam.Source `json:"source"         jsonschema:"enum=local,enum=configured,enum=learned"`
	// Size is the number of assignable host addresses.
	Size           uint64  `json:"size"`
	Used           int     `json:"used"`
	Free           uint64  `json:"free"`
	UtilizationPct float64 `json:"utilizationPct"`
	// InUse lists the addresses seen in use, in address order.
	InUse []string `json:"inUse"`
}

// IPAMScope is one DHCP scope's address utilization.
type IPAMScope struct {
	Name  string `json:"name,omitempty"`
	Start string `json:"start"`
	End   string `json:"end"`
	// Size is the number of addresses the scope leases.
	Size           uint64  `json:"size"`
	Used           int     `json:"used"`
	Free           uint64  `json:"free"`
	UtilizationPct float64 `json:"utilizationPct"`
	// Exhaustion is "near" at 90% in use and "full" with nothing left.
	Exhaustion ipam.Exhaustion `json:"exhaustion" jsonschema:"enum=none,enum=near,enum=full"`
	// InUse lists the scope's addresses seen in use, in address order.
	InUse []string `json:"inUse"`
}

// IPAMSubnetsResponse is GET /api/v1/ipam/subnets.
type IPAMSubnetsResponse struct {
	Subnets []IPAMSubnet `json:"subnets"`
	Scopes  []IPAMScope  `json:"scopes"`
}

// IPAMScopeRequest is POST /api/v1/ipam/scopes: a DHCP server's address pool,
// the inclusive IPv4 range from start to end.
type IPAMScopeRequest struct {
	Name  string `json:"name,omitempty"`
	Start string `json:"start"`
	End   string `json:"end"`
}

func (s *Server) ipamRoutes() []route.Route {
	return []route.Route{
		{
			Path:    APIVersionPrefix + "/ipam/subnets",
			Handler: s.handleIPAMSubnets,
			Methods: []string{http.MethodGet},
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/ipam/scopes",
			Handler: s.handleIPAMScopes,
			Methods: []string{http.MethodPost, http.MethodDelete},
			Scope:   roles.Operator,
			Auth:    true,
			CSRF:    true,
		},
	}
}

// handleIPAMSubnets reports the utilization of every network Seed knows
// about: the active interface's subnet and the target networks, less the
// learned ones an operator dismissed, and of every DHCP scope (P-D5).
func (s *Server) handleIPAMSubnets(w http.ResponseWriter, r *http.Request) {
	var localSubnet string
	var devices []*discovery.DiscoveredDevice
	if s.deviceDiscovery() != nil {
		localSubnet, _ = s.deviceDiscovery().GetSubnetInfo()
		devices = s.deviceDiscovery().GetDevices()
	}
	inUse, err := s.ipamAddressesInUse(r.Context(), devices)
	if err != nil {
		writeTopologyError(w, r, err, "Failed to read ARP bindings")
		return
	}

	usages := ipam.Utilization(ipamNetworks(localSubnet, s.discoverySettings.Subnets()), inUse)
	subnets := make([]IPAMSubnet, 0, len(usages))
	for _, usage := range usages {
		subnets = append(subnets, IPAMSubnet{
			CIDR:           usage.Prefix.String(),
			Name:           usage.Name,
			Source:         usage.Source,
			Size:           usage.Size,
			Used:           len(usage.InUse),
			Free:           usage.Free(),
			UtilizationPct: usage.Percent(),
			InUse:          addrStrings(usage.InUse),
		})
	}

	scopeUsages := ipam.ScopeUtilization(s.discoverySettings.Scopes(), inUse)
	scopes := make([]IPAMScope, 0, len(scopeUsages))
	for _, usage := range scopeUsages {
		scopes = append(scopes, IPAMScope{
			Name:           usage.Name,
			Start:          usage.Start.String(),
			End:            usage.End.String(),
			Size:           usage.Size(),
			Used:           len(usage.InUse),
			Free:           usage.Free(),
			UtilizationPct: usage.Percent(),
			Exhaustion:     usage.Exhaustion(),
			InUse:          addrStrings(usage.InUse),
		})
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK,
		IPAMSubnetsResponse{Subnets: subnets, Scopes: scopes})
}

// handleIPAMScopes adds a DHCP scope (POST) or removes the one starting at
// the start query parameter (DELETE).
func (s *Server) handleIPAMScopes(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	successMsg := "DHCP scope deleted"
	var err error
	if r.Method == http.MethodPost {
		var req IPAMScopeRequest
		if !decodeJSONStrict(w, r, &req, MaxBodySizeJSON) {
			return
		}
		successMsg = "DHCP scope added"
		err = s.discoverySettings.AddScope(req.Name, req.Start, req.End)
	} else {
		err = s.discoverySettings.DeleteScope(r.URL.Query().Get("start"))
	}

	switch {
	case errors.Is(err, ipam.ErrInvalidScope):
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest,
			"A DHCP scope is two IPv4 addresses, start first", "")
	case errors.Is(err, discoverysettings.ErrScopeOverlaps):
		sendErrorResponseWithDetails(w, logger, http.StatusConflict, ErrCodeConflict,
			"DHCP scope overlaps another", "")
	case errors.Is(err, discoverysettings.ErrScopeNotFound):
		sendErrorResponseWithDetails(w, logger, http.StatusNotFound, ErrCodeNotFound,
			"DHCP scope not found", "")
	case err != nil:
		logger.ErrorContext(r.Context(), "Failed to save DHCP scope change", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError, ErrCodeInternal,
			i18n.FromRequest(r).T("errors.settings.saveFailed"), "")
	default:
		sendJSONResponse(w, logger, http.StatusOK, StatusMessageResponse{
			Status: statusSuccess, Message: successMsg,
		})
	}
}

// alertExhaustedScopes raises a dhcp.scope.exhaustion alert for each DHCP
// scope that climbed to near exhaustion or ran out since the last sweep.
func (s *Server) alertExhaustedScopes(ctx context.Context, devices []*discovery.DiscoveredDevice) {
	if s.alertWriter == nil {
		return
	}
	inUse, err := s.ipamAddressesInUse(ctx, devices)
	if err != nil {
		return
	}
	usages := ipam.ScopeUtilization(s.discoverySettings.Scopes(), inUse)
	for _, usage := range s.dhcpScopeWatch.Rising(usages) {
		if createErr := s.alertWriter.Create(ctx, scopeExhaustionAlert(usage)); createErr != nil {
			logging.FromContext(ctx).WarnContext(ctx, "raise DHCP scope exhaustion alert failed",
				"start", usage.Start, "end", usage.End, "error", createErr)
		}
	}
}

func scopeExhaustionAlert(usage ipam.ScopeUsage) *alerts.Alert {
	evidence, _ := json.Marshal(alerts.DHCPScopeEvidence{
		Start: usage.Start.String(), End: usage.End.String(), Size: usage.Size(),
		Used: len(usage.InUse), UsedPercent: usage.Percent(), Exhaustion: string(usage.Exhaustion()),
	})
	span := usage.Start.String() + "-" + usage.End.String()
	label := span
	if usage.Name != "" {
		label = usage.Name + " (" + span + ")"
	}
	alert := &alerts.Alert{
		Type:     alerts.TypeConnectivity,
		Severity: alerts.SeverityWarning,
		Title:    fmt.Sprintf("DHCP scope %s is %.0f%% in use", label, usage.Percent()),
		Message: fmt.Sprintf("%d of the %d addresses in DHCP scope %s were seen in use; %d are left.",
			len(usage.InUse), usage.Size(), label, usage.Free()),
		Source:   span,
		Rule:     alerts.RuleDHCPScopeExhaustion,
		Metadata: string(evidence),
	}
	if usage.Exhaustion() == ipam.ExhaustionFull {
		alert.Severity = alerts.SeverityCritical
		alert.Title = fmt.Sprintf("DHCP scope %s has no address left", label)
	}
	return alert
}

// ipamAddressesInUse is every address seen in use: the devices' own and
// SNMP-walked addresses, and the routers' ARP entries of the last day.
func (s *Server) ipamAddressesInUse(ctx context.Context, devices []*discovery.DiscoveredDevice) ([]netip.Addr, error) {
	bindings, err := s.topologyQueries.ARP(ctx, topology.ARPListOptions{
		Since: time.Now().Add(-arpInUseWindow),
	})
	if err != nil {
		logging.FromContext(ctx).ErrorContext(ctx,
			"list topology_arp_bindings for IPAM failed", "error", err)
		return nil, err
	}
	return addressesInUse(devices, bindings), nil
}

func addrStrings(addrs []netip.Addr) []string {
	out := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		out = append(out, addr.String())
	}
	return out
}

// ipamNetworks lists the active interface's subnet first, so a target network
// that repeats it is reported as local. A learned network the operator
// dismissed is left out.
func ipamNetworks(localSubnet string, targets []config.SubnetConfig) []ipam.Network {
	var networks []ipam.Network
	if prefix, err := netip.ParsePrefix(localSubnet); err == nil {
		networks = append(networks, ipam.Network{Prefix: prefix, Source: ipam.SourceLocal})
	}
	for _, subnet := range targets {
		prefix, err := netip.ParsePrefix(subnet.CIDR)
		if err != nil || subnet.Decision == config.LearnedDismissed {
			continue
		}
		source := ipam.SourceConfigured
		if subnet.Learned {
			source = ipam.SourceLearned
		}
		networks = append(networks, ipam.Network{Prefix: prefix, Name: subnet.Name, Source: source})
	}
	return networks
}

// addressesInUse is every address seen in use: each swept device's own, the
// interface addresses of the devices an SNMP walk reached, and the routers'
// recent ARP entries.
func addressesInUse(devices []*discovery.DiscoveredDevice, bindings []*topology.ARPBinding) []netip.Addr {
	var out []netip.Addr
	add := func(address string) {
		if addr, err := netip.ParseAddr(address); err == nil {
			out = append(out, addr)
		}
	}
	for _, device := range devices {
		add(device.IP)
		if device.SNMPData != nil {
			for _, address := range device.SNMPData.IPAddresses {
				add(address.Address)
			}
		}
	}
	for _, binding := range bindings {
		add(binding.IPAddress)
	}
	return out
}
