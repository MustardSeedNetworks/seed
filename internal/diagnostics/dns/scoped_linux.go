// SPDX-License-Identifier: BUSL-1.1

//go:build linux

package dns

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// resolvedVarlinkSocket is systemd-resolved's varlink endpoint. Its
// DumpDNSConfiguration call needs no privilege and names every link's
// servers; systemd 259 no longer writes the per-link files below.
const resolvedVarlinkSocket = "/run/systemd/resolve/io.systemd.Resolve"

// dumpDNSConfigurationCall is the varlink request frame: a JSON object
// terminated by a NUL.
const dumpDNSConfigurationCall = `{"method":"io.systemd.Resolve.DumpDNSConfiguration","parameters":{}}` + "\x00"

// resolvedVarlinkTimeout bounds the whole exchange with systemd-resolved.
const resolvedVarlinkTimeout = 2 * time.Second

// resolvedNetifDir holds systemd-resolved's per-link state, one file per
// interface index, on the releases before 259. Its absence, with no varlink
// answer either, means this host does not run systemd-resolved and cannot say
// which interface a resolver belongs to.
const resolvedNetifDir = "/run/systemd/resolve/netif"

// interfaceResolversPlatform reads the resolvers systemd-resolved holds for
// iface. A host without systemd-resolved is not attributable: /etc/resolv.conf
// is one list for the whole machine, and splitting it per interface would be
// invention.
func interfaceResolversPlatform(iface string) ([]string, bool) {
	if reply, dumpErr := dumpResolvedDNSConfiguration(); dumpErr == nil {
		if servers, parseErr := parseDNSConfiguration(reply, iface); parseErr == nil {
			return servers, true
		}
	}

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

// dumpResolvedDNSConfiguration makes one varlink call to systemd-resolved and
// returns the reply frame without its terminating NUL.
func dumpResolvedDNSConfiguration() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), resolvedVarlinkTimeout)
	defer cancel()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", resolvedVarlinkSocket)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	if err = conn.SetDeadline(time.Now().Add(resolvedVarlinkTimeout)); err != nil {
		return nil, err
	}
	if _, err = conn.Write([]byte(dumpDNSConfigurationCall)); err != nil {
		return nil, err
	}
	reply, err := bufio.NewReader(conn).ReadBytes(0)
	if err != nil {
		return nil, err
	}
	return reply[:len(reply)-1], nil
}

// parseDNSConfiguration returns the servers systemd-resolved lists for iface
// in a DumpDNSConfiguration reply. A link the reply does not list, or lists
// without servers, has none. A varlink error reply (an older resolved without
// the call) is an error, so the caller can try the per-link files instead.
func parseDNSConfiguration(reply []byte, iface string) ([]string, error) {
	var msg struct {
		Error      string `json:"error"`
		Parameters struct {
			Configuration []struct {
				Ifname  string `json:"ifname"`
				Servers []struct {
					AddressString string `json:"addressString"`
				} `json:"servers"`
			} `json:"configuration"`
		} `json:"parameters"`
	}
	if err := json.Unmarshal(reply, &msg); err != nil {
		return nil, fmt.Errorf("decode resolved reply: %w", err)
	}
	if msg.Error != "" {
		return nil, fmt.Errorf("resolved: %s", msg.Error)
	}
	var servers []string
	for _, link := range msg.Parameters.Configuration {
		if link.Ifname != iface {
			continue
		}
		for _, srv := range link.Servers {
			if net.ParseIP(srv.AddressString) != nil {
				servers = append(servers, srv.AddressString)
			}
		}
	}
	return servers, nil
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
