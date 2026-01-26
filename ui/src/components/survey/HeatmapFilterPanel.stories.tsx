import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { HeatmapFilter, HeatmapMetric } from '../../hooks/useSurvey';
import { HeatmapFilterPanel } from './HeatmapFilterPanel';
import { sampleApLocations, samplePassiveSamples } from './storyData';

const meta = {
  title: 'Survey/HeatmapFilterPanel',
  component: HeatmapFilterPanel,
} satisfies Meta<typeof HeatmapFilterPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Passive: Story = {
  render: () => {
    const [metric, setMetric] = useState<HeatmapMetric>('rssi');
    const [filter, setFilter] = useState<HeatmapFilter | undefined>(undefined);
    return (
      <HeatmapFilterPanel
        metric={metric}
        onMetricChange={setMetric}
        filter={filter}
        onFilterChange={setFilter}
        samples={samplePassiveSamples}
        surveyType="passive"
        apLocations={sampleApLocations}
      />
    );
  },
};
