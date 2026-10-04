package config

// AlertsConfig holds the operator-configurable half of alerting. Detection and
// the inbox need no configuration; sending an alert somewhere else does, and
// that is the one thing here.
type AlertsConfig struct {
	Webhook AlertWebhookConfig `json:"webhook"`
	Email   AlertEmailConfig   `json:"email"`
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

// AlertEmailConfig is the outbound mail relay alerts are emailed through
// (#2997). An empty Host means email is off.
//
// TLS is not optional: TLS is "starttls" (the default, port 587) or "tls"
// (implicit, port 465). A relay that cannot encrypt is refused by name rather
// than sent device names and credentials in the clear.
type AlertEmailConfig struct {
	Host string `json:"host"`
	// Port is 0 for the TLS mode's conventional port.
	Port int    `json:"port"`
	TLS  string `json:"tls"`
	// Username is empty for a relay that admits by address.
	Username string `json:"username"`
	// Password is keyring ciphertext (ADR-0015, the `enc:` prefix), never
	// plaintext and never served back, like the webhook's Secret.
	Password string   `json:"password"`
	From     string   `json:"from"`
	To       []string `json:"to,omitempty"`
}
