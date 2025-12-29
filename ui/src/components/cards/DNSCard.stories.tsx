import type { Meta, StoryObj } from "@storybook/react-vite";
import { Globe } from "lucide-react";
import { cn, spacing } from "../../styles/theme";
import { Card, CardDivider, CardRow, CardValue } from "../ui/Card";
import { Skeleton } from "../ui/Skeleton";

/**
 * DNSCard displays DNS resolver status and resolution times.
 * Tests DNS connectivity to configured nameservers.
 *
 * This story demonstrates the card's visual states without context dependencies.
 */
const meta: Meta = {
  title: "Cards/DNSCard",
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

export const Healthy: StoryObj = {
  render: () => (
    <Card
      title="DNS"
      subtitle="Name Resolution"
      icon={<Globe className="w-4 h-4" />}
      status="success"
    >
      <CardValue value="8.8.8.8" size="lg" />
      <div className={cn(spacing.margin.top.content, spacing.stack.xs)}>
        <CardRow label="Status" value="Resolving" status="success" />
        <CardRow label="Resolution Time" value="12ms" status="success" />
        <CardDivider />
        <CardRow label="Primary DNS" value="8.8.8.8" />
        <CardRow label="Secondary DNS" value="8.8.4.4" />
        <CardRow label="Test Domain" value="google.com" />
      </div>
    </Card>
  ),
};

export const SlowResolution: StoryObj = {
  render: () => (
    <Card
      title="DNS"
      subtitle="Name Resolution"
      icon={<Globe className="w-4 h-4" />}
      status="warning"
    >
      <CardValue value="192.168.1.1" size="lg" />
      <div className={cn(spacing.margin.top.content, spacing.stack.xs)}>
        <CardRow label="Status" value="Slow" status="warning" />
        <CardRow label="Resolution Time" value="850ms" status="warning" />
        <CardDivider />
        <CardRow label="Primary DNS" value="192.168.1.1" />
        <CardRow label="Test Domain" value="google.com" />
        <p className={cn("caption text-status-warning", spacing.margin.top.inline)}>
          DNS resolution is slower than expected. Consider using a faster DNS server.
        </p>
      </div>
    </Card>
  ),
};

export const Failed: StoryObj = {
  render: () => (
    <Card
      title="DNS"
      subtitle="Name Resolution"
      icon={<Globe className="w-4 h-4" />}
      status="error"
    >
      <CardValue value="Failed" size="lg" status="error" />
      <div className={cn(spacing.margin.top.content, spacing.stack.xs)}>
        <CardRow label="Status" value="Unreachable" status="error" />
        <CardRow label="Primary DNS" value="8.8.8.8" />
        <CardDivider />
        <p className="caption text-status-danger">
          DNS server is not responding. Check network connectivity or try a different DNS server.
        </p>
      </div>
    </Card>
  ),
};

export const MultipleDns: StoryObj = {
  render: () => (
    <Card
      title="DNS"
      subtitle="Multiple Resolvers"
      icon={<Globe className="w-4 h-4" />}
      status="success"
    >
      <CardValue value="All Healthy" size="lg" status="success" />
      <div className={cn(spacing.margin.top.content, spacing.stack.xs)}>
        <CardRow label="8.8.8.8" value="12ms" status="success" />
        <CardRow label="8.8.4.4" value="15ms" status="success" />
        <CardRow label="1.1.1.1" value="8ms" status="success" />
        <CardRow label="1.0.0.1" value="10ms" status="success" />
      </div>
    </Card>
  ),
};

export const Loading: StoryObj = {
  render: () => (
    <Card
      title="DNS"
      subtitle="Name Resolution"
      icon={<Globe className="w-4 h-4" />}
      status="loading"
    >
      <Skeleton className={cn("h-8 w-32", spacing.margin.bottom.content)} />
      <div className={cn(spacing.stack.sm, spacing.margin.top.content)}>
        <div className="flex justify-between">
          <Skeleton className="h-3 w-16" />
          <Skeleton className="h-3 w-20" />
        </div>
        <div className="flex justify-between">
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-3 w-12" />
        </div>
      </div>
    </Card>
  ),
};

export const NoDns: StoryObj = {
  render: () => (
    <Card
      title="DNS"
      subtitle="Name Resolution"
      icon={<Globe className="w-4 h-4" />}
      status="unknown"
    >
      <CardValue value="Not Configured" size="md" />
      <p className={cn("caption text-text-muted", spacing.margin.top.inline)}>
        No DNS servers configured. Network may not resolve domain names.
      </p>
    </Card>
  ),
};
