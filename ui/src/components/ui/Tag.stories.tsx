import type { Meta, StoryFn, StoryObj } from '@storybook/react-vite';
import type React from 'react';
import { Tag } from './Tag';

const meta: Meta<typeof Tag> = {
  title: 'UI/Tag',
  component: Tag,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Inline label/badge primitive. Color scheme maps to the shared status + brand tokens, so dark/light parity is automatic.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    colorScheme: {
      control: 'select',
      options: ['gray', 'red', 'green', 'blue', 'yellow', 'purple', 'violet', 'cyan'],
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

export const Default: Story = {
  args: {
    children: 'Tag',
  },
};

export const Green: Story = {
  args: {
    colorScheme: 'green',
    children: 'Online',
  },
};

export const Red: Story = {
  args: {
    colorScheme: 'red',
    children: 'Failed',
  },
};

export const Yellow: Story = {
  args: {
    colorScheme: 'yellow',
    children: 'Warning',
  },
};

export const Blue: Story = {
  args: {
    colorScheme: 'blue',
    children: 'Info',
  },
};

export const AllColors: Story = {
  render: () => (
    <div class="flex flex-wrap items-center gap-2">
      <Tag colorScheme="gray">Gray</Tag>
      <Tag colorScheme="red">Red</Tag>
      <Tag colorScheme="green">Green</Tag>
      <Tag colorScheme="blue">Blue</Tag>
      <Tag colorScheme="yellow">Yellow</Tag>
      <Tag colorScheme="purple">Purple</Tag>
      <Tag colorScheme="violet">Violet</Tag>
      <Tag colorScheme="cyan">Cyan</Tag>
    </div>
  ),
};

export const NetworkStatuses: Story = {
  render: () => (
    <div class="flex flex-wrap items-center gap-2">
      <Tag colorScheme="green">DHCP active</Tag>
      <Tag colorScheme="blue">DNS resolving</Tag>
      <Tag colorScheme="yellow">ARP slow</Tag>
      <Tag colorScheme="red">TCP timeout</Tag>
      <Tag colorScheme="cyan">UDP healthy</Tag>
      <Tag colorScheme="violet">WiFi connected</Tag>
    </div>
  ),
};
