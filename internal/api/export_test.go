// Package api exports internal functions for testing.
package api

import (
	"context"
	"net/http"

	"github.com/krisarmstrong/seed/internal/config"
)

// SplitCIDR exports splitCIDR for testing.
var SplitCIDR = splitCIDR

// IsIPv4Address exports isIPv4Address for testing.
var IsIPv4Address = isIPv4Address

// ParsePrefix exports parsePrefix for testing.
var ParsePrefix = parsePrefix

// IsLinkLocal exports isLinkLocal for testing.
var IsLinkLocal = isLinkLocal

// IsUniqueLocal exports isUniqueLocal for testing.
var IsUniqueLocal = isUniqueLocal

// IPAddrInfo represents parsed IP address information for testing.
type IPAddrInfo struct {
	IsIPv4  bool
	Address string
	Subnet  string
	Prefix  int
	Scope   string
	Source  string
}

// ParseIPAddress exports parseIPAddress for testing.
var ParseIPAddress = func(addr string) IPAddrInfo {
	result := parseIPAddress(addr)
	return IPAddrInfo{
		IsIPv4:  result.isIPv4,
		Address: result.address,
		Subnet:  result.subnet,
		Prefix:  result.prefix,
		Scope:   result.scope,
		Source:  result.source,
	}
}

// GetTestStatus exports getTestStatus for testing.
var GetTestStatus = getTestStatus

// GetTLSVersionString exports getTLSVersionString for testing.
var GetTLSVersionString = getTLSVersionString

// RunTCPTest exports runTCPTest for testing.
var RunTCPTest = func(ctx context.Context, host string, port int) (float64, error) {
	return runTCPTest(ctx, host, port)
}

// RunExtendedPing exports runExtendedPing for testing.
var RunExtendedPing = runExtendedPing

// GetInterfaceFromRequest exports getInterfaceFromRequest for testing.
func (s *Server) GetInterfaceFromRequest(r *http.Request) string {
	return s.getInterfaceFromRequest(r)
}

// NormalizeHTTPURL exports normalizeHTTPURL for testing.
var NormalizeHTTPURL = normalizeHTTPURL

// ValidateIperfClientRequest exports validateIperfClientRequest for testing.
var ValidateIperfClientRequest = validateIperfClientRequest

// ParseLogQueryParams exports parseLogQueryParams for testing.
var ParseLogQueryParams = parseLogQueryParams

// MatchesLogFilters exports matchesLogFilters for testing.
var MatchesLogFilters = matchesLogFilters

// PaginateLogs exports paginateLogs for testing.
var PaginateLogs = paginateLogs

// ParseCSV exports parseCSV for testing.
var ParseCSV = parseCSV

// ContainsIgnoreCase exports containsIgnoreCase for testing.
var ContainsIgnoreCase = containsIgnoreCase

// IsValidIPOctet exports isValidIPOctet for testing.
var IsValidIPOctet = isValidIPOctet

// LogQueryParams is a type alias for logQueryParams for testing.
type LogQueryParams = logQueryParams

// BodyLimitMiddleware exports bodyLimitMiddleware for testing.
var BodyLimitMiddleware = bodyLimitMiddleware

// HandleStatus exports handleStatus for testing.
func (s *Server) HandleStatus(w http.ResponseWriter, r *http.Request) {
	s.handleStatus(w, r)
}

// HandleExport exports handleExport for testing.
func (s *Server) HandleExport(w http.ResponseWriter, r *http.Request) {
	s.handleExport(w, r)
}

// Limit returns the rate limiter's limit for testing.
func (rl *RateLimiter) Limit() int {
	return rl.limit
}

// SetConfig sets the server config for testing.
func (s *Server) SetConfig(cfg *config.Config) {
	s.config = cfg
}

// SetConfigPath sets the server config path for testing.
func (s *Server) SetConfigPath(path string) {
	s.configPath = path
}

// GetConfig returns the server config for testing.
func (s *Server) GetConfig() *config.Config {
	return s.config
}

// HandleConfigVersion exports handleConfigVersion for testing.
func (s *Server) HandleConfigVersion(w http.ResponseWriter, r *http.Request) {
	s.handleConfigVersion(w, r)
}

// HandleConfigBackups exports handleConfigBackups for testing.
func (s *Server) HandleConfigBackups(w http.ResponseWriter, r *http.Request) {
	s.handleConfigBackups(w, r)
}

// HandleConfigBackupCreate exports handleConfigBackupCreate for testing.
func (s *Server) HandleConfigBackupCreate(w http.ResponseWriter, r *http.Request) {
	s.handleConfigBackupCreate(w, r)
}

// HandleConfigRestore exports handleConfigRestore for testing.
func (s *Server) HandleConfigRestore(w http.ResponseWriter, r *http.Request) {
	s.handleConfigRestore(w, r)
}

// HandleConfigBackupDelete exports handleConfigBackupDelete for testing.
func (s *Server) HandleConfigBackupDelete(w http.ResponseWriter, r *http.Request) {
	s.handleConfigBackupDelete(w, r)
}
