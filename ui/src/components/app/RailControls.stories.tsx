import type { Meta, StoryObj } from '@storybook/react-vite';
import type { InterfaceInfo } from '../../types/generated/categorized-interfaces-response';
import type { Profile } from '../../types/profile';
import { RailControls } from './RailControls';

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
  title: 'App/RailControls',
  component: RailControls,
  parameters: { layout: 'centered' },
} satisfies Meta<typeof RailControls>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Expanded: Story = {
  args: {
    profiles,
    activeProfile: profiles.at(0) ?? null,
    profilesLoading: false,
    onProfileSwitch: async () => true,
    onProfileManage: () => {},
    logout: () => {},
    interfaces,
    currentInterface: 'eth0',
    isWifi: false,
    hasWifiInterface: true,
    onInterfaceChange: () => {},
    switchToInterfaceType: () => {},
    recommendedEthernet: 'eth0',
    toggleTheme: () => {},
    isDark: true,
    collapsed: false,
  },
};

/** The 64px rail: the controls stack and the panels open beside it. */
export const Collapsed: Story = {
  args: { ...Expanded.args, collapsed: true },
};

/** No Wi-Fi hardware — the mode control stays, with its warning pip. */
export const NoWifiHardware: Story = {
  args: { ...Expanded.args, hasWifiInterface: false },
};
