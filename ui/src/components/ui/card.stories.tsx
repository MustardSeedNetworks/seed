import type { Meta, StoryObj } from '@storybook/react-vite';
import { Cable, Server, Wifi } from 'lucide-react';
import { cn, spacing } from '../../styles/theme';
import { Card, CardDivider, CardRow, CardValue } from './Card';

const meta: Meta<typeof Card> = {
  title: 'UI/Card',
  component: Card,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Base card container used throughout the application for displaying information with status indicators.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    status: {
      control: 'select',
      options: ['success', 'warning', 'error', 'unknown', 'loading'],
      description: 'Status indicator color',
    },
    title: {
      control: 'text',
      description: 'Card title (required)',
    },
    subtitle: {
      control: 'text',
      description: 'Optional subtitle or description',
    },
    onClick: {
      action: 'clicked',
      description: 'Callback when card is clicked',
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default card with success status
export const Success: Story = {
  args: {
    title: 'Link Status',
    subtitle: 'eth0',
    status: 'success',
    icon: <Cable class="w-5 h-5" />,
    children: (
      <div class="stack-sm">
        <CardRow label="Speed" value="1000 Mbps" />
        <CardDivider />
        <CardRow label="Duplex" value="Full" />
        <CardRow label="MTU" value="1500" />
      </div>
    ),
  },
};

// Warning status
export const Warning: Story = {
  args: {
    title: 'WiFi Signal',
    subtitle: 'wlan0',
    status: 'warning',
    icon: <Wifi class="w-5 h-5" />,
    children: (
      <div class="stack-sm">
        <CardValue label="Signal Strength" value="-72" unit="dBm" status="warning" />
        <CardRow label="SSID" value="Office-5G" />
        <CardRow label="Channel" value="36" />
      </div>
    ),
  },
};

// Error status
export const ErrorStatus: Story = {
  args: {
    title: 'Gateway',
    subtitle: '192.168.1.1',
    status: 'error',
    icon: <Server class="w-5 h-5" />,
    children: (
      <div class="stack-sm">
        <CardValue label="Status" value="Unreachable" status="error" />
        <CardRow label="Last Seen" value="5 min ago" />
        <CardRow label="Packet Loss" value="100%" status="error" />
      </div>
    ),
  },
};

// Loading state
export const Loading: Story = {
  args: {
    title: 'Speed Test',
    subtitle: 'Running...',
    status: 'loading',
    children: (
      <div class="stack-sm">
        <CardValue value="Testing download speed..." size="sm" />
      </div>
    ),
  },
};

// Unknown/No Data state
export const Unknown: Story = {
  args: {
    title: 'Cable Test',
    subtitle: 'No data',
    status: 'unknown',
    icon: <Cable class="w-5 h-5" />,
    children: (
      <div class="stack-sm">
        <CardValue value="No data available" size="sm" />
      </div>
    ),
  },
};

// Interactive card with click handler
export const Interactive: Story = {
  args: {
    title: 'Network Discovery',
    subtitle: 'Click to view details',
    status: 'success',
    icon: <Server class="w-5 h-5" />,
    onClick: () => alert('Card clicked!'),
    children: (
      <div class="stack-sm">
        <CardRow label="Devices Found" value="12" />
        <CardRow label="Last Scan" value="2 min ago" />
      </div>
    ),
  },
};

// Card with header action
export const WithHeaderAction: Story = {
  args: {
    title: 'DNS Status',
    subtitle: 'Primary',
    status: 'success',
    headerAction: (
      <button
        type="button"
        class={cn(
          spacing.chip.sm,
          'caption bg-brand-primary text-text-inverse rounded-md hover:bg-brand-primary/90',
        )}
      >
        Refresh
      </button>
    ),
    children: (
      <div class="stack-sm">
        <CardRow label="Server" value="8.8.8.8" />
        <CardRow label="Latency" value="12ms" status="success" />
      </div>
    ),
  },
};
