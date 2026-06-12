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
)

// Sentinel errors the transport layer maps to HTTP status codes.
var (
	// ErrInvalidCIDR is returned when a subnet CIDR does not parse.
	ErrInvalidCIDR = errors.New("settings: invalid CIDR")
	// ErrSubnetExists is returned when adding a subnet whose CIDR is already present.
	ErrSubnetExists = errors.New("settings: subnet already exists")
	// ErrSubnetNotFound is returned when updating/deleting an absent subnet.
	ErrSubnetNotFound = errors.New("settings: subnet not found")
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
	SetAdditionalSubnets(cidrs []string) error
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
func (s *Service) Update(in Update) error {
	cur := s.store.Discovery()
	in.mergeInto(&cur)
	return s.store.SaveDiscovery(cur)
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

// Subnets returns the configured additional subnets.
func (s *Service) Subnets() []config.SubnetConfig {
	return s.store.Discovery().AdditionalSubnets
}

// AddSubnet validates and appends a subnet, then persists and re-syncs the
// scanner. Returns ErrInvalidCIDR for a malformed CIDR or ErrSubnetExists for a
// duplicate.
func (s *Service) AddSubnet(in config.SubnetConfig) error {
	if _, _, err := net.ParseCIDR(in.CIDR); err != nil {
		return ErrInvalidCIDR
	}
	cur := s.store.Discovery()
	for _, existing := range cur.AdditionalSubnets {
		if existing.CIDR == in.CIDR {
			return ErrSubnetExists
		}
	}
	cur.AdditionalSubnets = append(cur.AdditionalSubnets, in)
	return s.saveAndSync(cur)
}

// UpdateSubnet renames/toggles the subnet matching in.CIDR. Returns ErrInvalidCIDR
// for a malformed CIDR or ErrSubnetNotFound if no subnet matches.
func (s *Service) UpdateSubnet(in config.SubnetConfig) error {
	if _, _, err := net.ParseCIDR(in.CIDR); err != nil {
		return ErrInvalidCIDR
	}
	cur := s.store.Discovery()
	found := false
	for i := range cur.AdditionalSubnets {
		if cur.AdditionalSubnets[i].CIDR == in.CIDR {
			cur.AdditionalSubnets[i].Name = in.Name
			cur.AdditionalSubnets[i].Enabled = in.Enabled
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
// if absent.
func (s *Service) DeleteSubnet(cidr string) error {
	cur := s.store.Discovery()
	kept := make([]config.SubnetConfig, 0, len(cur.AdditionalSubnets))
	found := false
	for _, existing := range cur.AdditionalSubnets {
		if existing.CIDR == cidr {
			found = true
			continue
		}
		kept = append(kept, existing)
	}
	if !found {
		return ErrSubnetNotFound
	}
	cur.AdditionalSubnets = kept
	return s.saveAndSync(cur)
}

// saveAndSync persists the config and pushes the enabled subnet set to the live
// scanner. The scanner sync is best-effort — it never fails the save.
func (s *Service) saveAndSync(cur config.NetworkDiscoveryConfig) error {
	if err := s.store.SaveDiscovery(cur); err != nil {
		return err
	}
	enabled := make([]string, 0, len(cur.AdditionalSubnets))
	for _, sn := range cur.AdditionalSubnets {
		if sn.Enabled {
			enabled = append(enabled, sn.CIDR)
		}
	}
	_ = s.sink.SetAdditionalSubnets(enabled)
	return nil
}

// Update is the write model for network-discovery settings: the same field set
// the wire DTO carries, in milliseconds, so the merge rules (set-if-positive)
// match the original contract exactly. The transport layer maps its request DTO
// onto this domain input.
type Update struct {
	Enabled        bool
	ARPScanWorkers int
	PingTimeoutMs  int64
	ScanTimeoutMs  int64
	AutoScan       bool
	ScanIntervalMs int64
	OUIFilePath    string
	IPv6Enabled    bool

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
	Enabled         bool
	TCPPorts        string
	UDPPorts        string
	BannerTimeoutMs int64
}

// TCPProbeUpdate mirrors the TCP-probe write model.
type TCPProbeUpdate struct {
	TimeoutMs int64
	Workers   int
}

// TimingUpdate mirrors the discovery-timing write model.
type TimingUpdate struct {
	ProbeIntervalMs  int64
	RescanIntervalMs int64
	Workers          int
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
// booleans and ScanInterval are set unconditionally; counts/timeouts/intervals
// and paths are set only when a positive/non-empty value is supplied (treating
// zero/empty as "keep existing").
func (u Update) mergeInto(cur *config.NetworkDiscoveryConfig) {
	cur.Enabled = u.Enabled
	if u.ARPScanWorkers > 0 {
		cur.ARPScanWorkers = u.ARPScanWorkers
	}
	if u.PingTimeoutMs > 0 {
		cur.PingTimeout = msDuration(u.PingTimeoutMs)
	}
	if u.ScanTimeoutMs > 0 {
		cur.ScanTimeout = msDuration(u.ScanTimeoutMs)
	}
	cur.AutoScan = u.AutoScan
	cur.ScanInterval = msDuration(u.ScanIntervalMs)
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
	if o.PortScan.TCPPorts != "" {
		cur.PortScan.TCPPorts = o.PortScan.TCPPorts
	}
	if o.PortScan.UDPPorts != "" {
		cur.PortScan.UDPPorts = o.PortScan.UDPPorts
	}
	if o.PortScan.BannerTimeoutMs > 0 {
		cur.PortScan.BannerTimeout = msDuration(o.PortScan.BannerTimeoutMs)
	}
	if o.TCPProbe.TimeoutMs > 0 {
		cur.TCPProbe.Timeout = msDuration(o.TCPProbe.TimeoutMs)
	}
	if o.TCPProbe.Workers > 0 {
		cur.TCPProbe.Workers = o.TCPProbe.Workers
	}
}

func (t TimingUpdate) mergeInto(cur *config.DiscoveryTiming) {
	if t.ProbeIntervalMs > 0 {
		cur.ProbeInterval = msDuration(t.ProbeIntervalMs)
	}
	if t.RescanIntervalMs > 0 {
		cur.RescanInterval = msDuration(t.RescanIntervalMs)
	}
	if t.Workers > 0 {
		cur.Workers = t.Workers
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
