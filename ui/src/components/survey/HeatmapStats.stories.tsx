import type { Meta, StoryObj } from '@storybook/react-vite';
import { HeatmapStats } from './HeatmapStats';
import { samplePassiveSamples } from './storyData';

const meta = {
  title: 'Survey/HeatmapStats',
  component: HeatmapStats,
} satisfies Meta<typeof HeatmapStats>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Rssi: Story = {
  args: {
    samples: samplePassiveSamples,
    metric: 'rssi',
  },
};

export const Snr: Story = {
  args: {
    samples: samplePassiveSamples,
    metric: 'snr',
  },
};
