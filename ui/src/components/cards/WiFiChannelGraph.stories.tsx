//  lint/style/useNamingConvention: API response property names match backend schema (snake_case)
import type { Meta, StoryObj } from '@storybook/react-vite';
import { WifiChannelGraph } from './WiFiChannelGraph';

/**
 * WifiChannelGraph displays a visual representation of WiFi channel usage and overlap.
 * Helps identify channel congestion and optimal channel selection.
 *
 * Features:
 * - Channel overlap visualization for 2.4GHz, 5GHz, and 6GHz bands
 * - Signal strength visualization
 * - Connected network highlighting
 * - Interactive hover tooltips
 * - Band selection tabs
 */
const meta: Meta<typeof WifiChannelGraph> = {
  title: 'Cards/WiFiChannelGraph',
  component: WifiChannelGraph,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  decorators: [
    (StoryComponent: React.ComponentType): JSX.Element => (
      <div class="w-[640px]">
        <StoryComponent />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof WifiChannelGraph>;

/**
 * 2.4 GHz band with multiple overlapping networks
 * Shows typical home network scenario with channel congestion
 */
export const TwoPointFourGhz: Story = {
  args: {
    data: {
      available: true,
      data: {
        networks24Ghz: [
          {
            ssid: 'HomeNetwork',
            bssid: 'AA:BB:CC:DD:EE:01',
            channel: 1,
            centerFreq: 2412,
            channelWidth: 20,
            signal: -45,
            band: '2.4GHz',
            isConnected: true,
          },
          {
            ssid: 'Neighbor_WiFi',
            bssid: 'AA:BB:CC:DD:EE:02',
            channel: 6,
            centerFreq: 2437,
            channelWidth: 20,
            signal: -65,
            band: '2.4GHz',
            isConnected: false,
          },
          {
            ssid: 'Guest',
            bssid: 'AA:BB:CC:DD:EE:03',
            channel: 11,
            centerFreq: 2462,
            channelWidth: 20,
            signal: -70,
            band: '2.4GHz',
            isConnected: false,
          },
          {
            ssid: 'ApartmentWiFi',
            bssid: 'AA:BB:CC:DD:EE:04',
            channel: 1,
            centerFreq: 2412,
            channelWidth: 20,
            signal: -75,
            band: '2.4GHz',
            isConnected: false,
          },
        ],
        networks5Ghz: [],
        networks6Ghz: [],
        connectedBssid: 'AA:BB:CC:DD:EE:01',
        scanTime: new Date().toISOString(),
      },
    },
    loading: false,
    visible: true,
  },
};

/**
 * 5 GHz band with wide channels (80 MHz)
 * Shows less congestion typical of 5 GHz networks
 */
export const FiveGhz: Story = {
  args: {
    data: {
      available: true,
      data: {
        networks24Ghz: [],
        networks5Ghz: [
          {
            ssid: 'Home5G',
            bssid: 'AA:BB:CC:DD:EE:11',
            channel: 36,
            centerFreq: 5180,
            channelWidth: 80,
            signal: -50,
            band: '5GHz',
            isConnected: true,
          },
          {
            ssid: 'Office5G',
            bssid: 'AA:BB:CC:DD:EE:12',
            channel: 149,
            centerFreq: 5745,
            channelWidth: 80,
            signal: -60,
            band: '5GHz',
            isConnected: false,
          },
          {
            ssid: 'Neighbor5G',
            bssid: 'AA:BB:CC:DD:EE:13',
            channel: 100,
            centerFreq: 5500,
            channelWidth: 40,
            signal: -72,
            band: '5GHz',
            isConnected: false,
          },
        ],
        networks6Ghz: [],
        connectedBssid: 'AA:BB:CC:DD:EE:11',
        scanTime: new Date().toISOString(),
      },
    },
    loading: false,
    visible: true,
  },
};

/**
 * 6 GHz band with ultra-wide channels (160 MHz)
 * Shows modern WiFi 6E networks with minimal congestion
 */
export const SixGhz: Story = {
  args: {
    data: {
      available: true,
      data: {
        networks24Ghz: [],
        networks5Ghz: [],
        networks6Ghz: [
          {
            ssid: 'Home6E',
            bssid: 'AA:BB:CC:DD:EE:21',
            channel: 1,
            centerFreq: 5955,
            channelWidth: 160,
            signal: -40,
            band: '6GHz',
            isConnected: true,
          },
          {
            ssid: 'Office6E',
            bssid: 'AA:BB:CC:DD:EE:22',
            channel: 93,
            centerFreq: 6415,
            channelWidth: 160,
            signal: -55,
            band: '6GHz',
            isConnected: false,
          },
        ],
        connectedBssid: 'AA:BB:CC:DD:EE:21',
        scanTime: new Date().toISOString(),
      },
    },
    loading: false,
    visible: true,
  },
};

/**
 * Multi-band scenario with networks in all three bands
 * Shows typical modern dual/tri-band router setup
 */
export const MultiBand: Story = {
  args: {
    data: {
      available: true,
      data: {
        networks24Ghz: [
          {
            ssid: 'Home',
            bssid: 'AA:BB:CC:DD:EE:01',
            channel: 6,
            centerFreq: 2437,
            channelWidth: 20,
            signal: -50,
            band: '2.4GHz',
            isConnected: false,
          },
          {
            ssid: 'Neighbor',
            bssid: 'AA:BB:CC:DD:EE:02',
            channel: 1,
            centerFreq: 2412,
            channelWidth: 20,
            signal: -70,
            band: '2.4GHz',
            isConnected: false,
          },
        ],
        networks5Ghz: [
          {
            ssid: 'Home_5G',
            bssid: 'AA:BB:CC:DD:EE:11',
            channel: 36,
            centerFreq: 5180,
            channelWidth: 80,
            signal: -45,
            band: '5GHz',
            isConnected: true,
          },
          {
            ssid: 'Office_5G',
            bssid: 'AA:BB:CC:DD:EE:12',
            channel: 149,
            centerFreq: 5745,
            channelWidth: 80,
            signal: -65,
            band: '5GHz',
            isConnected: false,
          },
        ],
        networks6Ghz: [
          {
            ssid: 'Home_6E',
            bssid: 'AA:BB:CC:DD:EE:21',
            channel: 1,
            centerFreq: 5955,
            channelWidth: 160,
            signal: -42,
            band: '6GHz',
            isConnected: false,
          },
        ],
        connectedBssid: 'AA:BB:CC:DD:EE:11',
        scanTime: new Date().toISOString(),
      },
    },
    loading: false,
    visible: true,
  },
};

/**
 * Heavy congestion scenario in 2.4 GHz
 * Shows many overlapping networks competing for channels
 */
export const HeavyCongestion: Story = {
  args: {
    data: {
      available: true,
      data: {
        networks24Ghz: [
          {
            ssid: 'Home',
            bssid: 'AA:BB:CC:DD:EE:01',
            channel: 1,
            centerFreq: 2412,
            channelWidth: 20,
            signal: -45,
            band: '2.4GHz',
            isConnected: true,
          },
          {
            ssid: 'Apt101',
            bssid: 'AA:BB:CC:DD:EE:02',
            channel: 1,
            centerFreq: 2412,
            channelWidth: 20,
            signal: -60,
            band: '2.4GHz',
            isConnected: false,
          },
          {
            ssid: 'Apt102',
            bssid: 'AA:BB:CC:DD:EE:03',
            channel: 6,
            centerFreq: 2437,
            channelWidth: 20,
            signal: -55,
            band: '2.4GHz',
            isConnected: false,
          },
          {
            ssid: 'Apt103',
            bssid: 'AA:BB:CC:DD:EE:04',
            channel: 6,
            centerFreq: 2437,
            channelWidth: 20,
            signal: -70,
            band: '2.4GHz',
            isConnected: false,
          },
          {
            ssid: 'Apt104',
            bssid: 'AA:BB:CC:DD:EE:05',
            channel: 11,
            centerFreq: 2462,
            channelWidth: 20,
            signal: -65,
            band: '2.4GHz',
            isConnected: false,
          },
          {
            ssid: 'Apt105',
            bssid: 'AA:BB:CC:DD:EE:06',
            channel: 11,
            centerFreq: 2462,
            channelWidth: 20,
            signal: -75,
            band: '2.4GHz',
            isConnected: false,
          },
        ],
        networks5Ghz: [],
        networks6Ghz: [],
        connectedBssid: 'AA:BB:CC:DD:EE:01',
        scanTime: new Date().toISOString(),
      },
    },
    loading: false,
    visible: true,
  },
};

/**
 * Loading state while scanning
 */
export const Loading: Story = {
  args: {
    data: null,
    loading: true,
    visible: true,
  },
};

/**
 * No WiFi adapter available
 */
export const NoAdapter: Story = {
  args: {
    data: {
      available: false,
      error: 'No wireless adapter available. Connect a WiFi adapter to scan networks.',
    },
    loading: false,
    visible: true,
  },
};

/**
 * Error during scan
 */
export const ScanError: Story = {
  args: {
    data: {
      available: true,
      error: 'Failed to scan WiFi networks. Check permissions.',
    },
    loading: false,
    visible: true,
  },
};

/**
 * Empty scan (no networks detected)
 */
export const NoNetworks: Story = {
  args: {
    data: {
      available: true,
      data: {
        networks24Ghz: [],
        networks5Ghz: [],
        networks6Ghz: [],
        scanTime: new Date().toISOString(),
      },
    },
    loading: false,
    visible: true,
  },
};
