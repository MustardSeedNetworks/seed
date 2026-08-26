package config

import (
	"slices"
	"strconv"
	"strings"
)

// Port number bounds. Port 0 is reserved and never scannable.
const (
	minPortNumber = 1
	maxPortNumber = 65535
)

// ParsePortList expands a port-preset string into port numbers.
//
// The presets are written in the form operators recognise from nmap —
// comma-separated singles with inclusive ranges, "21,23,80,6000-6009" — because
// they are also surfaced in configuration. Callers that need to act on the
// ports rather than pass the string along need them expanded.
//
// Malformed entries are skipped rather than rejected. A preset is a constant in
// this package, so a bad entry is a typo by us, and dropping one port is a
// better failure than a scan that refuses to run at all. Out-of-range and
// reversed ranges are treated the same way.
//
// The result is sorted and deduplicated, so a caller can compare two lists
// without normalising first.
func ParsePortList(list string) []int {
	var ports []int

	for field := range strings.SplitSeq(list, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		low, high, isRange := strings.Cut(field, "-")
		if !isRange {
			if port, ok := parsePort(field); ok {
				ports = append(ports, port)
			}
			continue
		}

		start, startOK := parsePort(low)
		end, endOK := parsePort(high)
		if !startOK || !endOK || start > end {
			continue
		}
		for port := start; port <= end; port++ {
			ports = append(ports, port)
		}
	}

	slices.Sort(ports)
	return slices.Compact(ports)
}

// parsePort accepts a single port number within the valid range.
func parsePort(field string) (int, bool) {
	port, err := strconv.Atoi(strings.TrimSpace(field))
	if err != nil || port < minPortNumber || port > maxPortNumber {
		return 0, false
	}
	return port, true
}
