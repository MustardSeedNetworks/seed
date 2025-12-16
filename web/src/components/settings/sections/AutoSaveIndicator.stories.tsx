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

import type { Meta, StoryObj } from "@storybook/react-vite";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { spacing, radius } from "../../../styles/theme";

const meta: Meta<typeof AutoSaveIndicator> = {
  title: "Settings/AutoSaveIndicator",
  component: AutoSaveIndicator,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Small inline indicator showing the auto-save status of settings changes. Hidden when idle, displays status messages with color-coded feedback.",
      },
    },
  },
  tags: ["autodocs"],
  argTypes: {
    status: {
      control: "select",
      options: ["idle", "saving", "saved", "error"],
      description: "Current save status",
    },
  },
  decorators: [
    (Story) => (
      <div className={`${spacing.pad.default} bg-surface-base`}>
        <div className={`flex items-center ${spacing.gap.compact}`}>
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
    status: "idle",
  },
};

/**
 * Saving state - shows "Saving..." in muted text
 */
export const Saving: Story = {
  args: {
    status: "saving",
  },
};

/**
 * Saved state - shows "Saved" in success color
 */
export const Saved: Story = {
  args: {
    status: "saved",
  },
};

/**
 * Error state - shows "Error" in error color
 */
export const Error: Story = {
  args: {
    status: "error",
  },
};

/**
 * Multiple indicators showing different states
 */
export const AllStates: Story = {
  render: () => (
    <div className={`stack ${spacing.pad.default} bg-surface-base`}>
      <div className={`flex items-center ${spacing.gap.compact}`}>
        <span className="body-small">Idle (hidden):</span>
        <AutoSaveIndicator status="idle" />
      </div>
      <div className={`flex items-center ${spacing.gap.compact}`}>
        <span className="body-small">Saving:</span>
        <AutoSaveIndicator status="saving" />
      </div>
      <div className={`flex items-center ${spacing.gap.compact}`}>
        <span className="body-small">Saved:</span>
        <AutoSaveIndicator status="saved" />
      </div>
      <div className={`flex items-center ${spacing.gap.compact}`}>
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
    <div className={`w-[400px] ${spacing.pad.default} bg-surface-raised`}>
      <div className="stack">
        <label
          className={`flex items-center justify-between ${spacing.pad.sm} bg-surface-base border border-surface-border ${radius.lg}`}
        >
          <div className={`flex items-center ${spacing.gap.compact}`}>
            <span className="body-small text-text-primary font-medium">Enable Feature</span>
            <AutoSaveIndicator status="saved" />
          </div>
          <input type="checkbox" checked readOnly className="w-4 h-4" />
        </label>
        <label
          className={`flex items-center justify-between ${spacing.pad.sm} bg-surface-base border border-surface-border ${radius.lg}`}
        >
          <div className={`flex items-center ${spacing.gap.compact}`}>
            <span className="body-small text-text-primary font-medium">Auto-refresh</span>
            <AutoSaveIndicator status="saving" />
          </div>
          <input type="checkbox" checked readOnly className="w-4 h-4" />
        </label>
        <label
          className={`flex items-center justify-between ${spacing.pad.sm} bg-surface-base border border-surface-border ${radius.lg}`}
        >
          <div className={`flex items-center ${spacing.gap.compact}`}>
            <span className="body-small text-text-primary font-medium">Failed Setting</span>
            <AutoSaveIndicator status="error" />
          </div>
          <input type="checkbox" checked readOnly className="w-4 h-4" />
        </label>
      </div>
    </div>
  ),
};
