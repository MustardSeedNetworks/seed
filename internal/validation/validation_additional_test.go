package validation_test

import (
	"strings"
	"testing"

	"github.com/krisarmstrong/seed/internal/validation"
)

func TestValidateVLANID(t *testing.T) {
	tests := []struct {
		name    string
		vlanID  int
		wantErr bool
		errMsg  string
	}{
		// Valid VLAN IDs
		{"valid min", 1, false, ""},
		{"valid max", 4094, false, ""},
		{"valid mid", 100, false, ""},
		{"valid common", 1000, false, ""},

		// Invalid VLAN IDs
		{"zero", 0, true, "must be between 1 and 4094"},
		{"negative", -1, true, "must be between 1 and 4094"},
		{"reserved 4095", 4095, true, "must be between 1 and 4094"},
		{"too high", 5000, true, "must be between 1 and 4094"},
		{"very large", 65535, true, "must be between 1 and 4094"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateVLANID(tt.vlanID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVLANID(%d) error = %v, wantErr %v", tt.vlanID, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateVLANID(%d) error = %v, want to contain %q", tt.vlanID, err, tt.errMsg)
			}
		})
	}
}

func TestValidatePositiveInt(t *testing.T) {
	tests := []struct {
		name      string
		val       int
		fieldName string
		wantErr   bool
	}{
		{"zero is valid", 0, "count", false},
		{"positive is valid", 100, "count", false},
		{"negative is invalid", -1, "count", true},
		{"large positive", 1000000, "count", false},
		{"large negative", -1000000, "count", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidatePositiveInt(tt.val, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePositiveInt(%d, %q) error = %v, wantErr %v",
					tt.val, tt.fieldName, err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), tt.fieldName) {
				t.Errorf("ValidatePositiveInt(%d, %q) error should contain field name",
					tt.val, tt.fieldName)
			}
		})
	}
}

func TestValidateMTU(t *testing.T) {
	tests := []struct {
		name    string
		mtu     int
		wantErr bool
		errMsg  string
	}{
		// Valid MTU values
		{"min valid 68", 68, false, ""},
		{"standard ethernet 1500", 1500, false, ""},
		{"jumbo frames 9000", 9000, false, ""},
		{"mid value", 4500, false, ""},

		// Invalid MTU values
		{"too low", 67, true, "must be between 68 and 9000"},
		{"too high", 9001, true, "must be between 68 and 9000"},
		{"zero", 0, true, "must be between 68 and 9000"},
		{"negative", -1, true, "must be between 68 and 9000"},
		{"very high", 65535, true, "must be between 68 and 9000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateMTU(tt.mtu)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMTU(%d) error = %v, wantErr %v", tt.mtu, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateMTU(%d) error = %v, want to contain %q", tt.mtu, err, tt.errMsg)
			}
		})
	}
}

func TestValidateDNSAddress(t *testing.T) {
	tests := []struct {
		name    string
		dns     string
		wantErr bool
		errMsg  string
	}{
		// Valid DNS addresses
		{"valid IPv4 Google", "8.8.8.8", false, ""},
		{"valid IPv4 Cloudflare", "1.1.1.1", false, ""},
		{"valid IPv6", "2001:4860:4860::8888", false, ""},
		{"valid IPv6 short", "::1", false, ""},

		// Invalid DNS addresses
		{"empty", "", true, "DNS server address is required"},
		{"hostname", "dns.google.com", true, "must be a valid IP address"},
		{"invalid format", "not-an-ip", true, "must be a valid IP address"},
		{"partial IP", "192.168", true, "must be a valid IP address"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateDNSAddress(tt.dns)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDNSAddress(%q) error = %v, wantErr %v", tt.dns, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateDNSAddress(%q) error = %v, want to contain %q", tt.dns, err, tt.errMsg)
			}
		})
	}
}

func TestValidateDNSServers(t *testing.T) {
	tests := []struct {
		name    string
		servers []string
		wantErr bool
		errMsg  string
	}{
		// Valid DNS servers
		{"empty slice", []string{}, false, ""},
		{"single valid", []string{"8.8.8.8"}, false, ""},
		{"multiple valid", []string{"8.8.8.8", "1.1.1.1"}, false, ""},
		{"mixed IPv4 IPv6", []string{"8.8.8.8", "2001:4860:4860::8888"}, false, ""},

		// Invalid DNS servers
		{"first invalid", []string{"", "1.1.1.1"}, true, "DNS server 1"},
		{"second invalid", []string{"8.8.8.8", "invalid"}, true, "DNS server 2"},
		{"hostname not allowed", []string{"dns.google.com"}, true, "must be a valid IP address"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateDNSServers(tt.servers)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDNSServers(%v) error = %v, wantErr %v", tt.servers, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateDNSServers(%v) error = %v, want to contain %q",
					tt.servers, err, tt.errMsg)
			}
		})
	}
}

func TestValidateStringLength(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		fieldName string
		minLen    int
		maxLen    int
		wantErr   bool
		errMsg    string
	}{
		// Valid lengths
		{"exact min", "a", "field", 1, 10, false, ""},
		{"exact max", "abcdefghij", "field", 1, 10, false, ""},
		{"within range", "hello", "field", 1, 10, false, ""},
		{"empty when allowed", "", "field", 0, 10, false, ""},

		// Invalid lengths
		{"empty when required", "", "username", 1, 64, true, "username is required"},
		{"too short", "ab", "password", 8, 128, true, "must be at least 8 characters"},
		{"too long", "this is way too long", "name", 1, 10, true, "too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStringLength(tt.s, tt.fieldName, tt.minLen, tt.maxLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStringLength(%q, %q, %d, %d) error = %v, wantErr %v",
					tt.s, tt.fieldName, tt.minLen, tt.maxLen, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateStringLength(%q, %q, %d, %d) error = %v, want to contain %q",
					tt.s, tt.fieldName, tt.minLen, tt.maxLen, err, tt.errMsg)
			}
		})
	}
}

func TestValidateIntRange(t *testing.T) {
	tests := []struct {
		name      string
		val       int
		fieldName string
		minVal    int
		maxVal    int
		wantErr   bool
	}{
		// Valid ranges
		{"at min", 1, "port", 1, 65535, false},
		{"at max", 65535, "port", 1, 65535, false},
		{"within range", 8080, "port", 1, 65535, false},
		{"negative in range", -10, "offset", -100, 100, false},

		// Invalid ranges
		{"below min", 0, "port", 1, 65535, true},
		{"above max", 65536, "port", 1, 65535, true},
		{"negative out of range", -101, "offset", -100, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateIntRange(tt.val, tt.fieldName, tt.minVal, tt.maxVal)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIntRange(%d, %q, %d, %d) error = %v, wantErr %v",
					tt.val, tt.fieldName, tt.minVal, tt.maxVal, err, tt.wantErr)
			}
		})
	}
}

func TestValidateFloatRange(t *testing.T) {
	tests := []struct {
		name      string
		val       float64
		fieldName string
		minVal    float64
		maxVal    float64
		wantErr   bool
	}{
		// Valid ranges
		{"at min", 0.0, "percentage", 0.0, 100.0, false},
		{"at max", 100.0, "percentage", 0.0, 100.0, false},
		{"within range", 50.5, "percentage", 0.0, 100.0, false},
		{"negative in range", -10.5, "offset", -100.0, 100.0, false},

		// Invalid ranges
		{"below min", -0.1, "percentage", 0.0, 100.0, true},
		{"above max", 100.1, "percentage", 0.0, 100.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateFloatRange(tt.val, tt.fieldName, tt.minVal, tt.maxVal)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFloatRange(%f, %q, %f, %f) error = %v, wantErr %v",
					tt.val, tt.fieldName, tt.minVal, tt.maxVal, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		fieldName string
		wantErr   bool
		errMsg    string
	}{
		// Valid paths
		{"simple filename", "report.pdf", "file", false, ""},
		{"relative path", "subdir/report.pdf", "file", false, ""},
		{"deep path", "a/b/c/d.txt", "file", false, ""},

		// Invalid paths
		{"empty", "", "path", true, "is required"},
		{"absolute unix", "/etc/passwd", "path", true, "must not be an absolute path"},
		{"absolute windows", "\\Windows\\System32", "path", true, "must not be an absolute path"},
		{"parent traversal", "../../../etc/passwd", "path", true, "must not contain '..'"},
		{"mid traversal", "foo/../bar", "path", true, "must not contain '..'"},
		{"null byte", "file\x00.txt", "path", true, "contains invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidatePath(tt.path, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath(%q, %q) error = %v, wantErr %v",
					tt.path, tt.fieldName, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidatePath(%q, %q) error = %v, want to contain %q",
					tt.path, tt.fieldName, err, tt.errMsg)
			}
		})
	}
}

func TestValidateFilename(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		fieldName string
		wantErr   bool
		errMsg    string
	}{
		// Valid filenames
		{"simple", "report.pdf", "file", false, ""},
		{"with spaces", "my report.pdf", "file", false, ""},
		{"unicode", "reporte_espanol.txt", "file", false, ""},
		{"numbers", "2024-01-01-backup.zip", "file", false, ""},

		// Invalid filenames
		{"empty", "", "filename", true, "is required"},
		{"with forward slash", "foo/bar.txt", "filename", true, "must not contain path separators"},
		{"with backslash", "foo\\bar.txt", "filename", true, "must not contain path separators"},
		{"dot only", ".", "filename", true, "is not a valid filename"},
		{"dotdot only", "..", "filename", true, "is not a valid filename"},
		{"null byte", "file\x00.txt", "filename", true, "contains invalid characters"},
		{"control char", "file\x1f.txt", "filename", true, "contains invalid characters"},
		{"DEL char", "file\x7f.txt", "filename", true, "contains invalid characters"},
		{"too long", strings.Repeat("a", 256), "filename", true, "too long (max 255 characters)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateFilename(tt.filename, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilename(%q, %q) error = %v, wantErr %v",
					tt.filename, tt.fieldName, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateFilename(%q, %q) error = %v, want to contain %q",
					tt.filename, tt.fieldName, err, tt.errMsg)
			}
		})
	}
}

func TestValidateSurveyID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
		errMsg  string
	}{
		// Valid survey IDs
		{"alphanumeric", "survey123", false, ""},
		{"with hyphen", "survey-123", false, ""},
		{"with underscore", "survey_123", false, ""},
		{"mixed", "Survey_2024-01", false, ""},
		{"uppercase", "SURVEY123", false, ""},
		{"single char", "a", false, ""},
		{"max length", strings.Repeat("a", 64), false, ""},

		// Invalid survey IDs
		{"empty", "", true, "is required"},
		{"too long", strings.Repeat("a", 65), true, "too long (max 64 characters)"},
		{"with space", "survey 123", true, "contains invalid characters"},
		{"with special char", "survey@123", true, "contains invalid characters"},
		{"with dot", "survey.123", true, "contains invalid characters"},
		{"with slash", "survey/123", true, "contains invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateSurveyID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSurveyID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateSurveyID(%q) error = %v, want to contain %q",
					tt.id, err, tt.errMsg)
			}
		})
	}
}

func TestValidateImageDataURL(t *testing.T) {
	tests := []struct {
		name         string
		dataURL      string
		maxSizeBytes int
		wantErr      bool
		errMsg       string
	}{
		// Valid data URLs
		{
			name:         "valid PNG",
			dataURL:      "data:image/png;base64,iVBORw0KGgo=",
			maxSizeBytes: 1000,
			wantErr:      false,
		},
		{
			name:         "valid JPEG",
			dataURL:      "data:image/jpeg;base64,/9j/4AAQ",
			maxSizeBytes: 1000,
			wantErr:      false,
		},
		{
			name:         "valid JPG",
			dataURL:      "data:image/jpg;base64,/9j/4AAQ",
			maxSizeBytes: 1000,
			wantErr:      false,
		},
		{
			name:         "valid GIF",
			dataURL:      "data:image/gif;base64,R0lGODlh",
			maxSizeBytes: 1000,
			wantErr:      false,
		},
		{
			name:         "valid WebP",
			dataURL:      "data:image/webp;base64,UklGR",
			maxSizeBytes: 1000,
			wantErr:      false,
		},

		// Invalid data URLs
		{
			name:         "empty",
			dataURL:      "",
			maxSizeBytes: 1000,
			wantErr:      true,
			errMsg:       "image data is required",
		},
		{
			name:         "no data prefix",
			dataURL:      "image/png;base64,iVBORw0KGgo=",
			maxSizeBytes: 1000,
			wantErr:      true,
			errMsg:       "must be a data URL",
		},
		{
			name:         "no comma",
			dataURL:      "data:image/png;base64",
			maxSizeBytes: 1000,
			wantErr:      true,
			errMsg:       "invalid image data format",
		},
		{
			name:         "unsupported type BMP",
			dataURL:      "data:image/bmp;base64,Qk0=",
			maxSizeBytes: 1000,
			wantErr:      true,
			errMsg:       "unsupported image type",
		},
		{
			name:         "unsupported type SVG",
			dataURL:      "data:image/svg+xml;base64,PHN2Zz4=",
			maxSizeBytes: 1000,
			wantErr:      true,
			errMsg:       "unsupported image type",
		},
		{
			name:         "too large",
			dataURL:      "data:image/png;base64," + strings.Repeat("A", 1000),
			maxSizeBytes: 100,
			wantErr:      true,
			errMsg:       "image data too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateImageDataURL(tt.dataURL, tt.maxSizeBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateImageDataURL(%q, %d) error = %v, wantErr %v",
					tt.dataURL, tt.maxSizeBytes, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateImageDataURL(%q, %d) error = %v, want to contain %q",
					tt.dataURL, tt.maxSizeBytes, err, tt.errMsg)
			}
		})
	}
}

func TestValidateURLAdditionalCases(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		// Test URL parse errors
		{"invalid chars in host", "http://example[.com", true, "invalid"},
		// Test empty host after prefix
		{"only scheme", "https://", true, "must have a valid host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateURL(%q) error = %v, want to contain %q",
					tt.url, err, tt.errMsg)
			}
		})
	}
}

func TestValidateNetmaskAdditionalCases(t *testing.T) {
	tests := []struct {
		name    string
		netmask string
		wantErr bool
		errMsg  string
	}{
		// Valid netmasks
		{"/32 full", "255.255.255.255", false, ""},

		// Edge case: invalid netmask that is valid IP but not valid subnet mask
		{"invalid mask value", "255.0.255.0", true, "not a valid subnet mask"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateNetmask(tt.netmask)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNetmask(%q) error = %v, wantErr %v", tt.netmask, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateNetmask(%q) error = %v, want to contain %q",
					tt.netmask, err, tt.errMsg)
			}
		})
	}
}

func TestIsValidURLAdditionalCases(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// Test parse error case
		{"invalid URL chars", "http://example[.com", false},
		// Test invalid host
		{"invalid host chars", "http://!!!invalid!!!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validation.IsValidURL(tt.url)
			if got != tt.expected {
				t.Errorf("IsValidURL(%q) = %v, want %v", tt.url, got, tt.expected)
			}
		})
	}
}
