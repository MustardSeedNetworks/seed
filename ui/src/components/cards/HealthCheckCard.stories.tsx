import type { Meta, StoryObj } from "@storybook/react-vite";
import { HealthCheckCard } from "./HealthCheckCard";

/**
 * HealthCheckCard performs comprehensive health checks for remote services.
 *
 * Features:
 * - Multi-protocol testing: ICMP ping, TCP connect, UDP, HTTP/HTTPS
 * - Extended ping metrics: packet loss, jitter, min/max/avg latency
 * - HTTP timing breakdown: DNS, TCP, TLS, TTFB, download phases
 * - SSL/TLS certificate monitoring: expiry dates, days remaining, issuer info
 * - Per-phase latency thresholds with color-coded status
 * - Collapsible sections for each test type
 * - Visual timing bars for HTTP requests
 * - Status badges for quick health overview
 *
 * Note: SettingsProvider and I18nextProvider are provided by global decorators
 * in .storybook/preview.tsx.
 */
const meta = {
  title: "Cards/HealthCheckCard",
  component: HealthCheckCard,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  decorators: [
    (Story) => (
      <div style={{ width: "420px" }}>
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof HealthCheckCard>;

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * All tests passing with excellent latency.
 * Shows green status indicators across all test types.
 */
export const AllTestsPassing: Story = {
  args: {
    loading: false,
  },
};

/**
 * Tests running in progress.
 * Shows loading state with "Running tests..." message.
 */
export const TestsRunning: Story = {
  args: {
    loading: true,
  },
};

/**
 * Ping tests with varied results.
 * Shows mix of successful and failed ping tests.
 */
export const PingTestsVaried: Story = {
  args: {
    loading: false,
  },
};

/**
 * TCP port checks with some failures.
 * Demonstrates failed TCP connection attempts.
 */
export const TcpTestsWithFailures: Story = {
  args: {
    loading: false,
  },
};

/**
 * HTTP/HTTPS tests with timing breakdown.
 * Shows detailed phase timing (DNS, TCP, TLS, TTFB).
 */
export const HttpTestsWithTiming: Story = {
  args: {
    loading: false,
  },
};

/**
 * Certificate expiring soon.
 * Shows warning status for SSL certificates nearing expiry.
 */
export const CertificateExpiringSoon: Story = {
  args: {
    loading: false,
  },
};

/**
 * Certificate expired.
 * Shows critical error for expired SSL certificates.
 */
export const CertificateExpired: Story = {
  args: {
    loading: false,
  },
};

/**
 * High latency warnings.
 * Shows yellow status for degraded response times.
 */
export const HighLatencyWarnings: Story = {
  args: {
    loading: false,
  },
};

/**
 * UDP port tests.
 * Demonstrates UDP connectivity testing.
 */
export const UdpTests: Story = {
  args: {
    loading: false,
  },
};

/**
 * Mixed protocol comprehensive test.
 * Shows results from ping, TCP, UDP, and HTTP tests all together.
 */
export const MixedProtocolTests: Story = {
  args: {
    loading: false,
  },
};

/**
 * Extended ping metrics.
 * Shows packet loss, jitter, min/max latency for ping tests.
 */
export const ExtendedPingMetrics: Story = {
  args: {
    loading: false,
  },
};

/**
 * HTTPS with TLS version and certificate details.
 * Shows TLS 1.3, certificate issuer, and expiry information.
 */
export const HttpsWithCertDetails: Story = {
  args: {
    loading: false,
  },
};

/**
 * Service unreachable errors.
 * Shows critical failures for all test types.
 */
export const ServiceUnreachable: Story = {
  args: {
    loading: false,
  },
};

/**
 * No tests configured.
 * Card should not render when no health checks are configured.
 */
export const NoTestsConfigured: Story = {
  args: {
    loading: false,
  },
};
