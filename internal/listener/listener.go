package listener

import "context"

// Listener is one passive ingress endpoint. Implementations live in
// internal/listener/{snmp_trap,syslog,...}/.
//
// Lifecycle: Bind opens the configured port; the listener runs in a
// background goroutine emitting Events on the registered channel
// until Stop is called.
type Listener interface {
	// Name is a stable identifier (e.g. "snmp_trap", "syslog").
	Name() string

	// Bind opens the configured listening socket and begins emitting
	// Events. Returns an error if the bind fails (port in use, etc.).
	// Honors ctx cancellation for ongoing operation.
	Bind(ctx context.Context, cfg Config) error

	// Stop closes the listening socket and waits up to ctx deadline
	// for in-flight message processing to complete.
	Stop(ctx context.Context) error

	// Subscribe returns a channel that receives Events from this
	// Listener. The Registry typically owns the subscription
	// fan-out; individual listeners may not be subscribed directly.
	Subscribe() <-chan Event
}
