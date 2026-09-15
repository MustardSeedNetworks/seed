package api

import (
	"math"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/timeseries/retention"
)

// history_window.go resolves what a history request can actually be answered
// with. The store is tiered (internal/timeseries/retention): every tier keeps
// 7 days of raw rows, Starter adds 30 days of hourly rollups, Pro 90 days of
// hourly and 2 years of daily. #175's rule is that a request past a tier's
// horizon degrades to that horizon rather than erroring, so the resolution
// lives here, in one pure function, rather than in each handler.

// History bucket resolutions and sources, as the wire spells them.
const (
	historyResolutionHourly = "hourly"
	historyResolutionDaily  = "daily"

	// historySourceRaw aggregates the raw table on read. It is the source for
	// any window the raw horizon covers, on every tier: it is identical across
	// tiers, and it includes the bucket in progress, which the rollup tables
	// cannot (they are written after the bucket closes).
	historySourceRaw = "raw"
	// historySourceRollup reads a pre-aggregated rollup table.
	historySourceRollup = "rollup"
)

// hoursPerDay is the number of hours in a day, for turning a requested
// duration into the whole days the horizons are expressed in.
const hoursPerDay = 24

// historyHourlyMaxDays is the longest window served at hourly resolution.
// Beyond it a chart wants days, not 700+ points.
const historyHourlyMaxDays = 31

// historyWindow is the resolved answer to "how much of this can I have, and
// from where".
type historyWindow struct {
	// From and To bound the window. To is the request instant, not the last
	// closed bucket, so a raw-sourced series carries the bucket in progress.
	From time.Time
	To   time.Time
	// Days is the served window in whole days.
	Days int
	// RequestedDays is what the caller asked for, in whole days; it differs
	// from Days exactly when the tier clamped the request.
	RequestedDays int
	// Resolution is the bucket width: hourly or daily.
	Resolution string
	// Source says whether the series is aggregated from raw rows or read from
	// a rollup table.
	Source string
	// Clamped is true when the tier's horizon is shorter than the request.
	Clamped bool
}

// resolveHistoryWindow maps a requested duration onto what the tier retains.
func resolveHistoryWindow(now time.Time, requested time.Duration, h retention.TierHorizons) historyWindow {
	requestedDays := max(int(math.Ceil(requested.Hours()/hoursPerDay)), 1)

	horizon := max(h.RawDays, max(h.HourlyDays, h.DailyDays))
	days := min(requestedDays, horizon)

	w := historyWindow{
		From:          now.AddDate(0, 0, -days),
		To:            now,
		Days:          days,
		RequestedDays: requestedDays,
		Resolution:    historyResolutionHourly,
		Source:        historySourceRollup,
		Clamped:       days < requestedDays,
	}
	if days > historyHourlyMaxDays {
		w.Resolution = historyResolutionDaily
	}
	if days <= h.RawDays {
		w.Source = historySourceRaw
	}
	return w
}
