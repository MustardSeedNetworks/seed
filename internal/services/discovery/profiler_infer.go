package discovery

// profiler_infer.go contains the device-type / icon classifier that runs after
// probing finishes, plus the small string helpers shared with profiler_scan.go
// (HTML title extraction, log-string truncation) and the port→service name map.

import (
	"regexp"
	"strings"
)

// inferDeviceType infers device type and icons from the profile.
func (p *DeviceProfiler) inferDeviceType(profile *DeviceProfile) {
	icons := make(map[string]bool)
	deviceType := serviceUnknown

	deviceType = p.inferFromPorts(profile.OpenPorts, icons, deviceType)
	deviceType = p.inferFromHTTPInfo(profile.HTTPInfo, icons, deviceType)

	for icon := range icons {
		profile.DeviceIcons = append(profile.DeviceIcons, icon)
	}

	if deviceType == serviceUnknown && len(profile.OpenPorts) > 0 {
		deviceType = "host"
	}

	profile.DeviceType = deviceType
}

// inferFromPorts infers device type and icons from open ports.
func (p *DeviceProfiler) inferFromPorts(
	ports []OpenPort,
	icons map[string]bool,
	deviceType string,
) string {
	for _, op := range ports {
		deviceType = p.setIconsForPort(op.Port, icons, deviceType)
		deviceType = p.inferFromBanner(op.Banner, deviceType)
	}
	return deviceType
}

// setIconsForPort sets icons based on port number and returns updated device type.
func (p *DeviceProfiler) setIconsForPort(
	port int,
	icons map[string]bool,
	deviceType string,
) string {
	switch port {
	case portSSHProf:
		icons["ssh"] = true
	case portTelnet:
		icons["telnet"] = true
	case portHTTPProf, portHTTPAltP:
		icons["web"] = true
	case portHTTPSProf, portHTTPSAltP:
		icons["web-secure"] = true
	case portFTP:
		icons["ftp"] = true
	case portSMTP, portSMTPSubmit:
		icons["mail"] = true
	case portDNS:
		icons["dns"] = true
	case portSNMP:
		icons["snmp"] = true
	case portMySQL, portPostgreSQL:
		icons["database"] = true
	case portRedis:
		icons["cache"] = true
	case portJetDirect, portLPD, portIPP:
		icons["printer"] = true
		deviceType = deviceTypePrinter
	}
	return deviceType
}

// inferFromBanner infers device type from service banner.
func (p *DeviceProfiler) inferFromBanner(banner, deviceType string) string {
	bannerLower := strings.ToLower(banner)
	if !strings.Contains(bannerLower, "ssh") {
		return deviceType
	}
	if strings.Contains(bannerLower, "cisco") {
		return deviceTypeNetworkDevice
	}
	if strings.Contains(bannerLower, "ubuntu") || strings.Contains(bannerLower, "debian") {
		return deviceTypeServer
	}
	return deviceType
}

// inferFromHTTPInfo infers device type and icons from HTTP response.
func (p *DeviceProfiler) inferFromHTTPInfo(
	httpInfo *HTTPInfo,
	icons map[string]bool,
	deviceType string,
) string {
	if httpInfo == nil {
		return deviceType
	}

	titleLower := strings.ToLower(httpInfo.Title)
	serverLower := strings.ToLower(httpInfo.Server)

	if t, icon := matchHTTPDeviceType(titleLower, serverLower); t != "" {
		icons[icon] = true
		return t
	}

	return deviceType
}

// getHTTPDeviceMatchers returns the patterns for HTTP-based device detection.
func getHTTPDeviceMatchers() []httpDeviceMatch {
	return []httpDeviceMatch{
		{[]string{deviceTypeRouter}, []string{deviceTypeRouter}, deviceTypeRouter, deviceTypeRouter},
		{[]string{deviceTypeSwitch}, nil, deviceTypeSwitch, deviceTypeSwitch},
		{
			[]string{deviceTypeFirewall, "pfsense", "opnsense", vendorFortinet},
			nil,
			deviceTypeFirewall,
			deviceTypeFirewall,
		},
		{[]string{"nas", "synology", "qnap"}, nil, deviceTypeNAS, "storage"},
		{[]string{deviceTypePrinter, "hp ", "canon", "epson"}, nil, deviceTypePrinter, deviceTypePrinter},
		{nil, []string{webServerApache, "nginx"}, deviceTypeServer, deviceTypeServer},
	}
}

// matchHTTPDeviceType matches HTTP title/server against known device patterns.
func matchHTTPDeviceType(title, server string) (string, string) {
	for _, m := range getHTTPDeviceMatchers() {
		for _, p := range m.titlePatterns {
			if strings.Contains(title, p) {
				return m.deviceType, m.icon
			}
		}
		for _, p := range m.serverPatterns {
			if strings.Contains(server, p) {
				return m.deviceType, m.icon
			}
		}
	}
	return "", ""
}

// truncateString truncates a string to maxLen with ellipsis.
// Fixes #982: Guard against maxLen < 3 to prevent negative slice index panic.
func truncateString(s string, maxLen int) string {
	if maxLen < profilerMinTruncateLen {
		// Can't fit ellipsis, just truncate to maxLen (or return empty for 0 or negative)
		if maxLen <= 0 {
			return ""
		}
		if len(s) <= maxLen {
			return s
		}
		return s[:maxLen]
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// extractHTMLTitle extracts the <title> from HTML content.
func extractHTMLTitle(html string) string {
	re := regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		title := strings.TrimSpace(matches[1])
		// Truncate long titles
		if len(title) > profilerTitleMaxLen {
			title = title[:profilerTitleMaxLen] + "..."
		}
		return title
	}
	return ""
}

// portToService maps common ports to service names.
func portToService(port int) string {
	services := map[int]string{
		21:    "ftp",
		22:    "ssh",
		23:    "telnet",
		25:    "smtp",
		53:    "dns",
		80:    "http",
		110:   "pop3",
		143:   "imap",
		161:   "snmp",
		443:   "https",
		445:   "smb",
		515:   "lpd",
		587:   "submission",
		631:   "ipp",
		993:   "imaps",
		995:   "pop3s",
		3306:  "mysql",
		3389:  "rdp",
		5432:  "postgresql",
		5900:  "vnc",
		6379:  "redis",
		8080:  "http-alt",
		8443:  "https-alt",
		9100:  "jetdirect",
		27017: "mongodb",
	}

	if s, ok := services[port]; ok {
		return s
	}
	return ""
}
