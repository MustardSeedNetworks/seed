import type { Meta, StoryObj } from '@storybook/react-vite';
import { HeatmapLegend } from './HeatmapLegend';

const meta = {
  title: 'Survey/HeatmapLegend',
  component: HeatmapLegend,
} satisfies Meta<typeof HeatmapLegend>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Rssi: Story = {
  args: {
    metric: 'rssi',
    minValue: -90,
    maxValue: -30,
  },
};

export const Cochannel: Story = {
  args: {
    metric: 'cochannel',
    minValue: 0,
    maxValue: 8,
  },
};
