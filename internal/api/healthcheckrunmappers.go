package api

// healthcheckrunmappers.go contains the per-kind result mappers for the
// health-check /run endpoint (ADR-0027 P3). Each mapper converts a
// (database.Probe, probe.Result) pair into the wire DTO the HealthCheckCard
// renders. Static fields (host, port, driver, …) come from the probe's
// ParamsJSON; dynamic fields (server version, ack code, …) come from
// probe.Result.Metadata.

import (
	"encoding/json"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/probe"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
)

// mapSQLResult maps a KindSQL probe result. Driver/host/port come from
// ParamsJSON; server_version comes from Result.Metadata.
func mapSQLResult(p *database.Probe, r probe.Result) SQLTestResult {
	var ep config.SQLEndpoint
	if p.ParamsJSON != "" {
		_ = json.Unmarshal([]byte(p.ParamsJSON), &ep)
	}
	var md checkers.SQLRunMetadata
	if len(r.Metadata) > 0 {
		_ = json.Unmarshal(r.Metadata, &md)
	}
	return SQLTestResult{
		Name:          p.DisplayName,
		Success:       r.Success,
		Driver:        ep.Driver,
		Host:          ep.Host,
		Port:          ep.Port,
		ServerVersion: md.ServerVersion,
		Error:         r.Error,
		TotalTimeMs:   r.LatencyMs,
		ConnectTimeMs: r.LatencyMs,
	}
}

// mapFileShareResult maps a KindFileShare probe result. Protocol/host/share
// come from ParamsJSON; speed fields have no metadata source and stay zero.
func mapFileShareResult(p *database.Probe, r probe.Result) FileShareTestResult {
	var ep config.FileShareEndpoint
	if p.ParamsJSON != "" {
		_ = json.Unmarshal([]byte(p.ParamsJSON), &ep)
	}
	return FileShareTestResult{
		Name:          p.DisplayName,
		Success:       r.Success,
		Protocol:      ep.Protocol,
		Host:          ep.Host,
		Share:         ep.Share,
		Error:         r.Error,
		ConnectTimeMs: r.LatencyMs,
	}
}

// mapLDAPResult maps a KindLDAP probe result. UseTLS/host/port come from
// ParamsJSON; ServerInfo has no reliable metadata source and stays empty.
func mapLDAPResult(p *database.Probe, r probe.Result) LDAPTestResult {
	var ep config.LDAPEndpoint
	if p.ParamsJSON != "" {
		_ = json.Unmarshal([]byte(p.ParamsJSON), &ep)
	}
	return LDAPTestResult{
		Name:          p.DisplayName,
		Success:       r.Success,
		UseTLS:        ep.UseTLS,
		Host:          ep.Host,
		Port:          ep.Port,
		Error:         r.Error,
		TotalTimeMs:   r.LatencyMs,
		ConnectTimeMs: r.LatencyMs,
	}
}

// mapRTSPResult maps a KindRTSP probe result. URL comes from ParamsJSON;
// codec and resolution have no metadata source and stay empty.
func mapRTSPResult(p *database.Probe, r probe.Result) RTSPTestResult {
	var ep config.RTSPEndpoint
	if p.ParamsJSON != "" {
		_ = json.Unmarshal([]byte(p.ParamsJSON), &ep)
	}
	return RTSPTestResult{
		Name:          p.DisplayName,
		Success:       r.Success,
		URL:           ep.URL,
		Error:         r.Error,
		ConnectTimeMs: r.LatencyMs,
	}
}

// mapDICOMResult maps a KindDICOM probe result. Host/port come from
// ParamsJSON; AETitle is the CalledAE static field. Dynamic fields
// (ServerAETitle, EchoTimeMs) have no metadata source and stay zero/empty.
func mapDICOMResult(p *database.Probe, r probe.Result) DICOMTestResult {
	var ep config.DICOMEndpoint
	if p.ParamsJSON != "" {
		_ = json.Unmarshal([]byte(p.ParamsJSON), &ep)
	}
	return DICOMTestResult{
		Name:        p.DisplayName,
		Success:     r.Success,
		Host:        ep.Host,
		Port:        ep.Port,
		AETitle:     ep.CalledAE,
		Error:       r.Error,
		TotalTimeMs: r.LatencyMs,
	}
}

// mapHL7Result maps a KindHL7 probe result. Host/port come from ParamsJSON;
// ack_code comes from Result.Metadata. ServerVersion has no source and stays
// empty.
func mapHL7Result(p *database.Probe, r probe.Result) HL7TestResult {
	var ep config.HL7Endpoint
	if p.ParamsJSON != "" {
		_ = json.Unmarshal([]byte(p.ParamsJSON), &ep)
	}
	var md checkers.HL7RunMetadata
	if len(r.Metadata) > 0 {
		_ = json.Unmarshal(r.Metadata, &md)
	}
	return HL7TestResult{
		Name:        p.DisplayName,
		Success:     r.Success,
		Host:        ep.Host,
		Port:        ep.Port,
		AckCode:     md.AckCode,
		Error:       r.Error,
		TotalTimeMs: r.LatencyMs,
	}
}

// mapFHIRResult maps a KindFHIR probe result. BaseURL comes from ParamsJSON;
// fhir_version/server_name/resource_count come from Result.Metadata.
func mapFHIRResult(p *database.Probe, r probe.Result) FHIRTestResult {
	var ep config.FHIREndpoint
	if p.ParamsJSON != "" {
		_ = json.Unmarshal([]byte(p.ParamsJSON), &ep)
	}
	var md checkers.FHIRRunMetadata
	if len(r.Metadata) > 0 {
		_ = json.Unmarshal(r.Metadata, &md)
	}
	return FHIRTestResult{
		Name:          p.DisplayName,
		Success:       r.Success,
		BaseURL:       ep.BaseURL,
		FHIRVersion:   md.FHIRVersion,
		ServerName:    md.ServerName,
		Error:         r.Error,
		TotalTimeMs:   r.LatencyMs,
		ResourceCount: md.ResourceCount,
	}
}

// mapLTIResult maps a KindLTI probe result. LaunchURL comes from ParamsJSON;
// lti_version comes from Result.Metadata.
func mapLTIResult(p *database.Probe, r probe.Result) LTITestResult {
	var ep config.LTIEndpoint
	if p.ParamsJSON != "" {
		_ = json.Unmarshal([]byte(p.ParamsJSON), &ep)
	}
	var md checkers.LTIRunMetadata
	if len(r.Metadata) > 0 {
		_ = json.Unmarshal(r.Metadata, &md)
	}
	return LTITestResult{
		Name:        p.DisplayName,
		Success:     r.Success,
		LaunchURL:   ep.LaunchURL,
		LTIVersion:  md.LTIVersion,
		Error:       r.Error,
		TotalTimeMs: r.LatencyMs,
	}
}

// mapOPCUAResult maps a KindOPCUA probe result. EndpointURL comes from
// ParamsJSON; security_mode comes from Result.Metadata. ProductName and
// ServerState have no metadata source and stay empty.
func mapOPCUAResult(p *database.Probe, r probe.Result) OPCUATestResult {
	var ep config.OPCUAEndpoint
	if p.ParamsJSON != "" {
		_ = json.Unmarshal([]byte(p.ParamsJSON), &ep)
	}
	var md checkers.OPCUARunMetadata
	if len(r.Metadata) > 0 {
		_ = json.Unmarshal(r.Metadata, &md)
	}
	return OPCUATestResult{
		Name:         p.DisplayName,
		Success:      r.Success,
		EndpointURL:  ep.EndpointURL,
		SecurityMode: md.SecurityMode,
		Error:        r.Error,
		TotalTimeMs:  r.LatencyMs,
	}
}

// mapModbusResult maps a KindMODBUS probe result. Host/port/unitId come from
// ParamsJSON; register_value comes from Result.Metadata.
func mapModbusResult(p *database.Probe, r probe.Result) ModbusTestResult {
	var ep config.ModbusEndpoint
	if p.ParamsJSON != "" {
		_ = json.Unmarshal([]byte(p.ParamsJSON), &ep)
	}
	var md checkers.ModbusRunMetadata
	if len(r.Metadata) > 0 {
		_ = json.Unmarshal(r.Metadata, &md)
	}
	return ModbusTestResult{
		Name:          p.DisplayName,
		Success:       r.Success,
		Host:          ep.Host,
		Port:          ep.Port,
		UnitID:        ep.UnitID,
		Error:         r.Error,
		TotalTimeMs:   r.LatencyMs,
		RegisterValue: md.RegisterValue,
	}
}
