package probe

import "context"

// Checker is the per-kind probe implementation. Implementations live
// in internal/probe/checkers/ — one file per Kind.
type Checker interface {
	// Kind returns the probe kind this Checker handles. Must match
	// one of the Kind* constants in types.go.
	Kind() string

	// RequiredCapabilities returns hardware capability names this
	// Checker needs to run. The engine refuses to schedule a probe
	// whose target host lacks a required capability; the UI hides
	// the new-probe form for unsupported kinds.
	//
	// Capability strings reference internal/netif/detection/ fields
	// (e.g. "tdr", "monitor_mode", "raw_sockets"). Empty slice means
	// the Checker requires no special hardware.
	RequiredCapabilities() []string

	// Run executes one observation against the Probe's target and
	// returns the Result. Implementations must honor ctx cancellation
	// and never panic — return an error in Result.Error instead.
	Run(ctx context.Context, p Probe) Result
}

// Registry maps Kind to Checker. The engine consults this when
// dispatching probes.
type Registry interface {
	// Register adds a Checker. Re-registering the same Kind replaces
	// the previous Checker.
	Register(c Checker)

	// Get returns the Checker for the given Kind. The second return
	// is false if no Checker is registered.
	Get(kind string) (Checker, bool)

	// Kinds returns all registered Kinds, sorted.
	Kinds() []string
}
