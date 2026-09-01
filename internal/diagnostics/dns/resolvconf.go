//go:build linux || darwin

package dns

import (
	"bufio"
	"os"
	"strings"
)

// nameserverFieldCount is the minimum number of fields in a valid nameserver
// line: the keyword and the address.
const nameserverFieldCount = 2

// resolvConfPath is the primary resolver configuration file.
const resolvConfPath = "/etc/resolv.conf"

// parseResolvConf reads nameserver entries from a resolv.conf-style file.
// A file that cannot be opened yields no servers rather than an error: an
// absent resolver file is a normal state, not a failure.
func parseResolvConf(path string) []string {
	servers := []string{}

	file, err := os.Open(path)
	if err != nil {
		return servers
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) >= nameserverFieldCount {
				servers = append(servers, parts[1])
			}
		}
	}

	return servers
}
