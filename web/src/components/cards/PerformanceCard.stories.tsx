import type { Meta, StoryObj } from "@storybook/react-vite";
import { PerformanceCard } from "./PerformanceCard";

/**
 * PerformanceCard displays internet speed testing (speedtest.net) and LAN speed testing (iperf3).
 *
 * Features:
 * - Speedtest.net integration with download/upload speeds, latency, and server info
 * - iPerf3 support for LAN testing with TCP and UDP protocols
 * - Real-time progress indicators during tests
 * - Visual speed gauges for speedtest results
 * - Detailed metrics including jitter and packet loss (iperf3 UDP)
 * - Server mode support for accepting iperf3 connections
 *
 * Note: SettingsProvider and I18nextProvider are provided by global decorators
 * in .storybook/preview.tsx.
 */
const meta = {
  title: "Cards/PerformanceCard",
  component: PerformanceCard,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  decorators: [
    (Story) => (
      <div style={{ width: "400px" }}>
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof PerformanceCard>;

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Initial state before any tests have been run.
 * Shows "No results yet" for both speedtest and iperf.
 */
export const NoResults: Story = {
  args: {
    loading: false,
    runSpeedtestEnabled: true,
    runIperfEnabled: true,
  },
};

/**
 * Speedtest in progress showing testing phase and progress.
 * Displays animated progress ring and current phase (e.g., "Testing download...").
 */
export const SpeedtestRunning: Story = {
  args: {
    loading: false,
    runSpeedtestEnabled: true,
    runIperfEnabled: true,
  },
};

/**
 * Successful speedtest results with excellent speeds.
 * Shows download/upload speeds with visual gauges, latency, and server location.
 */
export const SpeedtestSuccess: Story = {
  args: {
    loading: false,
    runSpeedtestEnabled: true,
    runIperfEnabled: true,
  },
};

/**
 * Speedtest results showing slower connection speeds.
 * Demonstrates how the card displays lower-tier broadband speeds.
 */
export const SpeedtestModerateSpeed: Story = {
  args: {
    loading: false,
    runSpeedtestEnabled: true,
    runIperfEnabled: true,
  },
};

/**
 * Speedtest error state when test fails.
 * Shows error message explaining why the test failed.
 */
export const SpeedtestError: Story = {
  args: {
    loading: false,
    runSpeedtestEnabled: true,
    runIperfEnabled: true,
  },
};

/**
 * iPerf3 test in progress.
 * Shows progress indicator and current test phase.
 */
export const IperfRunning: Story = {
  args: {
    loading: false,
    runSpeedtestEnabled: true,
    runIperfEnabled: true,
  },
};

/**
 * Successful iperf3 TCP test results showing LAN speeds.
 * Displays bandwidth, transfer amount, and retransmit count.
 */
export const IperfTCPSuccess: Story = {
  args: {
    loading: false,
    runSpeedtestEnabled: true,
    runIperfEnabled: true,
  },
};

/**
 * Successful iperf3 UDP test results.
 * Shows bandwidth, jitter, and packet loss metrics.
 */
export const IperfUDPSuccess: Story = {
  args: {
    loading: false,
    runSpeedtestEnabled: true,
    runIperfEnabled: true,
  },
};

/**
 * iPerf3 bidirectional test results.
 * Displays separate download and upload bandwidth measurements.
 */
export const IperfBidirectional: Story = {
  args: {
    loading: false,
    runSpeedtestEnabled: true,
    runIperfEnabled: true,
  },
};

/**
 * iPerf3 not installed on the system.
 * Shows warning message prompting user to install iperf3.
 */
export const IperfNotInstalled: Story = {
  args: {
    loading: false,
    runSpeedtestEnabled: true,
    runIperfEnabled: true,
  },
};

/**
 * iPerf3 server mode active and listening.
 * Shows server status with listening port number.
 */
export const IperfServerRunning: Story = {
  args: {
    loading: false,
    runSpeedtestEnabled: true,
    runIperfEnabled: true,
  },
};

/**
 * Both speedtest and iperf results available.
 * Demonstrates card with complete test data from both testing methods.
 */
export const AllResultsAvailable: Story = {
  args: {
    loading: false,
    runSpeedtestEnabled: true,
    runIperfEnabled: true,
  },
};

/**
 * Performance tests disabled in settings.
 * Shows appropriate message when tests are turned off.
 */
export const TestsDisabled: Story = {
  args: {
    loading: false,
    runSpeedtestEnabled: false,
    runIperfEnabled: false,
  },
};

/**
 * iPerf3 server not configured.
 * Shows message prompting user to configure server in settings.
 */
export const IperfNotConfigured: Story = {
  args: {
    loading: false,
    runSpeedtestEnabled: true,
    runIperfEnabled: true,
  },
};
