package shell

import (
	"context"
	"fmt"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/dhcp"
	"github.com/krisarmstrong/seed/internal/discovery"
)

// DefaultInterface is the default network interface to use when none is configured.
const DefaultInterface = "eth0"

// DiscoveryService handles network device discovery.
type DiscoveryService struct {
	cfg             *config.Config
	db              *database.DB
	service         *discovery.Service
	deviceDiscovery *discovery.DeviceDiscovery
	cancel          context.CancelFunc
}

// NewDiscoveryService creates a new discovery service.
func NewDiscoveryService(cfg *config.Config, db *database.DB) *DiscoveryService {
	iface, ok := cfg.GetActiveInterface()
	if !ok || iface == "" {
		iface = DefaultInterface
	}

	return &DiscoveryService{
		cfg:             cfg,
		db:              db,
		service:         discovery.NewService(cfg, iface, nil),
		deviceDiscovery: discovery.NewDeviceDiscoveryWithOUI(iface, cfg.NetworkDiscovery.OUIFilePath, cfg.NetworkDiscovery.OUIMaxAge),
	}
}

// Start begins background device discovery.
func (s *DiscoveryService) Start(ctx context.Context) error {
	_, s.cancel = context.WithCancel(ctx)

	if err := s.service.Start(); err != nil {
		return fmt.Errorf("starting discovery service: %w", err)
	}

	return nil
}

// Stop halts device discovery.
func (s *DiscoveryService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.service != nil {
		s.service.Stop()
	}
	if s.deviceDiscovery != nil {
		s.deviceDiscovery.Stop()
	}
}

// Discover performs an on-demand discovery scan.
func (s *DiscoveryService) Discover(ctx context.Context, _ *DiscoveryOptions) (*DiscoveryResult, error) {
	if s.deviceDiscovery == nil {
		return nil, ErrNotInitialized
	}

	startTime := time.Now()

	// Perform scan
	if err := s.deviceDiscovery.Scan(ctx); err != nil {
		return nil, fmt.Errorf("running discovery scan: %w", err)
	}

	// Get discovered devices
	discovered := s.deviceDiscovery.GetDevices()

	// Convert to shell.Device type
	devices := make([]Device, 0, len(discovered))
	for _, d := range discovered {
		devices = append(devices, convertDiscoveredDevice(d))
	}

	return &DiscoveryResult{
		Devices:      devices,
		NewDevices:   len(devices), // TODO: Track delta from previous scans
		ScanDuration: time.Since(startTime),
		StartedAt:    startTime,
		CompletedAt:  time.Now(),
	}, nil
}

// GetDevices returns all discovered devices.
func (s *DiscoveryService) GetDevices(_ context.Context) ([]Device, error) {
	if s.deviceDiscovery == nil {
		return nil, ErrNotInitialized
	}

	discovered := s.deviceDiscovery.GetDevices()
	devices := make([]Device, 0, len(discovered))
	for _, d := range discovered {
		devices = append(devices, convertDiscoveredDevice(d))
	}

	return devices, nil
}

// GetDevice returns a device by ID.
func (s *DiscoveryService) GetDevice(_ context.Context, id string) (*Device, error) {
	if s.deviceDiscovery == nil {
		return nil, ErrNotInitialized
	}

	discovered := s.deviceDiscovery.GetDevices()
	for _, d := range discovered {
		if d.MAC == id || d.IP == id {
			device := convertDiscoveredDevice(d)
			return &device, nil
		}
	}

	return nil, fmt.Errorf("device not found: %s", id)
}

// VulnerabilityService handles vulnerability scanning.
type VulnerabilityService struct {
	cfg     *config.Config
	db      *database.DB
	scanner *discovery.VulnerabilityScanner
	cancel  context.CancelFunc
}

// NewVulnerabilityService creates a new vulnerability service.
func NewVulnerabilityService(cfg *config.Config, db *database.DB) *VulnerabilityService {
	// Configure vulnerability scanner from Security config
	vulnCfg := cfg.Security.VulnerabilityScanning
	scannerCfg := &discovery.VulnerabilityScannerConfig{
		Enabled:           vulnCfg.Enabled,
		CVEDatabase:       vulnCfg.CVEDatabase,
		NVDAPIKey:         vulnCfg.NVDAPIKey,
		UpdateInterval:    vulnCfg.UpdateInterval,
		SeverityThreshold: vulnCfg.SeverityThreshold,
		MaxConcurrent:     vulnCfg.MaxConcurrent,
		KEVEnabled:        true, // Enable CISA KEV enrichment by default
	}

	scanner, err := discovery.NewVulnerabilityScanner(scannerCfg)
	if err != nil {
		// Log error but don't fail - scanner will be nil and methods will return ErrNotInitialized
		scanner = nil
	}

	return &VulnerabilityService{
		cfg:     cfg,
		db:      db,
		scanner: scanner,
	}
}

// Stop halts any running scans.
func (s *VulnerabilityService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.scanner != nil {
		_ = s.scanner.Stop() // Ignore error on cleanup
	}
}

// Scan performs a vulnerability scan on specified targets.
func (s *VulnerabilityService) Scan(ctx context.Context, targets []string) (*VulnerabilityScan, error) {
	if s.scanner == nil {
		return nil, ErrNotInitialized
	}

	startTime := time.Now()

	// Scan each target by creating minimal device objects
	var allVulns []Vulnerability
	for _, target := range targets {
		// Create a minimal device for scanning
		device := &discovery.DiscoveredDevice{
			IP: target,
		}
		results, err := s.scanner.ScanDevice(ctx, device)
		if err != nil {
			continue // Skip failed scans
		}
		for _, v := range results.Vulnerabilities {
			allVulns = append(allVulns, convertVulnerability(&v))
		}
	}

	// Count by severity
	var critical, high, medium, low int
	for _, v := range allVulns {
		switch v.Severity {
		case SeverityCritical:
			critical++
		case SeverityHigh:
			high++
		case SeverityMedium:
			medium++
		case SeverityLow:
			low++
		case SeverityInfo:
			// Info severity not counted in totals
		}
	}

	return &VulnerabilityScan{
		ID:              fmt.Sprintf("scan-%d", time.Now().UnixNano()),
		Vulnerabilities: allVulns,
		DevicesScanned:  len(targets),
		TotalCritical:   critical,
		TotalHigh:       high,
		TotalMedium:     medium,
		TotalLow:        low,
		ScanDuration:    time.Since(startTime),
		StartedAt:       startTime,
		CompletedAt:     time.Now(),
	}, nil
}

// GetVulnerabilities returns all discovered vulnerabilities.
func (s *VulnerabilityService) GetVulnerabilities(_ context.Context) ([]Vulnerability, error) {
	if s.scanner == nil {
		return nil, ErrNotInitialized
	}

	results := s.scanner.GetAllVulnerabilities()
	var allVulns []Vulnerability
	for _, r := range results {
		for _, v := range r.Vulnerabilities {
			allVulns = append(allVulns, convertVulnerability(&v))
		}
	}

	return allVulns, nil
}

// GetDeviceVulnerabilities returns vulnerabilities for a specific device.
func (s *VulnerabilityService) GetDeviceVulnerabilities(_ context.Context, deviceIP string) ([]Vulnerability, error) {
	if s.scanner == nil {
		return nil, ErrNotInitialized
	}

	result := s.scanner.GetDeviceVulnerabilities(deviceIP)
	if result == nil {
		return nil, nil
	}

	vulns := make([]Vulnerability, 0, len(result.Vulnerabilities))
	for _, v := range result.Vulnerabilities {
		vulns = append(vulns, convertVulnerability(&v))
	}

	return vulns, nil
}

// UpdateStatus updates a vulnerability's status.
func (s *VulnerabilityService) UpdateStatus(_ context.Context, _ string, _ VulnStatus) error {
	// TODO: Implement when database storage is available
	return ErrNotImplemented
}

// PostureService assesses overall security posture.
type PostureService struct {
	cfg           *config.Config
	db            *database.DB
	discovery     *DiscoveryService
	vulnerability *VulnerabilityService
}

// NewPostureService creates a new posture service.
func NewPostureService(cfg *config.Config, db *database.DB, discovery *DiscoveryService, vuln *VulnerabilityService) *PostureService {
	return &PostureService{
		cfg:           cfg,
		db:            db,
		discovery:     discovery,
		vulnerability: vuln,
	}
}

// Assess performs a security posture assessment.
func (s *PostureService) Assess(ctx context.Context) (*PostureScore, error) {
	score := &PostureScore{
		Overall:    100,
		Categories: make(map[string]int),
		Issues:     make([]PostureIssue, 0),
		AssessedAt: time.Now(),
	}

	// Get vulnerability data
	if s.vulnerability != nil {
		vulns, err := s.vulnerability.GetVulnerabilities(ctx)
		if err == nil {
			// Deduct points for vulnerabilities
			for _, v := range vulns {
				switch v.Severity {
				case SeverityCritical:
					score.Overall -= 20
					score.Issues = append(score.Issues, PostureIssue{
						Category:    "vulnerabilities",
						Severity:    "critical",
						Description: fmt.Sprintf("Critical vulnerability: %s", v.CVEID),
						Remediation: v.Remediation,
					})
				case SeverityHigh:
					score.Overall -= 10
				case SeverityMedium:
					score.Overall -= 5
				case SeverityLow:
					score.Overall -= 2
				case SeverityInfo:
					// Info severity doesn't affect score
				}
			}
			score.Categories["vulnerabilities"] = max(0, 100-len(vulns)*5)
		}
	}

	// Ensure score doesn't go below 0
	if score.Overall < 0 {
		score.Overall = 0
	}

	return score, nil
}

// RogueService detects unauthorized devices.
type RogueService struct {
	cfg      *config.Config
	detector *dhcp.RogueDetector
	cancel   context.CancelFunc
}

// NewRogueService creates a new rogue detection service.
func NewRogueService(cfg *config.Config) *RogueService {
	iface, ok := cfg.GetActiveInterface()
	if !ok || iface == "" {
		iface = DefaultInterface
	}

	detectorCfg := &dhcp.RogueDetectorConfig{
		Interface:        iface,
		KnownServers:     cfg.DHCP.RogueDetection.KnownServers,
		AlertOnDetection: cfg.DHCP.RogueDetection.AlertOnDetection,
	}

	return &RogueService{
		cfg:      cfg,
		detector: dhcp.NewRogueDetector(detectorCfg),
	}
}

// Start begins rogue device detection.
func (s *RogueService) Start(ctx context.Context) error {
	_, s.cancel = context.WithCancel(ctx)

	if err := s.detector.Start(); err != nil {
		return fmt.Errorf("starting rogue detector: %w", err)
	}

	return nil
}

// Stop halts rogue detection.
func (s *RogueService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.detector != nil {
		_ = s.detector.Stop() // Ignore error on cleanup
	}
}

// GetRogueDevices returns all detected rogue devices.
func (s *RogueService) GetRogueDevices(_ context.Context) ([]RogueDevice, error) {
	if s.detector == nil {
		return nil, ErrNotInitialized
	}

	rogues := s.detector.GetRogueServers()
	devices := make([]RogueDevice, 0, len(rogues))
	for _, r := range rogues {
		devices = append(devices, RogueDevice{
			Device: Device{
				ID:         r.IP,
				MACAddress: r.MAC,
				FirstSeen:  r.FirstSeen,
				LastSeen:   r.LastSeen,
			},
			Reason:     fmt.Sprintf("Unauthorized DHCP server (%d offers detected)", r.OfferCount),
			RiskLevel:  "high",
			DetectedAt: r.FirstSeen,
		})
	}

	return devices, nil
}

// GetAlerts returns rogue device alerts.
func (s *RogueService) GetAlerts(_ context.Context) ([]RogueAlert, error) {
	if s.detector == nil {
		return nil, ErrNotInitialized
	}

	rogues := s.detector.GetRogueServers()
	alerts := make([]RogueAlert, 0, len(rogues))
	for i, r := range rogues {
		alerts = append(alerts, RogueAlert{
			ID: fmt.Sprintf("alert-%d", i),
			Device: RogueDevice{
				Device: Device{
					ID:        r.IP,
					FirstSeen: r.FirstSeen,
					LastSeen:  r.LastSeen,
				},
				Reason:     "Unauthorized DHCP server",
				RiskLevel:  "high",
				DetectedAt: r.FirstSeen,
			},
			AlertType: "rogue_dhcp",
			Message:   fmt.Sprintf("Rogue DHCP server detected at %s", r.IP),
			CreatedAt: r.FirstSeen,
		})
	}

	return alerts, nil
}

// AcknowledgeDevice marks a device as acknowledged.
func (s *RogueService) AcknowledgeDevice(_ context.Context, _ string) error {
	// TODO: Implement acknowledgement storage
	return ErrNotImplemented
}

// Helper functions

func convertDiscoveredDevice(d *discovery.DiscoveredDevice) Device {
	var services []Service

	// Extract services from Profile if available
	if d.Profile != nil {
		for _, port := range d.Profile.OpenPorts {
			services = append(services, Service{
				Port:     port.Port,
				Protocol: port.Protocol,
				Name:     port.Service,
				Banner:   port.Banner,
				State:    "open",
			})
		}
	}

	deviceType := DeviceTypeUnknown
	switch {
	case d.IsRouter:
		deviceType = DeviceTypeRouter
	case d.Profile != nil && len(d.Profile.OpenPorts) > 5:
		deviceType = DeviceTypeServer
	}

	// If profile has a device type, use it
	if d.Profile != nil && d.Profile.DeviceType != "" {
		deviceType = DeviceType(d.Profile.DeviceType)
	}

	return Device{
		ID:         d.MAC,
		MACAddress: d.MAC,
		Hostname:   d.Hostname,
		Vendor:     d.Vendor,
		DeviceType: deviceType,
		Services:   services,
		FirstSeen:  d.LastSeen, // Use LastSeen as approximation (no FirstSeen in DiscoveredDevice)
		LastSeen:   d.LastSeen,
		IsOnline:   true, // Assume online if in discovery results
		IsGateway:  d.IsRouter,
		Metadata:   make(map[string]string),
	}
}

func convertVulnerability(v *discovery.Vulnerability) Vulnerability {
	severity := SeverityInfo
	switch v.Severity {
	case "CRITICAL":
		severity = SeverityCritical
	case "HIGH":
		severity = SeverityHigh
	case "MEDIUM":
		severity = SeverityMedium
	case "LOW":
		severity = SeverityLow
	}

	return Vulnerability{
		ID:          v.CVEID,
		CVEID:       v.CVEID,
		Title:       v.Description,
		Description: v.Description,
		Severity:    severity,
		CVSSScore:   v.Score,
		References:  v.References,
		IsKEV:       v.ActivelyExploited,
		IsExploited: v.ActivelyExploited,
		Status:      VulnStatusNew,
	}
}

// Service returns the underlying discovery.Service for dependency injection.
func (s *DiscoveryService) Service() *discovery.Service {
	return s.service
}

// DeviceDiscovery returns the underlying DeviceDiscovery for dependency injection.
func (s *DiscoveryService) DeviceDiscovery() *discovery.DeviceDiscovery {
	return s.deviceDiscovery
}

// Scanner returns the underlying vulnerability scanner for dependency injection.
func (s *VulnerabilityService) Scanner() *discovery.VulnerabilityScanner {
	return s.scanner
}

// Detector returns the underlying rogue detector for dependency injection.
func (s *RogueService) Detector() *dhcp.RogueDetector {
	return s.detector
}
