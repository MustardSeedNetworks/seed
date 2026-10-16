package autotest

// record.go is the stored side of a run (P-E2): every completed run is kept as
// a record, and a record's JSON export is the Report itself, so importing it
// back restores the same record on this or another Seed.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Store errors. The HTTP layer maps ErrNotFound to 404, ErrExists to 409 and
// ErrUnavailable to 503.
var (
	ErrNotFound    = errors.New("autotest: record not found")
	ErrExists      = errors.New("autotest: record already exists")
	ErrUnavailable = errors.New("autotest: record store unavailable")
)

// Store keeps run records.
type Store interface {
	// Insert stores a new record; a record with the same ID is ErrExists.
	Insert(ctx context.Context, r *Report) error
	Get(ctx context.Context, id string) (*Report, error)
	// List returns records newest first.
	List(ctx context.Context) ([]Report, error)
}

func (n StageName) known() bool {
	switch n {
	case StageLink, StageCable, StageDHCP, StageDNS, StageGateway, StageVLAN, StageInternet:
		return true
	}
	return false
}

func (o Outcome) known() bool {
	switch o {
	case OutcomePass, OutcomeWarn, OutcomeFail, OutcomeSkipped:
		return true
	}
	return false
}

// ParseRecord decodes a JSON export for import. It refuses unknown fields and
// anything a run could not have produced, so an import is either the record
// that was exported or an error.
func ParseRecord(data []byte) (*Report, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var r Report
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("decode record: %w", err)
	}
	if dec.More() {
		return nil, errors.New("decode record: trailing data after the record")
	}
	if err := uuid.Validate(r.ID); err != nil {
		return nil, fmt.Errorf("id: %w", err)
	}
	if r.Interface == "" || r.StartedAt.IsZero() || r.DurationMs < 0 {
		return nil, errors.New("interface, startedAt and a non-negative durationMs are required")
	}
	if err := (Spec{JackLabel: r.JackLabel, AssetTag: r.AssetTag}).Validate(); err != nil {
		return nil, err
	}
	for _, st := range r.Stages {
		if !st.Stage.known() || !st.Outcome.known() {
			return nil, fmt.Errorf("stage %q outcome %q is not one a run reports", st.Stage, st.Outcome)
		}
	}
	if r.Verdict != verdict(r.Stages) {
		return nil, fmt.Errorf("verdict %q does not follow from the stages", r.Verdict)
	}
	r.StartedAt = r.StartedAt.UTC()
	return &r, nil
}
