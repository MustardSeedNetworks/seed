import type { Meta, StoryFn, StoryObj } from '@storybook/react-vite';
import type React from 'react';
import { useState } from 'react';
import { Button } from './Button';
import { ConfirmModal } from './ConfirmModal';

const meta: Meta<typeof ConfirmModal> = {
  title: 'UI/ConfirmModal',
  component: ConfirmModal,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: 'Confirmation dialog with a triangle icon and two buttons (cancel + confirm).',
      },
    },
  },
  tags: ['autodocs'],
  decorators: [
    (StoryComponent: StoryFn): React.ReactElement => (
      <div class="min-h-[60vh] p-4">
        <StoryComponent />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const DestructiveOpen: Story = {
  render: () => (
    <ConfirmModal
      isOpen={true}
      onConfirm={() => undefined}
      onCancel={() => undefined}
      title="Delete profile?"
      message="This will permanently remove the network profile and its history."
      confirmLabel="Delete"
      cancelLabel="Cancel"
      confirmTone="red"
    />
  ),
};

export const InfoOpen: Story = {
  render: () => (
    <ConfirmModal
      isOpen={true}
      onConfirm={() => undefined}
      onCancel={() => undefined}
      title="Apply DHCP changes?"
      message="The gateway will renew its lease. The interface may go down briefly."
      confirmLabel="Apply"
      cancelLabel="Cancel"
      confirmTone="blue"
    />
  ),
};

export const SuccessOpen: Story = {
  render: () => (
    <ConfirmModal
      isOpen={true}
      onConfirm={() => undefined}
      onCancel={() => undefined}
      title="Start the RFC 2544 benchmark?"
      message="This will run a 60-second throughput test on eth0."
      confirmLabel="Start"
      cancelLabel="Cancel"
      confirmTone="green"
    />
  ),
};

export const Closed: Story = {
  render: () => (
    <ConfirmModal
      isOpen={false}
      onConfirm={() => undefined}
      onCancel={() => undefined}
      title="Hidden dialog"
      message="This story documents the closed state — nothing should render."
    />
  ),
};

export const Interactive: Story = {
  render: () => {
    const [open, setOpen] = useState(false);
    return (
      <div class="space-y-4">
        <Button tone="red" onClick={() => setOpen(true)}>
          Delete profile…
        </Button>
        <ConfirmModal
          isOpen={open}
          onConfirm={() => setOpen(false)}
          onCancel={() => setOpen(false)}
          title="Delete profile?"
          message="This action cannot be undone."
          confirmLabel="Delete"
          confirmTone="red"
        />
      </div>
    );
  },
};
