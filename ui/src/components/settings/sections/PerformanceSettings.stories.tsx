/**
 * PerformanceSettings Storybook Stories
 *
 * Demonstrates the performance testing configuration component for speedtest.net
 * and iperf3 LAN speed testing.
 *
 * Variants:
 * - Default configuration: Both speedtest and iperf enabled
 * - Only speedtest: Internet speed testing only
 * - Only iperf: LAN speed testing only
 * - Both disabled: No performance tests
 * - With iperf server: iperf server mode enabled
 * - With iperf suggestions: Shows discovered iperf hosts
 * - Different protocols: TCP vs UDP
 * - Different directions: Download, upload, bidirectional
 */

import type { Decorator, Meta, StoryObj } from '@storybook/react-vite';
import type React from 'react';
import { useState } from 'react';
import { fn } from 'storybook/test';
import type {
  IperfSettings,
  IperfSuggestion,
  SaveStatus,
  TestsSettings,
} from '../../../types/settings';
import { DEFAULT_CARD_SETTINGS } from '../../../types/settings';
import { PerformanceSettings } from './PerformanceSettings';
import { expandSections } from './storyPlay';

const baseTestsSettings: TestsSettings = {
  dnsHostname: 'google.com',
  dnsServers: [],
  pingTargets: [],
  tcpPorts: [],
  udpPorts: [],
  httpEndpoints: [],
  runPerformance: true,
  runSpeedtest: true,
  runIperf: true,
  runDiscovery: true,
  speedtest: { serverId: '', autoRunOnLink: false },
  iperf: { autoRunOnLink: false },
};

const defaultIperfSettings: IperfSettings = {
  server: '',
  port: 5201,
  protocol: 'tcp',
  duration: 10,
  direction: 'download',
  enableServer: false,
  serverPort: 5201,
};

const mockIperfSuggestions: IperfSuggestion[] = [
  { host: '192.168.1.100', hostname: 'server1.local', latencyMs: 2 },
  { host: '192.168.1.101', hostname: 'nas.local', latencyMs: 5 },
  { host: '192.168.1.102', latencyMs: 8 },
];

const meta = {
  title: 'Settings/performance-settings',
  component: PerformanceSettings,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Performance testing configuration for speedtest.net and iperf3. Configure internet speed tests, LAN throughput testing, protocols, directions, and auto-run options.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    iperfStatus: {
      control: 'select',
      options: ['idle', 'saving', 'saved', 'error'],
      description: 'Auto-save status indicator',
    },
    iperfSuggestionsStatus: {
      control: 'select',
      options: ['idle', 'loading', 'error'],
      description: 'iperf server discovery status',
    },
  },
  decorators: [
    (Story: Parameters<Decorator>[0]) => (
      <div className="w-[550px] max-h-[700px] overflow-y-auto">
        <Story />
      </div>
    ),
  ],
  args: {
    testsSettings: baseTestsSettings,
    setTestsSettings: fn(),
    iperfSettings: defaultIperfSettings,
    setIperfSettings: fn(),
    iperfStatus: 'idle',
    iperfSuggestions: [],
    iperfSuggestionsStatus: 'idle',
    iperfSuggestionsError: null,
    fetchIperfSuggestions: fn(),
    cardSettings: DEFAULT_CARD_SETTINGS,
    updateCardSettings: fn(),
  },
  play: expandSections,
} satisfies Meta<typeof PerformanceSettings>;

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default configuration - both tests enabled
 */
export const Default: Story = {
  args: {
    testsSettings: baseTestsSettings,
    iperfSettings: defaultIperfSettings,
    iperfStatus: 'idle',
    iperfSuggestions: [],
    iperfSuggestionsStatus: 'idle',
    iperfSuggestionsError: null,
  },
};

/**
 * Only speedtest enabled
 */
export const OnlySpeedtest: Story = {
  args: {
    testsSettings: {
      ...baseTestsSettings,
      runSpeedtest: true,
      runIperf: false,
    },
    iperfSettings: defaultIperfSettings,
    iperfStatus: 'idle',
    iperfSuggestions: [],
    iperfSuggestionsStatus: 'idle',
    iperfSuggestionsError: null,
  },
};

/**
 * Only iperf enabled
 */
export const OnlyIperf: Story = {
  args: {
    testsSettings: {
      ...baseTestsSettings,
      runSpeedtest: false,
      runIperf: true,
    },
    iperfSettings: defaultIperfSettings,
    iperfStatus: 'idle',
    iperfSuggestions: [],
    iperfSuggestionsStatus: 'idle',
    iperfSuggestionsError: null,
  },
};

/**
 * Both tests disabled
 */
export const BothDisabled: Story = {
  args: {
    testsSettings: {
      ...baseTestsSettings,
      runSpeedtest: false,
      runIperf: false,
    },
    iperfSettings: defaultIperfSettings,
    iperfStatus: 'idle',
    iperfSuggestions: [],
    iperfSuggestionsStatus: 'idle',
    iperfSuggestionsError: null,
  },
};

/**
 * Auto-run on link up enabled
 */
export const AutoRunEnabled: Story = {
  args: {
    testsSettings: {
      ...baseTestsSettings,
      speedtest: { serverId: '', autoRunOnLink: true },
      iperf: { autoRunOnLink: true },
    },
    iperfSettings: defaultIperfSettings,
    iperfStatus: 'idle',
    iperfSuggestions: [],
    iperfSuggestionsStatus: 'idle',
    iperfSuggestionsError: null,
  },
};

/**
 * iperf with server configured
 */
export const IperfWithServer: Story = {
  args: {
    testsSettings: baseTestsSettings,
    iperfSettings: {
      server: '192.168.1.100',
      port: 5201,
      protocol: 'tcp',
      duration: 10,
      direction: 'download',
      enableServer: false,
      serverPort: 5201,
    },
    iperfStatus: 'idle',
    iperfSuggestions: [],
    iperfSuggestionsStatus: 'idle',
    iperfSuggestionsError: null,
  },
};

/**
 * iperf server mode enabled
 */
export const IperfServerMode: Story = {
  args: {
    testsSettings: baseTestsSettings,
    iperfSettings: {
      server: '192.168.1.100',
      port: 5201,
      protocol: 'tcp',
      duration: 10,
      direction: 'download',
      enableServer: true,
      serverPort: 5202,
    },
    iperfStatus: 'idle',
    iperfSuggestions: [],
    iperfSuggestionsStatus: 'idle',
    iperfSuggestionsError: null,
  },
};

/**
 * With iperf suggestions - shows discovered hosts
 */
export const WithIperfSuggestions: Story = {
  args: {
    testsSettings: baseTestsSettings,
    iperfSettings: defaultIperfSettings,
    iperfStatus: 'idle',
    iperfSuggestions: mockIperfSuggestions,
    iperfSuggestionsStatus: 'idle',
    iperfSuggestionsError: null,
  },
};

/**
 * iperf suggestions loading
 */
export const SuggestionsLoading: Story = {
  args: {
    testsSettings: baseTestsSettings,
    iperfSettings: defaultIperfSettings,
    iperfStatus: 'idle',
    iperfSuggestions: [],
    iperfSuggestionsStatus: 'loading',
    iperfSuggestionsError: null,
  },
};

/**
 * iperf suggestions error
 */
export const SuggestionsError: Story = {
  args: {
    testsSettings: baseTestsSettings,
    iperfSettings: defaultIperfSettings,
    iperfStatus: 'idle',
    iperfSuggestions: [],
    iperfSuggestionsStatus: 'error',
    iperfSuggestionsError: 'No iperf hosts found on network',
  },
};

/**
 * UDP protocol selected
 */
export const UdpProtocol: Story = {
  args: {
    testsSettings: baseTestsSettings,
    iperfSettings: {
      ...defaultIperfSettings,
      protocol: 'udp',
      server: '192.168.1.100',
    },
    iperfStatus: 'idle',
    iperfSuggestions: [],
    iperfSuggestionsStatus: 'idle',
    iperfSuggestionsError: null,
  },
};

/**
 * Upload direction selected
 */
export const UploadDirection: Story = {
  args: {
    testsSettings: baseTestsSettings,
    iperfSettings: {
      ...defaultIperfSettings,
      direction: 'upload',
      server: '192.168.1.100',
    },
    iperfStatus: 'idle',
    iperfSuggestions: [],
    iperfSuggestionsStatus: 'idle',
    iperfSuggestionsError: null,
  },
};

/**
 * Bidirectional test
 */
export const Bidirectional: Story = {
  args: {
    testsSettings: baseTestsSettings,
    iperfSettings: {
      ...defaultIperfSettings,
      direction: 'bidirectional',
      server: '192.168.1.100',
    },
    iperfStatus: 'idle',
    iperfSuggestions: [],
    iperfSuggestionsStatus: 'idle',
    iperfSuggestionsError: null,
  },
};

/**
 * Custom speedtest server ID
 */
export const CustomSpeedtestServer: Story = {
  args: {
    testsSettings: {
      ...baseTestsSettings,
      speedtest: { serverId: '12345', autoRunOnLink: false },
    },
    iperfSettings: defaultIperfSettings,
    iperfStatus: 'idle',
    iperfSuggestions: [],
    iperfSuggestionsStatus: 'idle',
    iperfSuggestionsError: null,
  },
};

/**
 * Saving state
 */
export const Saving: Story = {
  args: {
    testsSettings: baseTestsSettings,
    iperfSettings: defaultIperfSettings,
    iperfStatus: 'saving',
    iperfSuggestions: [],
    iperfSuggestionsStatus: 'idle',
    iperfSuggestionsError: null,
  },
};

/**
 * Interactive performance settings - fully functional
 */
export const Interactive: Story = {
  render: function interactiveStory() {
    const [testsSettings, setTestsSettings] = useState<TestsSettings>(baseTestsSettings);
    const [iperfSettings, setIperfSettings] = useState<IperfSettings>(defaultIperfSettings);
    const [status, setStatus] = useState<SaveStatus>('idle');
    const [suggestions, setSuggestions] = useState<IperfSuggestion[]>([]);
    const [suggestionsStatus, setSuggestionsStatus] = useState<'idle' | 'loading' | 'error'>(
      'idle',
    );

    const handleSetIperfSettings = (updater: React.SetStateAction<IperfSettings>): void => {
      setIperfSettings(updater);
      setStatus('saving');
      setTimeout(() => {
        setStatus('saved');
        setTimeout(() => setStatus('idle'), 2000);
      }, 800);
    };

    const handleFetchSuggestions = (): void => {
      setSuggestionsStatus('loading');
      setTimeout(() => {
        setSuggestions(mockIperfSuggestions);
        setSuggestionsStatus('idle');
      }, 2000);
    };

    return (
      <PerformanceSettings
        testsSettings={testsSettings}
        setTestsSettings={setTestsSettings}
        iperfSettings={iperfSettings}
        setIperfSettings={handleSetIperfSettings}
        iperfStatus={status}
        iperfSuggestions={suggestions}
        iperfSuggestionsStatus={suggestionsStatus}
        iperfSuggestionsError={null}
        fetchIperfSuggestions={handleFetchSuggestions}
        cardSettings={DEFAULT_CARD_SETTINGS}
        updateCardSettings={() => {}}
      />
    );
  },
};
