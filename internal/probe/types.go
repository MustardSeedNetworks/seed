package probe

import (
	"encoding/json"
	"time"
)

// Kind constants identify probe types. The engine selects a registered
// Checker by Kind. Adding a new kind is a new constant + a new
// Checker implementation in internal/probe/checkers/.
const (
	KindDNS         = "dns"
	KindTLS         = "tls"
	KindPing        = "ping"
	KindTCP         = "tcp"
	KindUDP         = "udp"
	KindHTTP        = "http"
	KindHTTPS       = "https"
	KindRTSP        = "rtsp"
	KindDICOM       = "dicom"
	KindHL7         = "hl7"
	KindFHIR        = "fhir"
	KindLTI         = "lti"
	KindLDAP        = "ldap"
	KindSQL         = "sql"
	KindFileShare   = "fileshare"
	KindOPCUA       = "opcua"
	KindMODBUS      = "modbus"
	KindNTP         = "ntp"
	KindSIP         = "sip"
	KindDot1X       = "dot1x"
	KindCable       = "cable"
	KindTransaction = "transaction"
)

// Probe is one configured observation that the engine schedules and
// dispatches. ClientID scopes it to a tenant (see
// SEED_ARCHITECTURE.md section 3.0).
type Probe struct {
	ID              string          `json:"id"`
	ClientID        string          `json:"client_id"`
	Kind            string          `json:"kind"`
	DisplayName     string          `json:"display_name"`
	Target          string          `json:"target"`
	Params          json.RawMessage `json:"params,omitempty"`
	IntervalSeconds int             `json:"interval_seconds"`
	Enabled         bool            `json:"enabled"`
	Warning         json.RawMessage `json:"warning,omitempty"`
	Critical        json.RawMessage `json:"critical,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// Result is one observation emitted by a Checker for a Probe.
// Persisted into probe_results during Stage A1 schema migration.
type Result struct {
	ProbeID   string          `json:"probe_id"`
	ClientID  string          `json:"client_id"`
	Kind      string          `json:"kind"`
	Timestamp time.Time       `json:"timestamp"`
	Success   bool            `json:"success"`
	LatencyMs float64         `json:"latency_ms"`
	Error     string          `json:"error,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

// Breach is a threshold violation detected during Result evaluation.
// Emitted alongside the Result to the alerts pipeline via channel.
type Breach struct {
	ProbeID   string    `json:"probe_id"`
	ClientID  string    `json:"client_id"`
	Severity  string    `json:"severity"`
	Field     string    `json:"field"`
	Threshold any       `json:"threshold"`
	Actual    any       `json:"actual"`
	Timestamp time.Time `json:"timestamp"`
}

// ResultEvent bundles a Result with any threshold Breaches detected.
// Subscribers (alerts pipeline) receive this on the engine's channel.
type ResultEvent struct {
	Result   Result   `json:"result"`
	Breaches []Breach `json:"breaches,omitempty"`
}
