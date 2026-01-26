import type { Meta, StoryObj } from '@storybook/react-vite';
import type { Capabilities } from '../../hooks/useCapabilities';
import { CapabilityWarnings } from './CapabilityWarnings';

const meta = {
  title: 'App/CapabilityWarnings',
  component: CapabilityWarnings,
} satisfies Meta<typeof CapabilityWarnings>;

export default meta;

type Story = StoryObj<typeof meta>;

const okCaps: Capabilities = { icmpAvailable: true };
const missingCaps: Capabilities = { icmpAvailable: false };

export const Healthy: Story = {
  args: {
    capabilities: okCaps,
  },
};

export const MissingIcmp: Story = {
  args: {
    capabilities: missingCaps,
  },
};
