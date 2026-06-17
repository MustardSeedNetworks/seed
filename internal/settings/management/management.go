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

// Service is the main-settings application service.
type Service struct {
	store Store
}

// NewService builds the settings management service over its Store port.
func NewService(store Store) *Service {
	return &Service{store: store}
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

		if len(applyErrors) > 0 {
			return ErrValidation
		}
		return nil
	})
}

// ETag returns the current settings concurrency token (the value the settings
// endpoints emit after a successful write).
func (s *Service) ETag() string {
	var etag string
	s.store.Read(func(cfg *config.Config) { etag = cfg.SettingsETagLocked() })
	return etag
}
