package discovery

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

// probeTLS probes a port for TLS certificate information.
func (f *Fingerprinter) probeTLS(ctx context.Context, ip string, port int) *TLSInfo {
	addr := fmt.Sprintf("%s:%d", ip, port)

	// Use context-aware dialing
	dialer := &net.Dialer{Timeout: f.timeout}
	rawConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil
	}
	// Wrap the raw connection with TLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // #nosec G402 -- We want to inspect any certificate
	}
	conn := tls.Client(rawConn, tlsConfig)
	if handshakeErr := conn.HandshakeContext(ctx); handshakeErr != nil {
		_ = rawConn.Close()
		return nil
	}
	defer func() { _ = conn.Close() }()

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil
	}

	cert := state.PeerCertificates[0]

	info := &TLSInfo{
		Port:              port,
		Version:           tlsVersionString(state.Version),
		CipherSuite:       tls.CipherSuiteName(state.CipherSuite),
		CommonName:        cert.Subject.CommonName,
		ValidFrom:         cert.NotBefore,
		ValidTo:           cert.NotAfter,
		DaysUntilExpiry:   int(time.Until(cert.NotAfter).Hours() / hoursPerDay),
		SelfSigned:        cert.Issuer.CommonName == cert.Subject.CommonName,
		SubjectAltNames:   cert.DNSNames,
		CertificateErrors: []string{},
	}

	// Build issuer string
	if len(cert.Issuer.Organization) > 0 {
		info.Issuer = cert.Issuer.Organization[0]
	} else {
		info.Issuer = cert.Issuer.CommonName
	}

	// Check for common certificate issues
	now := time.Now()
	if now.Before(cert.NotBefore) {
		info.CertificateErrors = append(info.CertificateErrors, "not yet valid")
	}
	if now.After(cert.NotAfter) {
		info.CertificateErrors = append(info.CertificateErrors, "expired")
	}
	if info.SelfSigned {
		info.CertificateErrors = append(info.CertificateErrors, "self-signed")
	}
	if info.DaysUntilExpiry > 0 && info.DaysUntilExpiry < 30 {
		info.CertificateErrors = append(info.CertificateErrors, "expiring soon")
	}

	return info
}

// tlsVersionString converts TLS version constant to string.
func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}
