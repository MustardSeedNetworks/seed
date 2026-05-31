// Package listener defines passive ingress endpoints — the 4th
// observation primitive in seed's unified architecture (alongside
// Probes, Metrics, Events). Listeners bind a UDP/TCP port and emit
// Events on incoming messages, in contrast to Probes (active
// outbound) and SNMP Collectors (active poll).
//
// V1.0 listeners: snmp_trap (UDP 162 — SNMPv1/v2c/v3 traps + informs),
// syslog (UDP 514 + TCP/TLS 6514, RFC 5424 + BSD).
//
// V1.1 listeners: netflow (UDP 2055), sflow (UDP 6343), ipfix
// (UDP 4739) — all aggregate into flow metrics.
//
// License gating: passive_listeners flag (Pro). Free/Starter cannot
// enable listeners.
//
// V1.0 NMS expansion — Stage A0 scaffold (2026-05-30). Implementations
// land in Stage A3.5.
package listener
