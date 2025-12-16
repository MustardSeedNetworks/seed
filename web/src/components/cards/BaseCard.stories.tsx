import type { Meta, StoryObj } from "@storybook/react-vite";
import { BaseCard, SimpleBaseCard } from "./BaseCard";
import { CardValue, CardRow } from "../ui/Card";
import { Wifi, Server, Globe } from "lucide-react";

interface SampleData {
  value: string;
  status: "healthy" | "warning" | "error";
  details: { label: string; value: string }[];
}

const sampleData: SampleData = {
  value: "192.168.1.1",
  status: "healthy",
  details: [
    { label: "Latency", value: "5ms" },
    { label: "Uptime", value: "99.9%" },
    { label: "Protocol", value: "IPv4" },
  ],
};

/**
 * BaseCard is the foundation for all dashboard cards. It handles:
 * - Loading states with skeletons
 * - Error states with error messages
 * - Empty/no-data states
 * - Status derivation from data
 */
const meta: Meta<typeof BaseCard<SampleData>> = {
  title: "Cards/BaseCard",
  component: BaseCard,
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
type Story = StoryObj<typeof BaseCard<SampleData>>;

export const Default: Story = {
  args: {
    title: "Network Status",
    subtitle: "Primary Interface",
    icon: <Globe className="w-4 h-4" />,
    data: sampleData,
    getStatus: (data) =>
      data.status === "healthy" ? "success" : data.status === "warning" ? "warning" : "error",
    children: (data) => (
      <>
        <CardValue value={data.value} size="lg" />
        {data.details.map((d) => (
          <CardRow key={d.label} label={d.label} value={d.value} />
        ))}
      </>
    ),
  },
};

export const Loading: Story = {
  args: {
    ...Default.args,
    data: null,
    loading: true,
  },
};

export const Error: Story = {
  args: {
    ...Default.args,
    data: null,
    error: "Failed to fetch network status. Please check your connection.",
  },
};

export const NoData: Story = {
  args: {
    ...Default.args,
    data: null,
    emptyMessage: "No network data available",
  },
};

export const WarningStatus: Story = {
  args: {
    ...Default.args,
    data: { ...sampleData, status: "warning" as const, value: "Degraded" },
  },
};

export const ErrorStatus: Story = {
  args: {
    ...Default.args,
    data: { ...sampleData, status: "error" as const, value: "Offline" },
  },
};

export const WithClick: Story = {
  args: {
    ...Default.args,
    onClick: () => alert("Card clicked!"),
  },
  parameters: {
    docs: {
      description: {
        story: "Cards can be made clickable for drill-down views.",
      },
    },
  },
};

// SimpleBaseCard stories
export const SimpleCard: StoryObj<typeof SimpleBaseCard> = {
  render: () => (
    <SimpleBaseCard
      title="Simple Card"
      subtitle="No data derivation"
      icon={<Server className="w-4 h-4" />}
      status="success"
    >
      <CardValue value="Active" size="lg" status="success" />
      <CardRow label="Connections" value="42" />
      <CardRow label="Memory" value="2.1 GB" />
    </SimpleBaseCard>
  ),
};

export const SimpleCardLoading: StoryObj<typeof SimpleBaseCard> = {
  render: () => (
    <SimpleBaseCard
      title="Simple Card"
      icon={<Wifi className="w-4 h-4" />}
      status="loading"
      loading={true}
    >
      {/* Children not shown during loading */}
      <CardValue value="Active" />
    </SimpleBaseCard>
  ),
};

export const CardGrid: Story = {
  render: () => (
    <div className="grid grid-cols-2 gap-4 w-[600px]">
      <BaseCard
        title="Gateway"
        icon={<Server className="w-4 h-4" />}
        data={sampleData}
        getStatus={() => "success"}
      >
        {(data) => <CardValue value={data.value} />}
      </BaseCard>
      <BaseCard
        title="DNS"
        icon={<Globe className="w-4 h-4" />}
        data={{ ...sampleData, status: "warning" } as SampleData}
        getStatus={(d) => (d.status === "healthy" ? "success" : d.status)}
      >
        {() => <CardValue value="8.8.8.8" />}
      </BaseCard>
      <BaseCard
        title="WiFi"
        icon={<Wifi className="w-4 h-4" />}
        data={null}
        loading={true}
        getStatus={() => "success"}
      >
        {() => null}
      </BaseCard>
      <BaseCard title="Link" data={null} error="Connection failed" getStatus={() => "error"}>
        {() => null}
      </BaseCard>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: "Multiple cards in a grid showing different states.",
      },
    },
  },
};
