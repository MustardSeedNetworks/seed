import type { Meta, StoryObj } from "@storybook/react-vite";
import { SystemHealthCard } from "./system-health-card";

/**
 * SystemHealthCard monitors system resources and displays health metrics.
 *
 * Features:
 * - CPU usage percentage with visual progress bar
 * - Memory usage with used/total display
 * - Disk usage with used/total display
 * - Load averages (1, 5, 15 minute)
 * - System uptime in human-readable format
 * - Process memory usage for The Seed itself
 * - Goroutine count for Go runtime monitoring
 * - System info: hostname, OS, architecture, CPU count
 * - Color-coded status bars: green (< 75%), yellow (75-90%), red (> 90%)
 *
 * This story demonstrates various system health states.
 */
const meta = {
  title: "Cards/system-health-card",
  component: SystemHealthCard,
  parameters: {
    layout: "centered",
    mockData: [
      {
        url: "/api/v1/sap/system/health",
        method: "GET",
        status: 200,
        response: {
          cpuPercent: 25.5,
          memoryPercent: 45.2,
          memoryUsed: 4831838208,
          memoryTotal: 10737418240,
          diskPercent: 62.8,
          diskUsed: 314572800000,
          diskTotal: 500107862016,
          uptime: 432000,
          loadAvg1: 1.5,
          loadAvg5: 1.2,
          loadAvg15: 0.9,
          goroutines: 42,
          processMemory: 52428800,
          hostname: "seed-server",
          os: "linux",
          arch: "amd64",
          numCpu: 4,
        },
      },
    ],
  },
  tags: ["autodocs"],
  decorators: [
    (Story) => (
      <div style={{ width: "400px" }}>
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof SystemHealthCard>;

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Healthy system with low resource usage.
 * All metrics show green status indicators.
 */
export const Healthy: Story = {};

/**
 * System under moderate load.
 * Shows warning-level resource usage (yellow indicators).
 */
export const ModerateLoad: Story = {
  parameters: {
    mockData: [
      {
        url: "/api/v1/sap/system/health",
        method: "GET",
        status: 200,
        response: {
          cpuPercent: 78.3,
          memoryPercent: 82.5,
          memoryUsed: 8858370048,
          memoryTotal: 10737418240,
          diskPercent: 76.2,
          diskUsed: 381082240000,
          diskTotal: 500107862016,
          uptime: 1296000,
          loadAvg1: 3.8,
          loadAvg5: 3.2,
          loadAvg15: 2.5,
          goroutines: 156,
          processMemory: 104857600,
          hostname: "seed-server",
          os: "linux",
          arch: "amd64",
          numCpu: 4,
        },
      },
    ],
  },
};

/**
 * System experiencing high load.
 * Critical resource usage with red indicators.
 */
export const HighLoad: Story = {
  parameters: {
    mockData: [
      {
        url: "/api/v1/sap/system/health",
        method: "GET",
        status: 200,
        response: {
          cpuPercent: 94.7,
          memoryPercent: 96.1,
          memoryUsed: 10316865536,
          memoryTotal: 10737418240,
          diskPercent: 91.8,
          diskUsed: 459099120000,
          diskTotal: 500107862016,
          uptime: 86400,
          loadAvg1: 7.2,
          loadAvg5: 6.8,
          loadAvg15: 5.9,
          goroutines: 423,
          processMemory: 209715200,
          hostname: "seed-server",
          os: "linux",
          arch: "amd64",
          numCpu: 4,
        },
      },
    ],
  },
};

/**
 * Disk space critically low.
 * Shows error state with very high disk usage.
 */
export const DiskSpaceCritical: Story = {
  parameters: {
    mockData: [
      {
        url: "/api/v1/sap/system/health",
        method: "GET",
        status: 200,
        response: {
          cpuPercent: 15.2,
          memoryPercent: 38.4,
          memoryUsed: 4123168768,
          memoryTotal: 10737418240,
          diskPercent: 98.5,
          diskUsed: 492606254080,
          diskTotal: 500107862016,
          uptime: 259200,
          loadAvg1: 0.8,
          loadAvg5: 0.6,
          loadAvg15: 0.5,
          goroutines: 38,
          processMemory: 41943040,
          hostname: "seed-server",
          os: "linux",
          arch: "amd64",
          numCpu: 4,
        },
      },
    ],
  },
};

/**
 * macOS system.
 * Shows system health on macOS platform.
 */
export const Macos: Story = {
  parameters: {
    mockData: [
      {
        url: "/api/v1/sap/system/health",
        method: "GET",
        status: 200,
        response: {
          cpuPercent: 32.1,
          memoryPercent: 52.3,
          memoryUsed: 8589934592,
          memoryTotal: 16416522240,
          diskPercent: 55.7,
          diskUsed: 556417433600,
          diskTotal: 1000204886016,
          uptime: 1728000,
          loadAvg1: 2.1,
          loadAvg5: 1.8,
          loadAvg15: 1.5,
          goroutines: 67,
          processMemory: 83886080,
          hostname: "macbook-pro",
          os: "darwin",
          arch: "arm64",
          numCpu: 8,
        },
      },
    ],
  },
};

/**
 * Windows system.
 * Shows system health on Windows platform.
 */
export const Windows: Story = {
  parameters: {
    mockData: [
      {
        url: "/api/v1/sap/system/health",
        method: "GET",
        status: 200,
        response: {
          cpuPercent: 28.5,
          memoryPercent: 48.9,
          memoryUsed: 8053063680,
          memoryTotal: 16458113024,
          diskPercent: 68.3,
          diskUsed: 683035238400,
          diskTotal: 1000204886016,
          uptime: 604800,
          loadAvg1: 0,
          loadAvg5: 0,
          loadAvg15: 0,
          goroutines: 52,
          processMemory: 62914560,
          hostname: "WIN-SERVER-01",
          os: "windows",
          arch: "amd64",
          numCpu: 6,
        },
      },
    ],
  },
};

/**
 * Recently booted system.
 * Shows low uptime (just minutes).
 */
export const RecentlyBooted: Story = {
  parameters: {
    mockData: [
      {
        url: "/api/v1/sap/system/health",
        method: "GET",
        status: 200,
        response: {
          cpuPercent: 18.3,
          memoryPercent: 28.7,
          memoryUsed: 3081510912,
          memoryTotal: 10737418240,
          diskPercent: 62.8,
          diskUsed: 314572800000,
          diskTotal: 500107862016,
          uptime: 420,
          loadAvg1: 0.5,
          loadAvg5: 0.3,
          loadAvg15: 0.1,
          goroutines: 34,
          processMemory: 31457280,
          hostname: "seed-server",
          os: "linux",
          arch: "amd64",
          numCpu: 4,
        },
      },
    ],
  },
};

/**
 * Long-running system.
 * Shows high uptime (many days).
 */
export const LongUptime: Story = {
  parameters: {
    mockData: [
      {
        url: "/api/v1/sap/system/health",
        method: "GET",
        status: 200,
        response: {
          cpuPercent: 22.1,
          memoryPercent: 64.5,
          memoryUsed: 6925950976,
          memoryTotal: 10737418240,
          diskPercent: 71.3,
          diskUsed: 356576870400,
          diskTotal: 500107862016,
          uptime: 5184000,
          loadAvg1: 1.2,
          loadAvg5: 1.0,
          loadAvg15: 0.8,
          goroutines: 58,
          processMemory: 73400320,
          hostname: "seed-server",
          os: "linux",
          arch: "amd64",
          numCpu: 4,
        },
      },
    ],
  },
};

/**
 * High goroutine count.
 * Shows system with many concurrent operations.
 */
export const HighGoroutines: Story = {
  parameters: {
    mockData: [
      {
        url: "/api/v1/sap/system/health",
        method: "GET",
        status: 200,
        response: {
          cpuPercent: 45.8,
          memoryPercent: 58.2,
          memoryUsed: 6249486336,
          memoryTotal: 10737418240,
          diskPercent: 62.8,
          diskUsed: 314572800000,
          diskTotal: 500107862016,
          uptime: 864000,
          loadAvg1: 2.8,
          loadAvg5: 2.3,
          loadAvg15: 1.9,
          goroutines: 892,
          processMemory: 157286400,
          hostname: "seed-server",
          os: "linux",
          arch: "amd64",
          numCpu: 4,
        },
      },
    ],
  },
};

/**
 * Loading state while fetching health data.
 * Shows skeleton/loading indicators.
 */
export const Loading: Story = {
  parameters: {
    mockData: [
      {
        url: "/api/v1/sap/system/health",
        method: "GET",
        delay: 999999,
        status: 200,
        response: {},
      },
    ],
  },
};
