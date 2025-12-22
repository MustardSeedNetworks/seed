import type { Meta, StoryObj } from "@storybook/react-vite";
import { CardValue, CardRow, CardDivider, Card } from "../ui/Card";
import { Cable } from "../ui/Icons";
import { Skeleton } from "../ui/Skeleton";
import { spacing, cn } from "../../styles/theme";

/**
 * LinkCard displays physical link layer (L2) and network layer (L3) status.
 *
 * This story demonstrates the card's visual states without the actual
 * LinkCard component to avoid context dependencies.
 */
const meta: Meta = {
  title: "Cards/LinkCard",
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

// Mock data for stories
const linkDataOnline = {
  linkUp: true,
  carrier: true,
  hasIP: true,
  speed: "1000Mb/s",
  duplex: "Full",
  mtu: 1500,
  autoNeg: true,
  flapCount24h: 0,
  uptimeMs: 86400000, // 24 hours
};

const linkDataNoIP = {
  linkUp: true,
  carrier: true,
  hasIP: false,
  speed: "100Mb/s",
  duplex: "Full",
  mtu: 1500,
};

export const Online: StoryObj = {
  render: () => (
    <Card
      title="Link"
      subtitle="en0"
      icon={<Cable className="w-4 h-4" />}
      status="success"
    >
      <CardValue value="Connected" size="lg" status="success" />
      <div className={cn(spacing.margin.top.content, spacing.stack.xs)}>
        <CardRow label="Speed" value={linkDataOnline.speed} />
        <CardRow label="Duplex" value={linkDataOnline.duplex} />
        <CardRow label="MTU" value={String(linkDataOnline.mtu)} />
        <CardDivider />
        <CardRow label="Auto-Neg" value="Enabled" />
        <CardRow label="Link Flaps (24h)" value="0" />
        <CardRow label="Uptime" value="24h 0m" />
      </div>
    </Card>
  ),
};

export const NoIPAddress: StoryObj = {
  render: () => (
    <Card
      title="Link"
      subtitle="en0"
      icon={<Cable className="w-4 h-4" />}
      status="warning"
    >
      <CardValue value="No IP" size="lg" status="warning" />
      <div className={cn(spacing.margin.top.content, spacing.stack.xs)}>
        <CardRow label="Speed" value={linkDataNoIP.speed} />
        <CardRow label="Duplex" value={linkDataNoIP.duplex} />
        <CardRow label="Carrier" value="Detected" />
        <CardDivider />
        <p className="caption text-status-warning">
          Physical link present but no IP address assigned. Check DHCP or static
          IP configuration.
        </p>
      </div>
    </Card>
  ),
};

export const Disconnected: StoryObj = {
  render: () => (
    <Card
      title="Link"
      subtitle="en0"
      icon={<Cable className="w-4 h-4" />}
      status="error"
    >
      <CardValue value="No Carrier" size="lg" status="error" />
      <div className={cn(spacing.margin.top.content, spacing.stack.xs)}>
        <CardRow label="Speed" value="—" />
        <CardRow label="Duplex" value="—" />
        <CardRow label="Carrier" value="Not Detected" status="error" />
        <CardDivider />
        <p className="caption text-status-danger">
          No physical connection detected. Check cable or wireless adapter.
        </p>
      </div>
    </Card>
  ),
};

export const Loading: StoryObj = {
  render: () => (
    <Card
      title="Link"
      subtitle="en0"
      icon={<Cable className="w-4 h-4" />}
      status="loading"
    >
      <Skeleton className={cn("h-8 w-32", spacing.margin.bottom.content)} />
      <div className={cn(spacing.stack.sm, spacing.margin.top.content)}>
        <div className="flex justify-between">
          <Skeleton className="h-3 w-16" />
          <Skeleton className="h-3 w-20" />
        </div>
        <div className="flex justify-between">
          <Skeleton className="h-3 w-12" />
          <Skeleton className="h-3 w-16" />
        </div>
        <div className="flex justify-between">
          <Skeleton className="h-3 w-20" />
          <Skeleton className="h-3 w-12" />
        </div>
      </div>
    </Card>
  ),
};

export const WithLinkFlaps: StoryObj = {
  render: () => (
    <Card
      title="Link"
      subtitle="en0"
      icon={<Cable className="w-4 h-4" />}
      status="warning"
    >
      <CardValue value="Unstable" size="lg" status="warning" />
      <div className={cn(spacing.margin.top.content, spacing.stack.xs)}>
        <CardRow label="Speed" value="1000Mb/s" />
        <CardRow label="Duplex" value="Full" />
        <CardRow label="Link Flaps (24h)" value="12" status="warning" />
        <CardDivider />
        <p className="caption text-status-warning">
          High number of link state changes detected. Check cable integrity.
        </p>
      </div>
    </Card>
  ),
};
