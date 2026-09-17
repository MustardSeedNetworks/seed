// SPDX-License-Identifier: BUSL-1.1

//go:build darwin

package dns

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// scutilTimeout bounds the scutil read. It is a local query against the
// dynamic store and answers in milliseconds; the bound is there so a wedged
// call cannot hold a diagnostic request open.
const scutilTimeout = 5 * time.Second

// interfaceResolversPlatform reads the resolvers macOS has scoped to iface.
//
// The scoped configuration lives in the dynamic store, not in a file:
// /etc/resolv.conf is the unscoped answer, which is precisely the one that
// must not be attributed to an interface. scutil is how a non-cgo program
// reads it, and the package already shells out per interface elsewhere
// (internal/diagnostics/dhcp reads a lease with `ipconfig getpacket`).
func interfaceResolversPlatform(iface string) ([]string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), scutilTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "scutil", "--dns").Output()
	if err != nil {
		return nil, false
	}
	return parseScutilScopedResolvers(string(out), iface)
}

// parseScutilScopedResolvers returns the nameservers scutil reports for iface,
// and whether the output can attribute resolvers to an interface at all.
//
// Only the "for scoped queries" section is read. The unscoped section above it
// lists the resolvers the host uses by default, and answering with those for an
// interface that has none of its own is the same defect as naming another
// link's gateway (#2690) — an interface with no scoped resolver returns none.
//
// Output without that section cannot say which interface any resolver belongs
// to, which is a different answer from "this interface has none": reporting it
// as an absence would put every interface, the one carrying the route
// included, behind "no resolvers on this interface".
func parseScutilScopedResolvers(out, iface string) ([]string, bool) {
	const scopedHeader = "DNS configuration (for scoped queries)"

	_, scoped, found := strings.Cut(out, scopedHeader)
	if !found {
		return nil, false
	}

	var (
		servers []string
		block   []string
	)
	flush := func(blockIface string) {
		if blockIface == iface {
			servers = append(servers, block...)
		}
		block = nil
	}

	blockIface := ""
	for line := range strings.SplitSeq(scoped, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "resolver #"):
			flush(blockIface)
			blockIface = ""
		case strings.HasPrefix(line, "nameserver["):
			if addr := valueAfterColon(line); addr != "" {
				block = append(block, addr)
			}
		case strings.HasPrefix(line, "if_index"):
			blockIface = ifaceFromIfIndex(line)
		}
	}
	flush(blockIface)

	return servers, true
}

// valueAfterColon returns the value of a "key : value" scutil line.
func valueAfterColon(line string) string {
	_, value, found := strings.Cut(line, ":")
	if !found {
		return ""
	}
	return strings.TrimSpace(value)
}

// ifaceFromIfIndex reads the name out of "if_index : 12 (en0)". The index
// itself is not used: the name is what the operator selected.
func ifaceFromIfIndex(line string) string {
	open := strings.Index(line, "(")
	closeIdx := strings.LastIndex(line, ")")
	if open < 0 || closeIdx < open {
		return ""
	}
	return line[open+1 : closeIdx]
}
