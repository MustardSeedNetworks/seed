import type { Meta, StoryObj } from '@storybook/react-vite';
import { HelpCircle, Info, Settings } from 'lucide-react';
import { cn, spacing } from '../../styles/theme';
import { Tooltip } from './tooltip';

/**
 * Tooltips provide contextual help text on hover or focus.
 * Use them to explain icons, abbreviations, or UI elements.
 */
const meta: Meta<typeof Tooltip> = {
  title: 'UI/Tooltip',
  component: Tooltip,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    side: {
      control: 'radio',
      options: ['top', 'bottom'],
    },
    text: {
      control: 'text',
    },
  },
};

export default meta;
type Story = StoryObj<typeof Tooltip>;

export const Default: Story = {
  args: {
    text: 'This is helpful information',
    children: <Info className="w-5 h-5 text-text-secondary cursor-help" />,
  },
};

export const PositionTop: Story = {
  args: {
    text: 'Tooltip appears above the element',
    side: 'top',
    children: <span className="text-text-secondary cursor-help underline">Hover me (top)</span>,
  },
};

export const PositionBottom: Story = {
  args: {
    text: 'Tooltip appears below the element',
    side: 'bottom',
    children: <span className="text-text-secondary cursor-help underline">Hover me (bottom)</span>,
  },
};

export const WithIcon: Story = {
  args: {
    text: 'Click to access settings',
    children: (
      <button
        type="button"
        className={cn(spacing.pad.xs, 'rounded-lg bg-surface-raised hover:bg-surface-hover')}
      >
        <Settings className="w-5 h-5 text-text-secondary" />
      </button>
    ),
  },
};

export const LongContent: Story = {
  args: {
    text: 'This is a much longer tooltip that explains a complex concept in detail. It will wrap to multiple lines if needed.',
    children: <HelpCircle className="w-5 h-5 text-text-secondary cursor-help" />,
  },
};

export const InContext: Story = {
  render: () => (
    <div className={cn('flex items-center', spacing.gap.compact)}>
      <span className="text-text-primary">Upload limit</span>
      <Tooltip text="Maximum file size for uploads is 10MB">
        <HelpCircle className="w-4 h-4 text-text-muted cursor-help" />
      </Tooltip>
    </div>
  ),
};
