import type { Decorator, Meta, StoryObj } from '@storybook/react-vite';
import { cn, layout, spacing } from '../../styles/theme';
import { SpeedGauge } from './SpeedGauge';

/**
 * SpeedGauge displays internet speed test results as an arc-based speedometer gauge.
 * Features auto-scaling (Mbps to Gbps), color-coded indicators, and running animation.
 */
const meta = {
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
    (Story: Parameters<Decorator>[0]) => (
      <div className={spacing.pad.xl}>
        <Story />
      </div>
    ),
  ],
  args: { value: 0 },
} satisfies Meta<typeof SpeedGauge>;

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
    <div className={layout.inline.spacious}>
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
    <div className={cn('flex items-end', spacing.gap.spacious)}>
      <div className={cn(layout.stack.default, 'items-center')}>
        <SpeedGauge value={125.5} maxValue={1000} label="Small" size="sm" />
        <span className="caption text-text-muted">100x60</span>
      </div>
      <div className={cn(layout.stack.default, 'items-center')}>
        <SpeedGauge value={125.5} maxValue={1000} label="Medium" size="md" />
        <span className="caption text-text-muted">140x85</span>
      </div>
      <div className={cn(layout.stack.default, 'items-center')}>
        <SpeedGauge value={125.5} maxValue={1000} label="Large" size="lg" />
        <span className="caption text-text-muted">180x110</span>
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
      <div className={cn('grid grid-cols-3', spacing.gap.spacious)}>
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
