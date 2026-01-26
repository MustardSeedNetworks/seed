import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { NetworkDiscoverySettings } from '../../../../types/settings';
import { DEFAULT_NETWORK_DISCOVERY_SETTINGS } from '../../../../types/settings';
import { DiscoveryToggles } from './DiscoveryToggles';

const meta = {
  title: 'Settings/DiscoveryToggles',
  component: DiscoveryToggles,
} satisfies Meta<typeof DiscoveryToggles>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => {
    const [settings, setSettings] = useState<NetworkDiscoverySettings>(
      DEFAULT_NETWORK_DISCOVERY_SETTINGS,
    );
    return <DiscoveryToggles settings={settings} onSettingsChange={setSettings} />;
  },
};
