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

// ErrCredentialNotFound is returned when a device-credential lookup misses,
// including when the row exists but belongs to a different client.
var ErrCredentialNotFound = errors.New("device credential not found")

// Credential mirrors a device_credentials row as stored: the three secret
// fields hold versioned DEK ciphertext ("enc:v<N>:...", ADR-0015), never
// plaintext. Decryption happens at poll time in the SNMP credential resolver;
// nothing in this package ever holds a decrypted secret.
type Credential struct {
	ID           string
	ClientID     string
	Name         string
	CommunityEnc string
	V3User       string
	V3AuthEnc    string
	V3PrivEnc    string
	V3AuthProto  string
	V3PrivProto  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
