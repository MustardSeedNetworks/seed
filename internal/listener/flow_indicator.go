package listener

import "time"

// FlowIndicatorKind is the event kind the flow collector publishes when a
// flow's endpoint is on the operator's threat indicator list (P-C5). Unlike
// syslog and trap events, its SourceAddr is not the exporter that sent the
// flow: it is the host that exchanged traffic with the listed address, the
// one an operator investigates, so alert suppression applies per host.
const FlowIndicatorKind = "flow-indicator"

// FlowIndicatorHit is the payload of a [FlowIndicatorKind] event: the flows
// in one collector batch between one host and one listed address.
type FlowIndicatorHit struct {
	// Host is the other endpoint of the flows; usually a local address.
	Host string `json:"host"`
	// Listed is the endpoint the list matched, and Indicator the most
	// specific list entry containing it.
	Listed    string    `json:"listed"`
	Indicator string    `json:"indicator"`
	Exporter  string    `json:"exporter"`
	Flows     int64     `json:"flows"`
	Bytes     uint64    `json:"bytes"`
	Packets   uint64    `json:"packets"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
}
