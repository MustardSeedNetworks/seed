package main

import (
	"testing"
)

func TestAllConstantsHaveValidValues(t *testing.T) {

	tests := []struct {
		name     string
		value    int
		minValue int
		maxValue int
	}{
		{
			name:     "logBroadcasterBufferSize",
			value:    GetLogBroadcasterBufferSize(),
			minValue: 100,
			maxValue: 10000,
		},
		{
			name:     "signalChannelBufferSize",
			value:    GetSignalChannelBufferSize(),
			minValue: 1,
			maxValue: 10,
		},
		{
			name:     "shutdownTimeoutSeconds",
			value:    GetShutdownTimeoutSeconds(),
			minValue: 5,
			maxValue: 120,
		},
		{
			name:     "userCheckTimeoutSeconds",
			value:    GetUserCheckTimeoutSeconds(),
			minValue: 1,
			maxValue: 30,
		},
		{
			name:     "commandTimeoutSeconds",
			value:    GetCommandTimeoutSeconds(),
			minValue: 5,
			maxValue: 120,
		},
		{
			name:     "defaultPasswordLength",
			value:    GetDefaultPasswordLength(),
			minValue: 16,
			maxValue: 64,
		},
		{
			name:     "expectedLinuxReleaseParts",
			value:    GetExpectedLinuxReleaseParts(),
			minValue: 2,
			maxValue: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			if tc.value < tc.minValue {
				t.Errorf("%s = %d, should be >= %d", tc.name, tc.value, tc.minValue)
			}
			if tc.value > tc.maxValue {
				t.Errorf("%s = %d, should be <= %d", tc.name, tc.value, tc.maxValue)
			}
		})
	}
}

func TestRedactedValueConstantFormat(t *testing.T) {

	redacted := GetRedactedValue()

	if redacted == "" {
		t.Error("redactedValue should not be empty")
	}

	// Should be visually distinct as redacted
	if redacted != "[REDACTED]" {
		t.Errorf("redactedValue should be '[REDACTED]', got %q", redacted)
	}
}

func TestSystemdServiceTemplateValid(t *testing.T) {

	tmpl := GetSystemdServiceTemplate()

	// Must have all required systemd sections
	requiredSections := []string{"[Unit]", "[Service]", "[Install]"}
	for _, section := range requiredSections {
		if !containsSubstring(tmpl, section) {
			t.Errorf("systemd service template should contain %q section", section)
		}
	}

	// Must have required directives
	requiredDirectives := []string{
		"Description=",
		"Type=simple",
		"ExecStart=",
		"Restart=",
		"WantedBy=",
	}
	for _, directive := range requiredDirectives {
		if !containsSubstring(tmpl, directive) {
			t.Errorf("systemd service template should contain %q directive", directive)
		}
	}

	// Must have template placeholders
	requiredPlaceholders := []string{
		"{{.BinaryPath}}",
		"{{.User}}",
		"{{.Group}}",
	}
	for _, placeholder := range requiredPlaceholders {
		if !containsSubstring(tmpl, placeholder) {
			t.Errorf("systemd service template should contain %q placeholder", placeholder)
		}
	}
}

func TestUserServiceTemplateValid(t *testing.T) {

	tmpl := GetUserServiceTemplate()

	// Must have all required systemd sections
	requiredSections := []string{"[Unit]", "[Service]", "[Install]"}
	for _, section := range requiredSections {
		if !containsSubstring(tmpl, section) {
			t.Errorf("user service template should contain %q section", section)
		}
	}

	// User service should target default.target
	if !containsSubstring(tmpl, "WantedBy=default.target") {
		t.Error("user service template should use WantedBy=default.target")
	}

	// User service should NOT contain user/group directives
	forbiddenContent := []string{"{{.User}}", "{{.Group}}", "{{.DataDir}}", "{{.ConfigDir}}"}
	for _, content := range forbiddenContent {
		if containsSubstring(tmpl, content) {
			t.Errorf("user service template should NOT contain %q", content)
		}
	}
}

func TestSystemdTemplateSecurityHardening(t *testing.T) {

	tmpl := GetSystemdServiceTemplate()

	// Check for security hardening directives
	securityDirectives := []string{
		"ProtectSystem=",
		"ProtectHome=",
		"PrivateTmp=",
	}

	for _, directive := range securityDirectives {
		if !containsSubstring(tmpl, directive) {
			t.Errorf("systemd service template should contain security directive %q", directive)
		}
	}
}

func TestSystemdTemplateHasReadWritePaths(t *testing.T) {

	tmpl := GetSystemdServiceTemplate()

	// When using ProtectSystem=strict, ReadWritePaths must be defined
	if containsSubstring(tmpl, "ProtectSystem=strict") {
		if !containsSubstring(tmpl, "ReadWritePaths=") {
			t.Error("systemd template with ProtectSystem=strict should define ReadWritePaths")
		}
	}
}

func TestSystemdTemplateHasCapabilities(t *testing.T) {

	tmpl := GetSystemdServiceTemplate()

	// Check for setcap in ExecStartPre for network capabilities
	if !containsSubstring(tmpl, "setcap") || !containsSubstring(tmpl, "cap_net_raw") {
		t.Error("systemd template should set network capabilities via setcap")
	}
}

func TestTimeoutConstantsRelationships(t *testing.T) {

	userCheck := GetUserCheckTimeoutSeconds()
	command := GetCommandTimeoutSeconds()
	shutdown := GetShutdownTimeoutSeconds()

	// User check should be faster than command timeout
	if userCheck >= command {
		t.Errorf(
			"userCheckTimeoutSeconds (%d) should be less than commandTimeoutSeconds (%d)",
			userCheck,
			command,
		)
	}

	// Shutdown should allow for graceful cleanup
	if shutdown < 10 {
		t.Errorf("shutdownTimeoutSeconds (%d) should be at least 10 seconds", shutdown)
	}
}

func TestBufferSizesAreReasonable(t *testing.T) {

	logBuffer := GetLogBroadcasterBufferSize()
	signalBuffer := GetSignalChannelBufferSize()

	// Log buffer should be much larger than signal buffer
	if logBuffer <= signalBuffer*10 {
		t.Errorf(
			"logBroadcasterBufferSize (%d) should be much larger than signalChannelBufferSize (%d)",
			logBuffer,
			signalBuffer,
		)
	}

	// Log buffer should handle burst of log entries
	if logBuffer < 500 {
		t.Errorf("logBroadcasterBufferSize (%d) should be at least 500 for burst handling", logBuffer)
	}
}

func TestPasswordLengthMeetsSecurity(t *testing.T) {

	pwdLen := GetDefaultPasswordLength()

	// NIST recommends at least 8 characters, but for auto-generated passwords
	// we should use much longer
	if pwdLen < 16 {
		t.Errorf("defaultPasswordLength (%d) should be at least 16 for security", pwdLen)
	}

	// But not excessively long that it causes issues
	if pwdLen > 64 {
		t.Errorf("defaultPasswordLength (%d) should not exceed 64", pwdLen)
	}
}
