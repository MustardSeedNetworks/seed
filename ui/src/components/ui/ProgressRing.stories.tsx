import type { Meta, StoryObj } from '@storybook/react-vite';
import { cn, spacing } from '../../styles/theme';
import { ProgressRing } from './SpeedGauge';

/**
 * ProgressRing is the circular progress indicator exported beside SpeedGauge.
 */
const meta = {
  title: 'UI/SpeedGauge/ProgressRing',
  component: ProgressRing,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Circular progress indicator for displaying percentage-based progress with optional label.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    progress: {
      control: { type: 'number', min: 0, max: 100 },
      description: 'Progress percentage (0-100)',
    },
    size: {
      control: { type: 'number', min: 24, max: 200 },
      description: 'Diameter of the ring in pixels',
    },
    strokeWidth: {
      control: { type: 'number', min: 2, max: 10 },
      description: 'Width of the progress ring stroke',
    },
    label: {
      control: 'text',
      description: 'Optional label displayed below the ring',
    },
  },
  args: { progress: 0 },
} satisfies Meta<typeof ProgressRing>;

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Progress ring at 0%
 */
export const Empty: Story = {
  args: {
    progress: 0,
    size: 64,
    strokeWidth: 4,
    label: 'Starting',
  },
};

/**
 * Progress ring at 25%
 */
export const Quarter: Story = {
  args: {
    progress: 25,
    size: 64,
    strokeWidth: 4,
    label: 'In Progress',
  },
};

/**
 * Progress ring at 50%
 */
export const Half: Story = {
  args: {
    progress: 50,
    size: 64,
    strokeWidth: 4,
    label: 'Halfway',
  },
};

/**
 * Progress ring at 75%
 */
export const ThreeQuarters: Story = {
  args: {
    progress: 75,
    size: 64,
    strokeWidth: 4,
    label: 'Almost Done',
  },
};

/**
 * Progress ring at 100%
 */
export const Complete: Story = {
  args: {
    progress: 100,
    size: 64,
    strokeWidth: 4,
    label: 'Complete',
  },
};

/**
 * Large progress ring
 */
export const Large: Story = {
  args: {
    progress: 65,
    size: 120,
    strokeWidth: 8,
    label: 'Download Progress',
  },
};

/**
 * Small progress ring
 */
export const Small: Story = {
  args: {
    progress: 42,
    size: 32,
    strokeWidth: 3,
  },
};

/**
 * Progress ring comparison showing multiple states
 */
export const States: Story = {
  render: () => (
    <div className={cn('flex items-end', spacing.gap.spacious)}>
      <ProgressRing progress={0} size={48} label="0%" />
      <ProgressRing progress={25} size={48} label="25%" />
      <ProgressRing progress={50} size={48} label="50%" />
      <ProgressRing progress={75} size={48} label="75%" />
      <ProgressRing progress={100} size={48} label="100%" />
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: 'All progress states from 0% to 100% displayed side-by-side.',
      },
    },
  },
};
