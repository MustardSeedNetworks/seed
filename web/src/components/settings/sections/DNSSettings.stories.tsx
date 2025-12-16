/**
 * DNSSettings Storybook Stories
 *
 * Demonstrates the DNS settings component for configuring DNS test hostname
 * and additional DNS servers for comparison testing.
 *
 * Variants:
 * - Default hostname: Standard google.com configuration
 * - Custom hostname: User-specified test target
 * - No DNS servers: Empty server list
 * - Multiple DNS servers: Several servers configured for testing
 * - Popular DNS providers: Google, Cloudflare, Quad9, etc.
 * - Interactive CRUD: Add/remove servers
 */

import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { DNSSettings } from "./DNSSettings";
import type { TestsSettings, SaveStatus } from "../../../types/settings";
import { spacing } from "../../../styles/theme";

const baseSettings: Omit<TestsSettings, "dnsHostname" | "dnsServers"> = {
  pingTargets: [],
  tcpPorts: [],
  udpPorts: [],
  httpEndpoints: [],
  runPerformance: true,
  runSpeedtest: true,
  runIperf: true,
  runDiscovery: true,
  speedtest: {
    serverId: "",
    autoRunOnLink: false,
  },
  iperf: {
    autoRunOnLink: false,
  },
};

const meta: Meta<typeof DNSSettings> = {
  title: "Settings/DNSSettings",
  component: DNSSettings,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "DNS configuration panel for setting test hostname and managing additional DNS servers for response time comparison. Supports adding/removing servers dynamically.",
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
    (Story) => (
      <div className="w-[450px]">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default configuration - google.com, no additional servers
 */
export const Default: Story = {
  args: {
    testsSettings: {
      ...baseSettings,
      dnsHostname: "google.com",
      dnsServers: [],
    },
    setTestsSettings: () => {},
    testsStatus: "idle",
  },
};

/**
 * Custom hostname - user-specified test target
 */
export const CustomHostname: Story = {
  args: {
    testsSettings: {
      ...baseSettings,
      dnsHostname: "example.com",
      dnsServers: [],
    },
    setTestsSettings: () => {},
    testsStatus: "idle",
  },
};

/**
 * With popular public DNS servers configured
 */
export const WithPopularDNS: Story = {
  args: {
    testsSettings: {
      ...baseSettings,
      dnsHostname: "google.com",
      dnsServers: [
        { id: "1", address: "8.8.8.8", enabled: true },
        { id: "2", address: "8.8.4.4", enabled: true },
        { id: "3", address: "1.1.1.1", enabled: true },
        { id: "4", address: "1.0.0.1", enabled: true },
      ],
    },
    setTestsSettings: () => {},
    testsStatus: "idle",
  },
};

/**
 * Multiple DNS providers for comparison
 */
export const MultipleDNSProviders: Story = {
  args: {
    testsSettings: {
      ...baseSettings,
      dnsHostname: "example.com",
      dnsServers: [
        { id: "1", address: "8.8.8.8", enabled: true }, // Google
        { id: "2", address: "1.1.1.1", enabled: true }, // Cloudflare
        { id: "3", address: "9.9.9.9", enabled: true }, // Quad9
        { id: "4", address: "208.67.222.222", enabled: true }, // OpenDNS
        { id: "5", address: "8.26.56.26", enabled: true }, // Comodo Secure DNS
      ],
    },
    setTestsSettings: () => {},
    testsStatus: "idle",
  },
};

/**
 * Single DNS server configured
 */
export const SingleServer: Story = {
  args: {
    testsSettings: {
      ...baseSettings,
      dnsHostname: "google.com",
      dnsServers: [{ id: "1", address: "1.1.1.1", enabled: true }],
    },
    setTestsSettings: () => {},
    testsStatus: "idle",
  },
};

/**
 * Empty DNS server list - ready to add servers
 */
export const EmptyServerList: Story = {
  args: {
    testsSettings: {
      ...baseSettings,
      dnsHostname: "google.com",
      dnsServers: [],
    },
    setTestsSettings: () => {},
    testsStatus: "idle",
  },
};

/**
 * Saving state
 */
export const Saving: Story = {
  args: {
    testsSettings: {
      ...baseSettings,
      dnsHostname: "google.com",
      dnsServers: [
        { id: "1", address: "8.8.8.8", enabled: true },
        { id: "2", address: "1.1.1.1", enabled: true },
      ],
    },
    setTestsSettings: () => {},
    testsStatus: "saving",
  },
};

/**
 * Saved state
 */
export const Saved: Story = {
  args: {
    testsSettings: {
      ...baseSettings,
      dnsHostname: "example.com",
      dnsServers: [
        { id: "1", address: "8.8.8.8", enabled: true },
        { id: "2", address: "1.1.1.1", enabled: true },
      ],
    },
    setTestsSettings: () => {},
    testsStatus: "saved",
  },
};

/**
 * Interactive DNS settings - fully functional CRUD operations
 */
export const Interactive: Story = {
  render: function InteractiveStory() {
    const [testsSettings, setTestsSettings] = useState<TestsSettings>({
      ...baseSettings,
      dnsHostname: "google.com",
      dnsServers: [
        { id: "1", address: "8.8.8.8", enabled: true },
        { id: "2", address: "1.1.1.1", enabled: true },
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
      <DNSSettings
        testsSettings={testsSettings}
        setTestsSettings={handleSetTestsSettings}
        testsStatus={status}
      />
    );
  },
};

/**
 * Comparison of different configurations
 */
export const Comparison: Story = {
  render: () => (
    <div className={`stack-lg ${spacing.pad.default}`}>
      <div>
        <p className={`caption text-text-muted ${spacing.margin.bottom.inline}`}>
          Default (no servers)
        </p>
        <DNSSettings
          testsSettings={{
            ...baseSettings,
            dnsHostname: "google.com",
            dnsServers: [],
          }}
          setTestsSettings={() => {}}
          testsStatus="idle"
        />
      </div>
      <div>
        <p className={`caption text-text-muted ${spacing.margin.bottom.inline}`}>
          With DNS servers
        </p>
        <DNSSettings
          testsSettings={{
            ...baseSettings,
            dnsHostname: "example.com",
            dnsServers: [
              { id: "1", address: "8.8.8.8", enabled: true },
              { id: "2", address: "1.1.1.1", enabled: true },
            ],
          }}
          setTestsSettings={() => {}}
          testsStatus="idle"
        />
      </div>
      <div>
        <p className={`caption text-text-muted ${spacing.margin.bottom.inline}`}>Saving state</p>
        <DNSSettings
          testsSettings={{
            ...baseSettings,
            dnsHostname: "google.com",
            dnsServers: [{ id: "1", address: "8.8.8.8", enabled: true }],
          }}
          setTestsSettings={() => {}}
          testsStatus="saving"
        />
      </div>
    </div>
  ),
};
