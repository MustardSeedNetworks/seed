import type { Meta, StoryObj } from "@storybook/react-vite";
import { NetworkDiscoveryCard } from "./NetworkDiscoveryCard";

/**
 * NetworkDiscoveryCard displays discovered network devices via multiple protocols
 * (ARP, NDP, LLDP, CDP, EDP, mDNS, ICMP ping).
 *
 * Features:
 * - Multi-protocol device discovery with method badges
 * - Device categorization (routers, servers, workstations, printers, mobile, network equipment)
 * - Search and filter capabilities across IP, hostname, vendor, MAC
 * - Sorting by IP, hostname, vendor, or last seen
 * - Expandable device details showing MAC, OS guess, TTL, discovery protocols
 * - LLDP/CDP/EDP switch information
 * - Auto-profiling with detected services and open ports
 * - Deep scan capability for port scanning
 * - Vulnerability indicators with CVE counts
 * - Separate local and extended network sections
 *
 * This story demonstrates various discovery states and device types.
 */
const meta = {
  title: "Cards/NetworkDiscoveryCard",
  component: NetworkDiscoveryCard,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  decorators: [
    (Story) => (
      <div style={{ width: "480px", maxHeight: "600px" }}>
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof NetworkDiscoveryCard>;

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Scanning in progress.
 * Shows loading state with "Scanning..." message.
 */
export const Scanning: Story = {
  args: {
    loading: true,
    data: null,
  },
};

/**
 * No devices discovered yet.
 * Shows empty state prompting user to start a scan.
 */
export const NoDevices: Story = {
  args: {
    loading: false,
    data: {
      devices: [],
      status: {
        scanning: false,
        deviceCount: 0,
        lastScan: new Date().toISOString(),
        subnet: "192.168.1.0/24",
        localIP: "192.168.1.100",
        interface: "en0",
      },
    },
  },
};

/**
 * Small home network with typical devices.
 * Shows router, a few workstations, and mobile devices.
 */
export const SmallHomeNetwork: Story = {
  args: {
    loading: false,
    data: {
      devices: [
        {
          ip: "192.168.1.1",
          mac: "aa:bb:cc:dd:ee:01",
          hostname: "router.local",
          vendor: "Ubiquiti Networks",
          osGuess: "EdgeOS",
          ttl: 64,
          discoveryMethod: ["arp", "ping", "lldp"],
          lastSeen: new Date(Date.now() - 30000).toISOString(),
          isLocal: true,
          isRouter: true,
          lldpInfo: {
            chassisId: "aa:bb:cc:dd:ee:01",
            portId: "eth0",
            systemName: "EdgeRouter-X",
            systemDescription: "EdgeOS v2.0.9",
            capabilities: ["Router", "Bridge"],
            managementAddress: "192.168.1.1",
          },
        },
        {
          ip: "192.168.1.10",
          mac: "aa:bb:cc:dd:ee:02",
          hostname: "macbook-pro.local",
          vendor: "Apple, Inc.",
          osGuess: "macOS",
          ttl: 64,
          discoveryMethod: ["arp", "mdns"],
          lastSeen: new Date(Date.now() - 120000).toISOString(),
          isLocal: true,
          profile: {
            profiledAt: new Date().toISOString(),
            deviceType: "workstation",
            deviceIcons: ["ssh", "web"],
            openPorts: [
              { port: 22, protocol: "tcp", service: "SSH", isOpen: true },
              { port: 80, protocol: "tcp", service: "HTTP", isOpen: true },
            ],
          },
        },
        {
          ip: "192.168.1.20",
          mac: "aa:bb:cc:dd:ee:03",
          hostname: "iphone.local",
          vendor: "Apple, Inc.",
          osGuess: "iOS",
          ttl: 64,
          discoveryMethod: ["arp", "mdns"],
          lastSeen: new Date(Date.now() - 60000).toISOString(),
          isLocal: true,
        },
      ],
      status: {
        scanning: false,
        deviceCount: 3,
        lastScan: new Date(Date.now() - 300000).toISOString(),
        subnet: "192.168.1.0/24",
        localIP: "192.168.1.100",
        interface: "en0",
      },
    },
  },
};

/**
 * Enterprise network with switches and servers.
 * Demonstrates LLDP/CDP discovery, multiple VLANs, and varied device types.
 */
export const EnterpriseNetwork: Story = {
  args: {
    loading: false,
    data: {
      devices: [
        {
          ip: "10.0.1.1",
          mac: "00:1a:2b:3c:4d:01",
          hostname: "core-switch-01",
          vendor: "Cisco Systems",
          osGuess: "IOS",
          ttl: 255,
          discoveryMethod: ["arp", "cdp", "lldp"],
          lastSeen: new Date(Date.now() - 10000).toISOString(),
          isLocal: true,
          cdpInfo: {
            deviceId: "CORE-SW-01",
            portId: "GigabitEthernet1/0/1",
            platform: "Catalyst 9300",
            softwareVersion: "17.9.4a",
            capabilities: ["Switch", "Router"],
            managementAddress: "10.0.1.1",
            nativeVlan: 1,
          },
        },
        {
          ip: "10.0.2.10",
          mac: "00:1a:2b:3c:4d:02",
          hostname: "db-server-01",
          vendor: "Dell Inc.",
          osGuess: "Linux",
          ttl: 64,
          discoveryMethod: ["arp", "ping"],
          lastSeen: new Date(Date.now() - 5000).toISOString(),
          isLocal: true,
          profile: {
            profiledAt: new Date().toISOString(),
            deviceType: "server",
            deviceIcons: ["database", "ssh", "web"],
            openPorts: [
              { port: 22, protocol: "tcp", service: "SSH", isOpen: true },
              { port: 3306, protocol: "tcp", service: "MySQL", isOpen: true },
            ],
            httpInfo: {
              port: 80,
              statusCode: 200,
              title: "Database Admin Portal",
              server: "nginx/1.24.0",
              isHttps: false,
            },
          },
        },
        {
          ip: "10.0.3.50",
          mac: "00:1a:2b:3c:4d:03",
          hostname: "printer-finance",
          vendor: "Hewlett Packard",
          osGuess: "Printer",
          ttl: 128,
          discoveryMethod: ["arp", "mdns"],
          lastSeen: new Date(Date.now() - 120000).toISOString(),
          isLocal: true,
          profile: {
            profiledAt: new Date().toISOString(),
            deviceType: "printer",
            deviceIcons: ["printer", "web"],
          },
        },
      ],
      status: {
        scanning: false,
        deviceCount: 3,
        lastScan: new Date(Date.now() - 600000).toISOString(),
        subnet: "10.0.0.0/16",
        localIP: "10.0.1.100",
        interface: "eth0",
      },
    },
  },
};

/**
 * Devices with vulnerabilities detected.
 * Shows CVE counts and severity indicators.
 */
export const DevicesWithVulnerabilities: Story = {
  args: {
    loading: false,
    data: {
      devices: [
        {
          ip: "192.168.1.50",
          mac: "aa:bb:cc:dd:ee:10",
          hostname: "old-nas.local",
          vendor: "Synology",
          osGuess: "Linux",
          ttl: 64,
          discoveryMethod: ["arp", "ping"],
          lastSeen: new Date(Date.now() - 30000).toISOString(),
          isLocal: true,
          vulnerabilities: {
            count: 12,
            highestSeverity: "CRITICAL",
          },
        },
        {
          ip: "192.168.1.51",
          mac: "aa:bb:cc:dd:ee:11",
          hostname: "iot-camera-01",
          vendor: "Hikvision",
          osGuess: "Embedded Linux",
          ttl: 64,
          discoveryMethod: ["arp"],
          lastSeen: new Date(Date.now() - 60000).toISOString(),
          isLocal: true,
          vulnerabilities: {
            count: 5,
            highestSeverity: "HIGH",
          },
        },
        {
          ip: "192.168.1.52",
          mac: "aa:bb:cc:dd:ee:12",
          hostname: "workstation-03",
          vendor: "Dell Inc.",
          osGuess: "Windows",
          ttl: 128,
          discoveryMethod: ["arp", "ping"],
          lastSeen: new Date(Date.now() - 15000).toISOString(),
          isLocal: true,
          vulnerabilities: {
            count: 3,
            highestSeverity: "MEDIUM",
          },
        },
      ],
      status: {
        scanning: false,
        deviceCount: 3,
        lastScan: new Date().toISOString(),
        subnet: "192.168.1.0/24",
        localIP: "192.168.1.100",
        interface: "en0",
      },
    },
  },
};

/**
 * Mixed local and extended networks.
 * Shows devices on both local subnet and extended networks (via routing).
 */
export const LocalAndExtended: Story = {
  args: {
    loading: false,
    data: {
      devices: [
        // Local devices
        {
          ip: "192.168.1.1",
          mac: "aa:bb:cc:dd:ee:01",
          hostname: "gateway",
          vendor: "Ubiquiti Networks",
          discoveryMethod: ["arp", "lldp"],
          lastSeen: new Date().toISOString(),
          isLocal: true,
        },
        {
          ip: "192.168.1.10",
          mac: "aa:bb:cc:dd:ee:02",
          hostname: "laptop",
          vendor: "Dell Inc.",
          discoveryMethod: ["arp"],
          lastSeen: new Date().toISOString(),
          isLocal: true,
        },
        // Extended network devices
        {
          ip: "10.20.0.5",
          mac: "00:00:00:00:00:00",
          hostname: "remote-server",
          vendor: "Unknown",
          discoveryMethod: ["ping"],
          lastSeen: new Date().toISOString(),
          isLocal: false,
        },
        {
          ip: "172.16.5.100",
          mac: "00:00:00:00:00:00",
          hostname: "vpn-client",
          vendor: "Unknown",
          discoveryMethod: ["ping"],
          lastSeen: new Date().toISOString(),
          isLocal: false,
        },
      ],
      status: {
        scanning: false,
        deviceCount: 4,
        lastScan: new Date().toISOString(),
        subnet: "192.168.1.0/24",
        localIP: "192.168.1.100",
        interface: "en0",
      },
    },
  },
};

/**
 * IPv6-enabled network.
 * Shows devices with both IPv4 and IPv6 addresses.
 */
export const IPv6Network: Story = {
  args: {
    loading: false,
    data: {
      devices: [
        {
          ip: "192.168.1.1",
          ipv6: "fe80::1",
          ipv6Addresses: ["fe80::1", "2001:db8::1"],
          mac: "aa:bb:cc:dd:ee:01",
          hostname: "router-v6",
          vendor: "Ubiquiti Networks",
          discoveryMethod: ["arp", "ndp"],
          lastSeen: new Date().toISOString(),
          isLocal: true,
          isRouter: true,
          ndpInfo: {
            linkLayerAddress: "aa:bb:cc:dd:ee:01",
            isRouter: true,
            lastAdvertisement: new Date().toISOString(),
          },
        },
        {
          ip: "192.168.1.10",
          ipv6: "fe80::a12:34ff:fe56:7890",
          mac: "aa:12:34:56:78:90",
          hostname: "dual-stack-host",
          vendor: "Apple, Inc.",
          discoveryMethod: ["arp", "ndp"],
          lastSeen: new Date().toISOString(),
          isLocal: true,
        },
      ],
      status: {
        scanning: false,
        deviceCount: 2,
        lastScan: new Date().toISOString(),
        subnet: "192.168.1.0/24",
        localIP: "192.168.1.100",
        interface: "en0",
      },
    },
  },
};

/**
 * Scan complete with categorized summary.
 * Shows device type breakdown in summary section.
 */
export const ScanComplete: Story = {
  args: {
    loading: false,
    data: {
      devices: [
        // 1 router
        {
          ip: "192.168.1.1",
          mac: "aa:bb:cc:00:00:01",
          hostname: "router",
          vendor: "Ubiquiti",
          discoveryMethod: ["arp"],
          lastSeen: new Date().toISOString(),
          isLocal: true,
          profile: {
            profiledAt: new Date().toISOString(),
            deviceType: "router",
            deviceIcons: ["router"],
          },
        },
        // 2 servers
        {
          ip: "192.168.1.10",
          mac: "aa:bb:cc:00:00:02",
          hostname: "nas",
          vendor: "Synology",
          discoveryMethod: ["arp"],
          lastSeen: new Date().toISOString(),
          isLocal: true,
          profile: {
            profiledAt: new Date().toISOString(),
            deviceType: "server",
            deviceIcons: ["server"],
          },
        },
        {
          ip: "192.168.1.11",
          mac: "aa:bb:cc:00:00:03",
          hostname: "db-server",
          vendor: "Dell",
          discoveryMethod: ["arp"],
          lastSeen: new Date().toISOString(),
          isLocal: true,
          profile: {
            profiledAt: new Date().toISOString(),
            deviceType: "server",
            deviceIcons: ["database"],
          },
        },
        // 1 printer
        {
          ip: "192.168.1.20",
          mac: "aa:bb:cc:00:00:04",
          hostname: "printer",
          vendor: "HP",
          discoveryMethod: ["arp"],
          lastSeen: new Date().toISOString(),
          isLocal: true,
          profile: {
            profiledAt: new Date().toISOString(),
            deviceType: "printer",
            deviceIcons: ["printer"],
          },
        },
        // 2 mobile
        {
          ip: "192.168.1.30",
          mac: "aa:bb:cc:00:00:05",
          hostname: "iphone",
          vendor: "Apple",
          discoveryMethod: ["arp"],
          lastSeen: new Date().toISOString(),
          isLocal: true,
        },
        {
          ip: "192.168.1.31",
          mac: "aa:bb:cc:00:00:06",
          hostname: "android",
          vendor: "Samsung",
          discoveryMethod: ["arp"],
          lastSeen: new Date().toISOString(),
          isLocal: true,
        },
      ],
      status: {
        scanning: false,
        deviceCount: 6,
        lastScan: new Date().toISOString(),
        subnet: "192.168.1.0/24",
        localIP: "192.168.1.100",
        interface: "en0",
      },
    },
  },
};
