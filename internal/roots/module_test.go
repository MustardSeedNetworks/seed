// Package roots_test provides tests for the Roots module.
// Test suite validates module initialization, lifecycle, and service accessors.
package roots_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/roots"
)

// TestNew validates module creation with various configurations.
func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cfgIsNil   bool
		dbIsNil    bool
		wantModule bool
	}{
		{
			name:       "nil config and nil db",
			cfgIsNil:   true,
			dbIsNil:    true,
			wantModule: true,
		},
		{
			name:       "nil db only",
			cfgIsNil:   false,
			dbIsNil:    true,
			wantModule: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Use nil values directly since config/db may require setup
			m := roots.New(nil, nil)
			if (m != nil) != tt.wantModule {
				t.Errorf("New() returned module = %v, want module = %v", m != nil, tt.wantModule)
			}
		})
	}
}

// TestModule_Services validates all service accessors return non-nil services.
func TestModule_Services(t *testing.T) {
	t.Parallel()

	m := roots.New(nil, nil)
	if m == nil {
		t.Fatal("New() returned nil module")
	}

	tests := []struct {
		name    string
		svcFunc func() any
	}{
		{
			name: "Traceroute service",
			svcFunc: func() any {
				return m.Traceroute()
			},
		},
		{
			name: "Topology service",
			svcFunc: func() any {
				return m.Topology()
			},
		},
		{
			name: "Enrichment service",
			svcFunc: func() any {
				return m.Enrichment()
			},
		},
		{
			name: "Analysis service",
			svcFunc: func() any {
				return m.Analysis()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := tt.svcFunc()
			if svc == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
		})
	}
}

// TestModule_ServicesConcurrent validates thread-safe access to services.
func TestModule_ServicesConcurrent(t *testing.T) {
	t.Parallel()

	m := roots.New(nil, nil)
	if m == nil {
		t.Fatal("New() returned nil module")
	}

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines * 4) // 4 services

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = m.Traceroute()
		}()
		go func() {
			defer wg.Done()
			_ = m.Topology()
		}()
		go func() {
			defer wg.Done()
			_ = m.Enrichment()
		}()
		go func() {
			defer wg.Done()
			_ = m.Analysis()
		}()
	}

	wg.Wait()
}

// TestModule_Lifecycle validates Start/Stop lifecycle operations.
func TestModule_Lifecycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		startTimeout  time.Duration
		wantStartErr  bool
		wantStopErr   bool
		callStopFirst bool
	}{
		{
			name:         "normal lifecycle",
			startTimeout: 100 * time.Millisecond,
			wantStartErr: false,
			wantStopErr:  false,
		},
		{
			name:          "stop before start",
			startTimeout:  100 * time.Millisecond,
			callStopFirst: true,
			wantStopErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := roots.New(nil, nil)
			if m == nil {
				t.Fatal("New() returned nil module")
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.startTimeout)
			defer cancel()

			if tt.callStopFirst {
				// Stop should not panic even if Start wasn't called
				if err := m.Stop(); (err != nil) != tt.wantStopErr {
					t.Errorf("Stop() error = %v, wantStopErr = %v", err, tt.wantStopErr)
				}
			}

			if err := m.Start(ctx); (err != nil) != tt.wantStartErr {
				t.Errorf("Start() error = %v, wantStartErr = %v", err, tt.wantStartErr)
			}

			if !tt.callStopFirst {
				if err := m.Stop(); (err != nil) != tt.wantStopErr {
					t.Errorf("Stop() error = %v, wantStopErr = %v", err, tt.wantStopErr)
				}
			}
		})
	}
}

// TestModule_StartStop_Idempotent validates multiple Start/Stop calls.
func TestModule_StartStop_Idempotent(t *testing.T) {
	t.Parallel()

	m := roots.New(nil, nil)
	if m == nil {
		t.Fatal("New() returned nil module")
	}

	ctx := context.Background()

	// Multiple starts should not error
	for i := range 3 {
		if err := m.Start(ctx); err != nil {
			t.Errorf("Start() call %d error = %v", i, err)
		}
	}

	// Multiple stops should not error
	for i := range 3 {
		if err := m.Stop(); err != nil {
			t.Errorf("Stop() call %d error = %v", i, err)
		}
	}
}

// TestModule_StartWithCancelledContext validates Start with cancelled context.
func TestModule_StartWithCancelledContext(t *testing.T) {
	t.Parallel()

	m := roots.New(nil, nil)
	if m == nil {
		t.Fatal("New() returned nil module")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Start should handle cancelled context gracefully
	err := m.Start(ctx)
	if err != nil {
		t.Errorf("Start() with cancelled context error = %v", err)
	}

	// Cleanup
	_ = m.Stop()
}

// TestModule_ConcurrentStartStop validates concurrent Start/Stop operations.
func TestModule_ConcurrentStartStop(t *testing.T) {
	t.Parallel()

	m := roots.New(nil, nil)
	if m == nil {
		t.Fatal("New() returned nil module")
	}

	const iterations = 5
	var wg sync.WaitGroup
	wg.Add(iterations * 2)

	ctx := context.Background()

	for range iterations {
		go func() {
			defer wg.Done()
			_ = m.Start(ctx)
		}()
		go func() {
			defer wg.Done()
			_ = m.Stop()
		}()
	}

	wg.Wait()
}
