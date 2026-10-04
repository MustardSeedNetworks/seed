package delivery

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
)

// The webhook channel (#368): a signed JSON POST to an operator-configured
// receiver.

// responseDrainLimit caps how much of a receiver's response body is read
// before the connection is returned to the pool. Nothing in the body is used;
// the read only keeps keep-alive working.
const responseDrainLimit = 4 << 10

// Header names the receiver verifies with.
const (
	signatureHeader = "X-Seed-Signature"
	timestampHeader = "X-Seed-Timestamp"
	alertIDHeader   = "X-Seed-Alert-Id"
)

// WebhookConfig wires a webhook Notifier. URL and Secret are required; the
// rest default.
type WebhookConfig struct {
	Options

	// URL is the receiver. Absolute, http or https, no userinfo.
	URL string
	// Secret is the HMAC-SHA256 signing material shared with the receiver.
	Secret string
	// Client overrides the HTTP client (tests, proxy-aware deployments).
	Client *http.Client
	// Timeout bounds one attempt. Ignored when Client is supplied.
	Timeout time.Duration
}

// envelope is the POST body. The alert is nested rather than sent bare so a
// later field (correlation id, site) does not collide with an alert field.
type envelope struct {
	Alert  *alerts.Alert `json:"alert"`
	SentAt time.Time     `json:"sentAt"`
}

// webhook posts one alert per request to one receiver.
type webhook struct {
	url    string
	secret []byte
	client *http.Client
	now    func() time.Time
}

// NewWebhook validates cfg and builds a webhook Notifier. It makes no request.
func NewWebhook(cfg WebhookConfig) (*Notifier, error) {
	if err := ValidateURL(cfg.URL); err != nil {
		return nil, err
	}
	if cfg.Secret == "" {
		return nil, fmt.Errorf("%w: signing material is required so the receiver can verify origin", ErrInvalidConfig)
	}
	client := cfg.Client
	if client == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = defaultTimeout
		}
		client = &http.Client{Timeout: timeout}
	}
	n := newNotifier(alerts.ChannelWebhook, redactURL(cfg.URL), cfg.Options)
	n.transport = &webhook{url: cfg.URL, secret: []byte(cfg.Secret), client: client, now: n.now}
	return n, nil
}

// ValidateURL rejects anything that cannot be a receiver: this is an outbound
// request driven by stored configuration, so the value is checked once, at
// construction, rather than at every send. It is exported because the settings
// service refuses a bad URL at the API (#2605) rather than storing one that
// silently never delivers, and both must judge it by the same rule.
func ValidateURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("%w: url is required", ErrInvalidConfig)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: url is unparseable: %w", ErrInvalidConfig, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: url scheme %q is not http or https", ErrInvalidConfig, u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("%w: url has no host", ErrInvalidConfig)
	}
	if u.User != nil {
		return fmt.Errorf("%w: url must not carry userinfo; the signature authenticates the sender", ErrInvalidConfig)
	}
	return nil
}

// redactURL keeps scheme://host and drops everything after it. The path and
// query are withheld because the common receivers (Slack, Teams, PagerDuty)
// carry their per-channel secret in the path. The input has already passed
// ValidateURL, so a parse failure here is not reachable; the fallback keeps
// the function total rather than asserting that.
func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "(unparseable)"
	}
	return u.Scheme + "://" + u.Host
}

// send makes one signed POST. A 4xx other than 408 and 429 is permanent: the
// receiver will refuse the same body again.
func (w *webhook) send(ctx context.Context, alert *alerts.Alert) (bool, error) {
	body, err := json.Marshal(envelope{Alert: alert, SentAt: w.now().UTC()})
	if err != nil {
		return true, fmt.Errorf("marshal alert %d: %w", alert.ID, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		return true, fmt.Errorf("build request: %w", err)
	}
	timestamp := strconv.FormatInt(w.now().Unix(), 10)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(timestampHeader, timestamp)
	req.Header.Set(alertIDHeader, strconv.FormatInt(alert.ID, 10))
	req.Header.Set(signatureHeader, sign(w.secret, timestamp, body))

	resp, err := w.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("post alert %d: %w", alert.ID, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, responseDrainLimit))
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return false, nil
	}
	retryable := resp.StatusCode >= 500 ||
		resp.StatusCode == http.StatusRequestTimeout ||
		resp.StatusCode == http.StatusTooManyRequests
	return !retryable, fmt.Errorf("receiver answered %d for alert %d", resp.StatusCode, alert.ID)
}

// sign returns the HMAC-SHA256 of "<timestamp>.<body>". Receivers must compare
// it in constant time (hmac.Equal, or the equivalent in their language); the
// timestamp is signed so a captured delivery cannot be replayed indefinitely.
func sign(secret []byte, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
