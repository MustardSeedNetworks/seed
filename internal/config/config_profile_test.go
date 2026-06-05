package config_test

import (
	"reflect"
	"testing"

	"github.com/krisarmstrong/seed/internal/config"
)

// TestProfileJSONRoundTrip is the previously-missing coverage for the profile
// save→apply contract (ADR-0009 follow-up). ToProfileJSON and ApplyProfileJSON
// must be symmetric — every per-profile section survives a round-trip — and
// ApplyProfileJSON must leave global settings (Server/Auth/Security/Logging/
// Database) untouched.
func TestProfileJSONRoundTrip(t *testing.T) {
	src := config.DefaultConfig() // realistic, fully-populated profile sections

	jsonStr, err := src.ToProfileJSON()
	if err != nil {
		t.Fatalf("ToProfileJSON: %v", err)
	}

	// Apply onto a fresh config whose globals are deliberately set, to prove
	// they are preserved.
	dst := &config.Config{}
	dst.Server.Port = 9999
	dst.Auth.DefaultUsername = "operator"

	if applyErr := dst.ApplyProfileJSON(jsonStr); applyErr != nil {
		t.Fatalf("ApplyProfileJSON: %v", applyErr)
	}

	// Every per-profile section must round-trip exactly (catches any field —
	// incl. snake_case nested ones like link.available_modes — being dropped).
	checks := []struct {
		name      string
		got, want any
	}{
		{"Thresholds", dst.Thresholds, src.Thresholds},
		{"HealthChecks", dst.HealthChecks, src.HealthChecks},
		{"Speedtest", dst.Speedtest, src.Speedtest},
		{"Iperf", dst.Iperf, src.Iperf},
		{"FABOptions", dst.FABOptions, src.FABOptions},
		{"DisplayOptions", dst.DisplayOptions, src.DisplayOptions},
		{"DNS", dst.DNS, src.DNS},
		{"SNMP", dst.SNMP, src.SNMP},
		{"NetworkDiscovery", dst.NetworkDiscovery, src.NetworkDiscovery},
		{"Link", dst.Link, src.Link},
		{"CableTest", dst.CableTest, src.CableTest},
	}
	for _, c := range checks {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s did not round-trip:\n got=%+v\nwant=%+v", c.name, c.got, c.want)
		}
	}

	// Global settings must be preserved (not overwritten by the profile).
	if dst.Server.Port != 9999 {
		t.Errorf("ApplyProfileJSON clobbered global Server.Port = %d, want 9999", dst.Server.Port)
	}
	if dst.Auth.DefaultUsername != "operator" {
		t.Errorf("ApplyProfileJSON clobbered global Auth.DefaultUsername = %q, want operator", dst.Auth.DefaultUsername)
	}
}

// TestApplyProfileJSONEmpty verifies the empty-string fast path is a no-op.
func TestApplyProfileJSONEmpty(t *testing.T) {
	c := config.DefaultConfig()
	before := c.DNS
	if err := c.ApplyProfileJSON(""); err != nil {
		t.Fatalf("ApplyProfileJSON(\"\"): %v", err)
	}
	if !reflect.DeepEqual(c.DNS, before) {
		t.Error("ApplyProfileJSON(\"\") mutated config")
	}
}
