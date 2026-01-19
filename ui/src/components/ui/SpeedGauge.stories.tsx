import type { Meta, StoryFn, StoryObj } from '@storybook/react-vite';
import type React from 'react';
import { cn, layout, spacing } from '../../styles/theme';
import { ProgressRing, PulsingDot, SpeedGauge } from './SpeedGauge';

/**
 * SpeedGauge displays internet speed test results as an arc-based speedometer gauge.
 * Features auto-scaling (Mbps to Gbps), color-coded indicators, and running animation.
 *
 * Also includes ProgressRing for circular progress bars and PulsingDot for animated indicators.
 */
const meta: Meta<typeof SpeedGauge> = {
  title: 'UI/SpeedGauge',
  component: SpeedGauge,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Visual speedometer gauge for displaying internet speed test results with color-coded indicators based on performance percentage.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    value: {
      control: { type: 'number', min: 0, max: 2000 },
      description: 'Current speed in Mbps',
    },
    maxValue: {
      control: { type: 'number', min: 100, max: 2000 },
      description: 'Maximum gauge scale value in Mbps',
    },
    label: {
      control: 'text',
      description: "Label displayed above gauge (e.g., 'Download', 'Upload')",
    },
    unit: {
      control: 'text',
      description: "Unit of measurement (defaults to 'Mbps')",
    },
    isRunning: {
      control: 'boolean',
      description: 'Shows pulsing animation when test is running',
    },
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg'],
      description: 'Gauge size variant',
    },
  },
  decorators: [
    (StoryComponent: StoryFn): React.ReactElement => (
      <div class={spacing.pad.xl}>
        <StoryComponent />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Low speed example (< 10 Mbps) - shows red/poor performance indicator
 */
export const LowSpeed: Story = {
  args: {
    value: 8.5,
    maxValue: 1000,
    label: 'Download',
    isRunning: false,
    size: 'md',
  },
  parameters: {
    docs: {
      description: {
        story: 'Low speed scenario showing red indicator for poor performance (< 1% of max).',
      },
    },
  },
};

/**
 * Medium speed example (10-100 Mbps) - shows yellow/warning indicator
 */
export const MediumSpeed: Story = {
  args: {
    value: 45.2,
    maxValue: 1000,
    label: 'Download',
    isRunning: false,
    size: 'md',
  },
  parameters: {
    docs: {
      description: {
        story: 'Medium speed scenario showing yellow indicator for moderate performance.',
      },
    },
  },
};

/**
 * High speed example (100-1000 Mbps) - shows green/good indicator
 */
export const HighSpeed: Story = {
  args: {
    value: 250.8,
    maxValue: 1000,
    label: 'Download',
    isRunning: false,
    size: 'md',
  },
  parameters: {
    docs: {
      description: {
        story: 'High speed scenario showing green indicator for good performance.',
      },
    },
  },
};

/**
 * Gigabit speed example (> 1000 Mbps) - auto-converts to Gbps display
 */
export const GigabitSpeed: Story = {
  args: {
    value: 1250.5,
    maxValue: 2000,
    label: 'Download',
    isRunning: false,
    size: 'md',
  },
  parameters: {
    docs: {
      description: {
        story:
          'Gigabit speed scenario demonstrating automatic conversion from Mbps to Gbps when value exceeds 1000.',
      },
    },
  },
};

/**
 * Zero speed - initial state before test begins
 */
export const ZeroSpeed: Story = {
  args: {
    value: 0,
    maxValue: 1000,
    label: 'Download',
    isRunning: false,
    size: 'md',
  },
  parameters: {
    docs: {
      description: {
        story: 'Initial state showing zero speed before test execution.',
      },
    },
  },
};

/**
 * Maximum speed - gauge at 100% capacity
 */
export const MaximumSpeed: Story = {
  args: {
    value: 1000,
    maxValue: 1000,
    label: 'Download',
    isRunning: false,
    size: 'md',
  },
  parameters: {
    docs: {
      description: {
        story: 'Maximum capacity showing gauge at 100% with full arc filled.',
      },
    },
  },
};

/**
 * Running animation - shows pulsing effect during active test
 */
export const RunningAnimation: Story = {
  args: {
    value: 150.5,
    maxValue: 1000,
    label: 'Testing',
    isRunning: true,
    size: 'md',
  },
  parameters: {
    docs: {
      description: {
        story:
          'Active test scenario with pulsing animation. The gauge pulses to indicate testing in progress.',
      },
    },
  },
};

/**
 * Running with zero value - initial test state
 */
export const RunningInitial: Story = {
  args: {
    value: 0,
    maxValue: 1000,
    label: 'Testing',
    isRunning: true,
    size: 'md',
  },
  parameters: {
    docs: {
      description: {
        story: 'Test starting state showing pulsing animation with dash (—) placeholder for value.',
      },
    },
  },
};

/**
 * Small size variant (100x60)
 */
export const SmallSize: Story = {
  args: {
    value: 125.7,
    maxValue: 1000,
    label: 'Download',
    isRunning: false,
    size: 'sm',
  },
  parameters: {
    docs: {
      description: {
        story: 'Compact gauge variant for space-constrained layouts (100x60 pixels).',
      },
    },
  },
};

/**
 * Large size variant (180x110)
 */
export const LargeSize: Story = {
  args: {
    value: 325.4,
    maxValue: 1000,
    label: 'Download',
    isRunning: false,
    size: 'lg',
  },
  parameters: {
    docs: {
      description: {
        story: 'Expanded gauge variant for prominent display (180x110 pixels).',
      },
    },
  },
};

/**
 * Upload speed comparison - typically lower than download
 */
export const UploadSpeed: Story = {
  args: {
    value: 35.2,
    maxValue: 1000,
    label: 'Upload',
    isRunning: false,
    size: 'md',
  },
  parameters: {
    docs: {
      description: {
        story: 'Upload speed example showing typically lower speeds compared to download.',
      },
    },
  },
};

/**
 * Side-by-side comparison of download and upload gauges
 */
export const DownloadUploadPair: Story = {
  render: () => (
    <div class={layout.inline.spacious}>
      <SpeedGauge value={450.8} maxValue={1000} label="Download" size="md" />
      <SpeedGauge value={52.3} maxValue={1000} label="Upload" size="md" />
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: 'Common use case showing download and upload speeds side-by-side for comparison.',
      },
    },
  },
};

/**
 * All size variants displayed together
 */
export const AllSizes: Story = {
  render: () => (
    <div class={cn('flex items-end', spacing.gap.spacious)}>
      <div class={cn(layout.stack.default, 'items-center')}>
        <SpeedGauge value={125.5} maxValue={1000} label="Small" size="sm" />
        <span class="caption text-text-muted">100x60</span>
      </div>
      <div class={cn(layout.stack.default, 'items-center')}>
        <SpeedGauge value={125.5} maxValue={1000} label="Medium" size="md" />
        <span class="caption text-text-muted">140x85</span>
      </div>
      <div class={cn(layout.stack.default, 'items-center')}>
        <SpeedGauge value={125.5} maxValue={1000} label="Large" size="lg" />
        <span class="caption text-text-muted">180x110</span>
      </div>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: 'All three size variants displayed together for comparison.',
      },
    },
  },
};

/**
 * Speed progression animation demonstration
 */
export const SpeedProgression: Story = {
  render: () => {
    const speeds = [0, 50, 150, 350, 650, 950];
    return (
      <div class={cn('grid grid-cols-3', spacing.gap.spacious)}>
        {speeds.map((speed) => (
          <SpeedGauge key={speed} value={speed} maxValue={1000} label={`${speed} Mbps`} size="md" />
        ))}
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          'Demonstrates gauge appearance across different speed ranges, showing color transitions from red to yellow to green.',
      },
    },
  },
};

// ============================================================================
// ProgressRing Stories
// ============================================================================

const PROGRESS_RING_META: Meta<typeof ProgressRing> = {
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
};

export const ProgressRingStories: Meta<typeof ProgressRing> = {
  ...PROGRESS_RING_META,
};

/**
 * Progress ring at 0%
 */
export const ProgressRingEmpty: StoryObj<typeof ProgressRing> = {
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
export const ProgressRingQuarter: StoryObj<typeof ProgressRing> = {
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
export const ProgressRingHalf: StoryObj<typeof ProgressRing> = {
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
export const ProgressRingThreeQuarters: StoryObj<typeof ProgressRing> = {
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
export const ProgressRingComplete: StoryObj<typeof ProgressRing> = {
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
export const ProgressRingLarge: StoryObj<typeof ProgressRing> = {
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
export const ProgressRingSmall: StoryObj<typeof ProgressRing> = {
  args: {
    progress: 42,
    size: 32,
    strokeWidth: 3,
  },
};

/**
 * Progress ring comparison showing multiple states
 */
export const ProgressRingStates: StoryObj<typeof ProgressRing> = {
  render: () => (
    <div class={cn('flex items-end', spacing.gap.spacious)}>
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

// ============================================================================
// PulsingDot Stories
// ============================================================================

const PULSING_DOT_META: Meta<typeof PulsingDot> = {
  title: 'UI/SpeedGauge/PulsingDot',
  component: PulsingDot,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Animated pulsing dot indicator for showing active/in-progress states with different color variants.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    color: {
      control: 'select',
      options: ['primary', 'success', 'warning', 'error'],
      description: 'Dot color variant based on status',
    },
    size: {
      control: 'select',
      options: ['sm', 'md'],
      description: 'Dot size variant',
    },
  },
};

export const PulsingDotStories: Meta<typeof PulsingDot> = {
  ...PULSING_DOT_META,
};

/**
 * Primary color pulsing dot (default)
 */
export const PulsingDotPrimary: StoryObj<typeof PulsingDot> = {
  args: {
    color: 'primary',
    size: 'md',
  },
};

/**
 * Success color pulsing dot
 */
export const PulsingDotSuccess: StoryObj<typeof PulsingDot> = {
  args: {
    color: 'success',
    size: 'md',
  },
};

/**
 * Warning color pulsing dot
 */
export const PulsingDotWarning: StoryObj<typeof PulsingDot> = {
  args: {
    color: 'warning',
    size: 'md',
  },
};

/**
 * Error color pulsing dot
 */
export const PulsingDotError: StoryObj<typeof PulsingDot> = {
  args: {
    color: 'error',
    size: 'md',
  },
};

/**
 * Small pulsing dot
 */
export const PulsingDotSmall: StoryObj<typeof PulsingDot> = {
  args: {
    color: 'primary',
    size: 'sm',
  },
};

/**
 * All pulsing dot variants displayed together
 */
export const PulsingDotAllColors: StoryObj<typeof PulsingDot> = {
  render: () => (
    <div class={cn('flex items-center', spacing.gap.spacious)}>
      <div class={cn(layout.stack.default, 'items-center')}>
        <PulsingDot color="primary" size="md" />
        <span class="caption text-text-muted">Primary</span>
      </div>
      <div class={cn(layout.stack.default, 'items-center')}>
        <PulsingDot color="success" size="md" />
        <span class="caption text-text-muted">Success</span>
      </div>
      <div class={cn(layout.stack.default, 'items-center')}>
        <PulsingDot color="warning" size="md" />
        <span class="caption text-text-muted">Warning</span>
      </div>
      <div class={cn(layout.stack.default, 'items-center')}>
        <PulsingDot color="error" size="md" />
        <span class="caption text-text-muted">Error</span>
      </div>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: 'All color variants of the pulsing dot indicator.',
      },
    },
  },
};

/**
 * Pulsing dots in context - showing active status indicators
 */
export const PulsingDotInContext: StoryObj<typeof PulsingDot> = {
  render: () => (
    <div class={spacing.section.default}>
      <div
        class={cn(
          layout.inline.comfortable,
          spacing.pad.sm,
          'bg-surface-raised border border-surface-border rounded-lg',
        )}
      >
        <PulsingDot color="primary" size="sm" />
        <span class="body-small text-text-primary">Network scan in progress...</span>
      </div>
      <div
        class={cn(
          layout.inline.comfortable,
          spacing.pad.sm,
          'bg-surface-raised border border-surface-border rounded-lg',
        )}
      >
        <PulsingDot color="success" size="sm" />
        <span class="body-small text-text-primary">Speed test running</span>
      </div>
      <div
        class={cn(
          layout.inline.comfortable,
          spacing.pad.sm,
          'bg-surface-raised border border-surface-border rounded-lg',
        )}
      >
        <PulsingDot color="warning" size="sm" />
        <span class="body-small text-text-primary">Waiting for response...</span>
      </div>
      <div
        class={cn(
          layout.inline.comfortable,
          spacing.pad.sm,
          'bg-surface-raised border border-surface-border rounded-lg',
        )}
      >
        <PulsingDot color="error" size="sm" />
        <span class="body-small text-text-primary">Connection unstable</span>
      </div>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: 'Real-world usage examples showing pulsing dots as status indicators in cards.',
      },
    },
  },
};
