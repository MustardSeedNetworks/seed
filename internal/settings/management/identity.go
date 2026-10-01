package management

import (
	"errors"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// buildIdentitySettings is the identity slice of the settings read model.
func buildIdentitySettings(cfg *config.Config) map[string]any {
	return map[string]any{
		"name":     cfg.Identity.Name,
		"location": cfg.Identity.Location,
	}
}

// applyIdentityUpdates applies the device name and location (#195). A field
// the update omits keeps its value. Surrounding whitespace is dropped, so a
// name of spaces is no name rather than a blank header line. The candidate is
// validated whole before anything is assigned, so a refused write leaves the
// live identity exactly as it was.
func applyIdentityUpdates(updates map[string]any, cfg *config.Config) error {
	val, exists := updates["identity"]
	if !exists {
		return nil
	}
	identity, ok := val.(map[string]any)
	if !ok {
		return errors.New("identity must be an object")
	}

	next := cfg.Identity
	for key, field := range map[string]*string{"name": &next.Name, "location": &next.Location} {
		value, present, err := extractString(identity, key, "identity")
		if err != nil {
			return err
		}
		if present {
			*field = strings.TrimSpace(value)
		}
	}
	if err := next.Validate(); err != nil {
		return err
	}
	cfg.Identity = next
	return nil
}
