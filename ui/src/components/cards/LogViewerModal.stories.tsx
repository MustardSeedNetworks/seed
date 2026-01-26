import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { LogViewerModal } from './LogViewerModal';

const meta = {
  title: 'Cards/LogViewerModal',
  component: LogViewerModal,
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof LogViewerModal>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Open: Story = {
  render: () => {
    const [open, setOpen] = useState(true);
    return <LogViewerModal isOpen={open} onClose={() => setOpen(false)} />;
  },
};
