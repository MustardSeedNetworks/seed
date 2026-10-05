// Package settings is the application service for network-discovery settings and
// the additional-subnet list (ADR-0020 clean-hexagonal). It owns the read/merge/
// validate/persist logic the transport layer used to carry inline: the HTTP
// handler decodes the request, calls one method here, and encodes the result.
// Persistence and the live-scanner side effect are reached through the
// consumer-defined Store and SubnetSink ports, satisfied by adapters in the
// composition root (internal/app).
package settings

import (
	"errors"
	"net"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/discovery/learn"
)

// Sentinel errors the transport layer maps to HTTP status codes.
var (
	// ErrInvalidCIDR is returned when a subnet CIDR does not parse.
	ErrInvalidCIDR = errors.New("settings: invalid CIDR")
	// ErrSubnetExists is returned when adding a subnet whose CIDR is already present.
	ErrSubnetExists = errors.New("settings: subnet already exists")
	// ErrSubnetNotFound is returned when updating/deleting an absent subnet.
	ErrSubnetNotFound = errors.New("settings: subnet not found")
	// ErrInvalidPortPreset is returned for a port-scan preset that is not one of
	// the four the config defines.
	ErrInvalidPortPreset = errors.New("settings: invalid port-scan preset")
	// ErrNotLearned is returned when deciding on a network an operator entered.
	ErrNotLearned = errors.New("settings: subnet was not learned")
	// ErrInvalidDecision is returned for an answer other than added or dismissed.
	ErrInvalidDecision = errors.New("settings: invalid learned-network decision")
)

// Store reads and persists the network-discovery configuration. Discovery
// returns a copy of the current config; SaveDiscovery atomically replaces it and
// persists it to disk. The adapter owns the config lock and the on-disk save.
type Store interface {
	Discovery() config.NetworkDiscoveryConfig
	SaveDiscovery(config.NetworkDiscoveryConfig) error
}

// SubnetSink pushes the active (enabled) subnet set to the live device-discovery
// scanner so a config change reconfigures scanning. A nil-backed sink is a no-op.
type SubnetSink interface {
	SetTargetNetworks(cidrs []string) error
}

// OptionsApplier applies a discovery-options change to the running enumeration
// service so an options update takes effect without a restart. A nil-backed
// applier is a no-op.
type OptionsApplier interface {
	ReloadOptions() error
}

// Service is the network-discovery settings application service.
type Service struct {
	store   Store
	sink    SubnetSink
	applier OptionsApplier
}

// NewService builds the settings service over its ports.
func NewService(store Store, sink SubnetSink, applier OptionsApplier) *Service {
	return &Service{store: store, sink: sink, applier: applier}
}

// Settings returns the current network-discovery configuration.
func (s *Service) Settings() config.NetworkDiscoveryConfig {
	return s.store.Discovery()
}

// Update merges in onto the current configuration and persists it. The merge is
// field-specific (some fields are set unconditionally, others only when a
// positive/non-empty value is supplied) — preserved verbatim from the original
// handler so the wire contract is unchanged.
//
// The running scanner caches its options and its rescan ticker, so a change to
// either is applied the way SetOptions applies it. The drawer auto-saves the
// whole form on every edit, so an update that changes neither leaves the
// scanner alone rather than restarting it.
func (s *Service) Update(in Update) error {
	switch config.PortPreset(in.Options.PortScan.Preset) {
	case "", config.PortPresetCommon, config.PortPresetSecure,
		config.PortPresetInsecure, config.PortPresetCustom:
	default:
		return ErrInvalidPortPreset
	}
	before := s.store.Discovery()
	cur := before
	in.mergeInto(&cur)
	if err := s.store.SaveDiscovery(cur); err != nil {
		return err
	}
	if cur.Options == before.Options && cur.Timing == before.Timing {
		return nil
	}
	return s.applier.ReloadOptions()
}

// SetOptions replaces the discovery options wholesale, persists, then applies the
// change to the running enumeration service. Persisting before the apply lets the
// applier read the new options off the live config (Reload re-reads it) and means
// a successful write is durable even if the live reload reports an error — the
// operator's setting survives a restart. Returns the applier error if the reload
// fails after a successful save.
func (s *Service) SetOptions(opts config.DiscoveryOptions) error {
	cur := s.store.Discovery()
	cur.Options = opts
	if err := s.store.SaveDiscovery(cur); err != nil {
		return err
	}
	return s.applier.ReloadOptions()
}

// Subnets returns the configured target networks.
func (s *Service) Subnets() []config.SubnetConfig {
	return s.store.Discovery().TargetNetworks
}

// AddSubnet validates and appends a subnet, then persists and re-syncs the
// scanner. Returns ErrInvalidCIDR for a malformed CIDR or ErrSubnetExists for a
// duplicate.
func (s *Service) AddSubnet(in config.SubnetConfig) error {
	if _, _, err := net.ParseCIDR(in.CIDR); err != nil {
		return ErrInvalidCIDR
	}
	cur := s.store.Discovery()
	for _, existing := range cur.TargetNetworks {
		if existing.CIDR == in.CIDR {
			return ErrSubnetExists
		}
	}
	cur.TargetNetworks = append(cur.TargetNetworks, in)
	return s.saveAndSync(cur)
}

// Learn merges candidate networks into the configured target networks as
// disabled entries and returns how many were new (seed#2695).
//
// Three rules make it safe to call after every sweep. A CIDR already present
// is left exactly as it is, so an operator who enabled or renamed a learned
// network keeps that across the next sweep and an operator's own entry is
// never relabelled. A candidate whose CIDR does not parse is dropped rather
// than persisted, because everything downstream of TargetNetworks parses it.
// And when nothing is new the config is not written at all — the learner runs
// every rescan interval, and rewriting the file each minute to save the same
// bytes is a cost with no change behind it.
func (s *Service) Learn(candidates []learn.Candidate) (int, error) {
	cur := s.store.Discovery()
	known := make(map[string]bool, len(cur.TargetNetworks))
	for _, existing := range cur.TargetNetworks {
		known[existing.CIDR] = true
	}

	added := 0
	for _, candidate := range candidates {
		if known[candidate.CIDR] {
			continue
		}
		if _, _, err := net.ParseCIDR(candidate.CIDR); err != nil {
			continue
		}
		known[candidate.CIDR] = true
		cur.TargetNetworks = append(cur.TargetNetworks, config.SubnetConfig{
			CIDR:    candidate.CIDR,
			Name:    learnedName(candidate),
			Enabled: false,
			Learned: true,
		})
		added++
	}

	if added == 0 {
		return 0, nil
	}
	return added, s.saveAndSync(cur)
}

// learnedName describes where a candidate came from, so the operator deciding
// whether to sweep it can see which device named it and from which table.
func learnedName(candidate learn.Candidate) string {
	switch candidate.Source {
	case learn.SourceAddressTable:
		return "Learned from " + candidate.Router + " (interface addresses)"
	case learn.SourceRouteTable:
		return "Learned from " + candidate.Router + " (routing table)"
	case learn.SourceHostRoute:
		return "Learned from this host's route via " + candidate.Router
	default:
		return "Learned from " + candidate.Router
	}
}

// Pending returns the learned networks awaiting the operator's decision, in
// the order they were learned.
func (s *Service) Pending() []config.SubnetConfig {
	var pending []config.SubnetConfig
	for _, subnet := range s.store.Discovery().TargetNetworks {
		if subnet.AwaitingDecision() {
			pending = append(pending, subnet)
		}
	}
	return pending
}

// Decide records the operator's answer to the learned network cidr (seed#3108).
// Added switches it on; dismissed switches it off. Either answer is kept, so
// the network is not offered again and the learner, which skips every CIDR
// already listed, never re-adds it. An operator may change an earlier answer.
func (s *Service) Decide(cidr string, decision config.LearnedDecision) error {
	if decision != config.LearnedAdded && decision != config.LearnedDismissed {
		return ErrInvalidDecision
	}
	cur := s.store.Discovery()
	for i := range cur.TargetNetworks {
		subnet := &cur.TargetNetworks[i]
		if subnet.CIDR != cidr {
			continue
		}
		if !subnet.Learned {
			return ErrNotLearned
		}
		subnet.Decision = decision
		subnet.Enabled = decision == config.LearnedAdded
		return s.saveAndSync(cur)
	}
	return ErrSubnetNotFound
}

// UpdateSubnet renames/toggles the subnet matching in.CIDR. Returns ErrInvalidCIDR
// for a malformed CIDR or ErrSubnetNotFound if no subnet matches.
func (s *Service) UpdateSubnet(in config.SubnetConfig) error {
	if _, _, err := net.ParseCIDR(in.CIDR); err != nil {
		return ErrInvalidCIDR
	}
	cur := s.store.Discovery()
	found := false
	for i := range cur.TargetNetworks {
		if cur.TargetNetworks[i].CIDR == in.CIDR {
			cur.TargetNetworks[i].Name = in.Name
			cur.TargetNetworks[i].Enabled = in.Enabled
			// Switching a learned network on in Settings answers its prompt.
			if in.Enabled && cur.TargetNetworks[i].Learned {
				cur.TargetNetworks[i].Decision = config.LearnedAdded
			}
			found = true
			break
		}
	}
	if !found {
		return ErrSubnetNotFound
	}
	return s.saveAndSync(cur)
}

// DeleteSubnet removes the subnet with the given CIDR. Returns ErrSubnetNotFound
// if absent. Deleting a learned network forgets the operator's decision with
// it, so the network is offered again the next time it is learned.
func (s *Service) DeleteSubnet(cidr string) error {
	cur := s.store.Discovery()
	kept := make([]config.SubnetConfig, 0, len(cur.TargetNetworks))
	found := false
	for _, existing := range cur.TargetNetworks {
		if existing.CIDR == cidr {
			found = true
			continue
		}
		kept = append(kept, existing)
	}
	if !found {
		return ErrSubnetNotFound
	}
	cur.TargetNetworks = kept
	return s.saveAndSync(cur)
}

// saveAndSync persists the config and pushes the enabled subnet set to the live
// scanner. The scanner sync is best-effort — it never fails the save.
func (s *Service) saveAndSync(cur config.NetworkDiscoveryConfig) error {
	if err := s.store.SaveDiscovery(cur); err != nil {
		return err
	}
	enabled := make([]string, 0, len(cur.TargetNetworks))
	for _, sn := range cur.TargetNetworks {
		if sn.Enabled {
			enabled = append(enabled, sn.CIDR)
		}
	}
	_ = s.sink.SetTargetNetworks(enabled)
	return nil
}

// Update is the write model for network-discovery settings: the same field set
// the wire DTO carries, in milliseconds, so the merge rules (set-if-positive)
// match the original contract exactly. The transport layer maps its request DTO
// onto this domain input.
type Update struct {
	Enabled       bool
	ScanTimeoutMs int64
	AutoScan      bool
	OUIFilePath   string
	IPv6Enabled   bool

	Options        OptionsUpdate
	Timing         TimingUpdate
	Profiler       ProfilerUpdate
	Fingerprinting FingerprintingUpdate
}

// OptionsUpdate mirrors the discovery options write model.
type OptionsUpdate struct {
	PassiveProtocols PassiveProtocolsUpdate
	ARPScan          bool
	ICMPScan         bool
	PortScan         PortScanUpdate
	TCPProbe         TCPProbeUpdate
	Traceroute       bool
	SNMPQuery        bool
}

// PassiveProtocolsUpdate mirrors the passive-protocol toggles.
type PassiveProtocolsUpdate struct {
	LLDP bool
	CDP  bool
	EDP  bool
	NDP  bool
}

// PortScanUpdate mirrors the port-scan write model.
type PortScanUpdate struct {
	Enabled  bool
	Preset   string
	TCPPorts string
	UDPPorts string
}

// TCPProbeUpdate mirrors the TCP-probe write model.
type TCPProbeUpdate struct {
	TimeoutMs int64
	Workers   int
}

// TimingUpdate mirrors the discovery-timing write model.
type TimingUpdate struct {
	RescanIntervalMs int64
}

// ProfilerUpdate mirrors the profiler write model.
type ProfilerUpdate struct {
	Enabled       bool
	TimeoutMs     int64
	MaxConcurrent int
	QuickPorts    []int
}

// FingerprintingUpdate mirrors the fingerprinting write model.
type FingerprintingUpdate struct {
	Enabled       bool
	OSDetection   bool
	ServiceProbes bool
}

// mergeInto applies the update onto cur with the original field-specific rules:
// booleans are set unconditionally; counts/timeouts/intervals
// and paths are set only when a positive/non-empty value is supplied (treating
// zero/empty as "keep existing").
func (u Update) mergeInto(cur *config.NetworkDiscoveryConfig) {
	cur.Enabled = u.Enabled
	if u.ScanTimeoutMs > 0 {
		cur.ScanTimeout = msDuration(u.ScanTimeoutMs)
	}
	cur.AutoScan = u.AutoScan
	if u.OUIFilePath != "" {
		cur.OUIFilePath = u.OUIFilePath
	}
	cur.IPv6Enabled = u.IPv6Enabled

	u.Options.mergeInto(&cur.Options)
	u.Timing.mergeInto(&cur.Timing)
	u.Profiler.mergeInto(&cur.Profiler)
	cur.Fingerprinting.Enabled = u.Fingerprinting.Enabled
	cur.Fingerprinting.OSDetection = u.Fingerprinting.OSDetection
	cur.Fingerprinting.ServiceProbes = u.Fingerprinting.ServiceProbes
}

func (o OptionsUpdate) mergeInto(cur *config.DiscoveryOptions) {
	cur.PassiveProtocols.LLDP = o.PassiveProtocols.LLDP
	cur.PassiveProtocols.CDP = o.PassiveProtocols.CDP
	cur.PassiveProtocols.EDP = o.PassiveProtocols.EDP
	cur.PassiveProtocols.NDP = o.PassiveProtocols.NDP
	cur.ARPScan = o.ARPScan
	cur.ICMPScan = o.ICMPScan
	cur.Traceroute = o.Traceroute
	cur.SNMPQuery = o.SNMPQuery

	cur.PortScan.Enabled = o.PortScan.Enabled
	if o.PortScan.Preset != "" {
		cur.PortScan.Preset = config.PortPreset(o.PortScan.Preset)
	}
	if o.PortScan.TCPPorts != "" {
		cur.PortScan.TCPPorts = o.PortScan.TCPPorts
	}
	if o.PortScan.UDPPorts != "" {
		cur.PortScan.UDPPorts = o.PortScan.UDPPorts
	}
	if o.TCPProbe.TimeoutMs > 0 {
		cur.TCPProbe.Timeout = msDuration(o.TCPProbe.TimeoutMs)
	}
	if o.TCPProbe.Workers > 0 {
		cur.TCPProbe.Workers = o.TCPProbe.Workers
	}
}

func (t TimingUpdate) mergeInto(cur *config.DiscoveryTiming) {
	if t.RescanIntervalMs > 0 {
		cur.RescanInterval = msDuration(t.RescanIntervalMs)
	}
}

func (p ProfilerUpdate) mergeInto(cur *config.DeviceProfilerConfig) {
	cur.Enabled = p.Enabled
	if p.TimeoutMs > 0 {
		cur.Timeout = msDuration(p.TimeoutMs)
	}
	if p.MaxConcurrent > 0 {
		cur.MaxConcurrent = p.MaxConcurrent
	}
	if len(p.QuickPorts) > 0 {
		cur.QuickPorts = p.QuickPorts
	}
}

func msDuration(ms int64) time.Duration { return time.Duration(ms) * time.Millisecond }
