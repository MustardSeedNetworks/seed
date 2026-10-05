// Package vulntriage is the use-case behind vulnerability triage (S6-2, #899,
// ADR-0020): the persisted findings, an operator's status decision on one, and
// the finding's remediation history. Persistence is reached through the Store
// port, implemented by internal/database's VulnerabilityRepository and wired in
// internal/app.
package vulntriage

import (
	"context"
	"errors"
	"time"
)

// Status is a finding's triage state; migration 00019's CHECK holds the column
// to these four values.
type Status string

// The scanner owns new and resolved; the operator owns acknowledged and
// ignored, and may move a finding back to new.
const (
	// StatusNew is a finding the last scan pass reported and no operator has
	// triaged.
	StatusNew Status = "new"
	// StatusAcknowledged is a finding an operator has seen. It stays open.
	StatusAcknowledged Status = "acknowledged"
	// StatusIgnored is a finding an operator accepted or judged a false
	// positive. It is kept but leaves the open counts.
	StatusIgnored Status = "ignored"
	// StatusResolved marks a finding a later scan pass of its device no longer
	// reported.
	StatusResolved Status = "resolved"
)

// Valid reports whether s names one of the four finding states.
func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusAcknowledged, StatusIgnored, StatusResolved:
		return true
	}
	return false
}

// ReasonMaxLen bounds the free-text reason kept in the history.
const ReasonMaxLen = 500

var (
	// ErrFindingNotFound is returned for a finding id that does not exist.
	ErrFindingNotFound = errors.New("vulnerability finding not found")
	// ErrTransition is returned for a status change the operator may not make:
	// to resolved (the scanner decides that), from resolved (nothing is left to
	// triage), or to the status the finding already has.
	ErrTransition = errors.New("vulnerability status change not allowed")
	// ErrReasonRequired is returned when a finding is ignored without a reason;
	// an ignored finding leaves the reports, so the record says why.
	ErrReasonRequired = errors.New("ignoring a vulnerability finding requires a reason")
	// ErrInvalidStatus is returned for a status that is not one of the four.
	ErrInvalidStatus = errors.New("unknown vulnerability status")
	// ErrReasonTooLong is returned for a reason over ReasonMaxLen bytes.
	ErrReasonTooLong = errors.New("vulnerability triage reason too long")
)

// Finding is a persisted finding with its device's address.
type Finding struct {
	ID                int64
	DeviceID          string
	DeviceIP          string
	Hostname          string
	CVEID             string
	Severity          string
	CVSSScore         float64
	Description       string
	AffectedComponent string
	AffectedVersion   string
	Status            Status
	DetectedAt        time.Time
	ResolvedAt        *time.Time
}

// StatusChange is one entry of a finding's remediation history. An empty Actor
// is the scanner.
type StatusChange struct {
	From      Status
	To        Status
	Actor     string
	Reason    string
	ChangedAt time.Time
}

// ListOptions filters List; zero values do not filter.
type ListOptions struct {
	Status   Status
	DeviceID string
	Limit    int
	Offset   int
}

// Store reads and writes the persisted findings. SetStatus enforces the
// transition rules (ErrTransition, ErrReasonRequired) against the finding's
// current state, and returns ErrFindingNotFound as History does.
type Store interface {
	ListFindings(ctx context.Context, opts ListOptions) ([]Finding, error)
	SetStatus(ctx context.Context, id int64, to Status, actor, reason string, at time.Time) error
	History(ctx context.Context, id int64) ([]StatusChange, error)
}

// Service answers the triage routes from a Store.
type Service struct {
	store Store
}

// NewService builds the triage use-case over store.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// List returns persisted findings, most severe first.
func (s *Service) List(ctx context.Context, opts ListOptions) ([]Finding, error) {
	return s.store.ListFindings(ctx, opts)
}

// SetStatus records an operator's decision on finding id, now.
func (s *Service) SetStatus(ctx context.Context, id int64, to Status, actor, reason string) error {
	if !to.Valid() {
		return ErrInvalidStatus
	}
	if len(reason) > ReasonMaxLen {
		return ErrReasonTooLong
	}
	return s.store.SetStatus(ctx, id, to, actor, reason, time.Now())
}

// History returns finding id's status changes, oldest first.
func (s *Service) History(ctx context.Context, id int64) ([]StatusChange, error) {
	return s.store.History(ctx, id)
}
