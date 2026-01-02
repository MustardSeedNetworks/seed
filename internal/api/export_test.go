// Package api exports internal functions for testing.
package api

import (
	"context"
	"net/http"
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
