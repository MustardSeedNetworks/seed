package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

// SystemConfig contains the minimal configuration needed to bootstrap the system.
// This is loaded from config.json and can be overridden by environment variables.
// All user-configurable settings (profiles) are stored in the database.
type SystemConfig struct {
	// Version is the schema version for migrations.
	Version int `json:"version"`

	// Database configuration for SQLite.
	Database SystemDatabaseConfig `json:"database"`

	// Server configuration for HTTP/HTTPS.
	Server SystemServerConfig `json:"server"`

	// Logging configuration.
	Logging SystemLoggingConfig `json:"logging"`

	// Auth configuration for session management.
	Auth SystemAuthConfig `json:"auth"`

	// Security configuration for rate limiting and CORS.
	Security SystemSecurityConfig `json:"security"`

	// MCP configuration for AI assistant integration (optional).
	MCP SystemMCPConfig `json:"mcp,omitzero"`
}

// SystemDatabaseConfig contains database settings.
type SystemDatabaseConfig struct {
	// Path to the SQLite database file.
	// Env: SEED_DB_PATH
	Path string `json:"path"`

	// RetentionDays sets how many days of historical data to keep (0 = forever).
	// Env: SEED_DB_RETENTION_DAYS
	RetentionDays int `json:"retention_days"`

	// EnableWAL enables Write-Ahead Logging for better concurrency.
	// Env: SEED_DB_ENABLE_WAL
	EnableWAL bool `json:"enable_wal"`

	// MaxConnections sets the maximum number of database connections.
	// Env: SEED_DB_MAX_CONNECTIONS
	MaxConnections int `json:"max_connections"`
}

// SystemServerConfig contains HTTP server settings.
type SystemServerConfig struct {
	// Port is the HTTPS port to listen on.
	// Env: SEED_HTTP_PORT
	Port int `json:"port"`

	// HTTPS enables TLS.
	// Env: SEED_HTTPS_ENABLED
	HTTPS bool `json:"https"`

	// HTTPRedirectPort is the port for HTTP->HTTPS redirect (0 = disabled).
	// Env: SEED_HTTP_REDIRECT_PORT
	HTTPRedirectPort int `json:"http_redirect_port,omitempty"`

	// CertFile is the path to the TLS certificate.
	// Env: SEED_TLS_CERT_FILE
	CertFile string `json:"cert_file"`

	// KeyFile is the path to the TLS private key.
	// Env: SEED_TLS_KEY_FILE
	KeyFile string `json:"key_file"`

	// ACME configuration for Let's Encrypt (optional).
	ACME SystemACMEConfig `json:"acme,omitzero"`
}

// SystemACMEConfig contains ACME/Let's Encrypt settings.
type SystemACMEConfig struct {
	// Enabled enables automatic certificate management.
	// Env: SEED_ACME_ENABLED
	Enabled bool `json:"enabled"`

	// Domain is the domain name for the certificate.
	// Env: SEED_ACME_DOMAIN
	Domain string `json:"domain"`

	// Email is the contact email for Let's Encrypt notifications.
	// Env: SEED_ACME_EMAIL
	Email string `json:"email"`

	// CacheDir is the directory to cache certificates.
	// Env: SEED_ACME_CACHE_DIR
	CacheDir string `json:"cache_dir,omitempty"`

	// Staging uses Let's Encrypt staging server (for testing).
	// Env: SEED_ACME_STAGING
	Staging bool `json:"staging,omitempty"`
}

// SystemLoggingConfig contains logging settings.
type SystemLoggingConfig struct {
	// Level is the log level (DEBUG, INFO, WARN, ERROR).
	// Env: SEED_LOG_LEVEL
	Level string `json:"level"`

	// Format is the log format (text or json).
	// Env: SEED_LOG_FORMAT
	Format string `json:"format"`

	// AddSource includes file:line in logs.
	// Env: SEED_LOG_ADD_SOURCE
	AddSource bool `json:"add_source"`

	// File is the log file path (empty = stdout only).
	// Env: SEED_LOG_FILE
	File string `json:"file"`

	// MaxSize is the max MB per log file before rotation.
	// Env: SEED_LOG_MAX_SIZE
	MaxSize int `json:"max_size"`

	// MaxBackups is the number of old files to keep.
	// Env: SEED_LOG_MAX_BACKUPS
	MaxBackups int `json:"max_backups"`

	// MaxAge is the days to keep old files.
	// Env: SEED_LOG_MAX_AGE
	MaxAge int `json:"max_age"`

	// Compress enables compression of rotated files.
	// Env: SEED_LOG_COMPRESS
	Compress bool `json:"compress"`
}

// SystemAuthConfig contains authentication settings.
type SystemAuthConfig struct {
	// DefaultUsername is the initial admin username.
	// Env: SEED_AUTH_DEFAULT_USERNAME
	DefaultUsername string `json:"default_username"`

	// DefaultPasswordHash is the bcrypt hash of the default password.
	// Env: SEED_AUTH_DEFAULT_PASSWORD_HASH
	DefaultPasswordHash string `json:"default_password_hash"`

	// SessionTimeout is the session duration.
	// Env: SEED_AUTH_SESSION_TIMEOUT (e.g., "24h")
	SessionTimeout time.Duration `json:"session_timeout"`

	// JWTSecret is the secret for signing JWT tokens.
	// Env: SEED_AUTH_JWT_SECRET
	JWTSecret string `json:"jwt_secret,omitempty"`

	// SSO contains SSO provider configurations.
	SSO SystemSSOConfig `json:"sso,omitzero"`
}

// SystemSSOConfig contains SSO settings.
type SystemSSOConfig struct {
	Providers []SystemSSOProviderConfig `json:"providers,omitempty"`
}

// SystemSSOProviderConfig contains settings for a single SSO provider.
type SystemSSOProviderConfig struct {
	Enabled      bool     `json:"enabled"`
	Name         string   `json:"name"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURL  string   `json:"redirect_url"`
	Scopes       []string `json:"scopes,omitempty"`
	TenantID     string   `json:"tenant_id,omitempty"` // Microsoft only
}

// SystemSecurityConfig contains security settings.
type SystemSecurityConfig struct {
	// AllowedOrigins specifies explicit origins allowed for CORS and WebSocket.
	// If empty, defaults to RFC 1918 private network ranges.
	// Use "*" to allow all origins (not recommended for production).
	// Env: SEED_SECURITY_ALLOWED_ORIGINS (comma-separated)
	AllowedOrigins []string `json:"allowed_origins"`

	// RateLimitPerMinute limits API requests per client per minute.
	// Env: SEED_SECURITY_RATE_LIMIT
	RateLimitPerMinute int `json:"rate_limit_per_minute"`

	// VulnerabilityScanning configures CVE scanning.
	VulnerabilityScanning SystemVulnScanConfig `json:"vulnerability_scanning,omitzero"`
}

// SystemVulnScanConfig contains vulnerability scanning settings.
type SystemVulnScanConfig struct {
	Enabled           bool   `json:"enabled"`
	CVEDatabase       string `json:"cve_database"`
	NVDAPIKey         string `json:"nvd_api_key"`
	UpdateInterval    int    `json:"update_interval"`
	SeverityThreshold string `json:"severity_threshold"`
}

// SystemMCPConfig contains MCP server settings.
type SystemMCPConfig struct {
	// Enabled enables the MCP server endpoint.
	// Env: SEED_MCP_ENABLED
	Enabled bool `json:"enabled"`

	// RequireAuth requires JWT authentication for MCP connections.
	// Env: SEED_MCP_REQUIRE_AUTH
	RequireAuth bool `json:"require_auth"`

	// RateLimitPerMinute limits requests per minute per client.
	// Env: SEED_MCP_RATE_LIMIT
	RateLimitPerMinute int `json:"rate_limit_per_minute"`

	// AllowedTools lists specific tools to expose via MCP.
	AllowedTools []string `json:"allowed_tools,omitempty"`
}

// DefaultSystemConfig returns the default system configuration.
func DefaultSystemConfig() *SystemConfig {
	return &SystemConfig{
		Version: ConfigVersion,
		Database: SystemDatabaseConfig{
			Path:           "data/seed.db",
			RetentionDays:  defaultDBRetentionDays,
			EnableWAL:      true,
			MaxConnections: defaultDBMaxConnections,
		},
		Server: SystemServerConfig{
			Port:  defaultHTTPSPort,
			HTTPS: true,
		},
		Logging: SystemLoggingConfig{
			Level:      "info",
			Format:     "text",
			MaxSize:    defaultLogMaxSizeMB,
			MaxBackups: defaultLogMaxBackups,
			MaxAge:     defaultLogMaxAgeDays,
		},
		Auth: SystemAuthConfig{
			DefaultUsername:     "admin",
			DefaultPasswordHash: "", // Must be set
			SessionTimeout:      time.Duration(defaultSessionTimeoutHours) * time.Hour,
		},
		Security: SystemSecurityConfig{
			RateLimitPerMinute: defaultRateLimitPerMinute,
		},
		MCP: SystemMCPConfig{
			RequireAuth:        true,
			RateLimitPerMinute: defaultRateLimitPerMinute,
		},
	}
}

// LoadSystemConfig loads the system configuration from a JSON file with env var overrides.
// The precedence is: defaults < config.json < environment variables.
func LoadSystemConfig(path string) (*SystemConfig, error) {
	cfg := DefaultSystemConfig()

	// Try to read config file
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read system config: %w", err)
		}
		// File doesn't exist, use defaults + env vars
		logging.GetLogger().Info("System config file not found, using defaults", "path", path)
	} else {
		if unmarshalErr := json.Unmarshal(data, cfg); unmarshalErr != nil {
			return nil, fmt.Errorf("parse system config JSON: %w", unmarshalErr)
		}
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	return cfg, nil
}

// applyEnvOverrides applies SEED_* environment variables to the config.
func applyEnvOverrides(cfg *SystemConfig) {
	applyDatabaseEnv(cfg)
	applyServerEnv(cfg)
	applyLoggingEnv(cfg)
	applyAuthEnv(cfg)
	applySecurityEnv(cfg)
	applyMCPEnv(cfg)
}

func applyDatabaseEnv(cfg *SystemConfig) {
	if v := os.Getenv("SEED_DB_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("SEED_DB_RETENTION_DAYS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Database.RetentionDays = i
		}
	}
	if v := os.Getenv("SEED_DB_ENABLE_WAL"); v != "" {
		cfg.Database.EnableWAL = parseBool(v)
	}
	if v := os.Getenv("SEED_DB_MAX_CONNECTIONS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Database.MaxConnections = i
		}
	}
}

func applyServerEnv(cfg *SystemConfig) {
	if v := os.Getenv("SEED_HTTP_PORT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = i
		}
	}
	if v := os.Getenv("SEED_HTTPS_ENABLED"); v != "" {
		cfg.Server.HTTPS = parseBool(v)
	}
	if v := os.Getenv("SEED_HTTP_REDIRECT_PORT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Server.HTTPRedirectPort = i
		}
	}
	if v := os.Getenv("SEED_TLS_CERT_FILE"); v != "" {
		cfg.Server.CertFile = v
	}
	if v := os.Getenv("SEED_TLS_KEY_FILE"); v != "" {
		cfg.Server.KeyFile = v
	}
	applyACMEEnv(cfg)
}

func applyACMEEnv(cfg *SystemConfig) {
	if v := os.Getenv("SEED_ACME_ENABLED"); v != "" {
		cfg.Server.ACME.Enabled = parseBool(v)
	}
	if v := os.Getenv("SEED_ACME_DOMAIN"); v != "" {
		cfg.Server.ACME.Domain = v
	}
	if v := os.Getenv("SEED_ACME_EMAIL"); v != "" {
		cfg.Server.ACME.Email = v
	}
	if v := os.Getenv("SEED_ACME_CACHE_DIR"); v != "" {
		cfg.Server.ACME.CacheDir = v
	}
	if v := os.Getenv("SEED_ACME_STAGING"); v != "" {
		cfg.Server.ACME.Staging = parseBool(v)
	}
}

func applyLoggingEnv(cfg *SystemConfig) {
	if v := os.Getenv("SEED_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("SEED_LOG_FORMAT"); v != "" {
		cfg.Logging.Format = v
	}
	if v := os.Getenv("SEED_LOG_ADD_SOURCE"); v != "" {
		cfg.Logging.AddSource = parseBool(v)
	}
	if v := os.Getenv("SEED_LOG_FILE"); v != "" {
		cfg.Logging.File = v
	}
	if v := os.Getenv("SEED_LOG_MAX_SIZE"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Logging.MaxSize = i
		}
	}
	if v := os.Getenv("SEED_LOG_MAX_BACKUPS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Logging.MaxBackups = i
		}
	}
	if v := os.Getenv("SEED_LOG_MAX_AGE"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Logging.MaxAge = i
		}
	}
	if v := os.Getenv("SEED_LOG_COMPRESS"); v != "" {
		cfg.Logging.Compress = parseBool(v)
	}
}

func applyAuthEnv(cfg *SystemConfig) {
	if v := os.Getenv("SEED_AUTH_DEFAULT_USERNAME"); v != "" {
		cfg.Auth.DefaultUsername = v
	}
	if v := os.Getenv("SEED_AUTH_DEFAULT_PASSWORD_HASH"); v != "" {
		cfg.Auth.DefaultPasswordHash = v
	}
	if v := os.Getenv("SEED_AUTH_SESSION_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Auth.SessionTimeout = d
		}
	}
	if v := os.Getenv("SEED_AUTH_JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
}

func applySecurityEnv(cfg *SystemConfig) {
	if v := os.Getenv("SEED_SECURITY_ALLOWED_ORIGINS"); v != "" {
		cfg.Security.AllowedOrigins = strings.Split(v, ",")
	}
	if v := os.Getenv("SEED_SECURITY_RATE_LIMIT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Security.RateLimitPerMinute = i
		}
	}
}

func applyMCPEnv(cfg *SystemConfig) {
	if v := os.Getenv("SEED_MCP_ENABLED"); v != "" {
		cfg.MCP.Enabled = parseBool(v)
	}
	if v := os.Getenv("SEED_MCP_REQUIRE_AUTH"); v != "" {
		cfg.MCP.RequireAuth = parseBool(v)
	}
	if v := os.Getenv("SEED_MCP_RATE_LIMIT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.MCP.RateLimitPerMinute = i
		}
	}
}

// parseBool parses common boolean string representations.
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

// Save writes the system configuration to a JSON file.
func (c *SystemConfig) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal system config: %w", err)
	}

	if writeErr := os.WriteFile(path, data, 0o600); writeErr != nil {
		return fmt.Errorf("write system config: %w", writeErr)
	}

	return nil
}

// Validate checks if the system configuration is valid.
func (c *SystemConfig) Validate() error {
	var errs []string
	errs = append(errs, c.validateDatabase()...)
	errs = append(errs, c.validateServer()...)
	errs = append(errs, c.validateLogging()...)
	errs = append(errs, c.validateAuth()...)

	if len(errs) > 0 {
		return fmt.Errorf("system config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

func (c *SystemConfig) validateDatabase() []string {
	var errs []string
	if c.Database.Path == "" {
		errs = append(errs, "database.path is required")
	}
	if c.Database.MaxConnections < 1 {
		errs = append(errs, "database.max_connections must be >= 1")
	}
	return errs
}

func (c *SystemConfig) validateServer() []string {
	var errs []string
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, "server.port must be 1-65535")
	}
	if c.Server.HTTPS && !c.Server.ACME.Enabled {
		if c.Server.CertFile == "" {
			errs = append(errs, "server.cert_file required when HTTPS enabled without ACME")
		}
		if c.Server.KeyFile == "" {
			errs = append(errs, "server.key_file required when HTTPS enabled without ACME")
		}
	}
	if c.Server.ACME.Enabled {
		if c.Server.ACME.Domain == "" {
			errs = append(errs, "server.acme.domain required when ACME enabled")
		}
		if c.Server.ACME.Email == "" {
			errs = append(errs, "server.acme.email required when ACME enabled")
		}
	}
	return errs
}

func (c *SystemConfig) validateLogging() []string {
	var errs []string
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[strings.ToLower(c.Logging.Level)] {
		errs = append(errs, "logging.level must be debug, info, warn, or error")
	}
	validFormats := map[string]bool{"text": true, "json": true}
	if !validFormats[strings.ToLower(c.Logging.Format)] {
		errs = append(errs, "logging.format must be text or json")
	}
	return errs
}

func (c *SystemConfig) validateAuth() []string {
	var errs []string
	if c.Auth.DefaultUsername == "" {
		errs = append(errs, "auth.default_username is required")
	}
	if c.Auth.SessionTimeout < time.Minute {
		errs = append(errs, "auth.session_timeout must be >= 1m")
	}
	return errs
}

// ToLegacyConfig converts SystemConfig to the legacy Config for backwards compatibility.
// This allows gradual migration while the codebase is updated.
func (c *SystemConfig) ToLegacyConfig() *Config {
	cfg := DefaultConfig()

	// Copy system config values to legacy config
	cfg.Version = c.Version
	cfg.Database.Path = c.Database.Path
	cfg.Database.RetentionDays = c.Database.RetentionDays
	cfg.Database.EnableWAL = c.Database.EnableWAL
	cfg.Database.MaxConnections = c.Database.MaxConnections

	cfg.Server.Port = c.Server.Port
	cfg.Server.HTTPS = c.Server.HTTPS
	cfg.Server.HTTPRedirectPort = c.Server.HTTPRedirectPort
	cfg.Server.CertFile = c.Server.CertFile
	cfg.Server.KeyFile = c.Server.KeyFile
	cfg.Server.ACME.Enabled = c.Server.ACME.Enabled
	cfg.Server.ACME.Domain = c.Server.ACME.Domain
	cfg.Server.ACME.Email = c.Server.ACME.Email
	cfg.Server.ACME.CacheDir = c.Server.ACME.CacheDir
	cfg.Server.ACME.Staging = c.Server.ACME.Staging

	cfg.Logging.Level = c.Logging.Level
	cfg.Logging.Format = c.Logging.Format
	cfg.Logging.AddSource = c.Logging.AddSource
	cfg.Logging.File = c.Logging.File
	cfg.Logging.MaxSize = c.Logging.MaxSize
	cfg.Logging.MaxBackups = c.Logging.MaxBackups
	cfg.Logging.MaxAge = c.Logging.MaxAge
	cfg.Logging.Compress = c.Logging.Compress

	cfg.Auth.DefaultUsername = c.Auth.DefaultUsername
	cfg.Auth.DefaultPasswordHash = c.Auth.DefaultPasswordHash
	cfg.Auth.SessionTimeout = c.Auth.SessionTimeout
	cfg.Auth.JWTSecret = c.Auth.JWTSecret

	// Copy SSO providers
	if len(c.Auth.SSO.Providers) > 0 {
		cfg.Auth.SSO.Providers = make([]SSOProviderConfig, len(c.Auth.SSO.Providers))
		for i, p := range c.Auth.SSO.Providers {
			cfg.Auth.SSO.Providers[i] = SSOProviderConfig(p)
		}
	}

	cfg.Security.AllowedOrigins = c.Security.AllowedOrigins
	cfg.Security.VulnerabilityScanning.Enabled = c.Security.VulnerabilityScanning.Enabled
	cfg.Security.VulnerabilityScanning.CVEDatabase = c.Security.VulnerabilityScanning.CVEDatabase
	cfg.Security.VulnerabilityScanning.NVDAPIKey = c.Security.VulnerabilityScanning.NVDAPIKey
	cfg.Security.VulnerabilityScanning.UpdateInterval = c.Security.VulnerabilityScanning.UpdateInterval
	cfg.Security.VulnerabilityScanning.SeverityThreshold = c.Security.VulnerabilityScanning.SeverityThreshold

	cfg.MCP.Enabled = c.MCP.Enabled
	cfg.MCP.RequireAuth = c.MCP.RequireAuth
	cfg.MCP.RateLimitPerMinute = c.MCP.RateLimitPerMinute
	cfg.MCP.AllowedTools = c.MCP.AllowedTools

	return cfg
}
