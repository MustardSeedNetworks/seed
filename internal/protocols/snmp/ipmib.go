package snmp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// IP-MIB OIDs (RFC 4293).
const (
	// OIDIpAdEntAddr is the IP-MIB OID for IP address (ipAddrTable, RFC 1213).
	OIDIpAdEntAddr = "1.3.6.1.2.1.4.20.1.1"
	// OIDIpAdEntIfIndex is the IP-MIB OID for interface index.
	OIDIpAdEntIfIndex = "1.3.6.1.2.1.4.20.1.2"
	// OIDIpAdEntNetMask is the IP-MIB OID for subnet mask.
	OIDIpAdEntNetMask = "1.3.6.1.2.1.4.20.1.3"

	// OIDIpAddressIfIndex is the IP-MIB OID for interface index (RFC 4293, IPv6 support).
	OIDIpAddressIfIndex = "1.3.6.1.2.1.4.34.1.3"
	// OIDIpAddressType is the IP-MIB OID for address type (unicast, broadcast, etc.).
	OIDIpAddressType = "1.3.6.1.2.1.4.34.1.4"
	// OIDIpAddressPrefix is the IP-MIB OID for address prefix.
	OIDIpAddressPrefix = "1.3.6.1.2.1.4.34.1.5"
	// OIDIpAddressOrigin is the IP-MIB OID for address origin (manual, dhcp, etc.).
	OIDIpAddressOrigin = "1.3.6.1.2.1.4.34.1.6"
	// OIDIpAddressStatus is the IP-MIB OID for address status.
	OIDIpAddressStatus = "1.3.6.1.2.1.4.34.1.7"
)

// IP address OID parsing constants.
const (
	// minOIDPartsIPAddrTable is the minimum OID parts for legacy ipAddrTable entries
	// (OID base + 4 IPv4 octets = 5 parts minimum).
	minOIDPartsIPAddrTable = 5
	// minOIDPartsIPAddressTable is the minimum OID parts for modern ipAddressTable entries
	// (OID base + type + length + address bytes = 6 parts minimum).
	minOIDPartsIPAddressTable = 6
	// ipv4OctetCount is the number of octets in an IPv4 address.
	ipv4OctetCount = 4
	// ipv6OctetCount is the number of octets in an IPv6 address.
	ipv6OctetCount = 16
	// ipv6GroupCount is the number of 16-bit groups in an IPv6 address.
	ipv6GroupCount = 8
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
func GetIPAddresses(
	ctx context.Context,
	ip string,
	cfg *config.SNMPConfig,
) ([]IPAddressEntry, error) {
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
func getIPAddrTable(
	ctx context.Context,
	ip string,
	cfg *config.SNMPConfig,
) ([]IPAddressEntry, error) {
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
func walkIPAddrTable(
	ctx context.Context,
	ip, community string,
	cfg *config.SNMPConfig,
) ([]IPAddressEntry, error) {
	params, err := newV2cWalkClient(ctx, ip, community, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkLegacyIPTable(params)
}

// walkIPAddrTableV3 walks the legacy ipAddrTable using SNMPv3.
func walkIPAddrTableV3(
	ctx context.Context,
	ip string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) ([]IPAddressEntry, error) {
	params, err := newV3WalkClient(ctx, ip, cred, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

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
		if len(parts) < minOIDPartsIPAddrTable {
			return nil
		}
		ipAddr := strings.Join(parts[len(parts)-ipv4OctetCount:], ".")

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
		logging.GetLogger().Debug("Failed to walk ipAdEntIfIndex", "error", walkErr)
	}

	// Walk ipAdEntNetMask to get subnet masks.
	walkErr = params.BulkWalk(OIDIpAdEntNetMask, func(pdu gosnmp.SnmpPDU) error {
		parts := strings.Split(pdu.Name, ".")
		if len(parts) < minOIDPartsIPAddrTable {
			return nil
		}
		ipAddr := strings.Join(parts[len(parts)-ipv4OctetCount:], ".")

		entry, exists := entries[ipAddr]
		if !exists {
			return nil
		}

		entry.NetMask = formatSNMPValue(pdu)
		entry.Prefix = netmaskToPrefix(entry.NetMask)
		return nil
	})
	if walkErr != nil {
		logging.GetLogger().Debug("Failed to walk ipAdEntNetMask", "error", walkErr)
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
func getIPAddressTable(
	ctx context.Context,
	ip string,
	cfg *config.SNMPConfig,
) ([]IPAddressEntry, error) {
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
func walkIPAddressTable(
	ctx context.Context,
	ip, community string,
	cfg *config.SNMPConfig,
) ([]IPAddressEntry, error) {
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
	walkIPAddressAttribute(
		params,
		OIDIpAddressType,
		entries,
		func(entry *IPAddressEntry, value string) {
			entry.Type = parseIPAddressType(value)
		},
	)

	// Walk ipAddressOrigin.
	walkIPAddressAttribute(
		params,
		OIDIpAddressOrigin,
		entries,
		func(entry *IPAddressEntry, value string) {
			entry.Origin = parseIPAddressOrigin(value)
		},
	)

	// Walk ipAddressStatus.
	walkIPAddressAttribute(
		params,
		OIDIpAddressStatus,
		entries,
		func(entry *IPAddressEntry, value string) {
			entry.Status = parseIPAddressStatus(value)
		},
	)

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
		logging.GetLogger().Debug("Failed to walk IP address attribute", "oid", oid, "error", err)
	}
}

// parseIPAddressFromOID extracts IP address from ipAddressTable OID.
// OID format: ...TYPE.LEN.ADDR_BYTES where TYPE is 1=ipv4, 2=ipv6.
func parseIPAddressFromOID(oid string) (string, string) {
	parts := strings.Split(oid, ".")
	if len(parts) < minOIDPartsIPAddressTable {
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

		if addrType == 1 && addrLen == ipv4OctetCount && i-3+ipv4OctetCount <= len(parts) {
			// IPv4: 4 octets.
			octets := parts[i-3 : i-3+ipv4OctetCount]
			ip := strings.Join(octets, ".")
			return ip, "ipv4"
		}

		if addrType == 2 && addrLen == ipv6OctetCount && i-3+ipv6OctetCount <= len(parts) {
			// IPv6: 16 octets.
			octets := parts[i-3 : i-3+ipv6OctetCount]
			ip := formatIPv6FromOctets(octets)
			return ip, "ipv6"
		}
	}

	return "", ""
}

// formatIPv6FromOctets formats IPv6 address from decimal octet strings.
func formatIPv6FromOctets(octets []string) string {
	if len(octets) != ipv6OctetCount {
		return ""
	}

	// Build IPv6 in standard format.
	groups := make([]string, ipv6GroupCount)
	for i := range ipv6GroupCount {
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
	if len(parts) != ipv4OctetCount {
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
