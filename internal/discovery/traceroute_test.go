package discovery

import (
	"context"
	"testing"
	"time"
)

func TestNewTracer(t *testing.T) {
	// Test with default values
	tracer := NewTracer(0, 0)
	if tracer.timeout != 3*time.Second {
		t.Errorf("expected default timeout 3s, got %v", tracer.timeout)
	}
	if tracer.maxHops != 30 {
		t.Errorf("expected default maxHops 30, got %d", tracer.maxHops)
	}

	// Test with custom values
	tracer = NewTracer(5*time.Second, 20)
	if tracer.timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", tracer.timeout)
	}
	if tracer.maxHops != 20 {
		t.Errorf("expected maxHops 20, got %d", tracer.maxHops)
	}
}

func TestTracer_TraceICMP_InvalidTarget(t *testing.T) {
	tracer := NewTracer(1*time.Second, 5)
	ctx := context.Background()

	// Test with invalid hostname
	result := tracer.TraceICMP(ctx, "invalid.hostname.that.does.not.exist.example")
	if result.Error == "" {
		t.Error("expected error for invalid hostname")
	}
	if result.Completed {
		t.Error("should not be completed for invalid target")
	}
}

func TestTracer_TraceUDP_InvalidTarget(t *testing.T) {
	tracer := NewTracer(1*time.Second, 5)
	ctx := context.Background()

	result := tracer.TraceUDP(ctx, "invalid.hostname.that.does.not.exist.example", 33434)
	if result.Error == "" {
		t.Error("expected error for invalid hostname")
	}
}

func TestTracer_TraceTCP_InvalidTarget(t *testing.T) {
	tracer := NewTracer(1*time.Second, 5)
	ctx := context.Background()

	result := tracer.TraceTCP(ctx, "invalid.hostname.that.does.not.exist.example", 80)
	if result.Error == "" {
		t.Error("expected error for invalid hostname")
	}
}

func TestTracer_TraceICMP_Localhost(t *testing.T) {
	tracer := NewTracer(2*time.Second, 5)
	ctx := context.Background()

	result := tracer.TraceICMP(ctx, "127.0.0.1")

	if result.Target != "127.0.0.1" {
		t.Errorf("expected target 127.0.0.1, got %s", result.Target)
	}
	if result.TargetIP != "127.0.0.1" {
		t.Errorf("expected targetIP 127.0.0.1, got %s", result.TargetIP)
	}
	if result.Protocol != "icmp" {
		t.Errorf("expected protocol icmp, got %s", result.Protocol)
	}
	// Localhost trace may or may not complete depending on system config
	t.Logf("Localhost trace result: completed=%v, hops=%d, error=%s",
		result.Completed, len(result.Hops), result.Error)
}

func TestTracer_ContextCancellation(t *testing.T) {
	tracer := NewTracer(5*time.Second, 30)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	result := tracer.TraceICMP(ctx, "8.8.8.8")
	if result.Error != "traceroute canceled" {
		t.Logf("trace result: %+v", result)
	}
}

func TestTracerouteResult_Structure(t *testing.T) {
	result := &TracerouteResult{
		Target:    "example.com",
		TargetIP:  "93.184.216.34",
		Protocol:  "icmp",
		Hops:      make([]TracerouteHop, 0),
		Completed: false,
	}

	if result.Target != "example.com" {
		t.Errorf("unexpected target: %s", result.Target)
	}
}

func TestTracerouteHop_Structure(t *testing.T) {
	hop := TracerouteHop{
		TTL:      5,
		IP:       "192.168.1.1",
		Hostname: "router.local",
		RTT:      10 * time.Millisecond,
		State:    "reply",
	}

	if hop.TTL != 5 {
		t.Errorf("unexpected TTL: %d", hop.TTL)
	}
	if hop.State != "reply" {
		t.Errorf("unexpected state: %s", hop.State)
	}
}
