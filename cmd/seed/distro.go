package main

import (
	"os"
	"strings"
)

const (
	distroFamilyDebian = "debian"
	// expectedLinuxReleaseParts is the number of parts expected when splitting key=value lines
	// in /etc/os-release (key and value).
	expectedLinuxReleaseParts = 2
)

// Distro represents a Linux distribution.
type Distro struct {
	ID      string // ubuntu, fedora, rhel, debian, arch, etc.
	Name    string // Human-readable name
	Version string // Version string
	Family  string // debian, rhel, arch
}

// DetectDistro detects the current Linux distribution.
func DetectDistro() *Distro {
	// Try /etc/os-release first (standard on modern distros)
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		return parseOSRelease(string(data))
	}

	// Fallback detection for older systems
	return detectFromLegacyFiles()
}

func parseOSRelease(content string) *Distro {
	d := &Distro{}
	for line := range strings.SplitSeq(content, "\n") {
		parts := strings.SplitN(line, "=", expectedLinuxReleaseParts)
		if len(parts) != expectedLinuxReleaseParts {
			continue
		}
		key := parts[0]
		value := strings.Trim(parts[1], `"'`)

		switch key {
		case "ID":
			d.ID = value
		case "NAME":
			d.Name = value
		case "VERSION_ID":
			d.Version = value
		case "ID_LIKE":
			d.Family = value
		}
	}

	// Set family if not set
	if d.Family == "" {
		switch d.ID {
		case "ubuntu", "debian", "linuxmint", "pop":
			d.Family = distroFamilyDebian
		case "fedora", "rhel", "centos", "rocky", "almalinux":
			d.Family = "rhel"
		case "arch", "manjaro", "endeavouros":
			d.Family = "arch"
		default:
			d.Family = d.ID
		}
	}

	return d
}

func detectFromLegacyFiles() *Distro {
	// Check for distro-specific files
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		return &Distro{ID: distroFamilyDebian, Name: "Debian", Family: distroFamilyDebian}
	}
	if _, err := os.Stat("/etc/redhat-release"); err == nil {
		return &Distro{ID: "rhel", Name: "Red Hat", Family: "rhel"}
	}
	if _, err := os.Stat("/etc/arch-release"); err == nil {
		return &Distro{ID: "arch", Name: "Arch Linux", Family: "arch"}
	}
	return &Distro{ID: "unknown", Name: "Unknown Linux", Family: "unknown"}
}
