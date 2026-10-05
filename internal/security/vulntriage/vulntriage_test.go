package vulntriage_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/security/vulntriage"
)

// recordingStore records the SetStatus call it receives.
type recordingStore struct {
	vulntriage.Store

	calls int
	to    vulntriage.Status
	at    time.Time
}

func (r *recordingStore) SetStatus(
	_ context.Context, _ int64, to vulntriage.Status, _, _ string, at time.Time,
) error {
	r.calls++
	r.to, r.at = to, at
	return nil
}

func TestSetStatusValidatesBeforeTheStore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		to     vulntriage.Status
		reason string
		want   error
	}{
		{name: "acknowledged", to: vulntriage.StatusAcknowledged},
		{
			name: "ignored at the reason limit", to: vulntriage.StatusIgnored,
			reason: strings.Repeat("x", vulntriage.ReasonMaxLen),
		},
		{name: "unknown status", to: "fixed", want: vulntriage.ErrInvalidStatus},
		{name: "empty status", to: "", want: vulntriage.ErrInvalidStatus},
		{
			name: "reason too long", to: vulntriage.StatusIgnored,
			reason: strings.Repeat("x", vulntriage.ReasonMaxLen+1), want: vulntriage.ErrReasonTooLong,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := &recordingStore{}
			before := time.Now()
			err := vulntriage.NewService(store).SetStatus(context.Background(), 1, tc.to, "alice", tc.reason)
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
				require.Zero(t, store.calls, "a refused request never reaches the store")
				return
			}
			require.NoError(t, err)
			require.Equal(t, 1, store.calls)
			require.Equal(t, tc.to, store.to)
			require.False(t, store.at.Before(before), "the decision is stamped now")
		})
	}
}

func TestStatusValid(t *testing.T) {
	t.Parallel()
	for _, s := range []vulntriage.Status{
		vulntriage.StatusNew, vulntriage.StatusAcknowledged, vulntriage.StatusIgnored, vulntriage.StatusResolved,
	} {
		require.True(t, s.Valid(), s)
	}
	for _, s := range []vulntriage.Status{"", "open", "New"} {
		require.False(t, s.Valid(), s)
	}
}
