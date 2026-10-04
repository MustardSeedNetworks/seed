package discovery_test

import (
	"math"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// errorSample is one SNMP reading of a switch port's error counters.
type errorSample struct {
	at        time.Duration // since the first reading
	upTime    uint32        // sysUpTime, hundredths of a second
	inErrors  uint64
	outErrors uint64
}

func portDevice(s errorSample) *discovery.DiscoveredDevice {
	epoch := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	return &discovery.DiscoveredDevice{
		MAC: "00:11:22:33:44:55",
		SNMPData: &discovery.SNMPFullData{
			CollectedAt: epoch.Add(s.at),
			System:      &snmp.SystemInfo{SysUpTime: s.upTime},
			Interfaces: []discovery.SNMPInterface{{
				Index:     3,
				Name:      "Gi0/3",
				InErrors:  s.inErrors,
				OutErrors: s.outErrors,
			}},
		},
	}
}

// scanSamples runs one detection scan per reading against a single detector
// at the default threshold of 10 errors per minute, and returns the
// interface-error findings of the last scan.
func scanSamples(t *testing.T, samples ...errorSample) []discovery.InterfaceErrorStats {
	t.Helper()
	det := discovery.NewProblemDetector()
	var res *discovery.ProblemDetectionResult
	for _, s := range samples {
		var err error
		res, err = det.Scan(t.Context(), []*discovery.DiscoveredDevice{portDevice(s)})
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
	}
	return res.InterfaceErrors
}

// TestProblemDetectorRatesInterfaceErrors is seed#2753: the per-minute
// thresholds are compared against the rate between readings, never against
// the agent's lifetime counter.
func TestProblemDetectorRatesInterfaceErrors(t *testing.T) {
	t.Parallel()

	const minute = time.Minute
	tests := []struct {
		name    string
		samples []errorSample
		flagged bool
	}{
		{
			name: "long uptime, high lifetime errors, none recent",
			samples: []errorSample{
				{at: 0, upTime: 900_000_000, inErrors: 1_000_000, outErrors: 500_000},
				{at: minute, upTime: 900_006_000, inErrors: 1_000_000, outErrors: 500_000},
			},
		},
		{
			name:    "a single reading cannot be rated",
			samples: []errorSample{{at: 0, upTime: 900_000_000, inErrors: 1_000_000}},
		},
		{
			name: "freshly booted port rising past the threshold",
			samples: []errorSample{
				{at: 0, upTime: 3000, inErrors: 0},
				{at: 30 * time.Second, upTime: 6000, inErrors: 8},
			},
			flagged: true,
		},
		{
			name: "output errors rising past the threshold",
			samples: []errorSample{
				{at: 0, upTime: 3000, outErrors: 2},
				{at: minute, upTime: 9000, outErrors: 40},
			},
			flagged: true,
		},
		{
			name: "rising but under the threshold",
			samples: []errorSample{
				{at: 0, upTime: 3000, inErrors: 100},
				{at: minute, upTime: 9000, inErrors: 105},
			},
		},
		{
			name: "device rebooted between readings",
			samples: []errorSample{
				{at: 0, upTime: 900_000_000, inErrors: 1_000_000},
				{at: minute, upTime: 500, inErrors: 20},
			},
		},
		{
			name: "counter wrapped through 2^32",
			samples: []errorSample{
				{at: 0, upTime: 3000, inErrors: math.MaxUint32 - 4},
				{at: minute, upTime: 9000, inErrors: 30},
			},
			flagged: true,
		},
		{
			name: "rescan of the same reading keeps the rating",
			samples: []errorSample{
				{at: 0, upTime: 3000, inErrors: 0},
				{at: minute, upTime: 9000, inErrors: 60},
				{at: minute, upTime: 9000, inErrors: 60},
			},
			flagged: true,
		},
		{
			name: "rate recovers once errors stop",
			samples: []errorSample{
				{at: 0, upTime: 3000, inErrors: 0},
				{at: minute, upTime: 9000, inErrors: 60},
				{at: 2 * minute, upTime: 15000, inErrors: 60},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := scanSamples(t, tt.samples...)
			if flagged := len(got) > 0; flagged != tt.flagged {
				t.Fatalf("flagged = %v (%+v), want %v", flagged, got, tt.flagged)
			}
		})
	}
}

func TestProblemDetectorReportsErrorRatePerMinute(t *testing.T) {
	t.Parallel()

	got := scanSamples(t,
		errorSample{at: 0, upTime: 3000, inErrors: 1000, outErrors: 7},
		errorSample{at: 30 * time.Second, upTime: 6000, inErrors: 1045, outErrors: 7},
	)
	if len(got) != 1 {
		t.Fatalf("got %d interface-error findings, want 1: %+v", len(got), got)
	}
	s := got[0]
	if s.InterfaceName != "Gi0/3" || s.InputErrorsPerMin != 90 || s.OutputErrorsPerMin != 0 {
		t.Fatalf("finding = %s in %v/min out %v/min, want Gi0/3 in 90/min out 0/min",
			s.InterfaceName, s.InputErrorsPerMin, s.OutputErrorsPerMin)
	}
	if s.InputErrors != 1045 || s.OutputErrors != 7 {
		t.Fatalf("lifetime counters = in %d out %d, want in 1045 out 7", s.InputErrors, s.OutputErrors)
	}
}

func TestProblemDetectorNeedsUptimeToRate(t *testing.T) {
	t.Parallel()

	det := discovery.NewProblemDetector()
	for _, s := range []errorSample{{at: 0, inErrors: 0}, {at: time.Minute, inErrors: 600}} {
		dev := portDevice(s)
		dev.SNMPData.System = nil
		res, err := det.Scan(t.Context(), []*discovery.DiscoveredDevice{dev})
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if len(res.InterfaceErrors) != 0 {
			t.Fatalf("rated without sysUpTime: %+v", res.InterfaceErrors)
		}
	}
}
