/**
 * HealthChecksSettings Storybook Stories
 *
 * Demonstrates the health checks configuration component for managing ping targets,
 * TCP/UDP ports, and HTTP endpoints for continuous monitoring.
 *
 * Variants:
 * - Empty configuration: No tests configured
 * - All test types configured: Ping, TCP, UDP, HTTP
 * - Individual test types: Each test category shown separately
 * - Production monitoring: Real-world monitoring example
 * - Interactive CRUD: Add/remove/edit tests
 */

import type { Meta, StoryFn, StoryObj } from "@storybook/react-vite";
import type React from "react";
import { useState } from "react";
import type { SaveStatus, TestsSettings } from "../../../types/settings";
import { HealthChecksSettings } from "./HealthChecksSettings";

const emptySettings: TestsSettings = {
  dnsHostname: "google.com",
  dnsServers: [],
  pingTargets: [],
  tcpPorts: [],
  udpPorts: [],
  httpEndpoints: [],
  runPerformance: true,
  runSpeedtest: true,
  runIperf: true,
  runDiscovery: true,
  speedtest: { serverId: "", autoRunOnLink: false },
  iperf: { autoRunOnLink: false },
};

const meta: Meta<typeof HealthChecksSettings> = {
  title: "Settings/health-checks-settings",
  component: HealthChecksSettings,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Comprehensive health check configuration panel. Allows users to define ping targets, TCP/UDP port tests, and HTTP endpoint monitoring with custom names and expected results.",
      },
    },
  },
  tags: ["autodocs"],
  argTypes: {
    testsStatus: {
      control: "select",
      options: ["idle", "saving", "saved", "error"],
      description: "Auto-save status indicator",
    },
  },
  decorators: [
    (StoryComponent: StoryFn): React.ReactElement => (
      <div class="w-[500px] max-h-[700px] overflow-y-auto">
        <StoryComponent />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Empty configuration - no health checks configured
 */
export const Empty: Story = {
  args: {
    testsSettings: emptySettings,
    setTestsSettings: (): void => {
      // intentionally empty
    },
    testsStatus: "idle",
  },
};

/**
 * Disabled health checks
 */
export const Disabled: Story = {
  args: {
    testsSettings: {
      ...emptySettings,
      runPerformance: false,
    },
    setTestsSettings: (): void => {
      // intentionally empty
    },
    testsStatus: "idle",
  },
};

/**
 * Only ping targets configured
 */
export const OnlyPingTargets: Story = {
  args: {
    testsSettings: {
      ...emptySettings,
      pingTargets: [
        {
          id: "1",
          name: "Gateway",
          host: "192.168.1.1",
          enabled: true,
          count: 3,
        },
        {
          id: "2",
          name: "DNS",
          host: "8.8.8.8",
          enabled: true,
          count: 3,
        },
        {
          id: "3",
          name: "Cloudflare",
          host: "1.1.1.1",
          enabled: true,
          count: 3,
        },
      ],
    },
    setTestsSettings: (): void => {
      // intentionally empty
    },
    testsStatus: "idle",
  },
};

/**
 * Only TCP port tests configured
 */
export const OnlyTcpPorts: Story = {
  args: {
    testsSettings: {
      ...emptySettings,
      tcpPorts: [
        {
          id: "1",
          name: "HTTP",
          host: "192.168.1.100",
          port: 80,
          enabled: true,
        },
        {
          id: "2",
          name: "HTTPS",
          host: "192.168.1.100",
          port: 443,
          enabled: true,
        },
        {
          id: "3",
          name: "SSH",
          host: "192.168.1.100",
          port: 22,
          enabled: true,
        },
      ],
    },
    setTestsSettings: (): void => {
      // intentionally empty
    },
    testsStatus: "idle",
  },
};

/**
 * Only UDP port tests configured
 */
export const OnlyUdpPorts: Story = {
  args: {
    testsSettings: {
      ...emptySettings,
      udpPorts: [
        { id: "1", name: "DNS", host: "8.8.8.8", port: 53, enabled: true },
        {
          id: "2",
          name: "NTP",
          host: "pool.ntp.org",
          port: 123,
          enabled: true,
        },
      ],
    },
    setTestsSettings: (): void => {
      // intentionally empty
    },
    testsStatus: "idle",
  },
};

/**
 * Only HTTP endpoints configured
 */
export const OnlyHttpEndpoints: Story = {
  args: {
    testsSettings: {
      ...emptySettings,
      httpEndpoints: [
        {
          id: "1",
          name: "API Health",
          url: "https://api.example.com/health",
          expectedStatus: 200,
          enabled: true,
        },
        {
          id: "2",
          name: "Website",
          url: "https://example.com",
          expectedStatus: 200,
          enabled: true,
        },
      ],
    },
    setTestsSettings: (): void => {
      // intentionally empty
    },
    testsStatus: "idle",
  },
};

/**
 * All test types configured - comprehensive monitoring
 */
export const AllTestTypes: Story = {
  args: {
    testsSettings: {
      ...emptySettings,
      pingTargets: [
        {
          id: "1",
          name: "Gateway",
          host: "192.168.1.1",
          enabled: true,
          count: 3,
        },
        { id: "2", name: "DNS", host: "8.8.8.8", enabled: true, count: 3 },
      ],
      tcpPorts: [
        {
          id: "1",
          name: "Web",
          host: "192.168.1.100",
          port: 80,
          enabled: true,
        },
        {
          id: "2",
          name: "SSH",
          host: "192.168.1.100",
          port: 22,
          enabled: true,
        },
      ],
      udpPorts: [{ id: "1", name: "DNS", host: "8.8.8.8", port: 53, enabled: true }],
      httpEndpoints: [
        {
          id: "1",
          name: "API",
          url: "https://api.example.com/health",
          expectedStatus: 200,
          enabled: true,
        },
      ],
    },
    setTestsSettings: (): void => {
      // intentionally empty
    },
    testsStatus: "idle",
  },
};

/**
 * Production monitoring setup - real-world example
 */
export const ProductionMonitoring: Story = {
  args: {
    testsSettings: {
      ...emptySettings,
      pingTargets: [
        {
          id: "1",
          name: "Core Router",
          host: "10.0.0.1",
          enabled: true,
          count: 5,
        },
        {
          id: "2",
          name: "Primary DNS",
          host: "10.0.1.10",
          enabled: true,
          count: 3,
        },
        {
          id: "3",
          name: "Secondary DNS",
          host: "10.0.1.11",
          enabled: true,
          count: 3,
        },
      ],
      tcpPorts: [
        {
          id: "1",
          name: "Web Server",
          host: "web.example.com",
          port: 443,
          enabled: true,
        },
        {
          id: "2",
          name: "DB Primary",
          host: "db1.internal",
          port: 5432,
          enabled: true,
        },
        {
          id: "3",
          name: "Redis Cache",
          host: "cache.internal",
          port: 6379,
          enabled: true,
        },
      ],
      udpPorts: [
        {
          id: "1",
          name: "Syslog",
          host: "syslog.internal",
          port: 514,
          enabled: true,
        },
      ],
      httpEndpoints: [
        {
          id: "1",
          name: "API Health",
          url: "https://api.example.com/health",
          expectedStatus: 200,
          enabled: true,
        },
        {
          id: "2",
          name: "Status Page",
          url: "https://status.example.com",
          expectedStatus: 200,
          enabled: true,
        },
        {
          id: "3",
          name: "Monitoring",
          url: "https://monitoring.example.com/ping",
          expectedStatus: 200,
          enabled: true,
        },
      ],
    },
    setTestsSettings: (): void => {
      // intentionally empty
    },
    testsStatus: "idle",
  },
};

/**
 * Saving state
 */
export const Saving: Story = {
  args: {
    testsSettings: {
      ...emptySettings,
      pingTargets: [
        {
          id: "1",
          name: "Gateway",
          host: "192.168.1.1",
          enabled: true,
          count: 3,
        },
      ],
    },
    setTestsSettings: (): void => {
      // intentionally empty
    },
    testsStatus: "saving",
  },
};

/**
 * Interactive health checks - fully functional CRUD
 */
export const Interactive: Story = {
  render: function interactiveStory() {
    const [testsSettings, setTestsSettings] = useState<TestsSettings>({
      ...emptySettings,
      pingTargets: [
        {
          id: "1",
          name: "Gateway",
          host: "192.168.1.1",
          enabled: true,
          count: 3,
        },
      ],
      tcpPorts: [
        {
          id: "1",
          name: "HTTP",
          host: "192.168.1.100",
          port: 80,
          enabled: true,
        },
      ],
    });
    const [status, setStatus] = useState<SaveStatus>("idle");

    const handleSetTestsSettings = (updater: React.SetStateAction<TestsSettings>) => {
      setTestsSettings(updater);
      setStatus("saving");

      setTimeout(() => {
        setStatus("saved");
        setTimeout(() => {
          setStatus("idle");
        }, 2000);
      }, 800);
    };

    return (
      <HealthChecksSettings
        testsSettings={testsSettings}
        setTestsSettings={handleSetTestsSettings}
        testsStatus={status}
      />
    );
  },
};
