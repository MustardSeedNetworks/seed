/**
 * Mock Data for Storybook Stories
 *
 * Centralized mock data for use in Storybook stories.
 * This avoids inline mocking hacks and provides consistent,
 * realistic test data across all stories.
 *
 * Usage:
 * ```tsx
 * import { mockLinkData, mockNetworkDiscoveryData } from '../../test/mockData';
 *
 * export const Default: Story = {
 *   args: {
 *     data: mockLinkData.connected,
 *   },
 * };
 * ```
 */

import type { CableData } from "../components/cards/CableCard";
import type { DnsData } from "../components/cards/DnsCard";
import type { GatewayData } from "../components/cards/GatewayCard";
import type { LinkData } from "../components/cards/LinkCard";
import type { DhcpData } from "../components/cards/NetworkCard";
import type { NetworkDiscoveryData } from "../components/cards/NetworkDiscoveryCard";
import type { PublicIpData } from "../components/cards/PublicIpCard";
import type { SwitchData, VlanData } from "../components/cards/SwitchCard";
import type { WiFiData } from "../components/cards/WiFiCard";

// ============================================================================
// Link Card Mock Data
// ============================================================================

export const mockLinkData = {
  connected: {
    linkUp: true,
    carrier: true,
    hasIp: true,
    speed: "1000Mb/s",
    duplex: "full",
    advertisedSpeeds: ["10Mb/s", "100Mb/s", "1000Mb/s"],
    mtu: 1500,
    autoNeg: true,
    flapCount24h: 0,
    uptimeMs: 86400000,
  } satisfies LinkData,

  noCarrier: {
    linkUp: true,
    carrier: false,
    hasIp: false,
    speed: "",
    duplex: "",
    advertisedSpeeds: [],
    mtu: 1500,
    autoNeg: true,
    flapCount24h: 3,
    uptimeMs: 0,
  } satisfies LinkData,

  noIp: {
    linkUp: true,
    carrier: true,
    hasIp: false,
    speed: "1000Mb/s",
    duplex: "full",
    advertisedSpeeds: ["10Mb/s", "100Mb/s", "1000Mb/s"],
    mtu: 1500,
    autoNeg: true,
    flapCount24h: 1,
    uptimeMs: 30000,
  } satisfies LinkData,

  slow: {
    linkUp: true,
    carrier: true,
    hasIp: true,
    speed: "100Mb/s",
    duplex: "half",
    advertisedSpeeds: ["10Mb/s", "100Mb/s"],
    mtu: 1500,
    autoNeg: false,
    flapCount24h: 5,
    uptimeMs: 3600000,
  } satisfies LinkData,

  tenGig: {
    linkUp: true,
    carrier: true,
    hasIp: true,
    speed: "10000Mb/s",
    duplex: "full",
    advertisedSpeeds: ["1000Mb/s", "10000Mb/s"],
    mtu: 9000,
    autoNeg: true,
    flapCount24h: 0,
    uptimeMs: 604800000,
  } satisfies LinkData,
};

// ============================================================================
// Switch Card Mock Data
// ============================================================================

export const mockSwitchData = {
  lldp: {
    protocol: "LLDP",
    switchName: "core-sw-01.example.com",
    portId: "GigabitEthernet0/1",
    portDescription: "User Access Port",
    managementIp: "10.0.1.1",
    systemDescription: "Cisco IOS Software, C3750 Software",
  } satisfies SwitchData,

  cdp: {
    protocol: "CDP",
    switchName: "access-sw-floor2",
    portId: "Fa0/24",
    portDescription: "Office 201",
    managementIp: "192.168.1.254",
    systemDescription: "Cisco Catalyst 2960",
  } satisfies SwitchData,

  noNeighbor: null,
};

export const mockVlanData = {
  tagged: {
    nativeVlan: 1,
    taggedVlans: [10, 20, 30, 100],
    voiceVlan: 100,
    configured: { enabled: true, id: 10 },
  } satisfies VlanData,

  untagged: {
    nativeVlan: 1,
    taggedVlans: [],
    voiceVlan: null,
    configured: { enabled: false, id: 0 },
  } satisfies VlanData,
};

// ============================================================================
// Network Card (DHCP) Mock Data
// ============================================================================

export const mockDhcpData = {
  dhcpFast: {
    mac: "aa:bb:cc:dd:ee:ff",
    mode: "dhcp",
    ipv4: {
      address: "192.168.1.100",
      subnet: "24",
      gateway: "192.168.1.1",
      dhcpServer: "192.168.1.1",
      leaseTime: 86400,
    },
    ipv6: [],
    dns: ["192.168.1.1", "8.8.8.8"],
    timing: {
      discover: 45,
      offer: 32,
      request: 28,
      ack: 41,
      total: 146,
    },
  } satisfies DhcpData,

  dhcpSlow: {
    mac: "aa:bb:cc:dd:ee:ff",
    mode: "dhcp",
    ipv4: {
      address: "192.168.1.100",
      subnet: "24",
      gateway: "192.168.1.1",
      dhcpServer: "192.168.1.1",
      leaseTime: 3600,
    },
    ipv6: [],
    dns: ["192.168.1.1"],
    timing: {
      discover: 320,
      offer: 280,
      request: 190,
      ack: 245,
      total: 1035,
    },
  } satisfies DhcpData,

  static: {
    mac: "aa:bb:cc:dd:ee:ff",
    mode: "static",
    ipv4: {
      address: "10.0.1.100",
      subnet: "24",
      gateway: "10.0.1.1",
      dhcpServer: null,
      leaseTime: null,
    },
    ipv6: [],
    dns: ["10.0.1.1", "1.1.1.1", "8.8.8.8"],
    timing: null,
  } satisfies DhcpData,

  dualStack: {
    mac: "aa:bb:cc:dd:ee:ff",
    mode: "dhcp",
    ipv4: {
      address: "192.168.1.100",
      subnet: "24",
      gateway: "192.168.1.1",
      dhcpServer: "192.168.1.1",
      leaseTime: 86400,
    },
    ipv6: [
      {
        address: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
        prefix: 64,
        scope: "global",
        source: "slaac",
      },
      {
        address: "fe80:0000:0000:0000:1234:5678:9abc:def0",
        prefix: 64,
        scope: "link-local",
        source: "slaac",
      },
    ],
    dns: ["2001:4860:4860::8888", "192.168.1.1"],
    timing: {
      discover: 52,
      offer: 38,
      request: 31,
      ack: 44,
      total: 165,
    },
  } satisfies DhcpData,

  noIp: {
    mac: "aa:bb:cc:dd:ee:ff",
    mode: "dhcp",
    ipv4: null,
    ipv6: [],
    dns: [],
    timing: null,
  } satisfies DhcpData,
};

// ============================================================================
// DNS Card Mock Data
// ============================================================================

export const mockDnsData = {
  allSuccess: {
    server: "192.168.1.1",
    servers: ["192.168.1.1", "8.8.8.8"],
    testHostname: "google.com",
    forward: {
      result: "success",
      time: 25,
      timeMs: 25,
      status: "ok",
      resolved: "142.250.185.46",
    },
    forwardIpv6: {
      result: "success",
      time: 30,
      timeMs: 30,
      status: "ok",
      resolved: "2607:f8b0:4004:800::200e",
    },
    reverse: {
      result: "success",
      time: 45,
      timeMs: 45,
      status: "ok",
      resolved: "dns.google",
    },
    reverseIpv6: null,
  } satisfies DnsData,

  slowDns: {
    server: "192.168.1.1",
    servers: ["192.168.1.1"],
    testHostname: "example.com",
    forward: {
      result: "success",
      time: 350,
      timeMs: 350,
      status: "warning",
      resolved: "93.184.216.34",
    },
    forwardIpv6: null,
    reverse: {
      result: "success",
      time: 420,
      timeMs: 420,
      status: "warning",
      resolved: "host.example.com",
    },
    reverseIpv6: null,
  } satisfies DnsData,

  failed: {
    server: "192.168.1.1",
    servers: ["192.168.1.1"],
    testHostname: "invalid.local",
    forward: {
      result: "error",
      time: 5000,
      timeMs: 5000,
      status: "error",
      error: "NXDOMAIN",
    },
    forwardIpv6: null,
    reverse: null,
    reverseIpv6: null,
  } satisfies DnsData,
};

// ============================================================================
// Gateway Card Mock Data
// ============================================================================

export const mockGatewayData = {
  reachable: {
    gateway: "192.168.1.1",
    reachable: true,
    sent: 10,
    received: 10,
    lossPercent: 0,
    minTime: 1.2,
    maxTime: 5.8,
    avgTime: 2.5,
    lastTime: 2.1,
    status: "success",
    ipv6: {
      gateway: "fe80::1",
      reachable: true,
      sent: 10,
      received: 10,
      lossPercent: 0,
      minTime: 1.5,
      maxTime: 6.2,
      avgTime: 2.8,
      lastTime: 2.3,
      status: "success",
    },
  } satisfies GatewayData,

  partialLoss: {
    gateway: "192.168.1.1",
    reachable: true,
    sent: 10,
    received: 8,
    lossPercent: 20,
    minTime: 5.2,
    maxTime: 125.8,
    avgTime: 45.5,
    lastTime: 38.1,
    status: "warning",
  } satisfies GatewayData,

  unreachable: {
    gateway: "192.168.1.1",
    reachable: false,
    sent: 10,
    received: 0,
    lossPercent: 100,
    minTime: 0,
    maxTime: 0,
    avgTime: 0,
    lastTime: 0,
    status: "error",
  } satisfies GatewayData,
};

// ============================================================================
// WiFi Card Mock Data
// ============================================================================

export const mockWiFiData = {
  strongSignal: {
    ssid: "CorporateWiFi",
    bssid: "aa:bb:cc:dd:ee:ff",
    signal: -45,
    channel: 36,
    frequency: 5180,
    security: "WPA3-Personal",
  } satisfies WiFiData,

  mediumSignal: {
    ssid: "GuestNetwork",
    bssid: "11:22:33:44:55:66",
    signal: -65,
    channel: 6,
    frequency: 2437,
    security: "WPA2-Personal",
  } satisfies WiFiData,

  weakSignal: {
    ssid: "FarAwayNetwork",
    bssid: "77:88:99:aa:bb:cc",
    signal: -82,
    channel: 11,
    frequency: 2462,
    security: "WPA2-Enterprise",
  } satisfies WiFiData,
};

// ============================================================================
// Cable Card Mock Data
// ============================================================================

export const mockCableData = {
  good: {
    supported: true,
    length: 15,
    status: "ok",
    faults: [],
  } satisfies CableData,

  short: {
    supported: true,
    length: 25,
    status: "short",
    faults: [{ pair: "A", status: "short", distance: 25 }],
  } satisfies CableData,

  open: {
    supported: true,
    length: 50,
    status: "open",
    faults: [
      { pair: "B", status: "open", distance: 50 },
      { pair: "D", status: "open", distance: 52 },
    ],
  } satisfies CableData,

  unsupported: {
    supported: false,
    length: null,
    status: "unknown",
    faults: [],
  } satisfies CableData,
};

// ============================================================================
// Network Discovery Mock Data
// ============================================================================

export const mockNetworkDiscoveryData = {
  withDevices: {
    devices: [
      {
        ip: "192.168.1.1",
        mac: "aa:bb:cc:dd:ee:ff",
        hostname: "router.local",
        vendor: "Cisco Systems",
        lastSeen: new Date(Date.now() - 60000).toISOString(),
        deviceType: "router",
        openPorts: [22, 80, 443],
      },
      {
        ip: "192.168.1.100",
        mac: "11:22:33:44:55:66",
        hostname: "workstation-01",
        vendor: "Dell Inc.",
        lastSeen: new Date(Date.now() - 30000).toISOString(),
        deviceType: "computer",
        openPorts: [22],
      },
      {
        ip: "192.168.1.150",
        mac: "77:88:99:aa:bb:cc",
        hostname: "printer-office",
        vendor: "HP Inc.",
        lastSeen: new Date(Date.now() - 120000).toISOString(),
        deviceType: "printer",
        openPorts: [9100],
      },
    ],
    status: {
      scanning: false,
      deviceCount: 3,
      lastScan: new Date(Date.now() - 60000).toISOString(),
      subnet: "192.168.1.0/24",
      localIP: "192.168.1.100",
      interface: "eth0",
    },
  } satisfies NetworkDiscoveryData,

  scanning: {
    devices: [],
    status: {
      scanning: true,
      deviceCount: 0,
      lastScan: "",
      subnet: "192.168.1.0/24",
      localIP: "192.168.1.100",
      interface: "eth0",
    },
  } satisfies NetworkDiscoveryData,

  empty: {
    devices: [],
    status: {
      scanning: false,
      deviceCount: 0,
      lastScan: new Date(Date.now() - 300000).toISOString(),
      subnet: "192.168.1.0/24",
      localIP: "192.168.1.100",
      interface: "eth0",
    },
  } satisfies NetworkDiscoveryData,
};

// ============================================================================
// Public IP Mock Data
// ============================================================================

export const mockPublicIpData = {
  dualStack: {
    ipv4: "203.0.113.42",
    ipv6: "2001:db8::42",
    lastChecked: new Date(Date.now() - 300000).toISOString(),
  } satisfies PublicIpData,

  ipv4Only: {
    ipv4: "198.51.100.123",
    lastChecked: new Date(Date.now() - 60000).toISOString(),
  } satisfies PublicIpData,

  error: {
    error: "Unable to determine public IP",
    lastChecked: new Date().toISOString(),
  } satisfies PublicIpData,
};
