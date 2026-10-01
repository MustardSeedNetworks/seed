import type { Meta, StoryObj } from '@storybook/react-vite';
import { cn, layout, spacing } from '../../styles/theme';
import { PulsingDot } from './SpeedGauge';

/**
 * PulsingDot is the animated status indicator exported beside SpeedGauge.
 */
const meta = {
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
} satisfies Meta<typeof PulsingDot>;

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Primary color pulsing dot (default)
 */
export const Primary: Story = {
  args: {
    color: 'primary',
    size: 'md',
  },
};

/**
 * Success color pulsing dot
 */
export const Success: Story = {
  args: {
    color: 'success',
    size: 'md',
  },
};

/**
 * Warning color pulsing dot
 */
export const Warning: Story = {
  args: {
    color: 'warning',
    size: 'md',
  },
};

/**
 * Error color pulsing dot
 */
export const ErrorState: Story = {
  args: {
    color: 'error',
    size: 'md',
  },
};

/**
 * Small pulsing dot
 */
export const Small: Story = {
  args: {
    color: 'primary',
    size: 'sm',
  },
};

/**
 * All pulsing dot variants displayed together
 */
export const AllColors: Story = {
  render: () => (
    <div className={cn('flex items-center', spacing.gap.spacious)}>
      <div className={cn(layout.stack.default, 'items-center')}>
        <PulsingDot color="primary" size="md" />
        <span className="caption text-text-muted">Primary</span>
      </div>
      <div className={cn(layout.stack.default, 'items-center')}>
        <PulsingDot color="success" size="md" />
        <span className="caption text-text-muted">Success</span>
      </div>
      <div className={cn(layout.stack.default, 'items-center')}>
        <PulsingDot color="warning" size="md" />
        <span className="caption text-text-muted">Warning</span>
      </div>
      <div className={cn(layout.stack.default, 'items-center')}>
        <PulsingDot color="error" size="md" />
        <span className="caption text-text-muted">Error</span>
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
export const InContext: Story = {
  render: () => (
    <div className={spacing.section.default}>
      <div
        className={cn(
          layout.inline.comfortable,
          spacing.pad.sm,
          'bg-surface-raised border border-surface-border rounded-lg',
        )}
      >
        <PulsingDot color="primary" size="sm" />
        <span className="body-small text-text-primary">Network scan in progress...</span>
      </div>
      <div
        className={cn(
          layout.inline.comfortable,
          spacing.pad.sm,
          'bg-surface-raised border border-surface-border rounded-lg',
        )}
      >
        <PulsingDot color="success" size="sm" />
        <span className="body-small text-text-primary">Speed test running</span>
      </div>
      <div
        className={cn(
          layout.inline.comfortable,
          spacing.pad.sm,
          'bg-surface-raised border border-surface-border rounded-lg',
        )}
      >
        <PulsingDot color="warning" size="sm" />
        <span className="body-small text-text-primary">Waiting for response...</span>
      </div>
      <div
        className={cn(
          layout.inline.comfortable,
          spacing.pad.sm,
          'bg-surface-raised border border-surface-border rounded-lg',
        )}
      >
        <PulsingDot color="error" size="sm" />
        <span className="body-small text-text-primary">Connection unstable</span>
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
