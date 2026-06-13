// Package observation holds the domain types for SNMP observations.
// These are the row-level types that flow from the snmp_observations
// table through the polling pipeline into the topology reconcilers
// and alert pipelines.
//
// This package is a pure-types leaf: it imports nothing from
// internal/database or any other infrastructure package, so callers
// (topology reconcilers, alert pipelines) can depend on it without
// taking a persistence dependency.
package observation

import "time"

// SNMPObservation is one row of snmp_observations. PayloadJSON
// carries the typed collector Observation struct serialized as JSON
// — callers decode it back into the kind-specific shape they own.
type SNMPObservation struct {
	ID          int64
	ClientID    string
	TargetID    string
	Kind        string
	ObservedAt  time.Time
	PayloadJSON string
	IngestedAt  time.Time
}

// ListOptions narrows the query — empty values disable that filter.
type ListOptions struct {
	ClientID string
	TargetID string
	Kind     string
	Since    time.Time
	Until    time.Time
	Limit    int
}
