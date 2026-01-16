import type { Meta, StoryObj } from "@storybook/react-vite";
import { NetworkCard } from "./NetworkCard";

/**
 * NetworkCard displays network configuration and DHCP information.
 *
 * Features:
 * - IPv4 and IPv6 address display
 * - MAC address and DHCP mode (dhcp/static/auto)
 * - DHCP timing breakdown (discover, offer, request, ACK phases)
 * - Lease time information
 * - DNS servers
 * - Public IP integration (optional)
 * - Color-coded timing thresholds (green/yellow/red)
 * - IPv6 scope grouping (global, unique-local, link-local)
 * - Collapsible DHCP timing details
 *
 * This story demonstrates various network configuration states.
 */
const meta: Meta<(typeof meta)["component"]> = {
  title: "Cards/NetworkCard",
  component: NetworkCard,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  decorators: [
    (StoryComponent: React.ComponentType): JSX.Element => (
      <div style={{ width: "400px" }}>
        <StoryComponent />
      </div>
    ),
  ],
} satisfies Meta<typeof NetworkCard>;

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * DHCP IPv4 configuration with fast timing.
 * Shows successful DHCP negotiation with green status indicators.
 */
export const Dhcpv4Success: Story = {
  args: {
    data: {
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
    },
    loading: false,
  },
};

/**
 * DHCP with slow timing showing warnings.
 * Yellow status indicators for degraded DHCP performance.
 */
export const DhcpSlowTiming: Story = {
  args: {
    data: {
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
    },
    loading: false,
  },
};

/**
 * DHCP with critical timing issues.
 * Red status indicators showing severe delays.
 */
export const DhcpCriticalTiming: Story = {
  args: {
    data: {
      mac: "aa:bb:cc:dd:ee:ff",
      mode: "dhcp",
      ipv4: {
        address: "192.168.1.100",
        subnet: "24",
        gateway: "192.168.1.1",
        dhcpServer: "192.168.1.1",
        leaseTime: 7200,
      },
      ipv6: [],
      dns: ["192.168.1.1", "1.1.1.1"],
      timing: {
        discover: 1250,
        offer: 1840,
        request: 920,
        ack: 1580,
        total: 5590,
      },
    },
    loading: false,
  },
};

/**
 * Static IPv4 configuration.
 * Shows manually configured IP without DHCP timing.
 */
export const StaticIpv4: Story = {
  args: {
    data: {
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
    },
    loading: false,
  },
};

/**
 * Dual-stack configuration with IPv4 and IPv6.
 * Shows both protocol versions configured.
 */
export const DualStack: Story = {
  args: {
    data: {
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
    },
    loading: false,
  },
};

/**
 * IPv6-only configuration.
 * Shows global and link-local IPv6 addresses without IPv4.
 */
export const Ipv6Only: Story = {
  args: {
    data: {
      mac: "aa:bb:cc:dd:ee:ff",
      mode: "auto",
      ipv4: null,
      ipv6: [
        {
          address: "2001:db8::1",
          prefix: 64,
          scope: "global",
          source: "dhcpv6",
        },
        {
          address: "fd00::1234:5678",
          prefix: 64,
          scope: "unique-local",
          source: "static",
        },
        {
          address: "fe80::1",
          prefix: 64,
          scope: "link-local",
          source: "slaac",
        },
      ],
      dns: ["2001:4860:4860::8888", "2001:4860:4860::8844"],
      timing: null,
    },
    loading: false,
  },
};

/**
 * Configuration with public IP information.
 * Shows network config plus external public IP addresses.
 */
export const WithPublicIp: Story = {
  args: {
    data: {
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
          address: "fe80::1",
          prefix: 64,
          scope: "link-local",
          source: "slaac",
        },
      ],
      dns: ["192.168.1.1", "8.8.8.8"],
      timing: {
        discover: 48,
        offer: 35,
        request: 29,
        ack: 39,
        total: 151,
      },
    },
    publicip: {
      ipv4: "203.0.113.42",
      ipv6: "2001:db8::42",
      lastChecked: new Date(Date.now() - 300000).toISOString(),
    },
    loading: false,
    showPublicIp: true,
  },
};

/**
 * No IP configuration available.
 * Shows warning state when no IP is assigned.
 */
export const NoIp: Story = {
  args: {
    data: {
      mac: "aa:bb:cc:dd:ee:ff",
      mode: "dhcp",
      ipv4: null,
      ipv6: [],
      dns: [],
      timing: null,
    },
    loading: false,
  },
};

/**
 * Link-local IPv6 only.
 * Shows device with only auto-configured link-local address.
 */
export const LinkLocalOnly: Story = {
  args: {
    data: {
      mac: "aa:bb:cc:dd:ee:ff",
      mode: "auto",
      ipv4: null,
      ipv6: [
        {
          address: "fe80::a12:34ff:fe56:7890",
          prefix: 64,
          scope: "link-local",
          source: "slaac",
        },
      ],
      dns: [],
      timing: null,
    },
    loading: false,
  },
};

/**
 * Loading state while fetching configuration.
 * Shows loading indicators.
 */
export const Loading: Story = {
  args: {
    data: null,
    loading: true,
  },
};

/**
 * Long lease time.
 * Shows DHCP configuration with extended lease period.
 */
export const LongLease: Story = {
  args: {
    data: {
      mac: "aa:bb:cc:dd:ee:ff",
      mode: "dhcp",
      ipv4: {
        address: "10.20.30.40",
        subnet: "22",
        gateway: "10.20.28.1",
        dhcpServer: "10.20.28.1",
        leaseTime: 604800,
      },
      ipv6: [],
      dns: ["10.20.28.1", "10.20.28.2"],
      timing: {
        discover: 38,
        offer: 42,
        request: 35,
        ack: 40,
        total: 155,
      },
    },
    loading: false,
  },
};
