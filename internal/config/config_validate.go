package config

// config_validate.go contains the Validate entry point, per-section
// validateXConfig helpers, and the one deprecation-warning helper that
// inspects the loaded config at startup.

import (
	"fmt"
	"strings"

	"github.com/krisarmstrong/seed/internal/logging"
)

// Validate checks if the configuration values are valid.
// This prevents the server from starting with invalid configuration.
func (c *Config) Validate() error {
	var errs []string
	errs = append(errs, c.validateServerConfig()...)
	errs = append(errs, c.validateInterfaceConfig()...)
	errs = append(errs, c.validateVLANConfig()...)
	errs = append(errs, c.validateIPConfig()...)
	errs = append(errs, c.validateTimeouts()...)
	errs = append(errs, c.validateConcurrency()...)
	errs = append(errs, c.validateAuthConfig()...)
	errs = append(errs, c.validateSNMPConfig()...)
	errs = append(errs, c.validateLoggingConfig()...)
	errs = append(errs, c.validateCORSConfig()...)

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// validateCORSConfig refuses `*` in Security.AllowedOrigins (#1256). Seed's
// corsMiddleware always responds with `Access-Control-Allow-Credentials:
// true`, and per the CORS spec wildcard origin with credentials is an
// invalid pair: browsers reject it. Honoring `*` here is at best
// non-functional and at worst the kind of subtle config footgun where
// credentialed responses leak across origins if a UA ever does honor it.
// Operators must list explicit origins; the WARN-only behavior of the
// pre-#1256 path silently allowed insecure setups.
func (c *Config) validateCORSConfig() []string {
	for _, origin := range c.Security.AllowedOrigins {
		if strings.TrimSpace(origin) == "*" {
			return []string{
				"security.allowed_origins: `*` is not allowed because " +
					"credentialed CORS (Access-Control-Allow-Credentials: true) " +
					"is invalid with a wildcard origin per the CORS spec; " +
					"list explicit origins (e.g. https://seed.example.com)",
			}
		}
	}
	return nil
}

// validateServerConfig checks server port configuration.
func (c *Config) validateServerConfig() []string {
	var errs []string
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(
			errs,
			fmt.Sprintf("server.port must be between 1-65535, got %d", c.Server.Port),
		)
	}
	return errs
}

// validateInterfaceConfig checks interface startup configuration.
func (c *Config) validateInterfaceConfig() []string {
	var errs []string
	if c.Interface.StartupRetries < 0 {
		errs = append(
			errs,
			fmt.Sprintf(
				"interface.startup_retries must be >= 0, got %d",
				c.Interface.StartupRetries,
			),
		)
	}
	if c.Interface.StartupRetryWait < 0 {
		errs = append(
			errs,
			fmt.Sprintf(
				"interface.startup_retry_wait must be >= 0, got %s",
				c.Interface.StartupRetryWait,
			),
		)
	}
	return errs
}

// validateVLANConfig checks VLAN ID configuration.
func (c *Config) validateVLANConfig() []string {
	var errs []string
	if c.VLAN.Enabled && (c.VLAN.ID < 1 || c.VLAN.ID > 4094) {
		errs = append(errs, fmt.Sprintf("vlan.id must be between 1-4094, got %d", c.VLAN.ID))
	}
	return errs
}

// validateIPConfig checks IP mode and static IP configuration.
func (c *Config) validateIPConfig() []string {
	var errs []string
	if c.IP.Mode != ipModeDHCP && c.IP.Mode != ipModeStatic {
		errs = append(errs, fmt.Sprintf("ip.mode must be 'dhcp' or 'static', got '%s'", c.IP.Mode))
	}
	if c.IP.Mode != ipModeStatic {
		return errs
	}
	// Fixes #896: Check for nil Static config before accessing fields
	if c.IP.Static == nil {
		return append(errs, "ip.static is required when ip.mode is 'static'")
	}
	errs = append(errs, c.validateStaticIPFields()...)
	return errs
}

// validateStaticIPFields validates individual static IP configuration fields.
func (c *Config) validateStaticIPFields() []string {
	var errs []string
	if c.IP.Static.Address == "" {
		errs = append(errs, "ip.static.address is required when ip.mode is 'static'")
	}
	if c.IP.Static.Netmask == "" {
		errs = append(errs, "ip.static.netmask is required when ip.mode is 'static'")
	}
	if c.IP.Static.Gateway == "" {
		errs = append(errs, "ip.static.gateway is required when ip.mode is 'static'")
	}
	return errs
}

// validateTimeouts checks all timeout configurations are positive.
func (c *Config) validateTimeouts() []string {
	var errs []string
	if c.Discovery.Timeout <= 0 {
		errs = append(errs, "discovery.timeout must be positive")
	}
	if c.NetworkDiscovery.PingTimeout <= 0 {
		errs = append(errs, "network_discovery.ping_timeout must be positive")
	}
	if c.NetworkDiscovery.ScanTimeout <= 0 {
		errs = append(errs, "network_discovery.scan_timeout must be positive")
	}
	if c.DNS.Timeout <= 0 {
		errs = append(errs, "dns.timeout must be positive")
	}
	return errs
}

// validateConcurrency checks worker/concurrency limits.
func (c *Config) validateConcurrency() []string {
	var errs []string
	if c.NetworkDiscovery.ARPScanWorkers < 1 || c.NetworkDiscovery.ARPScanWorkers > 500 {
		errs = append(
			errs,
			fmt.Sprintf(
				"network_discovery.arp_scan_workers must be between 1-500, got %d",
				c.NetworkDiscovery.ARPScanWorkers,
			),
		)
	}
	return errs
}

// validateAuthConfig checks authentication configuration.
func (c *Config) validateAuthConfig() []string {
	var errs []string
	if c.Auth.SessionTimeout <= 0 {
		errs = append(errs, "auth.session_timeout must be positive")
	}
	if c.Auth.DefaultUsername == "" {
		errs = append(errs, "auth.default_username is required")
	}
	if c.Auth.DefaultPasswordHash == "" {
		errs = append(errs, "auth.default_password_hash is required")
	}
	return errs
}

// validateSNMPConfig checks SNMP configuration.
func (c *Config) validateSNMPConfig() []string {
	var errs []string
	if c.SNMP.Port < 1 || c.SNMP.Port > 65535 {
		errs = append(errs, fmt.Sprintf("snmp.port must be between 1-65535, got %d", c.SNMP.Port))
	}
	if c.SNMP.Retries < 0 || c.SNMP.Retries > 10 {
		errs = append(
			errs,
			fmt.Sprintf("snmp.retries must be between 0-10, got %d", c.SNMP.Retries),
		)
	}
	if c.SNMP.Timeout <= 0 {
		errs = append(errs, "snmp.timeout must be positive")
	}
	return errs
}

// WarnDeprecatedSNMPSettings logs warnings for deprecated SNMP configurations.
// This function should be called after logging is initialized.
func (c *Config) WarnDeprecatedSNMPSettings() {
	c.RLock()
	defer c.RUnlock()

	// Check for MD5 authentication protocol in SNMPv3 credentials
	// MD5 is cryptographically broken and will be removed in the next major version
	for i := range c.SNMP.V3Credentials {
		cred := &c.SNMP.V3Credentials[i]
		if cred.AuthProtocol == "MD5" {
			logging.GetLogger().Warn(
				"SNMP MD5 authentication is deprecated and will be removed in the next major version",
				"credential_name",
				cred.Name,
				"username",
				cred.Username,
				"recommendation",
				"Use SHA256 or SHA512 for secure authentication",
			)
		}
	}
}

// validateLoggingConfig checks logging configuration.
func (c *Config) validateLoggingConfig() []string {
	var errs []string

	// Validate log level
	validLevels := map[string]bool{
		"debug": true, logLevelInfo: true, "warn": true, "warning": true, "error": true,
	}
	level := strings.ToLower(c.Logging.Level)
	if level != "" && !validLevels[level] {
		errs = append(
			errs,
			fmt.Sprintf(
				"logging.level must be one of debug, info, warn, error; got %q",
				c.Logging.Level,
			),
		)
	}

	// Validate format
	format := strings.ToLower(c.Logging.Format)
	if format != "" && format != logFormatText && format != logFormatJSON {
		errs = append(
			errs,
			fmt.Sprintf("logging.format must be 'text' or 'json'; got %q", c.Logging.Format),
		)
	}

	// Validate rotation settings
	if c.Logging.MaxSize < 0 {
		errs = append(errs, fmt.Sprintf("logging.max_size must be >= 0, got %d", c.Logging.MaxSize))
	}
	if c.Logging.MaxBackups < 0 {
		errs = append(
			errs,
			fmt.Sprintf("logging.max_backups must be >= 0, got %d", c.Logging.MaxBackups),
		)
	}
	if c.Logging.MaxAge < 0 {
		errs = append(errs, fmt.Sprintf("logging.max_age must be >= 0, got %d", c.Logging.MaxAge))
	}

	return errs
}
