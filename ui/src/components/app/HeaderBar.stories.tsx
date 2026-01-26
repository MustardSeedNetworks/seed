import type { Meta, StoryObj } from '@storybook/react-vite';
import type { Profile } from '../../types/profile';
import type { NetworkInterface } from '../ui/InterfaceSelector';
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

const interfaces: NetworkInterface[] = [
  {
    name: 'eth0',
    friendlyName: 'Primary Ethernet',
    type: 'ethernet',
    up: true,
    speedDisplay: '1 Gb/s',
  },
  {
    name: 'wlan0',
    friendlyName: 'WiFi Adapter',
    type: 'wifi',
    up: true,
    signalStrength: -47,
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
    activeProfile: profiles[0],
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
