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
	ID       int64   `json:"id"`
	Type     string  `json:"type"`     // e.g., "security", "performance", "connectivity"
	Severity string  `json:"severity"` // "info", "warning", "error", "critical"
	Title    string  `json:"title"`
	Message  string  `json:"message"`
	Source   string  `json:"source,omitempty"` // What generated the alert
	DeviceID *string `json:"deviceId,omitempty"`
	// Rule is the pipeline rule that raised this alert ("bgp.flap",
	// "iface.down", or an operator rule's name). It is the alert's identity
	// for anything reasoning about *why* it fired: the title is prose meant
	// for a human and changes when the copy is improved.
	Rule string `json:"rule,omitempty"`
	// RootCauseID names an earlier alert that probably caused this one — set
	// by internal/alerts/correlation, nil when nothing explains it.
	RootCauseID *int64 `json:"rootCauseId,omitempty"`
	// DeliveryStatus is what happened when the outbound webhook (#368) last
	// tried to send this alert: one of the Delivery* constants, or empty.
	// Empty means delivery never applied to this alert — no receiver is
	// configured, or the alert predates the one that is — and must never be
	// rendered as a failure.
	DeliveryStatus string `json:"deliveryStatus,omitempty"`
	// DeliveryAttemptedAt is when the last attempt finished, nil while the
	// alert is still queued and on every alert delivery never touched.
	DeliveryAttemptedAt *time.Time `json:"deliveryAttemptedAt,omitempty"`
	// DeliveryError is the last attempt's error text, so an operator can tell
	// a refused connection from a 401 without reading the daemon log.
	DeliveryError  string     `json:"deliveryError,omitempty"`
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

// Delivery states for DeliveryStatus. The zero value — the empty string — is
// deliberately not one of them: an alert nobody tried to send is not a failed
// delivery, and most installs configure no receiver at all.
const (
	// DeliveryPending means the alert is queued for the receiver and no
	// attempt has finished yet.
	DeliveryPending = "pending"
	// DeliveryDelivered means the receiver accepted it.
	DeliveryDelivered = "delivered"
	// DeliveryFailed means every bounded attempt failed; the alert is in the
	// inbox and the receiver never got it.
	DeliveryFailed = "failed"
	// DeliveryDropped means the delivery queue was full, so the alert was
	// never offered to the receiver. Distinct from failed because the cause
	// is Seed's own backpressure, not the receiver.
	DeliveryDropped = "dropped"
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
