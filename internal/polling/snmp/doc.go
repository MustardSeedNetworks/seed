// Package snmp drives SNMP polling against configured targets via an
// ordered collector chain per target. One poller per Seed instance;
// per-target collector chain stored in polling_targets.collector_chain
// (JSON array of collector names).
//
// Distinction from internal/protocols/snmp: this package owns the
// polling lifecycle and target scheduling. internal/protocols/snmp
// owns the protocol primitives (OID definitions, gosnmp session
// wrappers, MIB parsers). Collectors in this package call into
// internal/protocols/snmp for the wire-level work.
//
// V1.0 collectors: sys_info, if_table, lldp, arp, fdb, routing,
// host_resources, bgp4_mib, microburst_counters. Each emits some
// combination of metrics, events, and topology observations.
//
// V1.0 NMS expansion — Stage A0 scaffold (2026-05-30). Absorbs
// internal/services/estatepoll/, internal/services/servermon/, and
// the SNMP-polling half of internal/services/microburst/ during
// Stage A3.
package snmp
