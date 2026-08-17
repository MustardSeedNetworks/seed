// Package polling provides domain types for the SNMP polling subsystem.
// Persistence code lives in internal/database (which depends inward on this
// package); this package imports only the standard library.
package polling

import (
	"errors"
	"time"
)

// ErrTargetNotFound is returned when a polling target lookup misses.
var ErrTargetNotFound = errors.New("polling target not found")

// Target mirrors a polling_targets row. CollectorChain is decoded from
// the JSON column. Last* fields record the most recent poll's outcome
// and feed the operator-facing target status.
type Target struct {
	ID              string    `json:"id"`
	ClientID        string    `json:"clientId"`
	Name            string    `json:"name"`
	IPAddress       string    `json:"ipAddress"`
	SNMPVersion     string    `json:"snmpVersion"`
	CredentialsID   string    `json:"credentialsId,omitempty"`
	PollIntervalSec int       `json:"pollIntervalSeconds"`
	Enabled         bool      `json:"enabled"`
	CollectorChain  []string  `json:"collectorChain"`
	LastPolledAt    time.Time `json:"lastPolledAt,omitzero"`
	LastStatus      string    `json:"lastStatus,omitempty"`
	LastError       string    `json:"lastError,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// ErrCredentialsNotFound is returned when a credentials lookup misses.
var ErrCredentialsNotFound = errors.New("device credentials not found")

// Credentials mirrors a device_credentials row. The secret fields hold
// versioned ciphertext exactly as stored; this package never sees plaintext.
// Decryption happens at poll time in internal/polling/snmp, which owns the
// keyring seam.
type Credentials struct {
	ID              string    `json:"id"`
	ClientID        string    `json:"clientId"`
	Name            string    `json:"name"`
	SNMPCommunityCT string    `json:"-"`
	SNMPv3User      string    `json:"snmpV3User,omitempty"`
	SNMPv3AuthCT    string    `json:"-"`
	SNMPv3PrivCT    string    `json:"-"`
	SNMPv3AuthProto string    `json:"snmpV3AuthProto,omitempty"`
	SNMPv3PrivProto string    `json:"snmpV3PrivProto,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
