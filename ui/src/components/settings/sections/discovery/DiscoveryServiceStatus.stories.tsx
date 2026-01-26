import type { Meta, StoryObj } from '@storybook/react-vite';
import type { DiscoveryServiceStatus as DiscoveryServiceStatusType } from '../../../../types/settings';
import { DiscoveryServiceStatus } from './DiscoveryServiceStatus';

const meta = {
  title: 'Settings/DiscoveryServiceStatus',
  component: DiscoveryServiceStatus,
} satisfies Meta<typeof DiscoveryServiceStatus>;

export default meta;

type Story = StoryObj<typeof meta>;

const sampleStatus: DiscoveryServiceStatusType = {
  running: true,
  scanning: false,
  deviceCount: 12,
  interface: 'eth0',
  subnet: '192.168.1.0/24',
  localIP: '192.168.1.10',
  activeMethods: ['arp', 'icmp', 'lldp'],
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
