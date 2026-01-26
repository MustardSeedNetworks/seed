import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { Floor } from '../../hooks/useSurvey';
import { FloorSelector } from './FloorSelector';
import { sampleFloors } from './storyData';

const meta = {
  title: 'Survey/FloorSelector',
  component: FloorSelector,
} satisfies Meta<typeof FloorSelector>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => {
    const [floors, setFloors] = useState<Floor[]>(sampleFloors);
    const [active, setActive] = useState<string | undefined>(floors[0]?.id);
    return (
      <FloorSelector
        floors={floors}
        activeFloorId={active}
        onSelectFloor={setActive}
        onAddFloor={async (name, level) => {
          setFloors((prev) => [
            ...prev,
            {
              id: `floor-${prev.length + 1}`,
              name,
              level,
              floorPlan: { imageData: '', width: 800, height: 600, scaleM: 0.1 },
            },
          ]);
        }}
        onDeleteFloor={async (floorId) => {
          setFloors((prev) => prev.filter((floor) => floor.id !== floorId));
          setActive((prev) => (prev === floorId ? floors[0]?.id : prev));
        }}
        onRenameFloor={async (floorId, name, level) => {
          setFloors((prev) =>
            prev.map((floor) => (floor.id === floorId ? { ...floor, name, level } : floor)),
          );
        }}
      />
    );
  },
};
