package discovery

// pipeline_types.go contains the types and per-event-payload structs that
// drive the discovery pipeline: PipelineState/EventType enums, all phase
// payloads broadcast over the WebSocket, the PipelineConfig tree, plus the
// Phase, EventBroadcaster, and ConfigPipelineAdapter interfaces.

import (
	"context"
	"time"
)

// Pipeline phase-name string constants. These match the human-readable phase
// keys used in WebSocket payloads, log entries, and PhaseDurations maps.
const (
	phaseNameEnumeration = "enumeration"
	phaseNameResolution  = "resolution"
	phaseNameScanning    = "scanning"
	phaseNameAssessment  = "assessment"
)

// PipelineState represents the current state of the discovery pipeline.
type PipelineState string

// Pipeline state constants.
const (
	PipelineStateIdle        PipelineState = "idle"
	PipelineStateEnumerating PipelineState = "enumerating"
	PipelineStateResolving   PipelineState = "resolving"
	PipelineStateScanning    PipelineState = "scanning"
	PipelineStateAssessing   PipelineState = "assessing"
	PipelineStateComplete    PipelineState = "complete"
	PipelineStateFailed      PipelineState = "failed"
	PipelineStateCanceled    PipelineState = "canceled"
)

// PipelineEventType defines WebSocket event types for pipeline updates.
type PipelineEventType string

// Pipeline event type constants for WebSocket broadcasting.
const (
	EventPipelineStarted   PipelineEventType = "pipeline_started"
	EventPhaseStarted      PipelineEventType = "phase_started"
	EventPhaseProgress     PipelineEventType = "phase_progress"
	EventPhaseCompleted    PipelineEventType = "phase_completed"
	EventPhaseFailed       PipelineEventType = "phase_failed"
	EventDeviceDiscovered  PipelineEventType = "device_discovered"
	EventDeviceUpdated     PipelineEventType = "device_updated"
	EventPipelineCompleted PipelineEventType = "pipeline_completed"
	EventPipelineFailed    PipelineEventType = "pipeline_failed"
	EventPipelineCanceled  PipelineEventType = "pipeline_canceled"
)

// PipelineEvent is the WebSocket message for pipeline updates.
type PipelineEvent struct {
	Type      PipelineEventType `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	RunID     string            `json:"runId"`
	Payload   any               `json:"payload"`
}

// PipelineStartedPayload for EventPipelineStarted.
type PipelineStartedPayload struct {
	TotalPhases int      `json:"totalPhases"`
	Phases      []string `json:"phases"`
}

// PhaseStartedPayload for EventPhaseStarted.
type PhaseStartedPayload struct {
	Phase       string `json:"phase"`
	PhaseNumber int    `json:"phaseNumber"`
	TotalPhases int    `json:"totalPhases"`
	DeviceCount int    `json:"deviceCount"`
}

// PhaseProgressPayload for EventPhaseProgress.
type PhaseProgressPayload struct {
	Phase             string  `json:"phase"`
	ProcessedCount    int     `json:"processedCount"`
	TotalCount        int     `json:"totalCount"`
	PercentComplete   float64 `json:"percentComplete"`
	CurrentTarget     string  `json:"currentTarget,omitempty"`
	ElapsedMs         int64   `json:"elapsedMs"`
	EstimatedRemainMs int64   `json:"estimatedRemainMs,omitempty"`
}

// PhaseCompletedPayload for EventPhaseCompleted.
type PhaseCompletedPayload struct {
	Phase         string        `json:"phase"`
	DevicesFound  int           `json:"devicesFound,omitempty"`
	NamesResolved int           `json:"namesResolved,omitempty"`
	PortsOpen     int           `json:"portsOpen,omitempty"`
	VulnsFound    int           `json:"vulnsFound,omitempty"`
	Duration      time.Duration `json:"duration"`
	Errors        []string      `json:"errors,omitempty"`
}

// DeviceDiscoveredPayload for EventDeviceDiscovered.
type DeviceDiscoveredPayload struct {
	IP      string   `json:"ip"`
	MAC     string   `json:"mac,omitempty"`
	Vendor  string   `json:"vendor,omitempty"`
	Methods []string `json:"methods"`
	IsNew   bool     `json:"isNew"`
}

// PipelineCompletedPayload for EventPipelineCompleted.
type PipelineCompletedPayload struct {
	TotalDevices   int                      `json:"totalDevices"`
	NewDevices     int                      `json:"newDevices"`
	UpdatedDevices int                      `json:"updatedDevices"`
	StaleDevices   int                      `json:"staleDevices"`
	TotalDuration  time.Duration            `json:"totalDuration"`
	PhaseDurations map[string]time.Duration `json:"phaseDurations"`
}

// PortScanIntensity defines port scanning levels.
type PortScanIntensity string

// Port scan intensity level constants.
const (
	PortScanOff           PortScanIntensity = "off"
	PortScanQuick         PortScanIntensity = "quick"
	PortScanStandard      PortScanIntensity = "standard"
	PortScanComprehensive PortScanIntensity = "comprehensive"
	PortScanCustom        PortScanIntensity = "custom"
)

// ScanTimingProfile defines pre-configured timing settings.
type ScanTimingProfile string

// Scan timing profile constants for IDS-aware scanning.
const (
	ScanProfilePolite     ScanTimingProfile = "polite"
	ScanProfileNormal     ScanTimingProfile = "normal"
	ScanProfileAggressive ScanTimingProfile = "aggressive"
)

// PipelineConfig controls the sequential discovery pipeline.
type PipelineConfig struct {
	// Phases controls which pipeline phases are enabled.
	Phases PipelinePhaseConfig `yaml:"phases" json:"phases"`

	// Timing controls rate limiting and delays.
	Timing PipelineTiming `yaml:"timing" json:"timing"`

	// PortScan controls port scanning behavior and intensity.
	PortScan PipelinePortScanConfig `yaml:"port_scan" json:"portScan"`

	// SNMPCollection controls extended SNMP MIB collection.
	SNMPCollection SNMPCollectionConfig `yaml:"snmp_collection" json:"snmpCollection"`

	// Resolution controls name resolution methods.
	Resolution PipelineResolutionConfig `yaml:"resolution" json:"resolution"`

	// Persistence controls how results are stored.
	Persistence PipelinePersistenceConfig `yaml:"persistence" json:"persistence"`
}

// PipelineResolutionConfig controls Phase 2 name resolution methods.
type PipelineResolutionConfig struct {
	// DNS enables reverse DNS (PTR) lookups.
	DNS bool `yaml:"dns" json:"dns"`

	// NetBIOS enables NetBIOS name resolution for Windows devices.
	NetBIOS bool `yaml:"netbios" json:"netbios"`

	// MDNS enables mDNS name resolution for Apple/Linux devices.
	MDNS bool `yaml:"mdns" json:"mdns"`
}

// PipelinePhaseConfig controls which phases are executed.
type PipelinePhaseConfig struct {
	Enumeration      bool `yaml:"enumeration"       json:"enumeration"`      // Always true - core functionality
	NameResolution   bool `yaml:"name_resolution"   json:"nameResolution"`   // Default: true
	ServiceDiscovery bool `yaml:"service_discovery" json:"serviceDiscovery"` // Default: false (passive only)
	VulnAssessment   bool `yaml:"vuln_assessment"   json:"vulnAssessment"`   // Default: false
}

// PipelineTiming controls scan rate limiting.
type PipelineTiming struct {
	// ProbeDelay is the minimum time between probes to a single host.
	ProbeDelay time.Duration `yaml:"probe_delay" json:"probeDelay"`

	// HostDelay is the minimum time between starting scans of different hosts.
	HostDelay time.Duration `yaml:"host_delay" json:"hostDelay"`

	// MaxConcurrentHosts limits parallel host scanning.
	MaxConcurrentHosts int `yaml:"max_concurrent_hosts" json:"maxConcurrentHosts"`

	// PhaseTimeout is the max duration for any single phase.
	PhaseTimeout time.Duration `yaml:"phase_timeout" json:"phaseTimeout"`

	// Profile selects a pre-defined timing profile (overrides individual settings).
	Profile ScanTimingProfile `yaml:"profile" json:"profile"`
}

// PipelinePortScanConfig controls port scanning intensity.
type PipelinePortScanConfig struct {
	// Intensity controls which ports are scanned.
	Intensity PortScanIntensity `yaml:"intensity" json:"intensity"`

	// CustomPorts for Intensity="custom".
	CustomPorts []int `yaml:"custom_ports,omitempty" json:"customPorts,omitempty"`

	// BannerGrab enables service banner reading.
	BannerGrab bool `yaml:"banner_grab" json:"bannerGrab"`

	// ConnectTimeout for port connections.
	ConnectTimeout time.Duration `yaml:"connect_timeout" json:"connectTimeout"`
}

// SNMPCollectionConfig controls extended SNMP data collection.
type SNMPCollectionConfig struct {
	// Enabled turns on extended SNMP collection in Phase 3.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// MIBs specifies which MIB groups to collect.
	MIBs SNMPMIBSelection `yaml:"mibs" json:"mibs"`

	// WalkTimeout per MIB walk operation.
	WalkTimeout time.Duration `yaml:"walk_timeout" json:"walkTimeout"`

	// MaxOIDsPerRequest for bulk requests.
	MaxOIDsPerRequest int `yaml:"max_oids_per_request" json:"maxOidsPerRequest"`
}

// SNMPMIBSelection controls which MIBs are collected.
type SNMPMIBSelection struct {
	System      bool `yaml:"system"       json:"system"`      // SNMPv2-MIB::system (always on)
	Interfaces  bool `yaml:"interfaces"   json:"interfaces"`  // IF-MIB (ifTable, ifXTable)
	IPAddresses bool `yaml:"ip_addresses" json:"ipAddresses"` // IP-MIB (ipAddrTable)
	Routing     bool `yaml:"routing"      json:"routing"`     // IP-FORWARD-MIB
	Bridge      bool `yaml:"bridge"       json:"bridge"`      // BRIDGE-MIB (MAC table)
	Entity      bool `yaml:"entity"       json:"entity"`      // ENTITY-MIB (physical inventory)
	LLDP        bool `yaml:"lldp"         json:"lldp"`        // LLDP-MIB
	VLAN        bool `yaml:"vlan"         json:"vlan"`        // Q-BRIDGE-MIB
}

// PipelinePersistenceConfig controls database storage.
type PipelinePersistenceConfig struct {
	// StoreHistory keeps historical device state.
	StoreHistory bool `yaml:"store_history" json:"storeHistory"`

	// StalenessThreshold marks devices inactive after this duration.
	StalenessThreshold time.Duration `yaml:"staleness_threshold" json:"stalenessThreshold"`

	// PurgeAfter removes inactive devices after this duration.
	PurgeAfter time.Duration `yaml:"purge_after" json:"purgeAfter"`
}

// PipelineRun represents a single execution of the discovery pipeline.
type PipelineRun struct {
	ID             string                   `json:"id"`
	StartedAt      time.Time                `json:"startedAt"`
	CompletedAt    *time.Time               `json:"completedAt,omitempty"`
	Status         PipelineState            `json:"status"`
	Trigger        string                   `json:"trigger"` // manual, scheduled, startup, api
	Config         PipelineConfig           `json:"config"`
	CurrentPhase   string                   `json:"currentPhase,omitempty"`
	PhaseDurations map[string]time.Duration `json:"phaseDurations,omitempty"`
	DevicesFound   int                      `json:"devicesFound"`
	Errors         []string                 `json:"errors,omitempty"`
}

// Phase represents a single phase in the discovery pipeline.
type Phase interface {
	// Name returns the phase name.
	Name() string

	// Run executes the phase with the given devices from the previous phase.
	// Returns updated devices and any errors encountered.
	Run(
		ctx context.Context,
		devices []*DiscoveredDevice,
		progressCh chan<- PhaseProgressPayload,
	) ([]*DiscoveredDevice, error)
}

// EventBroadcaster is an interface for broadcasting pipeline events.
type EventBroadcaster interface {
	BroadcastPipelineEvent(event PipelineEvent)
}

// ConfigPipelineAdapter is an interface for adapting configuration to PipelineConfig.
// Implemented by config.PipelineConfig.
type ConfigPipelineAdapter interface {
	GetPhases() (enumeration, nameResolution, serviceDiscovery, vulnAssessment bool)
	GetTiming() (probeDelay, hostDelay, phaseTimeout time.Duration, maxConcurrentHosts int, profile string)
	GetPortScan() (intensity string, customPorts []int, bannerGrab bool, connectTimeout time.Duration)
	GetSNMP() (enabled, system, interfaces, ipAddresses, routing, bridge, entity, lldp, vlan bool, walkTimeout time.Duration, maxOIDsPerRequest int)
	GetPersistence() (storeHistory bool, stalenessThreshold, purgeAfter time.Duration)
}
