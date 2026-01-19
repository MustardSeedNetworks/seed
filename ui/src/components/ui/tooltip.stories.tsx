import type { Meta, StoryObj } from '@storybook/react-vite';
import { HelpCircle, Info, Settings } from 'lucide-react';
import { cn, spacing } from '../../styles/theme';
import { Tooltip } from './Tooltip';

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
    position: {
      control: 'radio',
      options: ['top', 'bottom'],
    },
    content: {
      control: 'text',
    },
  },
};

export default meta;
type Story = StoryObj<typeof Tooltip>;

export const Default: Story = {
  args: {
    content: 'This is helpful information',
    children: <Info class="w-5 h-5 text-text-secondary cursor-help" />,
  },
};

export const PositionTop: Story = {
  args: {
    content: 'Tooltip appears above the element',
    position: 'top',
    children: <span class="text-text-secondary cursor-help underline">Hover me (top)</span>,
  },
};

export const PositionBottom: Story = {
  args: {
    content: 'Tooltip appears below the element',
    position: 'bottom',
    children: <span class="text-text-secondary cursor-help underline">Hover me (bottom)</span>,
  },
};

export const WithIcon: Story = {
  args: {
    content: 'Click to access settings',
    children: (
      <button
        type="button"
        class={cn(spacing.pad.xs, 'rounded-lg bg-surface-raised hover:bg-surface-hover')}
      >
        <Settings class="w-5 h-5 text-text-secondary" />
      </button>
    ),
  },
};

export const LongContent: Story = {
  args: {
    content:
      'This is a much longer tooltip that explains a complex concept in detail. It will wrap to multiple lines if needed.',
    children: <HelpCircle class="w-5 h-5 text-text-secondary cursor-help" />,
  },
};

export const InContext: Story = {
  render: () => (
    <div class={cn('flex items-center', spacing.gap.compact)}>
      <span class="text-text-primary">Upload limit</span>
      <Tooltip content="Maximum file size for uploads is 10MB">
        <HelpCircle class="w-4 h-4 text-text-muted cursor-help" />
      </Tooltip>
    </div>
  ),
};
