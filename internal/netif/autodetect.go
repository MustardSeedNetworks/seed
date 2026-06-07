package netif

import (
	"github.com/MustardSeedNetworks/seed/internal/netif/detection"
)

// AutoDetectInterfaceName returns the name of the highest-scoring live interface.
//
// preferType, if non-empty, filters to interfaces of that type ("ethernet",
// "wifi", "fiber"). When no interface matches the preferred type, the call
// falls back to the global best-scoring interface so callers always get a
// usable name when one exists on the system.
//
// Returns "" when the system has no non-loopback interface or the underlying
// detector returns an error. Callers should treat "" as "use a hardcoded
// last-ditch default" rather than as an error.
//
// This is intended as a constructor-time helper for services that need a
// reasonable interface name before the live netif.Manager is wired up.
// Replaces hardcoded "eth0"/"wlan0" fallbacks (#572).
func AutoDetectInterfaceName(preferType string) string {
	det := detection.NewDetector()
	scores, err := det.DetectAll()
	if err != nil || len(scores) == 0 {
		return ""
	}

	if preferType != "" {
		for _, s := range scores {
			if s.Type == preferType {
				return s.Name
			}
		}
	}

	return scores[0].Name
}
