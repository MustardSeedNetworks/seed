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
// delivery. Wrapping the store rather than each pipeline means every alert
// Seed persists is delivered, including the ones a future pipeline writes.
//
// An alert the store rejected is never delivered: the inbox is the record of
// what happened, and a receiver must not learn about an alert an operator
// cannot then find.
func WrapWriter(store Writer, n *Notifier) Writer {
	if n == nil {
		return store
	}
	return &deliveringWriter{store: store, notifier: n}
}

type deliveringWriter struct {
	store    Writer
	notifier *Notifier
}

func (w *deliveringWriter) Create(ctx context.Context, alert *alerts.Alert) error {
	if err := w.store.Create(ctx, alert); err != nil {
		return err
	}
	w.notifier.Deliver(alert)
	return nil
}
