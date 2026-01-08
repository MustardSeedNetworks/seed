package vlan_test

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/krisarmstrong/seed/internal/sap/vlan"
)

// TestManagerConcurrencyStress tests concurrent access under high load.
func TestManagerConcurrencyStress(t *testing.T) {
	t.Parallel()

	manager := vlan.NewManager("eth0")

	var wg sync.WaitGroup
	numGoroutines := 50
	iterationsPerGoroutine := 100

	// Concurrent writers for SetInterface.
	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for j := range iterationsPerGoroutine {
				iface := "eth" + string(rune('0'+id%10))
				manager.SetInterface(iface)
				manager.SetConfigured(j%2 == 0, j%1000)
			}
		}(i)
	}

	// Concurrent readers for GetInfo.
	wg.Add(numGoroutines)
	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range iterationsPerGoroutine {
				info := manager.GetInfo()
				if info == nil {
					t.Error("GetInfo returned nil during concurrent access")
				}
			}
		}()
	}

	wg.Wait()
}

// TestManagerGetInfoWithLLDPConcurrent tests concurrent GetInfoWithLLDP.
func TestManagerGetInfoWithLLDPConcurrent(t *testing.T) {
	t.Parallel()

	manager := vlan.NewManager("eth0")

	var wg sync.WaitGroup
	numGoroutines := 20

	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			nativeVlan := id * 10
			voiceVlan := id*10 + 5
			for range 50 {
				info := manager.GetInfoWithLLDP(&nativeVlan, &voiceVlan)
				if info == nil {
					t.Error("GetInfoWithLLDP returned nil")
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestInfoJSONSerialization tests Info struct JSON serialization.
func TestInfoJSONSerialization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		info vlan.Info
	}{
		{
			name: "empty info",
			info: vlan.Info{},
		},
		{
			name: "with native vlan only",
			info: func() vlan.Info {
				nv := 100
				return vlan.Info{NativeVlan: &nv}
			}(),
		},
		{
			name: "with voice vlan only",
			info: func() vlan.Info {
				vv := 200
				return vlan.Info{VoiceVlan: &vv}
			}(),
		},
		{
			name: "with tagged vlans",
			info: vlan.Info{TaggedVlans: []int{10, 20, 30, 40, 50}},
		},
		{
			name: "with configured enabled",
			info: func() vlan.Info {
				info := vlan.Info{}
				info.Configured.Enabled = true
				info.Configured.ID = 100
				return info
			}(),
		},
		{
			name: "fully populated",
			info: func() vlan.Info {
				nv := 1
				vv := 100
				info := vlan.Info{
					NativeVlan:  &nv,
					TaggedVlans: []int{10, 20, 30},
					VoiceVlan:   &vv,
				}
				info.Configured.Enabled = true
				info.Configured.ID = 50
				return info
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Marshal to JSON.
			data, err := json.Marshal(tt.info)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}

			// Unmarshal back.
			var decoded vlan.Info
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}

			// Verify key fields.
			if tt.info.NativeVlan != nil {
				if decoded.NativeVlan == nil || *decoded.NativeVlan != *tt.info.NativeVlan {
					t.Error("NativeVlan mismatch after JSON round-trip")
				}
			}
			if tt.info.VoiceVlan != nil {
				if decoded.VoiceVlan == nil || *decoded.VoiceVlan != *tt.info.VoiceVlan {
					t.Error("VoiceVlan mismatch after JSON round-trip")
				}
			}
			if len(tt.info.TaggedVlans) != len(decoded.TaggedVlans) {
				t.Error("TaggedVlans length mismatch after JSON round-trip")
			}
		})
	}
}

// TestManagerStateTransitions tests state transitions.
func TestManagerStateTransitions(t *testing.T) {
	t.Parallel()

	manager := vlan.NewManager("eth0")

	// Initial state.
	if manager.ManagerEnabled() {
		t.Error("expected disabled initially")
	}
	if manager.ManagerConfiguredID() != 0 {
		t.Error("expected configuredID 0 initially")
	}

	// Enable -> Disable -> Enable.
	transitions := []struct {
		enabled   bool
		id        int
		wantState bool
		wantID    int
	}{
		{true, 100, true, 100},
		{false, 0, false, 0},
		{true, 200, true, 200},
		{true, 300, true, 300},   // Change ID while enabled.
		{false, 100, false, 100}, // Disable with different ID.
		{true, 0, true, 0},       // Enable with ID 0.
	}

	for i, tr := range transitions {
		manager.SetConfigured(tr.enabled, tr.id)
		if manager.ManagerEnabled() != tr.wantState {
			t.Errorf("transition %d: enabled = %v, want %v", i, manager.ManagerEnabled(), tr.wantState)
		}
		if manager.ManagerConfiguredID() != tr.wantID {
			t.Errorf("transition %d: configuredID = %d, want %d", i, manager.ManagerConfiguredID(), tr.wantID)
		}
	}
}

// TestManagerInterfaceNameVariants tests various interface name formats.
func TestManagerInterfaceNameVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		iface string
	}{
		{"standard ethernet", "eth0"},
		{"macos ethernet", "en0"},
		{"wireless", "wlan0"},
		{"bond", "bond0"},
		{"bridge", "br0"},
		{"docker bridge", "docker0"},
		{"veth pair", "veth123abc"},
		{"macvlan", "macvlan0"},
		{"vlan subinterface", "eth0.100"},
		{"loopback", "lo"},
		{"tun device", "tun0"},
		{"tap device", "tap0"},
		{"infiniband", "ib0"},
		{"team device", "team0"},
		{"dummy", "dummy0"},
		{"virtual", "virbr0"},
		{"openvswitch", "ovs-system"},
		{"wireguard", "wg0"},
		{"empty name", ""},
		{"long name", "verylonginterfacenamethatshouldstillwork"},
		{"numeric only", "12345"},
		{"with hyphen", "eth-0"},
		{"with underscore", "eth_0"},
		{"with colon", "eth0:0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			manager := vlan.NewManager(tt.iface)
			if manager.ManagerInterfaceName() != tt.iface {
				t.Errorf("interface name = %q, want %q", manager.ManagerInterfaceName(), tt.iface)
			}

			// Test SetInterface too.
			manager.SetInterface(tt.iface + "_modified")
			if manager.ManagerInterfaceName() != tt.iface+"_modified" {
				t.Errorf("after SetInterface: interface name = %q, want %q",
					manager.ManagerInterfaceName(), tt.iface+"_modified")
			}
		})
	}
}

// TestManagerGetInfoImmutability tests that GetInfo returns independent copies.
func TestManagerGetInfoImmutability(t *testing.T) {
	t.Parallel()

	manager := vlan.NewManager("eth0")
	manager.SetConfigured(true, 100)

	info1 := manager.GetInfo()
	info2 := manager.GetInfo()

	// Modify info1.
	info1.TaggedVlans = append(info1.TaggedVlans, 200, 300, 400)
	info1.Configured.Enabled = false
	info1.Configured.ID = 999

	// info2 should be unchanged.
	if len(info2.TaggedVlans) != 0 {
		t.Error("info2.TaggedVlans was affected by modifying info1")
	}
	// Note: Configured is a value type, so modifications to info1.Configured
	// don't affect info2.Configured.
}

// TestManagerDetectVlanSubinterfacesConsistency tests detect consistency.
func TestManagerDetectVlanSubinterfacesConsistency(t *testing.T) {
	t.Parallel()

	manager := vlan.NewManager("eth0")

	// Multiple calls should return consistent results.
	results := make([][]int, 10)
	for i := range results {
		results[i] = manager.DetectVlanSubinterfaces("eth0")
	}

	// All results should have the same length.
	for i := 1; i < len(results); i++ {
		if len(results[i]) != len(results[0]) {
			t.Errorf("inconsistent results: call %d returned %d items, call 0 returned %d",
				i, len(results[i]), len(results[0]))
		}
	}
}

// TestContainsAllVLANIDs tests contains with all possible VLAN IDs.
func TestContainsAllVLANIDs(t *testing.T) {
	t.Parallel()

	// Create slice with all valid VLAN IDs.
	allVLANs := make([]int, 4095)
	for i := range allVLANs {
		allVLANs[i] = i
	}

	// Test finding each.
	tests := []struct {
		name     string
		val      int
		expected bool
	}{
		{"VLAN 0", 0, true},
		{"VLAN 1", 1, true},
		{"VLAN 100", 100, true},
		{"VLAN 1000", 1000, true},
		{"VLAN 4094 (max valid)", 4094, true},
		{"VLAN 4095 (not in slice)", 4095, false},
		{"VLAN 5000", 5000, false},
		{"negative VLAN", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := vlan.ExportContains(allVLANs, tt.val)
			if result != tt.expected {
				t.Errorf("Contains(allVLANs, %d) = %v, want %v", tt.val, result, tt.expected)
			}
		})
	}
}

// TestVLANIDRanges tests VLAN ID handling at boundaries.
func TestVLANIDRanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		vlanID int
		valid  bool // Whether it's a valid 802.1Q VLAN ID (0-4094).
	}{
		{"VLAN 0 (null/untagged)", 0, true},
		{"VLAN 1 (default)", 1, true},
		{"VLAN 2", 2, true},
		{"VLAN 100", 100, true},
		{"VLAN 1000", 1000, true},
		{"VLAN 4093", 4093, true},
		{"VLAN 4094 (max)", 4094, true},
		{"VLAN 4095 (reserved)", 4095, false},
		{"negative VLAN", -1, false},
		{"large VLAN", 10000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// The manager accepts any int, but we can track valid ranges.
			manager := vlan.NewManager("eth0")
			manager.SetConfigured(true, tt.vlanID)

			if manager.ManagerConfiguredID() != tt.vlanID {
				t.Errorf("configuredID = %d, want %d", manager.ManagerConfiguredID(), tt.vlanID)
			}

			// Note: The package doesn't validate VLAN ID ranges; that's
			// the caller's responsibility. We just verify it stores the value.
		})
	}
}

// TestCreateDeleteVlanInterfaceEdgeCases tests edge cases for interface operations.
func TestCreateDeleteVlanInterfaceEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		parentIf string
		vlanID   int
	}{
		{"VLAN 0", "eth0", 0},
		{"VLAN 1", "eth0", 1},
		{"VLAN 4094", "eth0", 4094},
		{"VLAN 4095", "eth0", 4095},
		{"large VLAN ID", "eth0", 65535},
		{"negative VLAN ID", "eth0", -1},
		{"empty parent", "", 100},
		{"unusual parent name", "verylonginterfacename", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// These operations require root and may fail, but should not panic.
			_ = vlan.CreateVlanInterface(tt.parentIf, tt.vlanID)
			_ = vlan.DeleteVlanInterface(tt.parentIf, tt.vlanID)
		})
	}
}

// TestManagerGetInfoWithLLDPVariations tests GetInfoWithLLDP variations.
func TestManagerGetInfoWithLLDPVariations(t *testing.T) {
	t.Parallel()

	manager := vlan.NewManager("eth0")
	manager.SetConfigured(true, 100)

	tests := []struct {
		name       string
		nativeVlan *int
		voiceVlan  *int
		wantNative bool
		wantVoice  bool
	}{
		{"both nil", nil, nil, false, false},
		{"only native", intPtr(10), nil, true, false},
		{"only voice", nil, intPtr(50), false, true},
		{"both set", intPtr(1), intPtr(100), true, true},
		{"zero native", intPtr(0), nil, true, false},
		{"zero voice", nil, intPtr(0), false, true},
		{"same values", intPtr(100), intPtr(100), true, true},
		{"max VLAN native", intPtr(4094), nil, true, false},
		{"max VLAN voice", nil, intPtr(4094), false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			info := manager.GetInfoWithLLDP(tt.nativeVlan, tt.voiceVlan)

			hasNative := info.NativeVlan != nil
			hasVoice := info.VoiceVlan != nil

			if hasNative != tt.wantNative {
				t.Errorf("NativeVlan presence = %v, want %v", hasNative, tt.wantNative)
			}
			if hasVoice != tt.wantVoice {
				t.Errorf("VoiceVlan presence = %v, want %v", hasVoice, tt.wantVoice)
			}

			if tt.nativeVlan != nil && info.NativeVlan != nil {
				if *info.NativeVlan != *tt.nativeVlan {
					t.Errorf("NativeVlan = %d, want %d", *info.NativeVlan, *tt.nativeVlan)
				}
			}
			if tt.voiceVlan != nil && info.VoiceVlan != nil {
				if *info.VoiceVlan != *tt.voiceVlan {
					t.Errorf("VoiceVlan = %d, want %d", *info.VoiceVlan, *tt.voiceVlan)
				}
			}
		})
	}
}

// intPtr is a helper to create int pointers.
func intPtr(i int) *int {
	return &i
}

// BenchmarkManagerOperations benchmarks various manager operations.
func BenchmarkManagerOperations(b *testing.B) {
	b.Run("NewManager", func(b *testing.B) {
		for b.Loop() {
			_ = vlan.NewManager("eth0")
		}
	})

	b.Run("SetInterface", func(b *testing.B) {
		manager := vlan.NewManager("eth0")
		b.ResetTimer()
		for b.Loop() {
			manager.SetInterface("en0")
		}
	})

	b.Run("SetConfigured", func(b *testing.B) {
		manager := vlan.NewManager("eth0")
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			manager.SetConfigured(i%2 == 0, i%1000)
		}
	})

	b.Run("GetInfo", func(b *testing.B) {
		manager := vlan.NewManager("eth0")
		manager.SetConfigured(true, 100)
		b.ResetTimer()
		for b.Loop() {
			_ = manager.GetInfo()
		}
	})

	b.Run("GetInfoWithLLDP", func(b *testing.B) {
		manager := vlan.NewManager("eth0")
		nv := 10
		vv := 50
		b.ResetTimer()
		for b.Loop() {
			_ = manager.GetInfoWithLLDP(&nv, &vv)
		}
	})

	b.Run("DetectVlanSubinterfaces", func(b *testing.B) {
		manager := vlan.NewManager("eth0")
		b.ResetTimer()
		for b.Loop() {
			_ = manager.DetectVlanSubinterfaces("eth0")
		}
	})
}

// BenchmarkManagerConcurrentAccess benchmarks concurrent access patterns.
func BenchmarkManagerConcurrentAccess(b *testing.B) {
	manager := vlan.NewManager("eth0")

	b.Run("ReadHeavy", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = manager.GetInfo()
			}
		})
	})

	b.Run("WriteHeavy", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				manager.SetConfigured(i%2 == 0, i%1000)
				i++
			}
		})
	})

	b.Run("Mixed", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				if i%3 == 0 {
					manager.SetConfigured(i%2 == 0, i%1000)
				} else {
					_ = manager.GetInfo()
				}
				i++
			}
		})
	})
}
