import type { Meta, StoryFn, StoryObj } from '@storybook/react-vite';
import type React from 'react';
import { useState } from 'react';
import { Button } from './Button';
import { InputModal } from './InputModal';

const meta: Meta<typeof InputModal> = {
  title: 'UI/InputModal',
  component: InputModal,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'Single-field prompt dialog. Submits on Enter, cancels on Escape. Focuses and selects the field on open.',
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

export const Open: Story = {
  render: () => (
    <InputModal
      isOpen={true}
      onSubmit={() => undefined}
      onCancel={() => undefined}
      title="Rename profile"
      message="Pick a new name for this network profile."
      placeholder="Office gateway"
      defaultValue="Office gateway"
      submitLabel="Save"
    />
  ),
};

export const Closed: Story = {
  render: () => (
    <InputModal
      isOpen={false}
      onSubmit={() => undefined}
      onCancel={() => undefined}
      title="Hidden"
      message="Nothing should render in the closed state."
    />
  ),
};

export const DangerTone: Story = {
  render: () => (
    <InputModal
      isOpen={true}
      onSubmit={() => undefined}
      onCancel={() => undefined}
      title="Type DELETE to confirm"
      message="This will permanently remove the IP route table. Type DELETE to proceed."
      placeholder="DELETE"
      submitLabel="Delete"
      submitTone="red"
    />
  ),
};

export const Interactive: Story = {
  render: () => {
    const [open, setOpen] = useState(false);
    const [name, setName] = useState('Office gateway');
    return (
      <div class="space-y-3">
        <Button onClick={() => setOpen(true)}>Rename profile…</Button>
        <p class="text-sm text-text-muted">Current name: {name}</p>
        <InputModal
          isOpen={open}
          onSubmit={(value) => {
            setName(value);
            setOpen(false);
          }}
          onCancel={() => setOpen(false)}
          title="Rename profile"
          message="Pick a new name."
          defaultValue={name}
        />
      </div>
    );
  },
};
