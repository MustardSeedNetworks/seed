// SPDX-License-Identifier: BUSL-1.1

//go:build linux

package dns

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// resolvedNetifDir holds systemd-resolved's per-link state, one file per
// interface index. Its absence means this host does not run systemd-resolved
// and cannot say which interface a resolver belongs to.
const resolvedNetifDir = "/run/systemd/resolve/netif"

// interfaceResolversPlatform reads the resolvers systemd-resolved holds for
// iface. A host without systemd-resolved is not attributable: /etc/resolv.conf
// is one list for the whole machine, and splitting it per interface would be
// invention.
func interfaceResolversPlatform(iface string) ([]string, bool) {
	if _, err := os.Stat(resolvedNetifDir); err != nil {
		return nil, false
	}

	link, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, true
	}

	content, err := os.ReadFile(filepath.Join(resolvedNetifDir, strconv.Itoa(link.Index)))
	if err != nil {
		// The directory exists, so resolved is running and simply holds no
		// state for this link: it has no resolvers of its own.
		return nil, true
	}

	return parseResolvedLinkDNS(string(content)), true
}

// parseResolvedLinkDNS reads the DNS entries of a systemd-resolved link file.
// Servers appear either one per DNS= line or space-separated on one; both
// spellings occur, so both are read.
func parseResolvedLinkDNS(content string) []string {
	var servers []string

	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		value, found := strings.CutPrefix(line, "DNS=")
		if !found {
			continue
		}
		for field := range strings.FieldsSeq(value) {
			if net.ParseIP(field) != nil {
				servers = append(servers, field)
			}
		}
	}

	return servers
}
