package inbox_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/alerts/inbox"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

type fakeRepo struct {
	listOpts  database.AlertListOptions
	ackID     int64
	ackUser   string
	resolveID int64
	err       error
}

func (f *fakeRepo) List(_ context.Context, opts database.AlertListOptions) ([]*database.Alert, error) {
	f.listOpts = opts
	return []*database.Alert{{ID: 1}}, f.err
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

	alerts, err := svc.List(ctx, database.AlertListOptions{Severity: "warning"})
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
	if _, err := svc.List(context.Background(), database.AlertListOptions{}); !errors.Is(err, wantErr) {
		t.Errorf("repo error not propagated: %v", err)
	}
}
