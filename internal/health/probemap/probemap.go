package probemap

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
	"encoding/json"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// defaultHealthCheckIntervalSeconds is the scheduling interval assigned
// to migrated health-check probes. Endpoints carried no interval; the
// engine schedules each probe at this cadence.
const defaultHealthCheckIntervalSeconds = 60

// Kinds lists the probe kinds owned by the health-check
// settings surface. Used to scope the replace-by-kind save so the
// migration never touches probes of other kinds.
func Kinds() []string {
	return []string{
		probe.KindPing, probe.KindTCP, probe.KindUDP, probe.KindHTTP,
		probe.KindRTSP, probe.KindDICOM, probe.KindHL7, probe.KindFHIR,
		probe.KindSQL, probe.KindFileShare, probe.KindLDAP, probe.KindLTI,
		probe.KindOPCUA, probe.KindMODBUS,
	}
}

// endpointToProbe builds a probe.Probe from an endpoint: the endpoint is
// marshaled wholesale into Params (Name/Target/Enabled are also
// authoritative fields, so the duplicate keys in params are harmless
// and the columns win on read). ID and ClientID are left empty for
// the repository to fill (uuid + default client).
func endpointToProbe(kind, name, target string, enabled bool, endpoint any) (probe.Probe, error) {
	pj, err := json.Marshal(endpoint)
	if err != nil {
		return probe.Probe{}, fmt.Errorf("marshal %s endpoint params: %w", kind, err)
	}
	return probe.Probe{
		Kind:            kind,
		DisplayName:     name,
		Target:          target,
		Params:          json.RawMessage(pj),
		IntervalSeconds: defaultHealthCheckIntervalSeconds,
		Enabled:         enabled,
	}, nil
}

// endpointFromProbe unmarshals a probe's Params into the endpoint
// type T. The caller overwrites the identity fields (Name/Target/Enabled)
// from the authoritative columns afterwards.
func endpointFromProbe[T any](p probe.Probe) (T, error) {
	var e T
	if len(p.Params) == 0 {
		return e, nil
	}
	if err := json.Unmarshal(p.Params, &e); err != nil {
		return e, fmt.Errorf("unmarshal %s probe params: %w", p.Kind, err)
	}
	return e, nil
}

// --- per-kind forward mappings (config endpoint -> probe.Probe) ---

func pingTargetToProbe(e config.PingTarget) (probe.Probe, error) {
	return endpointToProbe(probe.KindPing, e.Name, e.Host, e.Enabled, e)
}

func tcpPortToProbe(e config.TCPPortTest) (probe.Probe, error) {
	return endpointToProbe(probe.KindTCP, e.Name, e.Host, e.Enabled, e)
}

func udpPortToProbe(e config.UDPPortTest) (probe.Probe, error) {
	return endpointToProbe(probe.KindUDP, e.Name, e.Host, e.Enabled, e)
}

func httpEndpointToProbe(e config.HTTPEndpoint) (probe.Probe, error) {
	return endpointToProbe(probe.KindHTTP, e.Name, e.URL, e.Enabled, e)
}

func rtspEndpointToProbe(e config.RTSPEndpoint) (probe.Probe, error) {
	return endpointToProbe(probe.KindRTSP, e.Name, e.URL, e.Enabled, e)
}

func dicomEndpointToProbe(e config.DICOMEndpoint) (probe.Probe, error) {
	return endpointToProbe(probe.KindDICOM, e.Name, e.Host, e.Enabled, e)
}

func hl7EndpointToProbe(e config.HL7Endpoint) (probe.Probe, error) {
	return endpointToProbe(probe.KindHL7, e.Name, e.Host, e.Enabled, e)
}

func fhirEndpointToProbe(e config.FHIREndpoint) (probe.Probe, error) {
	return endpointToProbe(probe.KindFHIR, e.Name, e.BaseURL, e.Enabled, e)
}

func sqlEndpointToProbe(e config.SQLEndpoint) (probe.Probe, error) {
	return endpointToProbe(probe.KindSQL, e.Name, e.Host, e.Enabled, e)
}

func fileShareEndpointToProbe(e config.FileShareEndpoint) (probe.Probe, error) {
	return endpointToProbe(probe.KindFileShare, e.Name, e.Host, e.Enabled, e)
}

func ldapEndpointToProbe(e config.LDAPEndpoint) (probe.Probe, error) {
	return endpointToProbe(probe.KindLDAP, e.Name, e.Host, e.Enabled, e)
}

func ltiEndpointToProbe(e config.LTIEndpoint) (probe.Probe, error) {
	return endpointToProbe(probe.KindLTI, e.Name, e.LaunchURL, e.Enabled, e)
}

func opcuaEndpointToProbe(e config.OPCUAEndpoint) (probe.Probe, error) {
	return endpointToProbe(probe.KindOPCUA, e.Name, e.EndpointURL, e.Enabled, e)
}

func modbusEndpointToProbe(e config.ModbusEndpoint) (probe.Probe, error) {
	return endpointToProbe(probe.KindMODBUS, e.Name, e.Host, e.Enabled, e)
}

// --- per-kind reverse mappings (probe.Probe -> config endpoint) ---

func probeToPingTarget(p probe.Probe) (config.PingTarget, error) {
	e, err := endpointFromProbe[config.PingTarget](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToTCPPort(p probe.Probe) (config.TCPPortTest, error) {
	e, err := endpointFromProbe[config.TCPPortTest](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToUDPPort(p probe.Probe) (config.UDPPortTest, error) {
	e, err := endpointFromProbe[config.UDPPortTest](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToHTTPEndpoint(p probe.Probe) (config.HTTPEndpoint, error) {
	e, err := endpointFromProbe[config.HTTPEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.URL, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToRTSPEndpoint(p probe.Probe) (config.RTSPEndpoint, error) {
	e, err := endpointFromProbe[config.RTSPEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.URL, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToDICOMEndpoint(p probe.Probe) (config.DICOMEndpoint, error) {
	e, err := endpointFromProbe[config.DICOMEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToHL7Endpoint(p probe.Probe) (config.HL7Endpoint, error) {
	e, err := endpointFromProbe[config.HL7Endpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToFHIREndpoint(p probe.Probe) (config.FHIREndpoint, error) {
	e, err := endpointFromProbe[config.FHIREndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.BaseURL, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToSQLEndpoint(p probe.Probe) (config.SQLEndpoint, error) {
	e, err := endpointFromProbe[config.SQLEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToFileShareEndpoint(p probe.Probe) (config.FileShareEndpoint, error) {
	e, err := endpointFromProbe[config.FileShareEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToLDAPEndpoint(p probe.Probe) (config.LDAPEndpoint, error) {
	e, err := endpointFromProbe[config.LDAPEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToLTIEndpoint(p probe.Probe) (config.LTIEndpoint, error) {
	e, err := endpointFromProbe[config.LTIEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.LaunchURL, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToOPCUAEndpoint(p probe.Probe) (config.OPCUAEndpoint, error) {
	e, err := endpointFromProbe[config.OPCUAEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.EndpointURL, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

func probeToModbusEndpoint(p probe.Probe) (config.ModbusEndpoint, error) {
	e, err := endpointFromProbe[config.ModbusEndpoint](p)
	if err != nil {
		return e, err
	}
	e.Name, e.Host, e.Enabled = p.DisplayName, p.Target, p.Enabled
	return e, nil
}

// EndpointsFromProbes assembles the health-check endpoint lists of a
// HealthChecksConfig from probe definitions. Probes of non-health-check kinds
// are ignored. A malformed params row aborts with the error.
func EndpointsFromProbes(probes []probe.Probe) (config.HealthChecksConfig, error) {
	var hc config.HealthChecksConfig
	for _, p := range probes {
		if err := appendProbeToConfig(&hc, p); err != nil {
			return hc, err
		}
	}
	return hc, nil
}

// appendProbeToConfig maps one probe.Probe onto its endpoint list in hc.
// Probes of non-health-check kinds are ignored.
//
//nolint:gocognit,cyclop,funlen // A flat dispatch over the fourteen probe kinds; one case each.
func appendProbeToConfig(hc *config.HealthChecksConfig, p probe.Probe) error {
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

// ProbesFromConfig flattens the endpoint lists of a
// HealthChecksConfig into the probe.Probe slice to persist. Order follows
// Kinds; a marshal failure aborts the whole set.
//
//nolint:gocognit,cyclop // A flat fan over the fourteen endpoint lists; one block each.
func ProbesFromConfig(hc *config.HealthChecksConfig) ([]probe.Probe, error) {
	var out []probe.Probe
	add := func(p probe.Probe, err error) error {
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
