package visibility_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/wifi/airspace"
	wifianomaly "github.com/MustardSeedNetworks/seed/internal/wifi/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/wifi/visibility"
)

// TestRunFeedsWiredMACsToTheRogueAPRule is #2882: the wired source the server
// injects reaches the detector, the status lists the rule as unavailable only
// while the wired set is empty, and a failed refresh keeps the last good set.
func TestRunFeedsWiredMACsToTheRogueAPRule(t *testing.T) {
	store := newFakeAnomalyStore()
	svc := visibility.New(
		visibility.WithCoordinator(wifiCoord(t, store)),
		visibility.WithEvalInterval(time.Hour),
		visibility.WithScanInterval(5*time.Millisecond),
	)
	if got := svc.Status().NeedsWiredData; len(got) != 1 || got[0].ID != wifianomaly.DefRogueAPOnLAN {
		t.Fatalf("before any wired data: NeedsWiredData = %v, want %s", got, wifianomaly.DefRogueAPOnLAN)
	}

	var calls atomic.Int32
	var fail atomic.Bool
	svc.SetWiredSource(func(context.Context) ([]string, error) {
		calls.Add(1)
		if fail.Load() {
			return nil, errors.New("database closed")
		}
		return []string{"00-00-5E-00-53-10"}, nil
	})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- svc.Run(ctx) }()
	defer func() {
		cancel()
		<-done
	}()

	waitFor(t, "a wired refresh", func() bool { return len(svc.Status().NeedsWiredData) == 0 })

	fail.Store(true)
	after := calls.Load()
	waitFor(t, "a failed refresh", func() bool { return calls.Load() > after+1 })
	if got := svc.Status().NeedsWiredData; len(got) != 0 {
		t.Errorf("a failed refresh dropped the wired set: NeedsWiredData = %v", got)
	}

	now := time.Now()
	svc.IngestScan([]airspace.BSSView{scannedBSS(scanBSSID, "corp", "WPA2")}, now)
	svc.Evaluate(t.Context(), now)
	if persisted := recordsToAnomalies(store.snapshot()); !hasAnomaly(persisted, wifianomaly.DefRogueAPOnLAN) {
		t.Errorf("persisted anomalies = %v, want %s", persisted, wifianomaly.DefRogueAPOnLAN)
	}
}

// TestEmptyWiredSetListsTheRuleAsUnavailable keeps an empty refresh distinct
// from a failed one: discovery answering "nothing on the wire" is a real state.
func TestEmptyWiredSetListsTheRuleAsUnavailable(t *testing.T) {
	svc := visibility.New(
		visibility.WithEvalInterval(time.Hour),
		visibility.WithScanInterval(5*time.Millisecond),
	)
	var empty atomic.Bool
	svc.SetWiredSource(func(context.Context) ([]string, error) {
		if empty.Load() {
			return nil, nil
		}
		return []string{scanBSSID}, nil
	})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- svc.Run(ctx) }()
	defer func() {
		cancel()
		<-done
	}()

	waitFor(t, "a wired refresh", func() bool { return len(svc.Status().NeedsWiredData) == 0 })
	empty.Store(true)
	waitFor(t, "the empty set", func() bool { return len(svc.Status().NeedsWiredData) == 1 })
}
