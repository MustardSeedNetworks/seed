// Package api exports internal functions for testing.
package api

import (
	"context"
	"net/http"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/logging"
)

// ExportSplitCIDR exposes splitCIDR for testing.
func ExportSplitCIDR(addr string) [2]string {
	return splitCIDR(addr)
}

// ExportIsIPv4Address exposes isIPv4Address for testing.
func ExportIsIPv4Address(addr string) bool {
	return isIPv4Address(addr)
}

// ExportParsePrefix exposes parsePrefix for testing.
func ExportParsePrefix(s string) int {
	return parsePrefix(s)
}

// ExportIsLinkLocal exposes isLinkLocal for testing.
func ExportIsLinkLocal(addr string) bool {
	return isLinkLocal(addr)
}

// ExportIsUniqueLocal exposes isUniqueLocal for testing.
func ExportIsUniqueLocal(addr string) bool {
	return isUniqueLocal(addr)
}

// IPAddrInfo represents parsed IP address information for testing.
type IPAddrInfo struct {
	IsIPv4  bool
	Address string
	Subnet  string
	Prefix  int
	Scope   string
	Source  string
}

// ExportParseIPAddress exposes parseIPAddress for testing.
func ExportParseIPAddress(addr string) IPAddrInfo {
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

// ExportGetTestStatus exposes getTestStatus for testing.
func ExportGetTestStatus(latencyMs float64, warningMs, criticalMs int64) string {
	return getTestStatus(latencyMs, warningMs, criticalMs)
}

// ExportGetTLSVersionString exposes getTLSVersionString for testing.
func ExportGetTLSVersionString(tlsVersion uint16) string {
	return getTLSVersionString(tlsVersion)
}

// ExportRunTCPTest exposes runTCPTest for testing.
func ExportRunTCPTest(ctx context.Context, host string, port int) (float64, error) {
	return runTCPTest(ctx, host, port)
}

// ExportRunExtendedPing exposes runExtendedPing for testing.
func ExportRunExtendedPing(host string, count int) (*PingStats, error) {
	return runExtendedPing(host, count)
}

// GetInterfaceFromRequest exports getInterfaceFromRequest for testing.
func (s *Server) GetInterfaceFromRequest(r *http.Request) string {
	return s.getInterfaceFromRequest(r)
}

// ExportNormalizeHTTPURL exposes normalizeHTTPURL for testing.
func ExportNormalizeHTTPURL(rawURL string) (string, bool) {
	return normalizeHTTPURL(rawURL)
}

// ExportValidateIperfClientRequest exposes validateIperfClientRequest for testing.
func ExportValidateIperfClientRequest(req *IperfClientRequest) error {
	return validateIperfClientRequest(req)
}

// ExportParseLogQueryParams exposes parseLogQueryParams for testing.
func ExportParseLogQueryParams(r *http.Request) LogQueryParams {
	return parseLogQueryParams(r)
}

// ExportMatchesLogFilters exposes matchesLogFilters for testing.
func ExportMatchesLogFilters(log *logging.LogEntry, params *LogQueryParams) bool {
	return matchesLogFilters(log, params)
}

// ExportPaginateLogs exposes paginateLogs for testing.
func ExportPaginateLogs(logs []*logging.LogEntry, offset, limit int) []*logging.LogEntry {
	return paginateLogs(logs, offset, limit)
}

// ExportParseCSV exposes parseCSV for testing.
func ExportParseCSV(s string) []string {
	return parseCSV(s)
}

// ExportContainsIgnoreCase exposes containsIgnoreCase for testing.
func ExportContainsIgnoreCase(slice []string, target string) bool {
	return containsIgnoreCase(slice, target)
}

// ExportIsValidIPOctet exposes isValidIPOctet for testing.
func ExportIsValidIPOctet(s string) bool {
	return isValidIPOctet(s)
}

// LogQueryParams is a type alias for logQueryParams for testing.
type LogQueryParams = logQueryParams

// ExportBodyLimitMiddleware exposes bodyLimitMiddleware for testing.
func ExportBodyLimitMiddleware(next http.Handler) http.Handler {
	return bodyLimitMiddleware(next)
}

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

// HandleRefreshToken exports handleRefreshToken for testing.
func (s *Server) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	s.handleRefreshToken(w, r)
}

// HandleLogout exports handleLogout for testing.
func (s *Server) HandleLogout(w http.ResponseWriter, r *http.Request) {
	s.handleLogout(w, r)
}

// HandleSetupComplete exports handleSetupComplete for testing.
func (s *Server) HandleSetupComplete(w http.ResponseWriter, r *http.Request) {
	s.handleSetupComplete(w, r)
}

// HandleSetupStatus exports handleSetupStatus for testing.
func (s *Server) HandleSetupStatus(w http.ResponseWriter, r *http.Request) {
	s.handleSetupStatus(w, r)
}

// AuthManager returns the server's auth manager for testing.
func (s *Server) AuthManager() *auth.Manager {
	return s.authManager
}

// SetupTokenManager returns the server's setup token manager for testing.
func (s *Server) SetupTokenManager() *SetupTokenManager {
	return s.setupTokenManager
}

// Mux returns the server's HTTP mux for testing.
func (s *Server) Mux() *http.ServeMux {
	return s.mux
}
