package config

// config_types_lifecycle.go contains the end-of-life tracking settings (P-D9).

// LifecycleConfig holds the end-of-life tracking settings.
type LifecycleConfig struct {
	EoXFeed EoXFeedConfig `json:"eox_feed,omitzero"`
}

// EoXFeedConfig is the opt-in EoX feed. Endpoint and TokenURL empty mean
// Cisco's own; ClientSecret is keyring ciphertext (ADR-0015, the `enc:`
// prefix), never stored or served in plaintext.
type EoXFeedConfig struct {
	Choice       OptInChoice `json:"choice,omitempty"`
	Endpoint     string      `json:"endpoint,omitempty"`
	TokenURL     string      `json:"token_url,omitempty"`
	ClientID     string      `json:"client_id,omitempty"`
	ClientSecret string      `json:"client_secret,omitempty"`
}
