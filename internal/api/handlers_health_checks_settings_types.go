package api

// handlers_health_checks_settings_types.go contains the request/response
// types for the health-checks settings endpoint. The handler logic and
// per-type builders live in handlers_health_checks_settings.go.

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
}

// FHIREndpointResponse contains a FHIR R4 endpoint configuration (Health Checks 100x).
type FHIREndpointResponse struct {
	Name     string `json:"name"`
	BaseURL  string `json:"baseUrl"`
	AuthType string `json:"authType"`
	Enabled  bool   `json:"enabled"`
}

// SQLEndpointResponse contains a SQL database endpoint configuration (Health Checks 100x).
type SQLEndpointResponse struct {
	Name     string `json:"name"`
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	SSLMode  string `json:"sslMode,omitempty"`
	Enabled  bool   `json:"enabled"`
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
}

// LTIEndpointResponse contains an LTI/LMS endpoint configuration (Health Checks 100x - Education).
type LTIEndpointResponse struct {
	Name       string `json:"name"`
	LaunchURL  string `json:"launchUrl"`
	LTIVersion string `json:"ltiVersion,omitempty"`
	Enabled    bool   `json:"enabled"`
}

// OPCUAEndpointResponse contains an OPC-UA endpoint configuration (Health Checks 100x - Manufacturing).
type OPCUAEndpointResponse struct {
	Name           string `json:"name"`
	EndpointURL    string `json:"endpointUrl"`
	SecurityMode   string `json:"securityMode,omitempty"`
	SecurityPolicy string `json:"securityPolicy,omitempty"`
	Enabled        bool   `json:"enabled"`
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
