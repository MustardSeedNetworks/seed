// SPDX-License-Identifier: BUSL-1.1

//go:build darwin

package dhcp

/*
#cgo LDFLAGS: -framework CoreFoundation -framework SystemConfiguration
#include <stdlib.h>
#include "lease_darwin.h"
*/
import "C"

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

// The DHCP options a lease is reported from (RFC 2132).
const (
	optionSubnetMask = 1
	optionRouter     = 3
	optionDNSServers = 6
	optionDomainName = 15
	optionLeaseTime  = 51
	optionServerID   = 54

	leaseTimeLen = 4 // option 51 is a uint32 count of seconds
)

// errStoreUnreadable reports that the dynamic store could not be read, which
// is not the same answer as "this interface holds no lease".
var errStoreUnreadable = errors.New("the network configuration store cannot be read")

// storeLease is the lease configd holds for one interface: the address the
// service bound and the options of the ACK that granted it.
type storeLease struct {
	address string
	start   time.Time
	expiry  time.Time
	options map[int][]byte
}

// readStoreLease reads iface's lease in process, from the per-service DHCP
// entity configd keeps in the dynamic store. It used to fork `ipconfig
// getpacket` on every read, and any exec from the daemon can wedge its child
// on macOS (#2530); the gateway check reads this on every poll. A nil lease
// with a nil error means the interface holds none.
func readStoreLease(iface string) (*storeLease, error) {
	name := C.CString(iface)
	defer C.free(unsafe.Pointer(name))
	table := C.dhcp_lease(name)
	if table == nil {
		return nil, errStoreUnreadable
	}
	defer C.free(unsafe.Pointer(table))
	return parseStoreLease(C.GoString(table)), nil
}

// parseStoreLease reads the table dhcp_lease writes. An empty table is no
// lease.
func parseStoreLease(table string) *storeLease {
	if table == "" {
		return nil
	}
	lease := &storeLease{options: map[int][]byte{}}
	for line := range strings.SplitSeq(table, "\n") {
		key, value, _ := strings.Cut(line, "\t")
		switch key {
		case "address":
			lease.address = value
		case "start":
			lease.start = unixTime(value)
		case "expiry":
			lease.expiry = unixTime(value)
		case "option":
			code, data, _ := strings.Cut(value, "\t")
			n, codeErr := strconv.Atoi(code)
			raw, dataErr := hex.DecodeString(data)
			if codeErr == nil && dataErr == nil {
				lease.options[n] = raw
			}
		}
	}
	return lease
}

func unixTime(seconds string) time.Time {
	n, err := strconv.ParseInt(seconds, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(n, 0)
}

// addresses decodes an option holding one or more IPv4 addresses.
func (l *storeLease) ips(code int) []string {
	raw := l.options[code]
	var out []string
	for len(raw) >= ipv4Len {
		out = append(out, net.IP(raw[:ipv4Len]).String())
		raw = raw[ipv4Len:]
	}
	return out
}

func (l *storeLease) ip(code int) string {
	if addrs := l.ips(code); len(addrs) > 0 {
		return addrs[0]
	}
	return ""
}

// fill copies the lease into result. The server is the server identifier
// (option 54), the address the client answers; the BOOTP siaddr field the
// getpacket parser preferred names a boot server, and reads 0.0.0.0 on most
// networks.
func (l *storeLease) fill(result *TestResult) {
	result.OfferedIP = l.address
	result.ServerIP = l.ip(optionServerID)
	result.SubnetMask = l.ip(optionSubnetMask)
	result.Gateway = l.ip(optionRouter)
	result.DNSServers = l.ips(optionDNSServers)
	result.DomainName = strings.TrimRight(string(l.options[optionDomainName]), "\x00")
	if raw := l.options[optionLeaseTime]; len(raw) == leaseTimeLen {
		seconds := binary.BigEndian.Uint32(raw)
		result.LeaseTime = time.Duration(seconds) * time.Second
		result.LeaseTimeSec = int(seconds)
	}
}

// testDHCPPlatform reports the lease iface holds. The read is local and
// in process, so there is nothing for the context to bound.
func testDHCPPlatform(_ context.Context, interfaceName string) *TestResult {
	result := &TestResult{
		Interface: interfaceName,
		TestedAt:  time.Now(),
	}

	lease, err := readStoreLease(interfaceName)
	switch {
	case err != nil:
		result.Error = err.Error()
	case lease == nil:
		result.Error = "no DHCP lease found"
	default:
		lease.fill(result)
		result.Success = result.OfferedIP != ""
		if !result.Success {
			result.Error = "no DHCP lease found"
		}
	}
	return result
}

// getCurrentLeasePlatform retrieves the current DHCP lease on macOS.
func getCurrentLeasePlatform(interfaceName string) (*LeaseInfo, error) {
	lease := &LeaseInfo{
		Interface: interfaceName,
	}

	if err := populateLeaseFromInterface(interfaceName, lease); err != nil {
		return nil, err
	}

	stored, err := readStoreLease(interfaceName)
	if err != nil {
		return nil, err
	}
	if stored == nil {
		// No DHCP lease, but a static address is still worth reporting.
		if lease.IPAddress != "" {
			return lease, nil
		}
		return nil, &InterfaceError{Message: "no DHCP lease"}
	}

	result := &TestResult{}
	stored.fill(result)
	copyResultToLease(result, lease)
	lease.ObtainedAt = stored.start
	lease.Expiry = stored.expiry

	return lease, nil
}

// populateLeaseFromInterface fills lease with IP info from the network interface.
func populateLeaseFromInterface(interfaceName string, lease *LeaseInfo) error {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return &InterfaceError{Message: "interface not found: " + err.Error()}
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return &InterfaceError{Message: "failed to get addresses: " + err.Error()}
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			lease.IPAddress = ipnet.IP.String()
			lease.SubnetMask = net.IP(ipnet.Mask).String()
			break
		}
	}
	return nil
}

// copyResultToLease copies relevant fields from TestResult to LeaseInfo.
func copyResultToLease(result *TestResult, lease *LeaseInfo) {
	if result.ServerIP != "" {
		lease.ServerIP = result.ServerIP
	}
	if result.Gateway != "" {
		lease.Gateway = result.Gateway
	}
	if result.DNSServers != nil {
		lease.DNSServers = result.DNSServers
	}
	if result.DomainName != "" {
		lease.DomainName = result.DomainName
	}
	if result.LeaseTime > 0 {
		lease.LeaseTime = result.LeaseTime
		lease.LeaseTimeSec = int(result.LeaseTime.Seconds())
	}
}
