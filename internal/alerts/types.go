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
	// configured receiver (#368 webhook, #2997 email, #3037 syslog): one entry per channel
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

// The rule ids of the built-in observation pipeline, spelled once for the
// pipeline that raises them and everything that reasons about them.
const (
	RuleInterfaceDown   = "iface.down"
	RuleBGPFlap         = "bgp.flap"
	RuleStorageHigh     = "storage.high"
	RuleStorageCritical = "storage.critical"
	// RuleSTPRootChange is a switch electing a different spanning-tree root
	// bridge; RuleSTPTopologyChange is any other topology change it counted.
	// A root change is never also reported as a topology change.
	RuleSTPRootChange     = "stp.root_change"
	RuleSTPTopologyChange = "stp.topology_change"
	// RuleRouteWithdrawn is prefixes leaving a router's table between polls;
	// RuleRouteChange is prefixes that stayed but moved to other next hops.
	RuleRouteWithdrawn = "route.withdrawn"
	RuleRouteChange    = "route.change"
)

// RuleFlowIndicator is the listener pipeline rule that alerts on a flow to
// or from an address on the operator's threat indicator list.
const RuleFlowIndicator = "flow.indicator"

// RuleVoIPQuality is the listener pipeline rule that alerts on an RTP
// stream window scored at or below the VoIP analyser's alert MOS.
const RuleVoIPQuality = "voip.quality"

// RuleConfigChange is the alert raised when a device's backed-up configuration
// differs from the one stored before it (P-D2).
const RuleConfigChange = "config.change"

// ConfigChangeEvidence is the Metadata of a RuleConfigChange alert: the two
// backups compared and the size of the change. It never carries configuration
// text, which holds the device's secrets and is read through the API.
type ConfigChangeEvidence struct {
	TargetID         string `json:"targetId"`
	BackupID         string `json:"backupId"`
	PreviousBackupID string `json:"previousBackupId"`
	Added            int    `json:"added"`
	Removed          int    `json:"removed"`
}

// RuleDHCPScopeExhaustion is the alert raised when a DHCP scope nears
// exhaustion or runs out of addresses (P-D5).
const RuleDHCPScopeExhaustion = "dhcp.scope.exhaustion"

// DHCPScopeEvidence is the Metadata of a RuleDHCPScopeExhaustion alert: the
// scope and how much of it was seen in use.
type DHCPScopeEvidence struct {
	Start       string  `json:"start"`
	End         string  `json:"end"`
	Size        uint64  `json:"size"`
	Used        int     `json:"used"`
	UsedPercent float64 `json:"usedPercent"`
	// Exhaustion is "near" or "full".
	Exhaustion string `json:"exhaustion"`
}

// InterfaceDownEvidence is the Metadata of a RuleInterfaceDown alert: the one
// ifTable row that went down, not the whole table it was read from.
type InterfaceDownEvidence struct {
	IfIndex uint32 `json:"ifIndex"`
	IfName  string `json:"ifName"`
	// IfOperStatus is the RFC 2863 value the interface went down to.
	IfOperStatus int `json:"ifOperStatus"`
}

// BGPPeerEvidence is the Metadata of a RuleBGPFlap alert: the peer that left
// Established and the RFC 4273 bgpPeerState it was in when polled.
type BGPPeerEvidence struct {
	RemoteAddr string `json:"remoteAddr"`
	RemoteAS   uint32 `json:"remoteAs"`
	State      int    `json:"state"`
}

// RouteWithdrawnEvidence is the Metadata of a RuleRouteWithdrawn alert.
// Withdrawn counts every prefix that left the table; Routes lists at most the
// first 50 of them by prefix.
type RouteWithdrawnEvidence struct {
	Withdrawn int               `json:"withdrawn"`
	Routes    []RouteWithdrawal `json:"routes"`
}

// RouteWithdrawal is one prefix that left the table, with the next hops and
// the RFC 2096 ipCidrRouteProto it had before it went.
type RouteWithdrawal struct {
	Prefix   string   `json:"prefix"`
	NextHops []string `json:"nextHops"`
	Proto    int      `json:"proto"`
}

// RouteChangeEvidence is the Metadata of a RuleRouteChange alert. Changed
// counts every prefix whose next hops differ; Routes lists at most the first
// 50 of them by prefix.
type RouteChangeEvidence struct {
	Changed int                  `json:"changed"`
	Routes  []RouteNextHopChange `json:"routes"`
}

// RouteNextHopChange is one prefix whose next hops differ between polls.
type RouteNextHopChange struct {
	Prefix string   `json:"prefix"`
	From   []string `json:"from"`
	To     []string `json:"to"`
}

// StorageEvidence is the Metadata of a RuleStorageHigh or RuleStorageCritical
// alert: the filesystem that crossed the threshold.
type StorageEvidence struct {
	Index       uint32  `json:"index"`
	Description string  `json:"description"`
	SizeBytes   uint64  `json:"sizeBytes"`
	UsedBytes   uint64  `json:"usedBytes"`
	UsedPercent float64 `json:"usedPercent"`
}

// STPRootChangeEvidence is the Metadata of a RuleSTPRootChange alert. Bridges
// are rendered priority/MAC.
type STPRootChangeEvidence struct {
	PreviousRoot string `json:"previousRoot"`
	Root         string `json:"root"`
	// RootPort is the bridge port toward the new root, 0 when this switch
	// became the root.
	RootPort int `json:"rootPort"`
}

// STPTopologyChangeEvidence is the Metadata of a RuleSTPTopologyChange alert:
// how far dot1dStpTopChanges rose since the previous poll and which of this
// switch's ports changed state meanwhile. No local port transitions means the
// change was learned from elsewhere in the spanning tree.
type STPTopologyChangeEvidence struct {
	Changes         uint32              `json:"changes"`
	TopologyChanges uint32              `json:"topologyChanges"`
	Ports           []STPPortTransition `json:"ports"`
}

// STPPortTransition is one port whose dot1dStpPortState changed between polls.
type STPPortTransition struct {
	Port    int `json:"port"`
	IfIndex int `json:"ifIndex"`
	From    int `json:"from"`
	To      int `json:"to"`
}

// Channel names an outbound transport. Each one records its own outcome on
// the alert, so a working webhook cannot hide a mail relay that refuses every
// message.
type Channel string

// The outbound channels.
const (
	ChannelWebhook Channel = "webhook"
	ChannelEmail   Channel = "email"
	ChannelSyslog  Channel = "syslog"
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
