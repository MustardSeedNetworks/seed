package iperf_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/iperf"
)

// TestGetVersionReadsTheResolvedBinaryOnce pins #2530: GetVersion used to run
// `iperf3 --version` on every call, after resolution had already run it once,
// and every exec from the daemon can wedge its child on macOS. The cached path
// does not exist, so any exec would fail; the version must come from the cache.
func TestGetVersionReadsTheResolvedBinaryOnce(t *testing.T) {
	iperf.KeepIperfBinary(t)
	iperf.SetIperfBinary("/nonexistent/iperf3", "v3.17")

	for range 3 {
		got, err := iperf.GetVersion()
		if err != nil {
			t.Fatalf("GetVersion ran the binary instead of reading the cache: %v", err)
		}
		if got != "v3.17" {
			t.Fatalf("GetVersion() = %q, want the cached v3.17", got)
		}
	}
}

func TestParseVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		out  string
		want string
	}{
		{"release line", "iperf 3.16 (cJSON 1.7.17)\nLinux host 6.8.0 #1 SMP x86_64\n", "v3.16"},
		{"bare line", "iperf 3.19\n", "v3.19"},
		{"single word", "iperf\n", "iperf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := iperf.ParseVersion(tt.out); got != tt.want {
				t.Errorf("ParseVersion(%q) = %q, want %q", tt.out, got, tt.want)
			}
		})
	}
}
