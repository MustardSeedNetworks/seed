//go:build linux

package vlan_test

import (
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/vlan"
)

// TestManagerConcurrentGetInfoWithLLDP tests concurrent GetInfoWithLLDP calls.
func TestManagerConcurrentGetInfoWithLLDP(t *testing.T) {
	t.Parallel()

	manager := vlan.NewManager("eth0")
	manager.SetConfigured(true, 100)

	var wg sync.WaitGroup
	numGoroutines := 50

	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			nv := id * 10
			vv := id*10 + 5
			for range 100 {
				info := manager.GetInfoWithLLDP(&nv, &vv)
				if info == nil {
					t.Error("GetInfoWithLLDP returned nil")
					return
				}
				if info.NativeVlan == nil || *info.NativeVlan != nv {
					t.Error("NativeVlan mismatch")
				}
				if info.VoiceVlan == nil || *info.VoiceVlan != vv {
					t.Error("VoiceVlan mismatch")
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestContainsWithDuplicates tests contains with duplicate values in slice.
func TestContainsWithDuplicates(t *testing.T) {
	t.Parallel()

	slice := []int{1, 2, 2, 3, 3, 3, 4, 4, 4, 4}

	// All values should be found.
	for _, v := range []int{1, 2, 3, 4} {
		if !vlan.ExportContains(slice, v) {
			t.Errorf("expected to find %d", v)
		}
	}

	// Non-existent values should not be found.
	for _, v := range []int{0, 5, 10, -1} {
		if vlan.ExportContains(slice, v) {
			t.Errorf("did not expect to find %d", v)
		}
	}
}

// TestInfoFieldMutability tests that Info struct fields can be modified.
func TestInfoFieldMutability(t *testing.T) {
	t.Parallel()

	manager := vlan.NewManager("eth0")

	info := manager.GetInfo()

	// Modify fields.
	nv := 100
	vv := 200
	info.NativeVlan = &nv
	info.VoiceVlan = &vv
	info.TaggedVlans = append(info.TaggedVlans, 10, 20, 30)
	info.Configured.Enabled = true
	info.Configured.ID = 999

	// Get new info and verify original manager state is unchanged.
	info2 := manager.GetInfo()
	if info2.NativeVlan != nil {
		t.Error("original NativeVlan should be nil")
	}
	if info2.VoiceVlan != nil {
		t.Error("original VoiceVlan should be nil")
	}
	if len(info2.TaggedVlans) != 0 {
		t.Error("original TaggedVlans should be empty")
	}
	if info2.Configured.Enabled {
		t.Error("original Configured.Enabled should be false")
	}
}

// BenchmarkVLANCoverage benchmarks the main code paths.
func BenchmarkVLANCoverage(b *testing.B) {
	b.Run("FullManagerCycle", func(b *testing.B) {
		for b.Loop() {
			m := vlan.NewManager("eth0")
			m.SetConfigured(true, 100)
			m.SetInterface("en0")
			_ = m.GetInfo()
			nv := 1
			vv := 100
			_ = m.GetInfoWithLLDP(&nv, &vv)
			_ = m.DetectVlanSubinterfaces("eth0")
		}
	})

	b.Run("ConcurrentManagerAccess", func(b *testing.B) {
		m := vlan.NewManager("eth0")
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.SetConfigured(true, 100)
				_ = m.GetInfo()
			}
		})
	})
}
