// Package settings is the application service for the security-settings endpoints
// (ADR-0020 clean-hexagonal, WS-A3/A5): SNMP credentials, rogue-DHCP detection,
// and vulnerability-scanner configuration. It owns the read/mask/encrypt/merge/persist logic the transport
// layer used to carry inline. Persistence is reached through the consumer-defined
// Store port; the live rogue-DHCP detector through the RogueDetector port. Both
// are satisfied by adapters in the composition root (internal/app).
package settings

import (
	"errors"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// PasswordPlaceholder is the masked value returned for stored SNMP passwords. A
// PUT carrying the placeholder for a credential leaves that stored password
// unchanged (never re-encrypts the mask).
const PasswordPlaceholder = "*****"

// Sentinel errors the transport layer maps to HTTP status / i18n messages.
var (
	// ErrEncryptAuth wraps a failure to encrypt an SNMPv3 auth password.
	ErrEncryptAuth = errors.New("settings: encrypt auth password")
	// ErrEncryptPriv wraps a failure to encrypt an SNMPv3 priv password.
	ErrEncryptPriv = errors.New("settings: encrypt priv password")
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

// SNMPv3Credential is the wire-neutral SNMPv3 credential. On read, AuthPassword/
// PrivPassword carry PasswordPlaceholder when a stored password exists; on write,
// the placeholder means "keep the stored password".
type SNMPv3Credential struct {
	Name          string
	Username      string
	AuthProtocol  string
	AuthPassword  string
	PrivProtocol  string
	PrivPassword  string
	ContextName   string
	SecurityLevel string
}

// SNMPView is the SNMP settings read model (passwords masked).
type SNMPView struct {
	Communities   []string
	V3Credentials []SNMPv3Credential
	TimeoutMs     int
	Retries       int
	Port          int
}

// SNMPUpdate is the SNMP settings write model.
type SNMPUpdate struct {
	Communities   []string
	V3Credentials []SNMPv3Credential
	TimeoutMs     int
	Retries       int
	Port          int
}

// SNMP returns the SNMP settings with passwords masked — actual stored passwords
// are never returned.
func (s *Service) SNMP() SNMPView {
	var view SNMPView
	s.store.Read(func(cfg *config.Config) {
		creds := make([]SNMPv3Credential, len(cfg.SNMP.V3Credentials))
		for i := range cfg.SNMP.V3Credentials {
			c := &cfg.SNMP.V3Credentials[i]
			creds[i] = SNMPv3Credential{
				Name:          c.Name,
				Username:      c.Username,
				AuthProtocol:  c.AuthProtocol,
				AuthPassword:  maskIfSet(c.AuthPassword),
				PrivProtocol:  c.PrivProtocol,
				PrivPassword:  maskIfSet(c.PrivPassword),
				ContextName:   c.ContextName,
				SecurityLevel: c.SecurityLevel,
			}
		}
		view = SNMPView{
			Communities:   cfg.SNMP.Communities,
			V3Credentials: creds,
			TimeoutMs:     int(cfg.SNMP.Timeout.Milliseconds()),
			Retries:       cfg.SNMP.Retries,
			Port:          cfg.SNMP.Port,
		}
	})
	return view
}

// UpdateSNMP encrypts any newly-supplied passwords (preserving stored ones sent as
// the placeholder) and persists the settings. Encryption runs under the write
// lock; the save happens after the lock is released. Returns ErrEncryptAuth /
// ErrEncryptPriv when a password cannot be encrypted.
func (s *Service) UpdateSNMP(in SNMPUpdate) error {
	return s.store.Write(func(cfg *config.Config) error {
		creds := make([]config.SNMPv3Credential, len(in.V3Credentials))
		for i := range in.V3Credentials {
			var existing *config.SNMPv3Credential
			if i < len(cfg.SNMP.V3Credentials) {
				existing = &cfg.SNMP.V3Credentials[i]
			}
			converted, err := convertCredential(cfg, in.V3Credentials[i], existing)
			if err != nil {
				return err
			}
			creds[i] = converted
		}
		cfg.SNMP.Communities = in.Communities
		cfg.SNMP.V3Credentials = creds
		cfg.SNMP.Timeout = time.Duration(in.TimeoutMs) * time.Millisecond
		cfg.SNMP.Retries = in.Retries
		cfg.SNMP.Port = in.Port
		return nil
	})
}

// convertCredential maps a wire credential to config form, encrypting a freshly
// supplied password and preserving the stored one when the placeholder is sent.
func convertCredential(
	cfg *config.Config, in SNMPv3Credential, existing *config.SNMPv3Credential,
) (config.SNMPv3Credential, error) {
	out := config.SNMPv3Credential{
		Name:          in.Name,
		Username:      in.Username,
		AuthProtocol:  in.AuthProtocol,
		PrivProtocol:  in.PrivProtocol,
		ContextName:   in.ContextName,
		SecurityLevel: in.SecurityLevel,
	}

	switch {
	case in.AuthPassword != "" && in.AuthPassword != PasswordPlaceholder:
		enc, err := cfg.EncryptCredentialValue(in.AuthPassword)
		if err != nil {
			return out, fmt.Errorf("%w: %w", ErrEncryptAuth, err)
		}
		out.AuthPassword = enc
	case existing != nil:
		out.AuthPassword = existing.AuthPassword
	}

	switch {
	case in.PrivPassword != "" && in.PrivPassword != PasswordPlaceholder:
		enc, err := cfg.EncryptCredentialValue(in.PrivPassword)
		if err != nil {
			return out, fmt.Errorf("%w: %w", ErrEncryptPriv, err)
		}
		out.PrivPassword = enc
	case existing != nil:
		out.PrivPassword = existing.PrivPassword
	}

	return out, nil
}

func maskIfSet(stored string) string {
	if stored != "" {
		return PasswordPlaceholder
	}
	return ""
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
