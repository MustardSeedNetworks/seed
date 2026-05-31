package orchestration

import (
	"context"
	"encoding/json"
	"time"
)

// Snapshot is a point-in-time frozen copy of a client's inventory +
// topology + active feature set. Persisted to baseline_snapshots
// during Stage A3.5; Payload carries the serialized state.
type Snapshot struct {
	ID         string          `json:"id"`
	ClientID   string          `json:"client_id"`
	Label      string          `json:"label"`
	CapturedAt time.Time       `json:"captured_at"`
	Payload    json.RawMessage `json:"payload"`
}

// DiffEntryKind identifies what kind of entity a DiffEntry describes.
const (
	DiffEntryKindDevice  = "device"
	DiffEntryKindNode    = "node"
	DiffEntryKindLink    = "link"
	DiffEntryKindFeature = "feature"
)

// DiffEntry is one added/removed/changed entity in a Diff.
type DiffEntry struct {
	Kind   string          `json:"kind"` // see DiffEntryKind* constants
	ID     string          `json:"id"`
	Detail json.RawMessage `json:"detail,omitempty"`
}

// Diff is the result of comparing a Snapshot to the current state.
type Diff struct {
	SnapshotID string      `json:"snapshot_id"`
	ComputedAt time.Time   `json:"computed_at"`
	Added      []DiffEntry `json:"added"`
	Removed    []DiffEntry `json:"removed"`
	Changed    []DiffEntry `json:"changed"`
}

// Capturer captures Snapshots and computes Diffs against them.
// Implementation lands in Stage A3.5.
type Capturer interface {
	// Capture freezes the current state for the given client and
	// returns the persisted Snapshot.
	Capture(ctx context.Context, clientID, label string) (Snapshot, error)

	// Diff compares the named Snapshot against current state for
	// the same client and returns the structured difference.
	Diff(ctx context.Context, snapshotID string) (Diff, error)
}
