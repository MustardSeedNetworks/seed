import type { Meta, StoryObj } from "@storybook/react-vite";
import { Card, CardValue, CardRow, CardDivider } from "../ui/Card";
import { Router } from "../ui/Icons";
import { Skeleton } from "../ui/Skeleton";
import { spacing } from "../../styles/theme";

/**
 * GatewayCard monitors network gateway (default router) reachability via ICMP ping.
 * Displays packet loss, latency statistics, and connection stability.
 *
 * This story demonstrates the card's visual states without context dependencies.
 */
const meta: Meta = {
  title: "Cards/GatewayCard",
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
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
      <div className={`${spacing.margin.top.content} ${spacing.stack.xs}`}>
        <CardRow label="Status" value="Reachable" status="success" />
        <CardRow label="Packet Loss" value="0%" status="success" />
        <CardDivider />
        <CardRow label="Latency (avg)" value="2.3ms" status="success" />
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
      <div className={`${spacing.margin.top.content} ${spacing.stack.xs}`}>
        <CardRow label="Status" value="Slow" status="warning" />
        <CardRow label="Packet Loss" value="5%" status="warning" />
        <CardDivider />
        <CardRow label="Latency (avg)" value="85ms" status="warning" />
        <CardRow label="Latency (min)" value="45ms" />
        <CardRow label="Latency (max)" value="250ms" status="error" />
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
      <div className={`${spacing.margin.top.content} ${spacing.stack.xs}`}>
        <CardRow label="Status" value="Unreachable" status="error" />
        <CardRow label="Packet Loss" value="100%" status="error" />
        <CardDivider />
        <p className="caption text-status-danger">
          Gateway is not responding to ICMP ping requests. Check network connectivity.
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
      <div className={spacing.stack.default}>
        <div>
          <p className={`caption text-text-muted ${spacing.margin.bottom.inline}`}>IPv4</p>
          <CardValue value="192.168.1.1" size="md" />
          <CardRow label="Latency" value="2.3ms" status="success" />
          <CardRow label="Loss" value="0%" />
        </div>
        <CardDivider />
        <div>
          <p className={`caption text-text-muted ${spacing.margin.bottom.inline}`}>IPv6</p>
          <CardValue value="fe80::1" size="md" />
          <CardRow label="Latency" value="1.8ms" status="success" />
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
      <Skeleton className={`h-8 w-32 ${spacing.margin.bottom.content}`} />
      <div className={`${spacing.stack.sm} ${spacing.margin.top.content}`}>
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
      <p className={`caption text-text-muted ${spacing.margin.top.inline}`}>
        No default gateway configured on this interface.
      </p>
    </Card>
  ),
};
