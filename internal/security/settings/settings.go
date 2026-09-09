// Package settings is the application service for the security-settings endpoints
// (ADR-0020 clean-hexagonal, WS-A3/A5): SNMP credentials, rogue-DHCP detection,
// and vulnerability-scanner configuration. It owns the read/mask/encrypt/merge/persist logic the transport
// layer used to carry inline. Persistence is reached through the consumer-defined
// Store port; the live rogue-DHCP detector through the RogueDetector port. Both
// are satisfied by adapters in the composition root (internal/app).
package settings

import (
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/validation"
)

// Store reads and persists the live config. Read runs fn under the config RLock;
// Write runs fn under the write lock and, when fn returns nil, persists to disk
// after releasing the lock (the #783 unlock-before-save pattern — Save acquires
// its own RLock, so saving while write-locked would deadlock).
type Store interface {
	Read(fn func(*config.Config))
	Write(fn func(*config.Config) error) error
}

// RogueDetector is the live rogue-DHCP detector surface the use-case needs: its
// effective config (interface/known-servers/alert) and a known-servers update.
type RogueDetector interface {
	Config() DetectorConfig
	UpdateKnownServers(servers []string)
}

// DetectorConfig mirrors the live detector's effective configuration.
type DetectorConfig struct {
	Interface        string
	KnownServers     []string
	AlertOnDetection bool
}

// Service is the security-settings application service.
type Service struct {
	store    Store
	detector RogueDetector
}

// NewService builds the service over its ports.
func NewService(store Store, detector RogueDetector) *Service {
	return &Service{store: store, detector: detector}
}

// ---------------------------------------------------------------------------
// SNMP
// ---------------------------------------------------------------------------

// SNMPView is the SNMP settings read model. It carries transport settings only:
// credentials live in the device-credential vault, which has its own API
// (#1799), and were never safe to return here.
type SNMPView struct {
	TimeoutMs int
	Retries   int
	Port      int
}

// SNMPUpdate is the SNMP settings write model.
type SNMPUpdate struct {
	TimeoutMs int
	Retries   int
	Port      int
}

// SNMP returns the SNMP transport settings.
func (s *Service) SNMP() SNMPView {
	var view SNMPView
	s.store.Read(func(cfg *config.Config) {
		view = SNMPView{
			TimeoutMs: int(cfg.SNMP.Timeout.Milliseconds()),
			Retries:   cfg.SNMP.Retries,
			Port:      cfg.SNMP.Port,
		}
	})
	return view
}

// UpdateSNMP persists the SNMP transport settings.
func (s *Service) UpdateSNMP(in SNMPUpdate) error {
	return s.store.Write(func(cfg *config.Config) error {
		cfg.SNMP.Timeout = time.Duration(in.TimeoutMs) * time.Millisecond
		cfg.SNMP.Retries = in.Retries
		cfg.SNMP.Port = in.Port
		return nil
	})
}

// ---------------------------------------------------------------------------
// Vulnerability scanning
// ---------------------------------------------------------------------------

// VulnUpdate is the vulnerability-scanner settings write model. It carries the
// six operator-settable fields; AutoScan is left untouched (it is not exposed by
// the settings endpoint — the original contract).
type VulnUpdate struct {
	Enabled           bool
	CVEDatabase       string
	NVDAPIKey         string
	UpdateInterval    int
	SeverityThreshold string
	MaxConcurrent     int
}

// Vuln returns the current vulnerability-scanner configuration.
func (s *Service) Vuln() config.VulnerabilityScanConfig {
	var out config.VulnerabilityScanConfig
	s.store.Read(func(c *config.Config) { out = c.Security.VulnerabilityScanning })
	return out
}

// VulnSeverity returns the configured severity threshold — the single field the
// scan-status endpoint surfaces, without reaching into the config directly.
func (s *Service) VulnSeverity() string {
	var out string
	s.store.Read(func(c *config.Config) { out = c.Security.VulnerabilityScanning.SeverityThreshold })
	return out
}

// UpdateVuln applies the six operator-settable fields and persists, leaving
// AutoScan untouched.
func (s *Service) UpdateVuln(in VulnUpdate) error {
	return s.store.Write(func(c *config.Config) error {
		v := &c.Security.VulnerabilityScanning
		v.Enabled = in.Enabled
		v.CVEDatabase = in.CVEDatabase
		v.NVDAPIKey = in.NVDAPIKey
		v.UpdateInterval = in.UpdateInterval
		v.SeverityThreshold = in.SeverityThreshold
		v.MaxConcurrent = in.MaxConcurrent
		return nil
	})
}

// ---------------------------------------------------------------------------
// Rogue DHCP detection
// ---------------------------------------------------------------------------

// RogueView is the rogue-DHCP configuration read model: Enabled from the config,
// the rest from the live detector.
type RogueView struct {
	Enabled          bool
	KnownServers     []string
	AlertOnDetection bool
	Interface        string
}

// RogueUpdate is the rogue-DHCP configuration write model; nil fields are left
// unchanged (the original partial-update contract).
type RogueUpdate struct {
	Enabled          *bool
	KnownServers     []string
	AlertOnDetection *bool
}

// RogueEnabled reports whether rogue-DHCP detection is enabled in the config —
// the precondition the detector-control handlers gate on, without reaching into
// the config directly.
func (s *Service) RogueEnabled() bool {
	var enabled bool
	s.store.Read(func(cfg *config.Config) { enabled = cfg.DHCP.RogueDetection.Enabled })
	return enabled
}

// RogueDHCP returns the effective rogue-DHCP configuration.
func (s *Service) RogueDHCP() RogueView {
	var enabled bool
	s.store.Read(func(cfg *config.Config) { enabled = cfg.DHCP.RogueDetection.Enabled })
	dc := s.detector.Config()
	return RogueView{
		Enabled:          enabled,
		KnownServers:     dc.KnownServers,
		AlertOnDetection: dc.AlertOnDetection,
		Interface:        dc.Interface,
	}
}

// UpdateRogueDHCP applies the partial update to the config, persists it, then
// syncs the live detector's known-server set when the caller supplied one.
func (s *Service) UpdateRogueDHCP(in RogueUpdate) error {
	err := s.store.Write(func(cfg *config.Config) error {
		if in.Enabled != nil {
			cfg.DHCP.RogueDetection.Enabled = *in.Enabled
		}
		if in.KnownServers != nil {
			cfg.DHCP.RogueDetection.KnownServers = in.KnownServers
		}
		if in.AlertOnDetection != nil {
			cfg.DHCP.RogueDetection.AlertOnDetection = *in.AlertOnDetection
		}
		return nil
	})
	if err != nil {
		return err
	}
	if in.KnownServers != nil {
		s.detector.UpdateKnownServers(in.KnownServers)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Guest-network isolation audit (#397)
// ---------------------------------------------------------------------------

// GuestAuditValidationError reports an invalid guest-audit setting. Kind is
// "target" (Value is the offending IP) or "port" (Value is the offending port).
// The transport layer maps Kind to the matching i18n message and surfaces Value
// as the error detail.
type GuestAuditValidationError struct {
	Kind  string
	Value string
}

func (e GuestAuditValidationError) Error() string {
	return "settings: invalid guest-audit " + e.Kind + ": " + e.Value
}

// GuestAudit returns the configured guest-network isolation audit settings.
func (s *Service) GuestAudit() config.GuestNetworkAuditConfig {
	var out config.GuestNetworkAuditConfig
	s.store.Read(func(c *config.Config) { out = c.Security.GuestNetworkAudit })
	return out
}

// UpdateGuestAudit validates and persists the guest-audit settings. Every target
// IP must parse and every port must be in 1..65535; the first offender is
// returned as a GuestAuditValidationError before anything is persisted.
func (s *Service) UpdateGuestAudit(in config.GuestNetworkAuditConfig) error {
	for _, t := range in.Targets {
		if !validation.IsValidIP(t.IP) {
			return GuestAuditValidationError{Kind: "target", Value: t.IP}
		}
	}
	for _, p := range in.Ports {
		if p < 1 || p > 65535 {
			return GuestAuditValidationError{Kind: "port", Value: strconv.Itoa(p)}
		}
	}
	return s.store.Write(func(c *config.Config) error {
		c.Security.GuestNetworkAudit = in
		return nil
	})
}
