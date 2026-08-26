package snmp

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// InterfaceCounters contains interface traffic counters for bandwidth monitoring.
type InterfaceCounters struct {
	Index       int    // ifIndex
	InOctets    uint64 // Input bytes (64-bit preferred if available)
	OutOctets   uint64 // Output bytes (64-bit preferred if available)
	InErrors    uint64 // Input errors
	OutErrors   uint64 // Output errors
	InDiscards  uint64 // Input discards
	OutDiscards uint64 // Output discards
	Timestamp   int64  // Unix timestamp when counters were collected
}

// GetInterfaceCounters retrieves traffic counters for all interfaces.
// It prefers 64-bit counters (ifHCInOctets/ifHCOutOctets) when available.
func GetInterfaceCounters(
	ctx context.Context,
	ip string,
	cfg *config.SNMPConfig,
) (map[int]*InterfaceCounters, error) {
	if cfg == nil {
		return nil, errors.New("SNMP config is nil")
	}

	return sweepCredentials(ctx, cfg, "failed to query interface counters with all configured credentials",
		func(cred *config.SNMPv3Credential) (map[int]*InterfaceCounters, error) {
			return walkCountersV3(ctx, ip, cred, cfg)
		},
		func(community string) (map[int]*InterfaceCounters, error) {
			return walkCounters(ctx, ip, community, cfg)
		},
	)
}

// walkCounters walks interface counters using SNMPv2c.
func walkCounters(
	ctx context.Context,
	ip, community string,
	cfg *config.SNMPConfig,
) (map[int]*InterfaceCounters, error) {
	params, err := newV2cWalkClient(ctx, ip, community, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return collectCounters(params)
}

// walkCountersV3 walks interface counters using SNMPv3.
func walkCountersV3(
	ctx context.Context,
	ip string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) (map[int]*InterfaceCounters, error) {
	params, err := newV3WalkClient(ctx, ip, cred, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return collectCounters(params)
}

// collectCounters collects interface counters via SNMP bulk walk.
func collectCounters(params *gosnmp.GoSNMP) (map[int]*InterfaceCounters, error) {
	counters := make(map[int]*InterfaceCounters)
	timestamp := time.Now().Unix()

	// Helper to get or create counter entry.
	getCounter := func(index int) *InterfaceCounters {
		if c, ok := counters[index]; ok {
			return c
		}
		c := &InterfaceCounters{Index: index, Timestamp: timestamp}
		counters[index] = c
		return c
	}

	// Walk 64-bit counters first (preferred).
	walkCounter64(
		params,
		OIDIfHCInOctets,
		getCounter,
		func(c *InterfaceCounters, v uint64) { c.InOctets = v },
	)
	walkCounter64(
		params,
		OIDIfHCOutOctets,
		getCounter,
		func(c *InterfaceCounters, v uint64) { c.OutOctets = v },
	)

	// Walk 32-bit counters as fallback for devices without 64-bit support.
	walkCounter32Fallback(
		params,
		OIDIfInOctets,
		getCounter,
		func(c *InterfaceCounters) *uint64 { return &c.InOctets },
	)
	walkCounter32Fallback(
		params,
		OIDIfOutOctets,
		getCounter,
		func(c *InterfaceCounters) *uint64 { return &c.OutOctets },
	)

	// Walk error and discard counters.
	walkCounter32(
		params,
		OIDIfInErrors,
		getCounter,
		func(c *InterfaceCounters, v uint64) { c.InErrors = v },
	)
	walkCounter32(
		params,
		OIDIfOutErrors,
		getCounter,
		func(c *InterfaceCounters, v uint64) { c.OutErrors = v },
	)
	walkCounter32(
		params,
		OIDIfInDiscards,
		getCounter,
		func(c *InterfaceCounters, v uint64) { c.InDiscards = v },
	)
	walkCounter32(
		params,
		OIDIfOutDiscards,
		getCounter,
		func(c *InterfaceCounters, v uint64) { c.OutDiscards = v },
	)

	if len(counters) == 0 {
		return nil, errors.New("no interface counters retrieved")
	}

	return counters, nil
}

// walkCounter64 walks a 64-bit counter OID and applies the value using the setter.
func walkCounter64(
	params *gosnmp.GoSNMP,
	oid string,
	getCounter func(int) *InterfaceCounters,
	setter func(*InterfaceCounters, uint64),
) {
	_ = params.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		if index := extractIndexFromOID(pdu.Name); index > 0 {
			setter(getCounter(index), parseCounter64(pdu))
		}
		return nil
	})
}

// walkCounter32 walks a 32-bit counter OID and applies the value using the setter.
func walkCounter32(
	params *gosnmp.GoSNMP,
	oid string,
	getCounter func(int) *InterfaceCounters,
	setter func(*InterfaceCounters, uint64),
) {
	_ = params.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		if index := extractIndexFromOID(pdu.Name); index > 0 {
			setter(getCounter(index), parseCounter32(pdu))
		}
		return nil
	})
}

// walkCounter32Fallback walks a 32-bit counter OID but only sets value if the field is currently 0.
func walkCounter32Fallback(
	params *gosnmp.GoSNMP,
	oid string,
	getCounter func(int) *InterfaceCounters,
	fieldPtr func(*InterfaceCounters) *uint64,
) {
	_ = params.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		if index := extractIndexFromOID(pdu.Name); index > 0 {
			c := getCounter(index)
			if ptr := fieldPtr(c); *ptr == 0 {
				*ptr = parseCounter32(pdu)
			}
		}
		return nil
	})
}

// extractIndexFromOID extracts the interface index from an OID string.
func extractIndexFromOID(oid string) int {
	parts := strings.Split(oid, ".")
	if len(parts) < minOIDPartsForIndex {
		return 0
	}
	idx, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0
	}
	return idx
}

// parseCounter32 parses a 32-bit counter value from an SNMP PDU.
func parseCounter32(pdu gosnmp.SnmpPDU) uint64 {
	switch v := pdu.Value.(type) {
	case uint:
		return uint64(v)
	case uint32:
		return uint64(v)
	case int:
		if v >= 0 {
			return uint64(v)
		}
	case int64:
		if v >= 0 {
			return uint64(v)
		}
	}
	return 0
}

// parseCounter64 parses a 64-bit counter value from an SNMP PDU.
func parseCounter64(pdu gosnmp.SnmpPDU) uint64 {
	switch v := pdu.Value.(type) {
	case uint64:
		return v
	case uint:
		return uint64(v)
	case uint32:
		return uint64(v)
	case int64:
		if v >= 0 {
			return uint64(v)
		}
	case int:
		if v >= 0 {
			return uint64(v)
		}
	}
	return 0
}
