package api

// health_checks_tls.go contains TLS certificate-expiry checks attached to
// health-check HTTP endpoint tests.

import (
	"context"
	"crypto/tls"
	"net"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// CertInfo holds certificate expiry information.
type CertInfo struct {
	DaysLeft   int
	Status     string // success, warning, error
	ExpiryDate string
	CommonName string
	TLSVersion string // TLS 1.0, TLS 1.1, TLS 1.2, TLS 1.3
	Issuer     string // Certificate issuer (for context)
}

// evaluateCertExpiry checks certificate expiry and updates test result.
func (s *Server) evaluateCertExpiry(
	result *CustomTestResult,
	url string,
	threshold config.CertExpiryThreshold,
) {
	certInfo := checkCertExpiry(url, threshold.Warning, threshold.Critical)
	result.CertDaysLeft = certInfo.DaysLeft
	result.CertStatus = certInfo.Status
	result.CertExpiry = certInfo.ExpiryDate
	result.CertCommonName = certInfo.CommonName
	result.TLSVersion = certInfo.TLSVersion
	result.CertIssuer = certInfo.Issuer

	if certInfo.Status == statusError && result.TestStatus != statusError {
		result.TestStatus = statusError
	} else if certInfo.Status == statusWarning && result.TestStatus == statusSuccess {
		result.TestStatus = statusWarning
	}
}

// checkCertExpiry checks the TLS certificate expiry for a URL.
func checkCertExpiry(url string, warningDays, criticalDays int) CertInfo {
	info := CertInfo{Status: statusSuccess}

	// Extract host from URL
	host := strings.TrimPrefix(url, "https://")
	host = strings.TrimPrefix(host, "http://")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, ":"); idx == -1 {
		host += ":443"
	}

	// Connect with TLS using context-aware dialing
	ctx, cancel := context.WithTimeout(context.Background(), certCheckTimeoutSec*time.Second)
	defer cancel()

	dialer := &net.Dialer{Timeout: certCheckTimeoutSec * time.Second}
	rawConn, err := dialer.DialContext(ctx, protoTCP, host)
	if err != nil {
		info.Status = statusError
		return info
	}

	// #nosec G402 - certificate verification intentionally skipped to inspect expiry
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	} // We want to check expiry even for self-signed
	conn := tls.Client(rawConn, tlsConfig)
	if hsErr := conn.HandshakeContext(ctx); hsErr != nil {
		_ = rawConn.Close()
		info.Status = statusError
		return info
	}
	defer func() { _ = conn.Close() }()

	// Get connection state for TLS info
	connState := conn.ConnectionState()

	// Get TLS version
	info.TLSVersion = getTLSVersionString(connState.Version)

	// Get certificate chain
	certs := connState.PeerCertificates
	if len(certs) == 0 {
		info.Status = statusError
		return info
	}

	// Check the leaf certificate
	cert := certs[0]
	info.CommonName = cert.Subject.CommonName
	info.ExpiryDate = cert.NotAfter.Format("2006-01-02")

	// Get issuer (org or CN)
	if len(cert.Issuer.Organization) > 0 {
		info.Issuer = cert.Issuer.Organization[0]
	} else if cert.Issuer.CommonName != "" {
		info.Issuer = cert.Issuer.CommonName
	}

	// Calculate days until expiry
	daysLeft := int(time.Until(cert.NotAfter).Hours() / hoursPerDay)
	info.DaysLeft = daysLeft

	// Determine status
	switch {
	case daysLeft <= 0:
		info.Status = statusError // Expired
	case daysLeft <= criticalDays:
		info.Status = statusError // Critical
	case daysLeft <= warningDays:
		info.Status = statusWarning // Warning
	default:
		info.Status = statusSuccess // OK
	}

	return info
}

// getTLSVersionString converts TLS version to human-readable string.
func getTLSVersionString(tlsVersion uint16) string {
	switch tlsVersion {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return "Unknown"
	}
}
