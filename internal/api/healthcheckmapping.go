package api

// healthcheckmapping.go converts between the health-check endpoint
// configuration shapes (config.*Endpoint) and probe.Probe definitions
// persisted in the probes table (ADR-0027 P2). The probes table is the
// store of record for the fourteen health-check kinds; the legacy
// config.HealthChecks endpoint lists are migrated onto it here.
//
// Each probe row keeps the endpoint's identity in dedicated columns —
// DisplayName (Name), Target (the host or URL), and Enabled — and stores
// the full endpoint JSON in params_json. The per-kind Checker
// (internal/probe/checkers) unmarshals the subset of params it needs and
// ignores the rest, while settings reads the whole endpoint back. The
// JSON key namespaces are kept aligned so one params blob serves both.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// defaultHealthCheckIntervalSeconds is the scheduling interval assigned
// to migrated health-check probes. Endpoints carried no interval; the
// engine schedules each probe at this cadence.
const defaultHealthCheckIntervalSeconds = 60

// healthCheckKinds lists the probe kinds owned by the health-check
// settings surface. Used to scope the replace-by-kind save so the
// migration never touches probes of other kinds.
func healthCheckKinds() []string {
	return []string{
		probe.KindPing, probe.KindTCP, probe.KindUDP, probe.KindHTTP,
		probe.KindRTSP, probe.KindDICOM, probe.KindHL7, probe.KindFHIR,
		probe.KindSQL, probe.KindFileShare, probe.KindLDAP, probe.KindLTI,
		probe.KindOPCUA, probe.KindMODBUS,
	}
}

// countHealthCheckProbes returns the total number of probes across the
// health-check kinds for the default client. Used by the first-run seed to
// detect an install that already holds a configured set (the upgrade path)
// so the factory defaults never overwrite it.
func countHealthCheckProbes(ctx context.Context, repo *database.ProbeRepository) (int, error) {
	total := 0
	for _, kind := range healthCheckKinds() {
		n, err := repo.CountProbes(ctx, database.DefaultClientID, kind)
		if err != nil {
			return 0, fmt.Errorf("count %s probes: %w", kind, err)
		}
		total += n
	}
	return total, nil
}

// endpointToProbe builds a probe row from an endpoint: the endpoint is
// marshaled wholesale into params_json (Name/Target/Enabled are also
// authoritative columns, so the duplicate keys in params are harmless
// and the columns win on read). ID and ClientID are left empty for
// CreateProbe to fill (uuid + default client).
func endpointToProbe(kind, name, target string, enabled bool, endpoint any) (*database.Probe, error) {
	pj, err := json.Marshal(endpoint)
	if err != nil {
		return nil, fmt.Errorf("marshal %s endpoint params: %w", kind, err)
	}
	return &database.Probe{
		Kind:            kind,
		DisplayName:     name,
		Target:          target,
		ParamsJSON:      string(pj),
		IntervalSeconds: defaultHealthCheckIntervalSeconds,
		Enabled:         enabled,
	}, nil
}

// endpointFromProbe unmarshals a probe's params_json into the endpoint
// type T. The caller overwrites the identity fields (Name/Target/Enabled)
// from the authoritative columns afterwards.
func endpointFromProbe[T any](p *database.Probe) (T, error) {
	var e T
	if p.ParamsJSON == "" {
		return e, nil
	}
	if err := json.Unmarshal([]byte(p.ParamsJSON), &e); err != nil {
		return e, fmt.Errorf("unmarshal %s probe params: %w", p.Kind, err)
	}
	return e, nil
}

// --- per-kind forward mappings (config endpoint -> probe row) ---

func pingTargetToProbe(e config.PingTarget) (*database.Probe, error) {
	return endpointToProbe(probe.KindPing, e.Name, e.Host, e.Enabled, e)
}

func tcpPortToProbe(e config.TCPPortTest) (*database.Probe, error) {
	return endpointToProbe(probe.KindTCP, e.Name, e.Host, e.Enabled, e)
}

func udpPortToProbe(e config.UDPPortTest) (*database.Probe, error) {
	return endpointToProbe(probe.KindUDP, e.Name, e.Host, e.Enabled, e)
}

func httpEndpointToProbe(e config.HTTPEndpoint) (*database.Probe, error) {
	return endpointToProbe(probe.KindHTTP, e.Name, e.URL, e.Enabled, e)
}

func rtspEndpointToProbe(e config.RTSPEndpoint) (*database.Probe, error) {
	return endpointToProbe(probe.KindRTSP, e.Name, e.URL, e.Enabled, e)
}

func dicomEndpointToProbe(e config.DICOMEndpoint) (*database.Probe, error) {
	return endpointToProbe(probe.KindDICOM, e.Name, e.Host, e.Enabled, e)
}

func hl7EndpointToProbe(e config.HL7Endpoint) (*database.Probe, error) {
	return endpointToProbe(probe.KindHL7, e.Name, e.Host, e.Enabled, e)
}

func fhirEndpointToProbe(e config.FHIREndpoint) (*database.Probe, error) {
	return endpointToProbe(probe.KindFHIR, e.Name, e.BaseURL, e.Enabled, e)
}

func sqlEndpointToProbe(e config.SQLEndpoint) (*database.Probe, error) {
	return endpointToProbe(probe.KindSQL, e.Name, e.Host, e.Enabled, e)
}

func fileShareEndpointToProbe(e config.FileShareEndpoint) (*database.Probe, error) {
	return endpointToProbe(probe.KindFileShare, e.Name, e.Host, e.Enabled, e)
}

func ldapEndpointToProbe(e config.LDAPEndpoint) (*database.Probe, error) {
	return endpointToProbe(probe.KindLDAP, e.Name, e.Host, e.Enabled, e)
}

func ltiEndpointToProbe(e config.LTIEndpoint) (*database.Probe, error) {
	return endpointToProbe(probe.KindLTI, e.Name, e.LaunchURL, e.Enabled, e)
}

func opcuaEndpointToProbe(e config.OPCUAEndpoint) (*database.Probe, error) {
	return endpointToProbe(probe.KindOPCUA, e.Name, e.EndpointURL, e.Enabled, e)
}

func modbusEndpointToProbe(e config.ModbusEndpoint) (*database.Probe, error) {
	return endpointToProbe(probe.KindMODBUS, e.Name, e.Host, e.Enabled, e)
}

// --- per-kind reverse mappings (probe row -> config endpoint) ---

func probeToPingTarget(p *database.Probe) (config.PingTarget, error) {
	e, err := endpointFromProbe[config.PingTarget](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToTCPPort(p *database.Probe) (config.TCPPortTest, error) {
	e, err := endpointFromProbe[config.TCPPortTest](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToUDPPort(p *database.Probe) (config.UDPPortTest, error) {
	e, err := endpointFromProbe[config.UDPPortTest](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToHTTPEndpoint(p *database.Probe) (config.HTTPEndpoint, error) {
	e, err := endpointFromProbe[config.HTTPEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.URL, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToRTSPEndpoint(p *database.Probe) (config.RTSPEndpoint, error) {
	e, err := endpointFromProbe[config.RTSPEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.URL, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToDICOMEndpoint(p *database.Probe) (config.DICOMEndpoint, error) {
	e, err := endpointFromProbe[config.DICOMEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToHL7Endpoint(p *database.Probe) (config.HL7Endpoint, error) {
	e, err := endpointFromProbe[config.HL7Endpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToFHIREndpoint(p *database.Probe) (config.FHIREndpoint, error) {
	e, err := endpointFromProbe[config.FHIREndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.BaseURL, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToSQLEndpoint(p *database.Probe) (config.SQLEndpoint, error) {
	e, err := endpointFromProbe[config.SQLEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToFileShareEndpoint(p *database.Probe) (config.FileShareEndpoint, error) {
	e, err := endpointFromProbe[config.FileShareEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToLDAPEndpoint(p *database.Probe) (config.LDAPEndpoint, error) {
	e, err := endpointFromProbe[config.LDAPEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToLTIEndpoint(p *database.Probe) (config.LTIEndpoint, error) {
	e, err := endpointFromProbe[config.LTIEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.LaunchURL, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToOPCUAEndpoint(p *database.Probe) (config.OPCUAEndpoint, error) {
	e, err := endpointFromProbe[config.OPCUAEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.EndpointURL, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToModbusEndpoint(p *database.Probe) (config.ModbusEndpoint, error) {
	e, err := endpointFromProbe[config.ModbusEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

// loadHealthCheckEndpoints reads the health-check probe definitions from
// the probes table and assembles them into the endpoint lists of a
// HealthChecksConfig. Only the endpoint lists are populated; the
// performance/discovery toggles live in the config file and are not
// sourced here. A malformed params row is skipped with the error
// returned so the caller can log it without dropping the whole set.
func loadHealthCheckEndpoints(ctx context.Context, repo *database.ProbeRepository) (config.HealthChecksConfig, error) {
	var hc config.HealthChecksConfig
	probes, err := repo.ListProbes(ctx, database.DefaultClientID, "")
	if err != nil {
		return hc, fmt.Errorf("list health-check probes: %w", err)
	}
	for _, p := range probes {
		if err = appendProbeToConfig(&hc, p); err != nil {
			return hc, err
		}
	}
	return hc, nil
}

// appendProbeToConfig maps one probe row onto its endpoint list in hc.
// Probes of non-health-check kinds are ignored.
//
//nolint:gocognit,cyclop,funlen // A flat dispatch over the fourteen probe kinds; one case each.
func appendProbeToConfig(hc *config.HealthChecksConfig, p *database.Probe) error {
	switch p.Kind {
	case probe.KindPing:
		e, err := probeToPingTarget(p)
		if err != nil {
			return err
		}
		hc.PingTargets = append(hc.PingTargets, e)
	case probe.KindTCP:
		e, err := probeToTCPPort(p)
		if err != nil {
			return err
		}
		hc.TCPPorts = append(hc.TCPPorts, e)
	case probe.KindUDP:
		e, err := probeToUDPPort(p)
		if err != nil {
			return err
		}
		hc.UDPPorts = append(hc.UDPPorts, e)
	case probe.KindHTTP:
		e, err := probeToHTTPEndpoint(p)
		if err != nil {
			return err
		}
		hc.HTTPEndpoints = append(hc.HTTPEndpoints, e)
	case probe.KindRTSP:
		e, err := probeToRTSPEndpoint(p)
		if err != nil {
			return err
		}
		hc.RTSPEndpoints = append(hc.RTSPEndpoints, e)
	case probe.KindDICOM:
		e, err := probeToDICOMEndpoint(p)
		if err != nil {
			return err
		}
		hc.DICOMEndpoints = append(hc.DICOMEndpoints, e)
	case probe.KindHL7:
		e, err := probeToHL7Endpoint(p)
		if err != nil {
			return err
		}
		hc.HL7Endpoints = append(hc.HL7Endpoints, e)
	case probe.KindFHIR:
		e, err := probeToFHIREndpoint(p)
		if err != nil {
			return err
		}
		hc.FHIREndpoints = append(hc.FHIREndpoints, e)
	case probe.KindSQL:
		e, err := probeToSQLEndpoint(p)
		if err != nil {
			return err
		}
		hc.SQLEndpoints = append(hc.SQLEndpoints, e)
	case probe.KindFileShare:
		e, err := probeToFileShareEndpoint(p)
		if err != nil {
			return err
		}
		hc.FileShareEndpoints = append(hc.FileShareEndpoints, e)
	case probe.KindLDAP:
		e, err := probeToLDAPEndpoint(p)
		if err != nil {
			return err
		}
		hc.LDAPEndpoints = append(hc.LDAPEndpoints, e)
	case probe.KindLTI:
		e, err := probeToLTIEndpoint(p)
		if err != nil {
			return err
		}
		hc.LTIEndpoints = append(hc.LTIEndpoints, e)
	case probe.KindOPCUA:
		e, err := probeToOPCUAEndpoint(p)
		if err != nil {
			return err
		}
		hc.OPCUAEndpoints = append(hc.OPCUAEndpoints, e)
	case probe.KindMODBUS:
		e, err := probeToModbusEndpoint(p)
		if err != nil {
			return err
		}
		hc.ModbusEndpoints = append(hc.ModbusEndpoints, e)
	}
	return nil
}

// healthCheckProbesFromConfig flattens the endpoint lists of a
// HealthChecksConfig into the probe rows to persist. Order follows
// healthCheckKinds; a marshal failure aborts the whole set.
//
//nolint:gocognit,cyclop // A flat fan over the fourteen endpoint lists; one block each.
func healthCheckProbesFromConfig(hc *config.HealthChecksConfig) ([]*database.Probe, error) {
	var out []*database.Probe
	add := func(p *database.Probe, err error) error {
		if err != nil {
			return err
		}
		out = append(out, p)
		return nil
	}
	for _, e := range hc.PingTargets {
		if err := add(pingTargetToProbe(e)); err != nil {
			return nil, err
		}
	}
	for _, e := range hc.TCPPorts {
		if err := add(tcpPortToProbe(e)); err != nil {
			return nil, err
		}
	}
	for _, e := range hc.UDPPorts {
		if err := add(udpPortToProbe(e)); err != nil {
			return nil, err
		}
	}
	for _, e := range hc.HTTPEndpoints {
		if err := add(httpEndpointToProbe(e)); err != nil {
			return nil, err
		}
	}
	for _, e := range hc.RTSPEndpoints {
		if err := add(rtspEndpointToProbe(e)); err != nil {
			return nil, err
		}
	}
	for _, e := range hc.DICOMEndpoints {
		if err := add(dicomEndpointToProbe(e)); err != nil {
			return nil, err
		}
	}
	for _, e := range hc.HL7Endpoints {
		if err := add(hl7EndpointToProbe(e)); err != nil {
			return nil, err
		}
	}
	for _, e := range hc.FHIREndpoints {
		if err := add(fhirEndpointToProbe(e)); err != nil {
			return nil, err
		}
	}
	for _, e := range hc.SQLEndpoints {
		if err := add(sqlEndpointToProbe(e)); err != nil {
			return nil, err
		}
	}
	for _, e := range hc.FileShareEndpoints {
		if err := add(fileShareEndpointToProbe(e)); err != nil {
			return nil, err
		}
	}
	for _, e := range hc.LDAPEndpoints {
		if err := add(ldapEndpointToProbe(e)); err != nil {
			return nil, err
		}
	}
	for _, e := range hc.LTIEndpoints {
		if err := add(ltiEndpointToProbe(e)); err != nil {
			return nil, err
		}
	}
	for _, e := range hc.OPCUAEndpoints {
		if err := add(opcuaEndpointToProbe(e)); err != nil {
			return nil, err
		}
	}
	for _, e := range hc.ModbusEndpoints {
		if err := add(modbusEndpointToProbe(e)); err != nil {
			return nil, err
		}
	}
	return out, nil
}
