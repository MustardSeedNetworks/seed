package topology

import (
	"log/slog"
	"time"
)

// reconcilerDefaults holds the per-reconciler tunables every
// implementation (sysinfo, iftable, lldp, arp, ...) needs.
// Extracted so the Logger/Now/Interval defaulting logic isn't
// duplicated per-reconciler — keeps constructors short and
// satisfies the dupl linter when adding the 3rd+ reconciler.
type reconcilerDefaults struct {
	logger   *slog.Logger
	now      func() time.Time
	interval time.Duration
}

// applyDefaults fills nil Logger/Now and clamps Interval. Returns
// the canonical values so callers can copy them into their own
// struct.
func applyDefaults(logger *slog.Logger, now func() time.Time, interval time.Duration) reconcilerDefaults {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	if interval <= 0 {
		interval = defaultPollInterval
	}
	if interval < minPollInterval {
		interval = minPollInterval
	}
	return reconcilerDefaults{logger: logger, now: now, interval: interval}
}
