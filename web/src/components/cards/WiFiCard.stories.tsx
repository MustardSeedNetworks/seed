import type { Meta, StoryObj } from "@storybook/react-vite";
import { Card, CardValue, CardRow, CardDivider } from "../ui/Card";
import { Wifi, WifiOff, Signal } from "lucide-react";
import { Skeleton } from "../ui/Skeleton";
import { spacing } from "../../styles/theme";

/**
 * WiFiCard displays wireless network connection status and signal quality.
 * Shows SSID, signal strength, channel, and security information.
 *
 * This story demonstrates the card's visual states.
 */
const meta: Meta = {
  title: "Cards/WiFiCard",
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

export const Connected: StoryObj = {
  render: () => (
    <Card
      title="WiFi"
      subtitle="Wireless Network"
      icon={<Wifi className="w-4 h-4" />}
      status="success"
    >
      <CardValue value="HomeNetwork" size="lg" />
      <div className={`${spacing.margin.top.content} ${spacing.stack.xs}`}>
        <CardRow label="Signal" value="-45 dBm" status="success" />
        <CardRow label="Quality" value="Excellent" status="success" />
        <CardDivider />
        <CardRow label="Channel" value="6 (2.4 GHz)" />
        <CardRow label="Security" value="WPA3" />
        <CardRow label="Speed" value="144 Mbps" />
        <CardRow label="BSSID" value="AA:BB:CC:DD:EE:FF" />
      </div>
    </Card>
  ),
};

export const WeakSignal: StoryObj = {
  render: () => (
    <Card
      title="WiFi"
      subtitle="Wireless Network"
      icon={<Wifi className="w-4 h-4" />}
      status="warning"
    >
      <CardValue value="OfficeWiFi" size="lg" />
      <div className={`${spacing.margin.top.content} ${spacing.stack.xs}`}>
        <CardRow label="Signal" value="-75 dBm" status="warning" />
        <CardRow label="Quality" value="Fair" status="warning" />
        <CardDivider />
        <CardRow label="Channel" value="36 (5 GHz)" />
        <CardRow label="Security" value="WPA2" />
        <CardRow label="Speed" value="72 Mbps" />
        <p className={`caption text-status-warning ${spacing.margin.top.inline}`}>
          Weak signal. Move closer to the access point or check for interference.
        </p>
      </div>
    </Card>
  ),
};

export const VeryWeakSignal: StoryObj = {
  render: () => (
    <Card
      title="WiFi"
      subtitle="Wireless Network"
      icon={<Wifi className="w-4 h-4" />}
      status="error"
    >
      <CardValue value="GuestNetwork" size="lg" />
      <div className={`${spacing.margin.top.content} ${spacing.stack.xs}`}>
        <CardRow label="Signal" value="-85 dBm" status="error" />
        <CardRow label="Quality" value="Poor" status="error" />
        <CardDivider />
        <CardRow label="Channel" value="11 (2.4 GHz)" />
        <p className={`caption text-status-danger ${spacing.margin.top.inline}`}>
          Very weak signal. Connection may be unstable or drop frequently.
        </p>
      </div>
    </Card>
  ),
};

export const Disconnected: StoryObj = {
  render: () => (
    <Card
      title="WiFi"
      subtitle="Wireless Network"
      icon={<WifiOff className="w-4 h-4" />}
      status="error"
    >
      <CardValue value="Not Connected" size="lg" status="error" />
      <div className={`${spacing.margin.top.content} ${spacing.stack.xs}`}>
        <CardRow label="Status" value="Disconnected" status="error" />
        <CardDivider />
        <p className="caption text-text-muted">
          No wireless network connection. Check WiFi adapter and available networks.
        </p>
      </div>
    </Card>
  ),
};

export const Scanning: StoryObj = {
  render: () => (
    <Card
      title="WiFi"
      subtitle="Scanning Networks"
      icon={<Wifi className="w-4 h-4" />}
      status="loading"
    >
      <Skeleton className={`h-8 w-40 ${spacing.margin.bottom.content}`} />
      <div className={`${spacing.stack.sm} ${spacing.margin.top.content}`}>
        <div className="flex justify-between">
          <Skeleton className="h-3 w-16" />
          <Skeleton className="h-3 w-20" />
        </div>
        <div className="flex justify-between">
          <Skeleton className="h-3 w-20" />
          <Skeleton className="h-3 w-16" />
        </div>
        <div className="flex justify-between">
          <Skeleton className="h-3 w-12" />
          <Skeleton className="h-3 w-24" />
        </div>
      </div>
    </Card>
  ),
};

export const MultipleNetworks: StoryObj = {
  render: () => (
    <Card
      title="WiFi"
      subtitle="Available Networks"
      icon={<Signal className="w-4 h-4" />}
      status="success"
    >
      <CardValue value="5 Networks Found" size="md" />
      <div className={`${spacing.margin.top.content} ${spacing.stack.sm}`}>
        <div
          className={`flex justify-between items-center ${spacing.compact.py} border-b border-surface-border/50 /* spacing.compact.py for compact list item */`}
        >
          <span className="body-small">HomeNetwork</span>
          <span className="caption text-status-success">-45 dBm</span>
        </div>
        <div
          className={`flex justify-between items-center ${spacing.compact.py} border-b border-surface-border/50 /* spacing.compact.py for compact list item */`}
        >
          <span className="body-small">Neighbor_5G</span>
          <span className="caption text-status-warning">-65 dBm</span>
        </div>
        <div
          className={`flex justify-between items-center ${spacing.compact.py} border-b border-surface-border/50 /* spacing.compact.py for compact list item */`}
        >
          <span className="body-small">Guest</span>
          <span className="caption text-status-warning">-70 dBm</span>
        </div>
        <div
          className={`flex justify-between items-center ${spacing.compact.py} border-b border-surface-border/50 /* spacing.compact.py for compact list item */`}
        >
          <span className="body-small text-text-muted">Hidden Network</span>
          <span className="caption text-status-danger">-82 dBm</span>
        </div>
      </div>
    </Card>
  ),
};

export const NoAdapter: StoryObj = {
  render: () => (
    <Card
      title="WiFi"
      subtitle="Wireless Network"
      icon={<WifiOff className="w-4 h-4" />}
      status="unknown"
    >
      <CardValue value="Not Available" size="md" />
      <p className={`caption text-text-muted ${spacing.margin.top.inline}`}>
        No wireless adapter detected on this system.
      </p>
    </Card>
  ),
};
