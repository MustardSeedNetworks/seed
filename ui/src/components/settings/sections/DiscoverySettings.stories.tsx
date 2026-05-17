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
 *
 * Story fixtures (defaultSettings, defaultSnmpSettings, baseArgs) live in
 * DiscoverySettings.fixtures.ts so each story can override only what it
 * exercises.
 */

import type { Meta, StoryFn, StoryObj } from '@storybook/react-vite';
import type React from 'react';
import { useState } from 'react';
import type {
  NetworkDiscoverySettings,
  SaveStatus,
  SNMPSettings,
  SubnetConfig,
} from '../../../types/settings';
import { DiscoverySettings } from './DiscoverySettings';
import { baseArgs, defaultSettings, defaultSnmpSettings } from './DiscoverySettings.fixtures';

const meta: Meta<typeof DiscoverySettings> = {
  title: 'Settings/discovery-settings',
  component: DiscoverySettings,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Network discovery configuration panel with granular control over discovery methods (passive protocols, ARP, ICMP, port scanning, traceroute, SNMP). Manages scan options, timing, subnets, SNMP settings, and service status monitoring.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    networkDiscoveryStatus: {
      control: 'select',
      options: ['idle', 'saving', 'saved', 'error'],
      description: 'Auto-save status indicator for discovery settings',
    },
    subnetsStatus: {
      control: 'select',
      options: ['idle', 'saving', 'saved', 'error'],
      description: 'Subnet save status',
    },
    snmpStatus: {
      control: 'select',
      options: ['idle', 'saving', 'saved', 'error'],
      description: 'Auto-save status indicator for SNMP settings',
    },
  },
  decorators: [
    (StoryComponent: StoryFn): React.ReactElement => (
      <div class="w-[550px] max-h-[700px] overflow-y-auto">
        <StoryComponent />
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
  args: baseArgs(),
};

/**
 * Passive discovery only - using link-layer protocols
 */
export const PassiveOnly: Story = {
  args: {
    ...baseArgs(),
    networkDiscoverySettings: {
      ...defaultSettings,
      options: {
        ...defaultSettings.options,
        passiveProtocols: { lldp: true, cdp: true, edp: true, ndp: true },
        arpScan: false,
        icmpScan: false,
      },
    },
  },
};

/**
 * Full discovery with all methods enabled
 */
export const FullDiscovery: Story = {
  args: {
    ...baseArgs([
      { cidr: '10.0.0.0/24', name: 'Server VLAN', enabled: true },
      { cidr: '172.16.0.0/16', name: 'Management', enabled: true },
    ]),
    networkDiscoverySettings: {
      ...defaultSettings,
      options: {
        ...defaultSettings.options,
        passiveProtocols: { lldp: true, cdp: true, edp: true, ndp: true },
        portScan: {
          enabled: true,
          preset: 'common',
          tcpPorts: '22,80,443,8080,8443',
          udpPorts: '53,161,162',
          bannerTimeoutMs: 3000,
        },
        traceroute: true,
        snmpQuery: true,
      },
    },
    snmpSettings: {
      communities: ['public', 'private'],
      v3Credentials: [],
      timeout: 5000,
      retries: 3,
      port: 161,
    },
  },
};

/**
 * Port scanning with common ports preset
 */
export const WithPortScanCommon: Story = {
  args: {
    ...baseArgs(),
    networkDiscoverySettings: {
      ...defaultSettings,
      options: {
        ...defaultSettings.options,
        portScan: {
          enabled: true,
          preset: 'common',
          tcpPorts: '22,80,443,8080',
          udpPorts: '53,161',
          bannerTimeoutMs: 3000,
        },
      },
    },
  },
};

/**
 * Port scanning with secure ports preset
 */
export const WithPortScanSecure: Story = {
  args: {
    ...baseArgs(),
    networkDiscoverySettings: {
      ...defaultSettings,
      options: {
        ...defaultSettings.options,
        portScan: {
          enabled: true,
          preset: 'secure',
          tcpPorts: '22,443,8443',
          udpPorts: '',
          bannerTimeoutMs: 3000,
        },
      },
    },
  },
};

/**
 * Port scanning with insecure ports preset
 */
export const WithPortScanInsecure: Story = {
  args: {
    ...baseArgs(),
    networkDiscoverySettings: {
      ...defaultSettings,
      options: {
        ...defaultSettings.options,
        portScan: {
          enabled: true,
          preset: 'insecure',
          tcpPorts: '21,23,25,80,110,143',
          udpPorts: '69,161',
          bannerTimeoutMs: 3000,
        },
      },
    },
  },
};

/**
 * Custom port ranges configuration
 */
export const CustomPorts: Story = {
  args: {
    ...baseArgs(),
    networkDiscoverySettings: {
      ...defaultSettings,
      options: {
        ...defaultSettings.options,
        portScan: {
          enabled: true,
          preset: 'custom',
          tcpPorts: '22,80,443,3000-3010,8000-8100',
          udpPorts: '53,161,500-600',
          bannerTimeoutMs: 5000,
        },
      },
    },
  },
};

/**
 * Discovery disabled
 */
export const Disabled: Story = {
  args: {
    ...baseArgs(),
    networkDiscoverySettings: { ...defaultSettings, enabled: false },
  },
};

/**
 * Auto-scan enabled with interval
 */
export const AutoScanEnabled: Story = {
  args: {
    ...baseArgs(),
    networkDiscoverySettings: {
      ...defaultSettings,
      autoScan: true,
      scanIntervalMs: 300000, // 5 minutes
    },
  },
};

/**
 * With multiple subnets configured
 */
export const WithSubnets: Story = {
  args: baseArgs([
    { cidr: '10.0.0.0/24', name: 'Server VLAN', enabled: true },
    { cidr: '10.0.1.0/24', name: 'IoT Devices', enabled: true },
    { cidr: '172.16.0.0/16', name: 'Management Network', enabled: false },
    { cidr: '192.168.100.0/24', name: 'Guest WiFi', enabled: true },
  ]),
};

/**
 * Subnet validation error
 */
export const SubnetError: Story = {
  args: {
    ...baseArgs(),
    subnetsStatus: 'error',
    newSubnetCidr: 'invalid-cidr',
    subnetError: 'Invalid CIDR format',
  },
};

/**
 * Fast timing settings - aggressive scan parameters
 */
export const FastTiming: Story = {
  args: {
    ...baseArgs(),
    networkDiscoverySettings: {
      ...defaultSettings,
      arpScanWorkers: 100,
      pingTimeoutMs: 200,
      scanTimeoutMs: 15000,
      timing: { probeIntervalMs: 50, rescanIntervalMs: 60000, workers: 100 },
      options: {
        ...defaultSettings.options,
        tcpProbe: { timeoutMs: 1000, workers: 20 },
        portScan: { ...defaultSettings.options.portScan, bannerTimeoutMs: 1000 },
      },
    },
  },
};

/**
 * Thorough timing settings - slow/careful scan parameters
 */
export const ThoroughTiming: Story = {
  args: {
    ...baseArgs(),
    networkDiscoverySettings: {
      ...defaultSettings,
      arpScanWorkers: 20,
      pingTimeoutMs: 2000,
      scanTimeoutMs: 120000,
      timing: { probeIntervalMs: 500, rescanIntervalMs: 600000, workers: 20 },
      options: {
        ...defaultSettings.options,
        tcpProbe: { timeoutMs: 10000, workers: 5 },
        portScan: { ...defaultSettings.options.portScan, bannerTimeoutMs: 10000 },
      },
    },
  },
};

/**
 * Saving state for discovery settings
 */
export const Saving: Story = {
  args: {
    ...baseArgs(),
    networkDiscoveryStatus: 'saving',
  },
};

/**
 * SNMP settings with multiple communities and v3 credentials
 */
export const WithSnmpSettings: Story = {
  args: {
    ...baseArgs(),
    networkDiscoverySettings: {
      ...defaultSettings,
      options: { ...defaultSettings.options, snmpQuery: true },
    },
    snmpSettings: {
      communities: ['public', 'private', 'secret'],
      v3Credentials: [
        {
          name: 'Admin User',
          username: 'admin',
          authProtocol: 'SHA',
          authPassword: 'authpass123',
          privProtocol: 'AES',
          privPassword: 'privpass123',
          contextName: '',
          securityLevel: 'authPriv',
        },
      ],
      timeout: 10000,
      retries: 5,
      port: 161,
    },
  },
};

/**
 * Interactive discovery settings - fully functional
 */
export const Interactive: Story = {
  render: function interactiveStory() {
    const [settings, setSettings] = useState<NetworkDiscoverySettings>(defaultSettings);
    const [status, setStatus] = useState<SaveStatus>('idle');
    const [snmpSettings, setSnmpSettings] = useState<SNMPSettings>(defaultSnmpSettings);
    const [snmpStatus, setSnmpStatus] = useState<SaveStatus>('idle');
    const subnets: SubnetConfig[] = [{ cidr: '10.0.0.0/24', name: 'Server VLAN', enabled: true }];
    const [newCidr, setNewCidr] = useState('');
    const [newName, setNewName] = useState('');
    const [error, setError] = useState<string | null>(null);

    const handleSetSettings = (updater: React.SetStateAction<NetworkDiscoverySettings>) => {
      setSettings(updater);
      setStatus('saving');
      setTimeout(() => {
        setStatus('saved');
        setTimeout(() => setStatus('idle'), 2000);
      }, 800);
    };

    const handleSetSnmpSettings = (updater: React.SetStateAction<SNMPSettings>) => {
      setSnmpSettings(updater);
      setSnmpStatus('saving');
      setTimeout(() => {
        setSnmpStatus('saved');
        setTimeout(() => setSnmpStatus('idle'), 2000);
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
