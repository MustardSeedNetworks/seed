package discovery_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestFingerprinter_OSDetectionPatterns(t *testing.T) {
	fp := discovery.NewFingerprinter(100 * time.Millisecond)

	// Test SSH banner patterns
	t.Run("ssh_banners", func(t *testing.T) {
		sshTests := []struct {
			name     string
			banner   string
			osFamily string
		}{
			{"ubuntu_ssh", "SSH-2.0-OpenSSH_8.4p1 Ubuntu-5ubuntu1.4", "linux"},
			{"debian_ssh", "SSH-2.0-OpenSSH_7.9p1 Debian-10+deb10u2", "linux"},
			{"centos_ssh", "SSH-2.0-OpenSSH_7.4 CentOS-7", "linux"},
			{"rhel_ssh", "SSH-2.0-OpenSSH_7.4 Red Hat Enterprise Linux", "linux"},
			{"freebsd_ssh", "SSH-2.0-OpenSSH_7.3 FreeBSD-20170902", "bsd"},
			{"cisco_ssh", "SSH-2.0-Cisco-1.25", "cisco"},
			{"windows_ssh", "SSH-2.0-OpenSSH_for_Windows_8.1", "windows"},
		}

		for _, tt := range sshTests {
			t.Run(tt.name, func(t *testing.T) {
				profile := &discovery.DeviceProfile{
					OpenPorts: []discovery.OpenPort{
						{Port: 22, Service: "ssh", Banner: tt.banner},
					},
				}

				ctx := context.Background()
				result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

				if result.OSFingerprint == nil {
					t.Fatal("Expected OS fingerprint")
				}
				if result.OSFingerprint.OSFamily != tt.osFamily {
					t.Errorf(
						"OSFamily should be %q, got %q",
						tt.osFamily,
						result.OSFingerprint.OSFamily,
					)
				}
			})
		}
	})

	// Test HTTP Server header patterns
	t.Run("http_server_headers", func(t *testing.T) {
		httpTests := []struct {
			name     string
			server   string
			osFamily string
		}{
			{"iis_10", "Microsoft-IIS/10.0", "windows"},
			{"iis_8_5", "Microsoft-IIS/8.5", "windows"},
			{"nginx_ubuntu", "nginx/1.18.0 (Ubuntu)", "linux"},
			{"apache_debian", "Apache/2.4.38 (Debian)", "linux"},
			{"apache_centos", "Apache/2.4.6 (CentOS)", "linux"},
			{"cisco_ios", "Cisco IOS HTTP Server", "cisco"},
			{"mikrotik_ros", "RouterOS MikroTik httpd", "mikrotik"},
			{"fortinet", "Fortinet Firewall", "fortinet"},
			{"fortigate", "FortiGate Application Control", "fortinet"},
			{"pfsense", "pfSense webConfigurator", "bsd"},
			{"opnsense", "OPNsense web interface", "bsd"},
			{"synology_dsm", "nginx Synology DSM 7.1", "linux"},
			{"qnap_qts", "QNAP QTS NAS", "linux"},
		}

		for _, tt := range httpTests {
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
					t.Fatal("Expected OS fingerprint")
				}
				if result.OSFingerprint.OSFamily != tt.osFamily {
					t.Errorf(
						"OSFamily for server %q should be %q, got %q",
						tt.server,
						tt.osFamily,
						result.OSFingerprint.OSFamily,
					)
				}
			})
		}
	})
}

func TestFingerprinter_ServiceVersionDetection(t *testing.T) {
	fp := discovery.NewFingerprinter(100 * time.Millisecond)

	t.Run("ssh_versions", func(t *testing.T) {
		tests := []struct {
			name    string
			banner  string
			product string
			version string
		}{
			{
				"openssh_8_4",
				"SSH-2.0-OpenSSH_8.4p1 Ubuntu",
				"OpenSSH",
				"8.4p1",
			},
			{
				"openssh_7_9",
				"SSH-2.0-OpenSSH_7.9p1 Debian-10",
				"OpenSSH",
				"7.9p1",
			},
			{
				"openssh_underscore",
				"SSH-2.0-OpenSSH_9.0",
				"OpenSSH",
				"9.0",
			},
			{
				"generic_ssh",
				"SSH-2.0-dropbear_2022.83",
				"SSH",
				"2.0",
			},
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

				if len(result.ServiceVersions) == 0 {
					t.Fatal("Expected service versions")
				}

				sv := result.ServiceVersions[0]
				if sv.Product != tt.product {
					t.Errorf("Product should be %q, got %q", tt.product, sv.Product)
				}
				if sv.Version != tt.version {
					t.Errorf("Version should be %q, got %q", tt.version, sv.Version)
				}
			})
		}
	})

	t.Run("ftp_versions", func(t *testing.T) {
		tests := []struct {
			name    string
			banner  string
			product string
		}{
			{"vsftpd", "220 (vsFTPd 3.0.3)", "vsftpd"},
			{"proftpd", "220 ProFTPD 1.3.6 Server ready", "ProFTPD"},
			{"pureftpd", "220 Welcome to Pure-FTPd Server", "Pure-FTPd"},
			{"msftp", "220 Microsoft FTP Service Ready", "Microsoft FTP"},
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
					t.Fatal("Expected service versions")
				}

				sv := result.ServiceVersions[0]
				if sv.Product != tt.product {
					t.Errorf("Product should be %q, got %q", tt.product, sv.Product)
				}
				if sv.Service != "ftp" {
					t.Errorf("Service should be 'ftp', got %q", sv.Service)
				}
			})
		}
	})

	t.Run("smtp_versions", func(t *testing.T) {
		tests := []struct {
			name    string
			port    int
			banner  string
			product string
		}{
			{"postfix_25", 25, "220 mail.example.com ESMTP Postfix (Ubuntu)", "Postfix"},
			{"sendmail_25", 25, "220 mail.example.com ESMTP Sendmail 8.15.2", "Sendmail"},
			{"exim_25", 25, "220 mail.example.com ESMTP Exim 4.93 #1", "Exim"},
			{"exchange_587", 587, "220 exchange.example.com Microsoft ESMTP MAIL Service", "Microsoft Exchange"},
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
					t.Fatal("Expected service versions")
				}

				sv := result.ServiceVersions[0]
				if sv.Product != tt.product {
					t.Errorf("Product should be %q, got %q", tt.product, sv.Product)
				}
			})
		}
	})

	t.Run("telnet_versions", func(t *testing.T) {
		tests := []struct {
			name    string
			banner  string
			product string
		}{
			{"cisco_telnet", "User Access Verification\n\nUsername: cisco", "Cisco IOS"},
			{"linux_telnet", "Debian GNU/Linux telnetd", "Linux telnetd"},
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
					t.Fatal("Expected service versions")
				}

				sv := result.ServiceVersions[0]
				if sv.Product != tt.product {
					t.Errorf("Product should be %q, got %q", tt.product, sv.Product)
				}
			})
		}
	})
}

func TestFingerprinter_EdgeCases(t *testing.T) {
	fp := discovery.NewFingerprinter(100 * time.Millisecond)

	t.Run("nil_profile", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Using TEST-NET-1 which is non-routable
		result := fp.ProbeDevice(ctx, "192.0.2.1", nil)

		if result == nil {
			t.Fatal("Result should not be nil")
		}
		if result.IP != "192.0.2.1" {
			t.Errorf("IP should be 192.0.2.1, got %q", result.IP)
		}
	})

	t.Run("empty_ports", func(t *testing.T) {
		profile := &discovery.DeviceProfile{
			OpenPorts: []discovery.OpenPort{},
		}

		ctx := context.Background()
		result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

		if len(result.ServiceVersions) != 0 {
			t.Errorf("Expected no service versions, got %d", len(result.ServiceVersions))
		}
	})

	t.Run("no_banner", func(t *testing.T) {
		profile := &discovery.DeviceProfile{
			OpenPorts: []discovery.OpenPort{
				{Port: 22, Service: "ssh", Banner: ""},
				{Port: 80, Service: "http", Banner: ""},
			},
		}

		ctx := context.Background()
		result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

		// Should still create service entries without product/version
		if len(result.ServiceVersions) != 2 {
			t.Errorf("Expected 2 service versions, got %d", len(result.ServiceVersions))
		}
	})

	t.Run("empty_http_info", func(t *testing.T) {
		profile := &discovery.DeviceProfile{
			OpenPorts: []discovery.OpenPort{
				{Port: 80, Service: "http"},
			},
			HTTPInfo: &discovery.HTTPInfo{
				Server: "",
			},
		}

		ctx := context.Background()
		result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

		// Should not crash with empty HTTPInfo.Server
		if result == nil {
			t.Fatal("Result should not be nil")
		}
	})

	t.Run("nil_http_info", func(t *testing.T) {
		profile := &discovery.DeviceProfile{
			OpenPorts: []discovery.OpenPort{
				{Port: 80, Service: "http"},
			},
			HTTPInfo: nil,
		}

		ctx := context.Background()
		result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

		// Should not crash with nil HTTPInfo
		if result == nil {
			t.Fatal("Result should not be nil")
		}
	})

	t.Run("multiple_ports_same_service", func(t *testing.T) {
		profile := &discovery.DeviceProfile{
			OpenPorts: []discovery.OpenPort{
				{Port: 22, Service: "ssh", Banner: "SSH-2.0-OpenSSH_8.4p1"},
				{Port: 2222, Service: "ssh", Banner: "SSH-2.0-OpenSSH_7.9p1"},
			},
		}

		ctx := context.Background()
		result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

		if len(result.ServiceVersions) != 2 {
			t.Errorf("Expected 2 service versions, got %d", len(result.ServiceVersions))
		}
	})
}

func TestFingerprinter_ContextCancellation(t *testing.T) {
	fp := discovery.NewFingerprinter(5 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should handle cancelled context gracefully
	result := fp.ProbeDevice(ctx, "192.0.2.1", nil)

	if result == nil {
		t.Fatal("Result should not be nil even with cancelled context")
	}
}

func TestFingerprinter_Timeouts(t *testing.T) {
	// Very short timeout
	fp := discovery.NewFingerprinter(1 * time.Millisecond)

	profile := &discovery.DeviceProfile{
		OpenPorts: []discovery.OpenPort{
			{Port: 443, Service: "https"},
		},
	}

	ctx := context.Background()
	result := fp.ProbeDevice(ctx, "192.0.2.1", profile)

	// Should complete without hanging
	if result == nil {
		t.Fatal("Result should not be nil")
	}
}

func TestTLSInfo_Comprehensive(t *testing.T) {
	now := time.Now()
	validFrom := now.Add(-90 * 24 * time.Hour) // 90 days ago
	validTo := now.Add(275 * 24 * time.Hour)   // 275 days from now

	tests := []struct {
		name      string
		tlsInfo   discovery.TLSInfo
		hasErrors bool
	}{
		{
			name: "valid_certificate",
			tlsInfo: discovery.TLSInfo{
				Port:              443,
				Version:           "TLS 1.3",
				CipherSuite:       "TLS_AES_256_GCM_SHA384",
				CommonName:        "example.com",
				Issuer:            "Let's Encrypt Authority X3",
				ValidFrom:         validFrom,
				ValidTo:           validTo,
				DaysUntilExpiry:   275,
				SelfSigned:        false,
				SubjectAltNames:   []string{"example.com", "www.example.com"},
				CertificateErrors: []string{},
			},
			hasErrors: false,
		},
		{
			name: "self_signed_certificate",
			tlsInfo: discovery.TLSInfo{
				Port:              443,
				Version:           "TLS 1.2",
				CipherSuite:       "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
				CommonName:        "localhost",
				Issuer:            "localhost",
				ValidFrom:         validFrom,
				ValidTo:           validTo,
				DaysUntilExpiry:   275,
				SelfSigned:        true,
				SubjectAltNames:   []string{"localhost"},
				CertificateErrors: []string{"self-signed"},
			},
			hasErrors: true,
		},
		{
			name: "expiring_soon",
			tlsInfo: discovery.TLSInfo{
				Port:              443,
				Version:           "TLS 1.2",
				CipherSuite:       "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
				CommonName:        "example.org",
				Issuer:            "DigiCert",
				ValidFrom:         now.Add(-335 * 24 * time.Hour),
				ValidTo:           now.Add(15 * 24 * time.Hour),
				DaysUntilExpiry:   15,
				SelfSigned:        false,
				SubjectAltNames:   []string{"example.org"},
				CertificateErrors: []string{"expiring soon"},
			},
			hasErrors: true,
		},
		{
			name: "expired_certificate",
			tlsInfo: discovery.TLSInfo{
				Port:              443,
				Version:           "TLS 1.2",
				CipherSuite:       "TLS_RSA_WITH_AES_128_GCM_SHA256",
				CommonName:        "expired.example.com",
				Issuer:            "Comodo",
				ValidFrom:         now.Add(-400 * 24 * time.Hour),
				ValidTo:           now.Add(-35 * 24 * time.Hour),
				DaysUntilExpiry:   -35,
				SelfSigned:        false,
				SubjectAltNames:   []string{},
				CertificateErrors: []string{"expired"},
			},
			hasErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tlsInfo.Port != 443 {
				t.Errorf("Port should be 443, got %d", tt.tlsInfo.Port)
			}

			hasErrors := len(tt.tlsInfo.CertificateErrors) > 0
			if hasErrors != tt.hasErrors {
				t.Errorf("hasErrors mismatch: expected %v, got %v", tt.hasErrors, hasErrors)
			}
		})
	}
}

func TestAdvancedProbeResult_Comprehensive(t *testing.T) {
	now := time.Now()

	result := discovery.AdvancedProbeResult{
		IP:       "10.0.0.1",
		ProbedAt: now,
		OSFingerprint: &discovery.OSFingerprint{
			OSFamily:    "linux",
			OSVersion:   "ubuntu",
			Confidence:  90,
			Methods:     []string{"banner", "http"},
			TTLObserved: 64,
		},
		ServiceVersions: []discovery.ServiceVersion{
			{Port: 22, Service: "ssh", Product: "OpenSSH", Version: "8.4p1", Confidence: 95},
			{Port: 80, Service: "http", Product: "nginx", Version: "1.18.0", Confidence: 90},
			{Port: 443, Service: "https", Confidence: 50},
		},
		TLSInfo: []discovery.TLSInfo{
			{
				Port:            443,
				Version:         "TLS 1.3",
				CommonName:      "example.com",
				DaysUntilExpiry: 100,
			},
		},
	}

	if result.IP != "10.0.0.1" {
		t.Errorf("IP mismatch: got %q", result.IP)
	}
	if !result.ProbedAt.Equal(now) {
		t.Error("ProbedAt should match now")
	}
	if result.OSFingerprint.Confidence != 90 {
		t.Errorf("Confidence should be 90, got %d", result.OSFingerprint.Confidence)
	}
	if len(result.ServiceVersions) != 3 {
		t.Errorf("Expected 3 service versions, got %d", len(result.ServiceVersions))
	}
	if len(result.TLSInfo) != 1 {
		t.Errorf("Expected 1 TLS info, got %d", len(result.TLSInfo))
	}
}

func TestOSFingerprint_ConfidenceLevels(t *testing.T) {
	tests := []struct {
		name       string
		fp         discovery.OSFingerprint
		isHighConf bool
	}{
		{
			name: "high_confidence_banner",
			fp: discovery.OSFingerprint{
				OSFamily:   "linux",
				OSVersion:  "ubuntu",
				Confidence: 95,
				Methods:    []string{"banner"},
			},
			isHighConf: true,
		},
		{
			name: "medium_confidence_ttl",
			fp: discovery.OSFingerprint{
				OSFamily:    "linux",
				Confidence:  50,
				Methods:     []string{"ttl"},
				TTLObserved: 64,
			},
			isHighConf: false,
		},
		{
			name: "low_confidence_guess",
			fp: discovery.OSFingerprint{
				OSFamily:   "unix",
				Confidence: 30,
				Methods:    []string{},
			},
			isHighConf: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isHighConf := tt.fp.Confidence >= 80
			if isHighConf != tt.isHighConf {
				t.Errorf("isHighConf mismatch: expected %v, got %v", tt.isHighConf, isHighConf)
			}
		})
	}
}

func TestServiceVersion_AllFields(t *testing.T) {
	sv := discovery.ServiceVersion{
		Port:       3306,
		Service:    "mysql",
		Product:    "MySQL",
		Version:    "8.0.28",
		ExtraInfo:  "MySQL Community Server - GPL",
		Confidence: 95,
	}

	if sv.Port != 3306 {
		t.Errorf("Port should be 3306, got %d", sv.Port)
	}
	if sv.Service != "mysql" {
		t.Errorf("Service should be 'mysql', got %q", sv.Service)
	}
	if sv.Product != "MySQL" {
		t.Errorf("Product should be 'MySQL', got %q", sv.Product)
	}
	if sv.Version != "8.0.28" {
		t.Errorf("Version should be '8.0.28', got %q", sv.Version)
	}
	if sv.ExtraInfo == "" {
		t.Error("ExtraInfo should be set")
	}
	if sv.Confidence != 95 {
		t.Errorf("Confidence should be 95, got %d", sv.Confidence)
	}
}
