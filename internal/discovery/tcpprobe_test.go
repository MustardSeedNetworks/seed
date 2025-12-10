package discovery

import (
	"context"
	"testing"
	"time"
)

func TestTCPProber_ProbeTCP(t *testing.T) {
	prober, err := NewTCPProber(2 * time.Second)
	if err != nil {
		t.Fatalf("failed to create prober: %v", err)
	}
	defer prober.Close()

	// Test probe to a well-known open port (Google DNS)
	ctx := context.Background()
	result := prober.ProbeTCP(ctx, "8.8.8.8", 53)

	// Should get a result (either open or filtered depending on network)
	if result.IP != "8.8.8.8" {
		t.Errorf("expected IP 8.8.8.8, got %s", result.IP)
	}
	if result.Port != 53 {
		t.Errorf("expected port 53, got %d", result.Port)
	}
}

func TestTCPProber_ProbeClosedPort(t *testing.T) {
	prober, err := NewTCPProber(1 * time.Second)
	if err != nil {
		t.Fatalf("failed to create prober: %v", err)
	}
	defer prober.Close()

	// Test probe to localhost on unlikely port
	ctx := context.Background()
	result := prober.ProbeTCP(ctx, "127.0.0.1", 59999)

	// Should be closed on localhost
	if result.State != PortClosed {
		t.Logf("expected closed state, got %s (may vary by system config)", result.State)
	}
}

func TestTCPProber_ScanPorts(t *testing.T) {
	prober, err := NewTCPProber(1 * time.Second)
	if err != nil {
		t.Fatalf("failed to create prober: %v", err)
	}
	defer prober.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ports := []int{22, 80, 443}
	results := prober.ScanPorts(ctx, "127.0.0.1", ports, 3)

	if len(results) != len(ports) {
		t.Errorf("expected %d results, got %d", len(ports), len(results))
	}

	// Verify each result has correct port
	for i, result := range results {
		if result.Port != ports[i] {
			t.Errorf("result %d: expected port %d, got %d", i, ports[i], result.Port)
		}
	}
}

func TestPortState_String(t *testing.T) {
	tests := []struct {
		state    PortState
		expected string
	}{
		{PortOpen, "open"},
		{PortClosed, "closed"},
		{PortFiltered, "filtered"},
	}

	for _, tt := range tests {
		if string(tt.state) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.state)
		}
	}
}

func TestCommonPorts(t *testing.T) {
	// Verify common ports slice is populated
	if len(CommonPorts) == 0 {
		t.Error("CommonPorts should not be empty")
	}

	// Verify it contains expected ports
	expected := map[int]bool{22: true, 80: true, 443: true}
	found := make(map[int]bool)
	for _, port := range CommonPorts {
		if expected[port] {
			found[port] = true
		}
	}

	for port := range expected {
		if !found[port] {
			t.Errorf("CommonPorts should contain port %d", port)
		}
	}
}
