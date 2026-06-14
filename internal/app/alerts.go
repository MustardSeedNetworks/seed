package app

// alerts.go wires the composition root to the alert-rule application (use-case)
// service (ADR-0020). The adapter implements the narrow rules.Store port over
// the database AlertRules repository, mapping the alerts.Rule row to the
// use-case model and the repo's ErrAlertRuleNotFound / "alert_rules:"-prefixed
// validation errors to the use-case's sentinels.

import (
	"context"
	"errors"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/rules"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

// NewAlertRules builds the alert-rule use-case (ADR-0020) over the lazy db
// accessor, assembling the rules.Store adapter over the database repository.
// The database is resolved through db on each call so the api test harness's
// later-set DB is honored; handlers gate on db() != nil first.
func NewAlertRules(db func() *database.DB) *rules.Service {
	return rules.NewService(alertRuleStore{db: db})
}

// alertRuleStore implements rules.Store over the database repository. The
// database is resolved lazily; handlers gate on db() != nil first.
type alertRuleStore struct {
	db func() *database.DB
}

func (a alertRuleStore) Available() bool { return a.db() != nil }

func toAppRule(r *alerts.Rule) rules.Rule {
	return rules.Rule{
		ID:                   r.ID,
		Name:                 r.Name,
		Enabled:              r.Enabled,
		MatchKind:            r.MatchKind,
		MatchSeverity:        r.MatchSeverity,
		MatchPayloadContains: r.MatchPayloadContains,
		AlertType:            r.AlertType,
		AlertSeverity:        r.AlertSeverity,
		AlertTitle:           r.AlertTitle,
		AlertMessage:         r.AlertMessage,
		WindowSeconds:        r.WindowSeconds,
		ThresholdCount:       r.ThresholdCount,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

func toDBRule(r rules.Rule) *alerts.Rule {
	return &alerts.Rule{
		ID:                   r.ID,
		Name:                 r.Name,
		Enabled:              r.Enabled,
		MatchKind:            r.MatchKind,
		MatchSeverity:        r.MatchSeverity,
		MatchPayloadContains: r.MatchPayloadContains,
		AlertType:            r.AlertType,
		AlertSeverity:        r.AlertSeverity,
		AlertTitle:           r.AlertTitle,
		AlertMessage:         r.AlertMessage,
		WindowSeconds:        r.WindowSeconds,
		ThresholdCount:       r.ThresholdCount,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

func mapAlertRuleErr(err error) error {
	if errors.Is(err, alerts.ErrRuleNotFound) {
		return rules.ErrNotFound
	}
	return err
}

func (a alertRuleStore) List(ctx context.Context, enabledOnly bool) ([]rules.Rule, error) {
	rows, err := a.db().AlertRules().List(ctx, enabledOnly)
	if err != nil {
		return nil, err
	}
	out := make([]rules.Rule, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAppRule(r))
	}
	return out, nil
}

func (a alertRuleStore) Get(ctx context.Context, id int64) (rules.Rule, error) {
	r, err := a.db().AlertRules().Get(ctx, id)
	if err != nil {
		return rules.Rule{}, mapAlertRuleErr(err)
	}
	return toAppRule(r), nil
}

func (a alertRuleStore) Create(ctx context.Context, r rules.Rule) (rules.Rule, error) {
	row := toDBRule(r)
	if err := a.db().AlertRules().Create(ctx, row); err != nil {
		// The repo signals validation failures with an "alert_rules:" prefix;
		// surface them verbatim so the handler returns a 400 with the message.
		if strings.HasPrefix(err.Error(), "alert_rules:") {
			return rules.Rule{}, &rules.ValidationError{Msg: err.Error()}
		}
		return rules.Rule{}, err
	}
	return toAppRule(row), nil
}

func (a alertRuleStore) Update(ctx context.Context, r rules.Rule) error {
	return mapAlertRuleErr(a.db().AlertRules().Update(ctx, toDBRule(r)))
}

func (a alertRuleStore) Delete(ctx context.Context, id int64) error {
	return mapAlertRuleErr(a.db().AlertRules().Delete(ctx, id))
}
