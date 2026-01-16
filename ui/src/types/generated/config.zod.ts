// biome-ignore-all lint/nursery/useExplicitType: Generated Zod schemas - types are inferred by Zod
import { z } from "zod";

// Server configuration schema
const serverSchema = z
  .object({
    port: z.number().int().optional(),
    host: z.string().optional(),
    tls: z.boolean().optional(),
  })
  .passthrough();

// Interface configuration schema
const interfaceSchema = z
  .object({
    name: z.string().optional(),
    preferred: z.string().optional(),
  })
  .passthrough();

// VLAN configuration schema
const vlanSchema = z
  .object({
    enabled: z.boolean().optional(),
    id: z.number().int().optional(),
    name: z.string().optional(),
  })
  .passthrough();

// IP configuration schema
const ipSchema = z
  .object({
    mode: z.enum(["dhcp", "static"]).optional(),
    address: z.string().optional(),
    netmask: z.string().optional(),
    gateway: z.string().optional(),
    dns: z.array(z.string()).optional(),
  })
  .passthrough();

// Discovery configuration schema
const discoverySchema = z
  .object({
    enabled: z.boolean().optional(),
    timeout: z.number().optional(),
  })
  .passthrough();

// Network discovery configuration schema
const networkDiscoverySchema = z
  .object({
    enabled: z.boolean().optional(),
    arpScanWorkers: z.number().int().optional(),
    pingTimeoutMs: z.number().int().optional(),
    scanTimeoutMs: z.number().int().optional(),
    autoScan: z.boolean().optional(),
    scanIntervalMs: z.number().int().optional(),
    ipv6Enabled: z.boolean().optional(),
    options: z
      .object({
        passiveProtocols: z
          .object({
            lldp: z.boolean().optional(),
            cdp: z.boolean().optional(),
            edp: z.boolean().optional(),
            ndp: z.boolean().optional(),
          })
          .passthrough()
          .optional(),
        arpScan: z.boolean().optional(),
        icmpScan: z.boolean().optional(),
        traceroute: z.boolean().optional(),
        snmpQuery: z.boolean().optional(),
      })
      .passthrough()
      .optional(),
    timing: z
      .object({
        probeIntervalMs: z.number().int().optional(),
        rescanIntervalMs: z.number().int().optional(),
        workers: z.number().int().optional(),
      })
      .passthrough()
      .optional(),
  })
  .passthrough();

// DNS configuration schema
const dnsSchema = z
  .object({
    servers: z.array(z.string()).optional(),
    timeout: z.number().optional(),
  })
  .passthrough();

// Test target schema
const pingTargetSchema = z.object({
  id: z.string().optional(),
  name: z.string(),
  host: z.string(),
  enabled: z.boolean(),
  count: z.number().int().optional(),
});

const httpEndpointSchema = z.object({
  id: z.string().optional(),
  name: z.string(),
  url: z.string(),
  expectedStatus: z.number().int(),
  enabled: z.boolean(),
});

const tcpPortSchema = z.object({
  id: z.string().optional(),
  name: z.string(),
  host: z.string(),
  port: z.number().int(),
  enabled: z.boolean(),
});

// Tests configuration schema
const testsSchema = z
  .object({
    dnsHostname: z.string().optional(),
    dnsServers: z
      .array(
        z.object({
          address: z.string(),
          enabled: z.boolean(),
        }),
      )
      .optional(),
    pingTargets: z.array(pingTargetSchema).optional(),
    tcpPorts: z.array(tcpPortSchema).optional(),
    udpPorts: z.array(tcpPortSchema).optional(),
    httpEndpoints: z.array(httpEndpointSchema).optional(),
    runPerformance: z.boolean().optional(),
    runSpeedtest: z.boolean().optional(),
    runIperf: z.boolean().optional(),
    runDiscovery: z.boolean().optional(),
  })
  .passthrough();

// Speedtest configuration schema
const speedtestSchema = z
  .object({
    serverId: z.string().optional(),
    autoRunOnLink: z.boolean().optional(),
  })
  .passthrough();

// iPerf configuration schema
const iperfSchema = z
  .object({
    server: z.string().optional(),
    port: z.number().int().optional(),
    protocol: z.enum(["tcp", "udp"]).optional(),
    direction: z.enum(["upload", "download", "bidirectional"]).optional(),
    duration: z.number().int().optional(),
    serverPort: z.number().int().optional(),
    enableServer: z.boolean().optional(),
    autoRunOnLink: z.boolean().optional(),
  })
  .passthrough();

// Threshold pair schema
const thresholdPairSchema = z.object({
  good: z.number(),
  warning: z.number(),
});

// Thresholds configuration schema
const thresholdsSchema = z
  .object({
    dns: thresholdPairSchema.optional(),
    gateway: thresholdPairSchema.optional(),
    wifi: thresholdPairSchema.optional(),
    customPing: thresholdPairSchema.optional(),
    customTcp: thresholdPairSchema.optional(),
    customHttp: thresholdPairSchema.optional(),
    httpTimings: z
      .object({
        dns: thresholdPairSchema.optional(),
        tcp: thresholdPairSchema.optional(),
        tls: thresholdPairSchema.optional(),
        ttfb: thresholdPairSchema.optional(),
      })
      .passthrough()
      .optional(),
  })
  .passthrough();

// Auth configuration schema
const authSchema = z
  .object({
    enabled: z.boolean().optional(),
    jwtSecret: z.string().optional(),
    tokenExpiry: z.number().optional(),
    refreshExpiry: z.number().optional(),
  })
  .passthrough();

// Security configuration schema
const securitySchema = z
  .object({
    cors: z
      .object({
        enabled: z.boolean().optional(),
        allowedOrigins: z.array(z.string()).optional(),
      })
      .passthrough()
      .optional(),
  })
  .passthrough();

// DHCP configuration schema
const dhcpSchema = z
  .object({
    enabled: z.boolean().optional(),
  })
  .passthrough();

// SNMP V3 credential schema
const snmpV3CredentialSchema = z.object({
  id: z.string().optional(),
  name: z.string(),
  username: z.string(),
  authProtocol: z.string().optional(),
  authPassword: z.string().optional(),
  privProtocol: z.string().optional(),
  privPassword: z.string().optional(),
  contextName: z.string().optional(),
  securityLevel: z.string().optional(),
});

// SNMP configuration schema
const snmpSchema = z
  .object({
    communities: z.array(z.string()).optional(),
    v3Credentials: z.array(snmpV3CredentialSchema).optional(),
    timeout: z.number().int().optional(),
    retries: z.number().int().optional(),
    port: z.number().int().optional(),
  })
  .passthrough();

// FAB options schema
const fabOptionsSchema = z
  .object({
    enabled: z.boolean().optional(),
    autoRun: z.boolean().optional(),
  })
  .passthrough();

// Display options schema
const displayOptionsSchema = z
  .object({
    showPublicIp: z.boolean().optional(),
    unitSystem: z.enum(["sae", "metric"]).optional(),
  })
  .passthrough();

// Logging configuration schema
const loggingSchema = z
  .object({
    level: z.enum(["debug", "info", "warn", "error"]).optional(),
    file: z.string().optional(),
  })
  .passthrough();

export const configSchema = z
  .object({
    version: z
      .number()
      .int()
      .gte(1)
      .describe("Configuration schema version for migrations")
      .default(1),
    server: serverSchema.optional(),
    interface: interfaceSchema.optional(),
    vlan: vlanSchema.optional(),
    ip: ipSchema.optional(),
    discovery: discoverySchema.optional(),
    // biome-ignore lint/style/useNamingConvention: Generated schema - properties match backend API using snake_case
    network_discovery: networkDiscoverySchema.optional(),
    dns: dnsSchema.optional(),
    tests: testsSchema.optional(),
    speedtest: speedtestSchema.optional(),
    iperf: iperfSchema.optional(),
    thresholds: thresholdsSchema.optional(),
    auth: authSchema.optional(),
    security: securitySchema.optional(),
    dhcp: dhcpSchema.optional(),
    snmp: snmpSchema.optional(),
    // biome-ignore lint/style/useNamingConvention: Generated schema - properties match backend API using snake_case
    fab_options: fabOptionsSchema.optional(),
    // biome-ignore lint/style/useNamingConvention: Generated schema - properties match backend API using snake_case
    display_options: displayOptionsSchema.optional(),
    logging: loggingSchema.optional(),
  })
  .passthrough()
  .describe("Configuration schema for The Seed network diagnostic tool");
