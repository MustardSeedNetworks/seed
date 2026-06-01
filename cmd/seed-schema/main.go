// seed-schema generates JSON Schemas for seed's HTTP API request DTOs
// directly from the Go types, so the wire contract documented in
// docs/schemas/ stays in sync with internal/api/.
//
// Mirrors the pattern documented in ADR 0001 of krisarmstrong/niac-go
// (docs/adr/0001-schema-generation-from-go-structs.md). Each top-level
// DTO becomes its own schema file under docs/schemas/api/; clients and
// future TypeScript codegen can consume them as a stable contract.
//
// Usage:
//
//	seed-schema -o docs/schemas/api
//
// The output directory is created if it doesn't exist. Existing files
// are overwritten — this command is the source of truth.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/invopop/jsonschema"

	"github.com/krisarmstrong/seed/internal/api"
)

// schemaTarget pairs a Go DTO with the on-disk schema filename and a
// human-readable title. The title is the Go type name (derived, never
// hand-written); the filename is explicit so legacy names that predate the
// kebab convention (login.schema.json, ipconfig-response.schema.json,
// set-mtu.schema.json …) stay stable for existing clients.
type schemaTarget struct {
	value    any    // pointer to a zero-value of the DTO
	filename string // filename without directory (e.g., "login.schema.json")
	title    string // human-readable schema title (the Go type name)
}

// reg is one row of the schema registry: a DTO pointer and the schema file it
// is written to. It exists so the registry below reads as a flat data table —
// one line per DTO — rather than a set of near-identical builder functions.
type reg struct {
	value    any
	filename string
}

// schemaTargets is the single source of truth for the DTOs we publish schemas
// for: one declarative row per DTO, grouped by comment only. It is a data
// table, not logic — adding a DTO is one line — so its length is expected to
// grow (funlen is intentionally relaxed for this file). The title is taken
// from the Go type name so it can never drift from the type.
//
// POLICY (RE_ARCHITECTURE_BLUEPRINT.md Phase 2 — flat + self-contained): a DTO
// belongs here iff it is flat or nests only local, purpose-built transport
// sub-structs in internal/api. DTOs that put an internal domain type on the
// wire (discovery.*/dhcp.*/netif.*/logging.*/survey.*/config), compose another
// top-level *Response/*Request, carry [json.RawMessage], or self-recurse
// (GatewayResponse.ipv6) are deferred to Phase 3, where they get hand-designed
// flat transport DTOs. Unexported lowercase DTOs cannot be referenced here.
//
// Function rather than a package-level var to keep gochecknoglobals happy and
// so `go run` doesn't pull internal/api into an init side effect.
func schemaTargets() []schemaTarget {
	rows := []reg{
		// Request DTOs — strict-decode + validator surface (#1102).
		{&api.LoginRequest{}, "login.schema.json"},
		{&api.SetupCompleteRequest{}, "setup-complete.schema.json"},
		{&api.RecoveryCompleteRequest{}, "recovery-complete.schema.json"},
		{&api.SetMTURequest{}, "set-mtu.schema.json"},
		{&api.PathRequest{}, "path.schema.json"},
		{&api.WiFiConnectRequest{}, "wifi-connect.schema.json"},
		{&api.TracerouteRequest{}, "traceroute-request.schema.json"},

		// Auth / status / recovery / config responses.
		{&api.StatusResponse{}, "status-response.schema.json"},
		{&api.LoginResponse{}, "login-response.schema.json"},
		{&api.CSRFTokenResponse{}, "csrf-token-response.schema.json"},
		{&api.SetupStatusResponse{}, "setup-status-response.schema.json"},
		{&api.LicenseStatusResponse{}, "license-status-response.schema.json"},
		{&api.ErrorResponse{}, "error-response.schema.json"},
		{&api.FeatureGateResponse{}, "feature-gate-response.schema.json"},
		{&api.RecoveryStatusResponse{}, "recovery-status-response.schema.json"},
		{&api.RecoveryInstructionsResponse{}, "recovery-instructions-response.schema.json"},
		{&api.RecoveryCompleteResponse{}, "recovery-complete-response.schema.json"},
		{&api.ConfigVersionResponse{}, "config-version-response.schema.json"},
		{&api.BackupListResponse{}, "backup-list-response.schema.json"},

		// SAP / network / discovery responses.
		{&api.CableResponse{}, "cable-response.schema.json"},
		{&api.VLANResponse{}, "vlan-response.schema.json"},
		{&api.WiFiResponse{}, "wifi-response.schema.json"},
		{&api.SpeedtestResponse{}, "speedtest-response.schema.json"},
		{&api.RogueDHCPResponse{}, "rogue-dhcp-response.schema.json"},
		{&api.IPConfigResponse{}, "ipconfig-response.schema.json"},
		{&api.DiscoveryResponse{}, "discovery-response.schema.json"},
		{&api.NetworkProblemsResponse{}, "network-problems-response.schema.json"},
		{&api.ProblemScanResponse{}, "problem-scan-response.schema.json"},

		// SAP / network settings.
		{&api.IPSettingsRequest{}, "ip-settings-request.schema.json"},
		{&api.IPSettingsResponse{}, "ip-settings-response.schema.json"},
		{&api.SubnetRequest{}, "subnet-request.schema.json"},
		{&api.SubnetResponse{}, "subnet-response.schema.json"},
		{&api.VLANInterfaceRequest{}, "vlan-interface-request.schema.json"},
		{&api.VLANTrafficResponse{}, "vlan-traffic-response.schema.json"},
		{&api.SpeedtestStatusResponse{}, "speedtest-status-response.schema.json"},
		{&api.RogueDHCPConfigResponse{}, "rogue-dhcp-config-response.schema.json"},
		{&api.DNSServerResponse{}, "dns-server-response.schema.json"},
		{&api.LinkResponse{}, "link-response.schema.json"},

		// Health-check endpoint responses (per protocol).
		{&api.DICOMEndpointResponse{}, "dicom-endpoint-response.schema.json"},
		{&api.FHIREndpointResponse{}, "fhir-endpoint-response.schema.json"},
		{&api.FileShareEndpointResponse{}, "file-share-endpoint-response.schema.json"},
		{&api.HL7EndpointResponse{}, "hl7-endpoint-response.schema.json"},
		{&api.HTTPEndpointResponse{}, "http-endpoint-response.schema.json"},
		{&api.LDAPEndpointResponse{}, "ldap-endpoint-response.schema.json"},
		{&api.LTIEndpointResponse{}, "lti-endpoint-response.schema.json"},
		{&api.ModbusEndpointResponse{}, "modbus-endpoint-response.schema.json"},
		{&api.OPCUAEndpointResponse{}, "opcua-endpoint-response.schema.json"},
		{&api.RTSPEndpointResponse{}, "rtsp-endpoint-response.schema.json"},
		{&api.SQLEndpointResponse{}, "sql-endpoint-response.schema.json"},
		{&api.PingTargetResponse{}, "ping-target-response.schema.json"},

		// Health-check / discovery settings value objects.
		{&api.TCPPortResponse{}, "tcp-port-response.schema.json"},
		{&api.UDPPortResponse{}, "udp-port-response.schema.json"},
		{&api.IperfSettingsResponse{}, "iperf-settings-response.schema.json"},
		{&api.SpeedtestSettingsResponse{}, "speedtest-settings-response.schema.json"},
		{&api.TCPProbeSettingsResponse{}, "tcp-probe-settings-response.schema.json"},
		{&api.PassiveProtocolResponse{}, "passive-protocol-response.schema.json"},
		{&api.PortScanResponse{}, "port-scan-response.schema.json"},
		{&api.ProfilerResponse{}, "profiler-response.schema.json"},
		{&api.TimingResponse{}, "timing-response.schema.json"},
		{&api.FingerprintingResponse{}, "fingerprinting-response.schema.json"},
	}

	targets := make([]schemaTarget, len(rows))
	for i, row := range rows {
		targets[i] = schemaTarget{
			value:    row.value,
			filename: row.filename,
			title:    reflect.TypeOf(row.value).Elem().Name(),
		}
	}

	return targets
}

func main() {
	outDir := flag.String("o", "docs/schemas/api", "output directory")
	flag.Parse()

	// 0o750 is the strictest mode that still lets the operator's group
	// list the dir; gosec G301 flags anything looser.
	if err := os.MkdirAll(*outDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir %s: %v\n", *outDir, err)
		os.Exit(1)
	}

	reflector := newReflector()

	for _, target := range schemaTargets() {
		schema := reflector.Reflect(target.value)
		schema.Title = target.title
		schema.Description = fmt.Sprintf(
			"%s — generated from the Go struct in internal/api; refresh with `make schema` after struct changes.",
			target.title,
		)
		schema.ID = jsonschema.ID(fmt.Sprintf(
			"https://raw.githubusercontent.com/krisarmstrong/seed/main/docs/schemas/api/%s",
			target.filename,
		))

		data, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal %s: %v\n", target.filename, err)
			os.Exit(1)
		}
		data = append(data, '\n')

		path := filepath.Join(*outDir, target.filename)
		if writeErr := os.WriteFile(path, data, 0o600); writeErr != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", path, writeErr)
			os.Exit(1)
		}
		// Routed through stderr so stdout can stay reserved for
		// generator output that downstream tooling might consume; the
		// trailing summary is operator feedback, not data.
		fmt.Fprintf(os.Stderr, "wrote %s\n", path)
	}
}

// newReflector returns a jsonschema.Reflector configured for HTTP API
// DTOs:
//
//   - FieldNameTag: "json" — schemas reflect the wire format clients see,
//     not the Go field casing
//   - AllowAdditionalProperties: false — schemas match the
//     DisallowUnknownFields posture in decodeJSONStrict (#1100/#1101)
//   - Anonymous: true — nested types are inlined rather than producing
//     $ref indirection, which makes schemas easier to consume by tools
//     that don't resolve refs across files
func newReflector() *jsonschema.Reflector {
	r := &jsonschema.Reflector{
		ExpandedStruct:            false,
		Anonymous:                 true,
		AllowAdditionalProperties: false,
	}
	r.KeyNamer = func(s string) string { return s }
	r.FieldNameTag = "json"
	return r
}
