import { z } from "zod";

export const configSchema = z
  .object({
    version: z
      .number()
      .int()
      .gte(1)
      .describe("Configuration schema version for migrations")
      .default(1),
    server: z.any(),
    interface: z.any(),
    vlan: z.any(),
    ip: z.any(),
    discovery: z.any(),
    network_discovery: z.any(),
    dns: z.any(),
    tests: z.any(),
    speedtest: z.any(),
    iperf: z.any(),
    thresholds: z.any(),
    auth: z.any(),
    security: z.any(),
    dhcp: z.any(),
    snmp: z.any(),
    fab_options: z.any(),
    display_options: z.any(),
    logging: z.any(),
  })
  .strict()
  .describe("Configuration schema for The Seed network diagnostic tool");
