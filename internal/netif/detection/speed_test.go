package detection_test

// Test suite for cross-platform speed detection and parsing functions.

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/netif/detection"
)

func TestCalculateSpeedBonus(t *testing.T) {
	tests := []struct {
		name  string
		speed int64
		want  int
	}{
		{"100 Gbps", 100_000_000_000, 500},
		{"50 Gbps (between 40G and 100G)", 50_000_000_000, 450},
		{"40 Gbps", 40_000_000_000, 450},
		{"30 Gbps (between 25G and 40G)", 30_000_000_000, 425},
		{"25 Gbps", 25_000_000_000, 425},
		{"15 Gbps (between 10G and 25G)", 15_000_000_000, 400},
		{"10 Gbps", 10_000_000_000, 400},
		{"7 Gbps (between 5G and 10G)", 7_000_000_000, 350},
		{"5 Gbps", 5_000_000_000, 350},
		{"3 Gbps (between 2.5G and 5G)", 3_000_000_000, 300},
		{"2.5 Gbps", 2_500_000_000, 300},
		{"1.5 Gbps (between 1G and 2.5G)", 1_500_000_000, 200},
		{"1 Gbps", 1_000_000_000, 200},
		{"500 Mbps (between 100M and 1G)", 500_000_000, 100},
		{"100 Mbps", 100_000_000, 100},
		{"50 Mbps (below 100M)", 50_000_000, 0},
		{"10 Mbps", 10_000_000, 0},
		{"0 (no speed)", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detection.CalculateSpeedBonus(tt.speed)
			if got != tt.want {
				t.Errorf("CalculateSpeedBonus(%d) = %d, want %d", tt.speed, got, tt.want)
			}
		})
	}
}

func TestGetSpeedBonuses(t *testing.T) {
	bonuses := detection.GetSpeedBonuses()

	// Verify we have the expected number of thresholds.
	if len(bonuses) != 8 {
		t.Errorf("GetSpeedBonuses() returned %d entries, want 8", len(bonuses))
	}

	// Verify bonuses are in descending order by speed.
	for i := 1; i < len(bonuses); i++ {
		if bonuses[i].MinSpeed >= bonuses[i-1].MinSpeed {
			t.Errorf("Bonuses not in descending order: %d >= %d at index %d",
				bonuses[i].MinSpeed, bonuses[i-1].MinSpeed, i)
		}
	}

	// Verify bonuses are in descending order by bonus amount.
	for i := 1; i < len(bonuses); i++ {
		if bonuses[i].Bonus >= bonuses[i-1].Bonus {
			t.Errorf("Bonus amounts not in descending order: %d >= %d at index %d",
				bonuses[i].Bonus, bonuses[i-1].Bonus, i)
		}
	}

	// Verify specific known values.
	expectedFirst := struct {
		MinSpeed int64
		Bonus    int
	}{100_000_000_000, 500}
	if bonuses[0] != expectedFirst {
		t.Errorf("First bonus = %+v, want %+v", bonuses[0], expectedFirst)
	}

	expectedLast := struct {
		MinSpeed int64
		Bonus    int
	}{100_000_000, 100}
	if bonuses[len(bonuses)-1] != expectedLast {
		t.Errorf("Last bonus = %+v, want %+v", bonuses[len(bonuses)-1], expectedLast)
	}
}

func TestSpeedConstants(t *testing.T) {
	// Verify speed constants are correctly defined.
	tests := []struct {
		name     string
		constant int64
		want     int64
	}{
		{"Speed100Gbps", detection.Speed100Gbps, 100_000_000_000},
		{"Speed40Gbps", detection.Speed40Gbps, 40_000_000_000},
		{"Speed25Gbps", detection.Speed25Gbps, 25_000_000_000},
		{"Speed10Gbps", detection.Speed10Gbps, 10_000_000_000},
		{"Speed5Gbps", detection.Speed5Gbps, 5_000_000_000},
		{"Speed2500Mbps", detection.Speed2500Mbps, 2_500_000_000},
		{"Speed1Gbps", detection.Speed1Gbps, 1_000_000_000},
		{"Speed100Mbps", detection.Speed100Mbps, 100_000_000},
		{"Speed10Mbps", detection.Speed10Mbps, 10_000_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.constant, tt.want)
			}
		})
	}
}
