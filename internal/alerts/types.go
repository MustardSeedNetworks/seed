// Package alerts holds the alert domain types — the Alert record, the
// alerting Rule, and their query options — shared by the alert pipeline,
// inbox, and rules subpackages. The types live here (not in internal/database)
// so those subpackages stay persistence-free: the alert/rule repositories in
// internal/database import this package and map SQL rows to these types, so the
// dependency points inward (database -> alerts), never the reverse.
package alerts

import (
	"errors"
	"time"
)

// ErrRuleNotFound is returned when an alert-rule lookup misses.
var ErrRuleNotFound = errors.New("alert rule not found")

// Alert represents a system alert.
type Alert struct {
	ID             int64      `json:"id"`
	Type           string     `json:"type"`     // e.g., "security", "performance", "connectivity"
	Severity       string     `json:"severity"` // "info", "warning", "error", "critical"
	Title          string     `json:"title"`
	Message        string     `json:"message"`
	Source         string     `json:"source,omitempty"` // What generated the alert
	DeviceID       *string    `json:"deviceId,omitempty"`
	Acknowledged   bool       `json:"acknowledged"`
	AcknowledgedBy *string    `json:"acknowledgedBy,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty"`
	Resolved       bool       `json:"resolved"`
	ResolvedAt     *time.Time `json:"resolvedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	Metadata       string     `json:"metadata,omitempty"` // JSON string for extra data
}

// Type constants for common alert types.
const (
	TypeSecurity     = "security"
	TypePerformance  = "performance"
	TypeConnectivity = "connectivity"
	TypeSystem       = "system"
	TypeDiscovery    = "discovery"
)

// Severity constants.
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityError    = "error"
	SeverityCritical = "critical"
)

// Rule is one alerting rule: a match predicate over inbound events plus the
// alert to raise when it fires.
type Rule struct {
	ID                   int64
	Name                 string
	Enabled              bool
	MatchKind            string
	MatchSeverity        string
	MatchPayloadContains string
	AlertType            string
	AlertSeverity        string
	AlertTitle           string
	AlertMessage         string
	WindowSeconds        int
	ThresholdCount       int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// ListOptions narrows an alert list query — empty values disable that filter.
type ListOptions struct {
	Type               string
	Severity           string
	DeviceID           string
	UnacknowledgedOnly bool
	UnresolvedOnly     bool
	Since              time.Time
	Limit              int
	Offset             int
}
