import type { Meta, StoryObj } from '@storybook/react-vite';
import { Card, CardValue, CardRow, CardDivider } from '../ui/Card';
import { StatusBadge } from '../ui/StatusBadge';
import { Router } from '../ui/Icons';
import { Skeleton } from '../ui/Skeleton';

/**
 * GatewayCard monitors network gateway (default router) reachability via ICMP ping.
 * Displays packet loss, latency statistics, and connection stability.
 *
 * This story demonstrates the card's visual states without context dependencies.
 */
const meta: Meta = {
  title: 'Cards/GatewayCard',
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <div className="w-80">
        <Story />
      </div>
    ),
  ],
};

export default meta;

export const Reachable: StoryObj = {
  render: () => (
    <Card
      title="Gateway"
      subtitle="Default Router"
      icon={<Router className="w-4 h-4" />}
      status="success"
    >
      <CardValue value="192.168.1.1" size="lg" />
      <div className="mt-3 space-y-1">
        <CardRow label="Status" value={<StatusBadge status="success" label="Reachable" />} />
        <CardRow label="Packet Loss" value="0%" valueStatus="success" />
        <CardDivider />
        <CardRow label="Latency (avg)" value="2.3ms" valueStatus="success" />
        <CardRow label="Latency (min)" value="1.1ms" />
        <CardRow label="Latency (max)" value="4.5ms" />
        <CardRow label="Last Ping" value="1.8ms" />
      </div>
    </Card>
  ),
};

export const HighLatency: StoryObj = {
  render: () => (
    <Card
      title="Gateway"
      subtitle="Default Router"
      icon={<Router className="w-4 h-4" />}
      status="warning"
    >
      <CardValue value="192.168.1.1" size="lg" />
      <div className="mt-3 space-y-1">
        <CardRow label="Status" value={<StatusBadge status="warning" label="Slow" />} />
        <CardRow label="Packet Loss" value="5%" valueStatus="warning" />
        <CardDivider />
        <CardRow label="Latency (avg)" value="85ms" valueStatus="warning" />
        <CardRow label="Latency (min)" value="45ms" />
        <CardRow label="Latency (max)" value="250ms" valueStatus="error" />
        <CardRow label="Last Ping" value="92ms" />
      </div>
    </Card>
  ),
};

export const Unreachable: StoryObj = {
  render: () => (
    <Card
      title="Gateway"
      subtitle="Default Router"
      icon={<Router className="w-4 h-4" />}
      status="error"
    >
      <CardValue value="192.168.1.1" size="lg" status="error" />
      <div className="mt-3 space-y-1">
        <CardRow label="Status" value={<StatusBadge status="error" label="Unreachable" />} />
        <CardRow label="Packet Loss" value="100%" valueStatus="error" />
        <CardDivider />
        <p className="caption text-status-danger">
          Gateway is not responding to ICMP ping requests.
          Check network connectivity.
        </p>
      </div>
    </Card>
  ),
};

export const DualStack: StoryObj = {
  render: () => (
    <Card
      title="Gateway"
      subtitle="IPv4 + IPv6"
      icon={<Router className="w-4 h-4" />}
      status="success"
    >
      <div className="space-y-3">
        <div>
          <p className="caption text-text-muted mb-1">IPv4</p>
          <CardValue value="192.168.1.1" size="md" />
          <CardRow label="Latency" value="2.3ms" valueStatus="success" />
          <CardRow label="Loss" value="0%" />
        </div>
        <CardDivider />
        <div>
          <p className="caption text-text-muted mb-1">IPv6</p>
          <CardValue value="fe80::1" size="md" />
          <CardRow label="Latency" value="1.8ms" valueStatus="success" />
          <CardRow label="Loss" value="0%" />
        </div>
      </div>
    </Card>
  ),
};

export const Loading: StoryObj = {
  render: () => (
    <Card
      title="Gateway"
      subtitle="Default Router"
      icon={<Router className="w-4 h-4" />}
      status="loading"
    >
      <Skeleton className="h-8 w-32 mb-3" />
      <div className="space-y-2 mt-4">
        <div className="flex justify-between">
          <Skeleton className="h-3 w-16" />
          <Skeleton className="h-3 w-20" />
        </div>
        <div className="flex justify-between">
          <Skeleton className="h-3 w-20" />
          <Skeleton className="h-3 w-12" />
        </div>
        <div className="flex justify-between">
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-3 w-16" />
        </div>
      </div>
    </Card>
  ),
};

export const NoGateway: StoryObj = {
  render: () => (
    <Card
      title="Gateway"
      subtitle="Default Router"
      icon={<Router className="w-4 h-4" />}
      status="unknown"
    >
      <CardValue value="Not Configured" size="md" />
      <p className="caption text-text-muted mt-2">
        No default gateway configured on this interface.
      </p>
    </Card>
  ),
};
