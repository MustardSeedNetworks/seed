import type { Meta, StoryObj } from '@storybook/react-vite';
import { fn } from 'storybook/test';
import { emptyInput, TargetForm } from './TargetForm';

const meta: Meta<typeof TargetForm> = {
  title: 'Pages/PollingTargets/TargetForm',
  component: TargetForm,
  parameters: { layout: 'fullscreen' },
  args: {
    onSubmit: fn(async () => {}),
    onCancel: fn(),
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Create: Story = {
  args: { mode: 'create', initial: emptyInput() },
};

export const Edit: Story = {
  args: {
    mode: 'edit',
    initial: {
      name: 'core-sw-01',
      ipAddress: '10.44.10.2',
      snmpVersion: 'v2c',
      credentialsId: 'cred-core',
      pollIntervalSeconds: 300,
      enabled: true,
      collectorChain: ['system', 'if_table', 'lldp'],
    },
  },
};
