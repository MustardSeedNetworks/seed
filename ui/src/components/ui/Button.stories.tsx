import type { Meta, StoryFn, StoryObj } from '@storybook/react-vite';
import { Download, Plus, Trash2 } from 'lucide-react';
import type React from 'react';
import { Button } from './Button';

const meta: Meta<typeof Button> = {
  title: 'UI/Button',
  component: Button,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Canonical button primitive shared by seed/stem/niac. Variant + tone selects the surface; size scales padding and text.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    variant: {
      control: 'radio',
      options: ['solid', 'outline', 'ghost', 'secondary'],
      description: 'Visual surface.',
    },
    tone: {
      control: 'select',
      options: ['violet', 'red', 'green', 'blue', 'gray'],
      description: 'Semantic color — violet is the brand default.',
    },
    size: {
      control: 'radio',
      options: ['xs', 'sm', 'md', 'lg'],
      description: 'Padding/text size.',
    },
    loading: {
      control: 'boolean',
      description: 'Replaces leftIcon with a spinner and disables the button.',
    },
    disabled: {
      control: 'boolean',
    },
  },
  decorators: [
    (StoryComponent: StoryFn): React.ReactElement => (
      <div class="p-2">
        <StoryComponent />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Primary: Story = {
  args: {
    variant: 'solid',
    tone: 'violet',
    children: 'Run diagnostics',
  },
};

export const Secondary: Story = {
  args: {
    variant: 'secondary',
    tone: 'gray',
    children: 'Cancel',
  },
};

export const Ghost: Story = {
  args: {
    variant: 'ghost',
    tone: 'violet',
    children: 'View details',
  },
};

export const Danger: Story = {
  args: {
    variant: 'solid',
    tone: 'red',
    leftIcon: <Trash2 class="h-4 w-4" />,
    children: 'Delete profile',
  },
};

export const Outline: Story = {
  args: {
    variant: 'outline',
    tone: 'violet',
    children: 'Outlined',
  },
};

export const WithIcons: Story = {
  args: {
    variant: 'solid',
    tone: 'green',
    leftIcon: <Plus class="h-4 w-4" />,
    children: 'Add server',
  },
};

export const Loading: Story = {
  args: {
    variant: 'solid',
    tone: 'violet',
    loading: true,
    children: 'Saving…',
  },
};

export const Disabled: Story = {
  args: {
    variant: 'solid',
    tone: 'violet',
    disabled: true,
    children: 'Unavailable',
  },
};

export const AllVariants: Story = {
  render: () => (
    <div class="space-y-4">
      <div class="flex flex-wrap items-center gap-2">
        <Button variant="solid" tone="violet">
          Solid
        </Button>
        <Button variant="outline" tone="violet">
          Outline
        </Button>
        <Button variant="ghost" tone="violet">
          Ghost
        </Button>
        <Button variant="secondary" tone="gray">
          Secondary
        </Button>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <Button variant="solid" tone="green" leftIcon={<Download class="h-4 w-4" />}>
          Success
        </Button>
        <Button variant="solid" tone="blue">
          Info
        </Button>
        <Button variant="solid" tone="red">
          Danger
        </Button>
        <Button variant="solid" tone="gray">
          Neutral
        </Button>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <Button size="xs">XS</Button>
        <Button size="sm">Small</Button>
        <Button size="md">Medium</Button>
        <Button size="lg">Large</Button>
      </div>
    </div>
  ),
};
