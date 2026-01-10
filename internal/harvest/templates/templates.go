package templates

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"maps"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// Common errors for template operations.
var (
	ErrTemplateNil       = errors.New("template is nil")
	ErrTemplateNotFound  = errors.New("template not found")
	ErrTemplateExists    = errors.New("template already exists")
	ErrBuiltInTemplate   = errors.New("cannot modify built-in template")
	ErrInvalidTemplateID = errors.New("invalid template ID")
	ErrEmptyTemplateName = errors.New("template name cannot be empty")
	ErrNoFormats         = errors.New("template must support at least one format")
	ErrInvalidFormat     = errors.New("invalid export format")
	ErrInvalidSection    = errors.New("invalid section configuration")
	ErrRenderFailed      = errors.New("template rendering failed")
)

// ReportType categorizes reports.
type ReportType string

// ReportType values.
const (
	ReportTypeExecutive     ReportType = "executive"
	ReportTypeDetailed      ReportType = "detailed"
	ReportTypeVulnerability ReportType = "vulnerability"
	ReportTypeCompliance    ReportType = "compliance"
	ReportTypeInventory     ReportType = "inventory"
	ReportTypePerformance   ReportType = "performance"
	ReportTypeIncident      ReportType = "incident"
	ReportTypeCustom        ReportType = "custom"
)

// ExportFormat specifies output format.
type ExportFormat string

// ExportFormat values.
const (
	FormatPDF      ExportFormat = "pdf"
	FormatHTML     ExportFormat = "html"
	FormatCSV      ExportFormat = "csv"
	FormatJSON     ExportFormat = "json"
	FormatExcel    ExportFormat = "xlsx"
	FormatMarkdown ExportFormat = "md"
)

// ValidFormats returns all valid export formats.
func ValidFormats() []ExportFormat {
	return []ExportFormat{
		FormatPDF,
		FormatHTML,
		FormatCSV,
		FormatJSON,
		FormatExcel,
		FormatMarkdown,
	}
}

// IsValidFormat checks if a format is valid.
func IsValidFormat(f ExportFormat) bool {
	return slices.Contains(ValidFormats(), f)
}

// Template represents a report template.
type Template struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Type        ReportType        `json:"type"`
	Formats     []ExportFormat    `json:"formats"`
	Sections    []Section         `json:"sections"`
	Content     string            `json:"content,omitempty"`
	IsBuiltIn   bool              `json:"isBuiltIn"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Section defines a section in a template.
type Section struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Title    string `json:"title"`
	Content  string `json:"content,omitempty"`
	Optional bool   `json:"optional"`
	Order    int    `json:"order"`
}

// Validate checks if the template is valid.
func (t *Template) Validate() error {
	if t == nil {
		return ErrTemplateNil
	}
	if t.Name == "" {
		return ErrEmptyTemplateName
	}
	if len(t.Formats) == 0 {
		return ErrNoFormats
	}
	for _, f := range t.Formats {
		if !IsValidFormat(f) {
			return fmt.Errorf("%w: %s", ErrInvalidFormat, f)
		}
	}
	for i, s := range t.Sections {
		if s.ID == "" || s.Name == "" {
			return fmt.Errorf("%w: section at index %d missing ID or Name", ErrInvalidSection, i)
		}
	}
	return nil
}

// Clone creates a deep copy of the template.
func (t *Template) Clone() *Template {
	if t == nil {
		return nil
	}
	clone := &Template{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Type:        t.Type,
		Content:     t.Content,
		IsBuiltIn:   t.IsBuiltIn,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
	if t.Formats != nil {
		clone.Formats = make([]ExportFormat, len(t.Formats))
		copy(clone.Formats, t.Formats)
	}
	if t.Sections != nil {
		clone.Sections = make([]Section, len(t.Sections))
		copy(clone.Sections, t.Sections)
	}
	if t.Metadata != nil {
		clone.Metadata = make(map[string]string, len(t.Metadata))
		maps.Copy(clone.Metadata, t.Metadata)
	}
	return clone
}

// SupportsFormat checks if the template supports the given format.
func (t *Template) SupportsFormat(f ExportFormat) bool {
	if t == nil {
		return false
	}
	return slices.Contains(t.Formats, f)
}

// GetSectionByID returns a section by its ID.
func (t *Template) GetSectionByID(id string) (*Section, bool) {
	if t == nil {
		return nil, false
	}
	for i := range t.Sections {
		if t.Sections[i].ID == id {
			return &t.Sections[i], true
		}
	}
	return nil, false
}

// GetOrderedSections returns sections sorted by their order.
func (t *Template) GetOrderedSections() []Section {
	if t == nil || len(t.Sections) == 0 {
		return nil
	}
	sections := make([]Section, len(t.Sections))
	copy(sections, t.Sections)
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Order < sections[j].Order
	})
	return sections
}

// GetRequiredSections returns only non-optional sections.
func (t *Template) GetRequiredSections() []Section {
	if t == nil {
		return nil
	}
	var required []Section
	for _, s := range t.Sections {
		if !s.Optional {
			required = append(required, s)
		}
	}
	return required
}

// RenderData contains data for template rendering.
type RenderData struct {
	Title        string
	ReportName   string
	GeneratedAt  time.Time
	Period       DateRange
	Metrics      map[string]any
	Sections     map[string]any
	CustomFields map[string]any
}

// DateRange represents a time period.
type DateRange struct {
	Start time.Time
	End   time.Time
}

const (
	hoursPerDay         = 24
	percentScale        = 100.0
	minTemplateIDLength = 2
	maxTemplateIDLength = 64
)

// Days returns the number of days in the range.
func (d DateRange) Days() int {
	return int(d.End.Sub(d.Start).Hours() / hoursPerDay)
}

// IsValid checks if the date range is valid.
func (d DateRange) IsValid() bool {
	return !d.Start.IsZero() && !d.End.IsZero() && !d.End.Before(d.Start)
}

// String returns a formatted string representation.
func (d DateRange) String() string {
	if d.Start.IsZero() || d.End.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s to %s", d.Start.Format("Jan 2, 2006"), d.End.Format("Jan 2, 2006"))
}

// Renderer handles template rendering.
type Renderer struct {
	funcMap  template.FuncMap
	mu       sync.RWMutex
	compiled map[string]*template.Template
}

// NewRenderer creates a new template renderer.
func NewRenderer() *Renderer {
	return &Renderer{
		funcMap:  defaultFuncMap(),
		compiled: make(map[string]*template.Template),
	}
}

// defaultFuncMap returns the default template function map.
func defaultFuncMap() template.FuncMap {
	return template.FuncMap{
		"formatDate": func(t time.Time, layout string) string {
			if t.IsZero() {
				return ""
			}
			return t.Format(layout)
		},
		"formatNumber": func(n float64, decimals int) string {
			format := fmt.Sprintf("%%.%df", decimals)
			return fmt.Sprintf(format, n)
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title, //nolint:staticcheck // Title is deprecated but works for simple cases
		"join":  strings.Join,
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"percent": func(n, total float64) string {
			if total == 0 {
				return "0%"
			}
			return fmt.Sprintf("%.1f%%", (n/total)*percentScale)
		},
		"default": func(def, val any) any {
			if val == nil || val == "" {
				return def
			}
			return val
		},
	}
}

// AddFunc adds a custom function to the renderer.
func (r *Renderer) AddFunc(name string, fn any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.funcMap[name] = fn
}

// Render renders a template with the given data.
func (r *Renderer) Render(tmpl *Template, data *RenderData) (string, error) {
	if tmpl == nil {
		return "", ErrTemplateNil
	}
	if tmpl.Content == "" {
		return "", fmt.Errorf("%w: template has no content", ErrRenderFailed)
	}

	r.mu.Lock()
	compiled, exists := r.compiled[tmpl.ID]
	if !exists || compiled == nil {
		var err error
		compiled, err = template.New(tmpl.ID).Funcs(r.funcMap).Parse(tmpl.Content)
		if err != nil {
			r.mu.Unlock()
			return "", errors.Join(ErrRenderFailed, err)
		}
		r.compiled[tmpl.ID] = compiled
	}
	r.mu.Unlock()

	var buf bytes.Buffer
	if err := compiled.Execute(&buf, data); err != nil {
		return "", errors.Join(ErrRenderFailed, err)
	}

	return buf.String(), nil
}

// RenderString renders a template string directly.
func (r *Renderer) RenderString(content string, data any) (string, error) {
	if content == "" {
		return "", nil
	}

	r.mu.RLock()
	funcMap := r.funcMap
	r.mu.RUnlock()

	tmpl, err := template.New("inline").Funcs(funcMap).Parse(content)
	if err != nil {
		return "", errors.Join(ErrRenderFailed, err)
	}

	var buf bytes.Buffer
	if execErr := tmpl.Execute(&buf, data); execErr != nil {
		return "", errors.Join(ErrRenderFailed, execErr)
	}

	return buf.String(), nil
}

// ClearCache clears the compiled template cache.
func (r *Renderer) ClearCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.compiled = make(map[string]*template.Template)
}

// InvalidateTemplate removes a specific template from the cache.
func (r *Renderer) InvalidateTemplate(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.compiled, id)
}

// Registry manages a collection of templates.
type Registry struct {
	templates map[string]*Template
	mu        sync.RWMutex
}

// NewRegistry creates a new template registry.
func NewRegistry() *Registry {
	return &Registry{
		templates: make(map[string]*Template),
	}
}

// Register adds a template to the registry.
func (r *Registry) Register(tmpl *Template) error {
	if err := tmpl.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.templates[tmpl.ID]; exists {
		return fmt.Errorf("%w: %s", ErrTemplateExists, tmpl.ID)
	}

	r.templates[tmpl.ID] = tmpl.Clone()
	return nil
}

// Update modifies an existing template.
func (r *Registry) Update(tmpl *Template) error {
	if err := tmpl.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.templates[tmpl.ID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrTemplateNotFound, tmpl.ID)
	}
	if existing.IsBuiltIn {
		return ErrBuiltInTemplate
	}

	tmpl.UpdatedAt = time.Now()
	r.templates[tmpl.ID] = tmpl.Clone()
	return nil
}

// Unregister removes a template from the registry.
func (r *Registry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tmpl, ok := r.templates[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrTemplateNotFound, id)
	}
	if tmpl.IsBuiltIn {
		return ErrBuiltInTemplate
	}

	delete(r.templates, id)
	return nil
}

// Get retrieves a template by ID.
func (r *Registry) Get(id string) (*Template, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tmpl, ok := r.templates[id]
	if !ok {
		return nil, false
	}
	return tmpl.Clone(), true
}

// List returns all templates.
func (r *Registry) List() []*Template {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Template, 0, len(r.templates))
	for _, tmpl := range r.templates {
		result = append(result, tmpl.Clone())
	}
	return result
}

// ListByType returns templates of a specific type.
func (r *Registry) ListByType(t ReportType) []*Template {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Template
	for _, tmpl := range r.templates {
		if tmpl.Type == t {
			result = append(result, tmpl.Clone())
		}
	}
	return result
}

// ListByFormat returns templates supporting a specific format.
func (r *Registry) ListByFormat(f ExportFormat) []*Template {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Template
	for _, tmpl := range r.templates {
		if tmpl.SupportsFormat(f) {
			result = append(result, tmpl.Clone())
		}
	}
	return result
}

// Count returns the number of registered templates.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.templates)
}

// Clear removes all templates from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.templates = make(map[string]*Template)
}

// IDValidator validates template IDs.
type IDValidator struct {
	pattern *regexp.Regexp
}

// NewIDValidator creates a new ID validator.
func NewIDValidator() *IDValidator {
	return &IDValidator{
		pattern: regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`),
	}
}

// IsValid checks if an ID is valid.
func (v *IDValidator) IsValid(id string) bool {
	if len(id) < minTemplateIDLength || len(id) > maxTemplateIDLength {
		return false
	}
	return v.pattern.MatchString(id)
}

// Validate returns an error if the ID is invalid.
func (v *IDValidator) Validate(id string) error {
	if !v.IsValid(id) {
		return fmt.Errorf(
			"%w: must be 2-64 chars, lowercase alphanumeric with hyphens, start with letter",
			ErrInvalidTemplateID,
		)
	}
	return nil
}

// SanitizeID converts a string to a valid ID.
func (v *IDValidator) SanitizeID(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > maxTemplateIDLength {
		s = s[:maxTemplateIDLength]
	}
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		s = "t-" + s
	}
	if len(s) < minTemplateIDLength {
		s = "template-" + s
	}
	return s
}
