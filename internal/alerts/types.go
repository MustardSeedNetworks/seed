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
	// Deliveries is what happened when Seed tried to send this alert to each
	// configured receiver (#368 webhook, #2997 email): one entry per channel
	// that was offered the alert. No entry for a channel means delivery never
	// applied there — no receiver was configured when the alert was raised —
	// and must never be rendered as a failure.
	Deliveries     []Delivery `json:"deliveries,omitempty"`
	Acknowledged   bool       `json:"acknowledged"`
	AcknowledgedBy *string    `json:"acknowledgedBy,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty"`
	Resolved       bool       `json:"resolved"`
	ResolvedAt     *time.Time `json:"resolvedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	Metadata       string     `json:"metadata,omitempty"` // JSON string for extra data

	// EscalationStage is the last stage of the rule's escalation ladder this
	// alert was sent at (P-B2), 0 before the first. EscalatedAt is when.
	EscalationStage int        `json:"escalationStage,omitempty"`
	EscalatedAt     *time.Time `json:"escalatedAt,omitempty"`
}

// Type constants for common alert types.
const (
	TypeSecurity     = "security"
	TypePerformance  = "performance"
	TypeConnectivity = "connectivity"
	TypeSystem       = "system"
	TypeDiscovery    = "discovery"
)

// Channel names an outbound transport. Each one records its own outcome on
// the alert, so a working webhook cannot hide a mail relay that refuses every
// message.
type Channel string

// The outbound channels.
const (
	ChannelWebhook Channel = "webhook"
	ChannelEmail   Channel = "email"
)

// Delivery is one channel's outcome for one alert.
type Delivery struct {
	Channel Channel `json:"channel"`
	// Status is one of the Delivery* constants.
	Status string `json:"status"`
	// AttemptedAt is when the last attempt finished, nil while the alert is
	// still queued.
	AttemptedAt *time.Time `json:"attemptedAt,omitempty"`
	// Error is the last attempt's error text, so an operator can tell a
	// refused connection from a rejected login without reading the daemon log.
	Error string `json:"error,omitempty"`
}

// Delivery states for Delivery.Status. There is no "not applicable" state:
// an alert nobody tried to send on a channel has no Delivery for it, because
// most installs configure no receiver at all.
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
