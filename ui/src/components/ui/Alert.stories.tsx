import type { Meta, StoryFn, StoryObj } from '@storybook/react-vite';
import type React from 'react';
import { Alert } from './Alert';

const meta: Meta<typeof Alert> = {
  title: 'UI/Alert',
  component: Alert,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Inline status message used for transient errors, warnings, and confirmations. Resolves through the shared status color tokens so dark/light parity comes for free.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    status: {
      control: 'select',
      options: ['success', 'error', 'warning', 'info'],
      description: 'Visual status — drives icon and color tokens.',
    },
    onDismiss: {
      action: 'dismissed',
      description: 'Optional callback. When provided, renders a close button.',
    },
  },
  decorators: [
    (StoryComponent: StoryFn): React.ReactElement => (
      <div class="w-[440px]">
        <StoryComponent />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Info: Story = {
  args: {
    status: 'info',
    children: 'New firmware available for the gateway. Schedule an update during off-hours.',
  },
};

export const Success: Story = {
  args: {
    status: 'success',
    children: 'DHCP lease renewed successfully on eth0.',
  },
};

export const Warning: Story = {
  args: {
    status: 'warning',
    children: 'Wi-Fi signal weak (-72 dBm). Consider relocating the access point.',
  },
};

export const ErrorStatus: Story = {
  name: 'Error',
  args: {
    status: 'error',
    children: 'TCP port 8443 unreachable. Check the firewall rules.',
  },
};

export const Dismissable: Story = {
  args: {
    status: 'info',
    children: 'DNS resolution for 8.8.8.8 completed in 12ms.',
    onDismiss: () => undefined,
  },
};

export const AllVariants: Story = {
  render: () => (
    <div class="space-y-3">
      <Alert status="info">Info: ARP cache contains 24 entries.</Alert>
      <Alert status="success">Success: RFC 2544 benchmark passed.</Alert>
      <Alert status="warning">Warning: UDP retransmits exceeded threshold.</Alert>
      <Alert status="error">Error: IP route table inconsistent.</Alert>
    </div>
  ),
};
