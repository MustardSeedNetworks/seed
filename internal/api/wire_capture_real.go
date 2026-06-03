//go:build cgo || windows

package api

import (
	"github.com/krisarmstrong/seed/internal/capture"
	"github.com/krisarmstrong/seed/internal/capture/pcap"
)

// defaultCaptureOpener returns the libpcap-backed capture adapter that powers
// live packet capture (LLDP/CDP/EDP discovery, DHCP sniffing, VLAN traffic).
//
// This file is compiled when CGO is enabled (libpcap is linkable on linux/darwin)
// or on Windows (where gopacket/pcap is CGO-free via wpcap), preserving live
// capture on every platform that supported it before the capture-port split. The
// CGO-free no-op variant for CGO_ENABLED=0 non-Windows builds lives in
// wire_capture_null.go. See docs/architecture/CGO_BUILD_STRATEGY.md.
func defaultCaptureOpener() capture.Opener { return pcap.New() }
