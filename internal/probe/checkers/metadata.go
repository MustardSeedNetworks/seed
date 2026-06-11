package checkers

// metadata.go defines typed views of the probe Result.Metadata that the
// on-demand /run mapping (internal/api/healthcheckrun.go) unmarshals to
// surface protocol detail on the health-check card. The metadata keys are
// the probe layer's internal snake_case format (each checker marshals them
// into Result.Metadata); co-locating these read-types with the checkers
// keeps that contract owned by the package that produces it, rather than
// re-declaring snake_case-tagged structs in the API layer.
//
// These are read-views: a checker may emit more keys than the corresponding
// type reads (extra keys are informational and preserved in the persisted
// metadata_json). Adding a rendered field means adding it here and in the
// emitting checker.

// HTTPRunMetadata is the http/https checker metadata the card renders: the
// status code, the per-phase timing breakdown, and (for https) the leaf-cert
// summary.
type HTTPRunMetadata struct {
	StatusCode int `json:"status_code"`
	Timings    struct {
		DNS  float64 `json:"dns"`
		TCP  float64 `json:"tcp"`
		TLS  float64 `json:"tls"`
		TTFB float64 `json:"ttfb"`
	} `json:"timings_ms"`
	TLS *struct {
		Issuer        string `json:"issuer"`
		NotAfter      string `json:"not_after"`
		DaysRemaining int    `json:"days_remaining"`
		TLSVersion    string `json:"tls_version"`
	} `json:"tls"`
}

// HL7RunMetadata is the HL7 checker metadata the card renders.
type HL7RunMetadata struct {
	AckCode string `json:"ack_code"`
}

// FHIRRunMetadata is the FHIR checker metadata the card renders.
type FHIRRunMetadata struct {
	FHIRVersion   string `json:"fhir_version"`
	ServerName    string `json:"server_name"`
	ResourceCount int    `json:"resource_count"`
}

// LTIRunMetadata is the LTI checker metadata the card renders.
type LTIRunMetadata struct {
	LTIVersion string `json:"lti_version"`
}

// OPCUARunMetadata is the OPC-UA checker metadata the card renders.
type OPCUARunMetadata struct {
	SecurityMode string `json:"security_mode"`
}

// ModbusRunMetadata is the Modbus checker metadata the card renders.
type ModbusRunMetadata struct {
	RegisterValue int `json:"register_value"`
}

// SQLRunMetadata is the SQL checker metadata the card renders.
type SQLRunMetadata struct {
	ServerVersion string `json:"server_version"`
}
