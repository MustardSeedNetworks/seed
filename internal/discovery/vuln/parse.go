package vuln

import (
	"regexp"
	"strings"
)

// CPE product constants for vulnerability scanning.
const cpeProductCiscoIOS = "cisco-ios"

// Regex match count constants for version extraction.
const (
	vulnRegexMatchCount2 = 2 // Expected match count for single capture group
	vulnRegexMatchCount3 = 3 // Expected match count for two capture groups
)

// parseCDPVersion extracts product and version from Cisco CDP info
// Example: Platform="Cisco 2960", Version="15.2(4)E" → "cisco-ios", "15.2(4)E".
func parseCDPVersion(platform, softwareVersion string) (string, string) {
	if softwareVersion == "" {
		return "", ""
	}

	// Normalize platform for product name
	platform = strings.ToLower(platform)
	product := "cisco-device"

	// Detect IOS, IOS-XE, NX-OS, etc.
	if strings.Contains(softwareVersion, "IOS") || strings.Contains(platform, "ios") {
		//nolint:gocritic // ifElseChain: order matters (XE before generic IOS)
		if strings.Contains(softwareVersion, "XE") {
			product = "cisco-ios-xe"
		} else if strings.Contains(softwareVersion, "NX-OS") {
			product = "cisco-nx-os"
		} else {
			product = cpeProductCiscoIOS
		}
	}

	// Extract version number (e.g., "15.2(4)E" from "Cisco IOS Software, Version 15.2(4)E")
	versionPattern := regexp.MustCompile(`\d+\.\d+[\(\)\w]*`)
	matches := versionPattern.FindString(softwareVersion)
	if matches != "" {
		return product, matches
	}

	return product, softwareVersion
}

// parseEDPVersion extracts product and version from Extreme EDP info
// Example: "ExtremeXOS 22.5.1.7" → "extremexos", "22.5.1.7".
func parseEDPVersion(softwareVersion string) (string, string) {
	if softwareVersion == "" {
		return "", ""
	}

	// Pattern: "ProductName X.Y.Z.W"
	pattern := regexp.MustCompile(`^(\w+)\s+([\d.]+)`)
	matches := pattern.FindStringSubmatch(softwareVersion)
	if len(matches) >= vulnRegexMatchCount3 {
		product := strings.ToLower(matches[1])
		version := matches[2]
		return product, version
	}

	return "extreme-device", softwareVersion
}

// parseSNMPSysDescr extracts product and version from SNMP sysDescr
// Example formats:
// - "Cisco IOS Software, Version 15.2(4)M" → "cisco-ios", "15.2(4)M".
// - "HP J9729A 2920-48G Switch, revision WB.16.04.0008" → "hp-2920", "WB.16.04.0008".
// snmpVendorMatcher defines a pattern for extracting product/version from SNMP sysDescr.
type snmpVendorMatcher struct {
	keywords      []string
	pattern       *regexp.Regexp
	product       string
	versionGroup  int
	productPrefix string // for dynamic product names
}

// getSNMPVendorMatchers returns vendor matchers for SNMP sysDescr parsing.
func getSNMPVendorMatchers() []snmpVendorMatcher {
	return []snmpVendorMatcher{
		{
			[]string{"cisco", "ios"},
			regexp.MustCompile(`version\s+([\d.()a-zA-Z]+)`),
			cpeProductCiscoIOS,
			1,
			"",
		},
		{
			[]string{"hp"},
			regexp.MustCompile(`(\d+\w*)\s+switch.*?revision\s+([\w.]+)`),
			"hp-switch",
			2,
			"hp-",
		},
		{
			[]string{"aruba"},
			regexp.MustCompile(`(\d+\w*)\s+switch.*?revision\s+([\w.]+)`),
			"hp-switch",
			2,
			"hp-",
		},
		{
			[]string{"juniper", "junos"},
			regexp.MustCompile(`junos\s+([\d.X-]+)`),
			"juniper-junos",
			1,
			"",
		},
	}
}

// matchVendorKeywords checks if all keywords match in the sysDescr.
func matchVendorKeywords(sysDescr string, keywords []string) bool {
	for _, kw := range keywords {
		if !strings.Contains(sysDescr, kw) {
			return false
		}
	}
	return true
}

// extractVendorProductVersion extracts product and version from a matcher's pattern match.
func extractVendorProductVersion(m snmpVendorMatcher, matches []string) (string, string) {
	if len(matches) > m.versionGroup {
		productName := m.product
		if m.productPrefix != "" && len(matches) > 1 {
			productName = m.productPrefix + matches[1]
		}
		return productName, matches[m.versionGroup]
	}
	return m.product, ""
}

// matchSNMPVendor attempts to match sysDescr against vendor-specific patterns.
// Returns product, version, and whether a match was found.
func matchSNMPVendor(sysDescr string) (string, string, bool) {
	for _, m := range getSNMPVendorMatchers() {
		if !matchVendorKeywords(sysDescr, m.keywords) {
			continue
		}
		matches := m.pattern.FindStringSubmatch(sysDescr)
		product, version := extractVendorProductVersion(m, matches)
		return product, version, true
	}
	return "", "", false
}

// extractGenericVendorVersion extracts version from sysDescr for known generic vendors.
// Returns vendor, version, and whether a match was found.
func extractGenericVendorVersion(sysDescr string) (string, string, bool) {
	vendors := []string{"cisco", "juniper", "arista", "dell", "fortinet", "paloalto", "mikrotik"}
	versionPattern := regexp.MustCompile(`\b(v?[\d.]+[\d.a-zA-Z()-]*)\b`)

	for _, vendor := range vendors {
		if !strings.Contains(sysDescr, vendor) {
			continue
		}
		version := findValidVersion(versionPattern, sysDescr)
		return vendor, version, true
	}
	return "", "", false
}

// findValidVersion searches for a valid version string in sysDescr.
func findValidVersion(pattern *regexp.Regexp, sysDescr string) string {
	matches := pattern.FindAllStringSubmatch(sysDescr, -1)
	for _, match := range matches {
		if len(match[1]) > 2 && strings.Contains(match[1], ".") {
			return match[1]
		}
	}
	return ""
}

// - "Juniper Networks, Inc. srx240h2 internet router, kernel JUNOS 12.1X47-D15.4" → "juniper-junos", "12.1X47-D15.4".
func parseSNMPSysDescr(sysDescr string) (string, string) {
	if sysDescr == "" {
		return "", ""
	}
	sysDescr = strings.ToLower(sysDescr)

	// Try specific vendor matchers first
	if product, version, ok := matchSNMPVendor(sysDescr); ok {
		return product, version
	}

	// Try generic vendor extraction
	if vendor, version, ok := extractGenericVendorVersion(sysDescr); ok {
		return vendor, version
	}

	return "", ""
}

// parseLLDPDescription extracts product and version from LLDP system description
// Example: "ProCurve J9850A Switch 2910al-48G, revision WB.16.04" → "procurve-2910", "WB.16.04".
func parseLLDPDescription(description string) (string, string) {
	if description == "" {
		return "", ""
	}

	description = strings.ToLower(description)

	// HP/Aruba ProCurve pattern
	if strings.Contains(description, "procurve") {
		pattern := regexp.MustCompile(`procurve.*?(\d+\w*),.*?revision\s+([\w.]+)`)
		matches := pattern.FindStringSubmatch(description)
		if len(matches) >= vulnRegexMatchCount3 {
			return "procurve-" + matches[1], matches[2]
		}
	}

	// Juniper pattern: "Juniper Networks, Inc. ex2200-48t-4g Ethernet Switch, kernel JUNOS 12.3R12.4"
	if strings.Contains(description, "juniper") || strings.Contains(description, "junos") {
		pattern := regexp.MustCompile(`junos\s+([\d.R]+)`)
		matches := pattern.FindStringSubmatch(description)
		if len(matches) >= vulnRegexMatchCount2 {
			return "junos", matches[1]
		}
	}

	// Generic version pattern
	versionPattern := regexp.MustCompile(`version\s+([\d.]+)`)
	matches := versionPattern.FindStringSubmatch(description)
	if len(matches) >= vulnRegexMatchCount2 {
		return "network-device", matches[1]
	}

	return "", ""
}

// parseServiceBanner extracts product and version from service banners
// Example: "OpenSSH_8.9p1 Ubuntu-3ubuntu0.1" → "openssh", "8.9p1".
func parseServiceBanner(banner string) (string, string) {
	if banner == "" {
		return "", ""
	}

	// Pattern: ProductName_X.Y[pN] or ProductName/X.Y
	pattern := regexp.MustCompile(`^(\w+)[_/]([\d.]+\w*)`)
	matches := pattern.FindStringSubmatch(banner)
	if len(matches) >= vulnRegexMatchCount3 {
		product := strings.ToLower(matches[1])
		version := matches[2]
		return product, version
	}

	// Alternative: "Product X.Y.Z"
	pattern2 := regexp.MustCompile(`^(\w+)\s+([\d.]+)`)
	matches2 := pattern2.FindStringSubmatch(banner)
	if len(matches2) >= vulnRegexMatchCount3 {
		product := strings.ToLower(matches2[1])
		version := matches2[2]
		return product, version
	}

	return "", ""
}

// parseHTTPServer extracts product and version from HTTP Server header
// Example: "nginx/1.18.0" → "nginx", "1.18.0".
func parseHTTPServer(server string) (string, string) {
	if server == "" {
		return "", ""
	}

	// Pattern: Product/X.Y.Z
	pattern := regexp.MustCompile(`^([\w-]+)/([\d.]+)`)
	matches := pattern.FindStringSubmatch(server)
	if len(matches) >= vulnRegexMatchCount3 {
		product := strings.ToLower(matches[1])
		version := matches[2]
		return product, version
	}

	// Handle "Microsoft-IIS/10.0"
	if after, ok := strings.CutPrefix(server, "Microsoft-IIS/"); ok {
		return "microsoft-iis", after
	}

	// Handle "Apache/2.4.41 (Ubuntu)"
	if strings.HasPrefix(server, "Apache/") {
		parts := strings.Fields(server)
		if len(parts) > 0 {
			version := strings.TrimPrefix(parts[0], "Apache/")
			return "apache", version
		}
	}

	return "", ""
}

// parseOSGuess extracts product and version from OS fingerprint guess
// Example: "Linux 2.6.x" → "linux-kernel", "2.6".
func parseOSGuess(osGuess string) (string, string) {
	if osGuess == "" {
		return "", ""
	}

	osGuess = strings.ToLower(osGuess)

	// Linux pattern
	if strings.Contains(osGuess, "linux") {
		pattern := regexp.MustCompile(`linux\s+([\d.]+)`)
		matches := pattern.FindStringSubmatch(osGuess)
		if len(matches) >= vulnRegexMatchCount2 {
			version := strings.TrimSuffix(matches[1], ".x")
			return "linux-kernel", version
		}
		return "linux-kernel", ""
	}

	// Windows pattern
	if strings.Contains(osGuess, "windows") {
		pattern := regexp.MustCompile(`windows\s+(\w+\s*[\d.]*)`)
		matches := pattern.FindStringSubmatch(osGuess)
		if len(matches) >= vulnRegexMatchCount2 {
			return "microsoft-windows", strings.TrimSpace(matches[1])
		}
		return "microsoft-windows", ""
	}

	return "", ""
}
