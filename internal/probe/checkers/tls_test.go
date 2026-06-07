package checkers_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
)

// fakeTLSDialer returns canned ConnState / errors for tests.
type fakeTLSDialer struct {
	state checkers.TLSConnState
	err   error

	gotAddr string
	gotSNI  string
}

func (f *fakeTLSDialer) Dial(_ context.Context, addr, sni string) (checkers.TLSConnState, error) {
	f.gotAddr = addr
	f.gotSNI = sni
	if f.err != nil {
		return checkers.TLSConnState{}, f.err
	}
	return f.state, nil
}

// makeCert generates a self-signed cert with the given NotAfter and
// common name. Used to populate fake handshake state.
func makeCert(t *testing.T, commonName string, notAfter time.Time) *x509.Certificate {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		Issuer:       pkix.Name{Organization: []string{"Test CA"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}
	return cert
}

func TestTLSChecker_Kind(t *testing.T) {
	t.Parallel()
	c := checkers.NewTLSChecker()
	if c.Kind() != "tls" {
		t.Errorf("Kind() = %q, want %q", c.Kind(), "tls")
	}
}

func TestTLSChecker_Run_SuccessAndCertMetadata(t *testing.T) {
	t.Parallel()
	notAfter := time.Now().Add(45 * 24 * time.Hour)
	cert := makeCert(t, "example.com", notAfter)

	fake := &fakeTLSDialer{
		state: checkers.TLSConnState{
			PeerCertificates: []*x509.Certificate{cert},
			Version:          tls.VersionTLS13,
		},
	}
	c := checkers.NewTLSChecker().WithTLSDialer(fake)

	p := probe.Probe{
		ID:       "p-1",
		ClientID: "default",
		Kind:     "tls",
		Target:   "example.com",
	}
	r := c.Run(context.Background(), p)

	if !r.Success {
		t.Fatalf("Result.Success = false, want true; error=%q", r.Error)
	}
	if fake.gotAddr != "example.com:443" {
		t.Errorf("dialer addr = %q, want example.com:443", fake.gotAddr)
	}
	if fake.gotSNI != "example.com" {
		t.Errorf("dialer SNI = %q, want example.com", fake.gotSNI)
	}

	var info checkers.TLSCertInfo
	if err := json.Unmarshal(r.Metadata, &info); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if info.Subject != "example.com" {
		t.Errorf("Subject = %q, want example.com", info.Subject)
	}
	// Self-signed cert: issuer mirrors subject.
	if info.Issuer != "example.com" {
		t.Errorf("Issuer = %q, want example.com (self-signed)", info.Issuer)
	}
	if info.TLSVersion != "TLS 1.3" {
		t.Errorf("TLSVersion = %q, want TLS 1.3", info.TLSVersion)
	}
	if info.SHA256Fingerprint == "" {
		t.Error("SHA256Fingerprint should be set")
	}
	// DaysRemaining ~ 44-45 depending on test execution time.
	if info.DaysRemaining < 40 || info.DaysRemaining > 46 {
		t.Errorf("DaysRemaining = %d, want ~45", info.DaysRemaining)
	}
}

func TestTLSChecker_Run_PortFromTarget(t *testing.T) {
	t.Parallel()
	cert := makeCert(t, "example.com", time.Now().Add(30*24*time.Hour))
	fake := &fakeTLSDialer{
		state: checkers.TLSConnState{
			PeerCertificates: []*x509.Certificate{cert},
			Version:          tls.VersionTLS12,
		},
	}
	c := checkers.NewTLSChecker().WithTLSDialer(fake)

	p := probe.Probe{Kind: "tls", Target: "mqtt.example.com:8883"}
	_ = c.Run(context.Background(), p)

	if fake.gotAddr != "mqtt.example.com:8883" {
		t.Errorf("dialer addr = %q, want mqtt.example.com:8883", fake.gotAddr)
	}
	if fake.gotSNI != "mqtt.example.com" {
		t.Errorf("dialer SNI = %q, want mqtt.example.com", fake.gotSNI)
	}
}

func TestTLSChecker_Run_PortFromParams(t *testing.T) {
	t.Parallel()
	cert := makeCert(t, "imap.example.com", time.Now().Add(30*24*time.Hour))
	fake := &fakeTLSDialer{
		state: checkers.TLSConnState{
			PeerCertificates: []*x509.Certificate{cert},
			Version:          tls.VersionTLS13,
		},
	}
	c := checkers.NewTLSChecker().WithTLSDialer(fake)

	p := probe.Probe{
		Kind:   "tls",
		Target: "imap.example.com",
		Params: json.RawMessage(`{"port":"993"}`),
	}
	_ = c.Run(context.Background(), p)

	if fake.gotAddr != "imap.example.com:993" {
		t.Errorf("dialer addr = %q, want imap.example.com:993", fake.gotAddr)
	}
}

func TestTLSChecker_Run_SNIOverride(t *testing.T) {
	t.Parallel()
	cert := makeCert(t, "example.com", time.Now().Add(30*24*time.Hour))
	fake := &fakeTLSDialer{
		state: checkers.TLSConnState{
			PeerCertificates: []*x509.Certificate{cert},
			Version:          tls.VersionTLS13,
		},
	}
	c := checkers.NewTLSChecker().WithTLSDialer(fake)

	p := probe.Probe{
		Kind:   "tls",
		Target: "10.0.0.5",
		Params: json.RawMessage(`{"sni":"prod.example.com"}`),
	}
	_ = c.Run(context.Background(), p)

	if fake.gotSNI != "prod.example.com" {
		t.Errorf("dialer SNI = %q, want prod.example.com", fake.gotSNI)
	}
	if fake.gotAddr != "10.0.0.5:443" {
		t.Errorf("dialer addr = %q, want 10.0.0.5:443", fake.gotAddr)
	}
}

func TestTLSChecker_Run_HandshakeError(t *testing.T) {
	t.Parallel()
	fake := &fakeTLSDialer{err: errors.New("connection refused")}
	c := checkers.NewTLSChecker().WithTLSDialer(fake)

	p := probe.Probe{Kind: "tls", Target: "down.example.com"}
	r := c.Run(context.Background(), p)

	if r.Success {
		t.Error("Result.Success = true, want false")
	}
	if r.Error != "connection refused" {
		t.Errorf("Result.Error = %q, want %q", r.Error, "connection refused")
	}
}

func TestTLSChecker_Run_NoPeerCertificates(t *testing.T) {
	t.Parallel()
	fake := &fakeTLSDialer{
		state: checkers.TLSConnState{
			PeerCertificates: nil,
			Version:          tls.VersionTLS13,
		},
	}
	c := checkers.NewTLSChecker().WithTLSDialer(fake)

	p := probe.Probe{Kind: "tls", Target: "example.com"}
	r := c.Run(context.Background(), p)

	if r.Success {
		t.Error("Result.Success = true, want false when no peer certs")
	}
}

func TestTLSChecker_Run_ExpiredCertStillSucceeds(t *testing.T) {
	t.Parallel()
	// An expired cert is still "the cert" — we report Success=true
	// because the handshake completed; days_remaining lets the
	// alerts pipeline decide what to do.
	cert := makeCert(t, "expired.example.com", time.Now().Add(-time.Hour))
	fake := &fakeTLSDialer{
		state: checkers.TLSConnState{
			PeerCertificates: []*x509.Certificate{cert},
			Version:          tls.VersionTLS13,
		},
	}
	c := checkers.NewTLSChecker().WithTLSDialer(fake)

	p := probe.Probe{Kind: "tls", Target: "expired.example.com"}
	r := c.Run(context.Background(), p)

	if !r.Success {
		t.Errorf("Result.Success = false, want true (handshake succeeded)")
	}

	var info checkers.TLSCertInfo
	if err := json.Unmarshal(r.Metadata, &info); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	// Integer division of negative hours/24 truncates toward zero,
	// so a cert that expired 1 hour ago reports days_remaining=0
	// (not -1). Either is "expired" semantically; the alerts
	// pipeline reads NotAfter for precision.
	if info.DaysRemaining > 0 {
		t.Errorf("DaysRemaining = %d, want <= 0 for expired cert", info.DaysRemaining)
	}
	// NotAfter must round-trip to a value in the past.
	notAfter, parseErr := time.Parse(time.RFC3339, info.NotAfter)
	if parseErr != nil {
		t.Fatalf("parse NotAfter %q: %v", info.NotAfter, parseErr)
	}
	if !notAfter.Before(time.Now()) {
		t.Errorf("NotAfter %v should be in the past", notAfter)
	}
}
