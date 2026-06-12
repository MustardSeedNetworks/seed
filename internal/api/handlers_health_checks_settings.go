package api

// handlers_health_checks_settings.go contains the GET/PUT handlers and the
// per-endpoint response builders for the health-checks settings endpoint.
// The corresponding request/response types live in
// handlers_health_checks_settings_types.go.

import (
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/config"
	healthsettings "github.com/MustardSeedNetworks/seed/internal/health/settings"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// handleHealthChecksSettings handles GET/PUT for health check settings.
func (s *Server) handleHealthChecksSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	switch r.Method {
	case http.MethodGet:
		s.getHealthChecksSettings(w, r)
	case http.MethodPut:
		s.updateHealthChecksSettings(w, r)
	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
	}
}

// buildDNSServersResponse converts config DNS servers to response format.
func buildDNSServersResponse(servers []config.DNSServer) []DNSServerResponse {
	resp := make([]DNSServerResponse, 0, len(servers))
	for _, d := range servers {
		resp = append(resp, DNSServerResponse{Address: d.Address, Enabled: d.Enabled})
	}
	return resp
}

// buildPingTargetsResponse converts ping targets to response format.
func buildPingTargetsResponse(eps []config.PingTarget) []PingTargetResponse {
	resp := make([]PingTargetResponse, 0, len(eps))
	for _, p := range eps {
		resp = append(resp, PingTargetResponse{Name: p.Name, Host: p.Host, Enabled: p.Enabled})
	}
	return resp
}

// buildTCPPortsResponse converts TCP ports to response format.
func buildTCPPortsResponse(eps []config.TCPPortTest) []TCPPortResponse {
	resp := make([]TCPPortResponse, 0, len(eps))
	for _, t := range eps {
		resp = append(resp, TCPPortResponse{Name: t.Name, Host: t.Host, Port: t.Port, Enabled: t.Enabled})
	}
	return resp
}

// buildUDPPortsResponse converts UDP ports to response format.
func buildUDPPortsResponse(eps []config.UDPPortTest) []UDPPortResponse {
	resp := make([]UDPPortResponse, 0, len(eps))
	for _, u := range eps {
		resp = append(resp, UDPPortResponse{Name: u.Name, Host: u.Host, Port: u.Port, Enabled: u.Enabled})
	}
	return resp
}

// buildHTTPEndpointsResponse converts HTTP endpoints to response format.
func buildHTTPEndpointsResponse(eps []config.HTTPEndpoint) []HTTPEndpointResponse {
	resp := make([]HTTPEndpointResponse, 0, len(eps))
	for _, h := range eps {
		resp = append(resp, HTTPEndpointResponse{
			Name:                 h.Name,
			URL:                  h.URL,
			ExpectedStatus:       h.ExpectedStatus,
			Enabled:              h.Enabled,
			BodyMatch:            h.BodyMatch,
			BodyMatchIsRegex:     h.BodyMatchIsRegex,
			CheckSecurityHeaders: h.CheckSecurityHeaders,
			FollowRedirects:      h.FollowRedirects,
			MaxRedirects:         h.MaxRedirects,
		})
	}
	return resp
}

// buildRTSPEndpointsResponse converts RTSP endpoints to response format.
func buildRTSPEndpointsResponse(eps []config.RTSPEndpoint) []RTSPEndpointResponse {
	resp := make([]RTSPEndpointResponse, 0, len(eps))
	for _, r := range eps {
		resp = append(resp, RTSPEndpointResponse{Name: r.Name, URL: r.URL, Enabled: r.Enabled})
	}
	return resp
}

// buildDICOMEndpointsResponse converts DICOM endpoints to response format.
func buildDICOMEndpointsResponse(eps []config.DICOMEndpoint) []DICOMEndpointResponse {
	resp := make([]DICOMEndpointResponse, 0, len(eps))
	for _, d := range eps {
		resp = append(resp, DICOMEndpointResponse{
			Name: d.Name, Host: d.Host, Port: d.Port,
			CalledAE: d.CalledAE, CallingAE: d.CallingAE, Enabled: d.Enabled,
		})
	}
	return resp
}

// buildHL7EndpointsResponse converts HL7 endpoints to response format.
func buildHL7EndpointsResponse(eps []config.HL7Endpoint) []HL7EndpointResponse {
	resp := make([]HL7EndpointResponse, 0, len(eps))
	for _, h := range eps {
		resp = append(resp, HL7EndpointResponse{
			Name: h.Name, Host: h.Host, Port: h.Port,
			SendingApp: h.SendingApp, SendingFac: h.SendingFac,
			ReceivingApp: h.ReceivingApp, ReceivingFac: h.ReceivingFac,
			Enabled: h.Enabled,
		})
	}
	return resp
}

// buildFHIREndpointsResponse converts FHIR endpoints to response format.
func buildFHIREndpointsResponse(eps []config.FHIREndpoint) []FHIREndpointResponse {
	resp := make([]FHIREndpointResponse, 0, len(eps))
	for _, f := range eps {
		resp = append(resp, FHIREndpointResponse{
			Name: f.Name, BaseURL: f.BaseURL, AuthType: f.AuthType,
			Enabled: f.Enabled,
		})
	}
	return resp
}

// buildSQLEndpointsResponse converts SQL endpoints to response format.
func buildSQLEndpointsResponse(eps []config.SQLEndpoint) []SQLEndpointResponse {
	resp := make([]SQLEndpointResponse, 0, len(eps))
	for _, sq := range eps {
		resp = append(resp, SQLEndpointResponse{
			Name: sq.Name, Driver: sq.Driver, Host: sq.Host, Port: sq.Port,
			Database: sq.Database, SSLMode: sq.SSLMode,
			Enabled: sq.Enabled,
		})
	}
	return resp
}

// buildFileShareEndpointsResponse converts file share endpoints to response format.
func buildFileShareEndpointsResponse(eps []config.FileShareEndpoint) []FileShareEndpointResponse {
	resp := make([]FileShareEndpointResponse, 0, len(eps))
	for _, fs := range eps {
		resp = append(resp, FileShareEndpointResponse{
			Name: fs.Name, Protocol: fs.Protocol, Host: fs.Host, Share: fs.Share, Path: fs.Path,
			TestReadPerformance: fs.TestReadPerformance, TestWritePerformance: fs.TestWritePerformance,
			TestFileSizeMB: fs.TestFileSizeMB, Enabled: fs.Enabled,
		})
	}
	return resp
}

// buildLDAPEndpointsResponse converts LDAP endpoints to response format.
func buildLDAPEndpointsResponse(eps []config.LDAPEndpoint) []LDAPEndpointResponse {
	resp := make([]LDAPEndpointResponse, 0, len(eps))
	for _, l := range eps {
		resp = append(resp, LDAPEndpointResponse{
			Name: l.Name, Host: l.Host, Port: l.Port, UseTLS: l.UseTLS, StartTLS: l.StartTLS,
			BaseDN: l.BaseDN, SearchFilter: l.SearchFilter, Enabled: l.Enabled,
		})
	}
	return resp
}

// buildLTIEndpointsResponse converts LTI endpoints to response format.
func buildLTIEndpointsResponse(eps []config.LTIEndpoint) []LTIEndpointResponse {
	resp := make([]LTIEndpointResponse, 0, len(eps))
	for _, lt := range eps {
		resp = append(resp, LTIEndpointResponse{
			Name: lt.Name, LaunchURL: lt.LaunchURL, LTIVersion: lt.LTIVersion,
			Enabled: lt.Enabled,
		})
	}
	return resp
}

// buildOPCUAEndpointsResponse converts OPC-UA endpoints to response format.
func buildOPCUAEndpointsResponse(eps []config.OPCUAEndpoint) []OPCUAEndpointResponse {
	resp := make([]OPCUAEndpointResponse, 0, len(eps))
	for _, opc := range eps {
		resp = append(resp, OPCUAEndpointResponse{
			Name: opc.Name, EndpointURL: opc.EndpointURL,
			SecurityMode: opc.SecurityMode, SecurityPolicy: opc.SecurityPolicy,
			Enabled: opc.Enabled,
		})
	}
	return resp
}

// buildModbusEndpointsResponse converts Modbus endpoints to response format.
func buildModbusEndpointsResponse(eps []config.ModbusEndpoint) []ModbusEndpointResponse {
	resp := make([]ModbusEndpointResponse, 0, len(eps))
	for _, mb := range eps {
		resp = append(resp, ModbusEndpointResponse{
			Name: mb.Name, Host: mb.Host, Port: mb.Port, UnitID: mb.UnitID,
			TestRegister: mb.TestRegister, RegisterType: mb.RegisterType,
			Enabled: mb.Enabled,
		})
	}
	return resp
}

// getHealthChecksSettings returns the check-target configuration. The
// health-check endpoint lists are sourced from the probes table (the
// store of record since ADR-0027 P2); the DNS, performance, and
// speedtest/iperf toggles remain config-file backed.
func (s *Server) getHealthChecksSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	st, err := s.healthSettings.Get(r.Context())
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to load health-check settings", "error", err)
		sendErrorResponseWithDetails(
			w, logger, http.StatusInternalServerError, ErrCodeInternal,
			i18n.FromRequest(r).T("errors.settings.loadFailed"), "",
		)
		return
	}

	hc := st.Endpoints
	resp := TestsSettingsResponse{
		DNSHostname:        st.DNSHostname,
		DNSServers:         buildDNSServersResponse(st.DNSServers),
		PingTargets:        buildPingTargetsResponse(hc.PingTargets),
		TCPPorts:           buildTCPPortsResponse(hc.TCPPorts),
		UDPPorts:           buildUDPPortsResponse(hc.UDPPorts),
		HTTPEndpoints:      buildHTTPEndpointsResponse(hc.HTTPEndpoints),
		RTSPEndpoints:      buildRTSPEndpointsResponse(hc.RTSPEndpoints),
		DICOMEndpoints:     buildDICOMEndpointsResponse(hc.DICOMEndpoints),
		HL7Endpoints:       buildHL7EndpointsResponse(hc.HL7Endpoints),
		FHIREndpoints:      buildFHIREndpointsResponse(hc.FHIREndpoints),
		SQLEndpoints:       buildSQLEndpointsResponse(hc.SQLEndpoints),
		FileShareEndpoints: buildFileShareEndpointsResponse(hc.FileShareEndpoints),
		LDAPEndpoints:      buildLDAPEndpointsResponse(hc.LDAPEndpoints),
		LTIEndpoints:       buildLTIEndpointsResponse(hc.LTIEndpoints),
		OPCUAEndpoints:     buildOPCUAEndpointsResponse(hc.OPCUAEndpoints),
		ModbusEndpoints:    buildModbusEndpointsResponse(hc.ModbusEndpoints),
		RunPerformance:     st.RunPerformance,
		RunSpeedtest:       st.RunSpeedtest,
		RunIperf:           st.RunIperf,
		RunDiscovery:       st.RunDiscovery,
		Speedtest: SpeedtestSettingsResponse{
			ServerID:      st.SpeedtestServerID,
			AutoRunOnLink: st.SpeedtestAutoRunOnLink,
		},
		Iperf: IperfSettingsResponse{
			AutoRunOnLink: st.IperfAutoRunOnLink,
		},
	}
	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// requestDNSServers maps the wire DNS server list to config form.
func requestDNSServers(req *TestsSettingsResponse) []config.DNSServer {
	servers := make([]config.DNSServer, 0, len(req.DNSServers))
	for _, d := range req.DNSServers {
		servers = append(servers, config.DNSServer{Address: d.Address, Enabled: d.Enabled})
	}
	return servers
}

// requestEndpointTargets builds a HealthChecksConfig from the request's
// endpoint lists, covering all fourteen health-check kinds. This is the
// inverse of the build*Response helpers; the resulting config is flattened
// into probe rows by healthCheckProbesFromConfig for persistence in the
// probes table (ADR-0027 P2). Only the fields the settings DTO carries are
// mapped; credential-bearing fields not yet exposed by the settings UI
// (FHIR/SQL/LDAP/OPC-UA/LTI secrets) are out of scope until the P4 frontend
// rework surfaces them.
//
//nolint:funlen // A flat fan over the fourteen endpoint lists; one block each.
func requestEndpointTargets(req *TestsSettingsResponse) config.HealthChecksConfig {
	var hc config.HealthChecksConfig

	hc.PingTargets = make([]config.PingTarget, 0, len(req.PingTargets))
	for _, p := range req.PingTargets {
		hc.PingTargets = append(hc.PingTargets,
			config.PingTarget{Name: p.Name, Host: p.Host, Enabled: p.Enabled})
	}

	hc.TCPPorts = make([]config.TCPPortTest, 0, len(req.TCPPorts))
	for _, t := range req.TCPPorts {
		hc.TCPPorts = append(hc.TCPPorts,
			config.TCPPortTest{Name: t.Name, Host: t.Host, Port: t.Port, Enabled: t.Enabled})
	}

	hc.UDPPorts = make([]config.UDPPortTest, 0, len(req.UDPPorts))
	for _, u := range req.UDPPorts {
		hc.UDPPorts = append(hc.UDPPorts,
			config.UDPPortTest{Name: u.Name, Host: u.Host, Port: u.Port, Enabled: u.Enabled})
	}

	hc.HTTPEndpoints = make([]config.HTTPEndpoint, 0, len(req.HTTPEndpoints))
	for _, h := range req.HTTPEndpoints {
		hc.HTTPEndpoints = append(hc.HTTPEndpoints, config.HTTPEndpoint{
			Name:                 h.Name,
			URL:                  h.URL,
			ExpectedStatus:       h.ExpectedStatus,
			Enabled:              h.Enabled,
			BodyMatch:            h.BodyMatch,
			BodyMatchIsRegex:     h.BodyMatchIsRegex,
			CheckSecurityHeaders: h.CheckSecurityHeaders,
			FollowRedirects:      h.FollowRedirects,
			MaxRedirects:         h.MaxRedirects,
		})
	}

	hc.RTSPEndpoints = make([]config.RTSPEndpoint, 0, len(req.RTSPEndpoints))
	for _, r := range req.RTSPEndpoints {
		hc.RTSPEndpoints = append(hc.RTSPEndpoints,
			config.RTSPEndpoint{Name: r.Name, URL: r.URL, Enabled: r.Enabled})
	}

	hc.DICOMEndpoints = make([]config.DICOMEndpoint, 0, len(req.DICOMEndpoints))
	for _, d := range req.DICOMEndpoints {
		hc.DICOMEndpoints = append(hc.DICOMEndpoints, config.DICOMEndpoint{
			Name: d.Name, Host: d.Host, Port: d.Port,
			CalledAE: d.CalledAE, CallingAE: d.CallingAE, Enabled: d.Enabled,
		})
	}

	hc.HL7Endpoints = make([]config.HL7Endpoint, 0, len(req.HL7Endpoints))
	for _, h := range req.HL7Endpoints {
		hc.HL7Endpoints = append(hc.HL7Endpoints, config.HL7Endpoint{
			Name: h.Name, Host: h.Host, Port: h.Port,
			SendingApp: h.SendingApp, SendingFac: h.SendingFac,
			ReceivingApp: h.ReceivingApp, ReceivingFac: h.ReceivingFac,
			Enabled: h.Enabled,
		})
	}

	hc.FHIREndpoints = make([]config.FHIREndpoint, 0, len(req.FHIREndpoints))
	for _, f := range req.FHIREndpoints {
		hc.FHIREndpoints = append(hc.FHIREndpoints, config.FHIREndpoint{
			Name: f.Name, BaseURL: f.BaseURL, AuthType: f.AuthType, Enabled: f.Enabled,
		})
	}

	hc.SQLEndpoints = make([]config.SQLEndpoint, 0, len(req.SQLEndpoints))
	for _, sq := range req.SQLEndpoints {
		hc.SQLEndpoints = append(hc.SQLEndpoints, config.SQLEndpoint{
			Name: sq.Name, Driver: sq.Driver, Host: sq.Host, Port: sq.Port,
			Database: sq.Database, SSLMode: sq.SSLMode, Enabled: sq.Enabled,
		})
	}

	hc.FileShareEndpoints = make([]config.FileShareEndpoint, 0, len(req.FileShareEndpoints))
	for _, fs := range req.FileShareEndpoints {
		hc.FileShareEndpoints = append(hc.FileShareEndpoints, config.FileShareEndpoint{
			Name: fs.Name, Protocol: fs.Protocol, Host: fs.Host, Share: fs.Share, Path: fs.Path,
			TestReadPerformance: fs.TestReadPerformance, TestWritePerformance: fs.TestWritePerformance,
			TestFileSizeMB: fs.TestFileSizeMB, Enabled: fs.Enabled,
		})
	}

	hc.LDAPEndpoints = make([]config.LDAPEndpoint, 0, len(req.LDAPEndpoints))
	for _, l := range req.LDAPEndpoints {
		hc.LDAPEndpoints = append(hc.LDAPEndpoints, config.LDAPEndpoint{
			Name: l.Name, Host: l.Host, Port: l.Port, UseTLS: l.UseTLS, StartTLS: l.StartTLS,
			BaseDN: l.BaseDN, SearchFilter: l.SearchFilter, Enabled: l.Enabled,
		})
	}

	hc.LTIEndpoints = make([]config.LTIEndpoint, 0, len(req.LTIEndpoints))
	for _, lt := range req.LTIEndpoints {
		hc.LTIEndpoints = append(hc.LTIEndpoints, config.LTIEndpoint{
			Name: lt.Name, LaunchURL: lt.LaunchURL, LTIVersion: lt.LTIVersion, Enabled: lt.Enabled,
		})
	}

	hc.OPCUAEndpoints = make([]config.OPCUAEndpoint, 0, len(req.OPCUAEndpoints))
	for _, opc := range req.OPCUAEndpoints {
		hc.OPCUAEndpoints = append(hc.OPCUAEndpoints, config.OPCUAEndpoint{
			Name: opc.Name, EndpointURL: opc.EndpointURL,
			SecurityMode: opc.SecurityMode, SecurityPolicy: opc.SecurityPolicy, Enabled: opc.Enabled,
		})
	}

	hc.ModbusEndpoints = make([]config.ModbusEndpoint, 0, len(req.ModbusEndpoints))
	for _, mb := range req.ModbusEndpoints {
		hc.ModbusEndpoints = append(hc.ModbusEndpoints, config.ModbusEndpoint{
			Name: mb.Name, Host: mb.Host, Port: mb.Port, UnitID: mb.UnitID,
			TestRegister: mb.TestRegister, RegisterType: mb.RegisterType, Enabled: mb.Enabled,
		})
	}

	return hc
}

// applyPerformanceSettings applies performance test configuration from request.
// updateHealthChecksSettings persists the health-checks settings. Thin transport
// (ADR-0020): decode, map the wire DTO to the domain model, and delegate the
// probe-table + config persistence + tester sync to the service.
func (s *Server) updateHealthChecksSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	var req TestsSettingsResponse
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	err := s.healthSettings.Update(r.Context(), healthsettings.Settings{
		Endpoints:              requestEndpointTargets(&req),
		DNSHostname:            req.DNSHostname,
		DNSServers:             requestDNSServers(&req),
		RunPerformance:         req.RunPerformance,
		RunSpeedtest:           req.RunSpeedtest,
		RunIperf:               req.RunIperf,
		RunDiscovery:           req.RunDiscovery,
		SpeedtestServerID:      req.Speedtest.ServerID,
		SpeedtestAutoRunOnLink: req.Speedtest.AutoRunOnLink,
		IperfAutoRunOnLink:     req.Iperf.AutoRunOnLink,
	})
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to save health-checks settings", "error", err)
		sendErrorResponseWithDetails(
			w, logger, http.StatusInternalServerError, ErrCodeInternal,
			localizer.T("errors.settings.saveFailed"), "",
		)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, statusResponse{
		Status:  statusSuccess,
		Message: "Health checks settings updated",
	})
}

// statusResponse is the JSON body returned by simple ack-style endpoints.
type statusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Common JSON status-field values reused across ack-style responses.
const (
	statusDeleted     = "deleted"
	statusSampleAdded = "sample added"
)

// Common JSON field-name keys reused across many SSE / response payloads.
const (
	jsonKeyInterface = "interface"
	jsonKeyTimestamp = "timestamp"
)
