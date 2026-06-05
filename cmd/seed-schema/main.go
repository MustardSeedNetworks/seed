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
// wire (discovery.*/dhcp.*/netif.*/logging.*/survey.*/config), carry
// [json.RawMessage], or self-recurse (a field typed as the DTO itself) are
// deferred to Phase 3, where they get hand-designed flat transport DTOs — e.g.
// GatewayResponse's recursive ipv6 field was split out into the flat
// GatewayPingResult value object so the published schema stays acyclic.
// Unexported lowercase DTOs cannot be referenced here.
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

		// telemetry / network / discovery responses.
		{&api.CableResponse{}, "cable-response.schema.json"},
		{&api.VLANResponse{}, "vlan-response.schema.json"},
		{&api.WiFiResponse{}, "wifi-response.schema.json"},
		{&api.SpeedtestResponse{}, "speedtest-response.schema.json"},
		{&api.RogueDHCPResponse{}, "rogue-dhcp-response.schema.json"},
		{&api.IPConfigResponse{}, "ipconfig-response.schema.json"},
		{&api.DiscoveryResponse{}, "discovery-response.schema.json"},
		{&api.NetworkProblemsResponse{}, "network-problems-response.schema.json"},
		{&api.ProblemScanResponse{}, "problem-scan-response.schema.json"},
		{&api.GatewayResponse{}, "gateway-response.schema.json"},

		// telemetry / network settings.
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

		// iperf / tools / DNS / engine request + result DTOs.
		{&api.IperfClientRequest{}, "iperf-client-request.schema.json"},
		{&api.IperfServerRequest{}, "iperf-server-request.schema.json"},
		{&api.IperfInfoResponse{}, "iperf-info-response.schema.json"},
		{&api.IperfResultResponse{}, "iperf-result-response.schema.json"},
		{&api.PortScanRequest{}, "port-scan-request.schema.json"},
		{&api.TCPProbeRequest{}, "tcp-probe-request.schema.json"},
		{&api.DNSResponse{}, "dns-response.schema.json"},
		{&api.DNSSecurityScanRequest{}, "dns-security-scan-request.schema.json"},
		{&api.EngineScanRequest{}, "engine-scan-request.schema.json"},
		{&api.SetInterfaceRequest{}, "set-interface-request.schema.json"},
		{&api.WiFiSettingsResponse{}, "wifi-settings-response.schema.json"},

		// Users / API tokens / update / SSO / logs.
		{&api.UserResponse{}, "user-response.schema.json"},
		{&api.CreateUserRequest{}, "create-user-request.schema.json"},
		{&api.UpdateUserRequest{}, "update-user-request.schema.json"},
		{&api.MintTokenRequest{}, "mint-token-request.schema.json"},
		{&api.MintTokenResponse{}, "mint-token-response.schema.json"},
		{&api.UpdateCheckResponse{}, "update-check-response.schema.json"},
		{&api.UpdateConfigRequest{}, "update-config-request.schema.json"},
		{&api.UpdateConfigResponse{}, "update-config-response.schema.json"},
		{&api.UpdateStatusResponse{}, "update-status-response.schema.json"},
		{&api.SSOProvidersResponse{}, "sso-providers-response.schema.json"},
		{&api.NVDAPIKeyValidateRequest{}, "nvd-api-key-validate-request.schema.json"},
		{&api.NVDAPIKeyValidateResponse{}, "nvd-api-key-validate-response.schema.json"},
		{&api.RestoreRequest{}, "restore-request.schema.json"},
		{&api.ClientLogRequest{}, "client-log-request.schema.json"},
		{&api.LogStatsResponse{}, "log-stats-response.schema.json"},
		{&api.SNMPv3CredentialResponse{}, "snmpv3-credential-response.schema.json"},

		// Survey (Wi-Fi) request DTOs + profile-import response.
		{&api.CreateSurveyRequest{}, "create-survey-request.schema.json"},
		{&api.AddFloorRequest{}, "add-floor-request.schema.json"},
		{&api.UpdateFloorRequest{}, "update-floor-request.schema.json"},
		{&api.UpdateFloorPlanRequest{}, "update-floor-plan-request.schema.json"},
		{&api.SetActiveFloorRequest{}, "set-active-floor-request.schema.json"},
		{&api.AddFloorSampleRequest{}, "add-floor-sample-request.schema.json"},
		{&api.AddSampleRequest{}, "add-sample-request.schema.json"},
		{&api.UpdateSurveySettingsRequest{}, "update-survey-settings-request.schema.json"},
		{&api.GenerateReportRequest{}, "generate-report-request.schema.json"},
		{&api.ProfileImportResponse{}, "profile-import-response.schema.json"},

		// Settings composers: top-level DTOs whose entire transitive closure is
		// flat, local transport sub-structs in internal/api (every composed
		// *Response is itself registered above). They compose, but never reach a
		// domain type, json.RawMessage, or a self-reference, so the published
		// schema stays self-contained and acyclic.
		{&api.NetworkDiscoverySettingsResponse{}, "network-discovery-settings-response.schema.json"},
		{&api.OptionsResponse{}, "options-response.schema.json"},
		{&api.SNMPSettingsResponse{}, "snmp-settings-response.schema.json"},
		{&api.TestsSettingsResponse{}, "tests-settings-response.schema.json"},
		{&api.IperfClientStatusResponse{}, "iperf-client-status-response.schema.json"},

		// Domain-nested DTOs that now carry flat transport mirrors (the handler
		// maps the discovery/dhcp/netif/logging domain value onto a local
		// purpose-built sub-struct), so the published schema no longer reaches
		// into a domain package.
		{&api.TCPProbeResponse{}, "tcp-probe-response.schema.json"},
		{&api.RogueServersResponse{}, "rogue-servers-response.schema.json"},
		{&api.CategorizedInterfacesResponse{}, "categorized-interfaces-response.schema.json"},
		{&api.LogQueryResponse{}, "log-query-response.schema.json"},
		{&api.BluetoothScanResponse{}, "bluetooth-scan-response.schema.json"},
		{&api.BluetoothDevicesResponse{}, "bluetooth-devices-response.schema.json"},
		{&api.BluetoothStatsResponse{}, "bluetooth-stats-response.schema.json"},
		{&api.WiFiDiscoveryScanResponse{}, "wifi-discovery-scan-response.schema.json"},
		{&api.WiFiDiscoveryNetworksResponse{}, "wifi-discovery-networks-response.schema.json"},
		{&api.WiFiDiscoveryAPsResponse{}, "wifi-discovery-aps-response.schema.json"},
		{&api.WiFiDiscoveryStatsResponse{}, "wifi-discovery-stats-response.schema.json"},
		{&api.PathResponse{}, "path-response.schema.json"},
		{&api.UpdateSurveyImportedDataRequest{}, "update-survey-imported-data-request.schema.json"},

		// Profile envelope DTOs. The backend treats the per-profile Config as an
		// opaque JSON blob (json.RawMessage) — it only ever inspects the
		// `interface` block for license gating and otherwise stores/forwards it
		// verbatim — so the published schema documents config as arbitrary JSON.
		// Modeling that blob as typed Go structs, and retiring the hand-written
		// profile.ts/settings.ts TS twins, is deferred to Phase 7 (frontend
		// re-architecture); here we just complete contract coverage of the
		// envelope itself.
		{&api.ProfileRequest{}, "profile-request.schema.json"},
		{&api.ProfileResponse{}, "profile-response.schema.json"},
		{&api.ProfileListResponse{}, "profile-list-response.schema.json"},
		{&api.ProfileImportRequest{}, "profile-import-request.schema.json"},
		{&api.ProfileExportResponse{}, "profile-export-response.schema.json"},

		// Jobs spine — unified job runner transport (ADR-0005, §8). The job
		// request params and the job result are deliberately opaque: params is
		// json.RawMessage (each kind validates its own shape) and result is a
		// bare interface (kind-specific payload), so both document as arbitrary
		// JSON. Registration was deferred in #1468 until a frontend consumer
		// existed; Phase 7 S1 adds the TS /jobs client that consumes these.
		{&api.CreateJobRequest{}, "create-job-request.schema.json"},
		{&api.JobResponse{}, "job-response.schema.json"},
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
