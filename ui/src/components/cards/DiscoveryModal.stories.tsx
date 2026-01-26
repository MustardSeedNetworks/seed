import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { mockNetworkDiscoveryData } from '../../test/mockData';
import { DiscoveryModal } from './DiscoveryModal';

const meta = {
  title: 'Cards/DiscoveryModal',
  component: DiscoveryModal,
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof DiscoveryModal>;

export default meta;

type Story = StoryObj<typeof meta>;

export const WithDevices: Story = {
  render: () => {
    const [open, setOpen] = useState(true);
    return (
      <DiscoveryModal
        isOpen={open}
        onClose={() => setOpen(false)}
        data={mockNetworkDiscoveryData.withDevices}
        onScan={() => {}}
        onDeepScan={async () => {}}
      />
    );
  },
};
