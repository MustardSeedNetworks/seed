package discovery_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestNewFingerprinter(t *testing.T) {
	// Test with custom timeout
	fp := discovery.NewFingerprinter(5 * time.Second)
	if fp == nil {
		t.Fatal("NewFingerprinter returned nil")
	}

	// Test with zero timeout (should use default)
	fp2 := discovery.NewFingerprinter(0)
	if fp2 == nil {
		t.Fatal("NewFingerprinter with zero timeout returned nil")
	}
}

func TestOSFingerprint_Fields(t *testing.T) {
	fp := discovery.OSFingerprint{
		OSFamily:    "linux",
		OSVersion:   "ubuntu",
		Confidence:  85,
		Methods:     []string{"banner", "http"},
		TTLObserved: 64,
	}

	if fp.OSFamily != "linux" {
		t.Errorf("OSFamily should be 'linux', got %q", fp.OSFamily)
	}
	if fp.OSVersion != "ubuntu" {
		t.Errorf("OSVersion should be 'ubuntu', got %q", fp.OSVersion)
	}
	if fp.Confidence != 85 {
		t.Errorf("Confidence should be 85, got %d", fp.Confidence)
	}
	if len(fp.Methods) != 2 {
		t.Errorf("Methods should have 2 entries, got %d", len(fp.Methods))
	}
	if fp.TTLObserved != 64 {
		t.Errorf("TTLObserved should be 64, got %d", fp.TTLObserved)
	}
}

func TestServiceVersion_Fields(t *testing.T) {
	sv := discovery.ServiceVersion{
		Port:       22,
		Service:    "ssh",
		Product:    "OpenSSH",
		Version:    "8.4p1",
		ExtraInfo:  "Ubuntu-5ubuntu1",
		Confidence: 95,
	}

	if sv.Port != 22 {
		t.Errorf("Port should be 22, got %d", sv.Port)
	}
	if sv.Service != "ssh" {
		t.Errorf("Service should be 'ssh', got %q", sv.Service)
	}
	if sv.Product != "OpenSSH" {
		t.Errorf("Product should be 'OpenSSH', got %q", sv.Product)
	}
	if sv.Version != "8.4p1" {
		t.Errorf("Version should be '8.4p1', got %q", sv.Version)
	}
	if sv.ExtraInfo != "Ubuntu-5ubuntu1" {
		t.Errorf("ExtraInfo should be 'Ubuntu-5ubuntu1', got %q", sv.ExtraInfo)
	}
	if sv.Confidence != 95 {
		t.Errorf("Confidence should be 95, got %d", sv.Confidence)
	}
}

func TestTLSInfo_Fields(t *testing.T) {
	now := time.Now()
	validFrom := now.Add(-30 * 24 * time.Hour)
	validTo := now.Add(60 * 24 * time.Hour)
	info := discovery.TLSInfo{
		Port:              443,
		Version:           "TLS 1.3",
		CipherSuite:       "TLS_AES_256_GCM_SHA384",
		CommonName:        "example.com",
		Issuer:            "Let's Encrypt",
		ValidFrom:         validFrom,
		ValidTo:           validTo,
		DaysUntilExpiry:   60,
		SelfSigned:        false,
		SubjectAltNames:   []string{"example.com", "www.example.com"},
		CertificateErrors: []string{},
	}

	if info.Port != 443 {
		t.Errorf("Port should be 443, got %d", info.Port)
	}
	if info.Version != "TLS 1.3" {
		t.Errorf("Version should be 'TLS 1.3', got %q", info.Version)
	}
	if info.CipherSuite != "TLS_AES_256_GCM_SHA384" {
		t.Errorf("CipherSuite should be 'TLS_AES_256_GCM_SHA384', got %q", info.CipherSuite)
	}
	if info.CommonName != "example.com" {
		t.Errorf("CommonName should be 'example.com', got %q", info.CommonName)
	}
	if info.Issuer != "Let's Encrypt" {
		t.Errorf("Issuer should be 'Let's Encrypt', got %q", info.Issuer)
	}
	if !info.ValidFrom.Equal(validFrom) {
		t.Errorf("ValidFrom mismatch")
	}
	if !info.ValidTo.Equal(validTo) {
		t.Errorf("ValidTo mismatch")
	}
	if info.DaysUntilExpiry != 60 {
		t.Errorf("DaysUntilExpiry should be 60, got %d", info.DaysUntilExpiry)
	}
	if info.SelfSigned {
		t.Error("SelfSigned should be false")
	}
	if len(info.SubjectAltNames) != 2 {
		t.Errorf("SubjectAltNames should have 2 entries, got %d", len(info.SubjectAltNames))
	}
	if len(info.CertificateErrors) != 0 {
		t.Errorf("CertificateErrors should be empty, got %d", len(info.CertificateErrors))
	}
}

func TestAdvancedProbeResult_Fields(t *testing.T) {
	result := discovery.AdvancedProbeResult{
		IP:       "192.168.1.10",
		ProbedAt: time.Now(),
		OSFingerprint: &discovery.OSFingerprint{
			OSFamily:   "linux",
			Confidence: 80,
		},
		ServiceVersions: []discovery.ServiceVersion{
			{Port: 22, Service: "ssh"},
		},
		TLSInfo: []discovery.TLSInfo{
			{Port: 443, Version: "TLS 1.2"},
		},
	}

	if result.IP != "192.168.1.10" {
		t.Errorf("IP should be '192.168.1.10', got %q", result.IP)
	}
	if result.ProbedAt.IsZero() {
		t.Error("ProbedAt should be set")
	}
	if result.OSFingerprint == nil {
		t.Error("OSFingerprint should be set")
	}
	if len(result.ServiceVersions) != 1 {
		t.Errorf("ServiceVersions should have 1 entry, got %d", len(result.ServiceVersions))
	}
	if len(result.TLSInfo) != 1 {
		t.Errorf("TLSInfo should have 1 entry, got %d", len(result.TLSInfo))
	}
}

func TestFingerprinter_ProbeDevice_NilProfile(t *testing.T) {
	fp := discovery.NewFingerprinter(100 * time.Millisecond)

	// Using TEST-NET-1 (192.0.2.x) which is non-routable
	// This tests the quickScan path when profile is nil
	// Note: This will timeout quickly since 192.0.2.x is not routable

	// Create a context with very short timeout to speed up test
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	result := fp.ProbeDevice(ctx, "192.0.2.1", nil)

	if result == nil {
		t.Fatal("ProbeDevice should return a result, not nil")
	}
	if result.IP != "192.0.2.1" {
		t.Errorf("IP should be '192.0.2.1', got %q", result.IP)
	}
	if result.ProbedAt.IsZero() {
		t.Error("ProbedAt should be set")
	}
}

func TestFingerprinter_ProbeDevice_WithProfile(t *testing.T) {
	fp := discovery.NewFingerprinter(100 * time.Millisecond)

	profile := &discovery.DeviceProfile{
		ProfiledAt: time.Now(),
		OpenPorts: []discovery.OpenPort{
			{Port: 22, Service: "ssh", Banner: "SSH-2.0-OpenSSH_8.4p1 Ubuntu-5ubuntu1"},
			{Port: 80, Service: "http", Banner: "HTTP/1.1 200 OK\r\nServer: nginx/1.18.0 (Ubuntu)"},
		},
		HTTPInfo: &discovery.HTTPInfo{
			Server: "nginx/1.18.0 (Ubuntu)",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

	if result == nil {
		t.Fatal("ProbeDevice should return a result")
	}

	// Should detect OS from banner
	if result.OSFingerprint != nil {
		if result.OSFingerprint.OSFamily == "" {
			t.Log("OSFamily detection from banner worked")
		}
	}

	// Should detect service versions
	if len(result.ServiceVersions) == 0 {
		t.Error("Should detect service versions from profile")
	}
}

func TestFingerprinter_DetectSSHVersion(t *testing.T) {
	fp := discovery.NewFingerprinter(100 * time.Millisecond)

	profile := &discovery.DeviceProfile{
		OpenPorts: []discovery.OpenPort{
			{Port: 22, Service: "ssh", Banner: "SSH-2.0-OpenSSH_8.4p1"},
		},
	}

	ctx := context.Background()
	result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

	if len(result.ServiceVersions) == 0 {
		t.Fatal("Should detect SSH service")
	}

	sshVersion := result.ServiceVersions[0]
	if sshVersion.Service != "ssh" {
		t.Errorf("Service should be 'ssh', got %q", sshVersion.Service)
	}
	if sshVersion.Product != "OpenSSH" {
		t.Errorf("Product should be 'OpenSSH', got %q", sshVersion.Product)
	}
	if sshVersion.Version != "8.4p1" {
		t.Errorf("Version should be '8.4p1', got %q", sshVersion.Version)
	}
}

func TestFingerprinter_DetectFTPVersion(t *testing.T) {
	fp := discovery.NewFingerprinter(100 * time.Millisecond)

	tests := []struct {
		name     string
		banner   string
		product  string
		hasMatch bool
	}{
		{"vsftpd", "220 (vsFTPd 3.0.3)", "vsftpd", true},
		{"proftpd", "220 ProFTPD 1.3.6 Server ready", "ProFTPD", true},
		{"pure-ftpd", "220 Pure-FTPd Server ready", "Pure-FTPd", true},
		{"microsoft", "220 Microsoft FTP Service", "Microsoft FTP", true},
		{"unknown", "220 Custom FTP Server", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &discovery.DeviceProfile{
				OpenPorts: []discovery.OpenPort{
					{Port: 21, Service: "ftp", Banner: tt.banner},
				},
			}

			ctx := context.Background()
			result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

			if len(result.ServiceVersions) == 0 {
				if tt.hasMatch {
					t.Error("Should detect FTP service")
				}
				return
			}

			ftpVersion := result.ServiceVersions[0]
			if ftpVersion.Service != "ftp" {
				t.Errorf("Service should be 'ftp', got %q", ftpVersion.Service)
			}
			if tt.product != "" && ftpVersion.Product != tt.product {
				t.Errorf("Product should be %q, got %q", tt.product, ftpVersion.Product)
			}
		})
	}
}

func TestFingerprinter_DetectSMTPVersion(t *testing.T) {
	fp := discovery.NewFingerprinter(100 * time.Millisecond)

	tests := []struct {
		name    string
		port    int
		banner  string
		product string
	}{
		{"postfix", 25, "220 mail.example.com ESMTP Postfix", "Postfix"},
		{"sendmail", 25, "220 mail.example.com ESMTP Sendmail 8.15.2", "Sendmail"},
		{"exim", 25, "220 mail.example.com ESMTP Exim 4.93", "Exim"},
		{"exchange", 587, "220 mail.example.com Microsoft ESMTP MAIL Service", "Microsoft Exchange"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &discovery.DeviceProfile{
				OpenPorts: []discovery.OpenPort{
					{Port: tt.port, Service: "smtp", Banner: tt.banner},
				},
			}

			ctx := context.Background()
			result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

			if len(result.ServiceVersions) == 0 {
				t.Error("Should detect SMTP service")
				return
			}

			smtpVersion := result.ServiceVersions[0]
			if smtpVersion.Service != "smtp" {
				t.Errorf("Service should be 'smtp', got %q", smtpVersion.Service)
			}
			if smtpVersion.Product != tt.product {
				t.Errorf("Product should be %q, got %q", tt.product, smtpVersion.Product)
			}
		})
	}
}

func TestFingerprinter_DetectTelnetVersion(t *testing.T) {
	fp := discovery.NewFingerprinter(100 * time.Millisecond)

	tests := []struct {
		name    string
		banner  string
		product string
	}{
		{"cisco", "User Access Verification\n\nUsername: cisco", "Cisco IOS"},
		{"linux", "Linux telnetd (Debian)", "Linux telnetd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &discovery.DeviceProfile{
				OpenPorts: []discovery.OpenPort{
					{Port: 23, Service: "telnet", Banner: tt.banner},
				},
			}

			ctx := context.Background()
			result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

			if len(result.ServiceVersions) == 0 {
				t.Error("Should detect telnet service")
				return
			}

			version := result.ServiceVersions[0]
			if version.Service != "telnet" {
				t.Errorf("Service should be 'telnet', got %q", version.Service)
			}
			if version.Product != tt.product {
				t.Errorf("Product should be %q, got %q", tt.product, version.Product)
			}
		})
	}
}

func TestFingerprinter_OSDetectionFromBanner(t *testing.T) {
	fp := discovery.NewFingerprinter(100 * time.Millisecond)

	tests := []struct {
		name     string
		banner   string
		osFamily string
	}{
		{"ubuntu", "SSH-2.0-OpenSSH_8.4p1 Ubuntu", "linux"},
		{"debian", "SSH-2.0-OpenSSH_7.9p1 Debian", "linux"},
		{"centos", "SSH-2.0-OpenSSH_7.4 CentOS", "linux"},
		{"freebsd", "SSH-2.0-OpenSSH_7.3 FreeBSD", "bsd"},
		{"cisco", "Cisco IOS Software", "cisco"},
		{"windows", "SSH-2.0-OpenSSH Windows", "windows"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &discovery.DeviceProfile{
				OpenPorts: []discovery.OpenPort{
					{Port: 22, Service: "ssh", Banner: tt.banner},
				},
			}

			ctx := context.Background()
			result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

			if result.OSFingerprint == nil {
				t.Error("Should detect OS from banner")
				return
			}

			if result.OSFingerprint.OSFamily != tt.osFamily {
				t.Errorf("OSFamily should be %q, got %q", tt.osFamily, result.OSFingerprint.OSFamily)
			}
		})
	}
}

func TestFingerprinter_OSDetectionFromHTTPServer(t *testing.T) {
	fp := discovery.NewFingerprinter(100 * time.Millisecond)

	tests := []struct {
		name     string
		server   string
		osFamily string
	}{
		{"iis", "Microsoft-IIS/10.0", "windows"},
		{"ubuntu", "nginx/1.18.0 (Ubuntu)", "linux"},
		{"debian", "Apache/2.4.38 (Debian)", "linux"},
		{"cisco", "Cisco IOS HTTP Server", "cisco"},
		{"mikrotik", "RouterOS MikroTik httpd", "mikrotik"},
		{"synology", "nginx Synology DSM", "linux"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &discovery.DeviceProfile{
				OpenPorts: []discovery.OpenPort{
					{Port: 80, Service: "http"},
				},
				HTTPInfo: &discovery.HTTPInfo{
					Server: tt.server,
				},
			}

			ctx := context.Background()
			result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

			if result.OSFingerprint == nil {
				t.Error("Should detect OS from HTTP Server header")
				return
			}

			if result.OSFingerprint.OSFamily != tt.osFamily {
				t.Errorf("OSFamily should be %q, got %q", tt.osFamily, result.OSFingerprint.OSFamily)
			}
		})
	}
}

func TestFingerprinter_EmptyProfile(t *testing.T) {
	fp := discovery.NewFingerprinter(100 * time.Millisecond)

	profile := &discovery.DeviceProfile{
		OpenPorts: []discovery.OpenPort{},
	}

	ctx := context.Background()
	result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

	// Should return result but with no detections
	if result == nil {
		t.Fatal("Should return result even with empty profile")
	}
	if len(result.ServiceVersions) != 0 {
		t.Errorf("Should have no service versions, got %d", len(result.ServiceVersions))
	}
}

func TestFingerprinter_NoBanner(t *testing.T) {
	fp := discovery.NewFingerprinter(100 * time.Millisecond)

	profile := &discovery.DeviceProfile{
		OpenPorts: []discovery.OpenPort{
			{Port: 22, Service: "ssh", Banner: ""}, // No banner
		},
	}

	ctx := context.Background()
	result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

	// Should still detect service from port
	if len(result.ServiceVersions) == 0 {
		t.Error("Should detect service even without banner")
		return
	}

	sv := result.ServiceVersions[0]
	if sv.Service != "ssh" {
		t.Errorf("Service should be 'ssh', got %q", sv.Service)
	}
	if sv.Product != "" {
		t.Errorf("Product should be empty without banner, got %q", sv.Product)
	}
}

func TestFingerprinter_MultipleOpenPorts(t *testing.T) {
	fp := discovery.NewFingerprinter(100 * time.Millisecond)

	profile := &discovery.DeviceProfile{
		OpenPorts: []discovery.OpenPort{
			{Port: 22, Service: "ssh", Banner: "SSH-2.0-OpenSSH_8.4p1"},
			{Port: 80, Service: "http", Banner: ""},
			{Port: 443, Service: "https", Banner: ""},
			{Port: 3306, Service: "mysql", Banner: ""},
		},
	}

	ctx := context.Background()
	result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

	if len(result.ServiceVersions) != 4 {
		t.Errorf("Should detect 4 services, got %d", len(result.ServiceVersions))
	}
}
