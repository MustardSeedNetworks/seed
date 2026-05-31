package snmp

import "time"

// Target is one SNMP-polled device. Mirrors a row of polling_targets;
// CollectorChain is decoded from the row's collector_chain JSON
// column.
type Target struct {
	ID              string    `json:"id"`
	ClientID        string    `json:"client_id"`
	Name            string    `json:"name"`
	IPAddress       string    `json:"ip_address"`
	SNMPVersion     string    `json:"snmp_version"`
	CredentialsID   string    `json:"credentials_id"`
	PollIntervalSec int       `json:"poll_interval_seconds"`
	CollectorChain  []string  `json:"collector_chain"`
	Enabled         bool      `json:"enabled"`
	LastPolledAt    time.Time `json:"last_polled_at,omitzero"`
	LastStatus      string    `json:"last_status,omitempty"`
	LastError       string    `json:"last_error,omitempty"`
}

// ResolvedCredentials carries the plaintext SNMP auth material a
// Collector consumes. Decrypted at poll time from device_credentials
// via license.Manager.DecryptSecret; never persisted in plaintext.
type ResolvedCredentials struct {
	SNMPCommunity    string
	SNMPv3User       string
	SNMPv3AuthSecret string
	SNMPv3PrivSecret string
	SNMPv3AuthProto  string
	SNMPv3PrivProto  string
}
