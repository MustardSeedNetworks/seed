package delivery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
)

// ErrNotConfigured is returned by SendTest for a channel with no receiver.
var ErrNotConfigured = errors.New("alert delivery: no receiver is configured on this channel")

// Manager owns the receivers the daemon delivers to, one per channel, and
// lets an operator change them from Settings without restarting (#2605).
//
// The alert store chain is wired once, at startup, around the Manager rather
// than around a Notifier: a Notifier is immutable and not restartable, so a
// chain built around one could only be re-pointed by rebuilding the pipelines.
// The Manager is the indirection that makes the swap possible — every alert
// asks it, at write time, which receivers there are.
//
// A configuration that cannot produce a delivery disables that channel and is
// logged, never fatal: the same judgement server_alert_delivery.go has always
// made, that a broken optional integration must not take the diagnostics down
// with it.
type Manager struct {
	recorder Recorder
	logger   *slog.Logger

	mu      sync.Mutex
	current map[alerts.Channel]*Notifier
	stopped bool
}

// NewManager returns a Manager with no receiver: the air-gapped default, in
// which an alert is stored and nothing leaves the host. recorder is the alert
// repository each Notifier writes delivery outcomes back through.
func NewManager(recorder Recorder, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{recorder: recorder, logger: logger, current: map[alerts.Channel]*Notifier{}}
}

// ApplyWebhook re-points the webhook channel at cfg's receiver. An empty
// cfg.URL turns it off, as does a cfg that cannot produce a signed request,
// with the reason logged.
func (m *Manager) ApplyWebhook(cfg WebhookConfig) {
	if cfg.URL == "" {
		m.swap(alerts.ChannelWebhook, nil)
		return
	}
	cfg.Options = m.fill(cfg.Options)
	n, err := NewWebhook(cfg)
	if err != nil {
		m.logger.Error("alert webhook not configured; alerts will not be delivered", "error", err)
	}
	m.swap(alerts.ChannelWebhook, n)
}

// ApplyEmail re-points the email channel at cfg's mail server. An empty
// cfg.Host turns it off, as does a cfg that could never deliver, with the
// reason logged.
func (m *Manager) ApplyEmail(cfg EmailConfig) {
	if cfg.Host == "" {
		m.swap(alerts.ChannelEmail, nil)
		return
	}
	cfg.Options = m.fill(cfg.Options)
	n, err := NewEmail(cfg)
	if err != nil {
		m.logger.Error("alert email not configured; alerts will not be emailed", "error", err)
	}
	m.swap(alerts.ChannelEmail, n)
}

// fill supplies the Manager's own recorder and logger wherever opts leaves
// them unset, so a caller states only what it is changing.
func (m *Manager) fill(opts Options) Options {
	if opts.Recorder == nil {
		opts.Recorder = m.recorder
	}
	if opts.Logger == nil {
		opts.Logger = m.logger
	}
	return opts
}

// swap installs next (nil turns the channel off) and stops what it replaced.
// The replaced Notifier is stopped outside the lock: Stop waits for an
// in-flight attempt, and a settings write must not hold the swap lock for a
// receiver's timeout.
func (m *Manager) swap(channel alerts.Channel, next *Notifier) {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		if next != nil {
			next.Stop(context.Background())
		}
		return
	}
	previous := m.current[channel]
	if next == nil {
		delete(m.current, channel)
	} else {
		m.current[channel] = next
	}
	m.mu.Unlock()

	if next != nil {
		next.Start()
		m.logger.Info("alert receiver configured", "channel", channel, "endpoint", next.Endpoint())
	}
	if previous != nil {
		previous.Stop(context.Background())
		m.logger.Info("previous alert receiver stopped", "channel", channel, "endpoint", previous.Endpoint())
	}
}

// receivers snapshots the current receivers, in channel order. The writer
// stamps and delivers from one snapshot, so a receiver swapped out between
// the two still answers for the alert it was stamped pending on: a retired
// Notifier records the alert as failed rather than leaving it pending.
func (m *Manager) receivers() []*Notifier {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Notifier, 0, len(m.current))
	for _, n := range m.current {
		out = append(out, n)
	}
	slices.SortFunc(out, func(a, b *Notifier) int { return strings.Compare(string(a.channel), string(b.channel)) })
	return out
}

// Status is each configured channel's delivery counters.
func (m *Manager) Status() map[alerts.Channel]Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[alerts.Channel]Status, len(m.current))
	for channel, n := range m.current {
		out[channel] = n.Status()
	}
	return out
}

// SendTest sends a synthetic alert on channel now and returns the receiver's
// answer. Nothing is stored and no counter moves: it answers "would an alert
// arrive", and the operator is waiting for that answer.
func (m *Manager) SendTest(ctx context.Context, channel alerts.Channel) error {
	m.mu.Lock()
	n := m.current[channel]
	m.mu.Unlock()
	if n == nil {
		return fmt.Errorf("%w: %s", ErrNotConfigured, channel)
	}
	return n.SendNow(ctx, &alerts.Alert{
		Type:      alerts.TypeSystem,
		Severity:  alerts.SeverityInfo,
		Title:     "Test alert",
		Message:   "This is a test alert sent from Seed Settings. No action is needed.",
		Source:    "settings",
		CreatedAt: time.Now().UTC(),
	})
}

// Stop shuts every receiver down and refuses later Apply calls, so a settings
// write racing shutdown cannot start a worker nothing will stop.
func (m *Manager) Stop(ctx context.Context) {
	m.mu.Lock()
	current := m.current
	m.current = map[alerts.Channel]*Notifier{}
	m.stopped = true
	m.mu.Unlock()

	for _, n := range current {
		n.Stop(ctx)
	}
}
