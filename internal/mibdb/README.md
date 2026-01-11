# MIB Database

The MIB database provides SNMP OID name-to-numeric resolution for the Seed application.
It is stored in SQLite and loaded on application startup with 918+ built-in OID definitions.

## Coverage Summary

**Total OIDs:** 918

### MIB Categories

| Category | Count | Description |
|----------|-------|-------------|
| MIB-II (RFC 1213) | 277 | Standard interface, IP, ICMP, TCP, UDP, SNMP objects |
| Layer 2 (CDP, LLDP, VLAN) | 94 | Cisco CDP, IEEE LLDP, 802.1Q VLAN objects |
| Other/Enterprise | 547 | Vendor-specific MIBs (Cisco, Synoptics, etc.) |

### Standard MIBs Included

- **SNMPv2-MIB** - System group (sysDescr, sysObjectID, sysUpTime, sysName, etc.)
- **IF-MIB** - Interface statistics (ifIndex, ifDescr, ifType, ifSpeed, etc.)
- **IP-MIB** - IP layer statistics (ipForwarding, ipInReceives, ipRouteTable, etc.)
- **TCP-MIB** - TCP statistics (tcpConnState, tcpConnLocalAddress, etc.)
- **UDP-MIB** - UDP statistics (udpInDatagrams, udpNoPorts, etc.)
- **ICMP** - ICMP statistics (icmpInMsgs, icmpOutMsgs, etc.)
- **RMON-MIB** - Remote monitoring (alarm, event, statistics tables)
- **ENTITY-MIB** - Physical inventory (entPhysicalDescr, entPhysicalClass, etc.)
- **LLDP-MIB** - IEEE 802.1AB neighbor discovery
- **Q-BRIDGE-MIB** - IEEE 802.1Q VLAN information

### Vendor-Specific MIBs

- **Cisco CDP** - Cisco Discovery Protocol (cdpCacheDeviceId, cdpCacheAddress, etc.)
- **Cisco Workgroup** - Catalyst switch MIBs (VLAN, port, spanning tree)
- **Synoptics** - Bay Networks/Nortel MIBs

## What's Missing

The following standard MIBs are NOT fully covered and may need to be added:

1. **BRIDGE-MIB (RFC 1493)** - Partial coverage; missing some dot1dTp* objects
2. **HOST-RESOURCES-MIB** - Not included; covers system resources (CPU, memory, disk)
3. **DISMAN-EVENT-MIB** - Not included; event management
4. **SNMP-TARGET-MIB** - Not included; SNMPv3 target configuration
5. **CISCO-VTP-MIB** - Partial; only basic VLAN objects
6. **CISCO-STACK-MIB** - Partial coverage
7. **IP-FORWARD-MIB** - Partial; need more inetCidr* objects for IPv6
8. **IPV6-MIB** - Not included; IPv6 specific objects

## Usage

The MIB database is automatically initialized on application startup:

```go
// In server initialization
mibDB := mibdb.New(db.Conn())
err := mibDB.LoadBuiltinOIDs()
```

### API Methods

```go
// Resolve an OID name to numeric form
numeric, err := mibDB.ResolveOIDName("sysDescr.0")
// Returns: "1.3.6.1.2.1.1.1.0"

// Look up an OID by name
entry, err := mibDB.GetOIDByName("ifDescr")

// Look up an OID by numeric value
entry, err := mibDB.GetOIDByNumeric("1.3.6.1.2.1.2.2.1.2")

// Search for OIDs by pattern
entries, err := mibDB.SearchOIDs("sys")

// Get all OIDs from a specific MIB
entries, err := mibDB.GetOIDsByMIB("SNMPv2-MIB")

// Get database statistics
stats, err := mibDB.Stats()
```

## Adding New OIDs

To add new OIDs, update `builtin_oids.go`:

```go
var builtinOIDs = []OIDEntry{
    // ... existing entries ...
    {
        Name:     "newOidName",
        OID:      "1.3.6.1.2.1.x.y.z",
        FullPath: "iso(1).org(3)....",
        MIBName:  "SOME-MIB",
    },
}
```

The OIDs are loaded into the database on each application startup using `INSERT OR REPLACE`,
so new entries are automatically added without requiring migrations.

## Database Schema

The MIB database uses the `mib_oid_names` table:

```sql
CREATE TABLE mib_oid_names (
    name TEXT PRIMARY KEY,          -- Human-readable name (e.g., "sysDescr")
    oid TEXT NOT NULL UNIQUE,       -- Numeric OID (e.g., "1.3.6.1.2.1.1.1")
    full_path TEXT,                 -- Full path (e.g., "iso(1).org(3)...")
    mib_name TEXT                   -- Source MIB (e.g., "SNMPv2-MIB")
);
```

## Source

The built-in OID definitions were ported from `fluke.niac.snmp.OidMap.java` in the original
Java implementation. This provides compatibility with existing network discovery workflows.
