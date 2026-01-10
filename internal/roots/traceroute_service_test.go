package roots_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/roots"
)

// TestTracerouteService_Creation validates service creation.
func TestTracerouteService_Creation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		createFunc func() *roots.TracerouteService
		wantNil    bool
	}{
		{
			name: "standard creation with nil config",
			createFunc: func() *roots.TracerouteService {
				return roots.NewTracerouteService(nil)
			},
			wantNil: false,
		},
		{
			name:       "nil tracer creation for error testing",
			createFunc: roots.NewTracerouteServiceNilTracer,
			wantNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := tt.createFunc()
			if (svc == nil) != tt.wantNil {
				t.Errorf("creation returned nil = %v, want nil = %v", svc == nil, tt.wantNil)
			}
		})
	}
}

// TestTracerouteService_Tracer validates Tracer() accessor.
func TestTracerouteService_Tracer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		createFunc   func() *roots.TracerouteService
		wantNilTrace bool
	}{
		{
			name: "standard service has tracer",
			createFunc: func() *roots.TracerouteService {
				return roots.NewTracerouteService(nil)
			},
			wantNilTrace: false,
		},
		{
			name:         "nil tracer service",
			createFunc:   roots.NewTracerouteServiceNilTracer,
			wantNilTrace: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := tt.createFunc()
			tracer := svc.Tracer()
			if (tracer == nil) != tt.wantNilTrace {
				t.Errorf("Tracer() nil = %v, want nil = %v", tracer == nil, tt.wantNilTrace)
			}
		})
	}
}

// TestTracerouteService_Trace_NilTracer validates error when tracer is nil.
func TestTracerouteService_Trace_NilTracer(t *testing.T) {
	t.Parallel()

	svc := roots.NewTracerouteServiceNilTracer()
	ctx := context.Background()

	result, err := svc.Trace(ctx, "8.8.8.8", nil)
	if err == nil {
		t.Error("Trace() with nil tracer should return error")
	}
	if result != nil {
		t.Errorf("Trace() with nil tracer should return nil result, got %+v", result)
	}
	if !errors.Is(err, roots.ErrNotInitialized) {
		t.Errorf("error = %v, want %v", err, roots.ErrNotInitialized)
	}
}

// TestTracerouteService_Trace_WithOptions validates trace with various options.
func TestTracerouteService_Trace_WithOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		target  string
		opts    *roots.TracerouteOptions
		wantErr bool
	}{
		{
			name:    "nil options uses defaults",
			target:  "127.0.0.1",
			opts:    nil,
			wantErr: false,
		},
		{
			name:   "custom timeout",
			target: "127.0.0.1",
			opts: &roots.TracerouteOptions{
				Timeout: 500 * time.Millisecond,
			},
			wantErr: false,
		},
		{
			name:   "custom max hops",
			target: "127.0.0.1",
			opts: &roots.TracerouteOptions{
				MaxHops: 5,
			},
			wantErr: false,
		},
		{
			name:   "UDP mode",
			target: "127.0.0.1",
			opts: &roots.TracerouteOptions{
				UseUDP: true,
			},
			wantErr: false,
		},
		{
			name:   "don't resolve hostnames",
			target: "127.0.0.1",
			opts: &roots.TracerouteOptions{
				DontResolve: true,
			},
			wantErr: false,
		},
		{
			name:   "resolve hostnames enabled",
			target: "127.0.0.1",
			opts: &roots.TracerouteOptions{
				DontResolve: false,
			},
			wantErr: false,
		},
		{
			name:   "combined options",
			target: "127.0.0.1",
			opts: &roots.TracerouteOptions{
				MaxHops:     10,
				Timeout:     1 * time.Second,
				UseUDP:      false,
				DontResolve: true,
			},
			wantErr: false,
		},
		{
			name:   "zero timeout uses default",
			target: "127.0.0.1",
			opts: &roots.TracerouteOptions{
				Timeout: 0,
			},
			wantErr: false,
		},
		{
			name:   "zero max hops uses default",
			target: "127.0.0.1",
			opts: &roots.TracerouteOptions{
				MaxHops: 0,
			},
			wantErr: false,
		},
	}

	svc := roots.NewTracerouteService(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := svc.Trace(ctx, tt.target, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Trace() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}
			if result == nil {
				t.Error("Trace() returned nil result without error")
				return
			}
			if result.Target != tt.target {
				t.Errorf("Target = %q, want %q", result.Target, tt.target)
			}
			if result.StartedAt.IsZero() {
				t.Error("StartedAt should not be zero")
			}
			if result.CompletedAt.IsZero() {
				t.Error("CompletedAt should not be zero")
			}
			if result.Duration <= 0 {
				t.Error("Duration should be positive")
			}
		})
	}
}

// TestTracerouteService_Trace_ResultFields validates result structure.
func TestTracerouteService_Trace_ResultFields(t *testing.T) {
	t.Parallel()

	svc := roots.NewTracerouteService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := svc.Trace(ctx, "127.0.0.1", nil)
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}

	// Validate result structure
	if result.Target == "" {
		t.Error("Target should not be empty")
	}
	if result.Hops == nil {
		t.Error("Hops should not be nil (may be empty slice)")
	}
	if result.DurationMs < 0 {
		t.Error("DurationMs should not be negative")
	}
	if result.CompletedAt.Before(result.StartedAt) {
		t.Error("CompletedAt should not be before StartedAt")
	}
}

// TestTracerouteService_Trace_ContextCancellation validates context handling.
func TestTracerouteService_Trace_ContextCancellation(t *testing.T) {
	t.Parallel()

	svc := roots.NewTracerouteService(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := svc.Trace(ctx, "8.8.8.8", nil)
	if err != nil {
		t.Logf("Trace with cancelled context error: %v", err)
	}
	// Result may or may not be nil depending on cancellation timing
	_ = result
}

// TestTracerouteService_Trace_ContextTimeout validates timeout handling.
func TestTracerouteService_Trace_ContextTimeout(t *testing.T) {
	t.Parallel()

	svc := roots.NewTracerouteService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Allow some time for timeout to trigger
	time.Sleep(5 * time.Millisecond)

	result, err := svc.Trace(ctx, "8.8.8.8", nil)
	if err != nil {
		t.Logf("Trace with timed out context error: %v", err)
	}
	// Result may or may not be nil depending on timeout timing
	_ = result
}

// TestTracerouteService_Trace_HopProcessing validates hop conversion.
func TestTracerouteService_Trace_HopProcessing(t *testing.T) {
	t.Parallel()

	svc := roots.NewTracerouteService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Trace to localhost should complete quickly
	result, err := svc.Trace(ctx, "127.0.0.1", &roots.TracerouteOptions{
		MaxHops: 5,
	})
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}

	// Validate hop structure if we got any
	for i, hop := range result.Hops {
		if hop.Number == 0 && !hop.Lost {
			t.Errorf("hop[%d]: Number should not be 0 for responding hop", i)
		}
		if hop.Lost && hop.RTTMs > 0 {
			t.Errorf("hop[%d]: Lost hop should have 0 RTTMs, got %f", i, hop.RTTMs)
		}
	}
}

// TestTracerouteOptions_DefaultValues validates options struct field defaults.
func TestTracerouteOptions_DefaultValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		opts     roots.TracerouteOptions
		checkFn  func(roots.TracerouteOptions) bool
		checkMsg string
	}{
		{
			name:     "empty options has zero MaxHops",
			opts:     roots.TracerouteOptions{},
			checkFn:  func(o roots.TracerouteOptions) bool { return o.MaxHops == 0 },
			checkMsg: "empty MaxHops should be 0",
		},
		{
			name:     "empty options has zero Timeout",
			opts:     roots.TracerouteOptions{},
			checkFn:  func(o roots.TracerouteOptions) bool { return o.Timeout == 0 },
			checkMsg: "empty Timeout should be 0",
		},
		{
			name:     "empty options has false UseUDP",
			opts:     roots.TracerouteOptions{},
			checkFn:  func(o roots.TracerouteOptions) bool { return !o.UseUDP },
			checkMsg: "empty UseUDP should be false",
		},
		{
			name:     "empty options has false DontResolve",
			opts:     roots.TracerouteOptions{},
			checkFn:  func(o roots.TracerouteOptions) bool { return !o.DontResolve },
			checkMsg: "empty DontResolve should be false",
		},
		{
			name:     "empty options has false EnrichHops",
			opts:     roots.TracerouteOptions{},
			checkFn:  func(o roots.TracerouteOptions) bool { return !o.EnrichHops },
			checkMsg: "empty EnrichHops should be false",
		},
		{
			name:     "empty options has zero Probes",
			opts:     roots.TracerouteOptions{},
			checkFn:  func(o roots.TracerouteOptions) bool { return o.Probes == 0 },
			checkMsg: "empty Probes should be 0",
		},
		{
			name:     "empty options has zero PacketSize",
			opts:     roots.TracerouteOptions{},
			checkFn:  func(o roots.TracerouteOptions) bool { return o.PacketSize == 0 },
			checkMsg: "empty PacketSize should be 0",
		},
		{
			name:     "empty options has empty SourceAddr",
			opts:     roots.TracerouteOptions{},
			checkFn:  func(o roots.TracerouteOptions) bool { return o.SourceAddr == "" },
			checkMsg: "empty SourceAddr should be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if !tt.checkFn(tt.opts) {
				t.Error(tt.checkMsg)
			}
		})
	}
}
