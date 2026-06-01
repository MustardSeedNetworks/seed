package harvest

// services_template.go contains TemplateService: a built-in + user-defined
// report template registry with concurrent-safe CRUD.

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/krisarmstrong/seed/internal/config"
)

// TemplateService manages report templates.
type TemplateService struct {
	cfg       *config.Config
	templates map[string]*Template
	mu        sync.RWMutex
}

// NewTemplateService creates a new template service.
func NewTemplateService(cfg *config.Config) *TemplateService {
	return &TemplateService{
		cfg:       cfg,
		templates: make(map[string]*Template),
	}
}

// Load loads all available templates.
func (s *TemplateService) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.loadBuiltInTemplates()
	return nil
}

func (s *TemplateService) loadBuiltInTemplates() {
	now := time.Now()

	// Executive Summary template
	s.templates["executive"] = &Template{
		ID:          "executive",
		Name:        "Executive Summary",
		Description: "High-level network health and security overview for management",
		Type:        ReportTypeExecutive,
		Formats:     []ExportFormat{FormatPDF, FormatHTML},
		Sections: []TemplateSection{
			{ID: "overview", Name: "Overview", Title: "Network Overview", Order: sectionOrderOverview},
			{ID: "security", Name: "Security", Title: "Security Posture", Order: sectionOrderSecondary},
			{ID: "performance", Name: "Performance", Title: "Performance Summary", Order: sectionOrderTertiary},
			{ID: "recommendations", Name: "Recommendations", Title: "Recommendations", Order: sectionOrderQuaternary},
		},
		IsBuiltIn: true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Vulnerability Report template
	s.templates["vulnerability"] = &Template{
		ID:          "vulnerability",
		Name:        "Vulnerability Report",
		Description: "Detailed vulnerability assessment and remediation guidance",
		Type:        ReportTypeVulnerability,
		Formats:     []ExportFormat{FormatPDF, FormatHTML, FormatCSV},
		Sections: []TemplateSection{
			{ID: "summary", Name: "Summary", Title: "Vulnerability Summary", Order: sectionOrderOverview},
			{ID: statusCritical, Name: "Critical", Title: "Critical Vulnerabilities", Order: sectionOrderSecondary},
			{ID: "high", Name: "High", Title: "High Severity", Order: sectionOrderTertiary},
			{ID: "medium", Name: "Medium", Title: "Medium Severity", Order: sectionOrderQuaternary, Optional: true},
			{ID: "low", Name: "Low", Title: "Low Severity", Order: sectionOrderQuinary, Optional: true},
			{ID: "remediation", Name: "Remediation", Title: "Remediation Plan", Order: sectionOrderRemediation},
		},
		IsBuiltIn: true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Device Inventory template
	s.templates["inventory"] = &Template{
		ID:          "inventory",
		Name:        "Device Inventory",
		Description: "Complete network device inventory with details",
		Type:        ReportTypeInventory,
		Formats:     []ExportFormat{FormatPDF, FormatHTML, FormatCSV, FormatExcel},
		Sections: []TemplateSection{
			{ID: "summary", Name: "Summary", Title: "Inventory Summary", Order: sectionOrderOverview},
			{ID: entityDevices, Name: "Devices", Title: "Device List", Order: sectionOrderSecondary},
			{
				ID:       "software",
				Name:     "Software",
				Title:    "Software Inventory",
				Order:    sectionOrderTertiary,
				Optional: true,
			},
			{ID: "changes", Name: "Changes", Title: "Recent Changes", Order: sectionOrderQuaternary, Optional: true},
		},
		IsBuiltIn: true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Performance Report template
	s.templates["performance"] = &Template{
		ID:          "performance",
		Name:        "Performance Report",
		Description: "Network performance metrics and trends analysis",
		Type:        ReportTypePerformance,
		Formats:     []ExportFormat{FormatPDF, FormatHTML, FormatJSON},
		Sections: []TemplateSection{
			{ID: "overview", Name: "Overview", Title: "Performance Overview", Order: sectionOrderOverview},
			{ID: "latency", Name: "Latency", Title: "Latency Analysis", Order: sectionOrderSecondary},
			{ID: "throughput", Name: "Throughput", Title: "Throughput Metrics", Order: sectionOrderTertiary},
			{ID: "availability", Name: "Availability", Title: "Service Availability", Order: sectionOrderQuaternary},
		},
		IsBuiltIn: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Get retrieves a template by ID.
func (s *TemplateService) Get(id string) (*Template, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.templates[id]
	return t, ok
}

// List returns all available templates.
func (s *TemplateService) List() []Template {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Template, 0, len(s.templates))
	for _, t := range s.templates {
		result = append(result, *t)
	}
	return result
}

// Create adds a custom template.
func (s *TemplateService) Create(tmpl *Template) error {
	if tmpl == nil {
		return errors.New("template is nil")
	}
	if tmpl.ID == "" {
		tmpl.ID = uuid.New().String()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.templates[tmpl.ID]; exists {
		return fmt.Errorf("template already exists: %s", tmpl.ID)
	}

	tmpl.IsBuiltIn = false
	tmpl.CreatedAt = time.Now()
	tmpl.UpdatedAt = time.Now()
	s.templates[tmpl.ID] = tmpl

	return nil
}

// Update modifies a custom template.
func (s *TemplateService) Update(tmpl *Template) error {
	if tmpl == nil {
		return errors.New("template is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.templates[tmpl.ID]
	if !ok {
		return fmt.Errorf("template not found: %s", tmpl.ID)
	}
	if existing.IsBuiltIn {
		return errors.New("cannot modify built-in template")
	}

	tmpl.UpdatedAt = time.Now()
	s.templates[tmpl.ID] = tmpl

	return nil
}

// Delete removes a custom template.
func (s *TemplateService) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tmpl, ok := s.templates[id]
	if !ok {
		return fmt.Errorf("template not found: %s", id)
	}
	if tmpl.IsBuiltIn {
		return errors.New("cannot delete built-in template")
	}

	delete(s.templates, id)
	return nil
}
