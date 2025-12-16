import type { Meta, StoryObj } from "@storybook/react-vite";
import { DataTable, Column } from "./DataTable";
import { spacing } from "../../styles/theme";

interface Device {
  ip: string;
  hostname: string;
  mac: string;
  status: "online" | "offline" | "unknown";
  lastSeen: string;
}

const sampleDevices: Device[] = [
  {
    ip: "192.168.1.1",
    hostname: "router.local",
    mac: "AA:BB:CC:DD:EE:FF",
    status: "online",
    lastSeen: "2 min ago",
  },
  {
    ip: "192.168.1.10",
    hostname: "desktop-pc",
    mac: "11:22:33:44:55:66",
    status: "online",
    lastSeen: "1 min ago",
  },
  {
    ip: "192.168.1.20",
    hostname: "laptop",
    mac: "77:88:99:AA:BB:CC",
    status: "offline",
    lastSeen: "2 hours ago",
  },
  {
    ip: "192.168.1.30",
    hostname: "printer.local",
    mac: "DD:EE:FF:00:11:22",
    status: "online",
    lastSeen: "5 min ago",
  },
  {
    ip: "192.168.1.40",
    hostname: "nas-server",
    mac: "33:44:55:66:77:88",
    status: "unknown",
    lastSeen: "Never",
  },
  {
    ip: "192.168.1.50",
    hostname: "smart-tv",
    mac: "99:AA:BB:CC:DD:EE",
    status: "online",
    lastSeen: "10 min ago",
  },
];

const deviceColumns: Column<Device>[] = [
  { key: "ip", header: "IP Address", accessor: (d) => d.ip, sortable: true },
  { key: "hostname", header: "Hostname", accessor: (d) => d.hostname, sortable: true },
  { key: "mac", header: "MAC Address", accessor: (d) => d.mac, hiddenOnMobile: true },
  {
    key: "status",
    header: "Status",
    accessor: (d) => d.status,
    sortable: true,
    render: (d) => (
      <span
        className={`px-2 py-0.5 rounded-full text-xs ${
          /* px-2 py-0.5 for compact status badge */
          d.status === "online"
            ? "bg-status-success/20 text-status-success"
            : d.status === "offline"
              ? "bg-status-danger/20 text-status-danger"
              : "bg-status-warning/20 text-status-warning"
        }`}
      >
        {d.status}
      </span>
    ),
  },
  { key: "lastSeen", header: "Last Seen", accessor: (d) => d.lastSeen, hiddenOnMobile: true },
];

/**
 * DataTable provides a feature-rich table with sorting, searching, filtering,
 * and customizable rendering. Used for displaying device lists, scan results, etc.
 */
const meta: Meta<typeof DataTable<Device>> = {
  title: "UI/DataTable",
  component: DataTable,
  parameters: {
    layout: "padded",
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof DataTable<Device>>;

export const Default: Story = {
  args: {
    data: sampleDevices,
    columns: deviceColumns,
    keyExtractor: (d) => d.ip,
    searchPlaceholder: "Search devices...",
    searchKeys: ["hostname", "ip", "mac"],
  },
};

export const WithSorting: Story = {
  args: {
    data: sampleDevices,
    columns: deviceColumns,
    keyExtractor: (d) => d.ip,
  },
  parameters: {
    docs: {
      description: {
        story: "Click on column headers with sort icons to sort the data.",
      },
    },
  },
};

export const WithFilters: Story = {
  args: {
    data: sampleDevices,
    columns: deviceColumns,
    keyExtractor: (d) => d.ip,
    searchKeys: ["hostname", "ip"],
    filterOptions: [
      {
        key: "status",
        label: "Filter by status",
        options: [
          { value: "online", label: "Online" },
          { value: "offline", label: "Offline" },
          { value: "unknown", label: "Unknown" },
        ],
      },
    ],
  },
  parameters: {
    docs: {
      description: {
        story: "Click the filter icon to show dropdown filters.",
      },
    },
  },
};

export const WithRowClick: Story = {
  args: {
    data: sampleDevices,
    columns: deviceColumns,
    keyExtractor: (d) => d.ip,
    onRowClick: (device) => alert(`Clicked: ${device.hostname}`),
  },
  parameters: {
    docs: {
      description: {
        story: "Rows are clickable and highlight on hover.",
      },
    },
  },
};

export const WithActions: Story = {
  args: {
    data: sampleDevices,
    columns: deviceColumns,
    keyExtractor: (d) => d.ip,
    actions: (device) => (
      <button
        onClick={(e) => {
          e.stopPropagation();
          alert(`Action on ${device.hostname}`);
        }}
        className={`${spacing.chip.sm} text-xs bg-brand-primary/20 text-brand-primary rounded hover:bg-brand-primary/30`}
      >
        Details
      </button>
    ),
  },
};

export const Empty: Story = {
  args: {
    data: [],
    columns: deviceColumns,
    keyExtractor: (d) => d.ip,
    emptyMessage: "No devices found. Try running a network scan.",
  },
};

export const CustomMaxHeight: Story = {
  args: {
    data: [...sampleDevices, ...sampleDevices, ...sampleDevices].map((d, i) => ({
      ...d,
      ip: `${d.ip}-${i}`,
    })),
    columns: deviceColumns,
    keyExtractor: (d: Device) => d.ip,
    maxHeight: "max-h-48",
  },
  parameters: {
    docs: {
      description: {
        story: "Table with restricted height and scrolling for many rows.",
      },
    },
  },
};
