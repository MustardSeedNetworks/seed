// Package probe defines the unified probe engine for seed — one
// engine, one config table, one results table for all probe-style
// observations (DNS, TLS, PING, TCP, UDP, HTTP, HTTPS, RTSP, DICOM,
// HL7, FHIR, LTI, LDAP, OPCUA, MODBUS, NTP, SIP, 802.1X, cable,
// multi-step transactions).
//
// Today's parallel-stack implementations (internal/api/health_checks_*.go,
// internal/services/dnsmon/, internal/services/sslmon/) are absorbed
// into this package during Stage A1 of the unified-architecture
// refactor. See msn-docs-internal/01-Strategy/SEED_ARCHITECTURE.md
// section 3.1.
//
// V1.0 NMS expansion — Stage A0 scaffold (2026-05-30).
package probe
