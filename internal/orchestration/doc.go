// Package orchestration defines three composition patterns over the
// probe engine: Sequence (named ordered probe runs — the AutoTest
// pattern), Baseline (point-in-time snapshot + diff over inventory
// + topology — the TotalView "what changed" pattern), and Transaction
// (multi-step probe with shared state — the synthetic-transaction
// pattern).
//
// All three are V1.0 parity must-haves; all build on probe.Engine
// rather than introducing new persistence shapes. Sequence and
// Baseline have small kind-specific result tables (sequence_results,
// baseline_snapshots); Transaction is a probe kind (kind="transaction")
// with steps embedded in Probe.Params.
//
// License gating:
//   - Free: no orchestration
//   - Starter: sequences only (uses per-client profiles)
//   - Pro: sequences + baseline + transactions
//
// V1.0 NMS expansion — Stage A0 scaffold (2026-05-30). Implementations
// land in Stage A3.5.
package orchestration
