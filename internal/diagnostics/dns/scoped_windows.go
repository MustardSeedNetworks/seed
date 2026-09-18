// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package dns

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// interfaceResolversPlatform reads the resolvers Windows holds for iface.
//
// `ipconfig /all` is already sectioned per adapter — the system-wide reader
// merges those sections, which is what made every interface report the same
// resolvers. Go names a Windows interface by its friendly name, which is what
// the section header carries after "adapter ".
func interfaceResolversPlatform(iface string) ([]string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), dnsTimeoutSeconds*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ipconfig", "/all").Output()
	if err != nil {
		return nil, false
	}
	return parseIPConfigAdapterDNS(string(out), iface)
}

// parseIPConfigAdapterDNS returns the DNS servers ipconfig lists under the
// adapter named iface, and whether that adapter appears at all. An adapter the
// output does not name is reported as not attributable rather than as having
// no resolvers: the second would be a claim about the link, and this only
// failed to find it.
func parseIPConfigAdapterDNS(out, iface string) ([]string, bool) {
	var (
		servers  []string
		seen     = make(map[string]bool)
		inTarget bool
		inDNS    bool
		found    bool
	)

	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if name, ok := adapterSectionName(line); ok {
			inTarget = name == iface
			inDNS = false
			found = found || inTarget
			continue
		}
		if !inTarget {
			continue
		}

		if strings.Contains(line, "DNS Servers") {
			inDNS = true
			addDNSServerFromLabelledLine(line, &servers, seen)
			continue
		}
		if inDNS {
			inDNS = addDNSServerFromContinuationLine(line, &servers, seen)
		}
	}

	if !found {
		return nil, false
	}
	return servers, true
}

// adapterSectionName reads the adapter name out of a section header such as
// "Ethernet adapter Ethernet 2:". Headers start at column 0; the indented
// lines beneath them are the adapter's properties.
func adapterSectionName(line string) (string, bool) {
	if line == "" || line[0] == ' ' || line[0] == '\t' {
		return "", false
	}
	trimmed := strings.TrimSpace(line)
	if !strings.HasSuffix(trimmed, ":") {
		return "", false
	}
	_, name, found := strings.Cut(trimmed, " adapter ")
	if !found {
		return "", false
	}
	return strings.TrimSuffix(name, ":"), true
}
