package config

// config_types_integrations.go contains the opt-in vendor API integrations
// (P-D12) and the answer type every opt-in outbound feature shares.

// OptInChoice is the operator's answer to whether Seed may call out for one
// feature. The zero value is "not asked yet", which calls nothing, so an
// install that never answers stays as silent as one that answered no.
type OptInChoice string

// The answers to an opt-in question.
const (
	OptInUnanswered OptInChoice = ""
	OptInOff        OptInChoice = "off"
	OptInOn         OptInChoice = "on"
)

// IntegrationsConfig holds the vendor API integrations. Each is absent until
// the operator answers its question.
type IntegrationsConfig struct {
	Meraki MerakiConfig `json:"meraki,omitzero"`
}

// MerakiConfig is the opt-in Cisco Meraki Dashboard API integration. An empty
// Endpoint means Meraki's own; APIKey is keyring ciphertext (ADR-0015, the
// `enc:` prefix), never stored or served in plaintext.
type MerakiConfig struct {
	Choice   OptInChoice `json:"choice,omitempty"`
	Endpoint string      `json:"endpoint,omitempty"`
	APIKey   string      `json:"api_key,omitempty"`
}
