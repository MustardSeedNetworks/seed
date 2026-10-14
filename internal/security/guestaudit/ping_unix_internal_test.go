//go:build !windows

package guestaudit

import (
	"slices"
	"testing"
	"time"
)

// macOS reads ping's -W as milliseconds and Linux as whole seconds; neither
// may stop waiting before the audit's budget does (seed#3310).
func TestPingArgsWaitUnitPerGOOS(t *testing.T) {
	tests := []struct {
		goos    string
		timeout time.Duration
		want    []string
	}{
		{"linux", 2 * time.Second, []string{"-c", "1", "-W", "2", "192.0.2.1"}},
		{"linux", 1500 * time.Millisecond, []string{"-c", "1", "-W", "2", "192.0.2.1"}},
		{"linux", 300 * time.Millisecond, []string{"-c", "1", "-W", "1", "192.0.2.1"}},
		{"darwin", 2 * time.Second, []string{"-c", "1", "-W", "2000", "192.0.2.1"}},
		{"darwin", 1500 * time.Millisecond, []string{"-c", "1", "-W", "1500", "192.0.2.1"}},
		{"darwin", 300 * time.Millisecond, []string{"-c", "1", "-W", "300", "192.0.2.1"}},
	}
	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.timeout.String(), func(t *testing.T) {
			got := pingArgs(tt.goos, "192.0.2.1", tt.timeout)
			if !slices.Equal(got, tt.want) {
				t.Errorf("pingArgs(%q, %v) = %q, want %q", tt.goos, tt.timeout, got, tt.want)
			}
		})
	}
}
