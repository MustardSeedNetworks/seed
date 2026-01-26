import type { Meta, StoryObj } from '@storybook/react-vite';
import { VlanControl } from './VlanControl';

const meta = {
  title: 'Settings/VlanControl',
  component: VlanControl,
} satisfies Meta<typeof VlanControl>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
