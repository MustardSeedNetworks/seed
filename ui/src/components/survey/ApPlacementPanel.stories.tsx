import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { ApLocation } from '../../hooks/useSurvey';
import { ApPlacementPanel } from './ApPlacementPanel';
import { sampleApLocations } from './storyData';

const meta = {
  title: 'Survey/ApPlacementPanel',
  component: ApPlacementPanel,
} satisfies Meta<typeof ApPlacementPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => {
    const [aps, setAps] = useState<ApLocation[]>(sampleApLocations);
    const [selected, setSelected] = useState<string | undefined>(aps[0]?.id);
    const [placementMode, setPlacementMode] = useState(false);
    return (
      <ApPlacementPanel
        apLocations={aps}
        onApLocationsChange={setAps}
        selectedApId={selected}
        onApSelect={setSelected}
        placementMode={placementMode}
        onPlacementModeChange={setPlacementMode}
      />
    );
  },
};
