/**
 * AutoSaveIndicator Storybook Stories
 *
 * Demonstrates the auto-save status indicator used throughout settings forms
 * to provide visual feedback for save operations.
 *
 * Variants:
 * - Idle: Hidden state (no unsaved changes)
 * - Saving: Shows "Saving..." with muted text
 * - Saved: Shows "Saved" with success color
 * - Error: Shows "Error" with error color
 */

import type { Decorator, Meta, StoryObj } from '@storybook/react-vite';
import { Toggle } from '../../../shared/Toggle';
import { cn, radius, spacing } from '../../../styles/theme';
import { AutoSaveIndicator } from './AutoSaveIndicator';

const noop = (): void => undefined;

const meta: Meta<typeof AutoSaveIndicator> = {
  title: 'Settings/AutoSaveIndicator',
  component: AutoSaveIndicator,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Small inline indicator showing the auto-save status of settings changes. Hidden when idle, displays status messages with color-coded feedback.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    status: {
      control: 'select',
      options: ['idle', 'saving', 'saved', 'error'],
      description: 'Current save status',
    },
  },
  decorators: [
    (Story: Parameters<Decorator>[0]) => (
      <div className={cn(spacing.pad.default, 'bg-surface-base')}>
        <div className={cn('flex items-center', spacing.gap.compact)}>
          <span className="body-small font-medium">Setting Name</span>
          <Story />
        </div>
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Idle state - indicator is hidden
 */
export const Idle: Story = {
  args: {
    status: 'idle',
  },
};

/**
 * Saving state - shows "Saving..." in muted text
 */
export const Saving: Story = {
  args: {
    status: 'saving',
  },
};

/**
 * Saved state - shows "Saved" in success color
 */
export const Saved: Story = {
  args: {
    status: 'saved',
  },
};

/**
 * Error state - shows "Error" in error color
 */
export const ErrorState: Story = {
  args: {
    status: 'error',
  },
};

/**
 * Multiple indicators showing different states
 */
export const AllStates: Story = {
  render: () => (
    <div className={cn('stack', spacing.pad.default, 'bg-surface-base')}>
      <div className={cn('flex items-center', spacing.gap.compact)}>
        <span className="body-small">Idle (hidden):</span>
        <AutoSaveIndicator status="idle" />
      </div>
      <div className={cn('flex items-center', spacing.gap.compact)}>
        <span className="body-small">Saving:</span>
        <AutoSaveIndicator status="saving" />
      </div>
      <div className={cn('flex items-center', spacing.gap.compact)}>
        <span className="body-small">Saved:</span>
        <AutoSaveIndicator status="saved" />
      </div>
      <div className={cn('flex items-center', spacing.gap.compact)}>
        <span className="body-small">Error:</span>
        <AutoSaveIndicator status="error" />
      </div>
    </div>
  ),
};

/**
 * In context with a typical settings field
 */
export const InContext: Story = {
  render: () => (
    <div className={cn('w-[400px]', spacing.pad.default, 'bg-surface-raised')}>
      <div className="stack">
        <Toggle
          label={
            <span className={cn('flex items-center', spacing.gap.compact)}>
              Enable Feature
              <AutoSaveIndicator status="saved" />
            </span>
          }
          checked={true}
          onChange={noop}
          className={cn(spacing.pad.sm, 'bg-surface-base border border-surface-border', radius.lg)}
        />
        <Toggle
          label={
            <span className={cn('flex items-center', spacing.gap.compact)}>
              Auto-refresh
              <AutoSaveIndicator status="saving" />
            </span>
          }
          checked={true}
          onChange={noop}
          className={cn(spacing.pad.sm, 'bg-surface-base border border-surface-border', radius.lg)}
        />
        <Toggle
          label={
            <span className={cn('flex items-center', spacing.gap.compact)}>
              Failed Setting
              <AutoSaveIndicator status="error" />
            </span>
          }
          checked={true}
          onChange={noop}
          className={cn(spacing.pad.sm, 'bg-surface-base border border-surface-border', radius.lg)}
        />
      </div>
    </div>
  ),
};
