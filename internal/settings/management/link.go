package management

import (
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// validLinkMode reports whether mode is an accepted combined speed/duplex value
// for a link override.
func validLinkMode(mode string) bool {
	switch mode {
	case "auto", "10/half", "10/full", "100/half", "100/full",
		"1000/full", "2500/full", "5000/full", "10000/full":
		return true
	default:
		return false
	}
}

// Link returns the current link settings, defaulting an unset mode to "auto".
func (s *Service) Link() config.LinkConfig {
	var out config.LinkConfig
	s.store.Read(func(cfg *config.Config) { out = cfg.Link })
	if out.Mode == "" {
		out.Mode = "auto"
	}
	return out
}

// UpdateLink validates and persists the link settings. An unrecognized mode
// returns ErrValidation; "" (unset) is allowed.
func (s *Service) UpdateLink(in config.LinkConfig) error {
	if in.Mode != "" && !validLinkMode(in.Mode) {
		return fmt.Errorf("%w: link mode %q", ErrValidation, in.Mode)
	}
	return s.store.Write(func(cfg *config.Config) error {
		cfg.Link = in
		return nil
	})
}

// CableTest returns the current cable-test settings.
func (s *Service) CableTest() config.CableTestConfig {
	var out config.CableTestConfig
	s.store.Read(func(cfg *config.Config) { out = cfg.CableTest })
	return out
}

// UpdateCableTest persists the cable-test settings.
func (s *Service) UpdateCableTest(in config.CableTestConfig) error {
	return s.store.Write(func(cfg *config.Config) error {
		cfg.CableTest = in
		return nil
	})
}
