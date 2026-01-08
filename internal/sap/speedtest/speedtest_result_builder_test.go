// Package speedtest_test provides tests for the result building logic in the speedtest package.
// These tests verify the Result struct construction and validation without network access.
package speedtest_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/speedtest"
)

// TestBuildTestResultFromParamsLocationFormatting tests location string formatting.
func TestBuildTestResultFromParamsLocationFormatting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		sponsor          string
		country          string
		expectedLocation string
	}{
		{
			name:             "standard US format",
			sponsor:          "Comcast",
			country:          "United States",
			expectedLocation: "Comcast, United States",
		},
		{
			name:             "international format",
			sponsor:          "Deutsche Telekom",
			country:          "Germany",
			expectedLocation: "Deutsche Telekom, Germany",
		},
		{
			name:             "empty sponsor",
			sponsor:          "",
			country:          "USA",
			expectedLocation: ", USA",
		},
		{
			name:             "empty country",
			sponsor:          "Provider",
			country:          "",
			expectedLocation: "Provider, ",
		},
		{
			name:             "both empty",
			sponsor:          "",
			country:          "",
			expectedLocation: ", ",
		},
		{
			name:             "with ampersand",
			sponsor:          "AT&T",
			country:          "USA",
			expectedLocation: "AT&T, USA",
		},
		{
			name:             "with slash",
			sponsor:          "Verizon/Fios",
			country:          "USA",
			expectedLocation: "Verizon/Fios, USA",
		},
		{
			name:             "unicode sponsor",
			sponsor:          "日本電信電話",
			country:          "Japan",
			expectedLocation: "日本電信電話, Japan",
		},
		{
			name:             "unicode country",
			sponsor:          "NTT",
			country:          "日本",
			expectedLocation: "NTT, 日本",
		},
		{
			name:             "with parentheses",
			sponsor:          "Local ISP (Regional)",
			country:          "USA (West)",
			expectedLocation: "Local ISP (Regional), USA (West)",
		},
		{
			name:             "very long names",
			sponsor:          "Very Long Internet Service Provider Name That Goes On And On",
			country:          "The United States of America and its Territories",
			expectedLocation: "Very Long Internet Service Provider Name That Goes On And On, The United States of America and its Territories",
		},
		{
			name:             "with quotes",
			sponsor:          "\"Best\" Provider",
			country:          "'Best' Country",
			expectedLocation: "\"Best\" Provider, 'Best' Country",
		},
		{
			name:             "whitespace only sponsor",
			sponsor:          "   ",
			country:          "USA",
			expectedLocation: "   , USA",
		},
		{
			name:             "newline in sponsor",
			sponsor:          "Provider\nName",
			country:          "USA",
			expectedLocation: "Provider\nName, USA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			result := tester.BuildTestResultFromParams(
				100.0, 50.0,
				10*time.Millisecond,
				"Server",
				tt.sponsor,
				tt.country,
				"host.example.com",
				50.0,
				time.Now(),
			)

			if result.Location != tt.expectedLocation {
				t.Errorf("Location: got %q, want %q", result.Location, tt.expectedLocation)
			}
		})
	}
}

// TestResultBuilderLatencyConversion tests latency conversion from duration to milliseconds.
func TestResultBuilderLatencyConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		latency         time.Duration
		expectedLatency float64
	}{
		{"zero", 0, 0},
		{"1 nanosecond", time.Nanosecond, 0},
		{"1 microsecond", time.Microsecond, 0},
		{"999 microseconds", 999 * time.Microsecond, 0},
		{"1 millisecond", time.Millisecond, 1},
		{"1.5 milliseconds", 1500 * time.Microsecond, 1},
		{"2 milliseconds", 2 * time.Millisecond, 2},
		{"10 milliseconds", 10 * time.Millisecond, 10},
		{"100 milliseconds", 100 * time.Millisecond, 100},
		{"500 milliseconds", 500 * time.Millisecond, 500},
		{"1 second", time.Second, 1000},
		{"2 seconds", 2 * time.Second, 2000},
		{"10 seconds (timeout)", 10 * time.Second, 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			result := tester.BuildTestResultFromParams(
				100.0, 50.0,
				tt.latency,
				"Server",
				"Sponsor",
				"Country",
				"host.example.com",
				50.0,
				time.Now(),
			)

			if result.Latency != tt.expectedLatency {
				t.Errorf("Latency: got %v, want %v", result.Latency, tt.expectedLatency)
			}
		})
	}
}

// TestResultBuilderTestDurationCalculation tests test duration calculation accuracy.
func TestResultBuilderTestDurationCalculation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		durationSec int
		tolerance   float64
	}{
		{"instant", 0, 0.5},
		{"1 second", 1, 0.5},
		{"5 seconds", 5, 0.5},
		{"10 seconds", 10, 0.5},
		{"15 seconds (typical)", 15, 0.5},
		{"20 seconds", 20, 0.5},
		{"30 seconds", 30, 0.5},
		{"60 seconds (slow)", 60, 0.5},
		{"120 seconds (very slow)", 120, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			startTime := time.Now().Add(-time.Duration(tt.durationSec) * time.Second)
			result := tester.BuildTestResultFromParams(
				100.0, 50.0,
				10*time.Millisecond,
				"Server",
				"Sponsor",
				"Country",
				"host.example.com",
				50.0,
				startTime,
			)

			expectedMin := float64(tt.durationSec) - tt.tolerance
			expectedMax := float64(tt.durationSec) + tt.tolerance
			if result.TestDuration < expectedMin || result.TestDuration > expectedMax {
				t.Errorf("TestDuration: got %v, want between %v and %v",
					result.TestDuration, expectedMin, expectedMax)
			}
		})
	}
}

// TestResultBuilderSpeedValues tests various speed combinations.
func TestResultBuilderSpeedValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		download float64
		upload   float64
	}{
		{"zero speeds", 0, 0},
		{"minimal speeds", 0.001, 0.001},
		{"dial-up era", 0.056, 0.033},
		{"basic DSL", 10, 1},
		{"fast DSL", 25, 5},
		{"basic cable", 100, 10},
		{"fast cable", 300, 30},
		{"fiber 100", 100, 100},
		{"fiber 500", 500, 500},
		{"fiber 1000", 1000, 1000},
		{"multi-gig", 2500, 2500},
		{"10 gig", 10000, 10000},
		{"asymmetric download", 1000, 10},
		{"asymmetric upload", 10, 1000},
		{"fractional speeds", 123.456789, 78.901234},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			result := tester.BuildTestResultFromParams(
				tt.download, tt.upload,
				10*time.Millisecond,
				"Server",
				"Sponsor",
				"Country",
				"host.example.com",
				50.0,
				time.Now(),
			)

			if result.Download != tt.download {
				t.Errorf("Download: got %v, want %v", result.Download, tt.download)
			}
			if result.Upload != tt.upload {
				t.Errorf("Upload: got %v, want %v", result.Upload, tt.upload)
			}
		})
	}
}

// TestResultBuilderDistanceValues tests distance field values.
func TestResultBuilderDistanceValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		distance float64
	}{
		{"zero distance (same location)", 0},
		{"very close (< 1km)", 0.5},
		{"nearby (1-10km)", 5.5},
		{"local (10-50km)", 25.0},
		{"regional (50-200km)", 150.0},
		{"national (200-1000km)", 500.0},
		{"continental (1000-5000km)", 3000.0},
		{"intercontinental (5000-10000km)", 8000.0},
		{"maximum possible (Earth circumference)", 20000.0},
		{"fractional distance", 123.456},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			result := tester.BuildTestResultFromParams(
				100.0, 50.0,
				10*time.Millisecond,
				"Server",
				"Sponsor",
				"Country",
				"host.example.com",
				tt.distance,
				time.Now(),
			)

			if result.Distance != tt.distance {
				t.Errorf("Distance: got %v, want %v", result.Distance, tt.distance)
			}
		})
	}
}

// TestResultBuilderServerInfo tests server information fields.
func TestResultBuilderServerInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		serverName string
		host       string
	}{
		{"empty values", "", ""},
		{"typical server", "Comcast Speed Test", "speedtest.comcast.net:8080"},
		{"with IP address", "Local Server", "192.168.1.100:8080"},
		{"with IPv6", "IPv6 Server", "[2001:db8::1]:8080"},
		{"localhost", "Development Server", "localhost:5201"},
		{"no port", "Simple Server", "speedtest.example.com"},
		{"custom port", "Custom Port Server", "speedtest.example.com:9999"},
		{"unicode server name", "テスト速度サーバー", "test.example.jp:8080"},
		{"special chars in name", "AT&T 'Fiber' Test <1Gbps>", "att.speed.com:8080"},
		{
			"very long name",
			"This Is A Very Long Server Name That Might Be Used In Some Edge Cases For Testing",
			"long.host.example.com:8080",
		},
		{"subdomain host", "Subdomain Test", "speed.test.cdn.provider.example.com:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			result := tester.BuildTestResultFromParams(
				100.0, 50.0,
				10*time.Millisecond,
				tt.serverName,
				"Sponsor",
				"Country",
				tt.host,
				50.0,
				time.Now(),
			)

			if result.Server != tt.serverName {
				t.Errorf("Server: got %q, want %q", result.Server, tt.serverName)
			}
			if result.Host != tt.host {
				t.Errorf("Host: got %q, want %q", result.Host, tt.host)
			}
		})
	}
}

// TestResultBuilderTimestamp tests that timestamp is set correctly.
func TestResultBuilderTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		startOffset time.Duration
	}{
		{"immediate", 0},
		{"short test", -5 * time.Second},
		{"typical test", -15 * time.Second},
		{"long test", -60 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			beforeCall := time.Now()
			startTime := time.Now().Add(tt.startOffset)

			result := tester.BuildTestResultFromParams(
				100.0, 50.0,
				10*time.Millisecond,
				"Server",
				"Sponsor",
				"Country",
				"host.example.com",
				50.0,
				startTime,
			)

			afterCall := time.Now()

			if result.Timestamp.Before(beforeCall) {
				t.Error("Timestamp should not be before function call")
			}
			if result.Timestamp.After(afterCall) {
				t.Error("Timestamp should not be after function call")
			}
		})
	}
}

// TestBuildTestResultFromParamsAllFieldsCombination tests complete result with all fields.
func TestBuildTestResultFromParamsAllFieldsCombination(t *testing.T) {
	t.Parallel()

	tester := speedtest.NewTester()
	startTime := time.Now().Add(-20 * time.Second)
	beforeCall := time.Now()

	result := tester.BuildTestResultFromParams(
		945.67,
		923.45,
		5*time.Millisecond,
		"Fiber ISP Speed Test",
		"Major Fiber Provider",
		"United States",
		"speedtest.fiber.example.com:8080",
		12.34,
		startTime,
	)

	afterCall := time.Now()

	// Verify all fields
	if result.Download != 945.67 {
		t.Errorf("Download: got %v, want 945.67", result.Download)
	}
	if result.Upload != 923.45 {
		t.Errorf("Upload: got %v, want 923.45", result.Upload)
	}
	if result.Latency != 5 {
		t.Errorf("Latency: got %v, want 5", result.Latency)
	}
	if result.Server != "Fiber ISP Speed Test" {
		t.Errorf("Server: got %q, want %q", result.Server, "Fiber ISP Speed Test")
	}
	if result.Location != "Major Fiber Provider, United States" {
		t.Errorf("Location: got %q, want %q", result.Location, "Major Fiber Provider, United States")
	}
	if result.Host != "speedtest.fiber.example.com:8080" {
		t.Errorf("Host: got %q, want %q", result.Host, "speedtest.fiber.example.com:8080")
	}
	if result.Distance != 12.34 {
		t.Errorf("Distance: got %v, want 12.34", result.Distance)
	}
	if result.TestDuration < 19.5 || result.TestDuration > 20.5 {
		t.Errorf("TestDuration: got %v, want ~20", result.TestDuration)
	}
	if result.Timestamp.Before(beforeCall) || result.Timestamp.After(afterCall) {
		t.Errorf("Timestamp out of range: %v", result.Timestamp)
	}
}

// TestResultTypicalConnectionTypes tests results for common connection types.
func TestResultTypicalConnectionTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		download     float64
		upload       float64
		latencyMs    int64
		description  string
		expectedFast bool
	}{
		{
			name:         "dial-up",
			download:     0.056,
			upload:       0.033,
			latencyMs:    150,
			description:  "Dial-up modem connection",
			expectedFast: false,
		},
		{
			name:         "basic DSL",
			download:     10,
			upload:       1,
			latencyMs:    30,
			description:  "Entry-level DSL",
			expectedFast: false,
		},
		{
			name:         "fast DSL",
			download:     50,
			upload:       10,
			latencyMs:    20,
			description:  "Premium DSL",
			expectedFast: false,
		},
		{
			name:         "cable",
			download:     200,
			upload:       20,
			latencyMs:    15,
			description:  "Typical cable internet",
			expectedFast: true,
		},
		{
			name:         "fiber 100",
			download:     100,
			upload:       100,
			latencyMs:    5,
			description:  "100 Mbps fiber",
			expectedFast: true,
		},
		{
			name:         "fiber 1000",
			download:     940,
			upload:       880,
			latencyMs:    3,
			description:  "Gigabit fiber",
			expectedFast: true,
		},
		{
			name:         "mobile 4G",
			download:     50,
			upload:       20,
			latencyMs:    35,
			description:  "4G LTE mobile",
			expectedFast: false,
		},
		{
			name:         "mobile 5G",
			download:     500,
			upload:       100,
			latencyMs:    15,
			description:  "5G mobile",
			expectedFast: true,
		},
		{
			name:         "satellite",
			download:     50,
			upload:       10,
			latencyMs:    600,
			description:  "Satellite internet",
			expectedFast: false,
		},
		{
			name:         "starlink",
			download:     150,
			upload:       20,
			latencyMs:    40,
			description:  "Starlink satellite",
			expectedFast: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tester := speedtest.NewTester()
			result := tester.BuildTestResultFromParams(
				tt.download,
				tt.upload,
				time.Duration(tt.latencyMs)*time.Millisecond,
				"Server",
				"Provider",
				"USA",
				"speedtest.example.com:8080",
				50.0,
				time.Now().Add(-15*time.Second),
			)

			if result.Download != tt.download {
				t.Errorf("Download: got %v, want %v", result.Download, tt.download)
			}
			if result.Upload != tt.upload {
				t.Errorf("Upload: got %v, want %v", result.Upload, tt.upload)
			}
			if result.Latency != float64(tt.latencyMs) {
				t.Errorf("Latency: got %v, want %v", result.Latency, float64(tt.latencyMs))
			}

			// Verify the fast threshold (>= 100 Mbps download)
			isFast := result.Download >= 100
			if isFast != tt.expectedFast {
				t.Errorf("Fast check for %s: got %v, want %v", tt.description, isFast, tt.expectedFast)
			}
		})
	}
}

// TestBuildTestResultFromParamsZeroStartTime tests behavior with zero start time.
func TestBuildTestResultFromParamsZeroStartTime(t *testing.T) {
	t.Parallel()

	tester := speedtest.NewTester()
	zeroTime := time.Time{}

	result := tester.BuildTestResultFromParams(
		100.0, 50.0,
		10*time.Millisecond,
		"Server",
		"Sponsor",
		"Country",
		"host.example.com",
		50.0,
		zeroTime,
	)

	// TestDuration will be very large (time since zero)
	// Just verify the result is not nil and fields are set
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Download != 100.0 {
		t.Errorf("Download: got %v, want 100.0", result.Download)
	}
	if result.TestDuration <= 0 {
		t.Errorf("TestDuration should be positive with zero start time, got %v", result.TestDuration)
	}
}

// TestBuildTestResultFromParamsFutureStartTime tests behavior with future start time.
func TestBuildTestResultFromParamsFutureStartTime(t *testing.T) {
	t.Parallel()

	tester := speedtest.NewTester()
	futureTime := time.Now().Add(10 * time.Second)

	result := tester.BuildTestResultFromParams(
		100.0, 50.0,
		10*time.Millisecond,
		"Server",
		"Sponsor",
		"Country",
		"host.example.com",
		50.0,
		futureTime,
	)

	// TestDuration will be negative
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.TestDuration >= 0 {
		t.Errorf("TestDuration should be negative with future start time, got %v", result.TestDuration)
	}
}
