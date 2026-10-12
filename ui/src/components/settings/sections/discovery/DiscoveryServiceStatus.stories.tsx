import type { Meta, StoryObj } from '@storybook/react-vite';
import type { DiscoveryServiceStatusResponse } from '../../../../types/generated/discovery-service-status-response';
import { DiscoveryServiceStatus } from './DiscoveryServiceStatus';

const meta = {
  title: 'Settings/DiscoveryServiceStatus',
  component: DiscoveryServiceStatus,
} satisfies Meta<typeof DiscoveryServiceStatus>;

export default meta;

type Story = StoryObj<typeof meta>;

const sampleStatus: DiscoveryServiceStatusResponse = {
  running: true,
  scanning: false,
  deviceCount: 12,
  interface: 'eth0',
  subnet: '192.168.1.0/24',
  subnets: ['192.168.1.0/24'],
  localIP: '192.168.1.10',
  activeMethods: ['arp', 'icmp', 'lldp'],
  lastScan: new Date(Date.now() - 30000).toISOString(),
  rescanIntervalMs: 60_000,
  snmpProbe: { silent: 0, tried: 0, answered: 0 },
  sweepCoverage: [{ subnet: '192.168.1.0/24', swept: 254, total: 254 }],
};

export const Running: Story = {
  args: {
    status: sampleStatus,
    loading: false,
    onRefresh: () => {},
  },
};

export const Scanning: Story = {
  args: {
    status: { ...sampleStatus, scanning: true },
    loading: true,
    onRefresh: () => {},
  },
};
