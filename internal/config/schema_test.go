package config_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
)

func TestNewSchemaValidator(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create schema validator: %v", err)
	}
	if validator == nil {
		t.Fatal("validator is nil")
	}
	if validator.Schema() == nil {
		t.Fatal("validator.Schema() is nil")
	}
}

func TestValidateConfig_ValidDefault(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	// Default config should be valid
	cfg := config.DefaultConfig()
	errors := validator.ValidateConfig(cfg)
	if errors != nil {
		t.Errorf("default config should be valid, got errors: %+v", errors)
		for _, e := range errors {
			t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
		}
	}
}

func TestValidateConfig_InvalidServerPort(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name string
		port int
		want bool // true = should have errors
	}{
		{"port too low", 0, true},
		{"port negative", -1, true},
		{"port too high", 65536, true},
		{"port valid min", 1, false},
		{"port valid max", 65535, false},
		{"port valid common", 8443, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Server.Port = tt.port
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for port %d, got none", tt.port)
				} else {
					t.Errorf("expected no validation errors for port %d, got: %+v", tt.port, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_InvalidVLANID(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name    string
		enabled bool
		id      int
		want    bool // true = should have errors
	}{
		// Schema enforces 0-4094 range regardless of enabled status
		// Additional business logic (0 invalid when enabled) is in Config.Validate()
		{"disabled vlan id 0", false, 0, false},
		{"disabled vlan id 100", false, 100, false},
		{"disabled vlan id too high", false, 5000, true}, // > 4094 is invalid
		{
			"enabled vlan id 0",
			true,
			0,
			false,
		}, // Schema allows 0, business logic rejects it
		{"enabled vlan id -1", true, -1, true},     // negative is invalid
		{"enabled vlan id 4095", true, 4095, true}, // > 4094 is invalid
		{"enabled vlan id 1", true, 1, false},
		{"enabled vlan id 4094", true, 4094, false},
		{"enabled vlan id 100", true, 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.VLAN.Enabled = tt.enabled
			cfg.VLAN.ID = tt.id
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors, got none")
				} else {
					t.Errorf("expected no validation errors, got: %+v", errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_InvalidIPMode(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name string
		mode string
		want bool // true = should have errors
	}{
		{"dhcp mode", "dhcp", false},
		{"static mode", "static", false},
		{"invalid mode", "auto", true},
		{"empty mode", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.IP.Mode = tt.mode
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for mode %q, got none", tt.mode)
				} else {
					t.Errorf("expected no validation errors for mode %q, got: %+v", tt.mode, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_PortPreset(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name   string
		preset config.PortPreset
		want   bool // true = should have errors
	}{
		{"common preset", config.PortPresetCommon, false},
		{"secure preset", config.PortPresetSecure, false},
		{"insecure preset", config.PortPresetInsecure, false},
		{"custom preset", config.PortPresetCustom, false},
		{"invalid preset", config.PortPreset("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.NetworkDiscovery.Options.PortScan.Preset = tt.preset
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for preset %q, got none", tt.preset)
				} else {
					t.Errorf("expected no validation errors for preset %q, got: %+v", tt.preset, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_DiscoveryProtocol(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name     string
		protocol string
		want     bool // true = should have errors
	}{
		{"auto protocol", "auto", false},
		{"lldp protocol", "lldp", false},
		{"cdp protocol", "cdp", false},
		{"edp protocol", "edp", false},
		{"fdp protocol", "fdp", false},
		{"invalid protocol", "snmp", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Discovery.Protocol = tt.protocol
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for protocol %q, got none", tt.protocol)
				} else {
					t.Errorf("expected no validation errors for protocol %q, got: %+v", tt.protocol, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_IperfProtocol(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name     string
		protocol string
		want     bool // true = should have errors
	}{
		{"tcp protocol", "tcp", false},
		{"udp protocol", "udp", false},
		{"invalid protocol", "sctp", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Iperf.Protocol = tt.protocol
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for protocol %q, got none", tt.protocol)
				} else {
					t.Errorf("expected no validation errors for protocol %q, got: %+v", tt.protocol, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_IperfDirection(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name      string
		direction string
		want      bool // true = should have errors
	}{
		{"upload direction", "upload", false},
		{"download direction", "download", false},
		{"bidirectional direction", "bidirectional", false},
		{"invalid direction", "duplex", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Iperf.Direction = tt.direction
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for direction %q, got none", tt.direction)
				} else {
					t.Errorf("expected no validation errors for direction %q, got: %+v", tt.direction, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_LogLevel(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name  string
		level string
		want  bool // true = should have errors
	}{
		{"debug level", "debug", false},
		{"info level", "info", false},
		{"warn level", "warn", false},
		{"warning level", "warning", false},
		{"error level", "error", false},
		{"invalid level", "trace", true},
		{"invalid level fatal", "fatal", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Logging.Level = tt.level
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for level %q, got none", tt.level)
				} else {
					t.Errorf("expected no validation errors for level %q, got: %+v", tt.level, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_LogFormat(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name   string
		format string
		want   bool // true = should have errors
	}{
		{"text format", "text", false},
		{"json format", "json", false},
		{"invalid format", "xml", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Logging.Format = tt.format
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for format %q, got none", tt.format)
				} else {
					t.Errorf("expected no validation errors for format %q, got: %+v", tt.format, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_SignalThreshold(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name     string
		warning  int
		critical int
		want     bool // true = should have errors
	}{
		{"valid thresholds", -70, -80, false},
		{"valid thresholds min", -100, -100, false},
		{"valid thresholds max", 0, 0, false},
		{"warning too low", -101, -80, true},
		{"critical too low", -70, -101, true},
		{"warning too high", 1, -80, true},
		{"critical too high", -70, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Thresholds.WiFi.Signal.Warning = tt.warning
			cfg.Thresholds.WiFi.Signal.Critical = tt.critical
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors, got none")
				} else {
					t.Errorf("expected no validation errors, got: %+v", errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_SNMPPort(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name string
		port int
		want bool // true = should have errors
	}{
		{"valid default port", 161, false},
		{"valid custom port", 1161, false},
		{"port too low", 0, true},
		{"port too high", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.SNMP.Port = tt.port
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for port %d, got none", tt.port)
				} else {
					t.Errorf("expected no validation errors for port %d, got: %+v", tt.port, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_SNMPRetries(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name    string
		retries int
		want    bool // true = should have errors
	}{
		{"retries 0", 0, false},
		{"retries 5", 5, false},
		{"retries 10", 10, false},
		{"retries negative", -1, true},
		{"retries too high", 11, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.SNMP.Retries = tt.retries
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for retries %d, got none", tt.retries)
				} else {
					t.Errorf("expected no validation errors for retries %d, got: %+v", tt.retries, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_SNMPMaxRepetitions(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name           string
		maxRepetitions uint32
		want           bool // true = should have errors
	}{
		{"max_repetitions 1", 1, false},
		{"max_repetitions 10", 10, false},
		{"max_repetitions 50", 50, false},
		{"max_repetitions 0", 0, true},
		{"max_repetitions too high", 51, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.SNMP.MaxRepetitions = tt.maxRepetitions
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for max_repetitions %d, got none", tt.maxRepetitions)
				} else {
					t.Errorf("expected no validation errors for max_repetitions %d, got: %+v", tt.maxRepetitions, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_VulnerabilitySeverity(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name     string
		severity string
		want     bool // true = should have errors
	}{
		{"low severity", "low", false},
		{"medium severity", "medium", false},
		{"high severity", "high", false},
		{"critical severity", "critical", false},
		{"invalid severity", "extreme", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Security.VulnerabilityScanning.SeverityThreshold = tt.severity
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for severity %q, got none", tt.severity)
				} else {
					t.Errorf("expected no validation errors for severity %q, got: %+v", tt.severity, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_DurationFormat(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name     string
		duration time.Duration
		want     bool // true = should have errors
	}{
		{"valid duration 5s", 5 * time.Second, false},
		{"valid duration 100ms", 100 * time.Millisecond, false},
		{"valid duration 1h", 1 * time.Hour, false},
		{"valid duration 24h", 24 * time.Hour, false},
		{"valid duration 500ms", 500 * time.Millisecond, false},
		{"valid duration 30s", 30 * time.Second, false},
		// Note: Invalid duration formats are caught at the Go type level,
		// not by JSON schema since time.Duration marshals as strings
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.DNS.Timeout = tt.duration
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for duration %v, got none", tt.duration)
				} else {
					t.Errorf("expected no validation errors for duration %v, got: %+v", tt.duration, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_HTTPExpectedStatus(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name   string
		status int
		want   bool // true = should have errors
	}{
		{"status 200", 200, false},
		{"status 404", 404, false},
		{"status 100", 100, false},
		{"status 599", 599, false},
		{"status too low", 99, true},
		{"status too high", 600, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.HealthChecks.HTTPEndpoints = []config.HTTPEndpoint{
				{
					Name:           "test",
					URL:            "http://example.com",
					ExpectedStatus: tt.status,
					Enabled:        true,
				},
			}
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for status %d, got none", tt.status)
				} else {
					t.Errorf("expected no validation errors for status %d, got: %+v", tt.status, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateWithSchema(t *testing.T) {
	// Test the convenience function
	cfg := config.DefaultConfig()
	errors := config.ValidateWithSchema(cfg)
	if errors != nil {
		t.Errorf("default config should be valid, got errors: %+v", errors)
		for _, e := range errors {
			t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
		}
	}
}

func TestValidateWithSchema_InvalidConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.Port = 99999 // Invalid port
	errors := config.ValidateWithSchema(cfg)
	if len(errors) == 0 {
		t.Error("expected validation errors for invalid port, got none")
	}
}

func TestValidateConfig_WorkerLimits(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name    string
		workers int
		want    bool // true = should have errors
	}{
		{"workers 1", 1, false},
		{"workers 50", 50, false},
		{"workers 500", 500, false},
		{"workers 0", 0, true},
		{"workers negative", -1, true},
		{"workers too high", 501, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.NetworkDiscovery.ARPScanWorkers = tt.workers
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for workers %d, got none", tt.workers)
				} else {
					t.Errorf("expected no validation errors for workers %d, got: %+v", tt.workers, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}

func TestValidateConfig_StartupRetries(t *testing.T) {
	validator, err := config.NewSchemaValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name    string
		retries int
		want    bool // true = should have errors
	}{
		{"retries 0", 0, false},
		{"retries 3", 3, false},
		{"retries 10", 10, false},
		{"retries negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Interface.StartupRetries = tt.retries
			errors := validator.ValidateConfig(cfg)

			hasErrors := len(errors) > 0
			if hasErrors != tt.want {
				if tt.want {
					t.Errorf("expected validation errors for retries %d, got none", tt.retries)
				} else {
					t.Errorf("expected no validation errors for retries %d, got: %+v", tt.retries, errors)
					for _, e := range errors {
						t.Logf("  - Path: %s, Message: %s", e.Path, e.Message)
					}
				}
			}
		})
	}
}
