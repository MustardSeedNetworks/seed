import type { Meta, StoryObj } from '@storybook/react-vite';
import type { InterfaceInfo } from '../../types/generated/categorized-interfaces-response';
import type { Profile } from '../../types/profile';
import { HeaderBar } from './HeaderBar';

const profiles: Profile[] = [
  {
    id: 'default',
    name: 'Default',
    description: 'Default profile',
    config: {},
    isDefault: true,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  },
  {
    id: 'client-01',
    name: 'Acme HQ',
    description: 'Primary office profile',
    config: {},
    isDefault: false,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  },
];

const interfaces: InterfaceInfo[] = [
  {
    name: 'eth0',
    friendlyName: 'Primary Ethernet',
    type: 'ethernet',
    up: true,
    running: true,
    hardwareAddr: '00:1b:21:aa:bb:01',
    mtu: 1500,
    addresses: ['192.168.1.10/24'],
    speedDisplay: '1 Gb/s',
  },
  {
    name: 'wlan0',
    friendlyName: 'WiFi Adapter',
    type: 'wifi',
    up: true,
    running: true,
    hardwareAddr: '00:1b:21:aa:bb:03',
    mtu: 1500,
    addresses: ['192.168.1.11/24'],
  },
];

const meta = {
  title: 'App/HeaderBar',
  component: HeaderBar,
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof HeaderBar>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Connected: Story = {
  args: {
    wsStatus: 'connected',
    onReconnect: () => {},
    profiles,
    activeProfile: profiles.at(0) ?? null,
    profilesLoading: false,
    onProfileSwitch: async () => true,
    onProfileManage: () => {},
    interfaces,
    currentInterface: 'eth0',
    isWifi: false,
    onInterfaceChange: () => {},
    hasEthernet: true,
    hasWifiInterface: true,
    switchToInterfaceType: () => {},
    toggleTheme: () => {},
    isDark: true,
    onHelpOpen: () => {},
    onSettingsOpen: () => {},
    logout: () => {},
    recommendedEthernet: 'eth0',
  },
};

export const Disconnected: Story = {
  args: {
    ...Connected.args,
    wsStatus: 'disconnected',
  },
};
