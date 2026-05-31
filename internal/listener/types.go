package listener

import (
	"crypto/tls"
	"encoding/json"
	"time"
)

// Config carries the parameters one Listener needs to bind. TLSConfig
// is optional — set only for TLS-enabled listeners (e.g. syslog over
// 6514).
type Config struct {
	BindAddr  string      // "0.0.0.0:162" / "[::]:514" / etc.
	TLSConfig *tls.Config // nil = plain TCP/UDP
}

// Event is one observation emitted by a Listener. ClientID + Target
// are populated by the enrichment step (resolves SourceAddr against
// polling_targets + discovered_devices). Unresolved sources show as
// TargetKind="unknown_ip" with ClientID set to the default client of
// the listening Seed instance.
type Event struct {
	Kind       string          `json:"kind"`        // "snmp_trap" | "syslog" | ...
	SourceAddr string          `json:"source_addr"` // "192.168.1.1:54321"
	ClientID   string          `json:"client_id"`
	TargetKind string          `json:"target_kind,omitempty"`
	TargetID   string          `json:"target_id,omitempty"`
	Severity   string          `json:"severity,omitempty"` // syslog severity, etc.
	Timestamp  time.Time       `json:"timestamp"`
	Payload    json.RawMessage `json:"payload"`
}
