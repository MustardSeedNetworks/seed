// Package management is the application service for the main settings endpoint
// (ADR-0020 clean-hexagonal, WS-A2). It owns the read/apply/persist logic the
// transport layer used to carry inline: the HTTP handler decodes the request,
// calls Get or Update here, and encodes the result. Persistence is reached
// through the consumer-defined Store port, satisfied by an adapter in the
// composition root (internal/app).
package management

import (
	"errors"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// Sentinel errors the transport layer maps to HTTP status codes.
var (
	// ErrConflict is returned when an If-Match ETag does not match the current
	// settings token (optimistic concurrency, HTTP 412).
	ErrConflict = errors.New("management: settings ETag mismatch")
	// ErrValidation is returned when one or more apply helpers reject the
	// update payload (HTTP 400).
	ErrValidation = errors.New("management: invalid update fields")
)

// Store reads and persists the main application settings. Read runs fn under
// the config read-lock; Write runs fn under the write-lock, then saves to
// disk if fn returns nil.
type Store interface {
	// Read calls fn with the live config held under the config RLock.
	Read(fn func(*config.Config))
	// Write calls fn with the live config held under the config Lock. If fn
	// returns nil, the config is saved to disk. The lock is released before
	// Save acquires its own RLock (fixes #783 deadlock pattern).
	Write(fn func(*config.Config) error) error
}

// Reconfigurer re-points a running component at settings that have just been
// written, so a Settings edit takes effect without a daemon restart (#2605).
// Today one component needs it: the outbound alert webhook.
//
// It takes no argument and returns none. The implementation reads the live
// config it already holds, so the component is re-pointed with exactly what
// was persisted — including a secret this write did not change — and a
// receiver it cannot build is a lost optional integration it logs, never an
// error that would suggest the settings were not saved. They were.
type Reconfigurer interface {
	ReconfigureAlerts()
}

// Service is the main-settings application service.
type Service struct {
	store    Store
	encrypt  Encrypter
	reconfig Reconfigurer
}

// NewService builds the settings management service over its Store port.
// encrypt is the keyring seam the alert-webhook secret is stored through;
// reconfig re-points the webhook after a write. Both may be nil in a
// composition that has neither — the settings that need them are then refused
// rather than half-applied.
func NewService(store Store, encrypt Encrypter, reconfig Reconfigurer) *Service {
	return &Service{store: store, encrypt: encrypt, reconfig: reconfig}
}

// Get returns the current settings map and the ETag header value. The map
// mirrors the getSettings read model verbatim (behavior-preserving).
func (s *Service) Get() (map[string]any, string) {
	var settings map[string]any
	var etag string
	s.store.Read(func(cfg *config.Config) {
		settings = map[string]any{
			"interface": map[string]any{
				"current":   cfg.Interface.Default,
				"available": []string{},
			},
			"vlan":       map[string]any{"enabled": cfg.VLAN.Enabled, "id": cfg.VLAN.ID},
			"ip":         map[string]any{"mode": cfg.IP.Mode},
			"thresholds": buildThresholdSettings(cfg),
			// Reflect the live config — these were previously hardcoded, so a GET
			// never echoed what a prior PUT wrote (and any concurrency token derived
			// from them would be incoherent).
			"healthChecks": map[string]any{
				"runPerformance": cfg.HealthChecks.RunPerformance,
				"runSpeedtest":   cfg.HealthChecks.RunSpeedtest,
				"runIperf":       cfg.HealthChecks.RunIperf,
				"runDiscovery":   cfg.HealthChecks.RunDiscovery,
			},
			"speedtest": map[string]any{
				"serverId":      cfg.Speedtest.ServerID,
				"autoRunOnLink": cfg.Speedtest.AutoRunOnLink,
			},
			"iperf": map[string]any{
				"autoRunOnLink": cfg.Iperf.AutoRunOnLink, "server": cfg.Iperf.Server,
				"port": cfg.Iperf.Port, "protocol": cfg.Iperf.Protocol,
				"direction": cfg.Iperf.Direction, "duration": cfg.Iperf.Duration,
				"serverPort": cfg.Iperf.ServerPort, "enableServer": cfg.Iperf.EnableServer,
			},
			"alerts":       buildAlertSettings(cfg),
			"cardSettings": buildCardSettings(),
			"displayOptions": map[string]any{
				"showPublicIP": cfg.DisplayOptions.ShowPublicIP,
				"unitSystem":   cfg.DisplayOptions.UnitSystem,
			},
		}
		etag = cfg.SettingsETagLocked()
	})
	return settings, etag
}

// Update applies updates to the live config and persists it. When ifMatch is
// non-empty it is compared to the current ETag; a mismatch returns ErrConflict.
// A type error in any apply helper returns ErrValidation.
func (s *Service) Update(updates map[string]any, ifMatch string) error {
	if err := s.write(updates, ifMatch); err != nil {
		return err
	}
	s.reconfigure()
	return nil
}

// write is Update's compare-and-apply half, split out so the reconfigure that
// follows it runs after the config lock is released: re-pointing the webhook
// stops the previous receiver, which waits for an in-flight HTTP attempt, and
// no settings write may hold the config lock for a receiver's timeout.
func (s *Service) write(updates map[string]any, ifMatch string) error {
	return s.store.Write(func(cfg *config.Config) error {
		// Compare-and-apply is atomic under the write lock: a concurrent writer
		// cannot slip between the ETag check and the mutations below.
		if ifMatch != "" && ifMatch != strings.Trim(cfg.SettingsETagLocked(), `"`) {
			return ErrConflict
		}

		var applyErrors []error
		if err := applyThresholdUpdates(updates, cfg); err != nil {
			applyErrors = append(applyErrors, err)
		}
		if err := applyHealthChecksUpdates(updates, cfg); err != nil {
			applyErrors = append(applyErrors, err)
		}
		if err := applySpeedtestUpdates(updates, cfg); err != nil {
			applyErrors = append(applyErrors, err)
		}
		if err := applyIperfUpdates(updates, cfg); err != nil {
			applyErrors = append(applyErrors, err)
		}
		if err := applyFABOptionsUpdates(updates, cfg); err != nil {
			applyErrors = append(applyErrors, err)
		}
		if err := applyDisplayOptionsUpdates(updates, cfg); err != nil {
			applyErrors = append(applyErrors, err)
		}
		if err := s.applyAlerts(updates, cfg); err != nil {
			applyErrors = append(applyErrors, err)
		}

		if len(applyErrors) > 0 {
			return ErrValidation
		}
		return nil
	})
}

// applyAlerts applies the alert-webhook section. Without an encrypter there is
// nowhere safe to put the signing material, so the section is refused rather
// than written in plaintext.
func (s *Service) applyAlerts(updates map[string]any, cfg *config.Config) error {
	if _, present := updates["alerts"]; !present {
		return nil
	}
	if s.encrypt == nil {
		return errors.New("alerts: no keyring is available to store the webhook secret")
	}
	return applyAlertsUpdates(updates, cfg, s.encrypt)
}

// reconfigure tells the running webhook to re-read what was just persisted.
func (s *Service) reconfigure() {
	if s.reconfig != nil {
		s.reconfig.ReconfigureAlerts()
	}
}

// ETag returns the current settings concurrency token (the value the settings
// endpoints emit after a successful write).
func (s *Service) ETag() string {
	var etag string
	s.store.Read(func(cfg *config.Config) { etag = cfg.SettingsETagLocked() })
	return etag
}
