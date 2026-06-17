package discovery

import (
	"regexp"
	"strings"
)

// detectServiceVersion analyzes a port's banner to determine service version.
func (f *Fingerprinter) detectServiceVersion(port OpenPort) *ServiceVersion {
	if port.Banner == "" && port.Service == "" {
		return nil
	}

	sv := &ServiceVersion{
		Port:       port.Port,
		Service:    port.Service,
		Confidence: defaultServiceConfidence,
	}

	if port.Banner == "" {
		return sv
	}

	bannerLower := strings.ToLower(port.Banner)
	f.detectSSHVersion(port.Port, bannerLower, sv)
	f.detectFTPVersion(port.Port, bannerLower, sv)
	f.detectSMTPVersion(port.Port, bannerLower, sv)
	f.detectTelnetVersion(port.Port, bannerLower, sv)

	return sv
}

// detectSSHVersion detects SSH service version from banner.
func (*Fingerprinter) detectSSHVersion(port int, banner string, sv *ServiceVersion) {
	if port != 22 && !strings.Contains(banner, "ssh") {
		return
	}
	sv.Service = "ssh"
	if match := regexp.MustCompile(`openssh[_\s]*([\d.p]+)`).FindStringSubmatch(banner); len(
		match,
	) > 1 {
		sv.Product = "OpenSSH"
		sv.Version = match[1]
		sv.Confidence = 95
	} else if sshMatch := regexp.MustCompile(`ssh-([\d.]+)`).FindStringSubmatch(banner); len(sshMatch) > 1 {
		sv.Product = "SSH"
		sv.Version = sshMatch[1]
		sv.Confidence = 80
	}
}

// detectFTPVersion detects FTP service version from banner.
func (*Fingerprinter) detectFTPVersion(port int, banner string, sv *ServiceVersion) {
	if port != 21 && !strings.HasPrefix(banner, "220") {
		return
	}
	sv.Service = "ftp"
	switch {
	case strings.Contains(banner, "vsftpd"):
		sv.Product = "vsftpd"
		if match := regexp.MustCompile(`vsftpd\s*([\d.]+)`).FindStringSubmatch(banner); len(
			match,
		) > 1 {
			sv.Version = match[1]
		}
		sv.Confidence = 90
	case strings.Contains(banner, "proftpd"):
		sv.Product = "ProFTPD"
		if match := regexp.MustCompile(`proftpd\s*([\d.]+)`).FindStringSubmatch(banner); len(
			match,
		) > 1 {
			sv.Version = match[1]
		}
		sv.Confidence = 90
	case strings.Contains(banner, "pure-ftpd"):
		sv.Product = "Pure-FTPd"
		sv.Confidence = 90
	case strings.Contains(banner, "microsoft"):
		sv.Product = "Microsoft FTP"
		sv.Confidence = 85
	}
}

// detectSMTPVersion detects SMTP service version from banner.
func (*Fingerprinter) detectSMTPVersion(port int, banner string, sv *ServiceVersion) {
	if port != 25 && port != 587 {
		return
	}
	sv.Service = "smtp"
	switch {
	case strings.Contains(banner, "postfix"):
		sv.Product = "Postfix"
		sv.Confidence = 90
	case strings.Contains(banner, "sendmail"):
		sv.Product = "Sendmail"
		sv.Confidence = 90
	case strings.Contains(banner, "exim"):
		sv.Product = "Exim"
		if match := regexp.MustCompile(`exim\s*([\d.]+)`).FindStringSubmatch(banner); len(
			match,
		) > 1 {
			sv.Version = match[1]
		}
		sv.Confidence = 90
	case strings.Contains(banner, "microsoft"):
		sv.Product = "Microsoft Exchange"
		sv.Confidence = 85
	}
}

// detectTelnetVersion detects Telnet service version from banner.
func (*Fingerprinter) detectTelnetVersion(port int, banner string, sv *ServiceVersion) {
	if port != telnetPort {
		return
	}
	sv.Service = "telnet"
	switch {
	case strings.Contains(banner, "cisco"):
		sv.Product = productCiscoIOS
		sv.Confidence = 95
	case strings.Contains(banner, "linux"):
		sv.Product = "Linux telnetd"
		sv.Confidence = 80
	}
}
