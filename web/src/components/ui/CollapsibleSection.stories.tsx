import type { Meta, StoryObj } from "@storybook/react-vite";
import { CollapsibleSection } from "./CollapsibleSection";
import { Settings } from "lucide-react";

const meta: Meta<typeof CollapsibleSection> = {
  title: "UI/CollapsibleSection",
  component: CollapsibleSection,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Collapsible/accordion section for organizing content within cards and modals.",
      },
    },
  },
  tags: ["autodocs"],
  argTypes: {
    title: {
      control: "text",
      description: "Section title",
    },
    defaultOpen: {
      control: "boolean",
      description: "Whether section is open by default",
    },
    count: {
      control: "number",
      description: "Item count to display in header",
    },
    status: {
      control: "select",
      options: ["success", "warning", "error", "unknown", "loading", undefined],
      description: "Status indicator",
    },
    variant: {
      control: "radio",
      options: ["default", "compact"],
      description: "Visual variant",
    },
  },
  decorators: [
    (Story) => (
      <div className="w-[400px]">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default variant (with border)
export const Default: Story = {
  args: {
    title: "Advanced Options",
    defaultOpen: false,
    children: (
      <div className="stack-sm">
        <p className="body-small text-text-muted">
          Configure advanced settings for network discovery.
        </p>
        <label className="flex items-center gap-2">
          <input type="checkbox" className="w-4 h-4" />
          <span className="body-small">Enable deep scanning</span>
        </label>
        <label className="flex items-center gap-2">
          <input type="checkbox" className="w-4 h-4" />
          <span className="body-small">Include SNMP queries</span>
        </label>
      </div>
    ),
  },
};

// Default open
export const DefaultOpen: Story = {
  args: {
    title: "Settings",
    defaultOpen: true,
    children: (
      <div className="stack-sm">
        <p className="body-small text-text-muted">
          These settings are shown by default.
        </p>
      </div>
    ),
  },
};

// Compact variant (for inside cards)
export const Compact: Story = {
  args: {
    title: "Server Results",
    variant: "compact",
    count: 3,
    defaultOpen: true,
    children: (
      <div className="stack-xs">
        <div className="flex justify-between body-small">
          <span>8.8.8.8</span>
          <span className="text-status-success">12ms</span>
        </div>
        <div className="flex justify-between body-small">
          <span>1.1.1.1</span>
          <span className="text-status-success">8ms</span>
        </div>
        <div className="flex justify-between body-small">
          <span>192.168.1.1</span>
          <span className="text-status-warning">45ms</span>
        </div>
      </div>
    ),
  },
};

// With status indicator
export const WithStatus: Story = {
  args: {
    title: "DNS Servers",
    status: "success",
    count: 2,
    defaultOpen: true,
    children: (
      <div className="stack-sm">
        <div className="flex justify-between body-small">
          <span>Primary: 8.8.8.8</span>
          <span className="text-status-success">Online</span>
        </div>
        <div className="flex justify-between body-small">
          <span>Secondary: 8.8.4.4</span>
          <span className="text-status-success">Online</span>
        </div>
      </div>
    ),
  },
};

// With warning status
export const WithWarningStatus: Story = {
  args: {
    title: "Network Interfaces",
    status: "warning",
    count: 3,
    defaultOpen: true,
    children: (
      <div className="stack-sm">
        <div className="flex justify-between body-small">
          <span>eth0</span>
          <span className="text-status-success">Connected</span>
        </div>
        <div className="flex justify-between body-small">
          <span>wlan0</span>
          <span className="text-status-warning">Weak Signal</span>
        </div>
        <div className="flex justify-between body-small">
          <span>eth1</span>
          <span className="text-text-muted">Disconnected</span>
        </div>
      </div>
    ),
  },
};

// With custom title (React node)
export const CustomTitle: Story = {
  args: {
    title: (
      <div className="flex items-center gap-2">
        <Settings className="w-4 h-4" />
        <span>Configuration</span>
      </div>
    ),
    defaultOpen: true,
    children: (
      <div className="stack-sm">
        <p className="body-small text-text-muted">
          Custom title with icon support.
        </p>
      </div>
    ),
  },
};

// Multiple sections example
export const MultipleSections: Story = {
  render: () => (
    <div className="stack">
      <CollapsibleSection title="Network Settings" status="success" defaultOpen>
        <div className="stack-sm">
          <p className="body-small">Interface: eth0</p>
          <p className="body-small">IP: 192.168.1.100</p>
        </div>
      </CollapsibleSection>
      <CollapsibleSection title="DNS Configuration" status="success">
        <div className="stack-sm">
          <p className="body-small">Primary: 8.8.8.8</p>
          <p className="body-small">Secondary: 8.8.4.4</p>
        </div>
      </CollapsibleSection>
      <CollapsibleSection title="WiFi Settings" status="warning">
        <div className="stack-sm">
          <p className="body-small">SSID: Office-5G</p>
          <p className="body-small">Signal: -72 dBm</p>
        </div>
      </CollapsibleSection>
    </div>
  ),
};
