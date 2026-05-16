package api

// handlers_health_checks_settings.go contains the GET/PUT handlers and the
// per-endpoint response builders for the health-checks settings endpoint.
// The corresponding request/response types live in
// handlers_health_checks_settings_types.go.

import (
	"encoding/json"
	"net/http"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/services/dns"
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
func (s *Server) buildDNSServersResponse() []DNSServerResponse {
	resp := make([]DNSServerResponse, 0, len(s.config.DNS.Servers))
	for _, d := range s.config.DNS.Servers {
		resp = append(resp, DNSServerResponse{Address: d.Address, Enabled: d.Enabled})
	}
	return resp
}

// buildPingTargetsResponse converts config ping targets to response format.
func (s *Server) buildPingTargetsResponse() []PingTargetResponse {
	resp := make([]PingTargetResponse, 0, len(s.config.HealthChecks.PingTargets))
	for _, p := range s.config.HealthChecks.PingTargets {
		resp = append(resp, PingTargetResponse{Name: p.Name, Host: p.Host, Enabled: p.Enabled})
	}
	return resp
}

// buildTCPPortsResponse converts config TCP ports to response format.
func (s *Server) buildTCPPortsResponse() []TCPPortResponse {
	resp := make([]TCPPortResponse, 0, len(s.config.HealthChecks.TCPPorts))
	for _, t := range s.config.HealthChecks.TCPPorts {
		resp = append(resp, TCPPortResponse{Name: t.Name, Host: t.Host, Port: t.Port, Enabled: t.Enabled})
	}
	return resp
}

// buildUDPPortsResponse converts config UDP ports to response format.
func (s *Server) buildUDPPortsResponse() []UDPPortResponse {
	resp := make([]UDPPortResponse, 0, len(s.config.HealthChecks.UDPPorts))
	for _, u := range s.config.HealthChecks.UDPPorts {
		resp = append(resp, UDPPortResponse{Name: u.Name, Host: u.Host, Port: u.Port, Enabled: u.Enabled})
	}
	return resp
}

// buildHTTPEndpointsResponse converts config HTTP endpoints to response format.
func (s *Server) buildHTTPEndpointsResponse() []HTTPEndpointResponse {
	resp := make([]HTTPEndpointResponse, 0, len(s.config.HealthChecks.HTTPEndpoints))
	for _, h := range s.config.HealthChecks.HTTPEndpoints {
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

// buildRTSPEndpointsResponse converts config RTSP endpoints to response format.
func (s *Server) buildRTSPEndpointsResponse() []RTSPEndpointResponse {
	resp := make([]RTSPEndpointResponse, 0, len(s.config.HealthChecks.RTSPEndpoints))
	for _, r := range s.config.HealthChecks.RTSPEndpoints {
		resp = append(resp, RTSPEndpointResponse{Name: r.Name, URL: r.URL, Enabled: r.Enabled})
	}
	return resp
}

// buildDICOMEndpointsResponse converts config DICOM endpoints to response format.
func (s *Server) buildDICOMEndpointsResponse() []DICOMEndpointResponse {
	resp := make([]DICOMEndpointResponse, 0, len(s.config.HealthChecks.DICOMEndpoints))
	for _, d := range s.config.HealthChecks.DICOMEndpoints {
		resp = append(resp, DICOMEndpointResponse{
			Name: d.Name, Host: d.Host, Port: d.Port,
			CalledAE: d.CalledAE, CallingAE: d.CallingAE, Enabled: d.Enabled,
		})
	}
	return resp
}

// buildHL7EndpointsResponse converts config HL7 endpoints to response format.
func (s *Server) buildHL7EndpointsResponse() []HL7EndpointResponse {
	resp := make([]HL7EndpointResponse, 0, len(s.config.HealthChecks.HL7Endpoints))
	for _, h := range s.config.HealthChecks.HL7Endpoints {
		resp = append(resp, HL7EndpointResponse{
			Name: h.Name, Host: h.Host, Port: h.Port,
			SendingApp: h.SendingApp, SendingFac: h.SendingFac,
			ReceivingApp: h.ReceivingApp, ReceivingFac: h.ReceivingFac,
			Enabled: h.Enabled, Criticality: h.Criticality,
		})
	}
	return resp
}

// buildFHIREndpointsResponse converts config FHIR endpoints to response format.
func (s *Server) buildFHIREndpointsResponse() []FHIREndpointResponse {
	resp := make([]FHIREndpointResponse, 0, len(s.config.HealthChecks.FHIREndpoints))
	for _, f := range s.config.HealthChecks.FHIREndpoints {
		resp = append(resp, FHIREndpointResponse{
			Name: f.Name, BaseURL: f.BaseURL, AuthType: f.AuthType,
			Enabled: f.Enabled, Criticality: f.Criticality,
		})
	}
	return resp
}

// buildSQLEndpointsResponse converts config SQL endpoints to response format.
func (s *Server) buildSQLEndpointsResponse() []SQLEndpointResponse {
	resp := make([]SQLEndpointResponse, 0, len(s.config.HealthChecks.SQLEndpoints))
	for _, sq := range s.config.HealthChecks.SQLEndpoints {
		resp = append(resp, SQLEndpointResponse{
			Name: sq.Name, Driver: sq.Driver, Host: sq.Host, Port: sq.Port,
			Database: sq.Database, SSLMode: sq.SSLMode,
			Enabled: sq.Enabled, Criticality: sq.Criticality,
		})
	}
	return resp
}

// buildFileShareEndpointsResponse converts config file share endpoints to response format.
func (s *Server) buildFileShareEndpointsResponse() []FileShareEndpointResponse {
	resp := make([]FileShareEndpointResponse, 0, len(s.config.HealthChecks.FileShareEndpoints))
	for _, fs := range s.config.HealthChecks.FileShareEndpoints {
		resp = append(resp, FileShareEndpointResponse{
			Name: fs.Name, Protocol: fs.Protocol, Host: fs.Host, Share: fs.Share, Path: fs.Path,
			TestReadPerformance: fs.TestReadPerformance, TestWritePerformance: fs.TestWritePerformance,
			TestFileSizeMB: fs.TestFileSizeMB, Enabled: fs.Enabled, Criticality: fs.Criticality,
		})
	}
	return resp
}

// buildLDAPEndpointsResponse converts config LDAP endpoints to response format.
func (s *Server) buildLDAPEndpointsResponse() []LDAPEndpointResponse {
	resp := make([]LDAPEndpointResponse, 0, len(s.config.HealthChecks.LDAPEndpoints))
	for _, l := range s.config.HealthChecks.LDAPEndpoints {
		resp = append(resp, LDAPEndpointResponse{
			Name: l.Name, Host: l.Host, Port: l.Port, UseTLS: l.UseTLS, StartTLS: l.StartTLS,
			BaseDN: l.BaseDN, SearchFilter: l.SearchFilter, Enabled: l.Enabled, Criticality: l.Criticality,
		})
	}
	return resp
}

// buildLTIEndpointsResponse converts config LTI endpoints to response format.
func (s *Server) buildLTIEndpointsResponse() []LTIEndpointResponse {
	resp := make([]LTIEndpointResponse, 0, len(s.config.HealthChecks.LTIEndpoints))
	for _, lt := range s.config.HealthChecks.LTIEndpoints {
		resp = append(resp, LTIEndpointResponse{
			Name: lt.Name, LaunchURL: lt.LaunchURL, LTIVersion: lt.LTIVersion,
			Enabled: lt.Enabled, Criticality: lt.Criticality,
		})
	}
	return resp
}

// buildOPCUAEndpointsResponse converts config OPC-UA endpoints to response format.
func (s *Server) buildOPCUAEndpointsResponse() []OPCUAEndpointResponse {
	resp := make([]OPCUAEndpointResponse, 0, len(s.config.HealthChecks.OPCUAEndpoints))
	for _, opc := range s.config.HealthChecks.OPCUAEndpoints {
		resp = append(resp, OPCUAEndpointResponse{
			Name: opc.Name, EndpointURL: opc.EndpointURL,
			SecurityMode: opc.SecurityMode, SecurityPolicy: opc.SecurityPolicy,
			Enabled: opc.Enabled, Criticality: opc.Criticality,
		})
	}
	return resp
}

// buildModbusEndpointsResponse converts config Modbus endpoints to response format.
func (s *Server) buildModbusEndpointsResponse() []ModbusEndpointResponse {
	resp := make([]ModbusEndpointResponse, 0, len(s.config.HealthChecks.ModbusEndpoints))
	for _, mb := range s.config.HealthChecks.ModbusEndpoints {
		resp = append(resp, ModbusEndpointResponse{
			Name: mb.Name, Host: mb.Host, Port: mb.Port, UnitID: mb.UnitID,
			TestRegister: mb.TestRegister, RegisterType: mb.RegisterType,
			Enabled: mb.Enabled, Criticality: mb.Criticality,
		})
	}
	return resp
}

func (s *Server) getHealthChecksSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	resp := TestsSettingsResponse{
		DNSHostname:        s.config.DNS.TestHostname,
		DNSServers:         s.buildDNSServersResponse(),
		PingTargets:        s.buildPingTargetsResponse(),
		TCPPorts:           s.buildTCPPortsResponse(),
		UDPPorts:           s.buildUDPPortsResponse(),
		HTTPEndpoints:      s.buildHTTPEndpointsResponse(),
		RTSPEndpoints:      s.buildRTSPEndpointsResponse(),
		DICOMEndpoints:     s.buildDICOMEndpointsResponse(),
		HL7Endpoints:       s.buildHL7EndpointsResponse(),
		FHIREndpoints:      s.buildFHIREndpointsResponse(),
		SQLEndpoints:       s.buildSQLEndpointsResponse(),
		FileShareEndpoints: s.buildFileShareEndpointsResponse(),
		LDAPEndpoints:      s.buildLDAPEndpointsResponse(),
		LTIEndpoints:       s.buildLTIEndpointsResponse(),
		OPCUAEndpoints:     s.buildOPCUAEndpointsResponse(),
		ModbusEndpoints:    s.buildModbusEndpointsResponse(),
		RunPerformance:     s.config.HealthChecks.RunPerformance,
		RunSpeedtest:       s.config.HealthChecks.RunSpeedtest,
		RunIperf:           s.config.HealthChecks.RunIperf,
		RunDiscovery:       s.config.HealthChecks.RunDiscovery,
		Speedtest: SpeedtestSettingsResponse{
			ServerID:      s.config.Speedtest.ServerID,
			AutoRunOnLink: s.config.Speedtest.AutoRunOnLink,
		},
		Iperf: IperfSettingsResponse{
			AutoRunOnLink: s.config.Iperf.AutoRunOnLink,
		},
	}
	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// applyDNSSettings applies DNS configuration from request.
func (s *Server) applyDNSSettings(req *TestsSettingsResponse) {
	if req.DNSHostname != "" {
		s.config.DNS.TestHostname = req.DNSHostname
		if s.dnsTester() != nil {
			s.dnsTester().SetTestHostname(req.DNSHostname)
		}
	}

	s.config.DNS.Servers = make([]config.DNSServer, 0, len(req.DNSServers))
	for _, d := range req.DNSServers {
		s.config.DNS.Servers = append(
			s.config.DNS.Servers,
			config.DNSServer{Address: d.Address, Enabled: d.Enabled},
		)
	}
	if s.dnsTester() != nil {
		configuredServers := make([]dns.ConfiguredServer, 0, len(s.config.DNS.Servers))
		for _, d := range s.config.DNS.Servers {
			configuredServers = append(
				configuredServers,
				dns.ConfiguredServer{Address: d.Address, Enabled: d.Enabled},
			)
		}
		s.dnsTester().SetConfiguredServers(configuredServers)
	}
}

// applyTestTargets applies test target configuration from request.
func (s *Server) applyTestTargets(req *TestsSettingsResponse) {
	s.config.HealthChecks.PingTargets = make([]config.PingTarget, 0, len(req.PingTargets))
	for _, p := range req.PingTargets {
		s.config.HealthChecks.PingTargets = append(
			s.config.HealthChecks.PingTargets,
			config.PingTarget{Name: p.Name, Host: p.Host, Enabled: p.Enabled},
		)
	}

	s.config.HealthChecks.TCPPorts = make([]config.TCPPortTest, 0, len(req.TCPPorts))
	for _, t := range req.TCPPorts {
		s.config.HealthChecks.TCPPorts = append(
			s.config.HealthChecks.TCPPorts,
			config.TCPPortTest{Name: t.Name, Host: t.Host, Port: t.Port, Enabled: t.Enabled},
		)
	}

	s.config.HealthChecks.UDPPorts = make([]config.UDPPortTest, 0, len(req.UDPPorts))
	for _, u := range req.UDPPorts {
		s.config.HealthChecks.UDPPorts = append(
			s.config.HealthChecks.UDPPorts,
			config.UDPPortTest{Name: u.Name, Host: u.Host, Port: u.Port, Enabled: u.Enabled},
		)
	}

	s.config.HealthChecks.HTTPEndpoints = make([]config.HTTPEndpoint, 0, len(req.HTTPEndpoints))
	for _, h := range req.HTTPEndpoints {
		s.config.HealthChecks.HTTPEndpoints = append(
			s.config.HealthChecks.HTTPEndpoints,
			config.HTTPEndpoint{
				Name:                 h.Name,
				URL:                  h.URL,
				ExpectedStatus:       h.ExpectedStatus,
				Enabled:              h.Enabled,
				BodyMatch:            h.BodyMatch,
				BodyMatchIsRegex:     h.BodyMatchIsRegex,
				CheckSecurityHeaders: h.CheckSecurityHeaders,
				FollowRedirects:      h.FollowRedirects,
				MaxRedirects:         h.MaxRedirects,
			},
		)
	}

	// RTSP endpoints (Issue #778)
	s.config.HealthChecks.RTSPEndpoints = make([]config.RTSPEndpoint, 0, len(req.RTSPEndpoints))
	for _, r := range req.RTSPEndpoints {
		s.config.HealthChecks.RTSPEndpoints = append(
			s.config.HealthChecks.RTSPEndpoints,
			config.RTSPEndpoint{Name: r.Name, URL: r.URL, Enabled: r.Enabled},
		)
	}

	// DICOM endpoints (Issue #777)
	s.config.HealthChecks.DICOMEndpoints = make([]config.DICOMEndpoint, 0, len(req.DICOMEndpoints))
	for _, d := range req.DICOMEndpoints {
		s.config.HealthChecks.DICOMEndpoints = append(
			s.config.HealthChecks.DICOMEndpoints,
			config.DICOMEndpoint{
				Name: d.Name, Host: d.Host, Port: d.Port,
				CalledAE: d.CalledAE, CallingAE: d.CallingAE, Enabled: d.Enabled,
			},
		)
	}
}

// applyPerformanceSettings applies performance test configuration from request.
func (s *Server) applyPerformanceSettings(req *TestsSettingsResponse) {
	s.config.HealthChecks.RunPerformance = req.RunPerformance
	s.config.HealthChecks.RunSpeedtest = req.RunSpeedtest
	s.config.HealthChecks.RunIperf = req.RunIperf
	s.config.HealthChecks.RunDiscovery = req.RunDiscovery

	s.config.Speedtest.ServerID = req.Speedtest.ServerID
	s.config.Speedtest.AutoRunOnLink = req.Speedtest.AutoRunOnLink
	if s.speedtestTester() != nil {
		s.speedtestTester().SetServerID(req.Speedtest.ServerID)
	}

	s.config.Iperf.AutoRunOnLink = req.Iperf.AutoRunOnLink
}

func (s *Server) updateHealthChecksSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req TestsSettingsResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WarnContext(r.Context(), "Invalid request body", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusBadRequest,
			ErrCodeBadRequest,
			localizer.T("errors.api.invalidRequestBody"),
			"",
		)
		return
	}

	// Lock config for write access (unlock before Save to avoid deadlock)
	s.config.Lock()
	s.applyDNSSettings(&req)
	s.applyTestTargets(&req)
	s.applyPerformanceSettings(&req)
	s.config.Unlock()

	if err := s.config.Save(s.configPath); err != nil {
		logger.ErrorContext(r.Context(), "Failed to save config", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			localizer.T("errors.settings.saveFailed"),
			"",
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
