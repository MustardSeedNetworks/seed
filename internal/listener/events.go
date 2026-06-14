package listener

import "time"

// EventRecord is one persisted listener_events row — the durable form of an
// [Event] after the sink writes it. It carries the storage-assigned ID and
// ingest timestamp that the in-flight Event does not. The type lives here (not
// in internal/database) so the sink and the alert pipeline that read/write
// these rows stay persistence-free: the listener-events repository in
// internal/database imports this package and maps SQL rows to it.
type EventRecord struct {
	ID          int64
	ClientID    string
	Kind        string
	SourceAddr  string
	TargetKind  string
	TargetID    string
	Severity    string
	ObservedAt  time.Time
	PayloadJSON string
	IngestedAt  time.Time
}

// EventListOptions narrows a listener-events list query — empty values disable
// that filter.
type EventListOptions struct {
	ClientID   string
	Kind       string
	SourceAddr string
	Since      time.Time
	Until      time.Time
	Limit      int
}
