package delivery

import (
	"context"
	"log/slog"
	"sync"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
)

// Manager owns the one receiver the daemon delivers to, and lets an operator
// change it from Settings without restarting (#2605).
//
// The alert store chain is wired once, at startup, around the Manager rather
// than around a Notifier: a Notifier is immutable and not restartable, so a
// chain built around one could only be re-pointed by rebuilding the pipelines.
// The Manager is the indirection that makes the swap possible — every alert
// asks it, at write time, whether there is a receiver.
//
// A configuration that cannot produce a signed request disables delivery and
// is logged, never fatal: the same judgement server_alert_delivery.go has
// always made, that a broken optional integration must not take the
// diagnostics down with it.
type Manager struct {
	recorder Recorder
	logger   *slog.Logger

	mu      sync.Mutex
	current *Notifier
	stopped bool
}

// NewManager returns a Manager with no receiver: the air-gapped default, in
// which an alert is stored and nothing leaves the host. recorder is the alert
// repository each Notifier writes delivery outcomes back through.
func NewManager(recorder Recorder, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{recorder: recorder, logger: logger}
}

// Apply re-points the Manager at cfg's receiver. An empty cfg.URL turns
// delivery off. A cfg that cannot produce a signed request — unparseable URL,
// wrong scheme, or no signing material — also turns it off, with the reason
// logged. The Manager's own recorder and logger fill in whatever cfg leaves
// unset, so a caller states only what it is changing.
//
// The replaced Notifier is stopped outside the lock: Stop waits for an
// in-flight attempt, and a settings write must not hold the swap lock for a
// receiver's timeout.
func (m *Manager) Apply(cfg Config) {
	var next *Notifier
	if cfg.URL != "" {
		if cfg.Recorder == nil {
			cfg.Recorder = m.recorder
		}
		if cfg.Logger == nil {
			cfg.Logger = m.logger
		}
		n, err := New(cfg)
		if err != nil {
			m.logger.Error("alert webhook not configured; alerts will not be delivered",
				"error", err)
		} else {
			next = n
		}
	}

	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		if next != nil {
			next.Stop(context.Background())
		}
		return
	}
	previous := m.current
	m.current = next
	m.mu.Unlock()

	if next != nil {
		next.Start()
		m.logger.Info("alert webhook configured", "endpoint", next.Endpoint())
	}
	if previous != nil {
		previous.Stop(context.Background())
		m.logger.Info("previous alert webhook stopped", "endpoint", previous.Endpoint())
	}
}

// Enabled reports whether a receiver is configured.
func (m *Manager) Enabled() bool { return m.notifier() != nil }

// Status is the current receiver's delivery counters, zero when there is none.
func (m *Manager) Status() Status {
	if n := m.notifier(); n != nil {
		return n.Status()
	}
	return Status{}
}

// Stop shuts the current receiver down and refuses later Apply calls, so a
// settings write racing shutdown cannot start a worker nothing will stop.
func (m *Manager) Stop(ctx context.Context) {
	m.mu.Lock()
	current := m.current
	m.current = nil
	m.stopped = true
	m.mu.Unlock()

	if current != nil {
		current.Stop(ctx)
	}
}

// deliver hands the alert to the current receiver, if there is one.
func (m *Manager) deliver(ctx context.Context, alert *alerts.Alert) {
	if n := m.notifier(); n != nil {
		n.Deliver(ctx, alert)
	}
}

func (m *Manager) notifier() *Notifier {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}
