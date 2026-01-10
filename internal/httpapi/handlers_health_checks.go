package httpapi

// handlers_health_checks.go contains core health check testing handlers.
// DNS, Speedtest, and iPerf handlers are split into separate files (Plan F).

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/http/httptrace"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/sap/dns"
	"github.com/krisarmstrong/seed/internal/validation"
)

// Test status and protocol constants.
const (
	statusError   = "error"
	statusWarning = "warning"
	statusSuccess = "success"
	protoTCP      = "tcp"
	protoUDP      = "udp"
)

// Health check timing and configuration constants.
const (
	// defaultPingCount is the number of ping probes per target for extended ping tests.
	defaultPingCount = 5

	// pingProbeTimeoutSec is the timeout in seconds for each individual ping probe.
	pingProbeTimeoutSec = 2

	// pingProbeDelayMs is the delay in milliseconds between consecutive ping probes.
	pingProbeDelayMs = 100

	// tcpTestTimeoutSec is the timeout in seconds for TCP connectivity tests.
	tcpTestTimeoutSec = 5

	// udpTestTimeoutSec is the timeout in seconds for UDP connectivity tests.
	udpTestTimeoutSec = 5

	// udpReadDeadlineSec is the deadline in seconds for reading UDP responses.
	udpReadDeadlineSec = 3

	// udpReadBufferBytes is the buffer size in bytes for reading UDP responses.
	udpReadBufferBytes = 1024

	// certCheckTimeoutSec is the timeout in seconds for TLS certificate checks.
	certCheckTimeoutSec = 5

	// hoursPerDay is the number of hours in a day for certificate expiry calculations.
	hoursPerDay = 24

	// percentageDivisor converts ratios to percentages (multiply by 100).
	percentageDivisor = 100

	// packetLossThresholdFull indicates complete packet loss (100%).
	packetLossThresholdFull = 100

	// packetLossThresholdHigh indicates severe packet loss threshold (50%).
	packetLossThresholdHigh = 50

	// packetLossThresholdLow indicates elevated packet loss threshold (10%).
	packetLossThresholdLow = 10

	// millisecondsPerSecond is the conversion factor from seconds to milliseconds.
	millisecondsPerSecond = 1000

	// dnsPort is the standard DNS service port.
	dnsPort = 53

	// httpClientTimeoutSec is the timeout in seconds for HTTP client requests.
	httpClientTimeoutSec = 10
)

// ============================================================================
// Health Checks Settings Types
// ============================================================================

// TestsSettingsResponse represents the custom tests configuration.
type TestsSettingsResponse struct {
	DNSHostname        string                      `json:"dnsHostname"`
	DNSServers         []DNSServerResponse         `json:"dnsServers"`
	PingTargets        []PingTargetResponse        `json:"pingTargets"`
	TCPPorts           []TCPPortResponse           `json:"tcpPorts"`
	UDPPorts           []UDPPortResponse           `json:"udpPorts"`
	HTTPEndpoints      []HTTPEndpointResponse      `json:"httpEndpoints"`
	RTSPEndpoints      []RTSPEndpointResponse      `json:"rtspEndpoints"`      // Issue #778
	DICOMEndpoints     []DICOMEndpointResponse     `json:"dicomEndpoints"`     // Issue #777
	HL7Endpoints       []HL7EndpointResponse       `json:"hl7Endpoints"`       // Health Checks 100x - Medical
	FHIREndpoints      []FHIREndpointResponse      `json:"fhirEndpoints"`      // Health Checks 100x - Medical
	SQLEndpoints       []SQLEndpointResponse       `json:"sqlEndpoints"`       // Health Checks 100x - Enterprise
	FileShareEndpoints []FileShareEndpointResponse `json:"fileShareEndpoints"` // Health Checks 100x - Enterprise
	LDAPEndpoints      []LDAPEndpointResponse      `json:"ldapEndpoints"`      // Health Checks 100x - Enterprise
	LTIEndpoints       []LTIEndpointResponse       `json:"ltiEndpoints"`       // Health Checks 100x - Education
	OPCUAEndpoints     []OPCUAEndpointResponse     `json:"opcuaEndpoints"`     // Health Checks 100x - Manufacturing
	ModbusEndpoints    []ModbusEndpointResponse    `json:"modbusEndpoints"`    // Health Checks 100x - Manufacturing
	Speedtest          SpeedtestSettingsResponse   `json:"speedtest"`
	Iperf              IperfSettingsResponse       `json:"iperf"`
	RunPerformance     bool                        `json:"runPerformance"`
	RunSpeedtest       bool                        `json:"runSpeedtest"`
	RunIperf           bool                        `json:"runIperf"`
	RunDiscovery       bool                        `json:"runDiscovery"`
}

// DNSServerResponse contains a DNS server address and its enabled state.
type DNSServerResponse struct {
	Address string `json:"address"`
	Enabled bool   `json:"enabled"`
}

// PingTargetResponse contains a ping target configuration with name and host.
type PingTargetResponse struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Enabled bool   `json:"enabled"`
}

// TCPPortResponse contains a TCP port test configuration with host and port.
type TCPPortResponse struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

// UDPPortResponse contains a UDP port test configuration with host and port.
type UDPPortResponse struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

// HTTPEndpointResponse contains an HTTP endpoint test configuration.
type HTTPEndpointResponse struct {
	Name                 string `json:"name"`
	URL                  string `json:"url"`
	ExpectedStatus       int    `json:"expectedStatus"`
	Enabled              bool   `json:"enabled"`
	BodyMatch            string `json:"bodyMatch,omitempty"`
	BodyMatchIsRegex     bool   `json:"bodyMatchIsRegex,omitempty"`
	CheckSecurityHeaders bool   `json:"checkSecurityHeaders,omitempty"`
	FollowRedirects      bool   `json:"followRedirects,omitempty"`
	MaxRedirects         int    `json:"maxRedirects,omitempty"`
}

// RTSPEndpointResponse contains an RTSP stream test configuration (Issue #778).
type RTSPEndpointResponse struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

// DICOMEndpointResponse contains a DICOM server test configuration (Issue #777).
type DICOMEndpointResponse struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	CalledAE  string `json:"calledAe"`
	CallingAE string `json:"callingAe"`
	Enabled   bool   `json:"enabled"`
}

// HL7EndpointResponse contains an HL7 MLLP endpoint configuration (Health Checks 100x).
type HL7EndpointResponse struct {
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	SendingApp   string `json:"sendingApp"`
	SendingFac   string `json:"sendingFacility"`
	ReceivingApp string `json:"receivingApp"`
	ReceivingFac string `json:"receivingFacility"`
	Enabled      bool   `json:"enabled"`
	Criticality  int    `json:"criticality"`
}

// FHIREndpointResponse contains a FHIR R4 endpoint configuration (Health Checks 100x).
type FHIREndpointResponse struct {
	Name        string `json:"name"`
	BaseURL     string `json:"baseUrl"`
	AuthType    string `json:"authType"`
	Enabled     bool   `json:"enabled"`
	Criticality int    `json:"criticality"`
}

// SQLEndpointResponse contains a SQL database endpoint configuration (Health Checks 100x).
type SQLEndpointResponse struct {
	Name        string `json:"name"`
	Driver      string `json:"driver"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Database    string `json:"database"`
	SSLMode     string `json:"sslMode,omitempty"`
	Enabled     bool   `json:"enabled"`
	Criticality int    `json:"criticality"`
}

// FileShareEndpointResponse contains a file share endpoint configuration (Health Checks 100x).
type FileShareEndpointResponse struct {
	Name                 string `json:"name"`
	Protocol             string `json:"protocol"`
	Host                 string `json:"host"`
	Share                string `json:"share"`
	Path                 string `json:"path,omitempty"`
	TestReadPerformance  bool   `json:"testReadPerformance,omitempty"`
	TestWritePerformance bool   `json:"testWritePerformance,omitempty"`
	TestFileSizeMB       int    `json:"testFileSizeMb,omitempty"`
	Enabled              bool   `json:"enabled"`
	Criticality          int    `json:"criticality"`
}

// LDAPEndpointResponse contains an LDAP/AD endpoint configuration (Health Checks 100x).
type LDAPEndpointResponse struct {
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	UseTLS       bool   `json:"useTls"`
	StartTLS     bool   `json:"startTls"`
	BaseDN       string `json:"baseDn"`
	SearchFilter string `json:"searchFilter,omitempty"`
	Enabled      bool   `json:"enabled"`
	Criticality  int    `json:"criticality"`
}

// LTIEndpointResponse contains an LTI/LMS endpoint configuration (Health Checks 100x - Education).
type LTIEndpointResponse struct {
	Name        string `json:"name"`
	LaunchURL   string `json:"launchUrl"`
	LTIVersion  string `json:"ltiVersion,omitempty"`
	Enabled     bool   `json:"enabled"`
	Criticality int    `json:"criticality"`
}

// OPCUAEndpointResponse contains an OPC-UA endpoint configuration (Health Checks 100x - Manufacturing).
type OPCUAEndpointResponse struct {
	Name           string `json:"name"`
	EndpointURL    string `json:"endpointUrl"`
	SecurityMode   string `json:"securityMode,omitempty"`
	SecurityPolicy string `json:"securityPolicy,omitempty"`
	Enabled        bool   `json:"enabled"`
	Criticality    int    `json:"criticality"`
}

// ModbusEndpointResponse contains a Modbus TCP endpoint configuration (Health Checks 100x - Manufacturing).
type ModbusEndpointResponse struct {
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	UnitID       int    `json:"unitId"`
	TestRegister int    `json:"testRegister"`
	RegisterType string `json:"registerType,omitempty"`
	Enabled      bool   `json:"enabled"`
	Criticality  int    `json:"criticality"`
}

// SpeedtestSettingsResponse contains speedtest configuration options.
type SpeedtestSettingsResponse struct {
	ServerID      string `json:"serverId"`
	AutoRunOnLink bool   `json:"autoRunOnLink"`
}

// IperfSettingsResponse contains iPerf3 configuration options.
type IperfSettingsResponse struct {
	AutoRunOnLink bool `json:"autoRunOnLink"`
}

// ============================================================================
// Health Checks Settings Handlers
// ============================================================================

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

func (s *Server) getHealthChecksSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	resp := TestsSettingsResponse{
		DNSHostname:        s.config.DNS.TestHostname,
		DNSServers:         make([]DNSServerResponse, 0, len(s.config.DNS.Servers)),
		PingTargets:        make([]PingTargetResponse, 0, len(s.config.HealthChecks.PingTargets)),
		TCPPorts:           make([]TCPPortResponse, 0, len(s.config.HealthChecks.TCPPorts)),
		UDPPorts:           make([]UDPPortResponse, 0, len(s.config.HealthChecks.UDPPorts)),
		HTTPEndpoints:      make([]HTTPEndpointResponse, 0, len(s.config.HealthChecks.HTTPEndpoints)),
		RTSPEndpoints:      make([]RTSPEndpointResponse, 0, len(s.config.HealthChecks.RTSPEndpoints)),       // Issue #778
		DICOMEndpoints:     make([]DICOMEndpointResponse, 0, len(s.config.HealthChecks.DICOMEndpoints)),     // Issue #777
		HL7Endpoints:       make([]HL7EndpointResponse, 0, len(s.config.HealthChecks.HL7Endpoints)),         // Health Checks 100x - Medical
		FHIREndpoints:      make([]FHIREndpointResponse, 0, len(s.config.HealthChecks.FHIREndpoints)),       // Health Checks 100x - Medical
		SQLEndpoints:       make([]SQLEndpointResponse, 0, len(s.config.HealthChecks.SQLEndpoints)),         // Health Checks 100x - Enterprise
		FileShareEndpoints: make([]FileShareEndpointResponse, 0, len(s.config.HealthChecks.FileShareEndpoints)), // Health Checks 100x - Enterprise
		LDAPEndpoints:      make([]LDAPEndpointResponse, 0, len(s.config.HealthChecks.LDAPEndpoints)),       // Health Checks 100x - Enterprise
		LTIEndpoints:       make([]LTIEndpointResponse, 0, len(s.config.HealthChecks.LTIEndpoints)),         // Health Checks 100x - Education
		OPCUAEndpoints:     make([]OPCUAEndpointResponse, 0, len(s.config.HealthChecks.OPCUAEndpoints)),     // Health Checks 100x - Manufacturing
		ModbusEndpoints:    make([]ModbusEndpointResponse, 0, len(s.config.HealthChecks.ModbusEndpoints)),   // Health Checks 100x - Manufacturing
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

	// DNS servers
	for _, d := range s.config.DNS.Servers {
		resp.DNSServers = append(resp.DNSServers, DNSServerResponse{
			Address: d.Address,
			Enabled: d.Enabled,
		})
	}

	for _, p := range s.config.HealthChecks.PingTargets {
		resp.PingTargets = append(resp.PingTargets, PingTargetResponse{
			Name:    p.Name,
			Host:    p.Host,
			Enabled: p.Enabled,
		})
	}

	for _, t := range s.config.HealthChecks.TCPPorts {
		resp.TCPPorts = append(resp.TCPPorts, TCPPortResponse{
			Name:    t.Name,
			Host:    t.Host,
			Port:    t.Port,
			Enabled: t.Enabled,
		})
	}

	for _, u := range s.config.HealthChecks.UDPPorts {
		resp.UDPPorts = append(resp.UDPPorts, UDPPortResponse{
			Name:    u.Name,
			Host:    u.Host,
			Port:    u.Port,
			Enabled: u.Enabled,
		})
	}

	for _, h := range s.config.HealthChecks.HTTPEndpoints {
		resp.HTTPEndpoints = append(resp.HTTPEndpoints, HTTPEndpointResponse{
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

	// RTSP endpoints (Issue #778)
	for _, r := range s.config.HealthChecks.RTSPEndpoints {
		resp.RTSPEndpoints = append(resp.RTSPEndpoints, RTSPEndpointResponse{
			Name:    r.Name,
			URL:     r.URL,
			Enabled: r.Enabled,
		})
	}

	// DICOM endpoints (Issue #777)
	for _, d := range s.config.HealthChecks.DICOMEndpoints {
		resp.DICOMEndpoints = append(resp.DICOMEndpoints, DICOMEndpointResponse{
			Name:      d.Name,
			Host:      d.Host,
			Port:      d.Port,
			CalledAE:  d.CalledAE,
			CallingAE: d.CallingAE,
			Enabled:   d.Enabled,
		})
	}

	// HL7 endpoints (Health Checks 100x)
	for _, h := range s.config.HealthChecks.HL7Endpoints {
		resp.HL7Endpoints = append(resp.HL7Endpoints, HL7EndpointResponse{
			Name:         h.Name,
			Host:         h.Host,
			Port:         h.Port,
			SendingApp:   h.SendingApp,
			SendingFac:   h.SendingFac,
			ReceivingApp: h.ReceivingApp,
			ReceivingFac: h.ReceivingFac,
			Enabled:      h.Enabled,
			Criticality:  h.Criticality,
		})
	}

	// FHIR endpoints (Health Checks 100x - Medical)
	for _, f := range s.config.HealthChecks.FHIREndpoints {
		resp.FHIREndpoints = append(resp.FHIREndpoints, FHIREndpointResponse{
			Name:        f.Name,
			BaseURL:     f.BaseURL,
			AuthType:    f.AuthType,
			Enabled:     f.Enabled,
			Criticality: f.Criticality,
		})
	}

	// SQL endpoints (Health Checks 100x - Enterprise)
	for _, sq := range s.config.HealthChecks.SQLEndpoints {
		resp.SQLEndpoints = append(resp.SQLEndpoints, SQLEndpointResponse{
			Name:        sq.Name,
			Driver:      sq.Driver,
			Host:        sq.Host,
			Port:        sq.Port,
			Database:    sq.Database,
			SSLMode:     sq.SSLMode,
			Enabled:     sq.Enabled,
			Criticality: sq.Criticality,
		})
	}

	// FileShare endpoints (Health Checks 100x - Enterprise)
	for _, fs := range s.config.HealthChecks.FileShareEndpoints {
		resp.FileShareEndpoints = append(resp.FileShareEndpoints, FileShareEndpointResponse{
			Name:                 fs.Name,
			Protocol:             fs.Protocol,
			Host:                 fs.Host,
			Share:                fs.Share,
			Path:                 fs.Path,
			TestReadPerformance:  fs.TestReadPerformance,
			TestWritePerformance: fs.TestWritePerformance,
			TestFileSizeMB:       fs.TestFileSizeMB,
			Enabled:              fs.Enabled,
			Criticality:          fs.Criticality,
		})
	}

	// LDAP endpoints (Health Checks 100x - Enterprise)
	for _, l := range s.config.HealthChecks.LDAPEndpoints {
		resp.LDAPEndpoints = append(resp.LDAPEndpoints, LDAPEndpointResponse{
			Name:         l.Name,
			Host:         l.Host,
			Port:         l.Port,
			UseTLS:       l.UseTLS,
			StartTLS:     l.StartTLS,
			BaseDN:       l.BaseDN,
			SearchFilter: l.SearchFilter,
			Enabled:      l.Enabled,
			Criticality:  l.Criticality,
		})
	}

	// LTI endpoints (Health Checks 100x - Education)
	for _, lt := range s.config.HealthChecks.LTIEndpoints {
		resp.LTIEndpoints = append(resp.LTIEndpoints, LTIEndpointResponse{
			Name:        lt.Name,
			LaunchURL:   lt.LaunchURL,
			LTIVersion:  lt.LTIVersion,
			Enabled:     lt.Enabled,
			Criticality: lt.Criticality,
		})
	}

	// OPC-UA endpoints (Health Checks 100x - Manufacturing)
	for _, opc := range s.config.HealthChecks.OPCUAEndpoints {
		resp.OPCUAEndpoints = append(resp.OPCUAEndpoints, OPCUAEndpointResponse{
			Name:           opc.Name,
			EndpointURL:    opc.EndpointURL,
			SecurityMode:   opc.SecurityMode,
			SecurityPolicy: opc.SecurityPolicy,
			Enabled:        opc.Enabled,
			Criticality:    opc.Criticality,
		})
	}

	// Modbus endpoints (Health Checks 100x - Manufacturing)
	for _, mb := range s.config.HealthChecks.ModbusEndpoints {
		resp.ModbusEndpoints = append(resp.ModbusEndpoints, ModbusEndpointResponse{
			Name:         mb.Name,
			Host:         mb.Host,
			Port:         mb.Port,
			UnitID:       mb.UnitID,
			TestRegister: mb.TestRegister,
			RegisterType: mb.RegisterType,
			Enabled:      mb.Enabled,
			Criticality:  mb.Criticality,
		})
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

	sendJSONResponse(
		w,
		logger,
		http.StatusOK,
		map[string]string{"status": "success", "message": "Health checks settings updated"},
	)
}

// ============================================================================
// Health Checks Test Types
// ============================================================================

// CustomTestResult represents the result of a single custom test.
type CustomTestResult struct {
	Name        string  `json:"name"`
	Host        string  `json:"host"`
	Port        int     `json:"port,omitempty"`
	URL         string  `json:"url,omitempty"`
	Success     bool    `json:"success"`
	Latency     float64 `json:"latency"` // ms
	DNSLatency  float64 `json:"dnsLatency,omitempty"`
	TCPConnect  float64 `json:"tcpConnect,omitempty"`
	TLSLatency  float64 `json:"tlsLatency,omitempty"`
	TTFBLatency float64 `json:"ttfbLatency,omitempty"` // Time to first byte (server processing + wait)
	Error       string  `json:"error,omitempty"`
	Status      int     `json:"status,omitempty"`     // HTTP status code
	TestStatus  string  `json:"testStatus,omitempty"` // success, warning, error
	// Per-phase status fields for HTTP timing breakdown
	DNSStatus  string `json:"dnsStatus,omitempty"`  // success, warning, error
	TCPStatus  string `json:"tcpStatus,omitempty"`  // success, warning, error
	TLSStatus  string `json:"tlsStatus,omitempty"`  // success, warning, error
	TTFBStatus string `json:"ttfbStatus,omitempty"` // success, warning, error
	// Extended ping fields
	PacketLoss float64 `json:"packetLoss,omitempty"` // Percentage
	Jitter     float64 `json:"jitter,omitempty"`     // ms
	MinLatency float64 `json:"minLatency,omitempty"` // ms
	MaxLatency float64 `json:"maxLatency,omitempty"` // ms
	// Certificate fields
	CertDaysLeft   int    `json:"certDaysLeft,omitempty"`   // Days until cert expires
	CertStatus     string `json:"certStatus,omitempty"`     // success, warning, error
	CertExpiry     string `json:"certExpiry,omitempty"`     // Expiry date string
	CertCommonName string `json:"certCommonName,omitempty"` // Certificate CN
	TLSVersion     string `json:"tlsVersion,omitempty"`     // TLS 1.2, TLS 1.3, etc.
	CertIssuer     string `json:"certIssuer,omitempty"`     // Certificate issuer
	// HTTP enhancements (Health Checks 100x)
	BodyMatchSuccess bool             `json:"bodyMatchSuccess,omitempty"` // True if body matched pattern
	BodyMatchStatus  string           `json:"bodyMatchStatus,omitempty"`  // success, error
	ResponseSize     int64            `json:"responseSize,omitempty"`     // Response body size in bytes
	HTTPVersion      string           `json:"httpVersion,omitempty"`      // HTTP/1.1, HTTP/2, HTTP/3
	SecurityHeaders  *SecurityHeaders `json:"securityHeaders,omitempty"`  // Security headers check results
	RedirectChain    []RedirectHop    `json:"redirectChain,omitempty"`    // Redirect chain details
}

// SecurityHeaders contains results of security header checks.
type SecurityHeaders struct {
	HSTS              *HeaderCheck `json:"hsts,omitempty"`              // Strict-Transport-Security
	CSP               *HeaderCheck `json:"csp,omitempty"`               // Content-Security-Policy
	XFrameOptions     *HeaderCheck `json:"xFrameOptions,omitempty"`     // X-Frame-Options
	XContentType      *HeaderCheck `json:"xContentType,omitempty"`      // X-Content-Type-Options
	XSSProtection     *HeaderCheck `json:"xssProtection,omitempty"`     // X-XSS-Protection
	ReferrerPolicy    *HeaderCheck `json:"referrerPolicy,omitempty"`    // Referrer-Policy
	PermissionsPolicy *HeaderCheck `json:"permissionsPolicy,omitempty"` // Permissions-Policy
	OverallStatus     string       `json:"overallStatus"`               // success, warning, error
	Score             int          `json:"score"`                       // 0-100 security score
}

// HeaderCheck represents the check result for a single security header.
type HeaderCheck struct {
	Present bool   `json:"present"`           // Whether header is present
	Value   string `json:"value,omitempty"`   // Header value if present
	Status  string `json:"status"`            // success, warning, error
	Message string `json:"message,omitempty"` // Recommendation/warning message
}

// RedirectHop represents a single hop in a redirect chain.
type RedirectHop struct {
	URL        string  `json:"url"`
	StatusCode int     `json:"statusCode"`
	LatencyMs  float64 `json:"latencyMs"` // Time taken for this hop
}

// CustomTestsResult represents results from all custom tests.
type CustomTestsResult struct {
	PingResults  []CustomTestResult `json:"pingResults"`
	TCPResults   []CustomTestResult `json:"tcpResults"`
	UDPResults   []CustomTestResult `json:"udpResults"`
	HTTPResults  []CustomTestResult `json:"httpResults"`
	RTSPResults  []CustomTestResult `json:"rtspResults"`  // Issue #778
	DICOMResults []CustomTestResult `json:"dicomResults"` // Issue #777
	HasTests     bool               `json:"hasTests"`
}

// ============================================================================
// Health Checks Test Handlers
// ============================================================================

// handleHealthChecks runs all configured health checks and returns results.
func (s *Server) handleHealthChecks(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	result := CustomTestsResult{
		PingResults:  s.runPingTests(),
		TCPResults:   s.runTCPTests(r.Context()),
		UDPResults:   s.runUDPTests(),
		HTTPResults:  s.runHTTPTests(r.Context(), logger),
		RTSPResults:  s.runRTSPTests(r.Context()),  // Issue #778
		DICOMResults: s.runDICOMTests(r.Context()), // Issue #777
	}

	result.HasTests = len(s.config.HealthChecks.PingTargets) > 0 ||
		len(s.config.HealthChecks.TCPPorts) > 0 ||
		len(s.config.HealthChecks.UDPPorts) > 0 ||
		len(s.config.HealthChecks.HTTPEndpoints) > 0 ||
		len(s.config.HealthChecks.RTSPEndpoints) > 0 ||
		len(s.config.HealthChecks.DICOMEndpoints) > 0

	sendJSONResponse(w, logger, http.StatusOK, result)
}

// ============================================================================
// Ping Tests
// ============================================================================

// runPingTests runs all configured ping tests and returns results.
func (s *Server) runPingTests() []CustomTestResult {
	results := make([]CustomTestResult, 0, len(s.config.HealthChecks.PingTargets))
	threshold := s.config.Thresholds.CustomTests.Ping

	for _, target := range s.config.HealthChecks.PingTargets {
		if !target.Enabled {
			continue
		}

		name := target.Name
		if name == "" {
			name = target.Host
		}

		testResult := CustomTestResult{Name: name, Host: target.Host}
		pingStats, err := runExtendedPing(target.Host, defaultPingCount)

		if err != nil {
			testResult.Success = false
			testResult.Error = "Ping test failed"
			testResult.TestStatus = statusError
		} else {
			testResult.Success = pingStats.PacketLoss < packetLossThresholdFull
			testResult.Latency = pingStats.AvgLatency
			testResult.MinLatency = pingStats.MinLatency
			testResult.MaxLatency = pingStats.MaxLatency
			testResult.PacketLoss = pingStats.PacketLoss
			testResult.Jitter = pingStats.Jitter
			testResult.TestStatus = s.evaluatePingStatus(pingStats, threshold)
		}
		results = append(results, testResult)
	}
	return results
}

// evaluatePingStatus determines ping test status based on packet loss and latency.
func (s *Server) evaluatePingStatus(stats *PingStats, threshold config.Threshold) string {
	switch {
	case stats.PacketLoss > packetLossThresholdHigh:
		return statusError
	case stats.PacketLoss > packetLossThresholdLow:
		return statusWarning
	default:
		return getTestStatus(
			stats.AvgLatency,
			threshold.Warning.Milliseconds(),
			threshold.Critical.Milliseconds(),
		)
	}
}

// PingStats holds extended ping statistics.
type PingStats struct {
	AvgLatency float64 // ms
	MinLatency float64 // ms
	MaxLatency float64 // ms
	PacketLoss float64 // percentage
	Jitter     float64 // ms (standard deviation)
}

// runExtendedPing runs multiple pings and returns statistics.
func runExtendedPing(host string, count int) (*PingStats, error) {
	var latencies []float64
	sent := 0
	received := 0

	for i := range count {
		sent++
		ctx, cancel := context.WithTimeout(context.Background(), pingProbeTimeoutSec*time.Second)

		start := time.Now()
		// Try TCP 80/443 as ping alternative (actual ICMP requires root)
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", host+":80")
		if err != nil {
			conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", host+":443")
		}
		cancel()

		if err == nil {
			latency := time.Since(start).Seconds() * millisecondsPerSecond
			latencies = append(latencies, latency)
			received++
			_ = conn.Close()
		}

		// Small delay between pings
		if i < count-1 {
			time.Sleep(pingProbeDelayMs * time.Millisecond)
		}
	}

	if len(latencies) == 0 {
		return &PingStats{PacketLoss: packetLossThresholdFull}, errors.New("host unreachable")
	}

	// Calculate statistics
	stats := &PingStats{
		PacketLoss: float64(sent-received) / float64(sent) * percentageDivisor,
	}

	// Min, max, avg
	stats.MinLatency = latencies[0]
	stats.MaxLatency = latencies[0]
	var sum float64
	for _, lat := range latencies {
		sum += lat
		if lat < stats.MinLatency {
			stats.MinLatency = lat
		}
		if lat > stats.MaxLatency {
			stats.MaxLatency = lat
		}
	}
	stats.AvgLatency = sum / float64(len(latencies))

	// Jitter (standard deviation)
	if len(latencies) > 1 {
		var variance float64
		for _, lat := range latencies {
			diff := lat - stats.AvgLatency
			variance += diff * diff
		}
		stats.Jitter = math.Sqrt(variance / float64(len(latencies)))
	}

	return stats, nil
}

// ============================================================================
// TCP Tests
// ============================================================================

// runTCPTests runs all configured TCP port tests and returns results.
func (s *Server) runTCPTests(ctx context.Context) []CustomTestResult {
	results := make([]CustomTestResult, 0, len(s.config.HealthChecks.TCPPorts))
	threshold := s.config.Thresholds.CustomTests.TCP

	for _, target := range s.config.HealthChecks.TCPPorts {
		if !target.Enabled {
			continue
		}

		name := target.Name
		if name == "" {
			name = net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
		}

		testResult := CustomTestResult{Name: name, Host: target.Host, Port: target.Port}
		latency, err := runTCPTest(ctx, target.Host, target.Port)

		if err != nil {
			testResult.Success = false
			testResult.Error = "TCP connection test failed"
			testResult.TestStatus = statusError
		} else {
			testResult.Success = true
			testResult.Latency = latency
			warningMs := threshold.Warning.Milliseconds()
			criticalMs := threshold.Critical.Milliseconds()
			testResult.TestStatus = getTestStatus(latency, warningMs, criticalMs)
		}
		results = append(results, testResult)
	}
	return results
}

// runTCPTest runs a TCP port test and returns latency in ms.
func runTCPTest(ctx context.Context, host string, port int) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, tcpTestTimeoutSec*time.Second)
	defer cancel()

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	start := time.Now()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, err
	}
	latency := time.Since(start).Seconds() * millisecondsPerSecond
	_ = conn.Close()
	return latency, nil
}

// ============================================================================
// UDP Tests
// ============================================================================

// runUDPTests runs all configured UDP port tests and returns results.
func (s *Server) runUDPTests() []CustomTestResult {
	results := make([]CustomTestResult, 0, len(s.config.HealthChecks.UDPPorts))
	threshold := s.config.Thresholds.CustomTests.UDP

	for _, target := range s.config.HealthChecks.UDPPorts {
		if !target.Enabled {
			continue
		}

		name := target.Name
		if name == "" {
			name = net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
		}

		testResult := CustomTestResult{Name: name, Host: target.Host, Port: target.Port}
		latency, err := runUDPTest(target.Host, target.Port)

		if err != nil {
			testResult.Success = false
			testResult.Error = "UDP connection test failed"
			testResult.TestStatus = statusError
		} else {
			testResult.Success = true
			testResult.Latency = latency
			warningMs := threshold.Warning.Milliseconds()
			criticalMs := threshold.Critical.Milliseconds()
			testResult.TestStatus = getTestStatus(latency, warningMs, criticalMs)
		}
		results = append(results, testResult)
	}
	return results
}

// runUDPTest runs a UDP port test and returns latency in ms.
// Note: UDP is connectionless, so we send a packet and wait for ICMP unreachable
// or application response. For DNS (53), NTP (123), etc. we can get actual responses.
func runUDPTest(host string, port int) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), udpTestTimeoutSec*time.Second)
	defer cancel()

	addr := net.JoinHostPort(host, strconv.Itoa(port))

	// For DNS port, try a simple DNS query
	if port == dnsPort {
		return testDNSPort(ctx, host)
	}

	// For other UDP ports, we try to connect (which on UDP just sets up local state)
	// and send a small probe packet
	start := time.Now()

	dialer := net.Dialer{Timeout: udpTestTimeoutSec * time.Second}
	conn, err := dialer.DialContext(ctx, "udp", addr)
	if err != nil {
		return 0, err
	}
	defer func() { _ = conn.Close() }()

	// Set deadline for response
	if deadlineErr := conn.SetDeadline(time.Now().Add(udpReadDeadlineSec * time.Second)); deadlineErr != nil {
		return 0, deadlineErr
	}

	// Send a small probe packet
	_, err = conn.Write([]byte{0x00})
	if err != nil {
		return 0, err
	}

	// Try to read response (may timeout for non-responding services)
	buf := make([]byte, udpReadBufferBytes)
	_, err = conn.Read(buf)

	latency := time.Since(start).Seconds() * millisecondsPerSecond

	// For UDP, no error on Write means the port is likely open
	// (no ICMP unreachable received)
	if err != nil {
		// Check if it's a timeout (which for UDP often means the port is open but not responding)
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			// Port is likely open but service didn't respond - still count as success
			return latency, nil
		}
		// Connection refused or other error means port is closed
		return 0, errors.New("port closed or filtered")
	}

	return latency, nil
}

// testDNSPort tests DNS port by sending a simple query.
func testDNSPort(ctx context.Context, host string) (float64, error) {
	// Use Go's resolver to test DNS
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(dialCtx context.Context, _, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: udpTestTimeoutSec * time.Second}
			return d.DialContext(dialCtx, "udp", host+":53")
		},
	}

	start := time.Now()
	_, err := resolver.LookupHost(ctx, "google.com")
	latency := time.Since(start).Seconds() * millisecondsPerSecond

	if err != nil {
		return 0, err
	}
	return latency, nil
}

// ============================================================================
// HTTP Tests
// ============================================================================

// runHTTPTests runs all configured HTTP endpoint tests and returns results.
func (s *Server) runHTTPTests(ctx context.Context, logger *slog.Logger) []CustomTestResult {
	results := make([]CustomTestResult, 0, len(s.config.HealthChecks.HTTPEndpoints))

	for _, endpoint := range s.config.HealthChecks.HTTPEndpoints {
		if !endpoint.Enabled {
			continue
		}

		if err := validation.ValidateURL(endpoint.URL); err != nil {
			logger.WarnContext(ctx, "Skipping invalid HTTP endpoint URL", "url", endpoint.URL, "error", err)
			continue
		}

		result := s.runSingleHTTPTest(ctx, endpoint)
		results = append(results, result)
	}
	return results
}

// runSingleHTTPTest runs a single HTTP endpoint test.
func (s *Server) runSingleHTTPTest(
	ctx context.Context,
	endpoint config.HTTPEndpoint,
) CustomTestResult {
	thresholds := s.config.Thresholds.CustomTests

	url, tryHTTPFallback := normalizeHTTPURL(endpoint.URL)
	name := endpoint.Name
	if name == "" {
		name = endpoint.URL
	}

	testResult := CustomTestResult{Name: name, URL: url}

	// Determine if we need enhanced testing (body match, security headers, or redirects)
	needsEnhanced := endpoint.BodyMatch != "" || endpoint.CheckSecurityHeaders || endpoint.FollowRedirects

	if needsEnhanced {
		// Use enhanced HTTP test with body reading and redirect following
		resp, err := runHTTPTestEnhanced(
			ctx,
			url,
			endpoint.ExpectedStatus,
			endpoint.FollowRedirects,
			endpoint.MaxRedirects,
		)

		// Try HTTP fallback if HTTPS failed
		if err != nil && tryHTTPFallback {
			httpURL := "http://" + endpoint.URL
			if httpResp, httpErr := runHTTPTestEnhanced(ctx, httpURL, endpoint.ExpectedStatus, endpoint.FollowRedirects, endpoint.MaxRedirects); httpErr == nil ||
				(httpResp != nil && httpResp.StatusCode > 0) {
				url = httpURL
				testResult.URL = httpURL
				resp, err = httpResp, httpErr
			}
		}

		if resp != nil {
			testResult.Status = resp.StatusCode
			testResult.Latency = resp.Timings.Total
			testResult.DNSLatency = resp.Timings.DNS
			testResult.TCPConnect = resp.Timings.Connect
			testResult.TLSLatency = resp.Timings.TLS
			testResult.TTFBLatency = resp.Timings.TTFB
			testResult.ResponseSize = resp.BodySize
			testResult.HTTPVersion = resp.HTTPVersion

			if len(resp.RedirectHops) > 0 {
				testResult.RedirectChain = resp.RedirectHops
			}

			if err == nil {
				testResult.Success = true
				s.evaluateHTTPTimings(&testResult, resp.Timings, &thresholds)

				// Check body match if configured
				if endpoint.BodyMatch != "" {
					matched, matchErr := checkBodyMatch(resp.Body, endpoint.BodyMatch, endpoint.BodyMatchIsRegex)
					testResult.BodyMatchSuccess = matched
					if matchErr != nil {
						testResult.BodyMatchStatus = statusError
						testResult.Error = matchErr.Error()
						testResult.Success = false
						testResult.TestStatus = statusError
					} else if !matched {
						testResult.BodyMatchStatus = statusError
						testResult.Success = false
						testResult.TestStatus = statusError
						testResult.Error = "Body content did not match expected pattern"
					} else {
						testResult.BodyMatchStatus = statusSuccess
					}
				}

				// Check security headers if configured
				if endpoint.CheckSecurityHeaders && testResult.Success {
					isHTTPS := strings.HasPrefix(url, "https://")
					testResult.SecurityHeaders = checkSecurityHeaders(resp.Headers, isHTTPS)
				}
			} else {
				testResult.Success = false
				testResult.Error = "HTTP request failed"
				testResult.TestStatus = statusError
			}
		} else if err != nil {
			testResult.Success = false
			testResult.Error = "HTTP request failed"
			testResult.TestStatus = statusError
		}
	} else {
		// Use standard HTTP test (faster, no body reading)
		statusCode, timings, err := runHTTPTest(ctx, url, endpoint.ExpectedStatus)

		// Try HTTP fallback if HTTPS failed
		if err != nil && tryHTTPFallback {
			httpURL := "http://" + endpoint.URL
			if httpStatus, httpTimings, httpErr := runHTTPTest(ctx, httpURL, endpoint.ExpectedStatus); httpErr == nil ||
				httpStatus > 0 {
				url = httpURL
				testResult.URL = httpURL
				statusCode, timings, err = httpStatus, httpTimings, httpErr
			}
		}

		testResult.Status = statusCode
		testResult.Latency = timings.Total
		testResult.DNSLatency = timings.DNS
		testResult.TCPConnect = timings.Connect
		testResult.TLSLatency = timings.TLS
		testResult.TTFBLatency = timings.TTFB

		if err != nil {
			testResult.Success = false
			testResult.Error = "HTTP request failed"
			testResult.TestStatus = statusError
		} else {
			testResult.Success = true
			s.evaluateHTTPTimings(&testResult, timings, &thresholds)
		}
	}

	// Check certificate expiry for HTTPS URLs
	if strings.HasPrefix(url, "https://") && testResult.Success {
		s.evaluateCertExpiry(&testResult, url, thresholds.CertExpiry)
	}

	return testResult
}

// normalizeHTTPURL adds scheme if missing and returns whether HTTP fallback should be tried.
func normalizeHTTPURL(rawURL string) (string, bool) {
	if rawURL == "" {
		return rawURL, false
	}
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL, false
	}
	return "https://" + rawURL, true
}

type httpTimings struct {
	DNS     float64
	Connect float64
	TLS     float64
	TTFB    float64 // Time to first byte (from request sent to first response byte)
	Total   float64
}

// httpResponse contains extended HTTP response data for enhanced checks.
type httpResponse struct {
	StatusCode   int
	Headers      http.Header
	Body         []byte // Limited to maxBodyReadBytes
	BodySize     int64  // Total content length (may be larger than Body)
	HTTPVersion  string // HTTP/1.1, HTTP/2, etc.
	Timings      httpTimings
	RedirectHops []RedirectHop
}

// Maximum bytes to read from response body for pattern matching.
const maxBodyReadBytes = 64 * 1024 // 64KB

// Default max redirects if not specified.
const defaultMaxRedirects = 10

// runHTTPTest runs an HTTP test and returns status code and timings in ms.
// Uses SafeTransport to prevent DNS rebinding SSRF attacks.
func runHTTPTest(ctx context.Context, url string, expectedStatus int) (int, httpTimings, error) {
	var timing httpTimings
	// Use SafeTransport to block connections to private IPs (prevents DNS rebinding)
	transport := validation.SafeTransport()
	client := &http.Client{
		Transport: transport,
		Timeout:   httpClientTimeoutSec * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	ctx, cancel := context.WithTimeout(ctx, client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return 0, timing, err
	}

	var dnsStart, connStart, tlsStart, wroteRequest time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(httptrace.DNSDoneInfo) {
			if !dnsStart.IsZero() {
				timing.DNS += time.Since(dnsStart).Seconds() * millisecondsPerSecond
			}
		},
		ConnectStart: func(_, _ string) {
			connStart = time.Now()
		},
		ConnectDone: func(_, _ string, _ error) {
			if !connStart.IsZero() {
				timing.Connect += time.Since(connStart).Seconds() * millisecondsPerSecond
			}
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			if !tlsStart.IsZero() {
				timing.TLS += time.Since(tlsStart).Seconds() * millisecondsPerSecond
			}
		},
		WroteRequest: func(httptrace.WroteRequestInfo) {
			wroteRequest = time.Now()
		},
		GotFirstResponseByte: func() {
			if !wroteRequest.IsZero() {
				timing.TTFB = time.Since(wroteRequest).Seconds() * millisecondsPerSecond
			}
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))

	start := time.Now()
	resp, err := client.Do(req)
	timing.Total = time.Since(start).Seconds() * millisecondsPerSecond

	if err != nil {
		return 0, timing, err
	}
	defer func() { _ = resp.Body.Close() }()

	statusCode := resp.StatusCode
	if expectedStatus > 0 && statusCode != expectedStatus {
		return statusCode, timing, fmt.Errorf("expected %d, got %d", expectedStatus, statusCode)
	}

	return statusCode, timing, nil
}

// runHTTPTestEnhanced runs an HTTP test with body reading and redirect following.
// Returns full response details for body matching and security header checks.
func runHTTPTestEnhanced(
	ctx context.Context,
	url string,
	expectedStatus int,
	followRedirects bool,
	maxRedirects int,
) (*httpResponse, error) {
	result := &httpResponse{
		RedirectHops: make([]RedirectHop, 0),
	}

	if maxRedirects <= 0 {
		maxRedirects = defaultMaxRedirects
	}

	transport := validation.SafeTransport()

	currentURL := url
	var lastTiming httpTimings

	for i := 0; i <= maxRedirects; i++ {
		hopStart := time.Now()
		resp, timing, err := runSingleHTTPRequest(ctx, currentURL, transport)
		if err != nil {
			return nil, err
		}

		lastTiming = timing

		// Check if this is a redirect
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			location := resp.Header.Get("Location")
			_ = resp.Body.Close()

			hop := RedirectHop{
				URL:        currentURL,
				StatusCode: resp.StatusCode,
				LatencyMs:  time.Since(hopStart).Seconds() * millisecondsPerSecond,
			}
			result.RedirectHops = append(result.RedirectHops, hop)

			if !followRedirects || location == "" {
				// Return the redirect response without following
				result.StatusCode = resp.StatusCode
				result.Headers = resp.Header
				result.HTTPVersion = formatHTTPVersion(resp.Proto)
				result.Timings = lastTiming
				return result, nil
			}

			// Resolve relative URLs
			if !strings.HasPrefix(location, "http://") && !strings.HasPrefix(location, "https://") {
				// Relative URL - resolve against current URL
				if strings.HasPrefix(location, "/") {
					// Absolute path
					idx := strings.Index(currentURL, "//")
					if idx >= 0 {
						hostEnd := strings.Index(currentURL[idx+2:], "/")
						if hostEnd >= 0 {
							currentURL = currentURL[:idx+2+hostEnd] + location
						} else {
							currentURL += location
						}
					}
				} else {
					// Relative path
					lastSlash := strings.LastIndex(currentURL, "/")
					if lastSlash > 8 { // After "https://"
						currentURL = currentURL[:lastSlash+1] + location
					}
				}
			} else {
				currentURL = location
			}
			continue
		}

		// Non-redirect response - read body and return
		result.StatusCode = resp.StatusCode
		result.Headers = resp.Header
		result.HTTPVersion = formatHTTPVersion(resp.Proto)
		result.Timings = lastTiming

		// Read body (limited)
		body, size := readLimitedBody(resp.Body)
		_ = resp.Body.Close()
		result.Body = body
		result.BodySize = size

		// Check expected status
		if expectedStatus > 0 && result.StatusCode != expectedStatus {
			return result, fmt.Errorf("expected %d, got %d", expectedStatus, result.StatusCode)
		}

		return result, nil
	}

	return nil, fmt.Errorf("too many redirects (max %d)", maxRedirects)
}

// runSingleHTTPRequest performs a single HTTP request without following redirects.
func runSingleHTTPRequest(
	ctx context.Context,
	url string,
	transport http.RoundTripper,
) (*http.Response, httpTimings, error) {
	var timing httpTimings
	client := &http.Client{
		Transport: transport,
		Timeout:   httpClientTimeoutSec * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	ctx, cancel := context.WithTimeout(ctx, client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, timing, err
	}

	var dnsStart, connStart, tlsStart, wroteRequest time.Time
	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone: func(httptrace.DNSDoneInfo) {
			if !dnsStart.IsZero() {
				timing.DNS += time.Since(dnsStart).Seconds() * millisecondsPerSecond
			}
		},
		ConnectStart: func(_, _ string) { connStart = time.Now() },
		ConnectDone: func(_, _ string, _ error) {
			if !connStart.IsZero() {
				timing.Connect += time.Since(connStart).Seconds() * millisecondsPerSecond
			}
		},
		TLSHandshakeStart: func() { tlsStart = time.Now() },
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			if !tlsStart.IsZero() {
				timing.TLS += time.Since(tlsStart).Seconds() * millisecondsPerSecond
			}
		},
		WroteRequest: func(httptrace.WroteRequestInfo) { wroteRequest = time.Now() },
		GotFirstResponseByte: func() {
			if !wroteRequest.IsZero() {
				timing.TTFB = time.Since(wroteRequest).Seconds() * millisecondsPerSecond
			}
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))

	start := time.Now()
	resp, err := client.Do(req)
	timing.Total = time.Since(start).Seconds() * millisecondsPerSecond

	return resp, timing, err
}

// formatHTTPVersion formats the HTTP protocol version.
func formatHTTPVersion(proto string) string {
	switch proto {
	case "HTTP/2.0":
		return "HTTP/2"
	case "HTTP/3":
		return "HTTP/3"
	default:
		return proto
	}
}

// readLimitedBody reads up to maxBodyReadBytes from body and returns total size.
func readLimitedBody(body io.ReadCloser) ([]byte, int64) {
	limitedReader := io.LimitReader(body, maxBodyReadBytes)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, 0
	}

	// Try to determine total size
	size := int64(len(data))

	// Read and discard remaining bytes to get total size
	remaining, _ := io.Copy(io.Discard, body)
	size += remaining

	return data, size
}

// checkBodyMatch checks if response body matches the expected pattern.
func checkBodyMatch(body []byte, pattern string, isRegex bool) (bool, error) {
	if pattern == "" {
		return true, nil
	}

	bodyStr := string(body)

	if isRegex {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false, fmt.Errorf("invalid regex pattern: %w", err)
		}
		return re.MatchString(bodyStr), nil
	}

	// Substring match
	return strings.Contains(bodyStr, pattern), nil
}

// checkSecurityHeaders evaluates security headers in the response.
func checkSecurityHeaders(headers http.Header, isHTTPS bool) *SecurityHeaders {
	result := &SecurityHeaders{}
	score := 0
	maxScore := 0

	// HSTS (Strict-Transport-Security) - only for HTTPS
	if isHTTPS {
		maxScore += 20
		result.HSTS = checkHSTSHeader(headers.Get("Strict-Transport-Security"))
		if result.HSTS.Present && result.HSTS.Status == statusSuccess {
			score += 20
		}
	}

	// Content-Security-Policy
	maxScore += 20
	result.CSP = checkCSPHeader(headers.Get("Content-Security-Policy"))
	if result.CSP.Present && result.CSP.Status == statusSuccess {
		score += 20
	}

	// X-Frame-Options
	maxScore += 15
	result.XFrameOptions = checkXFrameOptionsHeader(headers.Get("X-Frame-Options"))
	if result.XFrameOptions.Present && result.XFrameOptions.Status == statusSuccess {
		score += 15
	}

	// X-Content-Type-Options
	maxScore += 15
	result.XContentType = checkXContentTypeHeader(headers.Get("X-Content-Type-Options"))
	if result.XContentType.Present && result.XContentType.Status == statusSuccess {
		score += 15
	}

	// X-XSS-Protection (deprecated but still checked)
	maxScore += 10
	result.XSSProtection = checkXSSProtectionHeader(headers.Get("X-XSS-Protection"))
	if result.XSSProtection.Present {
		score += 10
	}

	// Referrer-Policy
	maxScore += 10
	result.ReferrerPolicy = checkReferrerPolicyHeader(headers.Get("Referrer-Policy"))
	if result.ReferrerPolicy.Present && result.ReferrerPolicy.Status == statusSuccess {
		score += 10
	}

	// Permissions-Policy
	maxScore += 10
	result.PermissionsPolicy = checkPermissionsPolicyHeader(headers.Get("Permissions-Policy"))
	if result.PermissionsPolicy.Present {
		score += 10
	}

	// Calculate overall score and status
	if maxScore > 0 {
		result.Score = (score * 100) / maxScore
	}

	switch {
	case result.Score >= 80:
		result.OverallStatus = statusSuccess
	case result.Score >= 50:
		result.OverallStatus = statusWarning
	default:
		result.OverallStatus = statusError
	}

	return result
}

func checkHSTSHeader(value string) *HeaderCheck {
	check := &HeaderCheck{Value: value}
	if value == "" {
		check.Present = false
		check.Status = statusError
		check.Message = "Missing: Add Strict-Transport-Security header"
		return check
	}
	check.Present = true
	// Check for max-age directive
	if strings.Contains(strings.ToLower(value), "max-age=") {
		check.Status = statusSuccess
		if strings.Contains(strings.ToLower(value), "includesubdomains") {
			check.Message = "Good: HSTS enabled with includeSubDomains"
		} else {
			check.Message = "OK: Consider adding includeSubDomains"
		}
	} else {
		check.Status = statusWarning
		check.Message = "Warning: max-age directive not found"
	}
	return check
}

func checkCSPHeader(value string) *HeaderCheck {
	check := &HeaderCheck{Value: value}
	if value == "" {
		check.Present = false
		check.Status = statusError
		check.Message = "Missing: Add Content-Security-Policy header"
		return check
	}
	check.Present = true
	// Check for unsafe directives
	lowerVal := strings.ToLower(value)
	if strings.Contains(lowerVal, "'unsafe-inline'") || strings.Contains(lowerVal, "'unsafe-eval'") {
		check.Status = statusWarning
		check.Message = "Warning: Contains unsafe directives"
	} else if strings.Contains(lowerVal, "default-src") || strings.Contains(lowerVal, "script-src") {
		check.Status = statusSuccess
		check.Message = "Good: CSP policy defined"
	} else {
		check.Status = statusWarning
		check.Message = "Warning: Consider adding script-src directive"
	}
	return check
}

func checkXFrameOptionsHeader(value string) *HeaderCheck {
	check := &HeaderCheck{Value: value}
	if value == "" {
		check.Present = false
		check.Status = statusError
		check.Message = "Missing: Add X-Frame-Options (DENY or SAMEORIGIN)"
		return check
	}
	check.Present = true
	upperVal := strings.ToUpper(value)
	if upperVal == "DENY" || upperVal == "SAMEORIGIN" {
		check.Status = statusSuccess
		check.Message = "Good: Clickjacking protection enabled"
	} else {
		check.Status = statusWarning
		check.Message = "Warning: Use DENY or SAMEORIGIN"
	}
	return check
}

func checkXContentTypeHeader(value string) *HeaderCheck {
	check := &HeaderCheck{Value: value}
	if value == "" {
		check.Present = false
		check.Status = statusError
		check.Message = "Missing: Add X-Content-Type-Options: nosniff"
		return check
	}
	check.Present = true
	if strings.EqualFold(value, "nosniff") {
		check.Status = statusSuccess
		check.Message = "Good: MIME type sniffing protection enabled"
	} else {
		check.Status = statusWarning
		check.Message = "Warning: Value should be 'nosniff'"
	}
	return check
}

func checkXSSProtectionHeader(value string) *HeaderCheck {
	check := &HeaderCheck{Value: value}
	if value == "" {
		check.Present = false
		check.Status = statusWarning
		check.Message = "Not present (deprecated header, CSP preferred)"
		return check
	}
	check.Present = true
	check.Status = statusSuccess
	check.Message = "Present (deprecated, rely on CSP instead)"
	return check
}

func checkReferrerPolicyHeader(value string) *HeaderCheck {
	check := &HeaderCheck{Value: value}
	if value == "" {
		check.Present = false
		check.Status = statusWarning
		check.Message = "Missing: Consider adding Referrer-Policy"
		return check
	}
	check.Present = true
	lowerVal := strings.ToLower(value)
	goodPolicies := []string{"strict-origin", "strict-origin-when-cross-origin", "no-referrer", "same-origin"}
	for _, policy := range goodPolicies {
		if strings.Contains(lowerVal, policy) {
			check.Status = statusSuccess
			check.Message = "Good: Secure referrer policy"
			return check
		}
	}
	check.Status = statusWarning
	check.Message = "Warning: Consider stricter referrer policy"
	return check
}

func checkPermissionsPolicyHeader(value string) *HeaderCheck {
	check := &HeaderCheck{Value: value}
	if value == "" {
		check.Present = false
		check.Status = statusWarning
		check.Message = "Missing: Consider adding Permissions-Policy"
		return check
	}
	check.Present = true
	check.Status = statusSuccess
	check.Message = "Good: Feature policy defined"
	return check
}

// evaluateHTTPTimings sets timing statuses and overall test status.
func (s *Server) evaluateHTTPTimings(
	result *CustomTestResult,
	timings httpTimings,
	thresholds *config.CustomThresholds,
) {
	httpTimingThresholds := thresholds.HTTPTimings

	result.DNSStatus = getTestStatus(
		timings.DNS,
		httpTimingThresholds.DNS.Warning.Milliseconds(),
		httpTimingThresholds.DNS.Critical.Milliseconds(),
	)
	result.TCPStatus = getTestStatus(
		timings.Connect,
		httpTimingThresholds.TCP.Warning.Milliseconds(),
		httpTimingThresholds.TCP.Critical.Milliseconds(),
	)
	result.TLSStatus = getTestStatus(
		timings.TLS,
		httpTimingThresholds.TLS.Warning.Milliseconds(),
		httpTimingThresholds.TLS.Critical.Milliseconds(),
	)
	result.TTFBStatus = getTestStatus(
		timings.TTFB,
		httpTimingThresholds.TTFB.Warning.Milliseconds(),
		httpTimingThresholds.TTFB.Critical.Milliseconds(),
	)

	switch {
	case result.DNSStatus == statusError || result.TCPStatus == statusError ||
		result.TLSStatus == statusError || result.TTFBStatus == statusError:
		result.TestStatus = statusError
	case result.DNSStatus == statusWarning || result.TCPStatus == statusWarning ||
		result.TLSStatus == statusWarning || result.TTFBStatus == statusWarning:
		result.TestStatus = statusWarning
	default:
		result.TestStatus = getTestStatus(
			timings.Total,
			thresholds.HTTP.Warning.Milliseconds(),
			thresholds.HTTP.Critical.Milliseconds(),
		)
	}
}

// ============================================================================
// Certificate Expiry Check
// ============================================================================

// CertInfo holds certificate expiry information.
type CertInfo struct {
	DaysLeft   int
	Status     string // success, warning, error
	ExpiryDate string
	CommonName string
	TLSVersion string // TLS 1.0, TLS 1.1, TLS 1.2, TLS 1.3
	Issuer     string // Certificate issuer (for context)
}

// evaluateCertExpiry checks certificate expiry and updates test result.
func (s *Server) evaluateCertExpiry(
	result *CustomTestResult,
	url string,
	threshold config.CertExpiryThreshold,
) {
	certInfo := checkCertExpiry(url, threshold.Warning, threshold.Critical)
	result.CertDaysLeft = certInfo.DaysLeft
	result.CertStatus = certInfo.Status
	result.CertExpiry = certInfo.ExpiryDate
	result.CertCommonName = certInfo.CommonName
	result.TLSVersion = certInfo.TLSVersion
	result.CertIssuer = certInfo.Issuer

	if certInfo.Status == statusError && result.TestStatus != statusError {
		result.TestStatus = statusError
	} else if certInfo.Status == statusWarning && result.TestStatus == statusSuccess {
		result.TestStatus = statusWarning
	}
}

// checkCertExpiry checks the TLS certificate expiry for a URL.
func checkCertExpiry(url string, warningDays, criticalDays int) CertInfo {
	info := CertInfo{Status: statusSuccess}

	// Extract host from URL
	host := strings.TrimPrefix(url, "https://")
	host = strings.TrimPrefix(host, "http://")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, ":"); idx == -1 {
		host += ":443"
	}

	// Connect with TLS using context-aware dialing
	ctx, cancel := context.WithTimeout(context.Background(), certCheckTimeoutSec*time.Second)
	defer cancel()

	dialer := &net.Dialer{Timeout: certCheckTimeoutSec * time.Second}
	rawConn, err := dialer.DialContext(ctx, protoTCP, host)
	if err != nil {
		info.Status = statusError
		return info
	}

	// #nosec G402 - certificate verification intentionally skipped to inspect expiry
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	} // We want to check expiry even for self-signed
	conn := tls.Client(rawConn, tlsConfig)
	if hsErr := conn.HandshakeContext(ctx); hsErr != nil {
		_ = rawConn.Close()
		info.Status = statusError
		return info
	}
	defer func() { _ = conn.Close() }()

	// Get connection state for TLS info
	connState := conn.ConnectionState()

	// Get TLS version
	info.TLSVersion = getTLSVersionString(connState.Version)

	// Get certificate chain
	certs := connState.PeerCertificates
	if len(certs) == 0 {
		info.Status = statusError
		return info
	}

	// Check the leaf certificate
	cert := certs[0]
	info.CommonName = cert.Subject.CommonName
	info.ExpiryDate = cert.NotAfter.Format("2006-01-02")

	// Get issuer (org or CN)
	if len(cert.Issuer.Organization) > 0 {
		info.Issuer = cert.Issuer.Organization[0]
	} else if cert.Issuer.CommonName != "" {
		info.Issuer = cert.Issuer.CommonName
	}

	// Calculate days until expiry
	daysLeft := int(time.Until(cert.NotAfter).Hours() / hoursPerDay)
	info.DaysLeft = daysLeft

	// Determine status
	switch {
	case daysLeft <= 0:
		info.Status = statusError // Expired
	case daysLeft <= criticalDays:
		info.Status = statusError // Critical
	case daysLeft <= warningDays:
		info.Status = statusWarning // Warning
	default:
		info.Status = statusSuccess // OK
	}

	return info
}

// getTLSVersionString converts TLS version to human-readable string.
func getTLSVersionString(tlsVersion uint16) string {
	switch tlsVersion {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return "Unknown"
	}
}

// ============================================================================
// RTSP Tests (Issue #778)
// ============================================================================

// RTSP protocol constants.
const (
	rtspDefaultPort     = 554
	rtspTestTimeoutSec  = 10
	rtspReadBufferBytes = 1024
)

// runRTSPTests runs all configured RTSP stream tests and returns results.
func (s *Server) runRTSPTests(ctx context.Context) []CustomTestResult {
	results := make([]CustomTestResult, 0, len(s.config.HealthChecks.RTSPEndpoints))
	threshold := s.config.Thresholds.CustomTests.TCP // Use TCP thresholds for RTSP

	for _, endpoint := range s.config.HealthChecks.RTSPEndpoints {
		if !endpoint.Enabled {
			continue
		}

		name := endpoint.Name
		if name == "" {
			name = endpoint.URL
		}

		testResult := CustomTestResult{Name: name, URL: endpoint.URL}
		latency, err := runRTSPTest(ctx, endpoint.URL)

		if err != nil {
			testResult.Success = false
			testResult.Error = "RTSP test failed: " + err.Error()
			testResult.TestStatus = statusError
		} else {
			testResult.Success = true
			testResult.Latency = latency
			warningMs := threshold.Warning.Milliseconds()
			criticalMs := threshold.Critical.Milliseconds()
			testResult.TestStatus = getTestStatus(latency, warningMs, criticalMs)
		}
		results = append(results, testResult)
	}
	return results
}

// runRTSPTest runs an RTSP OPTIONS request and returns latency in ms.
// RTSP uses a simple text-based protocol similar to HTTP.
func runRTSPTest(ctx context.Context, rtspURL string) (float64, error) {
	// Parse RTSP URL to extract host and port
	// rtsp://host:port/path
	url := strings.TrimPrefix(rtspURL, "rtsp://")
	url = strings.TrimPrefix(url, "rtsps://")

	// Split host:port from path
	hostPort, _, _ := strings.Cut(url, "/")

	// Add default port if not specified
	if !strings.Contains(hostPort, ":") {
		hostPort = hostPort + ":" + strconv.Itoa(rtspDefaultPort)
	}

	ctx, cancel := context.WithTimeout(ctx, rtspTestTimeoutSec*time.Second)
	defer cancel()

	start := time.Now()

	// Connect via TCP
	dialer := net.Dialer{Timeout: rtspTestTimeoutSec * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", hostPort)
	if err != nil {
		return 0, fmt.Errorf("connection failed: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Set deadline for the entire exchange
	if deadlineErr := conn.SetDeadline(time.Now().Add(rtspTestTimeoutSec * time.Second)); deadlineErr != nil {
		return 0, deadlineErr
	}

	// Send RTSP OPTIONS request
	request := fmt.Sprintf("OPTIONS %s RTSP/1.0\r\nCSeq: 1\r\n\r\n", rtspURL)
	_, err = conn.Write([]byte(request))
	if err != nil {
		return 0, fmt.Errorf("failed to send OPTIONS: %w", err)
	}

	// Read response
	buf := make([]byte, rtspReadBufferBytes)
	n, err := conn.Read(buf)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	latency := time.Since(start).Seconds() * millisecondsPerSecond

	// Check for valid RTSP response
	response := string(buf[:n])
	if !strings.Contains(response, "RTSP/1.0") {
		return latency, errors.New("invalid RTSP response")
	}

	// Check for 200 OK status
	if !strings.Contains(response, "200 OK") && !strings.Contains(response, "200") {
		return latency, errors.New("RTSP server returned error")
	}

	return latency, nil
}

// ============================================================================
// DICOM Tests (Issue #777)
// ============================================================================

// DICOM protocol constants.
const (
	dicomDefaultPort    = 104
	dicomTestTimeoutSec = 10
	// DICOM PDU types.
	dicomPDUAssocRQ = 0x01 // A-ASSOCIATE-RQ.
	dicomPDUAssocAC = 0x02 // A-ASSOCIATE-AC.
	dicomPDUAssocRJ = 0x03 // A-ASSOCIATE-RJ.
	// DICOM item types.
	dicomItemAppContext      = 0x10 // Application Context Item.
	dicomItemPresentationCtx = 0x20 // Presentation Context Item.
	dicomItemAbstractSyntax  = 0x30 // Abstract Syntax Sub-Item.
	dicomItemTransferSyntax  = 0x40 // Transfer Syntax Sub-Item.
	dicomItemUserInfo        = 0x50 // User Information Item.
	dicomItemMaxPDULength    = 0x51 // Maximum Length Sub-Item.
	// DICOM byte values.
	dicomReservedByte          = 0x00 // Reserved byte value.
	dicomProtocolVersionMSB    = 0x00 // Protocol version (1) MSB.
	dicomProtocolVersionLSB    = 0x01 // Protocol version (1) LSB.
	dicomPresentationContextID = 0x01 // Presentation context ID.
	dicomMaxPDULengthByte      = 0x04 // Length of max PDU sub-item (4 bytes).
	dicomMaxPDUValue           = 0x40 // High byte of 16384 (0x00004000).
	// DICOM size constants.
	dicomAETitleLength       = 16 // AE Title length.
	dicomReservedBlockLength = 32 // Reserved block length in A-ASSOCIATE-RQ.
	dicomMaxPDUItemSize      = 8  // Max PDU Length Sub-Item size.
	dicomPDUHeaderSize       = 6  // PDU header size (type + reserved + 4-byte length).
	// Bit shift constants.
	dicomShiftByteHigh    = 24 // Shift for high byte in 32-bit value.
	dicomShiftByteMidHigh = 16 // Shift for mid-high byte.
	dicomShiftByteMidLow  = 8  // Shift for mid-low byte.
)

// runDICOMTests runs all configured DICOM server tests and returns results.
func (s *Server) runDICOMTests(ctx context.Context) []CustomTestResult {
	results := make([]CustomTestResult, 0, len(s.config.HealthChecks.DICOMEndpoints))
	threshold := s.config.Thresholds.CustomTests.TCP // Use TCP thresholds for DICOM

	for _, endpoint := range s.config.HealthChecks.DICOMEndpoints {
		if !endpoint.Enabled {
			continue
		}

		name := endpoint.Name
		if name == "" {
			name = net.JoinHostPort(endpoint.Host, strconv.Itoa(endpoint.Port))
		}

		testResult := CustomTestResult{Name: name, Host: endpoint.Host, Port: endpoint.Port}
		latency, err := runDICOMTest(ctx, endpoint.Host, endpoint.Port, endpoint.CalledAE, endpoint.CallingAE)

		if err != nil {
			testResult.Success = false
			testResult.Error = "DICOM test failed: " + err.Error()
			testResult.TestStatus = statusError
		} else {
			testResult.Success = true
			testResult.Latency = latency
			warningMs := threshold.Warning.Milliseconds()
			criticalMs := threshold.Critical.Milliseconds()
			testResult.TestStatus = getTestStatus(latency, warningMs, criticalMs)
		}
		results = append(results, testResult)
	}
	return results
}

// runDICOMTest runs a DICOM C-ECHO (association request) and returns latency in ms.
// This tests the DICOM server's ability to accept associations (like a ping).
func runDICOMTest(ctx context.Context, host string, port int, calledAE, callingAE string) (float64, error) {
	if port == 0 {
		port = dicomDefaultPort
	}
	if calledAE == "" {
		calledAE = "ANY-SCP"
	}
	if callingAE == "" {
		callingAE = "SEED-SCU"
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))

	ctx, cancel := context.WithTimeout(ctx, dicomTestTimeoutSec*time.Second)
	defer cancel()

	start := time.Now()

	// Connect via TCP
	dialer := net.Dialer{Timeout: dicomTestTimeoutSec * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("connection failed: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Set deadline for the entire exchange
	if deadlineErr := conn.SetDeadline(time.Now().Add(dicomTestTimeoutSec * time.Second)); deadlineErr != nil {
		return 0, deadlineErr
	}

	// Build A-ASSOCIATE-RQ PDU
	pdu := buildDICOMAssociateRQ(calledAE, callingAE)

	// Send A-ASSOCIATE-RQ
	_, err = conn.Write(pdu)
	if err != nil {
		return 0, fmt.Errorf("failed to send A-ASSOCIATE-RQ: %w", err)
	}

	// Read response PDU header
	header := make([]byte, dicomPDUHeaderSize)
	_, err = conn.Read(header)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	latency := time.Since(start).Seconds() * millisecondsPerSecond

	// Check PDU type
	pduType := header[0]
	switch pduType {
	case dicomPDUAssocAC:
		// Association accepted - server is healthy
		return latency, nil
	case dicomPDUAssocRJ:
		// Association rejected - but server responded, so it's reachable
		// This still means the DICOM server is running
		return latency, nil
	default:
		return latency, fmt.Errorf("unexpected PDU type: 0x%02x", pduType)
	}
}

// buildDICOMAssociateRQ builds a minimal DICOM A-ASSOCIATE-RQ PDU.
// This is a simplified version that requests Verification SOP Class (C-ECHO).
func buildDICOMAssociateRQ(calledAE, callingAE string) []byte {
	// Pad AE titles to 16 chars
	calledAE = padAETitle(calledAE)
	callingAE = padAETitle(callingAE)

	// Verification SOP Class UID (1.2.840.10008.1.1)
	verificationUID := "1.2.840.10008.1.1"
	// Implicit VR Little Endian Transfer Syntax UID (1.2.840.10008.1.2)
	implicitVRUID := "1.2.840.10008.1.2"

	// Build Application Context Item
	appContextUID := "1.2.840.10008.3.1.1.1"
	appContextItem := buildDICOMItem(dicomItemAppContext, []byte(appContextUID))

	// Build Abstract Syntax Sub-Item
	abstractSyntaxItem := buildDICOMItem(dicomItemAbstractSyntax, []byte(verificationUID))

	// Build Transfer Syntax Sub-Item
	transferSyntaxItem := buildDICOMItem(dicomItemTransferSyntax, []byte(implicitVRUID))

	// Build Presentation Context Item
	pcContent := make([]byte, 0)
	pcContent = append(pcContent, dicomPresentationContextID) // Presentation Context ID
	pcContent = append(pcContent, dicomReservedByte)          // Reserved
	pcContent = append(pcContent, dicomReservedByte)          // Reserved
	pcContent = append(pcContent, dicomReservedByte)          // Reserved
	pcContent = append(pcContent, abstractSyntaxItem...)
	pcContent = append(pcContent, transferSyntaxItem...)
	presentationContextItem := buildDICOMItem(dicomItemPresentationCtx, pcContent)

	// Build User Information Item with Max PDU Length Sub-Item
	maxPDULength := make([]byte, dicomMaxPDUItemSize)
	maxPDULength[0] = dicomItemMaxPDULength // Item type
	maxPDULength[1] = dicomReservedByte     // Reserved
	maxPDULength[2] = dicomReservedByte     // Length MSB
	maxPDULength[3] = dicomMaxPDULengthByte // Length LSB (4 bytes)
	maxPDULength[4] = dicomReservedByte     // Max PDU length (16384 = 0x00004000)
	maxPDULength[5] = dicomReservedByte     // ...
	maxPDULength[6] = dicomMaxPDUValue      // ...
	maxPDULength[7] = dicomReservedByte     // ...
	userInfoItem := buildDICOMItem(dicomItemUserInfo, maxPDULength)

	// Build PDU content
	pduContent := make([]byte, 0)
	pduContent = append(pduContent, dicomProtocolVersionMSB, dicomProtocolVersionLSB) // Protocol Version
	pduContent = append(pduContent, dicomReservedByte, dicomReservedByte)             // Reserved
	pduContent = append(pduContent, []byte(calledAE)...)
	pduContent = append(pduContent, []byte(callingAE)...)
	pduContent = append(pduContent, make([]byte, dicomReservedBlockLength)...) // Reserved (32 bytes)
	pduContent = append(pduContent, appContextItem...)
	pduContent = append(pduContent, presentationContextItem...)
	pduContent = append(pduContent, userInfoItem...)

	// Build full PDU
	pdu := make([]byte, 0)
	pdu = append(pdu, dicomPDUAssocRQ)   // PDU Type
	pdu = append(pdu, dicomReservedByte) // Reserved
	// PDU Length (4 bytes, big endian)
	pduLen := len(pduContent)
	pdu = append(pdu, byte(pduLen>>dicomShiftByteHigh), byte(pduLen>>dicomShiftByteMidHigh),
		byte(pduLen>>dicomShiftByteMidLow), byte(pduLen))
	pdu = append(pdu, pduContent...)

	return pdu
}

// buildDICOMItem builds a DICOM item with type, reserved, length, and content.
func buildDICOMItem(itemType byte, content []byte) []byte {
	item := make([]byte, 0)
	item = append(item, itemType)
	item = append(item, dicomReservedByte) // Reserved
	// Length (2 bytes, big endian)
	length := len(content)
	item = append(item, byte(length>>dicomShiftByteMidLow), byte(length))
	item = append(item, content...)
	return item
}

// padAETitle pads an AE title to dicomAETitleLength characters with spaces.
func padAETitle(ae string) string {
	if len(ae) > dicomAETitleLength {
		return ae[:dicomAETitleLength]
	}
	return ae + strings.Repeat(" ", dicomAETitleLength-len(ae))
}

// ============================================================================
// Utility Functions
// ============================================================================

// getTestStatus returns status based on latency and thresholds.
func getTestStatus(latencyMs float64, warningMs, criticalMs int64) string {
	if latencyMs < float64(warningMs) {
		return statusSuccess
	}
	if latencyMs < float64(criticalMs) {
		return statusWarning
	}
	return statusError
}
