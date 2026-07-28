//go:build darwin

package detection_test

import (
	"sync/atomic"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/netif/detection"
)

func TestIdentifyByPlatformCachesSystemProfile(t *testing.T) {
	var calls atomic.Int32
	db := detection.NewChipsetDatabase()
	chipsets := db.GetAll()
	db.SetPlatformIdentifier(func(string) *detection.ChipsetInfo {
		calls.Add(1)
		return &chipsets[0]
	})

	first := db.IdentifyByInterface("en0", "")
	second := db.IdentifyByInterface("en0", "")
	if calls.Load() != 1 {
		t.Fatalf("platform identification calls = %d, want 1", calls.Load())
	}
	if first == nil || second != first {
		t.Fatal("cached platform identification was not reused")
	}
}
