package engine

// reporter.go defines the optional Status() surface engines can
// implement to expose richer health information to /api/v1/engines.
// The interface is deliberately optional — adoption is per-engine,
// no compile-time pressure on engines that have nothing useful to
// say beyond "I'm registered".
//
// Operators care about three things:
//
//   1. Is this engine running and ticking? (LastTickAt)
//   2. Is anything broken? (LastError + State)
//   3. How busy is it? (Inflight, optional)
//
// The /api/v1/engines handler type-asserts each registered engine
// to Reporter; engines that don't implement it default to a
// StatusOK() payload so the wire response is uniform regardless of
// adoption status.

import "time"

// State enumerates the values Status.State may take. Locked as
// constants so handler dashboards and operator tooling can switch
// on them without inventing magic strings.
const (
	// StateOK means the engine is registered and either idle or
	// ticking on schedule.
	StateOK = "ok"

	// StateDegraded means the engine is registered but its last
	// tick failed OR its last tick is older than 2× its expected
	// interval. The engine package doesn't enforce the timing
	// check — engines decide their own thresholds because
	// "expected interval" varies wildly (probe = seconds,
	// retention = hours).
	StateDegraded = "degraded"

	// StateStopped means the engine has been explicitly stopped
	// (e.g. via Registry.Stop). Engines that have never started
	// still report StateOK because "never started" is observable
	// via LastTickAt being zero.
	StateStopped = "stopped"
)

// Status is the payload Reporter returns. Every field is optional;
// handlers tolerate zero values.
type Status struct {
	// State is one of StateOK / StateDegraded / StateStopped.
	State string

	// LastTickAt is the time the engine completed its most recent
	// scan / tick. Zero means "never ticked" (engine just started).
	LastTickAt time.Time

	// LastError is the formatted error string from the most recent
	// tick that returned an error; empty when the most recent tick
	// succeeded.
	LastError string

	// Inflight is the count of in-progress operations the engine
	// is tracking (e.g. concurrent SNMP probes). Optional — engines
	// that don't maintain a counter leave this at zero.
	Inflight int
}

// Reporter is the optional surface engines implement to expose
// per-engine status. Handlers MUST tolerate engines that do not
// implement Reporter (type-assert and fall back to StatusOK).
type Reporter interface {
	Status() Status
}

// StatusOK returns the default Status payload for engines that
// don't implement Reporter or are reporting a healthy state with
// no rich data to add. Use this rather than a struct literal so
// the default shape stays consistent across call sites.
func StatusOK() Status {
	return Status{State: StateOK}
}
