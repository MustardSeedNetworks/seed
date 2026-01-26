import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { NetworkDiscoverySettings } from '../../../../types/settings';
import { DEFAULT_NETWORK_DISCOVERY_SETTINGS } from '../../../../types/settings';
import { DiscoveryCustomOptions } from './DiscoveryCustomOptions';

const meta = {
  title: 'Settings/DiscoveryCustomOptions',
  component: DiscoveryCustomOptions,
} satisfies Meta<typeof DiscoveryCustomOptions>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => {
    const [settings, setSettings] = useState<NetworkDiscoverySettings>(
      DEFAULT_NETWORK_DISCOVERY_SETTINGS,
    );
    return <DiscoveryCustomOptions settings={settings} onSettingsChange={setSettings} />;
  },
};
