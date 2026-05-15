package main

import (
	"testing"
)

func TestParseOSRelease(t *testing.T) {

	tests := []struct {
		name     string
		content  string
		expected Distro
	}{
		{
			name: "Ubuntu 22.04",
			content: `NAME="Ubuntu"
VERSION_ID="22.04"
ID=ubuntu
ID_LIKE=debian
PRETTY_NAME="Ubuntu 22.04.3 LTS"
`,
			expected: Distro{
				ID:      "ubuntu",
				Name:    "Ubuntu",
				Version: "22.04",
				Family:  "debian",
			},
		},
		{
			name: "Fedora 39",
			content: `NAME="Fedora Linux"
VERSION_ID="39"
ID=fedora
ID_LIKE="rhel centos"
PRETTY_NAME="Fedora Linux 39 (Workstation Edition)"
`,
			expected: Distro{
				ID:      "fedora",
				Name:    "Fedora Linux",
				Version: "39",
				Family:  "rhel centos",
			},
		},
		{
			name: "Arch Linux",
			content: `NAME="Arch Linux"
ID=arch
PRETTY_NAME="Arch Linux"
`,
			expected: Distro{
				ID:     "arch",
				Name:   "Arch Linux",
				Family: "arch",
			},
		},
		{
			name: "Debian 12",
			content: `NAME="Debian GNU/Linux"
VERSION_ID="12"
ID=debian
PRETTY_NAME="Debian GNU/Linux 12 (bookworm)"
`,
			expected: Distro{
				ID:      "debian",
				Name:    "Debian GNU/Linux",
				Version: "12",
				Family:  "debian",
			},
		},
		{
			name: "Rocky Linux",
			content: `NAME="Rocky Linux"
VERSION_ID="9.3"
ID=rocky
ID_LIKE="rhel centos fedora"
PRETTY_NAME="Rocky Linux 9.3 (Blue Onyx)"
`,
			expected: Distro{
				ID:      "rocky",
				Name:    "Rocky Linux",
				Version: "9.3",
				Family:  "rhel centos fedora",
			},
		},
		{
			name: "Linux Mint",
			content: `NAME="Linux Mint"
VERSION_ID="21.2"
ID=linuxmint
PRETTY_NAME="Linux Mint 21.2"
`,
			expected: Distro{
				ID:      "linuxmint",
				Name:    "Linux Mint",
				Version: "21.2",
				Family:  "debian",
			},
		},
		{
			name: "Pop!_OS",
			content: `NAME="Pop!_OS"
VERSION_ID="22.04"
ID=pop
PRETTY_NAME="Pop!_OS 22.04 LTS"
`,
			expected: Distro{
				ID:      "pop",
				Name:    "Pop!_OS",
				Version: "22.04",
				Family:  "debian",
			},
		},
		{
			name: "CentOS",
			content: `NAME="CentOS Stream"
VERSION_ID="9"
ID=centos
ID_LIKE="rhel fedora"
PRETTY_NAME="CentOS Stream 9"
`,
			expected: Distro{
				ID:      "centos",
				Name:    "CentOS Stream",
				Version: "9",
				Family:  "rhel fedora",
			},
		},
		{
			name: "Manjaro",
			content: `NAME="Manjaro Linux"
ID=manjaro
PRETTY_NAME="Manjaro Linux"
`,
			expected: Distro{
				ID:     "manjaro",
				Name:   "Manjaro Linux",
				Family: "arch",
			},
		},
		{
			name:    "Empty content",
			content: ``,
			expected: Distro{
				Family: "",
			},
		},
		{
			name: "Malformed lines",
			content: `NAME
VERSION_ID=
ID=unknown
=value
INVALID
`,
			expected: Distro{
				ID:     "unknown",
				Family: "unknown",
			},
		},
		{
			name: "Quoted values with single quotes",
			content: `NAME='Test Distro'
VERSION_ID='1.0'
ID=test
`,
			expected: Distro{
				ID:      "test",
				Name:    "Test Distro",
				Version: "1.0",
				Family:  "test",
			},
		},
		{
			name: "RHEL",
			content: `NAME="Red Hat Enterprise Linux"
VERSION_ID="9.3"
ID=rhel
PRETTY_NAME="Red Hat Enterprise Linux 9.3"
`,
			expected: Distro{
				ID:      "rhel",
				Name:    "Red Hat Enterprise Linux",
				Version: "9.3",
				Family:  "rhel",
			},
		},
		{
			name: "EndeavourOS",
			content: `NAME="EndeavourOS"
ID=endeavouros
PRETTY_NAME="EndeavourOS Linux"
`,
			expected: Distro{
				ID:     "endeavouros",
				Name:   "EndeavourOS",
				Family: "arch",
			},
		},
		{
			name: "AlmaLinux",
			content: `NAME="AlmaLinux"
VERSION_ID="9.3"
ID=almalinux
PRETTY_NAME="AlmaLinux 9.3"
`,
			expected: Distro{
				ID:      "almalinux",
				Name:    "AlmaLinux",
				Version: "9.3",
				Family:  "rhel",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseOSRelease(tc.content)

			if result.ID != tc.expected.ID {
				t.Errorf("ID: got %q, want %q", result.ID, tc.expected.ID)
			}
			if result.Name != tc.expected.Name {
				t.Errorf("Name: got %q, want %q", result.Name, tc.expected.Name)
			}
			if result.Version != tc.expected.Version {
				t.Errorf("Version: got %q, want %q", result.Version, tc.expected.Version)
			}
			if result.Family != tc.expected.Family {
				t.Errorf("Family: got %q, want %q", result.Family, tc.expected.Family)
			}
		})
	}
}

func TestDistroStruct(t *testing.T) {

	distro := &Distro{
		ID:      "ubuntu",
		Name:    "Ubuntu",
		Version: "22.04",
		Family:  "debian",
	}

	if distro.ID != "ubuntu" {
		t.Errorf("Expected ID 'ubuntu', got %q", distro.ID)
	}
	if distro.Name != "Ubuntu" {
		t.Errorf("Expected Name 'Ubuntu', got %q", distro.Name)
	}
	if distro.Version != "22.04" {
		t.Errorf("Expected Version '22.04', got %q", distro.Version)
	}
	if distro.Family != "debian" {
		t.Errorf("Expected Family 'debian', got %q", distro.Family)
	}
}

func TestExpectedLinuxReleaseParts(t *testing.T) {

	// Verify the constant is set correctly for splitting key=value pairs
	if expectedLinuxReleaseParts != 2 {
		t.Errorf("Expected expectedLinuxReleaseParts to be 2, got %d", expectedLinuxReleaseParts)
	}
}
