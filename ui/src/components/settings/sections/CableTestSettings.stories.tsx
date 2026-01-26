import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { CableTestSettings as CableTestSettingsType } from '../../../types/settings';
import { DEFAULT_CABLE_TEST_SETTINGS } from '../../../types/settings';
import { CableTestSettings } from './CableTestSettings';

const meta = {
  title: 'Settings/CableTestSettings',
  component: CableTestSettings,
} satisfies Meta<typeof CableTestSettings>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => {
    const [settings, setSettings] = useState<CableTestSettingsType>(DEFAULT_CABLE_TEST_SETTINGS);
    return (
      <CableTestSettings
        cableTestSettings={settings}
        setCableTestSettings={setSettings}
        cableTestStatus="saved"
      />
    );
  },
};
