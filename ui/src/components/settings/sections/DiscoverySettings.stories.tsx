/**
 * DiscoverySettings Storybook Stories
 *
 * Demonstrates the network discovery configuration component with granular
 * control over discovery methods and scan options.
 *
 * Variants:
 * - Discovery methods: Passive protocols, ARP, ICMP, port scanning, traceroute, SNMP
 * - Port scan presets: Common, secure, insecure, custom
 * - Service status: Running, stopped, scanning
 * - With subnets: Additional target networks configured
 * - Timing settings: Workers, timeouts, intervals
 * - SNMP configuration: Communities, v3 credentials, timeout, retries
 */

import type { Meta, StoryObj } from "@storybook/react-vite";
import type React from "react";
import { useState } from "react";
import type {
  NetworkDiscoverySettings,
  SaveStatus,
  SNMPSettings,
  SubnetConfig,
} from "../../../types/settings";
import { DiscoverySettings } from "./DiscoverySettings";

const defaultSettings: NetworkDiscoverySettings = {
  enabled: true,
  arpScanWorkers: 50,
  pingTimeoutMs: 500,
  scanTimeoutMs: 30000,
  autoScan: false,
  scanIntervalMs: 0,
  ouiFilePath: "data/oui.txt",
  options: {
    passiveProtocols: {
      lldp: true,
      cdp: true,
      edp: false,
      ndp: false,
    },
    arpScan: true,
    icmpScan: true,
    portScan: {
      enabled: false,
      preset: "common",
      tcpPorts: "",
      udpPorts: "",
      bannerTimeoutMs: 3000,
    },
    tcpProbe: {
      timeoutMs: 3000,
      workers: 10,
    },
    traceroute: false,
    snmpQuery: false,
  },
  timing: {
    probeIntervalMs: 100,
    rescanIntervalMs: 300000,
    workers: 50,
  },
  profiler: {
    enabled: true,
    timeoutMs: 5000,
    maxConcurrent: 10,
    quickPorts: [22, 80, 443],
  },
  fingerprinting: {
    enabled: true,
    osDetection: true,
    serviceProbes: true,
  },
  ipv6Enabled: false,
};

const defaultSnmpSettings: SNMPSettings = {
  communities: ["public"],
  v3Credentials: [],
  timeout: 5000,
  retries: 2,
  port: 161,
};

const meta: Meta<typeof DiscoverySettings> = {
  title: "Settings/DiscoverySettings",
  component: DiscoverySettings,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Network discovery configuration panel with granular control over discovery methods (passive protocols, ARP, ICMP, port scanning, traceroute, SNMP). Manages scan options, timing, subnets, SNMP settings, and service status monitoring.",
      },
    },
  },
  tags: ["autodocs"],
  argTypes: {
    networkDiscoveryStatus: {
      control: "select",
      options: ["idle", "saving", "saved", "error"],
      description: "Auto-save status indicator for discovery settings",
    },
    subnetsStatus: {
      control: "select",
      options: ["idle", "saving", "saved", "error"],
      description: "Subnet save status",
    },
    snmpStatus: {
      control: "select",
      options: ["idle", "saving", "saved", "error"],
      description: "Auto-save status indicator for SNMP settings",
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
 * Default discovery settings with basic methods enabled
 */
export const Default: Story = {
  args: {
    networkDiscoverySettings: defaultSettings,
    setNetworkDiscoverySettings: () => {
      // intentionally empty
    },
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {
      // intentionally empty
    },
    newSubnetName: "",
    setNewSubnetName: () => {
      // intentionally empty
    },
    subnetError: null,
    setSubnetError: () => {
      // intentionally empty
    },
    addSubnet: () => {
      // intentionally empty
    },
    toggleSubnet: () => {
      // intentionally empty
    },
    deleteSubnet: () => {
      // intentionally empty
    },
    snmpSettings: defaultSnmpSettings,
    setSnmpSettings: () => {
      // intentionally empty
    },
    snmpStatus: "idle",
  },
};

/**
 * Passive discovery only - using link-layer protocols
 */
export const PassiveOnly: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      options: {
        passiveProtocols: {
          lldp: true,
          cdp: true,
          edp: true,
          ndp: true,
        },
        arpScan: false,
        icmpScan: false,
        portScan: {
          enabled: false,
          preset: "common",
          tcpPorts: "",
          udpPorts: "",
          bannerTimeoutMs: 3000,
        },
        tcpProbe: {
          timeoutMs: 3000,
          workers: 10,
        },
        traceroute: false,
        snmpQuery: false,
      },
    },
    setNetworkDiscoverySettings: () => {
      // intentionally empty
    },
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {
      // intentionally empty
    },
    newSubnetName: "",
    setNewSubnetName: () => {
      // intentionally empty
    },
    subnetError: null,
    setSubnetError: () => {
      // intentionally empty
    },
    addSubnet: () => {
      // intentionally empty
    },
    toggleSubnet: () => {
      // intentionally empty
    },
    deleteSubnet: () => {
      // intentionally empty
    },
    snmpSettings: defaultSnmpSettings,
    setSnmpSettings: () => {
      // intentionally empty
    },
    snmpStatus: "idle",
  },
};

/**
 * Full discovery with all methods enabled
 */
export const FullDiscovery: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      options: {
        passiveProtocols: {
          lldp: true,
          cdp: true,
          edp: true,
          ndp: true,
        },
        arpScan: true,
        icmpScan: true,
        portScan: {
          enabled: true,
          preset: "common",
          tcpPorts: "22,80,443,8080,8443",
          udpPorts: "53,161,162",
          bannerTimeoutMs: 3000,
        },
        tcpProbe: {
          timeoutMs: 3000,
          workers: 10,
        },
        traceroute: true,
        snmpQuery: true,
      },
    },
    setNetworkDiscoverySettings: () => {
      // intentionally empty
    },
    networkDiscoveryStatus: "idle",
    subnets: [
      { cidr: "10.0.0.0/24", name: "Server VLAN", enabled: true },
      { cidr: "172.16.0.0/16", name: "Management", enabled: true },
    ],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {
      // intentionally empty
    },
    newSubnetName: "",
    setNewSubnetName: () => {
      // intentionally empty
    },
    subnetError: null,
    setSubnetError: () => {
      // intentionally empty
    },
    addSubnet: () => {
      // intentionally empty
    },
    toggleSubnet: () => {
      // intentionally empty
    },
    deleteSubnet: () => {
      // intentionally empty
    },
    snmpSettings: {
      communities: ["public", "private"],
      v3Credentials: [],
      timeout: 5000,
      retries: 3,
      port: 161,
    },
    setSnmpSettings: () => {
      // intentionally empty
    },
    snmpStatus: "idle",
  },
};

/**
 * Port scanning with common ports preset
 */
export const WithPortScanCommon: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      options: {
        ...defaultSettings.options,
        portScan: {
          enabled: true,
          preset: "common",
          tcpPorts: "22,80,443,8080",
          udpPorts: "53,161",
          bannerTimeoutMs: 3000,
        },
      },
    },
    setNetworkDiscoverySettings: () => {
      // intentionally empty
    },
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {
      // intentionally empty
    },
    newSubnetName: "",
    setNewSubnetName: () => {
      // intentionally empty
    },
    subnetError: null,
    setSubnetError: () => {
      // intentionally empty
    },
    addSubnet: () => {
      // intentionally empty
    },
    toggleSubnet: () => {
      // intentionally empty
    },
    deleteSubnet: () => {
      // intentionally empty
    },
    snmpSettings: defaultSnmpSettings,
    setSnmpSettings: () => {
      // intentionally empty
    },
    snmpStatus: "idle",
  },
};

/**
 * Port scanning with secure ports preset
 */
export const WithPortScanSecure: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      options: {
        ...defaultSettings.options,
        portScan: {
          enabled: true,
          preset: "secure",
          tcpPorts: "22,443,8443",
          udpPorts: "",
          bannerTimeoutMs: 3000,
        },
      },
    },
    setNetworkDiscoverySettings: () => {
      // intentionally empty
    },
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {
      // intentionally empty
    },
    newSubnetName: "",
    setNewSubnetName: () => {
      // intentionally empty
    },
    subnetError: null,
    setSubnetError: () => {
      // intentionally empty
    },
    addSubnet: () => {
      // intentionally empty
    },
    toggleSubnet: () => {
      // intentionally empty
    },
    deleteSubnet: () => {
      // intentionally empty
    },
    snmpSettings: defaultSnmpSettings,
    setSnmpSettings: () => {
      // intentionally empty
    },
    snmpStatus: "idle",
  },
};

/**
 * Port scanning with insecure ports preset
 */
export const WithPortScanInsecure: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      options: {
        ...defaultSettings.options,
        portScan: {
          enabled: true,
          preset: "insecure",
          tcpPorts: "21,23,25,80,110,143",
          udpPorts: "69,161",
          bannerTimeoutMs: 3000,
        },
      },
    },
    setNetworkDiscoverySettings: () => {
      // intentionally empty
    },
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {
      // intentionally empty
    },
    newSubnetName: "",
    setNewSubnetName: () => {
      // intentionally empty
    },
    subnetError: null,
    setSubnetError: () => {
      // intentionally empty
    },
    addSubnet: () => {
      // intentionally empty
    },
    toggleSubnet: () => {
      // intentionally empty
    },
    deleteSubnet: () => {
      // intentionally empty
    },
    snmpSettings: defaultSnmpSettings,
    setSnmpSettings: () => {
      // intentionally empty
    },
    snmpStatus: "idle",
  },
};

/**
 * Custom port ranges configuration
 */
export const CustomPorts: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      options: {
        ...defaultSettings.options,
        portScan: {
          enabled: true,
          preset: "custom",
          tcpPorts: "22,80,443,3000-3010,8000-8100",
          udpPorts: "53,161,500-600",
          bannerTimeoutMs: 5000,
        },
      },
    },
    setNetworkDiscoverySettings: () => {
      // intentionally empty
    },
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {
      // intentionally empty
    },
    newSubnetName: "",
    setNewSubnetName: () => {
      // intentionally empty
    },
    subnetError: null,
    setSubnetError: () => {
      // intentionally empty
    },
    addSubnet: () => {
      // intentionally empty
    },
    toggleSubnet: () => {
      // intentionally empty
    },
    deleteSubnet: () => {
      // intentionally empty
    },
    snmpSettings: defaultSnmpSettings,
    setSnmpSettings: () => {
      // intentionally empty
    },
    snmpStatus: "idle",
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
    setNetworkDiscoverySettings: () => {
      // intentionally empty
    },
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {
      // intentionally empty
    },
    newSubnetName: "",
    setNewSubnetName: () => {
      // intentionally empty
    },
    subnetError: null,
    setSubnetError: () => {
      // intentionally empty
    },
    addSubnet: () => {
      // intentionally empty
    },
    toggleSubnet: () => {
      // intentionally empty
    },
    deleteSubnet: () => {
      // intentionally empty
    },
    snmpSettings: defaultSnmpSettings,
    setSnmpSettings: () => {
      // intentionally empty
    },
    snmpStatus: "idle",
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
    setNetworkDiscoverySettings: () => {
      // intentionally empty
    },
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {
      // intentionally empty
    },
    newSubnetName: "",
    setNewSubnetName: () => {
      // intentionally empty
    },
    subnetError: null,
    setSubnetError: () => {
      // intentionally empty
    },
    addSubnet: () => {
      // intentionally empty
    },
    toggleSubnet: () => {
      // intentionally empty
    },
    deleteSubnet: () => {
      // intentionally empty
    },
    snmpSettings: defaultSnmpSettings,
    setSnmpSettings: () => {
      // intentionally empty
    },
    snmpStatus: "idle",
  },
};

/**
 * With multiple subnets configured
 */
export const WithSubnets: Story = {
  args: {
    networkDiscoverySettings: defaultSettings,
    setNetworkDiscoverySettings: () => {
      // intentionally empty
    },
    networkDiscoveryStatus: "idle",
    subnets: [
      { cidr: "10.0.0.0/24", name: "Server VLAN", enabled: true },
      { cidr: "10.0.1.0/24", name: "IoT Devices", enabled: true },
      { cidr: "172.16.0.0/16", name: "Management Network", enabled: false },
      { cidr: "192.168.100.0/24", name: "Guest WiFi", enabled: true },
    ],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {
      // intentionally empty
    },
    newSubnetName: "",
    setNewSubnetName: () => {
      // intentionally empty
    },
    subnetError: null,
    setSubnetError: () => {
      // intentionally empty
    },
    addSubnet: () => {
      // intentionally empty
    },
    toggleSubnet: () => {
      // intentionally empty
    },
    deleteSubnet: () => {
      // intentionally empty
    },
    snmpSettings: defaultSnmpSettings,
    setSnmpSettings: () => {
      // intentionally empty
    },
    snmpStatus: "idle",
  },
};

/**
 * Subnet validation error
 */
export const SubnetError: Story = {
  args: {
    networkDiscoverySettings: defaultSettings,
    setNetworkDiscoverySettings: () => {
      // intentionally empty
    },
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "error",
    newSubnetCidr: "invalid-cidr",
    setNewSubnetCidr: () => {
      // intentionally empty
    },
    newSubnetName: "",
    setNewSubnetName: () => {
      // intentionally empty
    },
    subnetError: "Invalid CIDR format",
    setSubnetError: () => {
      // intentionally empty
    },
    addSubnet: () => {
      // intentionally empty
    },
    toggleSubnet: () => {
      // intentionally empty
    },
    deleteSubnet: () => {
      // intentionally empty
    },
    snmpSettings: defaultSnmpSettings,
    setSnmpSettings: () => {
      // intentionally empty
    },
    snmpStatus: "idle",
  },
};

/**
 * Fast timing settings - aggressive scan parameters
 */
export const FastTiming: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      arpScanWorkers: 100,
      pingTimeoutMs: 200,
      scanTimeoutMs: 15000,
      timing: {
        probeIntervalMs: 50,
        rescanIntervalMs: 60000, // 1 minute
        workers: 100,
      },
      options: {
        ...defaultSettings.options,
        tcpProbe: {
          timeoutMs: 1000,
          workers: 20,
        },
        portScan: {
          ...defaultSettings.options.portScan,
          bannerTimeoutMs: 1000,
        },
      },
    },
    setNetworkDiscoverySettings: () => {
      // intentionally empty
    },
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {
      // intentionally empty
    },
    newSubnetName: "",
    setNewSubnetName: () => {
      // intentionally empty
    },
    subnetError: null,
    setSubnetError: () => {
      // intentionally empty
    },
    addSubnet: () => {
      // intentionally empty
    },
    toggleSubnet: () => {
      // intentionally empty
    },
    deleteSubnet: () => {
      // intentionally empty
    },
    snmpSettings: defaultSnmpSettings,
    setSnmpSettings: () => {
      // intentionally empty
    },
    snmpStatus: "idle",
  },
};

/**
 * Thorough timing settings - slow/careful scan parameters
 */
export const ThoroughTiming: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      arpScanWorkers: 20,
      pingTimeoutMs: 2000,
      scanTimeoutMs: 120000,
      timing: {
        probeIntervalMs: 500,
        rescanIntervalMs: 600000, // 10 minutes
        workers: 20,
      },
      options: {
        ...defaultSettings.options,
        tcpProbe: {
          timeoutMs: 10000,
          workers: 5,
        },
        portScan: {
          ...defaultSettings.options.portScan,
          bannerTimeoutMs: 10000,
        },
      },
    },
    setNetworkDiscoverySettings: () => {
      // intentionally empty
    },
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {
      // intentionally empty
    },
    newSubnetName: "",
    setNewSubnetName: () => {
      // intentionally empty
    },
    subnetError: null,
    setSubnetError: () => {
      // intentionally empty
    },
    addSubnet: () => {
      // intentionally empty
    },
    toggleSubnet: () => {
      // intentionally empty
    },
    deleteSubnet: () => {
      // intentionally empty
    },
    snmpSettings: defaultSnmpSettings,
    setSnmpSettings: () => {
      // intentionally empty
    },
    snmpStatus: "idle",
  },
};

/**
 * Saving state for discovery settings
 */
export const Saving: Story = {
  args: {
    networkDiscoverySettings: defaultSettings,
    setNetworkDiscoverySettings: () => {
      // intentionally empty
    },
    networkDiscoveryStatus: "saving",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {
      // intentionally empty
    },
    newSubnetName: "",
    setNewSubnetName: () => {
      // intentionally empty
    },
    subnetError: null,
    setSubnetError: () => {
      // intentionally empty
    },
    addSubnet: () => {
      // intentionally empty
    },
    toggleSubnet: () => {
      // intentionally empty
    },
    deleteSubnet: () => {
      // intentionally empty
    },
    snmpSettings: defaultSnmpSettings,
    setSnmpSettings: () => {
      // intentionally empty
    },
    snmpStatus: "idle",
  },
};

/**
 * SNMP settings with multiple communities and v3 credentials
 */
export const WithSnmpSettings: Story = {
  args: {
    networkDiscoverySettings: {
      ...defaultSettings,
      options: {
        ...defaultSettings.options,
        snmpQuery: true,
      },
    },
    setNetworkDiscoverySettings: () => {
      // intentionally empty
    },
    networkDiscoveryStatus: "idle",
    subnets: [],
    subnetsStatus: "idle",
    newSubnetCidr: "",
    setNewSubnetCidr: () => {
      // intentionally empty
    },
    newSubnetName: "",
    setNewSubnetName: () => {
      // intentionally empty
    },
    subnetError: null,
    setSubnetError: () => {
      // intentionally empty
    },
    addSubnet: () => {
      // intentionally empty
    },
    toggleSubnet: () => {
      // intentionally empty
    },
    deleteSubnet: () => {
      // intentionally empty
    },
    snmpSettings: {
      communities: ["public", "private", "secret"],
      v3Credentials: [
        {
          name: "Admin User",
          username: "admin",
          authProtocol: "SHA",
          authPassword: "authpass123",
          privProtocol: "AES",
          privPassword: "privpass123",
          contextName: "",
          securityLevel: "authPriv",
        },
      ],
      timeout: 10000,
      retries: 5,
      port: 161,
    },
    setSnmpSettings: () => {
      // intentionally empty
    },
    snmpStatus: "idle",
  },
};

/**
 * Interactive discovery settings - fully functional
 */
export const Interactive: Story = {
  render: function InteractiveStory() {
    const [settings, setSettings] = useState<NetworkDiscoverySettings>(defaultSettings);
    const [status, setStatus] = useState<SaveStatus>("idle");
    const [snmpSettings, setSnmpSettings] = useState<SNMPSettings>(defaultSnmpSettings);
    const [snmpStatus, setSnmpStatus] = useState<SaveStatus>("idle");
    const subnets: SubnetConfig[] = [{ cidr: "10.0.0.0/24", name: "Server VLAN", enabled: true }];
    const [newCidr, setNewCidr] = useState("");
    const [newName, setNewName] = useState("");
    const [error, setError] = useState<string | null>(null);

    const handleSetSettings = (updater: React.SetStateAction<NetworkDiscoverySettings>) => {
      setSettings(updater);
      setStatus("saving");
      setTimeout(() => {
        setStatus("saved");
        setTimeout(() => setStatus("idle"), 2000);
      }, 800);
    };

    const handleSetSnmpSettings = (updater: React.SetStateAction<SNMPSettings>) => {
      setSnmpSettings(updater);
      setSnmpStatus("saving");
      setTimeout(() => {
        setSnmpStatus("saved");
        setTimeout(() => setSnmpStatus("idle"), 2000);
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
        addSubnet={() => {
          // intentionally empty
        }}
        toggleSubnet={() => {
          // intentionally empty
        }}
        deleteSubnet={() => {
          // intentionally empty
        }}
        snmpSettings={snmpSettings}
        setSnmpSettings={handleSetSnmpSettings}
        snmpStatus={snmpStatus}
      />
    );
  },
};
