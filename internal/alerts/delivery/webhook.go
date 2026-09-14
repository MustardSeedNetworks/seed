// Package delivery is the one outbound transport for alerts (#368): a signed
// JSON POST to an operator-configured receiver, so a customer can bridge Seed
// into the Slack, PagerDuty or SIEM they already run. Seed detects, records and
// displays alerts everywhere else; this is the only place it sends one.
//
// Three properties are load-bearing and are asserted by the package tests:
//
//   - No URL configured means the Notifier is never constructed, so an
//     air-gapped deployment makes no outbound request and logs no error.
//   - Delivery never blocks the alert pipeline. Deliver enqueues onto a bounded
//     channel and drops (recording the drop) when the receiver is slow enough to
//     fill it; the alert is already in the store by then.
//   - Retry is bounded — a fixed, small number of attempts with linear backoff,
//     not a durable queue. A receiver that is down loses the delivery and leaves
//     a visible failure in Status, never unbounded growth.
package delivery

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
)

// ErrInvalidConfig is returned by New when the operator-supplied webhook
// configuration cannot produce a signed request.
var ErrInvalidConfig = errors.New("alert webhook: invalid configuration")

// Tunables — production defaults.
const (
	defaultMaxAttempts = 3
	defaultBackoff     = 2 * time.Second
	defaultTimeout     = 10 * time.Second
	defaultQueueSize   = 256

	// responseDrainLimit caps how much of a receiver's response body is read
	// before the connection is returned to the pool. Nothing in the body is
	// used; the read only keeps keep-alive working.
	responseDrainLimit = 4 << 10
)

// Header names the receiver verifies with.
const (
	signatureHeader = "X-Seed-Signature"
	timestampHeader = "X-Seed-Timestamp"
	alertIDHeader   = "X-Seed-Alert-Id"
)

// Config wires a Notifier. URL and Secret are required; the rest default.
type Config struct {
	// URL is the receiver. Absolute, http or https, no userinfo.
	URL string
	// Secret is the HMAC-SHA256 signing material shared with the receiver.
	Secret string
	// Client overrides the HTTP client (tests, proxy-aware deployments).
	Client *http.Client
	// MaxAttempts is the total number of tries per alert, retries included.
	MaxAttempts int
	// Backoff is the base delay between attempts; attempt n waits n*Backoff.
	Backoff time.Duration
	// Timeout bounds one attempt. Ignored when Client is supplied.
	Timeout time.Duration
	// QueueSize bounds the pending-delivery channel.
	QueueSize int
	Logger    *slog.Logger
	Now       func() time.Time
}

// Status is the last-delivery picture an operator needs to notice a webhook
// that is configured but not arriving.
type Status struct {
	Delivered     uint64    `json:"delivered"`
	Failed        uint64    `json:"failed"`
	Dropped       uint64    `json:"dropped"`
	LastAttemptAt time.Time `json:"lastAttemptAt,omitzero"`
	LastSuccessAt time.Time `json:"lastSuccessAt,omitzero"`
	LastError     string    `json:"lastError,omitempty"`
}

// envelope is the POST body. The alert is nested rather than sent bare so a
// later field (correlation id, site) does not collide with an alert field.
type envelope struct {
	Alert  *alerts.Alert `json:"alert"`
	SentAt time.Time     `json:"sentAt"`
}

// Notifier posts alerts to one receiver on a single worker goroutine.
type Notifier struct {
	url         string
	endpoint    string
	secret      []byte
	client      *http.Client
	maxAttempts int
	backoff     time.Duration
	logger      *slog.Logger
	now         func() time.Time

	queue chan *alerts.Alert

	mu      sync.Mutex
	status  Status
	started bool
	stopped bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// New validates cfg and builds a Notifier. It makes no request.
func New(cfg Config) (*Notifier, error) {
	if err := validateURL(cfg.URL); err != nil {
		return nil, err
	}
	if cfg.Secret == "" {
		return nil, fmt.Errorf("%w: signing material is required so the receiver can verify origin", ErrInvalidConfig)
	}

	n := &Notifier{
		url:         cfg.URL,
		endpoint:    redactURL(cfg.URL),
		secret:      []byte(cfg.Secret),
		client:      cfg.Client,
		maxAttempts: cfg.MaxAttempts,
		backoff:     cfg.Backoff,
		logger:      cfg.Logger,
		now:         cfg.Now,
	}
	if n.client == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = defaultTimeout
		}
		n.client = &http.Client{Timeout: timeout}
	}
	if n.maxAttempts < 1 {
		n.maxAttempts = defaultMaxAttempts
	}
	if n.backoff <= 0 {
		n.backoff = defaultBackoff
	}
	if n.logger == nil {
		n.logger = slog.Default()
	}
	if n.now == nil {
		n.now = time.Now
	}
	size := cfg.QueueSize
	if size < 1 {
		size = defaultQueueSize
	}
	n.queue = make(chan *alerts.Alert, size)
	return n, nil
}

// validateURL rejects anything that cannot be a receiver: this is an outbound
// request driven by stored configuration, so the value is checked once, at
// construction, rather than at every send.
func validateURL(raw string) error {
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

// Endpoint is the receiver in a form safe to log: scheme and host only. The
// path and query are withheld because the common receivers (Slack, Teams,
// PagerDuty) carry their per-channel secret in the path.
func (n *Notifier) Endpoint() string { return n.endpoint }

// redactURL keeps scheme://host and drops everything after it. The input has
// already passed validateURL, so a parse failure here is not reachable; the
// fallback keeps the function total rather than asserting that.
func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "(unparseable)"
	}
	return u.Scheme + "://" + u.Host
}

// Start launches the delivery worker. Calling it twice is a no-op.
func (n *Notifier) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.started || n.stopped {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	n.cancel = cancel
	n.started = true
	n.wg.Go(func() { n.run(ctx) })
}

// Stop cancels the worker and waits for the in-flight attempt, or until ctx is
// done. The Notifier is not restartable.
func (n *Notifier) Stop(ctx context.Context) {
	n.mu.Lock()
	if n.stopped {
		n.mu.Unlock()
		return
	}
	n.stopped = true
	cancel := n.cancel
	n.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	done := make(chan struct{})
	go func() {
		n.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// Deliver queues alert for delivery. It never blocks: when the queue is full
// the alert is dropped and counted, because the alert pipeline's tick must not
// wait on someone else's HTTP endpoint.
func (n *Notifier) Deliver(alert *alerts.Alert) {
	if alert == nil {
		return
	}
	select {
	case n.queue <- alert:
	default:
		n.mu.Lock()
		n.status.Dropped++
		n.status.LastError = "delivery queue full"
		n.mu.Unlock()
		n.logger.Error("alert webhook queue full; delivery dropped",
			"alert_id", alert.ID, "queue_size", cap(n.queue))
	}
}

// Status returns a copy of the last-delivery counters.
func (n *Notifier) Status() Status {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.status
}

func (n *Notifier) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case alert := <-n.queue:
			n.send(ctx, alert)
		}
	}
}

// send makes up to maxAttempts tries, stopping early on success or on a
// response the receiver will give again for the same body.
func (n *Notifier) send(ctx context.Context, alert *alerts.Alert) {
	body, err := json.Marshal(envelope{Alert: alert, SentAt: n.now().UTC()})
	if err != nil {
		n.record(false, fmt.Errorf("marshal alert %d: %w", alert.ID, err))
		return
	}

	var lastErr error
	for attempt := 1; attempt <= n.maxAttempts; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				n.record(false, fmt.Errorf("delivery abandoned at shutdown: %w", lastErr))
				return
			case <-time.After(time.Duration(attempt-1) * n.backoff):
			}
		}

		permanent, attemptErr := n.post(ctx, alert, body)
		if attemptErr == nil {
			n.record(true, nil)
			return
		}
		lastErr = attemptErr
		if permanent {
			break
		}
	}
	n.record(false, lastErr)
}

// post makes one attempt. It reports whether the failure is permanent — a
// receiver that rejects this body will reject it again, so retrying only
// duplicates load.
func (n *Notifier) post(ctx context.Context, alert *alerts.Alert, body []byte) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return true, fmt.Errorf("build request: %w", err)
	}
	timestamp := strconv.FormatInt(n.now().Unix(), 10)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(timestampHeader, timestamp)
	req.Header.Set(alertIDHeader, strconv.FormatInt(alert.ID, 10))
	req.Header.Set(signatureHeader, sign(n.secret, timestamp, body))

	resp, err := n.client.Do(req)
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

func (n *Notifier) record(ok bool, err error) {
	now := n.now()
	n.mu.Lock()
	defer n.mu.Unlock()
	n.status.LastAttemptAt = now
	if ok {
		n.status.Delivered++
		n.status.LastSuccessAt = now
		n.status.LastError = ""
		return
	}
	n.status.Failed++
	if err != nil {
		n.status.LastError = err.Error()
		n.logger.Error("alert webhook delivery failed", "error", err)
	}
}
