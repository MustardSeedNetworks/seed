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
// Example: Platform="cisco WS-C2960-24TT-L",
// Version="Cisco IOS Software, Version 12.2(58)SE2" → "cisco-ios", "12.2(58)SE2".
// A platform with no OS named in either field stays "cisco-device": CDP does not
// promise to say which OS is running, and guessing would file the device's CVEs
// under the wrong product.
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

	// Shared extractor rather than a local pattern: the one that used to live
	// here could not cross a second dot, so "16.9.4" was reported as "16.9" and
	// looked up against the wrong release's CVEs.
	if version := extractVersion(softwareVersion); version != "" {
		return product, version
	}

	return product, softwareVersion
}

// parseEDPVersion extracts product and version from Extreme EDP info
// Example: "ExtremeXOS 22.5.1.7" → "extremexos", "22.5.1.7".
func parseEDPVersion(softwareVersion string) (string, string) {
	if softwareVersion == "" {
		return "", ""
	}

	// Product name first, version from the shared extractor.
	pattern := regexp.MustCompile(`^(\w+)\s`)
	matches := pattern.FindStringSubmatch(softwareVersion)
	if version := extractVersion(softwareVersion); version != "" &&
		len(matches) >= vulnRegexMatchCount2 {
		return strings.ToLower(matches[1]), version
	}

	return "extreme-device", softwareVersion
}

// parseSNMPSysDescr extracts product and version from SNMP sysDescr
// Example formats:
// - "Cisco IOS Software, Version 15.2(4)M" → "cisco-ios", "15.2(4)m".
// - "HP J9729A 2920-48G Switch, revision WB.16.04.0008" → "hp-2920", "wb.16.04.0008".
// Output is lowercased: sysDescr is folded before matching, and CPE lookups
// are case-insensitive.
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
			// (\d+)[\w-]* rather than (\d+\w*): \w does not cross the hyphen in
			// "2920-48g", so the engine backtracked and captured "48g" — the port
			// count — as the model number.
			regexp.MustCompile(`(\d+)[\w-]*\s+switch.*?revision\s+([\w.]+)`),
			"hp-switch",
			2,
			"hp-",
		},
		{
			[]string{"aruba"},
			regexp.MustCompile(`(\d+)[\w-]*\s+switch.*?revision\s+([\w.]+)`),
			"hp-switch",
			2,
			"hp-",
		},
		{
			[]string{"juniper", "junos"},
			// parseSNMPSysDescr lowercases before matching, so an uppercase X in
			// the class never matched and "12.1X47-D15.4" was truncated to "12.1".
			regexp.MustCompile(`junos\s+(\d[\w.-]*)`),
			"juniper-junos",
			1,
			"",
		},
	}
}

// matchVendorKeywords checks if all keywords match in the sysDescr.
//
// Word-delimited, not substring: [strings.Contains] matched "hp" inside
// "ICX6430-24-HPOE", so every Brocade switch in the corpus was reported as an
// HP switch and looked up against the wrong vendor's CVEs.
func matchVendorKeywords(sysDescr string, keywords []string) bool {
	for _, kw := range keywords {
		if !containsWord(sysDescr, kw) {
			return false
		}
	}
	return true
}

// extractVendorProductVersion names the product from a matcher's model capture
// and takes the version from the shared extractor.
//
// The matchers used to carry their own version capture group, which is where
// the per-vendor version bugs lived — an uppercase X against lowercased input,
// a class that could not cross a second dot, a pattern that only fired on the
// one keyword its author expected. The model is vendor-shaped and stays here;
// the version is not, and does not.
func extractVendorProductVersion(m snmpVendorMatcher, matches []string, sysDescr string) (string, string) {
	productName := m.product
	if m.productPrefix != "" && len(matches) > 1 {
		productName = m.productPrefix + matches[1]
	}
	return productName, extractVersion(sysDescr)
}

// matchSNMPVendor attempts to match sysDescr against vendor-specific patterns.
// Returns product, version, and whether a match was found.
func matchSNMPVendor(sysDescr string) (string, string, bool) {
	for _, m := range getSNMPVendorMatchers() {
		if !matchVendorKeywords(sysDescr, m.keywords) {
			continue
		}
		matches := m.pattern.FindStringSubmatch(sysDescr)
		product, version := extractVendorProductVersion(m, matches, sysDescr)
		return product, version, true
	}
	return "", "", false
}

// extractGenericVendorVersion extracts version from sysDescr for known generic vendors.
// Returns vendor, version, and whether a match was found.
func extractGenericVendorVersion(sysDescr string) (string, string, bool) {
	// Every entry below appears in testdata/sysdescr-corpus.txt. This list is
	// data-driven on purpose: adding vendors nobody has actually seen is how a
	// table like this turns into speculation nobody can verify.
	//
	// Order matters where one name contains another — "extremexos" must be
	// tried before "extreme", or an ExtremeXOS switch is reported as the
	// less specific product.
	vendors := []string{
		"extremexos", "arubaos-cx", "arubaos", "pan-os", "palo alto",
		"fortiswitch", "fortigate", "fortios", "fortinet",
		"cisco", "juniper", "arista", "dell", "mikrotik", "routeros",
		"brocade", "extreme", "netgear", "ubiquiti", "zte", "alcatel-lucent",
		"huawei", "vmware", "3com", "force10", "h3c", "linux",
	}

	for _, vendor := range vendors {
		if !containsWord(sysDescr, vendor) {
			continue
		}
		return vendor, extractVersion(sysDescr), true
	}
	return "", "", false
}

// - "Juniper ... kernel JUNOS 12.1X47-D15.4" → "juniper-junos", "12.1x47-d15.4".
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
// Example: "ProCurve J9850A Switch 2910al-48G, revision WB.16.04" → "procurve-2910", "wb.16.04".
func parseLLDPDescription(description string) (string, string) {
	if description == "" {
		return "", ""
	}

	description = strings.ToLower(description)

	version := extractVersion(description)

	// HP/Aruba ProCurve: the model number identifies the product.
	if containsWord(description, "procurve") {
		pattern := regexp.MustCompile(`procurve.*?(\d+)[\w-]*,`)
		if matches := pattern.FindStringSubmatch(description); len(matches) >= vulnRegexMatchCount2 {
			return "procurve-" + matches[1], version
		}
	}

	if containsWord(description, "juniper") || containsWord(description, "junos") {
		return "junos", version
	}

	if version != "" {
		return "network-device", version
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
			// TrimRight, not TrimSuffix(".x"): [\d.]+ already consumed the dot
			// and stopped before the x, so the capture is "2.6." not "2.6.x".
			version := strings.TrimRight(matches[1], ".")
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
