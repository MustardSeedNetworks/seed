// Package apischema is the registry of the HTTP API's published DTOs and the
// reflector that turns each into a JSON Schema. cmd/seed-schema writes one
// schema file per DTO; cmd/seed-openapi publishes the same schemas as the
// OpenAPI document's components, so the two descriptions share one source.
package apischema

import (
	"reflect"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"
	"github.com/invopop/jsonschema"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/iperf"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/multicast"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/packetcapture"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/qos"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/bonjour"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/wifi"
)

// Target pairs a Go DTO with the on-disk schema filename and a human-readable
// title. The title is the Go type name (derived, never hand-written); the
// filename is explicit so legacy names that predate the kebab convention
// (login.schema.json, ipconfig-response.schema.json, set-mtu.schema.json …)
// stay stable for existing clients.
type Target struct {
	Value    any    // pointer to a zero-value of the DTO
	Filename string // filename without directory (e.g., "login.schema.json")
	Title    string // human-readable schema title (the Go type name)
}

// reg is one row of the schema registry: a DTO pointer and the schema file it
// is written to. It exists so the registry below reads as a flat data table —
// one line per DTO — rather than a set of near-identical builder functions.
type reg struct {
	value    any
	filename string
}

// Targets is the single source of truth for the DTOs we publish schemas
// for: one declarative row per DTO, grouped by comment only. It is a data
// table, not logic — adding a DTO is one line — so its length is expected to
// grow (funlen is intentionally relaxed for this file). The title is taken
// from the Go type name so it can never drift from the type.
//
// POLICY (RE_ARCHITECTURE_BLUEPRINT.md Phase 2 — flat + self-contained): a DTO
// belongs here iff it is flat or nests only local, purpose-built transport
// sub-structs in internal/api. DTOs that put an internal domain type on the
// wire (discovery.*/dhcp.*/netif.*/logging.*/config), carry
// [json.RawMessage], or self-recurse (a field typed as the DTO itself) are
// deferred to Phase 3, where they get hand-designed flat transport DTOs — e.g.
// GatewayResponse's recursive ipv6 field was split out into the flat
// GatewayPingResult value object so the published schema stays acyclic.
// Unexported lowercase DTOs cannot be referenced here.
//
// EXCEPTION (ADR-0008): genuinely PURE-DATA domain types whose JSON shape the
// endpoint already serializes at runtime may be reflected directly, rather than
// hand-mirrored. EngineDiscoveryResponse uses this for the discovery.* result
// cluster (DiscoveredDevice/EngineStats/ScanResult and their pure-data closure):
// internal/discovery is CGO-free, the /discovery/engine endpoint already emits
// these structs, and the cluster carries no behavior/channels/funcs. The
// exception is narrow — behavior- or I/O-bearing domain types still get a flat
// mirror. The round-trip + acyclicity guardrails enforce it either way.
//
// Function rather than a package-level var to keep gochecknoglobals happy and
// so `go run` doesn't pull internal/api into an init side effect.
func Targets() []Target {
	rows := []reg{
		// Request DTOs — strict-decode + validator surface (#1102).
		{&api.LoginRequest{}, "login.schema.json"},
		{&api.SetupCompleteRequest{}, "setup-complete.schema.json"},
		{&api.SetupCompleteResponse{}, "setup-complete-response.schema.json"},
		{&api.RecoveryCompleteRequest{}, "recovery-complete.schema.json"},
		{&api.SetMTURequest{}, "set-mtu.schema.json"},
		{&api.SetMTUResponse{}, "set-mtu-response.schema.json"},
		{&api.PathRequest{}, "path.schema.json"},
		{&api.WiFiConnectRequest{}, "wifi-connect.schema.json"},
		{&api.TracerouteRequest{}, "traceroute-request.schema.json"},

		// Auth / status / recovery / config responses.
		{&api.StatusResponse{}, "status-response.schema.json"},
		{&api.HealthResponse{}, "health-response.schema.json"},
		{&api.EnginesResponse{}, "engines-response.schema.json"},
		{&api.NeighbourCacheResponse{}, "neighbour-cache-response.schema.json"},
		{&api.DriverStatsResponse{}, "driver-stats-response.schema.json"},
		{&api.LoginResponse{}, "login-response.schema.json"},
		{&api.LogoutResponse{}, "logout-response.schema.json"},
		{&api.TOTPLoginRequest{}, "totp-login-request.schema.json"},
		{&api.TOTPSetupResponse{}, "totp-setup-response.schema.json"},
		{&api.TOTPVerifyRequest{}, "totp-verify-request.schema.json"},
		{&api.TOTPDisableRequest{}, "totp-disable-request.schema.json"},
		{&api.MFAToggleResponse{}, "mfa-toggle-response.schema.json"},
		{&api.MFAStatusResponse{}, "mfa-status-response.schema.json"},
		{&api.WebAuthnCreationOptionsResponse{}, "webauthn-creation-options-response.schema.json"},
		{&api.WebAuthnLoginBeginRequest{}, "webauthn-login-begin-request.schema.json"},
		{&api.WebAuthnRequestOptionsResponse{}, "webauthn-request-options-response.schema.json"},
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
		{&api.WiFiAirspaceResponse{}, "wifi-airspace-response.schema.json"},
		{&api.WiFiAnomaliesResponse{}, "wifi-anomalies-response.schema.json"},
		{&api.SpeedtestResponse{}, "speedtest-response.schema.json"},
		{&api.RogueDHCPResponse{}, "rogue-dhcp-response.schema.json"},
		{&api.IPConfigResponse{}, "ipconfig-response.schema.json"},
		{&api.DiscoveryResponse{}, "discovery-response.schema.json"},
		{&api.ReportsResponse{}, "reports-response.schema.json"},
		{&api.GenerateReportRequest{}, "generate-report-request.schema.json"},
		{&api.ReportScheduleRequest{}, "report-schedule-request.schema.json"},
		{&api.ReportScheduleInfo{}, "report-schedule-info.schema.json"},
		{&api.ReportSchedulesResponse{}, "report-schedules-response.schema.json"},
		{&api.NetworkProblemsResponse{}, "network-problems-response.schema.json"},
		{&api.ProblemScanResponse{}, "problem-scan-response.schema.json"},
		{&api.GatewayResponse{}, "gateway-response.schema.json"},

		// telemetry / network settings.
		{&api.IPSettingsRequest{}, "ip-settings-request.schema.json"},
		{&api.IPSettingsResponse{}, "ip-settings-response.schema.json"},
		{&api.SubnetRequest{}, "subnet-request.schema.json"},
		{&api.SubnetResponse{}, "subnet-response.schema.json"},
		{&api.SubnetDecisionRequest{}, "subnet-decision-request.schema.json"},
		{&api.VLANInterfaceRequest{}, "vlan-interface-request.schema.json"},
		{&api.VLANInterfaceResponse{}, "vlan-interface-response.schema.json"},
		{&api.SpeedtestStartResponse{}, "speedtest-start-response.schema.json"},
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
		{&api.PassiveProtocolResponse{}, "passive-protocol-response.schema.json"},
		{&api.PortScanSettingsResponse{}, "port-scan-settings-response.schema.json"},
		{&api.TimingResponse{}, "timing-response.schema.json"},
		{&api.FingerprintingResponse{}, "fingerprinting-response.schema.json"},

		// iperf / tools / DNS / engine request + result DTOs.
		{&api.IperfClientRequest{}, "iperf-client-request.schema.json"},
		{&api.IperfServerRequest{}, "iperf-server-request.schema.json"},
		{&api.IperfInfoResponse{}, "iperf-info-response.schema.json"},
		{&api.IperfResultResponse{}, "iperf-result-response.schema.json"},
		{&api.IperfSuggestion{}, "iperf-suggestion.schema.json"},
		{&api.PortScanRequest{}, "port-scan-request.schema.json"},
		{&api.PortScanResponse{}, "port-scan-response.schema.json"},
		{&api.TCPProbeRequest{}, "tcp-probe-request.schema.json"},
		{&api.DNSResponse{}, "dns-response.schema.json"},
		{&api.EngineScanRequest{}, "engine-scan-request.schema.json"},
		{&api.VulnScanRequest{}, "vuln-scan-request.schema.json"},
		{&api.SetInterfaceRequest{}, "set-interface-request.schema.json"},
		{&api.SetInterfaceResponse{}, "set-interface-response.schema.json"},
		{&api.InterfaceSelectionResponse{}, "interface-selection-response.schema.json"},
		{&api.WiFiSettingsResponse{}, "wifi-settings-response.schema.json"},
		{&api.WiFiStatusResponse{}, "wifi-status-response.schema.json"},

		// Users / API tokens / update / SSO / logs.
		{&api.UserResponse{}, "user-response.schema.json"},
		{&api.CreateUserRequest{}, "create-user-request.schema.json"},
		{&api.UpdateUserRequest{}, "update-user-request.schema.json"},
		{&api.MintTokenRequest{}, "mint-token-request.schema.json"},
		{&api.MintTokenResponse{}, "mint-token-response.schema.json"},
		{&api.TokenListItem{}, "token-list-item.schema.json"},
		{&api.SSOProvidersResponse{}, "sso-providers-response.schema.json"},
		{&api.SSOSettingsResponse{}, "sso-settings-response.schema.json"},
		{&api.SSOUpdateRequest{}, "sso-update-request.schema.json"},
		{&api.NVDAPIKeyValidateRequest{}, "nvd-api-key-validate-request.schema.json"},
		{&api.NVDAPIKeyValidateResponse{}, "nvd-api-key-validate-response.schema.json"},
		{&api.VulnerabilityStatusResponse{}, "vulnerability-status-response.schema.json"},
		{&api.VulnerabilitySettings{}, "vulnerability-settings.schema.json"},
		{&api.VulnerabilitySettingsUpdate{}, "vulnerability-settings-update.schema.json"},
		{&api.VulnerabilityResultsResponse{}, "vulnerability-results-response.schema.json"},
		{&api.VulnFindingsResponse{}, "vuln-findings-response.schema.json"},
		{&api.VulnFindingStatusRequest{}, "vuln-finding-status-request.schema.json"},
		{&api.VulnFindingStatusResponse{}, "vuln-finding-status-response.schema.json"},
		{&api.VulnFindingHistoryResponse{}, "vuln-finding-history-response.schema.json"},
		{&api.RestoreRequest{}, "restore-request.schema.json"},
		{&api.ConfigBackupResponse{}, "config-backup-response.schema.json"},
		{&api.ClientLogRequest{}, "client-log-request.schema.json"},
		{&api.ClientLogResponse{}, "client-log-response.schema.json"},
		{&api.LogStatsResponse{}, "log-stats-response.schema.json"},
		{&api.LogsRecentResponse{}, "logs-recent-response.schema.json"},
		{&api.LogFileResponse{}, "log-file-response.schema.json"},

		// Profile-import response.
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
		{&api.WiFiDiscoveryScanResponse{}, "wifi-discovery-scan-response.schema.json"},
		{&api.WiFiDiscoveryNetworksResponse{}, "wifi-discovery-networks-response.schema.json"},
		{&api.WiFiDiscoveryAPsResponse{}, "wifi-discovery-aps-response.schema.json"},
		{&api.WiFiDiscoveryStatsResponse{}, "wifi-discovery-stats-response.schema.json"},
		{&api.PathResponse{}, "path-response.schema.json"},
		{&api.IPAMSubnetsResponse{}, "ipam-subnets-response.schema.json"},

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
		{&api.ProfileDuplicateRequest{}, "profile-duplicate-request.schema.json"},
		{&api.ProfileSwitchRequest{}, "profile-switch-request.schema.json"},
		{&api.ProfileSwitchResponse{}, "profile-switch-response.schema.json"},

		// Jobs spine — unified job runner transport (ADR-0005, §8). The job
		// request params and the job result are deliberately opaque: params is
		// json.RawMessage (each kind validates its own shape) and result is a
		// bare interface (kind-specific payload), so both document as arbitrary
		// JSON. Registration was deferred in #1468 until a frontend consumer
		// existed; Phase 7 S1 adds the TS /jobs client that consumes these.
		{&api.CreateJobRequest{}, "create-job-request.schema.json"},
		{&api.JobResponse{}, "job-response.schema.json"},
		{&api.ProbeHistoryResponse{}, "probe-history-response.schema.json"},
		{&api.AnomalyHistoryResponse{}, "anomaly-history-response.schema.json"},
		{&api.InterfaceStatsListResponse{}, "interface-stats-list-response.schema.json"},
		{&api.InterfaceHistoryResponse{}, "interface-history-response.schema.json"},

		// Top-N reads over collected flows (P-C3, P-C4), consumed by the flow
		// explorer (UI-SEED-23).
		{&api.FlowTalkersResponse{}, "flow-talkers-response.schema.json"},
		{&api.FlowConversationsResponse{}, "flow-conversations-response.schema.json"},
		{&api.FlowApplicationsResponse{}, "flow-applications-response.schema.json"},

		// The operator-editable tables the flow collector names and checks
		// flows against (P-C4, P-C5).
		{&api.AppSignaturesRequest{}, "app-signatures-request.schema.json"},
		{&api.AppSignaturesResponse{}, "app-signatures-response.schema.json"},
		{&api.FlowIndicatorsRequest{}, "flow-indicators-request.schema.json"},
		{&api.FlowIndicatorsResponse{}, "flow-indicators-response.schema.json"},

		// The caller's own dashboard layout (UI-SEED-22): PUT body and the
		// GET/PUT response.
		{&api.DashboardLayoutRequest{}, "dashboard-layout-request.schema.json"},
		{&api.DashboardLayout{}, "dashboard-layout.schema.json"},

		// #364's Bonjour browse. Registered as the bonjour package's own type
		// rather than an api mirror of it: a mirror is one more thing to drift
		// and the handler serves this struct verbatim.
		{&bonjour.BrowseResult{}, "bonjour-browse-response.schema.json"},

		// #399's multicast listen and observation, jobs-spine kinds: the request is
		// the kind's params and the result is the job's result. Registered as the
		// multicast package's own types for the same reason as the browse.
		{&multicast.ListenRequest{}, "multicast-listen-request.schema.json"},
		{&multicast.ListenResult{}, "multicast-listen-response.schema.json"},
		{&multicast.ObserveRequest{}, "multicast-observe-request.schema.json"},
		{&multicast.ObserveResult{}, "multicast-observe-response.schema.json"},

		// #400's DSCP preservation check, three jobs-spine kinds (one per host,
		// or both on one host), registered as the qos package's own types like
		// the listen above.
		{&qos.SendRequest{}, "qos-send-request.schema.json"},
		{&qos.SendResult{}, "qos-send-response.schema.json"},
		{&qos.ListenRequest{}, "qos-listen-request.schema.json"},
		{&qos.ListenResult{}, "qos-listen-response.schema.json"},
		{&qos.SingleHostRequest{}, "qos-single-host-request.schema.json"},
		{&qos.SingleHostResult{}, "qos-single-host-response.schema.json"},

		// The iperf3 server's state, which /telemetry/iperf/server/status
		// serves as it is: flat, pure data.
		{&iperf.ServerStatus{}, "iperf-server-status.schema.json"},

		// The public address lookup, which /telemetry/publicip serves as the
		// publicip package's own flat type, named for the API.
		{&api.PublicIPResponse{}, "public-ip-response.schema.json"},

		// The guest-network isolation audit's report, which
		// /security/guest-audit/run serves as the guestaudit package's own
		// flat type, named for the API.
		{&api.GuestAuditReport{}, "guest-audit-report.schema.json"},

		// The diagnostic export document /reporting/export downloads, the
		// export use-case's own flat card types, named for the API.
		{&api.ExportResponse{}, "export-response.schema.json"},

		// #326's packet capture, a jobs-spine kind registered as the
		// packetcapture package's own types like the two above.
		{&packetcapture.Request{}, "packet-capture-request.schema.json"},
		{&packetcapture.Result{}, "packet-capture-response.schema.json"},

		// #165's continuous path monitor, a jobs-spine kind. Each round's
		// accumulated view arrives as a `pathMonitor` SSE frame, and the job's
		// result is the same shape at the moment it stopped.
		{&api.PathMonitorRequest{}, "path-monitor-request.schema.json"},
		{&api.PathMonitorUpdate{}, "path-monitor-update.schema.json"},

		// #395's multi-path egress run and #435's path MTU discovery, two
		// jobs-spine kinds whose results are the discovery engines' own types.
		{&api.MultiPathRequest{}, "path-multipath-request.schema.json"},
		{&discovery.MultiPathResult{}, "path-multipath-response.schema.json"},
		{&api.PathMTURequest{}, "path-mtu-request.schema.json"},
		{&discovery.PMTUDResult{}, "path-mtu-response.schema.json"},

		// The problem detector's thresholds, which /security/problems/thresholds
		// reads and replaces as they are: flat, pure data.
		{&discovery.ProblemThresholds{}, "problem-thresholds.schema.json"},

		// Discovery status reads, and /__version's build metadata.
		{&api.DeviceDiscoveryStatusResponse{}, "device-discovery-status-response.schema.json"},
		{&api.DevicesResponse{}, "devices-response.schema.json"},
		{&api.DeviceScanResponse{}, "device-scan-response.schema.json"},
		// One host's advanced probe (POST /security/discovery/fingerprint):
		// pure data the handler already serializes, reflected directly.
		{&api.FingerprintRequest{}, "fingerprint-request.schema.json"},
		{&discovery.AdvancedProbeResult{}, "advanced-probe-result.schema.json"},
		{&api.DiscoveryOptionsResponse{}, "discovery-options-response.schema.json"},
		{&api.DiscoveryServiceStatusResponse{}, "discovery-service-status-response.schema.json"},
		{&discovery.EngineCapabilities{}, "engine-capabilities.schema.json"},
		{&api.BuildVersionResponse{}, "build-version-response.schema.json"},
		// One /__capabilities manifest entry: the registrar's own pure-data
		// policy record, which the manifest serializes as it is.
		{&route.Policy{}, "route-policy.schema.json"},

		// SNMP polling targets and the device-credential vault (#1799). The
		// vault answers polling.Credentials as it is: pure data whose
		// ciphertext fields are json:"-", so no schema can carry a secret.
		{&api.PollingTargetRequest{}, "polling-target-request.schema.json"},
		{&api.PollingTargetResponse{}, "polling-target-response.schema.json"},
		{&api.PollingTargetListResponse{}, "polling-target-list-response.schema.json"},
		{&api.DeviceCredentialRequest{}, "device-credential-request.schema.json"},
		{&polling.Credentials{}, "device-credential.schema.json"},
		{&api.DeviceCredentialListResponse{}, "device-credential-list-response.schema.json"},

		// The topology graph read API (Stage A5.1).
		{&api.TopologyNodeListResponse{}, "topology-node-list-response.schema.json"},
		{&api.TopologyNodeDetailResponse{}, "topology-node-detail-response.schema.json"},
		{&api.TopologyLinkListResponse{}, "topology-link-list-response.schema.json"},
		{&api.TopologyARPListResponse{}, "topology-arp-list-response.schema.json"},

		// The Wi-Fi radio: neighbor scan, saved networks, interface selection
		// and the connect/disconnect result, which /wifi/wifi/disconnect
		// serves as the wifi package's own flat type.
		{&api.WiFiScanResponse{}, "wifi-scan-response.schema.json"},
		{&api.WiFiChannelGraphResponse{}, "wifi-channel-graph-response.schema.json"},
		{&api.WiFiSavedNetworksResponse{}, "wifi-saved-networks-response.schema.json"},
		{&api.WiFiSettingsRequest{}, "wifi-settings-request.schema.json"},
		{&wifi.ConnectionResult{}, "wifi-connection-result.schema.json"},

		// Acknowledgement-only success bodies.
		{&api.StatusMessageResponse{}, "status-message-response.schema.json"},
		{&api.StatusUpdatedResponse{}, "status-updated-response.schema.json"},
		{&api.RenewDHCPLeaseRequest{}, "renew-dhcp-lease-request.schema.json"},
		{&api.RenewDHCPLeaseResponse{}, "renew-dhcp-lease-response.schema.json"},
		{&api.DHCPLeaseRequest{}, "dhcp-lease-request.schema.json"},
		{&api.DHCPLeaseResponse{}, "dhcp-lease-response.schema.json"},

		// The health-check run and the probe engine's active anomalies.
		{&api.HealthCheckRunResponse{}, "health-check-run-response.schema.json"},
		{&api.ProbeAnomaliesResponse{}, "probe-anomalies-response.schema.json"},

		// The host and daemon health answer.
		{&api.SystemHealthResponse{}, "system-health-response.schema.json"},

		// The alert test-send action on a saved receiver (#2997).
		{&api.AlertTestSendRequest{}, "alert-test-send-request.schema.json"},
		{&api.AlertTestSendResponse{}, "alert-test-send-response.schema.json"},

		// Operator-defined alert rules (Stage A5.10).
		{&api.AlertRuleRequest{}, "alert-rule-request.schema.json"},
		{&api.AlertRuleResponse{}, "alert-rule-response.schema.json"},
		{&api.AlertRuleListResponse{}, "alert-rule-list-response.schema.json"},

		// The alert inbox and its acknowledge/resolve actions (Stage A5.2).
		{&api.AlertListResponse{}, "alert-list-response.schema.json"},
		{&api.AlertAcknowledgeResponse{}, "alert-acknowledge-response.schema.json"},
		{&api.AlertResolveResponse{}, "alert-resolve-response.schema.json"},

		// Profile/settings config — code-first model of the per-profile
		// config.Config blob (ADR-0007/0008, Phase 7 S6). The profile Config
		// is applied via Config.ApplyProfileJSON, so config.Config is its
		// canonical shape; this generates the Config TS type that replaces the
		// hand-written profile.ts/settings.ts twins. Pure data (the mutex is
		// json:"-").
		{&config.Config{}, "config.schema.json"},

		// Backend defaults for the settings pages (GET /settings/defaults).
		// config.DefaultSettings is already a purpose-built wire shape of
		// pure data, so it is reflected directly and replaces the
		// hand-written ui/src/types/defaults.ts twin.
		{&config.DefaultSettings{}, "default-settings.schema.json"},

		// GET /settings: the stored settings the settings drawer edits.
		{&api.SettingsResponse{}, "settings-response.schema.json"},

		// Engine discovery result — the ADR-0008 pure-data exception. Reflects
		// the discovery.* result cluster (DiscoveredDevice/EngineStats/
		// ScanResult and their pure-data closure) directly rather than
		// hand-mirroring ~150 fields: internal/discovery is CGO-free, the
		// /discovery/engine endpoint already serializes these structs, and the
		// cluster carries no behavior. Unblocks the Phase 7 frontend consuming
		// typed engine results off the /jobs spine.
		{&api.EngineDiscoveryResponse{}, "engine-discovery-response.schema.json"},
	}

	targets := make([]Target, len(rows))
	for i, row := range rows {
		targets[i] = Target{
			Value:    row.value,
			Filename: row.filename,
			Title:    reflect.TypeOf(row.value).Elem().Name(),
		}
	}

	return targets
}

// NewReflector returns a jsonschema.Reflector configured for HTTP API
// DTOs:
//
//   - FieldNameTag: "json" — schemas reflect the wire format clients see,
//     not the Go field casing
//   - AllowAdditionalProperties: false — schemas match the
//     DisallowUnknownFields posture in decodeJSONStrict (#1100/#1101)
//   - Anonymous: true — nested types are inlined rather than producing
//     $ref indirection, which makes schemas easier to consume by tools
//     that don't resolve refs across files
func NewReflector() *jsonschema.Reflector {
	r := &jsonschema.Reflector{
		ExpandedStruct:            false,
		Anonymous:                 true,
		AllowAdditionalProperties: false,
	}
	r.KeyNamer = func(s string) string { return s }
	r.FieldNameTag = "json"
	return r
}
