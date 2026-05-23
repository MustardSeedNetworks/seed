package config

// config_defaults.go contains DefaultConfig and the per-section default*
// helpers it composes. Keeping these in one file makes the "what does a fresh
// install look like?" question easy to answer.

import "time"

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Version: ConfigVersion,
		Server:  ServerConfig{Port: defaultHTTPSPort, HTTPS: true},
		Interface: InterfaceConfig{
			Default:          "",
			Fallbacks:        []string{},
			StartupRetries:   defaultStartupRetries,
			StartupRetryWait: defaultStartupRetryWaitSec * time.Second,
		},
		VLAN: VLANConfig{Enabled: false, ID: 0},
		IP:   IPConfig{Mode: ipModeDHCP},
		Discovery: DiscoveryConfig{
			Protocol: "auto",
			Timeout:  defaultDiscoveryTimeoutSec * time.Second,
		},
		NetworkDiscovery: defaultNetworkDiscoveryConfig(),
		SNMP: SNMPConfig{
			Communities:    []string{"public"},
			Timeout:        defaultSNMPTimeoutSec * time.Second,
			Retries:        defaultSNMPRetries,
			Port:           defaultSNMPPort,
			MaxRepetitions: defaultSNMPMaxRepetitions,
		},
		DNS: DNSConfig{
			TestHostname: "google.com",
			Timeout:      defaultDNSTimeoutSec * time.Second,
		},
		HealthChecks: defaultHealthChecksConfig(),
		Speedtest:    SpeedtestConfig{ServerID: "", AutoRunOnLink: true},
		Thresholds:   defaultThresholdsConfig(),
		Auth:         defaultAuthConfig(),
		Security:     defaultSecurityConfig(),
		Iperf: IperfConfig{
			AutoRunOnLink: false,
			Server:        "",
			Port:          defaultIperfPort,
			Protocol:      "tcp",
			Direction:     "download",
			Duration:      defaultIperfDurationSec,
			ServerPort:    defaultIperfPort,
			EnableServer:  true,
		},
		FABOptions: FABOptionsConfig{
			RunLink:             true,
			RunSwitch:           true,
			RunVLAN:             true,
			RunIPConfig:         true,
			RunGateway:          true,
			RunDNS:              true,
			RunHealthChecks:     true,
			RunNetworkDiscovery: true,
			RunSpeedtest:        true,
			RunIperf:            false,
			RunPerformance:      true,
			AutoScanOnLink:      true,
		},
		DisplayOptions: DisplayOptionsConfig{ShowPublicIP: true, UnitSystem: "sae"},
		Logging: LoggingConfig{
			Level:      logLevelInfo,
			Format:     logFormatText,
			AddSource:  false,
			File:       "",
			MaxSize:    defaultLogMaxSizeMB,
			MaxBackups: defaultLogMaxBackups,
			MaxAge:     defaultLogMaxAgeDays,
			Compress:   true,
		},
		Database: DatabaseConfig{
			Path:           "data/seed.db",
			RetentionDays:  defaultDBRetentionDays,
			EnableWAL:      true,
			MaxConnections: defaultDBMaxConnections,
		},
		Pipeline: defaultPipelineConfig(),
	}
}

// defaultNetworkDiscoveryConfig returns the default network discovery configuration.
func defaultNetworkDiscoveryConfig() NetworkDiscoveryConfig {
	return NetworkDiscoveryConfig{
		Options: DiscoveryOptions{
			PassiveProtocols: PassiveProtocolConfig{LLDP: true, CDP: true, EDP: true, NDP: true},
			ARPScan:          true, ICMPScan: true,
			PortScan: PortScanConfig{
				Enabled:       false,
				Preset:        PortPresetCommon,
				TCPPorts:      "",
				UDPPorts:      "",
				BannerTimeout: defaultBannerTimeoutSec * time.Second,
			},
			TCPProbe: TCPProbeConfig{
				Timeout: defaultTracerouteTimeoutSec * time.Second,
				Workers: defaultTracerouteWorkers,
			}, Traceroute: false, SNMPQuery: false,
		},
		Profiler: DeviceProfilerConfig{
			Enabled:       true,
			Timeout:       defaultMDNSTimeoutSec * time.Second,
			MaxConcurrent: defaultMDNSMaxConcurrent,
			QuickPorts:    []int{portSSH, portHTTP, portHTTPS, portHTTPAlt},
		},
		Timing: DiscoveryTiming{
			ProbeInterval:  defaultProbeIntervalMs * time.Millisecond,
			RescanInterval: defaultRescanIntervalMin * time.Minute,
			Workers:        defaultARPWorkers,
		},
		Fingerprinting: FingerprintingConfig{
			Enabled:       false,
			OSDetection:   false,
			ServiceProbes: false,
		},
		IPv6Enabled:       true,
		Enabled:           true,
		ARPScanWorkers:    defaultARPWorkers,
		PingTimeout:       defaultPingTimeoutMs * time.Millisecond,
		ScanTimeout:       defaultScanTimeoutSec * time.Second,
		AutoScan:          true,
		ScanInterval:      0,
		OUIFilePath:       "data/oui.txt",
		OUIMaxAge:         defaultOUIMaxAgeDays * 24 * time.Hour,
		AdditionalSubnets: []SubnetConfig{},
	}
}

// defaultHealthChecksConfig returns the default health checks configuration.
func defaultHealthChecksConfig() HealthChecksConfig {
	return HealthChecksConfig{
		PingTargets: []PingTarget{
			{Name: "Google DNS", Host: "8.8.8.8", Enabled: true},
			{Name: "Cloudflare", Host: "1.1.1.1", Enabled: true},
		},
		TCPPorts: []TCPPortTest{
			{
				Name:    "HTTPS",
				Host:    "www.google.com",
				Port:    portHTTPS,
				Enabled: true,
			},
			{Name: "DICOM", Host: "dicomserver.co.uk", Port: portDICOM, Enabled: true},
			{
				Name:    "FTP",
				Host:    "ftp.debian.org",
				Port:    portFTP,
				Enabled: true,
			},
			{Name: "SMB", Host: "files.example.com", Port: portSMB, Enabled: false},
			{
				Name:    "RTSP",
				Host:    "wowzaec2demo.streamlock.net",
				Port:    portRTSP,
				Enabled: true,
			},
			{Name: "PostgreSQL", Host: "db.example.com", Port: portPostgreSQL, Enabled: false},
			{Name: "SFTP", Host: "sftp.example.com", Port: portSSH, Enabled: false},
		},
		UDPPorts: []UDPPortTest{
			{Name: "DNS", Host: "8.8.8.8", Port: portDNS, Enabled: true},
			{Name: "NTP", Host: "time.google.com", Port: portNTP, Enabled: true},
		},
		HTTPEndpoints: []HTTPEndpoint{
			{
				Name:           "Google HTTPS",
				URL:            "https://www.google.com",
				ExpectedStatus: httpStatusOK,
				Enabled:        true,
			},
			{
				Name:           "Cloudflare",
				URL:            "https://www.cloudflare.com",
				ExpectedStatus: httpStatusOK,
				Enabled:        true,
			},
			{
				Name:           "Example HTTP",
				URL:            "http://example.com",
				ExpectedStatus: httpStatusOK,
				Enabled:        true,
			},
		},
		// Issue #778: RTSP stream health checks
		RTSPEndpoints: []RTSPEndpoint{
			{
				Name:    "Wowza Demo Stream",
				URL:     "rtsp://wowzaec2demo.streamlock.net/vod/mp4:BigBuckBunny_115k.mp4",
				Enabled: true,
			},
		},
		// Issue #777: DICOM server health checks
		DICOMEndpoints: []DICOMEndpoint{
			{
				Name:      "Public DICOM Server",
				Host:      "dicomserver.co.uk",
				Port:      portDICOM,
				CalledAE:  "ANY-SCP",
				CallingAE: "SEED-SCU",
				Enabled:   true,
			},
		},
		RunPerformance: true, RunSpeedtest: true, RunIperf: true, RunDiscovery: true,
	}
}

// defaultThresholdsConfig returns the default thresholds configuration.
func defaultThresholdsConfig() ThresholdsConfig {
	return ThresholdsConfig{
		DHCP: DHCPThresholds{
			Total: Threshold{
				Warning:  thresholdDHCPTotalWarningMs * time.Millisecond,
				Critical: defaultBannerTimeoutSec * time.Second,
			},
			PerPhase: Threshold{
				Warning:  thresholdDHCPPhaseWarningMs * time.Millisecond,
				Critical: 1 * time.Second,
			},
		},
		DNS: Threshold{
			Warning:  thresholdDNSWarningMs * time.Millisecond,
			Critical: thresholdDHCPTotalWarningMs * time.Millisecond,
		},
		Ping: Threshold{
			Warning:  thresholdPingWarningMs * time.Millisecond,
			Critical: thresholdPingCriticalMs * time.Millisecond,
		},
		WiFi: WiFiThresholds{
			Signal: SignalThreshold{
				Warning:  thresholdWiFiSignalWarningDBm,
				Critical: thresholdWiFiSignalCriticalDBm,
			},
		},
		Link: LinkThresholds{
			FlapCount24h: IntThreshold{
				Warning:  thresholdLinkFlapWarning,
				Critical: thresholdLinkFlapCritical,
			},
		},
		CustomTests: CustomThresholds{
			Ping: Threshold{
				Warning:  thresholdPingWarningMs * time.Millisecond,
				Critical: thresholdCustomPingCriticalMs * time.Millisecond,
			},
			TCP: Threshold{
				Warning:  thresholdTCPWarningMs * time.Millisecond,
				Critical: thresholdDHCPTotalWarningMs * time.Millisecond,
			},
			UDP: Threshold{
				Warning:  thresholdTCPWarningMs * time.Millisecond,
				Critical: thresholdDHCPTotalWarningMs * time.Millisecond,
			},
			HTTP: Threshold{
				Warning:  thresholdHTTPWarningMs * time.Millisecond,
				Critical: defaultBannerTimeoutSec * time.Second,
			},
			HTTPTimings: HTTPTimingThresholds{
				DNS: Threshold{
					Warning:  thresholdDNSWarningMs * time.Millisecond,
					Critical: thresholdDHCPTotalWarningMs * time.Millisecond,
				},
				TCP: Threshold{
					Warning:  thresholdTCPWarningMs * time.Millisecond,
					Critical: thresholdDHCPTotalWarningMs * time.Millisecond,
				},
				TLS: Threshold{
					Warning:  thresholdTLSWarningMs * time.Millisecond,
					Critical: thresholdDHCPTotalWarningMs * time.Millisecond,
				},
				TTFB: Threshold{
					Warning:  thresholdHTTPWarningMs * time.Millisecond,
					Critical: defaultBannerTimeoutSec * time.Second,
				},
			},
			CertExpiry: CertExpiryThreshold{
				Warning:  thresholdCertExpiryWarningDays,
				Critical: thresholdCertExpiryCriticalDays,
			},
		},
	}
}

// defaultAuthConfig returns the default authentication configuration.
func defaultAuthConfig() AuthConfig {
	return AuthConfig{
		DefaultUsername:     "admin",
		DefaultPasswordHash: "",
		SessionTimeout:      defaultSessionTimeoutHours * time.Hour,
		JWTSecret:           "",
		SSO: SSOConfig{
			Providers: []SSOProviderConfig{
				{Name: "google", Enabled: false},
				{Name: "microsoft", Enabled: false},
				{Name: "github", Enabled: false},
			},
		},
	}
}

// defaultSecurityConfig returns the default security configuration.
func defaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		AllowedOrigins: []string{},
		VulnerabilityScanning: VulnerabilityScanConfig{
			Enabled:           true,
			CVEDatabase:       "nvd",
			NVDAPIKey:         "",
			UpdateInterval:    defaultVulnUpdateIntervalSec,
			SeverityThreshold: "medium",
			MaxConcurrent:     defaultMDNSMaxConcurrent,
			AutoScan:          true,
		},
	}
}

// defaultPipelineConfig returns the default pipeline configuration.
func defaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		Phases: PipelinePhaseConfig{
			Enumeration:      true,
			NameResolution:   true,
			ServiceDiscovery: true,
			VulnAssessment:   false,
		},
		Timing: PipelineTimingConfig{
			ProbeDelay:         defaultPipelineProbeDelayMs * time.Millisecond,
			HostDelay:          defaultPipelineHostDelayMs * time.Millisecond,
			MaxConcurrentHosts: defaultPipelineMaxConcurrentHosts,
			PhaseTimeout:       defaultPipelinePhaseTimeoutMin * time.Minute,
			Profile:            "normal",
		},
		PortScan: PipelinePortScanConfig{
			Intensity:      "off",
			BannerGrab:     true,
			ConnectTimeout: defaultBannerTimeoutSec * time.Second,
		},
		SNMPCollection: PipelineSNMPConfig{
			Enabled: true,
			MIBs: PipelineSNMPMIBs{
				System:      true,
				Interfaces:  true,
				IPAddresses: true,
				Routing:     false,
				Bridge:      false,
				Entity:      false,
				LLDP:        true,
				VLAN:        false,
			},
			WalkTimeout: defaultSNMPWalkTimeoutSec * time.Second, MaxOIDsPerRequest: defaultSNMPMaxOIDsPerRequest,
		},
		Persistence: PipelinePersistenceConfig{
			StoreHistory:       true,
			StalenessThreshold: defaultStalenessThresholdHours * time.Hour,
			PurgeAfter:         defaultPurgeAfterDays * 24 * time.Hour,
		},
	}
}
