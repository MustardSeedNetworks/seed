package api

// The health-check boundaries in the licence catalogue (seed#2327, owner
// decision 2026-09-03). dns_monitoring and ssl_cert_monitoring are counts, not
// switches: the probe engine and its recurring checks are Free, and the paid
// tiers raise how many configured DNS servers are tested and how many HTTPS
// endpoints have their certificate evaluated. Neither has a Pro string; Pro is
// the tier with no limit.

import "github.com/MustardSeedNetworks/seed/internal/license"

// Configured DNS servers each tier tests, on top of the interface's own
// resolvers, which are always tested.
const (
	freeDNSServers    = 1
	starterDNSServers = 5
)

// HTTPS endpoints each tier gets a certificate summary for. Free covers the
// two HTTPS endpoints a new install ships with.
const (
	freeCertChecks    = 2
	starterCertChecks = 5
)

// tierLimit is the limit for the current licence: 0 (unlimited) at Pro, the
// paid limit with feature, else the Free limit.
func (s *Server) tierLimit(feature string, free, paid int) int {
	switch {
	case s.effectiveTier() >= license.TierPro:
		return 0
	case s.hasFeature(feature):
		return paid
	default:
		return free
	}
}

// dnsServerLimit is how many enabled configured DNS servers the licence
// tests; 0 is unlimited.
func (s *Server) dnsServerLimit() int {
	return s.tierLimit("dns_monitoring", freeDNSServers, starterDNSServers)
}

// certCheckLimit is how many HTTPS endpoints the licence evaluates the
// certificate of; 0 is unlimited.
func (s *Server) certCheckLimit() int {
	return s.tierLimit("ssl_cert_monitoring", freeCertChecks, starterCertChecks)
}

// certAllowance hands out certificate summaries in result order until the
// limit is spent. A limit of 0 never runs out.
type certAllowance struct {
	limit, used int
}

func (a *certAllowance) take() bool {
	if a.limit > 0 && a.used == a.limit {
		return false
	}
	a.used++
	return true
}
