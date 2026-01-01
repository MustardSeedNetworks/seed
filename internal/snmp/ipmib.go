// Package snmp provides SNMP query functionality for network device discovery.
// This file implements IP-MIB (RFC 4293) collection for IP address information.
package snmp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/seed/internal/config"
)

// IP-MIB OIDs (RFC 4293).
const (
	// ipAddrTable (deprecated but widely supported) - RFC 1213.
	OIDIpAdEntAddr    = "1.3.6.1.2.1.4.20.1.1" // ipAdEntAddr - IP address
	OIDIpAdEntIfIndex = "1.3.6.1.2.1.4.20.1.2" // ipAdEntIfIndex - interface index
	OIDIpAdEntNetMask = "1.3.6.1.2.1.4.20.1.3" // ipAdEntNetMask - subnet mask

	// ipAddressTable (RFC 4293) - modern, supports IPv6.
	OIDIpAddressIfIndex = "1.3.6.1.2.1.4.34.1.3" // ipAddressIfIndex
	OIDIpAddressType    = "1.3.6.1.2.1.4.34.1.4" // ipAddressType (unicast, broadcast, etc.)
	OIDIpAddressPrefix  = "1.3.6.1.2.1.4.34.1.5" // ipAddressPrefix
	OIDIpAddressOrigin  = "1.3.6.1.2.1.4.34.1.6" // ipAddressOrigin (manual, dhcp, etc.)
	OIDIpAddressStatus  = "1.3.6.1.2.1.4.34.1.7" // ipAddressStatus
)

// IPAddressEntry contains an IP address from IP-MIB.
type IPAddressEntry struct {
	Address   string // IP address (IPv4 or IPv6)
	IfIndex   int    // Interface index
	NetMask   string // Subnet mask (IPv4) or prefix length (IPv6)
	Prefix    int    // Prefix length calculated from netmask
	Type      string // unicast, broadcast, anycast
	Origin    string // manual, dhcp, linklayer, random
	Status    string // preferred, deprecated, invalid
	AddressIP string // "ipv4" or "ipv6"
}

// GetIPAddresses retrieves all IP addresses from a device using IP-MIB.
// It tries the modern ipAddressTable first, then falls back to legacy ipAddrTable.
func GetIPAddresses(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]IPAddressEntry, error) {
	if cfg == nil {
		return nil, errors.New("SNMP config is nil")
	}

	// Try modern ipAddressTable first (supports IPv6).
	entries, err := getIPAddressTable(ctx, ip, cfg)
	if err == nil && len(entries) > 0 {
		return entries, nil
	}

	// Fall back to legacy ipAddrTable (IPv4 only).
	return getIPAddrTable(ctx, ip, cfg)
}

// getIPAddrTable retrieves IP addresses from the legacy ipAddrTable (RFC 1213).
// This is widely supported but only provides IPv4 addresses.
// Security: SNMPv3 is preferred over v2c when both are configured.
func getIPAddrTable(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]IPAddressEntry, error) {
	// Try SNMPv3 credentials first (more secure).
	for i := range cfg.V3Credentials {
		entries, err := walkIPAddrTableV3(ctx, ip, &cfg.V3Credentials[i], cfg)
		if err == nil {
			return entries, nil
		}
	}

	// Fall back to v2c community strings if v3 fails or not configured.
	for _, community := range cfg.Communities {
		entries, err := walkIPAddrTable(ctx, ip, community, cfg)
		if err == nil {
			return entries, nil
		}
	}

	return nil, errors.New("failed to query ipAddrTable with all configured credentials")
}

// walkIPAddrTable walks the legacy ipAddrTable using SNMPv2c.
func walkIPAddrTable(ctx context.Context, ip, community string, cfg *config.SNMPConfig) ([]IPAddressEntry, error) {
	params := &gosnmp.GoSNMP{
		Target:         ip,
		Port:           uint16(cfg.Port), // #nosec G115 -- Port validated by config (1-65535)
		Community:      community,
		Version:        gosnmp.Version2c,
		Timeout:        cfg.Timeout,
		Retries:        cfg.Retries,
		MaxRepetitions: getMaxRepetitions(cfg),
	}

	if err := params.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = params.Conn.Close() }()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return walkLegacyIPTable(params)
}

// walkIPAddrTableV3 walks the legacy ipAddrTable using SNMPv3.
func walkIPAddrTableV3(
	ctx context.Context,
	ip string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) ([]IPAddressEntry, error) {
	params := &gosnmp.GoSNMP{
		Target:         ip,
		Port:           uint16(cfg.Port), // #nosec G115 -- Port validated by config (1-65535)
		Version:        gosnmp.Version3,
		Timeout:        cfg.Timeout,
		Retries:        cfg.Retries,
		MaxRepetitions: getMaxRepetitions(cfg),
		SecurityModel:  gosnmp.UserSecurityModel,
		MsgFlags:       gosnmp.AuthPriv,
		SecurityParameters: &gosnmp.UsmSecurityParameters{
			UserName:                 cred.Username,
			AuthenticationProtocol:   getAuthProtocol(cred.AuthProtocol),
			AuthenticationPassphrase: cred.AuthPassword,
			PrivacyProtocol:          getPrivProtocol(cred.PrivProtocol),
			PrivacyPassphrase:        cred.PrivPassword,
		},
	}

	if err := params.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = params.Conn.Close() }()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return walkLegacyIPTable(params)
}

// walkLegacyIPTable walks the legacy ipAddrTable.
func walkLegacyIPTable(params *gosnmp.GoSNMP) ([]IPAddressEntry, error) {
	entries := make(map[string]*IPAddressEntry)

	// Walk ipAdEntAddr to discover all IP addresses.
	err := params.BulkWalk(OIDIpAdEntAddr, func(pdu gosnmp.SnmpPDU) error {
		// OID format: .1.3.6.1.2.1.4.20.1.1.IP_OCTETS
		ipAddr := formatSNMPValue(pdu)
		if ipAddr == "" {
			return nil
		}

		entries[ipAddr] = &IPAddressEntry{
			Address:   ipAddr,
			AddressIP: "ipv4",
			Type:      "unicast",
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk ipAdEntAddr: %w", err)
	}

	// Walk ipAdEntIfIndex to get interface associations.
	walkErr := params.BulkWalk(OIDIpAdEntIfIndex, func(pdu gosnmp.SnmpPDU) error {
		parts := strings.Split(pdu.Name, ".")
		if len(parts) < 5 {
			return nil
		}
		ipAddr := strings.Join(parts[len(parts)-4:], ".")

		entry, exists := entries[ipAddr]
		if !exists {
			return nil
		}

		ifIndex, parseErr := strconv.Atoi(formatSNMPValue(pdu))
		if parseErr == nil {
			entry.IfIndex = ifIndex
		}
		return nil
	})
	if walkErr != nil {
		slog.Debug("Failed to walk ipAdEntIfIndex", "error", walkErr)
	}

	// Walk ipAdEntNetMask to get subnet masks.
	walkErr = params.BulkWalk(OIDIpAdEntNetMask, func(pdu gosnmp.SnmpPDU) error {
		parts := strings.Split(pdu.Name, ".")
		if len(parts) < 5 {
			return nil
		}
		ipAddr := strings.Join(parts[len(parts)-4:], ".")

		entry, exists := entries[ipAddr]
		if !exists {
			return nil
		}

		entry.NetMask = formatSNMPValue(pdu)
		entry.Prefix = netmaskToPrefix(entry.NetMask)
		return nil
	})
	if walkErr != nil {
		slog.Debug("Failed to walk ipAdEntNetMask", "error", walkErr)
	}

	// Convert map to slice.
	result := make([]IPAddressEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, *entry)
	}

	return result, nil
}

// getIPAddressTable retrieves IP addresses from the modern ipAddressTable (RFC 4293).
// This table supports both IPv4 and IPv6 addresses.
// Security: SNMPv3 is preferred over v2c when both are configured.
func getIPAddressTable(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]IPAddressEntry, error) {
	// Try SNMPv3 credentials first (more secure).
	for i := range cfg.V3Credentials {
		entries, err := walkIPAddressTableV3(ctx, ip, &cfg.V3Credentials[i], cfg)
		if err == nil {
			return entries, nil
		}
	}

	// Fall back to v2c community strings if v3 fails or not configured.
	for _, community := range cfg.Communities {
		entries, err := walkIPAddressTable(ctx, ip, community, cfg)
		if err == nil {
			return entries, nil
		}
	}

	return nil, errors.New("failed to query ipAddressTable with all configured credentials")
}

// walkIPAddressTable walks the modern ipAddressTable using SNMPv2c.
func walkIPAddressTable(ctx context.Context, ip, community string, cfg *config.SNMPConfig) ([]IPAddressEntry, error) {
	params := &gosnmp.GoSNMP{
		Target:         ip,
		Port:           uint16(cfg.Port), // #nosec G115 -- Port validated by config (1-65535)
		Community:      community,
		Version:        gosnmp.Version2c,
		Timeout:        cfg.Timeout,
		Retries:        cfg.Retries,
		MaxRepetitions: getMaxRepetitions(cfg),
	}

	if err := params.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = params.Conn.Close() }()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return walkModernIPTable(params)
}

// walkIPAddressTableV3 walks the modern ipAddressTable using SNMPv3.
func walkIPAddressTableV3(
	ctx context.Context,
	ip string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) ([]IPAddressEntry, error) {
	params := &gosnmp.GoSNMP{
		Target:         ip,
		Port:           uint16(cfg.Port), // #nosec G115 -- Port validated by config (1-65535)
		Version:        gosnmp.Version3,
		Timeout:        cfg.Timeout,
		Retries:        cfg.Retries,
		MaxRepetitions: getMaxRepetitions(cfg),
		SecurityModel:  gosnmp.UserSecurityModel,
		MsgFlags:       gosnmp.AuthPriv,
		SecurityParameters: &gosnmp.UsmSecurityParameters{
			UserName:                 cred.Username,
			AuthenticationProtocol:   getAuthProtocol(cred.AuthProtocol),
			AuthenticationPassphrase: cred.AuthPassword,
			PrivacyProtocol:          getPrivProtocol(cred.PrivProtocol),
			PrivacyPassphrase:        cred.PrivPassword,
		},
	}

	if err := params.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = params.Conn.Close() }()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return walkModernIPTable(params)
}

// walkModernIPTable walks the modern ipAddressTable.
func walkModernIPTable(params *gosnmp.GoSNMP) ([]IPAddressEntry, error) {
	entries := make(map[string]*IPAddressEntry)

	// Walk ipAddressIfIndex to discover all IP addresses with interface associations.
	err := params.BulkWalk(OIDIpAddressIfIndex, func(pdu gosnmp.SnmpPDU) error {
		// OID format: .1.3.6.1.2.1.4.34.1.3.TYPE.LEN.ADDR_BYTES
		ipAddr, addrType := parseIPAddressFromOID(pdu.Name)
		if ipAddr == "" {
			return nil
		}

		ifIndex, err := strconv.Atoi(formatSNMPValue(pdu))
		if err != nil {
			ifIndex = 0
		}

		entries[ipAddr] = &IPAddressEntry{
			Address:   ipAddr,
			IfIndex:   ifIndex,
			AddressIP: addrType,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk ipAddressIfIndex: %w", err)
	}

	// Walk ipAddressType.
	walkIPAddressAttribute(params, OIDIpAddressType, entries, func(entry *IPAddressEntry, value string) {
		entry.Type = parseIPAddressType(value)
	})

	// Walk ipAddressOrigin.
	walkIPAddressAttribute(params, OIDIpAddressOrigin, entries, func(entry *IPAddressEntry, value string) {
		entry.Origin = parseIPAddressOrigin(value)
	})

	// Walk ipAddressStatus.
	walkIPAddressAttribute(params, OIDIpAddressStatus, entries, func(entry *IPAddressEntry, value string) {
		entry.Status = parseIPAddressStatus(value)
	})

	// Convert map to slice.
	result := make([]IPAddressEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, *entry)
	}

	return result, nil
}

// walkIPAddressAttribute walks an IP address table attribute and applies a function.
func walkIPAddressAttribute(
	params *gosnmp.GoSNMP,
	oid string,
	entries map[string]*IPAddressEntry,
	updateFunc func(*IPAddressEntry, string),
) {
	err := params.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		ipAddr, _ := parseIPAddressFromOID(pdu.Name)
		if ipAddr == "" {
			return nil
		}

		entry, exists := entries[ipAddr]
		if !exists {
			return nil
		}

		updateFunc(entry, formatSNMPValue(pdu))
		return nil
	})
	if err != nil {
		slog.Debug("Failed to walk IP address attribute", "oid", oid, "error", err)
	}
}

// parseIPAddressFromOID extracts IP address from ipAddressTable OID.
// OID format: ...TYPE.LEN.ADDR_BYTES where TYPE is 1=ipv4, 2=ipv6.
func parseIPAddressFromOID(oid string) (string, string) {
	parts := strings.Split(oid, ".")
	if len(parts) < 6 {
		return "", ""
	}

	// Find the address type (1=ipv4, 2=ipv6).
	// The format is: ...column.addressType.addressLen.addr[0].addr[1]...
	// We need to find where the address starts.
	for i := len(parts) - 1; i >= 6; i-- {
		addrType, err := strconv.Atoi(parts[i-5])
		if err != nil || (addrType != 1 && addrType != 2) {
			continue
		}

		addrLen, err := strconv.Atoi(parts[i-4])
		if err != nil {
			continue
		}

		if addrType == 1 && addrLen == 4 && i-3+4 <= len(parts) {
			// IPv4: 4 octets.
			octets := parts[i-3 : i-3+4]
			ip := strings.Join(octets, ".")
			return ip, "ipv4"
		}

		if addrType == 2 && addrLen == 16 && i-3+16 <= len(parts) {
			// IPv6: 16 octets.
			octets := parts[i-3 : i-3+16]
			ip := formatIPv6FromOctets(octets)
			return ip, "ipv6"
		}
	}

	return "", ""
}

// formatIPv6FromOctets formats IPv6 address from decimal octet strings.
func formatIPv6FromOctets(octets []string) string {
	if len(octets) != 16 {
		return ""
	}

	// Build IPv6 in standard format.
	groups := make([]string, 8)
	for i := range 8 {
		high, err1 := strconv.Atoi(octets[i*2])
		low, err2 := strconv.Atoi(octets[i*2+1])
		if err1 != nil || err2 != nil {
			return ""
		}
		groups[i] = fmt.Sprintf("%02x%02x", high, low)
	}

	return strings.Join(groups, ":")
}

// netmaskToPrefix converts subnet mask to CIDR prefix length.
func netmaskToPrefix(mask string) int {
	parts := strings.Split(mask, ".")
	if len(parts) != 4 {
		return 0
	}

	prefix := 0
	for _, part := range parts {
		octet, err := strconv.Atoi(part)
		if err != nil {
			return 0
		}
		// Count bits in octet.
		for octet > 0 {
			prefix += octet & 1
			octet >>= 1
		}
	}

	return prefix
}

// parseIPAddressType converts ipAddressType value to string.
func parseIPAddressType(value string) string {
	switch value {
	case "1":
		return "unicast"
	case "2":
		return "anycast"
	case "3":
		return "broadcast"
	default:
		return StatusUnknown
	}
}

// parseIPAddressOrigin converts ipAddressOrigin value to string.
func parseIPAddressOrigin(value string) string {
	switch value {
	case "1":
		return MACTypeOther
	case "2":
		return "manual"
	case "4":
		return "dhcp"
	case "5":
		return "linklayer"
	case "6":
		return "random"
	default:
		return StatusUnknown
	}
}

// parseIPAddressStatus converts ipAddressStatus value to string.
func parseIPAddressStatus(value string) string {
	switch value {
	case "1":
		return "preferred"
	case "2":
		return "deprecated"
	case "3":
		return "invalid"
	case "4":
		return "inaccessible"
	case "5":
		return StatusUnknown
	case "6":
		return "tentative"
	case "7":
		return "duplicate"
	case "8":
		return "optimistic"
	default:
		return StatusUnknown
	}
}
