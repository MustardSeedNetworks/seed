import type { Meta, StoryObj } from '@storybook/react-vite';
import { PublicIPCard } from './PublicIPCard';

/**
 * PublicIPCard displays public IPv4 and IPv6 addresses as seen from the internet.
 *
 * Features:
 * - Dual-stack support: shows both IPv4 and IPv6 public addresses
 * - Last checked timestamp with human-readable formatting
 * - Error handling for lookup failures
 * - Status indication: success (has IP), error (lookup failed), unknown (no data)
 * - Useful for verifying NAT configuration and external connectivity
 *
 * This story demonstrates various public IP detection states.
 */
const meta = {
  title: 'Cards/PublicIPCard',
  component: PublicIPCard,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <div style={{ width: '380px' }}>
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof PublicIPCard>;

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * IPv4 only configuration.
 * Shows public IPv4 address without IPv6.
 */
export const IPv4Only: Story = {
  args: {
    data: {
      ipv4: '203.0.113.42',
      lastChecked: new Date(Date.now() - 120000).toISOString(),
    },
    loading: false,
  },
};

/**
 * IPv6 only configuration.
 * Shows public IPv6 address without IPv4.
 */
export const IPv6Only: Story = {
  args: {
    data: {
      ipv6: '2001:0db8:85a3:0000:0000:8a2e:0370:7334',
      lastChecked: new Date(Date.now() - 180000).toISOString(),
    },
    loading: false,
  },
};

/**
 * Dual-stack configuration.
 * Shows both IPv4 and IPv6 public addresses.
 */
export const DualStack: Story = {
  args: {
    data: {
      ipv4: '198.51.100.123',
      ipv6: '2001:db8::1',
      lastChecked: new Date(Date.now() - 60000).toISOString(),
    },
    loading: false,
  },
};

/**
 * Recently checked.
 * Shows "just now" for timestamp within last minute.
 */
export const JustChecked: Story = {
  args: {
    data: {
      ipv4: '192.0.2.1',
      ipv6: '2001:db8::42',
      lastChecked: new Date(Date.now() - 30000).toISOString(),
    },
    loading: false,
  },
};

/**
 * Checked hours ago.
 * Shows hour-based timestamp formatting.
 */
export const CheckedHoursAgo: Story = {
  args: {
    data: {
      ipv4: '203.0.113.5',
      lastChecked: new Date(Date.now() - 7200000).toISOString(),
    },
    loading: false,
  },
};

/**
 * Checked days ago.
 * Shows full date when checked more than 24 hours ago.
 */
export const CheckedDaysAgo: Story = {
  args: {
    data: {
      ipv4: '198.51.100.50',
      ipv6: '2001:db8:cafe::1',
      lastChecked: new Date(Date.now() - 172800000).toISOString(),
    },
    loading: false,
  },
};

/**
 * Lookup error.
 * Shows error message when unable to detect public IP.
 */
export const LookupError: Story = {
  args: {
    data: {
      lastChecked: new Date().toISOString(),
      error: 'Unable to contact IP lookup service',
    },
    loading: false,
  },
};

/**
 * Network unreachable error.
 * Shows specific error for no internet connectivity.
 */
export const NetworkUnreachable: Story = {
  args: {
    data: {
      lastChecked: new Date().toISOString(),
      error: 'No internet connectivity',
    },
    loading: false,
  },
};

/**
 * Partial success with error.
 * Shows IPv4 available but IPv6 lookup failed.
 */
export const PartialSuccess: Story = {
  args: {
    data: {
      ipv4: '203.0.113.99',
      lastChecked: new Date().toISOString(),
      error: 'IPv6 lookup failed',
    },
    loading: false,
  },
};

/**
 * Loading state.
 * Shows loading indicator while checking public IP.
 */
export const Loading: Story = {
  args: {
    data: null,
    loading: true,
  },
};

/**
 * No data available.
 * Initial state before any lookup has been performed.
 */
export const NoData: Story = {
  args: {
    data: null,
    loading: false,
  },
};

/**
 * Compressed IPv6 address.
 * Shows IPv6 with zero compression (::).
 */
export const CompressedIPv6: Story = {
  args: {
    data: {
      ipv4: '192.0.2.42',
      ipv6: '2001:db8::42',
      lastChecked: new Date(Date.now() - 300000).toISOString(),
    },
    loading: false,
  },
};

/**
 * Link-local fallback.
 * Shows when only link-local IPv6 is available (not routable).
 */
export const LinkLocalIPv6: Story = {
  args: {
    data: {
      ipv4: '198.51.100.77',
      ipv6: 'fe80::1',
      lastChecked: new Date().toISOString(),
    },
    loading: false,
  },
};

/**
 * Behind CGNAT.
 * Shows carrier-grade NAT address (100.64.0.0/10).
 */
export const BehindCGNAT: Story = {
  args: {
    data: {
      ipv4: '100.96.0.42',
      lastChecked: new Date().toISOString(),
      error: 'You may be behind carrier-grade NAT',
    },
    loading: false,
  },
};
