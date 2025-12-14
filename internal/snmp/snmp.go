// Package snmp provides SNMP query functionality for network device discovery.
package snmp

import (
	"context"
	"fmt"

	"github.com/gosnmp/gosnmp"
	"github.com/krisarmstrong/luminetiq/internal/config"
)

// Standard System MIB OIDs
const (
	OIDSysDescr    = "1.3.6.1.2.1.1.1.0" // System description
	OIDSysObjectID = "1.3.6.1.2.1.1.2.0" // System object ID
	OIDSysUpTime   = "1.3.6.1.2.1.1.3.0" // System uptime
	OIDSysContact  = "1.3.6.1.2.1.1.4.0" // System contact
	OIDSysName     = "1.3.6.1.2.1.1.5.0" // System name
	OIDSysLocation = "1.3.6.1.2.1.1.6.0" // System location
)

// Vendor-specific version OIDs
const (
	OIDCiscoVersion  = "1.3.6.1.4.1.9.9.25.1.1.1.2"       // Cisco IOS version
	OIDHPVersion     = "1.3.6.1.4.1.11.2.14.11.5.1.1.2.0" // HP/Aruba version
	OIDJuniperVersion = "1.3.6.1.4.1.2636.3.1.2.0"        // Juniper JUNOS version
)

// SNMPSystemInfo contains standard SNMP system information.
type SNMPSystemInfo struct {
	SysDescr    string
	SysObjectID string
	SysName     string
	SysContact  string
	SysLocation string
	SysUpTime   uint32
}

// Query performs a single SNMP GET query.
func Query(ctx context.Context, ip, oid string, cfg *config.SNMPConfig) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("SNMP config is nil")
	}

	// Try each community string
	for _, community := range cfg.Communities {
		result, err := queryWithCommunity(ctx, ip, oid, community, cfg)
		if err == nil {
			return result, nil
		}
	}

	// Try SNMPv3 credentials
	for _, cred := range cfg.V3Credentials {
		result, err := queryWithV3(ctx, ip, oid, &cred, cfg)
		if err == nil {
			return result, nil
		}
	}

	return "", fmt.Errorf("SNMP query failed for all configured credentials")
}

// QueryMultiple performs multiple SNMP GET queries in a single request.
func QueryMultiple(ctx context.Context, ip string, oids []string, cfg *config.SNMPConfig) (map[string]string, error) {
	if cfg == nil {
		return nil, fmt.Errorf("SNMP config is nil")
	}

	// Try each community string
	for _, community := range cfg.Communities {
		results, err := queryMultipleWithCommunity(ctx, ip, oids, community, cfg)
		if err == nil {
			return results, nil
		}
	}

	// Try SNMPv3 credentials
	for _, cred := range cfg.V3Credentials {
		results, err := queryMultipleWithV3(ctx, ip, oids, &cred, cfg)
		if err == nil {
			return results, nil
		}
	}

	return nil, fmt.Errorf("SNMP query failed for all configured credentials")
}

// GetSystemInfo retrieves standard SNMP system information.
func GetSystemInfo(ctx context.Context, ip string, cfg *config.SNMPConfig) (*SNMPSystemInfo, error) {
	oids := []string{
		OIDSysDescr,
		OIDSysObjectID,
		OIDSysName,
		OIDSysContact,
		OIDSysLocation,
		OIDSysUpTime,
	}

	results, err := QueryMultiple(ctx, ip, oids, cfg)
	if err != nil {
		return nil, err
	}

	info := &SNMPSystemInfo{
		SysDescr:    results[OIDSysDescr],
		SysObjectID: results[OIDSysObjectID],
		SysName:     results[OIDSysName],
		SysContact:  results[OIDSysContact],
		SysLocation: results[OIDSysLocation],
	}

	return info, nil
}

// queryWithCommunity performs SNMP v1/v2c query with community string.
func queryWithCommunity(ctx context.Context, ip, oid, community string, cfg *config.SNMPConfig) (string, error) {
	params := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      uint16(cfg.Port),
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   cfg.Timeout,
		Retries:   cfg.Retries,
	}

	err := params.Connect()
	if err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}
	defer params.Conn.Close()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	result, err := params.Get([]string{oid})
	if err != nil {
		return "", fmt.Errorf("SNMP GET failed: %w", err)
	}

	if len(result.Variables) == 0 {
		return "", fmt.Errorf("no variables returned")
	}

	return formatSNMPValue(result.Variables[0]), nil
}

// queryMultipleWithCommunity performs multiple SNMP queries with community string.
func queryMultipleWithCommunity(ctx context.Context, ip string, oids []string, community string, cfg *config.SNMPConfig) (map[string]string, error) {
	params := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      uint16(cfg.Port),
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   cfg.Timeout,
		Retries:   cfg.Retries,
	}

	err := params.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer params.Conn.Close()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	result, err := params.Get(oids)
	if err != nil {
		return nil, fmt.Errorf("SNMP GET failed: %w", err)
	}

	results := make(map[string]string)
	for _, variable := range result.Variables {
		results[variable.Name[1:]] = formatSNMPValue(variable) // Remove leading dot
	}

	return results, nil
}

// queryWithV3 performs SNMP v3 query with credentials.
func queryWithV3(ctx context.Context, ip, oid string, cred *config.SNMPv3Credential, cfg *config.SNMPConfig) (string, error) {
	params := &gosnmp.GoSNMP{
		Target:  ip,
		Port:    uint16(cfg.Port),
		Version: gosnmp.Version3,
		Timeout: cfg.Timeout,
		Retries: cfg.Retries,
		SecurityModel: gosnmp.UserSecurityModel,
		MsgFlags:      gosnmp.AuthPriv,
		SecurityParameters: &gosnmp.UsmSecurityParameters{
			UserName:                 cred.Username,
			AuthenticationProtocol:   getAuthProtocol(cred.AuthProtocol),
			AuthenticationPassphrase: cred.AuthPassword,
			PrivacyProtocol:          getPrivProtocol(cred.PrivProtocol),
			PrivacyPassphrase:        cred.PrivPassword,
		},
	}

	err := params.Connect()
	if err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}
	defer params.Conn.Close()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	result, err := params.Get([]string{oid})
	if err != nil {
		return "", fmt.Errorf("SNMP GET failed: %w", err)
	}

	if len(result.Variables) == 0 {
		return "", fmt.Errorf("no variables returned")
	}

	return formatSNMPValue(result.Variables[0]), nil
}

// queryMultipleWithV3 performs multiple SNMP queries with v3 credentials.
func queryMultipleWithV3(ctx context.Context, ip string, oids []string, cred *config.SNMPv3Credential, cfg *config.SNMPConfig) (map[string]string, error) {
	params := &gosnmp.GoSNMP{
		Target:  ip,
		Port:    uint16(cfg.Port),
		Version: gosnmp.Version3,
		Timeout: cfg.Timeout,
		Retries: cfg.Retries,
		SecurityModel: gosnmp.UserSecurityModel,
		MsgFlags:      gosnmp.AuthPriv,
		SecurityParameters: &gosnmp.UsmSecurityParameters{
			UserName:                 cred.Username,
			AuthenticationProtocol:   getAuthProtocol(cred.AuthProtocol),
			AuthenticationPassphrase: cred.AuthPassword,
			PrivacyProtocol:          getPrivProtocol(cred.PrivProtocol),
			PrivacyPassphrase:        cred.PrivPassword,
		},
	}

	err := params.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer params.Conn.Close()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	result, err := params.Get(oids)
	if err != nil {
		return nil, fmt.Errorf("SNMP GET failed: %w", err)
	}

	results := make(map[string]string)
	for _, variable := range result.Variables {
		results[variable.Name[1:]] = formatSNMPValue(variable) // Remove leading dot
	}

	return results, nil
}

// formatSNMPValue converts SNMP variable to string.
func formatSNMPValue(variable gosnmp.SnmpPDU) string {
	switch variable.Type {
	case gosnmp.OctetString:
		bytes := variable.Value.([]byte)
		return string(bytes)
	case gosnmp.Integer, gosnmp.Counter32, gosnmp.Gauge32, gosnmp.TimeTicks, gosnmp.Counter64:
		return fmt.Sprintf("%d", gosnmp.ToBigInt(variable.Value))
	case gosnmp.ObjectIdentifier:
		return variable.Value.(string)
	case gosnmp.IPAddress:
		return variable.Value.(string)
	default:
		return fmt.Sprintf("%v", variable.Value)
	}
}

// getAuthProtocol converts auth protocol string to gosnmp type.
func getAuthProtocol(protocol string) gosnmp.SnmpV3AuthProtocol {
	switch protocol {
	case "MD5":
		return gosnmp.MD5
	case "SHA":
		return gosnmp.SHA
	case "SHA224":
		return gosnmp.SHA224
	case "SHA256":
		return gosnmp.SHA256
	case "SHA384":
		return gosnmp.SHA384
	case "SHA512":
		return gosnmp.SHA512
	default:
		return gosnmp.NoAuth
	}
}

// getPrivProtocol converts privacy protocol string to gosnmp type.
func getPrivProtocol(protocol string) gosnmp.SnmpV3PrivProtocol {
	switch protocol {
	case "DES":
		return gosnmp.DES
	case "AES":
		return gosnmp.AES
	case "AES192":
		return gosnmp.AES192
	case "AES256":
		return gosnmp.AES256
	case "AES192C":
		return gosnmp.AES192C
	case "AES256C":
		return gosnmp.AES256C
	default:
		return gosnmp.NoPriv
	}
}

// GetVendorVersion attempts to retrieve vendor-specific version information.
func GetVendorVersion(ctx context.Context, ip string, cfg *config.SNMPConfig) (string, error) {
	// Try Cisco
	version, err := Query(ctx, ip, OIDCiscoVersion, cfg)
	if err == nil && version != "" {
		return version, nil
	}

	// Try HP/Aruba
	version, err = Query(ctx, ip, OIDHPVersion, cfg)
	if err == nil && version != "" {
		return version, nil
	}

	// Try Juniper
	version, err = Query(ctx, ip, OIDJuniperVersion, cfg)
	if err == nil && version != "" {
		return version, nil
	}

	return "", fmt.Errorf("no vendor-specific version found")
}
