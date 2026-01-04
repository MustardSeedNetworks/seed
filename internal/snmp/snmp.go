// Package snmp provides SNMP query functionality for network device discovery.
package snmp

import (
	"context"
	"errors"
	"fmt"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/logging"
)

// Standard System MIB OIDs.
const (
	OIDSysDescr    = "1.3.6.1.2.1.1.1.0" // System description
	OIDSysObjectID = "1.3.6.1.2.1.1.2.0" // System object ID
	OIDSysUpTime   = "1.3.6.1.2.1.1.3.0" // System uptime
	OIDSysContact  = "1.3.6.1.2.1.1.4.0" // System contact
	OIDSysName     = "1.3.6.1.2.1.1.5.0" // System name
	OIDSysLocation = "1.3.6.1.2.1.1.6.0" // System location
)

// Vendor-specific version OIDs.
const (
	OIDCiscoVersion   = "1.3.6.1.4.1.9.9.25.1.1.1.2"       // Cisco IOS version
	OIDHPVersion      = "1.3.6.1.4.1.11.2.14.11.5.1.1.2.0" // HP/Aruba version
	OIDJuniperVersion = "1.3.6.1.4.1.2636.3.1.2.0"         // Juniper JUNOS version
)

// AuthProtocolMD5 is the deprecated MD5 authentication protocol.
//
// Deprecated: MD5 is cryptographically broken and will be removed in the next major version.
// Use SHA256 or SHA512 instead for secure authentication.
const AuthProtocolMD5 = "MD5"

// SystemInfo contains standard SNMP system information.
type SystemInfo struct {
	SysDescr    string
	SysObjectID string
	SysName     string
	SysContact  string
	SysLocation string
	SysUpTime   uint32
}

// Query performs a single SNMP GET query.
// Security: SNMPv3 is preferred over v2c when both are configured.
func Query(ctx context.Context, ip, oid string, cfg *config.SNMPConfig) (string, error) {
	if cfg == nil {
		return "", errors.New("SNMP config is nil")
	}

	// Try SNMPv3 credentials first (more secure)
	for i := range cfg.V3Credentials {
		result, err := queryWithV3(ctx, ip, oid, &cfg.V3Credentials[i], cfg)
		if err == nil {
			return result, nil
		}
	}

	// Fall back to v2c community strings if v3 fails or not configured
	for _, community := range cfg.Communities {
		result, err := queryWithCommunity(ctx, ip, oid, community, cfg)
		if err == nil {
			return result, nil
		}
	}

	return "", errors.New("SNMP query failed for all configured credentials")
}

// QueryMultiple performs multiple SNMP GET queries in a single request.
// Security: SNMPv3 is preferred over v2c when both are configured.
func QueryMultiple(ctx context.Context, ip string, oids []string, cfg *config.SNMPConfig) (map[string]string, error) {
	if cfg == nil {
		return nil, errors.New("SNMP config is nil")
	}

	// Try SNMPv3 credentials first (more secure)
	for i := range cfg.V3Credentials {
		results, err := queryMultipleWithV3(ctx, ip, oids, &cfg.V3Credentials[i], cfg)
		if err == nil {
			return results, nil
		}
	}

	// Fall back to v2c community strings if v3 fails or not configured
	for _, community := range cfg.Communities {
		results, err := queryMultipleWithCommunity(ctx, ip, oids, community, cfg)
		if err == nil {
			return results, nil
		}
	}

	return nil, errors.New("SNMP query failed for all configured credentials")
}

// GetSystemInfo retrieves standard SNMP system information.
func GetSystemInfo(ctx context.Context, ip string, cfg *config.SNMPConfig) (*SystemInfo, error) {
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

	info := &SystemInfo{
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
	// Fixes #936: Check context cancellation before establishing connection
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	params := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      uint16(cfg.Port), // #nosec G115 -- Port validated by config (1-65535)
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   cfg.Timeout,
		Retries:   cfg.Retries,
	}

	err := params.Connect()
	if err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = params.Conn.Close() }()

	result, err := params.Get([]string{oid})
	if err != nil {
		return "", fmt.Errorf("SNMP GET failed: %w", err)
	}

	if len(result.Variables) == 0 {
		return "", errors.New("no variables returned")
	}

	return formatSNMPValue(result.Variables[0]), nil
}

// queryMultipleWithCommunity performs multiple SNMP queries with community string.
func queryMultipleWithCommunity(
	ctx context.Context,
	ip string,
	oids []string,
	community string,
	cfg *config.SNMPConfig,
) (map[string]string, error) {
	// Fixes #936: Check context cancellation before establishing connection
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	params := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      uint16(cfg.Port), // #nosec G115 -- Port validated by config (1-65535)
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   cfg.Timeout,
		Retries:   cfg.Retries,
	}

	err := params.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = params.Conn.Close() }()

	result, err := params.Get(oids)
	if err != nil {
		return nil, fmt.Errorf("SNMP GET failed: %w", err)
	}

	results := make(map[string]string)
	for _, variable := range result.Variables {
		// Fixes #897: Check bounds before removing leading dot to prevent panic
		name := variable.Name
		if len(name) > 0 && name[0] == '.' {
			name = name[1:]
		}
		results[name] = formatSNMPValue(variable)
	}

	return results, nil
}

// queryWithV3 performs SNMP v3 query with credentials.
func queryWithV3(
	ctx context.Context,
	ip, oid string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) (string, error) {
	// Fixes #943, #944: Check context cancellation before establishing connection
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Validate credentials to fail fast instead of silently (fixes #832)
	if cred.Username == "" {
		return "", fmt.Errorf("SNMPv3 credential '%s' has empty username", cred.Name)
	}
	if cred.AuthProtocol != "" && cred.AuthProtocol != "NoAuth" && cred.AuthPassword == "" {
		logging.GetLogger().Warn("SNMPv3 credential has auth protocol but empty password - authentication will likely fail",
			"credential_name", cred.Name,
			"auth_protocol", cred.AuthProtocol,
			"target", ip)
	}
	if cred.PrivProtocol != "" && cred.PrivProtocol != "NoPriv" && cred.PrivPassword == "" {
		logging.GetLogger().Warn("SNMPv3 credential has privacy protocol but empty password - encryption will likely fail",
			"credential_name", cred.Name,
			"priv_protocol", cred.PrivProtocol,
			"target", ip)
	}

	// Warn if MD5 authentication is being used.
	// MD5 is cryptographically broken and will be removed in the next major version.
	if cred.AuthProtocol == AuthProtocolMD5 {
		logging.GetLogger().Warn("SNMP MD5 authentication is deprecated and will be removed in the next major version",
			"target", ip,
			"credential_name", cred.Name,
			"recommendation", "Use SHA256 or SHA512 for secure authentication")
	}

	params := &gosnmp.GoSNMP{
		Target:        ip,
		Port:          uint16(cfg.Port), // #nosec G115 -- Port validated by config (1-65535)
		Version:       gosnmp.Version3,
		Timeout:       cfg.Timeout,
		Retries:       cfg.Retries,
		SecurityModel: gosnmp.UserSecurityModel,
		MsgFlags:      gosnmp.AuthPriv,
		SecurityParameters: &gosnmp.UsmSecurityParameters{
			UserName: cred.Username,
			AuthenticationProtocol: getAuthProtocol(
				cred.AuthProtocol,
			),
			AuthenticationPassphrase: cred.AuthPassword,
			PrivacyProtocol:          getPrivProtocol(cred.PrivProtocol),
			PrivacyPassphrase:        cred.PrivPassword,
		},
	}

	err := params.Connect()
	if err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = params.Conn.Close() }()

	result, err := params.Get([]string{oid})
	if err != nil {
		return "", fmt.Errorf("SNMP GET failed: %w", err)
	}

	if len(result.Variables) == 0 {
		return "", errors.New("no variables returned")
	}

	return formatSNMPValue(result.Variables[0]), nil
}

// queryMultipleWithV3 performs multiple SNMP queries with v3 credentials.
func queryMultipleWithV3(
	ctx context.Context,
	ip string,
	oids []string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) (map[string]string, error) {
	// Fixes #943, #944: Check context cancellation before establishing connection
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Validate credentials to fail fast instead of silently (fixes #832)
	if cred.Username == "" {
		return nil, fmt.Errorf("SNMPv3 credential '%s' has empty username", cred.Name)
	}
	if cred.AuthProtocol != "" && cred.AuthProtocol != "NoAuth" && cred.AuthPassword == "" {
		logging.GetLogger().Warn("SNMPv3 credential has auth protocol but empty password - authentication will likely fail",
			"credential_name", cred.Name,
			"auth_protocol", cred.AuthProtocol,
			"target", ip)
	}
	if cred.PrivProtocol != "" && cred.PrivProtocol != "NoPriv" && cred.PrivPassword == "" {
		logging.GetLogger().Warn("SNMPv3 credential has privacy protocol but empty password - encryption will likely fail",
			"credential_name", cred.Name,
			"priv_protocol", cred.PrivProtocol,
			"target", ip)
	}

	// Warn if MD5 authentication is being used.
	// MD5 is cryptographically broken and will be removed in the next major version.
	if cred.AuthProtocol == AuthProtocolMD5 {
		logging.GetLogger().Warn("SNMP MD5 authentication is deprecated and will be removed in the next major version",
			"target", ip,
			"credential_name", cred.Name,
			"recommendation", "Use SHA256 or SHA512 for secure authentication")
	}

	params := &gosnmp.GoSNMP{
		Target:        ip,
		Port:          uint16(cfg.Port), // #nosec G115 -- Port validated by config (1-65535)
		Version:       gosnmp.Version3,
		Timeout:       cfg.Timeout,
		Retries:       cfg.Retries,
		SecurityModel: gosnmp.UserSecurityModel,
		MsgFlags:      gosnmp.AuthPriv,
		SecurityParameters: &gosnmp.UsmSecurityParameters{
			UserName: cred.Username,
			AuthenticationProtocol: getAuthProtocol(
				cred.AuthProtocol,
			),
			AuthenticationPassphrase: cred.AuthPassword,
			PrivacyProtocol:          getPrivProtocol(cred.PrivProtocol),
			PrivacyPassphrase:        cred.PrivPassword,
		},
	}

	err := params.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = params.Conn.Close() }()

	result, err := params.Get(oids)
	if err != nil {
		return nil, fmt.Errorf("SNMP GET failed: %w", err)
	}

	results := make(map[string]string)
	for _, variable := range result.Variables {
		// Fixes #897: Check bounds before removing leading dot to prevent panic
		name := variable.Name
		if len(name) > 0 && name[0] == '.' {
			name = name[1:]
		}
		results[name] = formatSNMPValue(variable)
	}

	return results, nil
}

// formatSNMPValue converts SNMP variable to string.
func formatSNMPValue(variable gosnmp.SnmpPDU) string {
	// Check for nil value to prevent panic on corrupted responses (fixes #850)
	if variable.Value == nil {
		return ""
	}
	//nolint:exhaustive // gosnmp.Asn1BER has many ASN.1 types, default handles uncommon ones
	switch variable.Type {
	case gosnmp.OctetString:
		bytes, ok := variable.Value.([]byte)
		if !ok {
			return fmt.Sprintf("%v", variable.Value)
		}
		return string(bytes)
	case gosnmp.Integer, gosnmp.Counter32, gosnmp.Gauge32, gosnmp.TimeTicks, gosnmp.Counter64:
		return fmt.Sprintf("%d", gosnmp.ToBigInt(variable.Value))
	case gosnmp.ObjectIdentifier:
		str, ok := variable.Value.(string)
		if !ok {
			return fmt.Sprintf("%v", variable.Value)
		}
		return str
	case gosnmp.IPAddress:
		str, ok := variable.Value.(string)
		if !ok {
			return fmt.Sprintf("%v", variable.Value)
		}
		return str
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

// getMaxRepetitions returns the MaxRepetitions value from config, defaulting to 10.
// This controls how many OID values are returned per GetBulk request.
// Lower values reduce memory usage and network load on slow devices.
func getMaxRepetitions(cfg *config.SNMPConfig) uint32 {
	if cfg == nil || cfg.MaxRepetitions == 0 {
		return 10 // Default value
	}
	if cfg.MaxRepetitions > 50 {
		return 50 // Cap at 50 to avoid overwhelming slow devices
	}
	return cfg.MaxRepetitions
}

// newV3WalkClient creates and connects an SNMPv3 client configured for walk operations.
// The caller is responsible for closing the connection: defer func() { _ = client.Conn.Close() }().
func newV3WalkClient(
	ctx context.Context,
	ip string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) (*gosnmp.GoSNMP, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

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

	return params, nil
}

// newV2cWalkClient creates and connects an SNMPv2c client configured for walk operations.
// The caller is responsible for closing the connection: defer func() { _ = client.Conn.Close() }().
func newV2cWalkClient(
	ctx context.Context,
	ip, community string,
	cfg *config.SNMPConfig,
) (*gosnmp.GoSNMP, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

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

	return params, nil
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

	return "", errors.New("no vendor-specific version found")
}
