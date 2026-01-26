import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { SnmpSettings } from '../../../../types/settings';
import { DEFAULT_SNMP_SETTINGS } from '../../../../types/settings';
import { SnmpSettingsSection } from './SnmpSettingsSection';

const meta = {
  title: 'Settings/SnmpSettingsSection',
  component: SnmpSettingsSection,
} satisfies Meta<typeof SnmpSettingsSection>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => {
    const [settings, setSettings] = useState<SnmpSettings>(DEFAULT_SNMP_SETTINGS);
    return <SnmpSettingsSection snmpSettings={settings} setSnmpSettings={setSettings} snmpStatus="saved" />;
  },
};
