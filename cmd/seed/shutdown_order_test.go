package main

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// recordingServer records the drain against a shared order slice.
type recordingServer struct {
	order *[]string
	err   error
}

func (r *recordingServer) Shutdown(context.Context) error {
	*r.order = append(*r.order, "drain")
	return r.err
}

// TestStopServingDrainsBeforeStoppingComponents pins #2748's cmd-layer half:
// components.Stop() used to run first, so the report scheduler, the Wi-Fi loops
// and the outbox relay were gone before the HTTP server had drained and a
// request still in flight ran against stopped components.
func TestStopServingDrainsBeforeStoppingComponents(t *testing.T) {
	tests := []struct {
		name          string
		drainErr      error
		componentsErr error
		hasComponents bool
		want          []string
	}{
		{
			name:          "drain then components",
			hasComponents: true,
			want:          []string{"drain", "components"},
		},
		{
			name:          "a failed drain still stops the components",
			drainErr:      errors.New("deadline exceeded"),
			hasComponents: true,
			want:          []string{"drain", "components"},
		},
		{
			name:          "a failing component stop is reported, not fatal",
			componentsErr: errors.New("relay stuck"),
			hasComponents: true,
			want:          []string{"drain", "components"},
		},
		{
			name: "no components is not a nil call",
			want: []string{"drain"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var order []string

			var stopComponents func() error
			if tt.hasComponents {
				stopComponents = func() error {
					order = append(order, "components")
					return tt.componentsErr
				}
			}

			stopServing(t.Context(), &recordingServer{order: &order, err: tt.drainErr}, stopComponents)

			if !slices.Equal(order, tt.want) {
				t.Errorf("shutdown order = %v, want %v", order, tt.want)
			}
		})
	}
}
