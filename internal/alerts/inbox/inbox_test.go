package inbox_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/inbox"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
)

type fakeRepo struct {
	listOpts  alerts.ListOptions
	effects   []*alerts.Alert
	causeIDs  []int64
	ackID     int64
	ackUser   string
	resolveID int64
	peaks     ifrate.ErrorPeaks
	peakReads []peakRead
	peaksErr  error
	err       error
}

type peakRead struct {
	target   string
	ifIndex  uint32
	from, to time.Time
}

func (f *fakeRepo) List(_ context.Context, opts alerts.ListOptions) ([]*alerts.Alert, error) {
	f.listOpts = opts
	return []*alerts.Alert{{ID: 1}}, f.err
}

func (f *fakeRepo) ListEffects(_ context.Context, causeIDs []int64) ([]*alerts.Alert, error) {
	f.causeIDs = causeIDs
	return f.effects, f.err
}

func (f *fakeRepo) DeviceNames(context.Context) (map[string]string, error) {
	return map[string]string{"tgt-1": "core-sw1"}, f.err
}

func (f *fakeRepo) InterfaceErrorPeaks(
	_ context.Context, target string, ifIndex uint32, from, to time.Time,
) (ifrate.ErrorPeaks, error) {
	f.peakReads = append(f.peakReads, peakRead{target, ifIndex, from, to})
	return f.peaks, f.peaksErr
}

func (f *fakeRepo) Acknowledge(_ context.Context, id int64, user string) error {
	f.ackID, f.ackUser = id, user
	return f.err
}

func (f *fakeRepo) Resolve(_ context.Context, id int64) error {
	f.resolveID = id
	return f.err
}

func TestServiceDelegates(t *testing.T) {
	repo := &fakeRepo{}
	svc := inbox.NewService(repo)
	ctx := context.Background()

	alerts, err := svc.List(ctx, alerts.ListOptions{Severity: "warning"})
	if err != nil || len(alerts) != 1 || repo.listOpts.Severity != "warning" {
		t.Errorf("List did not delegate: alerts=%v err=%v opts=%+v", alerts, err, repo.listOpts)
	}
	if err = svc.Acknowledge(ctx, 7, "alice"); err != nil || repo.ackID != 7 || repo.ackUser != "alice" {
		t.Errorf("Acknowledge did not delegate: err=%v id=%d user=%q", err, repo.ackID, repo.ackUser)
	}
	if err = svc.Resolve(ctx, 9); err != nil || repo.resolveID != 9 {
		t.Errorf("Resolve did not delegate: err=%v id=%d", err, repo.resolveID)
	}
}

func TestServicePropagatesRepoError(t *testing.T) {
	wantErr := errors.New("boom")
	svc := inbox.NewService(&fakeRepo{err: wantErr})
	if _, err := svc.List(context.Background(), alerts.ListOptions{}); !errors.Is(err, wantErr) {
		t.Errorf("repo error not propagated: %v", err)
	}
}

func TestNarrativesExplainCausesWithTheirEffects(t *testing.T) {
	causeID := int64(1)
	downAt := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	cause := &alerts.Alert{
		ID: causeID, Rule: alerts.RuleInterfaceDown, Source: "tgt-1", CreatedAt: downAt,
		Metadata: `{"ifIndex":2,"ifName":"eth1","ifOperStatus":2}`,
	}
	// The effect is outside the page; it is found by its cause.
	effect := &alerts.Alert{
		ID: 5, Rule: alerts.RuleBGPFlap, Source: "sw1", RootCauseID: &causeID,
		Metadata: `{"remoteAddr":"192.0.2.9","remoteAs":64512,"state":3}`,
	}
	explained := &alerts.Alert{ID: 4, Rule: alerts.RuleBGPFlap, RootCauseID: &causeID, Metadata: effect.Metadata}
	operator := &alerts.Alert{ID: 3, Rule: "db.2", Metadata: `{}`}
	repo := &fakeRepo{effects: []*alerts.Alert{effect}, peaks: ifrate.ErrorPeaks{Polls: 3, InErrors: 0.5}}

	got, err := inbox.NewService(repo).Narratives(context.Background(),
		[]*alerts.Alert{explained, operator, cause})
	if err != nil {
		t.Fatal(err)
	}
	if want := []int64{3, 1}; !slices.Equal(repo.causeIDs, want) {
		t.Errorf("effects read for %v, want only the uncaused alerts %v", repo.causeIDs, want)
	}
	if len(got) != 1 {
		t.Fatalf("narratives for %d alerts, want only the cause: %+v", len(got), got)
	}
	// Only the interface-down cause reads counters, over the window ending
	// when it went down.
	wantRead := peakRead{"tgt-1", 2, downAt.Add(-15 * time.Minute), downAt}
	if len(repo.peakReads) != 1 || repo.peakReads[0] != wantRead {
		t.Errorf("counter reads = %+v, want only %+v", repo.peakReads, wantRead)
	}
	n := got[causeID]
	if n.Summary.Data["peers"] != 1 || len(n.Evidence) != 3 {
		t.Errorf("cause narrative did not include its counters and effect: %+v", n)
	}
	if n.Evidence[1].Data["counter"] != "ifInErrors" {
		t.Errorf("evidence[1] = %+v, want the input error peak", n.Evidence[1])
	}
	if n.Summary.Data["device"] != "core-sw1" {
		t.Errorf("device = %v, want the polling target's name", n.Summary.Data["device"])
	}
}

func TestNarrativesPropagatesRepoError(t *testing.T) {
	wantErr := errors.New("boom")
	svc := inbox.NewService(&fakeRepo{err: wantErr})
	if _, err := svc.Narratives(context.Background(), []*alerts.Alert{{ID: 1}}); !errors.Is(err, wantErr) {
		t.Errorf("repo error not propagated: %v", err)
	}
}

func TestNarrativesPropagatesCounterReadError(t *testing.T) {
	wantErr := errors.New("boom")
	svc := inbox.NewService(&fakeRepo{peaksErr: wantErr})
	cause := &alerts.Alert{
		ID:       1,
		Rule:     alerts.RuleInterfaceDown,
		Metadata: `{"ifIndex":2,"ifName":"eth1","ifOperStatus":2}`,
	}
	if _, err := svc.Narratives(context.Background(), []*alerts.Alert{cause}); !errors.Is(err, wantErr) {
		t.Errorf("counter read error not propagated: %v", err)
	}
}
