import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { FloorPlan } from '../../hooks/useSurvey';
import { ScaleCalibrationPanel } from './ScaleCalibrationPanel';
import { sampleFloors } from './storyData';

const meta = {
  title: 'Survey/ScaleCalibrationPanel',
  component: ScaleCalibrationPanel,
} satisfies Meta<typeof ScaleCalibrationPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => {
    const [floorPlan, setFloorPlan] = useState<FloorPlan>(
      sampleFloors[0]?.floorPlan ?? { imageData: '', width: 800, height: 600, scaleM: 0.1 },
    );
    return (
      <ScaleCalibrationPanel
        floorPlan={floorPlan}
        onUpdate={(updates) => setFloorPlan((prev) => ({ ...prev, ...updates }))}
        onStartCalibration={() => {}}
      />
    );
  },
};
