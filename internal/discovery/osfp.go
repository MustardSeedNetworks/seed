package discovery

import (
	"regexp"
	"strings"
)

// fingerprintFromBanners analyzes port banners for OS information.
func (f *Fingerprinter) fingerprintFromBanners(profile *DeviceProfile, fp *OSFingerprint) {
	for _, port := range profile.OpenPorts {
		if port.Banner == "" {
			continue
		}
		bannerLower := strings.ToLower(port.Banner)
		if osInfo := f.parseOSFromBanner(bannerLower); osInfo != nil {
			f.updateFingerprintIfBetter(fp, osInfo, "banner")
		}
	}
}

// fingerprintFromHTTP analyzes HTTP Server header for OS information.
func (f *Fingerprinter) fingerprintFromHTTP(profile *DeviceProfile, fp *OSFingerprint) {
	if profile.HTTPInfo == nil || profile.HTTPInfo.Server == "" {
		return
	}
	serverLower := strings.ToLower(profile.HTTPInfo.Server)
	if osInfo := f.parseOSFromServer(serverLower); osInfo != nil {
		f.updateFingerprintIfBetter(fp, osInfo, "http")
	}
}

// updateFingerprintIfBetter updates the fingerprint if the new info has higher confidence.
func (*Fingerprinter) updateFingerprintIfBetter(fp, osInfo *OSFingerprint, method string) {
	fp.Methods = append(fp.Methods, method)
	if osInfo.Confidence > fp.Confidence {
		fp.OSFamily = osInfo.OSFamily
		fp.OSVersion = osInfo.OSVersion
		fp.Confidence = osInfo.Confidence
	}
}

// osMatch defines a pattern for OS detection.
type osMatch struct {
	patterns   []string // all patterns must match
	osFamily   string
	osVersion  string
	confidence int
}

// getSSHOSMatchers returns OS patterns for SSH banners.
func getSSHOSMatchers() []osMatch {
	return []osMatch{
		{[]string{"ubuntu"}, osLinux, "ubuntu", 90},
		{[]string{"debian"}, osLinux, "debian", 90},
		{[]string{"centos"}, osLinux, "rhel", 90},
		{[]string{"red hat"}, osLinux, "rhel", 90},
		{[]string{"freebsd"}, "bsd", "freebsd", 90},
		{[]string{"cisco"}, osCisco, "", 95},
		{[]string{"windows"}, osWindows, "", 90},
		{[]string{"openssh"}, "unix", "", 50},
	}
}

// getGenericOSMatchers returns OS patterns for generic banners.
func getGenericOSMatchers() []osMatch {
	return []osMatch{
		{[]string{"linux"}, osLinux, "", 80},
		{[]string{"windows"}, osWindows, "", 80},
		{[]string{"cisco"}, osCisco, "", 95},
		{[]string{"junos"}, "juniper", "", 95},
		{[]string{"vsftpd"}, osLinux, "", 75},
		{[]string{"proftpd"}, osLinux, "", 75},
		{[]string{"microsoft", "ftp"}, osWindows, "", 85},
	}
}

// parseOSFromBanner extracts OS info from service banners.
func (f *Fingerprinter) parseOSFromBanner(banner string) *OSFingerprint {
	fp := &OSFingerprint{}

	if strings.Contains(banner, "ssh") {
		f.matchOSPatterns(banner, getSSHOSMatchers(), fp)
	}
	if fp.OSFamily == "" {
		f.matchOSPatterns(banner, getGenericOSMatchers(), fp)
	}

	if fp.OSFamily == "" {
		return nil
	}
	return fp
}

// matchOSPatterns checks banner against patterns and sets fingerprint if matched.
func (*Fingerprinter) matchOSPatterns(banner string, matchers []osMatch, fp *OSFingerprint) {
	for _, m := range matchers {
		matched := true
		for _, p := range m.patterns {
			if !strings.Contains(banner, p) {
				matched = false
				break
			}
		}
		if matched {
			fp.OSFamily = m.osFamily
			fp.OSVersion = m.osVersion
			fp.Confidence = m.confidence
			return
		}
	}
}

// getServerOSMatchers returns OS patterns for HTTP Server headers.
func getServerOSMatchers() []osMatch {
	return []osMatch{
		{[]string{"ubuntu"}, osLinux, "ubuntu", 85},
		{[]string{"debian"}, osLinux, "debian", 85},
		{[]string{"centos"}, osLinux, "rhel", 85},
		{[]string{"red hat"}, osLinux, "rhel", 85},
		{[]string{"cisco"}, osCisco, "", 90},
		{
			[]string{"routeros"},
			"mikrotik",
			"",
			95,
		}, //nolint:misspell // RouterOS is MikroTik's product name
		{[]string{"fortinet"}, "fortinet", "", 95},
		{[]string{"fortigate"}, "fortinet", "", 95},
		{[]string{"pfsense"}, "bsd", "firewall", 90},
		{[]string{"opnsense"}, "bsd", "firewall", 90},
		{[]string{"synology"}, osLinux, "dsm", 95},
		{[]string{"qnap"}, osLinux, "qts", 95},
	}
}

// parseOSFromServer extracts OS info from HTTP Server header.
func (f *Fingerprinter) parseOSFromServer(server string) *OSFingerprint {
	fp := &OSFingerprint{}

	// Windows indicators (special case for IIS version extraction)
	if strings.Contains(server, "microsoft") || strings.Contains(server, "iis") {
		fp.OSFamily = osWindows
		if match := regexp.MustCompile(`iis[/\s]*([\d.]+)`).FindStringSubmatch(server); len(
			match,
		) > 1 {
			fp.OSVersion = "IIS " + match[1]
		}
		fp.Confidence = 85
		return fp
	}

	// Try pattern matching
	f.matchOSPatterns(server, getServerOSMatchers(), fp)

	// Fallback for generic web servers
	if fp.OSFamily == "" &&
		(strings.Contains(server, "lighttpd") || strings.Contains(server, "nginx")) {
		fp.OSFamily = "unix"
		fp.Confidence = 50
	}

	if fp.OSFamily == "" {
		return nil
	}
	return fp
}
