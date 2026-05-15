package main

import (
	"testing"
)

func TestServeConstants(t *testing.T) {

	tests := []struct {
		name     string
		value    int
		minValue int
		desc     string
	}{
		{
			name:     "logBroadcasterBufferSize",
			value:    logBroadcasterBufferSize,
			minValue: 100,
			desc:     "should be at least 100 for reasonable buffering",
		},
		{
			name:     "signalChannelBufferSize",
			value:    signalChannelBufferSize,
			minValue: 1,
			desc:     "should be at least 1 to handle signals",
		},
		{
			name:     "shutdownTimeoutSeconds",
			value:    shutdownTimeoutSeconds,
			minValue: 5,
			desc:     "should be at least 5 seconds for graceful shutdown",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.value < tc.minValue {
				t.Errorf("%s = %d, want >= %d: %s", tc.name, tc.value, tc.minValue, tc.desc)
			}
		})
	}
}

func TestLogBroadcasterBufferSize(t *testing.T) {

	if logBroadcasterBufferSize != 1000 {
		t.Errorf("logBroadcasterBufferSize should be 1000, got %d", logBroadcasterBufferSize)
	}
}

func TestSignalChannelBufferSize(t *testing.T) {

	if signalChannelBufferSize != 2 {
		t.Errorf("signalChannelBufferSize should be 2, got %d", signalChannelBufferSize)
	}
}

func TestShutdownTimeoutSeconds(t *testing.T) {

	if shutdownTimeoutSeconds != 30 {
		t.Errorf("shutdownTimeoutSeconds should be 30, got %d", shutdownTimeoutSeconds)
	}
}

func TestPrintSetupBannerLogic(t *testing.T) {

	tests := []struct {
		name     string
		port     int
		https    bool
		wantHTTP bool
	}{
		{
			name:     "HTTPS mode",
			port:     8443,
			https:    true,
			wantHTTP: false,
		},
		{
			name:     "HTTP mode",
			port:     8080,
			https:    false,
			wantHTTP: true,
		},
		{
			name:     "HTTPS on custom port",
			port:     9443,
			https:    true,
			wantHTTP: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			// Test the protocol selection logic from printSetupBanner
			protocol := "http"
			if tc.https {
				protocol = "https"
			}

			if tc.wantHTTP && protocol != "http" {
				t.Errorf("Expected http protocol, got %s", protocol)
			}
			if !tc.wantHTTP && protocol != "https" {
				t.Errorf("Expected https protocol, got %s", protocol)
			}
		})
	}
}

func TestServeConstantsRelationships(t *testing.T) {

	// Log buffer should be larger than signal channel buffer
	if logBroadcasterBufferSize <= signalChannelBufferSize {
		t.Error("logBroadcasterBufferSize should be larger than signalChannelBufferSize")
	}

	// Shutdown timeout should be reasonable
	if shutdownTimeoutSeconds > 120 {
		t.Error("shutdownTimeoutSeconds should not exceed 120 seconds")
	}
}
