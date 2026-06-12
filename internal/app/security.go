package app

// security.go wires the composition root to the security-settings application
// service (ADR-0020): SNMP credentials and rogue-DHCP detection configuration.
// The adapters implement the narrow ports declared in internal/security/settings
// over the live config and the live rogue-DHCP detector, so the handlers depend
// on a use-case instead of reaching into s.config / the detector directly. The
// detector is resolved lazily so a later-set value (the api test harness) is
// honored; a nil detector degrades the rogue reads/updates to empty no-ops.

import (
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/security/settings"
)

// NewSecuritySettings builds the security-settings use-case over the live config
// (read/encrypt/persist) and a lazy accessor for the rogue-DHCP detector.
func NewSecuritySettings(
	cfg *config.Config, path string, detector func() *dhcp.RogueDetector,
) *settings.Service {
	return settings.NewService(
		securitySettingsStore{cfg: cfg, path: path},
		securityRogueDetector{detector: detector},
	)
}

// securitySettingsStore implements settings.Store over the live config, owning the
// lock + on-disk save the port abstracts away. Write releases the lock before
// Save (Save acquires its own RLock; saving while write-locked would deadlock —
// the #783 pattern).
type securitySettingsStore struct {
	cfg  *config.Config
	path string
}

func (s securitySettingsStore) Read(fn func(*config.Config)) {
	s.cfg.RLock()
	defer s.cfg.RUnlock()
	fn(s.cfg)
}

func (s securitySettingsStore) Write(fn func(*config.Config) error) error {
	s.cfg.Lock()
	err := fn(s.cfg)
	s.cfg.Unlock()
	if err != nil {
		return err
	}
	return s.cfg.Save(s.path)
}

// securityRogueDetector implements settings.RogueDetector over the live detector,
// resolved lazily; a nil detector yields an empty config and a no-op update.
type securityRogueDetector struct {
	detector func() *dhcp.RogueDetector
}

func (a securityRogueDetector) Config() settings.DetectorConfig {
	d := a.detector()
	if d == nil {
		return settings.DetectorConfig{}
	}
	c := d.GetConfig()
	return settings.DetectorConfig{
		Interface:        c.Interface,
		KnownServers:     c.KnownServers,
		AlertOnDetection: c.AlertOnDetection,
	}
}

func (a securityRogueDetector) UpdateKnownServers(servers []string) {
	if d := a.detector(); d != nil {
		d.UpdateKnownServers(servers)
	}
}
