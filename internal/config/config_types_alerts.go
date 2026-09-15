package config

// AlertsConfig holds the operator-configurable half of alerting. Detection and
// the inbox need no configuration; sending an alert somewhere else does, and
// that is the one thing here.
type AlertsConfig struct {
	Webhook AlertWebhookConfig `json:"webhook"`
}

// AlertWebhookConfig is the outbound alert receiver (#368, #2605).
//
// An empty URL means delivery is off: nothing is constructed and no outbound
// request is made, so an air-gapped install loses a feature rather than
// function.
type AlertWebhookConfig struct {
	// URL is the receiver. Absolute, http or https, no userinfo — validated on
	// the way in by the settings service, which refuses a URL that could never
	// receive a POST rather than storing one that silently never delivers.
	URL string `json:"url"`

	// Secret is the HMAC signing material the receiver verifies with, held as
	// keyring ciphertext (ADR-0015, the `enc:` prefix). It is never written in
	// plaintext and never served back: the settings read model reports only
	// whether one is set.
	Secret string `json:"secret"`
}
