package shell

// This file is only compiled during testing (due to _test.go suffix)
// and provides access to internal implementation details.

import (
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

// ExportConvertDiscoveredDevice exposes convertDiscoveredDevice for testing.
func ExportConvertDiscoveredDevice(d *discovery.DiscoveredDevice) Device {
	return convertDiscoveredDevice(d)
}

// ExportConvertVulnerability exposes convertVulnerability for testing.
func ExportConvertVulnerability(v *discovery.Vulnerability) Vulnerability {
	return convertVulnerability(v)
}

// ExportPerfectSecurityScore exposes perfectSecurityScore constant for testing.
const ExportPerfectSecurityScore = perfectSecurityScore

// ExportVulnerabilityPenaltyMultiplier exposes vulnerabilityPenaltyMultiplier for testing.
const ExportVulnerabilityPenaltyMultiplier = vulnerabilityPenaltyMultiplier

// DiscoveryServiceTestAccessor provides access to DiscoveryService's private fields for testing.
type DiscoveryServiceTestAccessor struct {
	Service *DiscoveryService
}

// GetCfg returns the service's config.
func (a *DiscoveryServiceTestAccessor) GetCfg() interface{} {
	return a.Service.cfg
}

// GetDB returns the service's database.
func (a *DiscoveryServiceTestAccessor) GetDB() interface{} {
	return a.Service.db
}

// VulnerabilityServiceTestAccessor provides access to VulnerabilityService's private fields.
type VulnerabilityServiceTestAccessor struct {
	Service *VulnerabilityService
}

// GetCfg returns the service's config.
func (a *VulnerabilityServiceTestAccessor) GetCfg() interface{} {
	return a.Service.cfg
}

// GetDB returns the service's database.
func (a *VulnerabilityServiceTestAccessor) GetDB() interface{} {
	return a.Service.db
}

// GetScanner returns the service's scanner.
func (a *VulnerabilityServiceTestAccessor) GetScanner() *discovery.VulnerabilityScanner {
	return a.Service.scanner
}

// PostureServiceTestAccessor provides access to PostureService's private fields for testing.
type PostureServiceTestAccessor struct {
	Service *PostureService
}

// GetCfg returns the service's config.
func (a *PostureServiceTestAccessor) GetCfg() interface{} {
	return a.Service.cfg
}

// GetDB returns the service's database.
func (a *PostureServiceTestAccessor) GetDB() interface{} {
	return a.Service.db
}

// GetDiscovery returns the service's discovery service.
func (a *PostureServiceTestAccessor) GetDiscovery() *DiscoveryService {
	return a.Service.discovery
}

// GetVulnerability returns the service's vulnerability service.
func (a *PostureServiceTestAccessor) GetVulnerability() *VulnerabilityService {
	return a.Service.vulnerability
}

// RogueServiceTestAccessor provides access to RogueService's private fields for testing.
type RogueServiceTestAccessor struct {
	Service *RogueService
}

// GetCfg returns the service's config.
func (a *RogueServiceTestAccessor) GetCfg() interface{} {
	return a.Service.cfg
}

// ModuleTestAccessor provides access to Module's private fields for testing.
type ModuleTestAccessor struct {
	Module *Module
}

// GetCfg returns the module's config.
func (a *ModuleTestAccessor) GetCfg() interface{} {
	return a.Module.cfg
}

// GetDB returns the module's database.
func (a *ModuleTestAccessor) GetDB() interface{} {
	return a.Module.db
}

// GetDiscoveryService returns the module's discovery service.
func (a *ModuleTestAccessor) GetDiscoveryService() *DiscoveryService {
	return a.Module.discovery
}

// GetVulnerabilityService returns the module's vulnerability service.
func (a *ModuleTestAccessor) GetVulnerabilityService() *VulnerabilityService {
	return a.Module.vulnerability
}

// GetPostureService returns the module's posture service.
func (a *ModuleTestAccessor) GetPostureService() *PostureService {
	return a.Module.posture
}

// GetRogueService returns the module's rogue service.
func (a *ModuleTestAccessor) GetRogueService() *RogueService {
	return a.Module.rogue
}

// DiscoveryServiceWithNilDeviceDiscovery creates a DiscoveryService with nil deviceDiscovery for testing.
func DiscoveryServiceWithNilDeviceDiscovery() *DiscoveryService {
	return &DiscoveryService{
		cfg:             nil,
		db:              nil,
		service:         nil,
		deviceDiscovery: nil,
		cancel:          nil,
	}
}

// VulnerabilityServiceWithNilScanner creates a VulnerabilityService with nil scanner for testing.
func VulnerabilityServiceWithNilScanner() *VulnerabilityService {
	return &VulnerabilityService{
		cfg:     nil,
		db:      nil,
		scanner: nil,
		cancel:  nil,
	}
}

// RogueServiceWithNilDetector creates a RogueService with nil detector for testing.
func RogueServiceWithNilDetector() *RogueService {
	return &RogueService{
		cfg:      nil,
		detector: nil,
		cancel:   nil,
	}
}

// SetDiscoveryServiceCancel sets the cancel function for testing.
func (a *DiscoveryServiceTestAccessor) SetCancel(cancel func()) {
	a.Service.cancel = cancel
}

// SetVulnerabilityServiceCancel sets the cancel function for testing.
func (a *VulnerabilityServiceTestAccessor) SetCancel(cancel func()) {
	a.Service.cancel = cancel
}

// SetRogueServiceCancel sets the cancel function for testing.
func (a *RogueServiceTestAccessor) SetCancel(cancel func()) {
	a.Service.cancel = cancel
}

// TestablePostureService creates a PostureService that can be tested with mock data.
type TestablePostureService struct {
	Service        *PostureService
	MockVulns      []Vulnerability
	MockVulnsError error
}

// NewTestablePostureService creates a testable posture service with mock vulnerability data.
func NewTestablePostureService(vulns []Vulnerability) *TestablePostureService {
	return &TestablePostureService{
		Service: &PostureService{
			cfg:           nil,
			db:            nil,
			discovery:     nil,
			vulnerability: nil,
		},
		MockVulns: vulns,
	}
}

// AssessWithMockVulns performs assessment with mock vulnerability data.
// This directly tests the scoring logic without needing a real vulnerability service.
func AssessWithMockVulns(vulns []Vulnerability) *PostureScore {
	score := &PostureScore{
		Overall:    perfectSecurityScore,
		Categories: make(map[string]int),
		Issues:     make([]PostureIssue, 0),
		AssessedAt: time.Now(),
	}

	// Deduct points for vulnerabilities
	for _, v := range vulns {
		switch v.Severity {
		case SeverityCritical:
			score.Overall -= 20
			score.Issues = append(score.Issues, PostureIssue{
				Category:    "vulnerabilities",
				Severity:    "critical",
				Description: "Critical vulnerability: " + v.CVEID,
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
	score.Categories["vulnerabilities"] = max(0, perfectSecurityScore-len(vulns)*vulnerabilityPenaltyMultiplier)

	// Ensure score doesn't go below 0
	if score.Overall < 0 {
		score.Overall = 0
	}

	return score
}

// CountVulnsBySeverity counts vulnerabilities by severity like the Scan method does.
func CountVulnsBySeverity(vulns []Vulnerability) (critical, high, medium, low int) {
	for _, v := range vulns {
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
			// Info severity not counted
		}
	}
	return
}
