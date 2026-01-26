import type { Meta, StoryObj } from '@storybook/react-vite';
import type { PipelineStatus } from '../../hooks/usePipelineStatus';
import { PipelineProgress } from './PipelineProgress';

const sampleStatus: PipelineStatus = {
  state: 'scanning',
  runId: 'run-123',
  currentPhase: 'scanning',
  phaseNumber: 3,
  totalPhases: 4,
  enabledPhases: ['enumeration', 'resolution', 'scanning', 'assessment'],
  processedCount: 42,
  totalCount: 120,
  percentComplete: 35,
  currentTarget: '192.168.1.42',
  elapsedMs: 65234,
  estimatedRemainMs: 112000,
  devicesFound: 18,
  phaseDurations: {
    enumeration: 12000,
    resolution: 8000,
  },
  errors: [],
};

const meta = {
  title: 'Cards/PipelineProgress',
  component: PipelineProgress,
} satisfies Meta<typeof PipelineProgress>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Running: Story = {
  args: {
    status: sampleStatus,
    onCancel: () => {},
  },
};

export const Completed: Story = {
  args: {
    status: {
      ...sampleStatus,
      state: 'completed',
      currentPhase: 'assessment',
      phaseNumber: 4,
      percentComplete: 100,
      processedCount: 120,
      currentTarget: '',
      estimatedRemainMs: 0,
      phaseDurations: {
        enumeration: 12000,
        resolution: 8000,
        scanning: 45000,
        assessment: 15000,
      },
    },
  },
};
