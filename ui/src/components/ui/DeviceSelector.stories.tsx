import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { DeviceSelector } from './DeviceSelector';

const meta = {
  title: 'UI/DeviceSelector',
  component: DeviceSelector,
} satisfies Meta<typeof DeviceSelector>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => {
    const [value, setValue] = useState('192.168.1.1');
    return <DeviceSelector value={value} onChange={setValue} />;
  },
};

export const Disabled: Story = {
  render: () => <DeviceSelector value="192.168.1.1" onChange={() => {}} disabled />,
};
