import type { Meta, StoryObj } from '@storybook/react-vite';
import { MtuControl } from './MtuControl';

const meta = {
  title: 'Settings/MtuControl',
  component: MtuControl,
} satisfies Meta<typeof MtuControl>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
