import type { Meta, StoryFn, StoryObj } from '@storybook/react-vite';
import type React from 'react';
import { useState } from 'react';
import { Button } from './Button';
import { Modal, ModalBody, ModalFooter, ModalHeader } from './Modal';

const meta: Meta<typeof Modal> = {
  title: 'UI/Modal',
  component: Modal,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'Base modal primitive with focus trap, backdrop click and Escape close, and ModalHeader/Body/Footer slots.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    size: {
      control: 'radio',
      options: ['sm', 'md', 'lg', 'xl', 'full'],
    },
    showCloseButton: { control: 'boolean' },
    closeOnBackdropClick: { control: 'boolean' },
    closeOnEscape: { control: 'boolean' },
  },
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

export const Open: Story = {
  render: () => (
    <Modal isOpen={true} onClose={() => undefined} title="Modal title" size="md">
      <p class="text-text-secondary">
        Modal body. Tab is trapped inside the dialog; Escape closes when closeOnEscape is on.
      </p>
    </Modal>
  ),
};

export const Closed: Story = {
  render: () => (
    <Modal isOpen={false} onClose={() => undefined} title="Hidden">
      <p>Nothing should render in the closed state.</p>
    </Modal>
  ),
};

export const WithSlots: Story = {
  name: 'With ModalHeader/Body/Footer',
  render: () => (
    <Modal isOpen={true} onClose={() => undefined} size="lg">
      <ModalHeader>
        <h2 class="text-lg font-semibold text-text-primary">Edit DHCP profile</h2>
        <p class="text-sm text-text-muted">Changes apply to all interfaces using this profile.</p>
      </ModalHeader>
      <ModalBody>
        <p class="text-text-secondary">
          DNS, ARP, and TCP/UDP defaults are inherited unless overridden per interface.
        </p>
      </ModalBody>
      <ModalFooter>
        <Button variant="outline" onClick={() => undefined}>
          Cancel
        </Button>
        <Button onClick={() => undefined}>Save</Button>
      </ModalFooter>
    </Modal>
  ),
};

export const Interactive: Story = {
  render: () => {
    const [open, setOpen] = useState(false);
    return (
      <div class="space-y-3">
        <Button onClick={() => setOpen(true)}>Open modal</Button>
        <Modal isOpen={open} onClose={() => setOpen(false)} title="Hello" size="md">
          <p class="text-text-secondary">Click the backdrop or press Escape to close.</p>
          <div class="mt-4 flex justify-end">
            <Button onClick={() => setOpen(false)}>Close</Button>
          </div>
        </Modal>
      </div>
    );
  },
};

export const Sizes: Story = {
  render: () => (
    <Modal isOpen={true} onClose={() => undefined} title="Full size modal" size="full">
      <p class="text-text-secondary">
        Use size=&quot;full&quot; for content-heavy dialogs (reports, log viewers).
      </p>
    </Modal>
  ),
};
