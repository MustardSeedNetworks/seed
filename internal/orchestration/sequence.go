package orchestration

import (
	"context"
	"encoding/json"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// Sequence is a named ordered list of probe invocations executed by
// the Runner. Stored in the existing profiles table (a profile may
// reference a sequence) and/or in a dedicated sequences table during
// Stage A3.5 implementation.
type Sequence struct {
	ID        string         `json:"id"`
	ClientID  string         `json:"client_id"`
	Name      string         `json:"name"`
	OnFailure string         `json:"on_failure"` // "stop" | "continue"
	Steps     []SequenceStep `json:"steps"`
}

// SequenceStep is one step in a Sequence. Resolves to an ad-hoc
// probe.Probe invocation via probe.Engine.RunDefinition (the result
// is NOT persisted as a normal probe_result; instead it's bundled
// into SequenceResult.Steps).
type SequenceStep struct {
	Kind       string          `json:"kind"`
	Target     string          `json:"target"`
	Params     json.RawMessage `json:"params,omitempty"`
	Warning    json.RawMessage `json:"warning,omitempty"`
	Critical   json.RawMessage `json:"critical,omitempty"`
	TimeoutSec int             `json:"timeout_seconds,omitempty"`
}

// SequenceResult captures one execution of a Sequence. Persisted to
// sequence_results during Stage A3.5; the per-step probe.Result is
// nested in Steps rather than written to probe_results separately.
type SequenceResult struct {
	SequenceID    string         `json:"sequence_id"`
	ClientID      string         `json:"client_id"`
	StartedAt     time.Time      `json:"started_at"`
	CompletedAt   time.Time      `json:"completed_at"`
	OverallStatus string         `json:"overall_status"` // "pass" | "fail" | "partial"
	Steps         []probe.Result `json:"steps"`
}

// Runner executes a Sequence. Implementation lands in Stage A3.5.
type Runner interface {
	// Run executes the Sequence in order. OnFailure="stop" aborts on
	// the first failing step; OnFailure="continue" runs all steps
	// and reports OverallStatus="partial" if any failed.
	Run(ctx context.Context, s Sequence) SequenceResult
}
