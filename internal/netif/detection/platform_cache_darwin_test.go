//go:build darwin

package detection

import (
	"sync/atomic"
	"testing"
)

func TestIdentifyByPlatformCachesSystemProfile(t *testing.T) {
	var calls atomic.Int32
	db := NewChipsetDatabase()
	db.identifyPlatform = func(string) *ChipsetInfo {
		calls.Add(1)
		return &db.chipsets[0]
	}

	first := db.identifyByPlatform("en0")
	second := db.identifyByPlatform("en0")
	if calls.Load() != 1 {
		t.Fatalf("platform identification calls = %d, want 1", calls.Load())
	}
	if first == nil || second != first {
		t.Fatal("cached platform identification was not reused")
	}
}
