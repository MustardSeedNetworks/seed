//go:build darwin

package dns

import (
	"os"
	"path/filepath"
)

// getSystemDNSPlatform reads DNS servers on macOS from resolver config files.
// This reads the resolver configuration directly instead of calling scutil.
func getSystemDNSPlatform() []string {
	servers := []string{}
	seen := make(map[string]bool)

	// Read from /etc/resolv.conf first
	if s := parseResolvConf(resolvConfPath); len(s) > 0 {
		for _, server := range s {
			if !seen[server] {
				seen[server] = true
				servers = append(servers, server)
			}
		}
	}

	// Read from /etc/resolver/* for additional DNS configs
	resolverDir := "/etc/resolver"
	entries, err := os.ReadDir(resolverDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(resolverDir, entry.Name())
			for _, server := range parseResolvConf(path) {
				if !seen[server] {
					seen[server] = true
					servers = append(servers, server)
				}
			}
		}
	}

	return servers
}
