import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { NetworkDiscoverySettings } from '../../../../types/settings';
import { DEFAULT_NETWORK_DISCOVERY_SETTINGS } from '../../../../types/settings';
import { DiscoveryTimingSettings } from './DiscoveryTimingSettings';

const meta = {
  title: 'Settings/DiscoveryTimingSettings',
  component: DiscoveryTimingSettings,
} satisfies Meta<typeof DiscoveryTimingSettings>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => {
    const [settings, setSettings] = useState<NetworkDiscoverySettings>(
      DEFAULT_NETWORK_DISCOVERY_SETTINGS,
    );
    return <DiscoveryTimingSettings settings={settings} onSettingsChange={setSettings} />;
  },
};
