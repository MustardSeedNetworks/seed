/**
 * DiscoverySettings Storybook Stories
 *
 * Demonstrates the network discovery configuration component with multiple
 * discovery profiles and customizable scan options.
 *
 * Variants:
 * - Discovery profiles: Stealth, Standard, Full Scan, Custom
 * - Service status: Running, stopped, scanning
 * - With subnets: Additional target networks configured
 * - Custom options: Granular control over discovery methods
 * - Timing settings: Workers, timeouts, intervals
 */

import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { DiscoverySettings } from "./DiscoverySettings";
import type {
  NetworkDiscoverySettings,
  SubnetConfig,
  SaveStatus,
} from "../../../types/settings";

const defaultSettings: NetworkDiscoverySettings = {
  enabled: true,
  profile: "standard",
  arpScanWorkers: 50,
  pingTimeoutMs: 500,
  scanTimeoutMs: 30000,
  autoScan: false,
  scanIntervalMs: 0,
  ouiFilePath: "oui.txt",
  customOptions: {
    passiveListen: true,
    arpScan: true,
    icmpScan: true,
    portScan: { enabled: false, ports: [], topPorts: 100 },
    traceroute: false,
    snmpQuery: false,
  },
};

const meta: Meta<typeof DiscoverySettings> = {
  title: "Settings/DiscoverySettings",
  component: DiscoverySettings,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Network discovery configuration panel with multiple profiles (stealth, standard, full scan, custom). Manages scan methods, timing, subnets, and service status monitoring.",
      },
    },
  },
  tags: ["autodocs"],
  argTypes: {
    networkDiscoveryStatus: {
      control: "select",
      options: ["idle", "saving", "saved", "error"],
      description: "Auto-save status indicator",
    },
    subnetsStatus: {
      control: "select",
      options: ["idle", "saving", "saved", "error"],
      description: "Subnet save status",
    },
  },
  decorators: [
    (Story) => (
      <div className="w-[550px] max-h-[700px] overflow-y-auto">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Stealth profile - passive listening only
 */
export const StealthProfile: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      profile: "stealth",
    },
    setNetworkDiscoverySettings: () => {},
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {},
    newSubnetName: "",
    setNewSubnetName: () => {},
    subnetError: null,
    setSubnetError: () => {},
    addSubnet: () => {},
    toggleSubnet: () => {},
    deleteSubnet: () => {},
  },
};

/**
 * Standard profile - ARP and ICMP on local subnet
 */
export const StandardProfile: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      profile: "standard",
    },
    setNetworkDiscoverySettings: () => {},
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {},
    newSubnetName: "",
    setNewSubnetName: () => {},
    subnetError: null,
    setSubnetError: () => {},
    addSubnet: () => {},
    toggleSubnet: () => {},
    deleteSubnet: () => {},
  },
};

/**
 * Full scan profile - all methods including port scanning
 */
export const FullScanProfile: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      profile: "full_scan",
    },
    setNetworkDiscoverySettings: () => {},
    networkDiscoveryStatus: "idle",
    subnets: [
      { cidr: "10.0.0.0/24", name: "Server VLAN", enabled: true },
      { cidr: "172.16.0.0/16", name: "Management", enabled: true },
    ],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {},
    newSubnetName: "",
    setNewSubnetName: () => {},
    subnetError: null,
    setSubnetError: () => {},
    addSubnet: () => {},
    toggleSubnet: () => {},
    deleteSubnet: () => {},
  },
};

/**
 * Custom profile - granular control
 */
export const CustomProfile: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      profile: "custom",
      customOptions: {
        passiveListen: true,
        arpScan: true,
        icmpScan: false,
        portScan: { enabled: true, ports: [22, 80, 443], topPorts: 100 },
        traceroute: true,
        snmpQuery: true,
      },
    },
    setNetworkDiscoverySettings: () => {},
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {},
    newSubnetName: "",
    setNewSubnetName: () => {},
    subnetError: null,
    setSubnetError: () => {},
    addSubnet: () => {},
    toggleSubnet: () => {},
    deleteSubnet: () => {},
  },
};

/**
 * Discovery disabled
 */
export const Disabled: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      enabled: false,
    },
    setNetworkDiscoverySettings: () => {},
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {},
    newSubnetName: "",
    setNewSubnetName: () => {},
    subnetError: null,
    setSubnetError: () => {},
    addSubnet: () => {},
    toggleSubnet: () => {},
    deleteSubnet: () => {},
  },
};

/**
 * Auto-scan enabled with interval
 */
export const AutoScanEnabled: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      autoScan: true,
      scanIntervalMs: 300000, // 5 minutes
    },
    setNetworkDiscoverySettings: () => {},
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {},
    newSubnetName: "",
    setNewSubnetName: () => {},
    subnetError: null,
    setSubnetError: () => {},
    addSubnet: () => {},
    toggleSubnet: () => {},
    deleteSubnet: () => {},
  },
};

/**
 * With multiple subnets configured
 */
export const WithSubnets: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      profile: "full_scan",
    },
    setNetworkDiscoverySettings: () => {},
    networkDiscoveryStatus: "idle",
    subnets: [
      { cidr: "10.0.0.0/24", name: "Server VLAN", enabled: true },
      { cidr: "10.0.1.0/24", name: "IoT Devices", enabled: true },
      { cidr: "172.16.0.0/16", name: "Management Network", enabled: false },
      { cidr: "192.168.100.0/24", name: "Guest WiFi", enabled: true },
    ],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {},
    newSubnetName: "",
    setNewSubnetName: () => {},
    subnetError: null,
    setSubnetError: () => {},
    addSubnet: () => {},
    toggleSubnet: () => {},
    deleteSubnet: () => {},
  },
};

/**
 * Subnet validation error
 */
export const SubnetError: Story = {
  args: {
    networkDiscoverySettings: defaultSettings,
    setNetworkDiscoverySettings: () => {},
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "error",
    newSubnetCidr: "invalid-cidr",
    setNewSubnetCidr: () => {},
    newSubnetName: "",
    setNewSubnetName: () => {},
    subnetError: "Invalid CIDR format",
    setSubnetError: () => {},
    addSubnet: () => {},
    toggleSubnet: () => {},
    deleteSubnet: () => {},
  },
};

/**
 * Custom timing settings - fast scan
 */
export const FastScan: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      arpScanWorkers: 100,
      pingTimeoutMs: 200,
      scanTimeoutMs: 15000,
    },
    setNetworkDiscoverySettings: () => {},
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {},
    newSubnetName: "",
    setNewSubnetName: () => {},
    subnetError: null,
    setSubnetError: () => {},
    addSubnet: () => {},
    toggleSubnet: () => {},
    deleteSubnet: () => {},
  },
};

/**
 * Custom timing settings - slow/thorough scan
 */
export const ThoroughScan: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      arpScanWorkers: 20,
      pingTimeoutMs: 2000,
      scanTimeoutMs: 120000,
    },
    setNetworkDiscoverySettings: () => {},
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {},
    newSubnetName: "",
    setNewSubnetName: () => {},
    subnetError: null,
    setSubnetError: () => {},
    addSubnet: () => {},
    toggleSubnet: () => {},
    deleteSubnet: () => {},
  },
};

/**
 * Saving state
 */
export const Saving: Story = {
  args: {
    networkDiscoverySettings: defaultSettings,
    setNetworkDiscoverySettings: () => {},
    networkDiscoveryStatus: "saving",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {},
    newSubnetName: "",
    setNewSubnetName: () => {},
    subnetError: null,
    setSubnetError: () => {},
    addSubnet: () => {},
    toggleSubnet: () => {},
    deleteSubnet: () => {},
  },
};

/**
 * Interactive discovery settings - fully functional
 */
export const Interactive: Story = {
  render: function InteractiveStory() {
    const [settings, setSettings] =
      useState<NetworkDiscoverySettings>(defaultSettings);
    const [status, setStatus] = useState<SaveStatus>("idle");
    const subnets: SubnetConfig[] = [
      { cidr: "10.0.0.0/24", name: "Server VLAN", enabled: true },
    ];
    const [newCidr, setNewCidr] = useState("");
    const [newName, setNewName] = useState("");
    const [error, setError] = useState<string | null>(null);

    const handleSetSettings = (
      updater: React.SetStateAction<NetworkDiscoverySettings>,
    ) => {
      setSettings(updater);
      setStatus("saving");
      setTimeout(() => {
        setStatus("saved");
        setTimeout(() => setStatus("idle"), 2000);
      }, 800);
    };

    return (
      <DiscoverySettings
        networkDiscoverySettings={settings}
        setNetworkDiscoverySettings={handleSetSettings}
        networkDiscoveryStatus={status}
        subnets={subnets}
        subnetsStatus="idle"
        newSubnetCidr={newCidr}
        setNewSubnetCidr={setNewCidr}
        newSubnetName={newName}
        setNewSubnetName={setNewName}
        subnetError={error}
        setSubnetError={setError}
        addSubnet={() => {}}
        toggleSubnet={() => {}}
        deleteSubnet={() => {}}
      />
    );
  },
};
