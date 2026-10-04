// Package delivery sends alerts somewhere else: a signed JSON POST to an
// operator-configured receiver (#368), so a customer can bridge Seed into the
// Slack, PagerDuty or SIEM they already run, email through the operator's
// own mail relay (#2997), and RFC 5424 syslog to their collector (#3037).
// Seed detects, records and displays alerts everywhere else; this is the only
// place it sends one.
//
// Every channel runs on the same Notifier, and these properties are
// load-bearing and asserted by the package tests:
//
//   - No receiver configured means no Notifier is constructed, so an
//     air-gapped deployment makes no outbound connection and logs no error.
//   - Delivery never blocks the alert pipeline. Deliver enqueues onto a bounded
//     channel and drops (recording the drop) when the receiver is slow enough to
//     fill it; the alert is already in the store by then.
//   - Retry is bounded — a fixed, small number of attempts with linear backoff,
//     not a durable queue. A receiver that is down loses the delivery and leaves
//     a visible failure in Status, never unbounded growth.
//   - Every outcome is written back onto the alert it was for, per channel,
//     through the Recorder. The counters alone were invisible — nothing served
//     them on any route — so a receiver that had been refusing every message
//     for a week looked exactly like one that was working.
package delivery

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
)

// ErrInvalidConfig is returned when operator-supplied receiver configuration
// cannot produce a delivery.
var ErrInvalidConfig = errors.New("alert delivery: invalid configuration")

// Tunables — production defaults.
const (
	defaultMaxAttempts = 3
	defaultBackoff     = 2 * time.Second
	defaultTimeout     = 10 * time.Second
	defaultQueueSize   = 256
)

// Options are the parts of a Notifier every channel shares. All of them
// default.
type Options struct {
	// Recorder persists each delivery's outcome onto the alert it was for, so
	// a receiver that stopped accepting is visible in the inbox rather than
	// only in counters nothing serves. Optional: nil means the outcome is
	// counted but not written back.
	Recorder Recorder
	// MaxAttempts is the total number of tries per alert, retries included.
	MaxAttempts int
	// Backoff is the base delay between attempts; attempt n waits n*Backoff.
	Backoff time.Duration
	// QueueSize bounds the pending-delivery channel.
	QueueSize int
	Logger    *slog.Logger
	Now       func() time.Time
}

// Recorder writes one delivery's outcome onto the alert it was for. It is the
// narrow half of the alert repository this package needs, declared here so
// delivery does not import internal/database.
//
// status is one of the alerts.Delivery* constants.
type Recorder interface {
	RecordDelivery(
		ctx context.Context,
		alertID int64,
		channel alerts.Channel,
		status string,
		attemptedAt time.Time,
		deliveryErr string,
	) error
}

// Status is the last-delivery picture an operator needs to notice a receiver
// that is configured but not arriving.
type Status struct {
	Delivered     uint64    `json:"delivered"`
	Failed        uint64    `json:"failed"`
	Dropped       uint64    `json:"dropped"`
	LastAttemptAt time.Time `json:"lastAttemptAt,omitzero"`
	LastSuccessAt time.Time `json:"lastSuccessAt,omitzero"`
	LastError     string    `json:"lastError,omitempty"`
}

// transport makes one attempt to hand an alert to one receiver.
type transport interface {
	// send reports whether a failure is permanent — a receiver that rejects
	// this alert will reject it again, so retrying only duplicates load.
	send(ctx context.Context, alert *alerts.Alert) (permanent bool, err error)
}

// Notifier delivers alerts on one channel from a single worker goroutine.
type Notifier struct {
	channel     alerts.Channel
	endpoint    string
	transport   transport
	maxAttempts int
	backoff     time.Duration
	logger      *slog.Logger
	now         func() time.Time
	recorder    Recorder

	queue chan *alerts.Alert

	mu      sync.Mutex
	status  Status
	started bool
	stopped bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// newNotifier applies the shared defaults. The caller sets transport, which
// may itself need the defaulted clock.
func newNotifier(channel alerts.Channel, endpoint string, opts Options) *Notifier {
	n := &Notifier{
		channel:     channel,
		endpoint:    endpoint,
		maxAttempts: opts.MaxAttempts,
		backoff:     opts.Backoff,
		logger:      opts.Logger,
		now:         opts.Now,
		recorder:    opts.Recorder,
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
	size := opts.QueueSize
	if size < 1 {
		size = defaultQueueSize
	}
	n.queue = make(chan *alerts.Alert, size)
	return n
}

// Channel is the transport this Notifier delivers on.
func (n *Notifier) Channel() alerts.Channel { return n.channel }

// Endpoint is the receiver in a form safe to log: no path, query or
// credentials.
func (n *Notifier) Endpoint() string { return n.endpoint }

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

// Deliver queues alert for delivery and reports whether it was queued. It
// never blocks: when the queue is full the alert is dropped and counted,
// because the alert pipeline's tick must not wait on someone else's server.
//
// The drop is written back to the alert here, synchronously, rather than
// left to the worker — nothing will ever revisit a dropped alert, so a row
// left reading "pending" would read that way forever.
func (n *Notifier) Deliver(ctx context.Context, alert *alerts.Alert) bool {
	if alert == nil {
		return false
	}
	// A stopped Notifier has no worker, so an enqueued alert would sit in the
	// channel reading "pending" for the life of the process. The Manager
	// replaces a Notifier when the operator re-points the receiver, so this is
	// reachable by a settings write landing beside an alert.
	if n.retired() {
		n.recordOnAlert(ctx, alert, alerts.DeliveryFailed, "receiver was reconfigured before this alert was sent")
		return false
	}
	select {
	case n.queue <- alert:
		return true
	default:
		n.mu.Lock()
		n.status.Dropped++
		n.status.LastError = "delivery queue full"
		n.mu.Unlock()
		n.logger.ErrorContext(ctx, "alert delivery queue full; delivery dropped",
			"channel", n.channel, "alert_id", alert.ID, "queue_size", cap(n.queue))
		n.recordOnAlert(ctx, alert, alerts.DeliveryDropped, "delivery queue full")
		return false
	}
}

// SendNow makes one attempt to deliver alert, synchronously, outside the
// queue and without writing an outcome onto any alert. It is the Settings
// test-send: the operator is waiting for the answer, and the receiver's own
// reason is the answer.
func (n *Notifier) SendNow(ctx context.Context, alert *alerts.Alert) error {
	_, err := n.transport.send(ctx, alert)
	return err
}

// retired reports whether Stop has run, so Deliver can refuse rather than
// enqueue onto a channel nothing is reading any more.
func (n *Notifier) retired() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.stopped
}

// recordOnAlert writes one outcome onto the alert. A failure to write is
// logged and dropped: the delivery itself already happened (or already failed)
// and the counters hold it, so turning a bookkeeping error into anything more
// would cost the caller nothing it can act on.
//
// The ctx is the caller's — the worker's run context for an attempt, the
// pipeline's for a drop. At shutdown the run context is already cancelled, so
// an outcome landing in that window is lost rather than written by a detached
// context that would outlive the daemon's own deadline.
func (n *Notifier) recordOnAlert(ctx context.Context, alert *alerts.Alert, status, errText string) {
	if n.recorder == nil {
		return
	}
	if err := n.recorder.RecordDelivery(ctx, alert.ID, n.channel, status, n.now().UTC(), errText); err != nil {
		n.logger.ErrorContext(ctx, "could not record alert delivery status",
			"error", err, "channel", n.channel, "alert_id", alert.ID, "delivery_status", status)
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
			n.attempt(ctx, alert)
		}
	}
}

// attempt makes up to maxAttempts tries, stopping early on success or on a
// failure the receiver will give again for the same alert.
func (n *Notifier) attempt(ctx context.Context, alert *alerts.Alert) {
	wire := forTheWire(alert)
	var lastErr error
	for try := 1; try <= n.maxAttempts; try++ {
		if try > 1 {
			select {
			case <-ctx.Done():
				n.fail(ctx, alert, fmt.Errorf("delivery abandoned at shutdown: %w", lastErr))
				return
			case <-time.After(time.Duration(try-1) * n.backoff):
			}
		}

		permanent, err := n.transport.send(ctx, wire)
		if err == nil {
			n.record(true, nil)
			n.recordOnAlert(ctx, alert, alerts.DeliveryDelivered, "")
			return
		}
		lastErr = err
		if permanent {
			break
		}
	}
	n.fail(ctx, alert, lastErr)
}

// fail records one exhausted delivery in both places the outcome is read: the
// aggregate counters and the alert's own delivery state.
func (n *Notifier) fail(ctx context.Context, alert *alerts.Alert, err error) {
	n.record(false, err)
	text := ""
	if err != nil {
		text = err.Error()
	}
	n.recordOnAlert(ctx, alert, alerts.DeliveryFailed, text)
}

// forTheWire strips Seed's own delivery bookkeeping from the copy that goes to
// the receiver. The alert is stamped "pending" before it is stored, so without
// this the delivered alert would carry state about the very delivery it is
// part of — meaningless to a receiver, and a wire-contract field nobody wants
// to be held to later.
func forTheWire(alert *alerts.Alert) *alerts.Alert {
	wire := *alert
	wire.Deliveries = nil
	return &wire
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
		n.logger.Error("alert delivery failed", "channel", n.channel, "error", err)
	}
}

// isCertificateError reports whether err is a certificate the collector will
// present again.
func isCertificateError(err error) bool {
	_, certErr := errors.AsType[*tls.CertificateVerificationError](err)
	_, hostErr := errors.AsType[x509.HostnameError](err)
	_, authErr := errors.AsType[x509.UnknownAuthorityError](err)
	return certErr || hostErr || authErr
}
