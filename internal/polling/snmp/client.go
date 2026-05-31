package snmp

import "context"

// Varbind is one (OID, value) result from a Get or Walk. Value is the
// raw decoded gosnmp value — Collectors that need typed bytes / ints
// type-assert at the call site.
type Varbind struct {
	OID   string
	Value any
}

// Client is the narrowed gosnmp surface the collector chain needs.
// One Client per Target (or per chain run) — Collectors do not own
// the connection lifecycle; the Poller does.
//
// Implementations live in internal/polling/snmp/snmpclient/. Tests
// inject a fake to drive collector behavior without a live device.
type Client interface {
	// Get retrieves one or more scalar OIDs. Order of the returned
	// slice matches order of the input. A non-existent OID yields a
	// Varbind with a nil Value rather than an error.
	Get(ctx context.Context, oids []string) ([]Varbind, error)

	// Walk traverses a subtree rooted at prefix and returns every
	// non-end-of-mib varbind. Callers MUST treat Walk as bounded by
	// ctx — a misbehaving agent that never returns end-of-mib must
	// not hang the collector chain forever.
	Walk(ctx context.Context, prefix string) ([]Varbind, error)
}

// ClientFactory builds a Client for a Target + decoded credentials.
// Stage A3.x's real factory dials gosnmp; tests inject a factory
// that returns a fake Client.
type ClientFactory func(target Target, creds ResolvedCredentials) (Client, error)
