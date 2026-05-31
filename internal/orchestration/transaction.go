package orchestration

import (
	"context"
	"encoding/json"
	"time"
)

// TransactionStepKind values identify the per-step operation.
const (
	TransactionStepKindHTTP    = "http"
	TransactionStepKindHTTPS   = "https"
	TransactionStepKindTCP     = "tcp"
	TransactionStepKindExtract = "extract"
	TransactionStepKindWait    = "wait"
)

// TransactionStep is one step in a Transaction. Each step may
// reference variables extracted by previous steps via the Vars map
// passed through TransactionResult.
type TransactionStep struct {
	Kind   string          `json:"kind"` // see TransactionStepKind* constants
	Name   string          `json:"name"`
	Params json.RawMessage `json:"params"`
}

// Transaction is a multi-step probe with shared state across steps.
// Stored as a probe.Probe with Kind="transaction" and Params holding
// the step list; executed by the probe engine's transaction Checker
// (internal/probe/checkers/transaction.go in Stage A3.5).
type Transaction struct {
	Steps []TransactionStep `json:"steps"`
}

// StepResult captures one step's outcome within a Transaction.
type StepResult struct {
	Name      string          `json:"name"`
	Kind      string          `json:"kind"`
	Success   bool            `json:"success"`
	LatencyMs float64         `json:"latency_ms"`
	Error     string          `json:"error,omitempty"`
	Output    json.RawMessage `json:"output,omitempty"`
}

// TransactionResult is the aggregate outcome of a Transaction run.
// The Transaction Checker returns this as Result.Metadata so it
// fits in the unified probe_results table without a new schema.
type TransactionResult struct {
	StartedAt   time.Time    `json:"started_at"`
	CompletedAt time.Time    `json:"completed_at"`
	Steps       []StepResult `json:"steps"`
	Status      string       `json:"status"` // "pass" | "fail" | "partial"
}

// Executor runs a Transaction's step list. The probe engine's
// transaction Checker delegates to an Executor. Defined for
// testability — production wires it to one implementation.
type Executor interface {
	// Execute runs all steps in order, threading shared state via
	// the per-step output map.
	Execute(ctx context.Context, tx Transaction) TransactionResult
}
