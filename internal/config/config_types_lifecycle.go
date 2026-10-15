package config

// config_types_lifecycle.go contains the end-of-life tracking settings (P-D9).

// LifecycleConfig holds the end-of-life tracking settings.
type LifecycleConfig struct {
	EoXFeed EoXFeedConfig `json:"eox_feed,omitzero"`
}

// EoXFeedChoice is the operator's answer to whether Seed may fetch Cisco EoX
// data itself. The zero value is "not asked yet", which fetches nothing, so
// an install that never answers stays as silent as one that answered no.
type EoXFeedChoice string

// The answers to the EoX feed question.
const (
	EoXFeedUnanswered EoXFeedChoice = ""
	EoXFeedOff        EoXFeedChoice = "off"
	EoXFeedOn         EoXFeedChoice = "on"
)

// EoXFeedConfig is the opt-in EoX feed. Endpoint and TokenURL empty mean
// Cisco's own; ClientSecret is keyring ciphertext (ADR-0015, the `enc:`
// prefix), never stored or served in plaintext.
type EoXFeedConfig struct {
	Choice       EoXFeedChoice `json:"choice,omitempty"`
	Endpoint     string        `json:"endpoint,omitempty"`
	TokenURL     string        `json:"token_url,omitempty"`
	ClientID     string        `json:"client_id,omitempty"`
	ClientSecret string        `json:"client_secret,omitempty"`
}
