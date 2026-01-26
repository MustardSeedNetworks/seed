import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { ReportPreviewModal } from './ReportPreviewModal';
import { sampleReport } from './storyData';

const meta = {
  title: 'Survey/ReportPreviewModal',
  component: ReportPreviewModal,
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof ReportPreviewModal>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Open: Story = {
  render: () => {
    const [open, setOpen] = useState(true);
    return <ReportPreviewModal isOpen={open} onClose={() => setOpen(false)} report={sampleReport} />;
  },
};
