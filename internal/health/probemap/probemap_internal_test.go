package probemap

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// TestHL7EndpointRoundTrip verifies a full HL7 endpoint survives the
// config -> probe -> config conversion unchanged, and that params_json
// uses the checker-aligned key (sending_facility) so the HL7 checker can
// read the facility fields back.
func TestHL7EndpointRoundTrip(t *testing.T) {
	t.Parallel()
	in := config.HL7Endpoint{
		Name: "Lab interface", Host: "hl7.example.com", Port: 2575,
		SendingApp: "SEED", SendingFac: "MAIN", ReceivingApp: "EPIC",
		ReceivingFac: "WARD", Enabled: true,
	}
	p, err := hl7EndpointToProbe(in)
	if err != nil {
		t.Fatalf("hl7EndpointToProbe: %v", err)
	}
	if p.Kind != probe.KindHL7 || p.Target != in.Host || p.DisplayName != in.Name || !p.Enabled {
		t.Errorf(
			"probe columns wrong: kind=%s target=%s name=%s enabled=%v",
			p.Kind,
			p.Target,
			p.DisplayName,
			p.Enabled,
		)
	}
	if !strings.Contains(string(p.Params), `"sending_facility":"MAIN"`) {
		t.Errorf("Params should carry checker-aligned key sending_facility; got %s", string(p.Params))
	}
	out, err := probeToHL7Endpoint(p)
	if err != nil {
		t.Fatalf("probeToHL7Endpoint: %v", err)
	}
	if out != in {
		t.Errorf("round trip mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

// TestFHIREndpointPreservesSettingsOnlyFields verifies that fields the
// checker never reads (ClientSecret, TokenURL) still round-trip, since
// Params stores the whole endpoint.
func TestFHIREndpointPreservesSettingsOnlyFields(t *testing.T) {
	t.Parallel()
	in := config.FHIREndpoint{
		Name: "EHR", BaseURL: "https://fhir.example.com/r4", AuthType: "oauth2",
		BearerToken: "tok", ClientID: "cid", ClientSecret: "secret",
		TokenURL: "https://auth.example.com/token", Enabled: true,
	}
	p, err := fhirEndpointToProbe(in)
	if err != nil {
		t.Fatalf("fhirEndpointToProbe: %v", err)
	}
	if p.Target != in.BaseURL {
		t.Errorf("target should be base URL; got %s", p.Target)
	}
	out, err := probeToFHIREndpoint(p)
	if err != nil {
		t.Fatalf("probeToFHIREndpoint: %v", err)
	}
	if out != in {
		t.Errorf("settings-only fields lost:\n in=%+v\nout=%+v", in, out)
	}
}

// TestHealthCheckProbesFromConfigAndBack verifies the bulk flatten +
// per-kind reassembly preserves a multi-kind set.
func TestHealthCheckProbesFromConfigAndBack(t *testing.T) {
	t.Parallel()
	hc := config.HealthChecksConfig{
		PingTargets: []config.PingTarget{{Name: "gw", Host: "192.0.2.1", Enabled: true}},
		ModbusEndpoints: []config.ModbusEndpoint{
			{
				Name:         "plc",
				Host:         "10.0.0.5",
				Port:         502,
				UnitID:       3,
				TestRegister: 40001,
				RegisterType: "holding",
				Enabled:      true,
			},
		},
	}
	probes, err := ProbesFromConfig(&hc)
	if err != nil {
		t.Fatalf("ProbesFromConfig: %v", err)
	}
	if len(probes) != 2 {
		t.Fatalf("expected 2 probes, got %d", len(probes))
	}
	var got config.HealthChecksConfig
	for _, p := range probes {
		if err = appendProbeToConfig(&got, p); err != nil {
			t.Fatalf("appendProbeToConfig: %v", err)
		}
	}
	if len(got.PingTargets) != 1 || got.PingTargets[0] != hc.PingTargets[0] {
		t.Errorf("ping round trip mismatch: %+v", got.PingTargets)
	}
	if len(got.ModbusEndpoints) != 1 || got.ModbusEndpoints[0] != hc.ModbusEndpoints[0] {
		t.Errorf("modbus round trip mismatch: %+v", got.ModbusEndpoints)
	}
}

// TestNonHealthCheckKindIgnored verifies appendProbeToConfig drops a
// probe of an unrelated kind rather than misfiling it.
func TestNonHealthCheckKindIgnored(t *testing.T) {
	t.Parallel()
	var hc config.HealthChecksConfig
	params, _ := json.Marshal(map[string]any{"x": 1})
	dnsProbe := probe.Probe{Kind: "dns", Target: "8.8.8.8", Params: json.RawMessage(params)}
	if err := appendProbeToConfig(&hc, dnsProbe); err != nil {
		t.Fatalf("appendProbeToConfig: %v", err)
	}
	if len(hc.PingTargets)+len(hc.HTTPEndpoints)+len(hc.ModbusEndpoints) != 0 {
		t.Error("a non-health-check kind should not be filed into any endpoint list")
	}
}
