package templates_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/krisarmstrong/seed/internal/harvest/templates"
)

// ----------------------------------------------------------------------------
// ExportFormat Tests
// ----------------------------------------------------------------------------

func TestValidFormats(t *testing.T) {
	formats := templates.ValidFormats()
	assert.NotEmpty(t, formats)

	expectedFormats := []templates.ExportFormat{
		templates.FormatPDF,
		templates.FormatHTML,
		templates.FormatCSV,
		templates.FormatJSON,
		templates.FormatExcel,
		templates.FormatMarkdown,
	}

	assert.ElementsMatch(t, expectedFormats, formats)
}

func TestIsValidFormat(t *testing.T) {
	tests := []struct {
		name   string
		format templates.ExportFormat
		want   bool
	}{
		{name: "valid pdf", format: templates.FormatPDF, want: true},
		{name: "valid html", format: templates.FormatHTML, want: true},
		{name: "valid csv", format: templates.FormatCSV, want: true},
		{name: "valid json", format: templates.FormatJSON, want: true},
		{name: "valid excel", format: templates.FormatExcel, want: true},
		{name: "valid markdown", format: templates.FormatMarkdown, want: true},
		{name: "invalid format", format: templates.ExportFormat("invalid"), want: false},
		{name: "empty format", format: templates.ExportFormat(""), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := templates.IsValidFormat(tt.format)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ----------------------------------------------------------------------------
// Template Tests
// ----------------------------------------------------------------------------

func TestTemplate_Validate(t *testing.T) {
	tests := []struct {
		name      string
		template  *templates.Template
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "nil template",
			template:  nil,
			wantErr:   true,
			errSubstr: "nil",
		},
		{
			name: "valid template",
			template: &templates.Template{
				ID:      "test-template",
				Name:    "Test Template",
				Type:    templates.ReportTypeCustom,
				Formats: []templates.ExportFormat{templates.FormatPDF},
				Sections: []templates.Section{
					{ID: "section-1", Name: "Section 1", Order: 1},
				},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			template: &templates.Template{
				ID:      "test",
				Name:    "",
				Formats: []templates.ExportFormat{templates.FormatPDF},
			},
			wantErr:   true,
			errSubstr: "name cannot be empty",
		},
		{
			name: "no formats",
			template: &templates.Template{
				ID:      "test",
				Name:    "Test",
				Formats: []templates.ExportFormat{},
			},
			wantErr:   true,
			errSubstr: "at least one format",
		},
		{
			name: "nil formats",
			template: &templates.Template{
				ID:      "test",
				Name:    "Test",
				Formats: nil,
			},
			wantErr:   true,
			errSubstr: "at least one format",
		},
		{
			name: "invalid format",
			template: &templates.Template{
				ID:      "test",
				Name:    "Test",
				Formats: []templates.ExportFormat{"invalid"},
			},
			wantErr:   true,
			errSubstr: "invalid export format",
		},
		{
			name: "section missing ID",
			template: &templates.Template{
				ID:      "test",
				Name:    "Test",
				Formats: []templates.ExportFormat{templates.FormatPDF},
				Sections: []templates.Section{
					{ID: "", Name: "Section"},
				},
			},
			wantErr:   true,
			errSubstr: "invalid section",
		},
		{
			name: "section missing name",
			template: &templates.Template{
				ID:      "test",
				Name:    "Test",
				Formats: []templates.ExportFormat{templates.FormatPDF},
				Sections: []templates.Section{
					{ID: "sec-1", Name: ""},
				},
			},
			wantErr:   true,
			errSubstr: "invalid section",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.template.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTemplate_Clone(t *testing.T) {
	tests := []struct {
		name     string
		template *templates.Template
	}{
		{
			name:     "nil template",
			template: nil,
		},
		{
			name: "full template",
			template: &templates.Template{
				ID:          "test-id",
				Name:        "Test Name",
				Description: "Test Description",
				Type:        templates.ReportTypeExecutive,
				Formats:     []templates.ExportFormat{templates.FormatPDF, templates.FormatHTML},
				Sections: []templates.Section{
					{ID: "s1", Name: "Section 1", Title: "Title 1", Order: 1},
					{ID: "s2", Name: "Section 2", Title: "Title 2", Order: 2, Optional: true},
				},
				Content:   "template content",
				IsBuiltIn: true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Metadata:  map[string]string{"key": "value"},
			},
		},
		{
			name: "minimal template",
			template: &templates.Template{
				ID:   "minimal",
				Name: "Minimal",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clone := tt.template.Clone()
			if tt.template == nil {
				assert.Nil(t, clone)
				return
			}

			require.NotNil(t, clone)
			assert.NotSame(t, tt.template, clone)

			// Verify all fields are copied
			assert.Equal(t, tt.template.ID, clone.ID)
			assert.Equal(t, tt.template.Name, clone.Name)
			assert.Equal(t, tt.template.Description, clone.Description)
			assert.Equal(t, tt.template.Type, clone.Type)
			assert.Equal(t, tt.template.Content, clone.Content)
			assert.Equal(t, tt.template.IsBuiltIn, clone.IsBuiltIn)
			assert.Equal(t, tt.template.CreatedAt, clone.CreatedAt)
			assert.Equal(t, tt.template.UpdatedAt, clone.UpdatedAt)

			// Verify slices are independent
			if tt.template.Formats != nil {
				assert.Equal(t, tt.template.Formats, clone.Formats)
				// Modify clone and verify original unchanged
				if len(clone.Formats) > 0 {
					clone.Formats[0] = "modified"
					assert.NotEqual(t, tt.template.Formats[0], clone.Formats[0])
				}
			}

			// Verify map is independent
			if tt.template.Metadata != nil {
				assert.Equal(t, tt.template.Metadata, clone.Metadata)
				clone.Metadata["new"] = "value"
				_, exists := tt.template.Metadata["new"]
				assert.False(t, exists)
			}
		})
	}
}

func TestTemplate_SupportsFormat(t *testing.T) {
	tests := []struct {
		name     string
		template *templates.Template
		format   templates.ExportFormat
		want     bool
	}{
		{
			name:     "nil template",
			template: nil,
			format:   templates.FormatPDF,
			want:     false,
		},
		{
			name: "supported format",
			template: &templates.Template{
				Formats: []templates.ExportFormat{templates.FormatPDF, templates.FormatHTML},
			},
			format: templates.FormatPDF,
			want:   true,
		},
		{
			name: "unsupported format",
			template: &templates.Template{
				Formats: []templates.ExportFormat{templates.FormatPDF},
			},
			format: templates.FormatCSV,
			want:   false,
		},
		{
			name: "empty formats",
			template: &templates.Template{
				Formats: []templates.ExportFormat{},
			},
			format: templates.FormatPDF,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.template.SupportsFormat(tt.format)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTemplate_GetSectionByID(t *testing.T) {
	tmpl := &templates.Template{
		Sections: []templates.Section{
			{ID: "overview", Name: "Overview", Order: 1},
			{ID: "details", Name: "Details", Order: 2},
			{ID: "summary", Name: "Summary", Order: 3},
		},
	}

	tests := []struct {
		name      string
		template  *templates.Template
		sectionID string
		wantFound bool
	}{
		{
			name:      "nil template",
			template:  nil,
			sectionID: "overview",
			wantFound: false,
		},
		{
			name:      "existing section",
			template:  tmpl,
			sectionID: "details",
			wantFound: true,
		},
		{
			name:      "non-existent section",
			template:  tmpl,
			sectionID: "nonexistent",
			wantFound: false,
		},
		{
			name:      "empty ID",
			template:  tmpl,
			sectionID: "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section, found := tt.template.GetSectionByID(tt.sectionID)
			assert.Equal(t, tt.wantFound, found)
			if tt.wantFound {
				require.NotNil(t, section)
				assert.Equal(t, tt.sectionID, section.ID)
			}
		})
	}
}

func TestTemplate_GetOrderedSections(t *testing.T) {
	tests := []struct {
		name      string
		template  *templates.Template
		wantOrder []string
	}{
		{
			name:      "nil template",
			template:  nil,
			wantOrder: nil,
		},
		{
			name: "unordered sections",
			template: &templates.Template{
				Sections: []templates.Section{
					{ID: "third", Name: "Third", Order: 3},
					{ID: "first", Name: "First", Order: 1},
					{ID: "second", Name: "Second", Order: 2},
				},
			},
			wantOrder: []string{"first", "second", "third"},
		},
		{
			name: "empty sections",
			template: &templates.Template{
				Sections: []templates.Section{},
			},
			wantOrder: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sections := tt.template.GetOrderedSections()
			if tt.wantOrder == nil {
				assert.Nil(t, sections)
				return
			}

			require.Len(t, sections, len(tt.wantOrder))
			for i, id := range tt.wantOrder {
				assert.Equal(t, id, sections[i].ID)
			}
		})
	}
}

func TestTemplate_GetRequiredSections(t *testing.T) {
	tests := []struct {
		name     string
		template *templates.Template
		wantLen  int
	}{
		{
			name:     "nil template",
			template: nil,
			wantLen:  0,
		},
		{
			name: "mixed sections",
			template: &templates.Template{
				Sections: []templates.Section{
					{ID: "req1", Name: "Required 1", Optional: false},
					{ID: "opt1", Name: "Optional 1", Optional: true},
					{ID: "req2", Name: "Required 2", Optional: false},
				},
			},
			wantLen: 2,
		},
		{
			name: "all optional",
			template: &templates.Template{
				Sections: []templates.Section{
					{ID: "opt1", Name: "Optional 1", Optional: true},
					{ID: "opt2", Name: "Optional 2", Optional: true},
				},
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sections := tt.template.GetRequiredSections()
			assert.Len(t, sections, tt.wantLen)

			for _, s := range sections {
				assert.False(t, s.Optional)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// DateRange Tests
// ----------------------------------------------------------------------------

func TestDateRange_Days(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		wantDays int
	}{
		{
			name:     "one day",
			start:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			wantDays: 1,
		},
		{
			name:     "one week",
			start:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2025, 1, 8, 0, 0, 0, 0, time.UTC),
			wantDays: 7,
		},
		{
			name:     "same day",
			start:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			wantDays: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := templates.DateRange{Start: tt.start, End: tt.end}
			assert.Equal(t, tt.wantDays, dr.Days())
		})
	}
}

func TestDateRange_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		start time.Time
		end   time.Time
		want  bool
	}{
		{
			name:  "valid range",
			start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2025, 1, 7, 0, 0, 0, 0, time.UTC),
			want:  true,
		},
		{
			name:  "same time",
			start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			want:  true,
		},
		{
			name:  "end before start",
			start: time.Date(2025, 1, 7, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			want:  false,
		},
		{
			name:  "zero start",
			start: time.Time{},
			end:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			want:  false,
		},
		{
			name:  "zero end",
			start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Time{},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := templates.DateRange{Start: tt.start, End: tt.end}
			assert.Equal(t, tt.want, dr.IsValid())
		})
	}
}

func TestDateRange_String(t *testing.T) {
	tests := []struct {
		name  string
		start time.Time
		end   time.Time
		want  string
	}{
		{
			name:  "valid range",
			start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2025, 1, 7, 0, 0, 0, 0, time.UTC),
			want:  "Jan 1, 2025 to Jan 7, 2025",
		},
		{
			name:  "zero start",
			start: time.Time{},
			end:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			want:  "",
		},
		{
			name:  "zero end",
			start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Time{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := templates.DateRange{Start: tt.start, End: tt.end}
			assert.Equal(t, tt.want, dr.String())
		})
	}
}

// ----------------------------------------------------------------------------
// Renderer Tests
// ----------------------------------------------------------------------------

func TestNewRenderer(t *testing.T) {
	r := templates.NewRenderer()
	require.NotNil(t, r)
}

func TestRenderer_Render(t *testing.T) {
	r := templates.NewRenderer()

	tests := []struct {
		name      string
		template  *templates.Template
		data      *templates.RenderData
		wantErr   bool
		errSubstr string
		contains  string
	}{
		{
			name:      "nil template",
			template:  nil,
			data:      nil,
			wantErr:   true,
			errSubstr: "nil",
		},
		{
			name: "empty content",
			template: &templates.Template{
				ID:      "empty",
				Content: "",
			},
			wantErr:   true,
			errSubstr: "no content",
		},
		{
			name: "simple template",
			template: &templates.Template{
				ID:      "simple",
				Content: "Hello, {{.Title}}!",
			},
			data: &templates.RenderData{
				Title: "World",
			},
			wantErr:  false,
			contains: "Hello, World!",
		},
		{
			name: "template with date",
			template: &templates.Template{
				ID:      "dated",
				Content: "Generated: {{formatDate .GeneratedAt \"2006-01-02\"}}",
			},
			data: &templates.RenderData{
				GeneratedAt: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			wantErr:  false,
			contains: "Generated: 2025-01-15",
		},
		{
			name: "invalid template syntax",
			template: &templates.Template{
				ID:      "invalid",
				Content: "{{.Invalid",
			},
			wantErr:   true,
			errSubstr: "rendering failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Render(tt.template, tt.data)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				if tt.contains != "" {
					assert.Contains(t, result, tt.contains)
				}
			}
		})
	}
}

func TestRenderer_RenderString(t *testing.T) {
	r := templates.NewRenderer()

	tests := []struct {
		name      string
		content   string
		data      any
		wantErr   bool
		contains  string
		errSubstr string
	}{
		{
			name:     "empty content",
			content:  "",
			data:     nil,
			wantErr:  false,
			contains: "",
		},
		{
			name:    "simple string",
			content: "Hello, {{.Name}}!",
			data: map[string]string{
				"Name": "Test",
			},
			wantErr:  false,
			contains: "Hello, Test!",
		},
		{
			name:    "with number formatting",
			content: "Value: {{formatNumber .Value 2}}",
			data: map[string]float64{
				"Value": 123.456,
			},
			wantErr:  false,
			contains: "Value: 123.46",
		},
		{
			name:      "invalid syntax",
			content:   "{{.Broken",
			data:      nil,
			wantErr:   true,
			errSubstr: "rendering failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.RenderString(tt.content, tt.data)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				assert.Contains(t, result, tt.contains)
			}
		})
	}
}

func TestRenderer_AddFunc(t *testing.T) {
	r := templates.NewRenderer()

	// Add custom function
	r.AddFunc("double", func(n int) int {
		return n * 2
	})

	result, err := r.RenderString("Double of 5 is {{double .N}}", map[string]int{"N": 5})
	require.NoError(t, err)
	assert.Equal(t, "Double of 5 is 10", result)
}

func TestRenderer_ClearCache(t *testing.T) {
	r := templates.NewRenderer()

	tmpl := &templates.Template{
		ID:      "cached",
		Content: "Original: {{.Title}}",
	}

	// First render - should cache
	result1, err := r.Render(tmpl, &templates.RenderData{Title: "First"})
	require.NoError(t, err)
	assert.Contains(t, result1, "Original: First")

	// Clear cache
	r.ClearCache()

	// Render again should work
	result2, err := r.Render(tmpl, &templates.RenderData{Title: "Second"})
	require.NoError(t, err)
	assert.Contains(t, result2, "Original: Second")
}

func TestRenderer_InvalidateTemplate(t *testing.T) {
	r := templates.NewRenderer()

	tmpl := &templates.Template{
		ID:      "to-invalidate",
		Content: "Value: {{.Title}}",
	}

	// Render to cache
	_, err := r.Render(tmpl, &templates.RenderData{Title: "Test"})
	require.NoError(t, err)

	// Invalidate
	r.InvalidateTemplate("to-invalidate")

	// Should still work after invalidation
	result, err := r.Render(tmpl, &templates.RenderData{Title: "After"})
	require.NoError(t, err)
	assert.Contains(t, result, "Value: After")
}

func TestRenderer_BuiltInFunctions(t *testing.T) {
	r := templates.NewRenderer()

	tests := []struct {
		name     string
		template string
		data     any
		want     string
	}{
		{
			name:     "upper function",
			template: "{{upper .Text}}",
			data:     map[string]string{"Text": "hello"},
			want:     "HELLO",
		},
		{
			name:     "lower function",
			template: "{{lower .Text}}",
			data:     map[string]string{"Text": "HELLO"},
			want:     "hello",
		},
		{
			name:     "join function",
			template: "{{join .Items \", \"}}",
			data:     map[string][]string{"Items": {"a", "b", "c"}},
			want:     "a, b, c",
		},
		{
			name:     "add function",
			template: "{{add .A .B}}",
			data:     map[string]int{"A": 5, "B": 3},
			want:     "8",
		},
		{
			name:     "sub function",
			template: "{{sub .A .B}}",
			data:     map[string]int{"A": 5, "B": 3},
			want:     "2",
		},
		{
			name:     "mul function",
			template: "{{mul .A .B}}",
			data:     map[string]int{"A": 5, "B": 3},
			want:     "15",
		},
		{
			name:     "div function",
			template: "{{div .A .B}}",
			data:     map[string]int{"A": 10, "B": 3},
			want:     "3",
		},
		{
			name:     "div by zero",
			template: "{{div .A .B}}",
			data:     map[string]int{"A": 10, "B": 0},
			want:     "0",
		},
		{
			name:     "percent function",
			template: "{{percent .N .Total}}",
			data:     map[string]float64{"N": 25, "Total": 100},
			want:     "25.0%",
		},
		{
			name:     "percent with zero total",
			template: "{{percent .N .Total}}",
			data:     map[string]float64{"N": 25, "Total": 0},
			want:     "0%",
		},
		{
			name:     "default with value",
			template: "{{default \"fallback\" .Value}}",
			data:     map[string]string{"Value": "actual"},
			want:     "actual",
		},
		{
			name:     "default with empty",
			template: "{{default \"fallback\" .Value}}",
			data:     map[string]string{"Value": ""},
			want:     "fallback",
		},
		{
			name:     "formatDate with zero time",
			template: "{{formatDate .Time \"2006-01-02\"}}",
			data:     map[string]time.Time{"Time": {}},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.RenderString(tt.template, tt.data)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRenderer_Concurrency(t *testing.T) {
	r := templates.NewRenderer()
	tmpl := &templates.Template{
		ID:      "concurrent",
		Content: "Hello, {{.Title}}!",
	}

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := &templates.RenderData{Title: "World"}
			result, err := r.Render(tmpl, data)
			if err != nil {
				errors <- err
				return
			}
			if result != "Hello, World!" {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent render error: %v", err)
	}
}

// ----------------------------------------------------------------------------
// Registry Tests
// ----------------------------------------------------------------------------

func TestNewRegistry(t *testing.T) {
	r := templates.NewRegistry()
	require.NotNil(t, r)
	assert.Equal(t, 0, r.Count())
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name      string
		template  *templates.Template
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid template",
			template: &templates.Template{
				ID:      "test-1",
				Name:    "Test Template",
				Formats: []templates.ExportFormat{templates.FormatPDF},
			},
			wantErr: false,
		},
		{
			name:      "nil template",
			template:  nil,
			wantErr:   true,
			errSubstr: "nil",
		},
		{
			name: "invalid template",
			template: &templates.Template{
				ID:   "test-2",
				Name: "",
			},
			wantErr:   true,
			errSubstr: "name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := templates.NewRegistry()
			err := r.Register(tt.template)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, 1, r.Count())
			}
		})
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	r := templates.NewRegistry()

	tmpl := &templates.Template{
		ID:      "duplicate-test",
		Name:    "Test",
		Formats: []templates.ExportFormat{templates.FormatPDF},
	}

	err := r.Register(tmpl)
	require.NoError(t, err)

	err = r.Register(tmpl)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestRegistry_Update(t *testing.T) {
	r := templates.NewRegistry()

	original := &templates.Template{
		ID:        "update-test",
		Name:      "Original Name",
		Formats:   []templates.ExportFormat{templates.FormatPDF},
		IsBuiltIn: false,
	}
	require.NoError(t, r.Register(original))

	tests := []struct {
		name      string
		template  *templates.Template
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid update",
			template: &templates.Template{
				ID:      "update-test",
				Name:    "Updated Name",
				Formats: []templates.ExportFormat{templates.FormatPDF, templates.FormatHTML},
			},
			wantErr: false,
		},
		{
			name:      "nil template",
			template:  nil,
			wantErr:   true,
			errSubstr: "nil",
		},
		{
			name: "non-existent template",
			template: &templates.Template{
				ID:      "nonexistent",
				Name:    "Test",
				Formats: []templates.ExportFormat{templates.FormatPDF},
			},
			wantErr:   true,
			errSubstr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.Update(tt.template)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				updated, found := r.Get(tt.template.ID)
				require.True(t, found)
				assert.Equal(t, tt.template.Name, updated.Name)
			}
		})
	}
}

func TestRegistry_UpdateBuiltIn(t *testing.T) {
	r := templates.NewRegistry()

	builtIn := &templates.Template{
		ID:        "built-in",
		Name:      "Built-In Template",
		Formats:   []templates.ExportFormat{templates.FormatPDF},
		IsBuiltIn: true,
	}
	require.NoError(t, r.Register(builtIn))

	modified := builtIn.Clone()
	modified.Name = "Modified"

	err := r.Update(modified)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "built-in")
}

func TestRegistry_Unregister(t *testing.T) {
	r := templates.NewRegistry()

	// Register templates
	custom := &templates.Template{
		ID:        "custom",
		Name:      "Custom",
		Formats:   []templates.ExportFormat{templates.FormatPDF},
		IsBuiltIn: false,
	}
	builtIn := &templates.Template{
		ID:        "built-in",
		Name:      "Built-In",
		Formats:   []templates.ExportFormat{templates.FormatPDF},
		IsBuiltIn: true,
	}
	require.NoError(t, r.Register(custom))
	require.NoError(t, r.Register(builtIn))

	tests := []struct {
		name      string
		id        string
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "unregister custom",
			id:      "custom",
			wantErr: false,
		},
		{
			name:      "unregister built-in",
			id:        "built-in",
			wantErr:   true,
			errSubstr: "built-in",
		},
		{
			name:      "unregister non-existent",
			id:        "nonexistent",
			wantErr:   true,
			errSubstr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.Unregister(tt.id)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				_, found := r.Get(tt.id)
				assert.False(t, found)
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	r := templates.NewRegistry()

	tmpl := &templates.Template{
		ID:      "get-test",
		Name:    "Get Test",
		Formats: []templates.ExportFormat{templates.FormatPDF},
	}
	require.NoError(t, r.Register(tmpl))

	tests := []struct {
		name      string
		id        string
		wantFound bool
	}{
		{
			name:      "existing template",
			id:        "get-test",
			wantFound: true,
		},
		{
			name:      "non-existent template",
			id:        "nonexistent",
			wantFound: false,
		},
		{
			name:      "empty ID",
			id:        "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := r.Get(tt.id)
			assert.Equal(t, tt.wantFound, found)
			if tt.wantFound {
				require.NotNil(t, result)
				assert.Equal(t, tt.id, result.ID)
				// Verify it's a clone
				assert.NotSame(t, tmpl, result)
			}
		})
	}
}

func TestRegistry_List(t *testing.T) {
	r := templates.NewRegistry()

	// Empty registry
	list := r.List()
	assert.Empty(t, list)

	// Add templates
	for i := range 3 {
		tmpl := &templates.Template{
			ID:      "list-" + string(rune('a'+i)),
			Name:    "List Test " + string(rune('A'+i)),
			Formats: []templates.ExportFormat{templates.FormatPDF},
		}
		require.NoError(t, r.Register(tmpl))
	}

	list = r.List()
	assert.Len(t, list, 3)

	// Verify all are clones
	for _, tmpl := range list {
		original, _ := r.Get(tmpl.ID)
		assert.NotSame(t, original, tmpl)
	}
}

func TestRegistry_ListByType(t *testing.T) {
	r := templates.NewRegistry()

	// Register templates of different types
	testTmpls := []*templates.Template{
		{
			ID:      "exec-1",
			Name:    "Exec 1",
			Type:    templates.ReportTypeExecutive,
			Formats: []templates.ExportFormat{templates.FormatPDF},
		},
		{
			ID:      "exec-2",
			Name:    "Exec 2",
			Type:    templates.ReportTypeExecutive,
			Formats: []templates.ExportFormat{templates.FormatPDF},
		},
		{
			ID:      "vuln-1",
			Name:    "Vuln 1",
			Type:    templates.ReportTypeVulnerability,
			Formats: []templates.ExportFormat{templates.FormatPDF},
		},
		{
			ID:      "perf-1",
			Name:    "Perf 1",
			Type:    templates.ReportTypePerformance,
			Formats: []templates.ExportFormat{templates.FormatPDF},
		},
	}

	for _, tmpl := range testTmpls {
		require.NoError(t, r.Register(tmpl))
	}

	// Test listing by type
	execTemplates := r.ListByType(templates.ReportTypeExecutive)
	assert.Len(t, execTemplates, 2)

	vulnTemplates := r.ListByType(templates.ReportTypeVulnerability)
	assert.Len(t, vulnTemplates, 1)

	customTemplates := r.ListByType(templates.ReportTypeCustom)
	assert.Empty(t, customTemplates)
}

func TestRegistry_ListByFormat(t *testing.T) {
	r := templates.NewRegistry()

	// Register templates with different formats
	tmpls := []*templates.Template{
		{
			ID:      "pdf-only",
			Name:    "PDF Only",
			Formats: []templates.ExportFormat{templates.FormatPDF},
		},
		{
			ID:      "html-only",
			Name:    "HTML Only",
			Formats: []templates.ExportFormat{templates.FormatHTML},
		},
		{
			ID:      "multi-format",
			Name:    "Multi Format",
			Formats: []templates.ExportFormat{templates.FormatPDF, templates.FormatHTML, templates.FormatCSV},
		},
	}

	for _, tmpl := range tmpls {
		require.NoError(t, r.Register(tmpl))
	}

	pdfTemplates := r.ListByFormat(templates.FormatPDF)
	assert.Len(t, pdfTemplates, 2)

	htmlTemplates := r.ListByFormat(templates.FormatHTML)
	assert.Len(t, htmlTemplates, 2)

	csvTemplates := r.ListByFormat(templates.FormatCSV)
	assert.Len(t, csvTemplates, 1)

	excelTemplates := r.ListByFormat(templates.FormatExcel)
	assert.Empty(t, excelTemplates)
}

func TestRegistry_Count(t *testing.T) {
	r := templates.NewRegistry()

	assert.Equal(t, 0, r.Count())

	for i := range 5 {
		tmpl := &templates.Template{
			ID:      "count-" + string(rune('a'+i)),
			Name:    "Count Test",
			Formats: []templates.ExportFormat{templates.FormatPDF},
		}
		require.NoError(t, r.Register(tmpl))
	}

	assert.Equal(t, 5, r.Count())
}

func TestRegistry_Clear(t *testing.T) {
	r := templates.NewRegistry()

	for i := range 3 {
		tmpl := &templates.Template{
			ID:      "clear-" + string(rune('a'+i)),
			Name:    "Clear Test",
			Formats: []templates.ExportFormat{templates.FormatPDF},
		}
		require.NoError(t, r.Register(tmpl))
	}

	assert.Equal(t, 3, r.Count())

	r.Clear()

	assert.Equal(t, 0, r.Count())
	assert.Empty(t, r.List())
}

func TestRegistry_Concurrency(t *testing.T) {
	r := templates.NewRegistry()

	var wg sync.WaitGroup
	errChan := make(chan error, 200)

	// Concurrent writes
	for i := range 50 {
		wg.Go(func() {
			n := i
			tmpl := &templates.Template{
				ID:      "concurrent-" + string(rune('a'+n%26)) + "-" + string(rune('0'+n%10)),
				Name:    "Concurrent Test",
				Formats: []templates.ExportFormat{templates.FormatPDF},
			}
			if err := r.Register(tmpl); err != nil {
				// Duplicate errors are expected
				if !assert.Contains(t, err.Error(), "already exists") {
					errChan <- err
				}
			}
		})
	}

	// Concurrent reads
	for range 50 {
		wg.Go(func() {
			_ = r.List()
			_ = r.Count()
			_, _ = r.Get("nonexistent")
		})
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("concurrent operation error: %v", err)
	}
}

// ----------------------------------------------------------------------------
// IDValidator Tests
// ----------------------------------------------------------------------------

func TestNewIDValidator(t *testing.T) {
	v := templates.NewIDValidator()
	require.NotNil(t, v)
}

func TestIDValidator_IsValid(t *testing.T) {
	v := templates.NewIDValidator()

	tests := []struct {
		name string
		id   string
		want bool
	}{
		{name: "valid simple", id: "test", want: true},
		{name: "valid with hyphen", id: "my-template", want: true},
		{name: "valid with numbers", id: "template123", want: true},
		{name: "valid complex", id: "my-report-template-v2", want: true},
		{name: "minimum length", id: "ab", want: true},
		{name: "too short", id: "a", want: false},
		{name: "starts with number", id: "1template", want: false},
		{name: "contains uppercase", id: "myTemplate", want: false},
		{name: "contains underscore", id: "my_template", want: false},
		{name: "ends with hyphen", id: "template-", want: false},
		{name: "starts with hyphen", id: "-template", want: false},
		{name: "contains space", id: "my template", want: false},
		{name: "empty string", id: "", want: false},
		{
			name: "too long",
			id:   "this-is-a-very-long-template-id-that-exceeds-the-maximum-allowed-length",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.IsValid(tt.id)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIDValidator_Validate(t *testing.T) {
	v := templates.NewIDValidator()

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{name: "valid", id: "my-template", wantErr: false},
		{name: "invalid", id: "INVALID", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.id)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid template ID")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIDValidator_SanitizeID(t *testing.T) {
	v := templates.NewIDValidator()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "already valid", input: "my-template", want: "my-template"},
		{name: "uppercase", input: "My-Template", want: "my-template"},
		{name: "with spaces", input: "My Template", want: "my-template"},
		{name: "with underscores", input: "my_template", want: "my-template"},
		{name: "multiple special", input: "My___Template!!!", want: "my-template"},
		{name: "starts with number", input: "123template", want: "t-123template"},
		{name: "very short", input: "a", want: "template-a"},
		{name: "leading/trailing spaces", input: "  template  ", want: "template"},
		{name: "leading/trailing hyphens", input: "-template-", want: "template"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.SanitizeID(tt.input)
			assert.Equal(t, tt.want, got)
			// Verify result is valid
			assert.True(t, v.IsValid(got), "sanitized ID should be valid")
		})
	}
}

// ----------------------------------------------------------------------------
// Error Type Tests
// ----------------------------------------------------------------------------

func TestErrors(t *testing.T) {
	errors := []error{
		templates.ErrTemplateNil,
		templates.ErrTemplateNotFound,
		templates.ErrTemplateExists,
		templates.ErrBuiltInTemplate,
		templates.ErrInvalidTemplateID,
		templates.ErrEmptyTemplateName,
		templates.ErrNoFormats,
		templates.ErrInvalidFormat,
		templates.ErrInvalidSection,
		templates.ErrRenderFailed,
	}

	for _, err := range errors {
		require.NotNil(t, err)
		assert.NotEmpty(t, err.Error())
	}
}

// ----------------------------------------------------------------------------
// ReportType Tests
// ----------------------------------------------------------------------------

func TestReportTypes(t *testing.T) {
	types := []templates.ReportType{
		templates.ReportTypeExecutive,
		templates.ReportTypeDetailed,
		templates.ReportTypeVulnerability,
		templates.ReportTypeCompliance,
		templates.ReportTypeInventory,
		templates.ReportTypePerformance,
		templates.ReportTypeIncident,
		templates.ReportTypeCustom,
	}

	for _, rt := range types {
		assert.NotEmpty(t, string(rt))
	}
}
