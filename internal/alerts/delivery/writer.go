package delivery

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
)

// Writer is the alert-store surface the alert pipelines write through — the
// same narrow interface they already declare, restated here so the decorator
// does not import the pipeline package.
type Writer interface {
	Create(ctx context.Context, alert *alerts.Alert) error
}

// WrapWriter returns a Writer that stores the alert and then queues it for
// delivery on every configured channel. Wrapping the store rather than each
// pipeline means every alert Seed persists is delivered, including the ones a
// future pipeline writes.
//
// The decorator wraps the Manager, not a Notifier, and is installed whether or
// not a receiver is configured: receivers are operator settings (#2605), so
// the chain has to outlive any particular one.
//
// An alert the store rejected is never delivered: the inbox is the record of
// what happened, and a receiver must not learn about an alert an operator
// cannot then find.
func WrapWriter(store Writer, m *Manager) Writer {
	if m == nil {
		return store
	}
	return &deliveringWriter{store: store, manager: m}
}

type deliveringWriter struct {
	store   Writer
	manager *Manager
}

func (w *deliveringWriter) Create(ctx context.Context, alert *alerts.Alert) error {
	// Stamp the state before the insert, not after: one write, and the inbox
	// never shows an alert whose delivery state is missing rather than
	// pending. Only configured channels are stamped, so an alert raised while
	// delivery is off has no delivery state at all — the distinction the inbox
	// reads as "nobody ever tried to send this".
	receivers := w.manager.receivers()
	for _, n := range receivers {
		alert.Deliveries = append(alert.Deliveries,
			alerts.Delivery{Channel: n.Channel(), Status: alerts.DeliveryPending})
	}
	if err := w.store.Create(ctx, alert); err != nil {
		return err
	}
	for _, n := range receivers {
		n.Deliver(ctx, alert)
	}
	return nil
}
