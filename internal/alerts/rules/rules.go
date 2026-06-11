// Package rules holds the alert-rule application (use-case) layer (ADR-0020). It
// owns the operator-defined alert-rule orchestration (and, over time, the alert
// acknowledge/resolve flow) that previously reached into the database
// repositories from the api.Server handlers, behind a narrow consumer-defined
// Store port. Handlers keep transport concerns: request decode, field
// validation, JSON encoding, and error→HTTP mapping. The adapter satisfying the
// Store port lives in the composition root (internal/app).
package rules

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when no alert rule has the given id.
var ErrNotFound = errors.New("alert rule not found")

// ValidationError carries a repository-level validation message verbatim so the
// handler can echo it as a 400, preserving the pre-strangle response body.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

// Rule is the use-case alert-rule model. The adapter maps it to/from
// database.AlertRule so the app layer stays free of persistence types.
type Rule struct {
	ID                   int64
	Name                 string
	Enabled              bool
	MatchKind            string
	MatchSeverity        string
	MatchPayloadContains string
	AlertType            string
	AlertSeverity        string
	AlertTitle           string
	AlertMessage         string
	WindowSeconds        int
	ThresholdCount       int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// Store is the persistence surface the rule use-case needs, defined at the
// consumer (ADR-0020). The adapter satisfies it over the database AlertRules
// repository, mapping ErrAlertRuleNotFound -> ErrNotFound and the repo's
// "alert_rules:"-prefixed validation errors -> *ValidationError.
type Store interface {
	List(ctx context.Context, enabledOnly bool) ([]Rule, error)
	Get(ctx context.Context, id int64) (Rule, error)
	// Create stores r and returns it with the generated ID + timestamps.
	Create(ctx context.Context, r Rule) (Rule, error)
	Update(ctx context.Context, r Rule) error
	Delete(ctx context.Context, id int64) error
}

// Service is the alert-rule use-case.
type Service struct {
	store Store
}

// NewService builds the use-case over its Store port.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// minThreshold mirrors the pre-strangle max(threshold, 1): a rule fires on the
// first matching event unless a higher count is set.
func minThreshold(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// List returns alert rules, optionally only the enabled ones.
func (s *Service) List(ctx context.Context, enabledOnly bool) ([]Rule, error) {
	return s.store.List(ctx, enabledOnly)
}

// Get returns one alert rule (ErrNotFound when absent).
func (s *Service) Get(ctx context.Context, id int64) (Rule, error) {
	return s.store.Get(ctx, id)
}

// Create stores a new rule and returns the persisted row.
func (s *Service) Create(ctx context.Context, r Rule) (Rule, error) {
	r.ID = 0
	r.ThresholdCount = minThreshold(r.ThresholdCount)
	return s.store.Create(ctx, r)
}

// Update writes the rule at id and returns the freshly-read row so callers see
// the new updated_at; on a post-write read failure it echoes the written rule.
func (s *Service) Update(ctx context.Context, id int64, r Rule) (Rule, error) {
	r.ID = id
	r.ThresholdCount = minThreshold(r.ThresholdCount)
	if err := s.store.Update(ctx, r); err != nil {
		return Rule{}, err
	}
	// Best-effort re-read so the echo reflects updated_at; the write already
	// succeeded, so a failed re-read falls back to the written rule.
	if current, getErr := s.store.Get(ctx, id); getErr == nil {
		return current, nil
	}
	return r, nil
}

// Delete removes the rule at id (ErrNotFound when absent).
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.store.Delete(ctx, id)
}
