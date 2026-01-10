/**
 * Sparkline Component Stories
 *
 * Showcases the Sparkline, SparklineWithLabel, and HealthScoreBadge components
 * for displaying health check trends and scores.
 */

import type { Meta, StoryObj } from "@storybook/react";
import { HealthScoreBadge, Sparkline, SparklineWithLabel } from "./sparkline";

// Generate sample data with variations
const generateAvailabilityData = (count: number, baseValue = 99): number[] =>
  Array.from({ length: count }, () => baseValue + (Math.random() - 0.3) * 2);

const generateLatencyData = (count: number, baseValue = 50): number[] =>
  Array.from({ length: count }, () => baseValue + (Math.random() - 0.5) * 30);

// Sample data sets
const healthyAvailability = generateAvailabilityData(24, 99.5);
const degradedAvailability = generateAvailabilityData(24, 95);
const criticalAvailability = [98, 97, 95, 92, 88, 85, 82, 79, 75, 78, 80, 82];

const goodLatency = generateLatencyData(24, 40);
const warningLatency = generateLatencyData(24, 80);
const criticalLatency = generateLatencyData(24, 150);

// Sparkline Meta
const meta: Meta<typeof Sparkline> = {
  title: "UI/Sparkline",
  component: Sparkline,
  tags: ["autodocs"],
  argTypes: {
    type: {
      control: "select",
      options: ["availability", "latency", "score"],
    },
    size: {
      control: "select",
      options: ["sm", "md", "lg"],
    },
    showArea: {
      control: "boolean",
    },
    threshold: {
      control: "number",
    },
  },
};

export default meta;
type Story = StoryObj<typeof Sparkline>;

// Sparkline Stories
export const AvailabilityHealthy: Story = {
  args: {
    data: healthyAvailability,
    type: "availability",
    size: "md",
    showArea: true,
  },
};

export const AvailabilityDegraded: Story = {
  args: {
    data: degradedAvailability,
    type: "availability",
    size: "md",
    showArea: true,
  },
};

export const AvailabilityCritical: Story = {
  args: {
    data: criticalAvailability,
    type: "availability",
    size: "md",
    showArea: true,
  },
};

export const LatencyGood: Story = {
  args: {
    data: goodLatency,
    type: "latency",
    size: "md",
    threshold: 100,
    showArea: true,
  },
};

export const LatencyWarning: Story = {
  args: {
    data: warningLatency,
    type: "latency",
    size: "md",
    threshold: 100,
    showArea: true,
  },
};

export const LatencyCritical: Story = {
  args: {
    data: criticalLatency,
    type: "latency",
    size: "md",
    threshold: 100,
    showArea: true,
  },
};

export const SizeSmall: Story = {
  args: {
    data: healthyAvailability,
    type: "availability",
    size: "sm",
    showArea: true,
  },
};

export const SizeLarge: Story = {
  args: {
    data: healthyAvailability,
    type: "availability",
    size: "lg",
    showArea: true,
  },
};

export const NoAreaFill: Story = {
  args: {
    data: healthyAvailability,
    type: "availability",
    size: "md",
    showArea: false,
  },
};

export const EmptyData: Story = {
  args: {
    data: [],
    type: "availability",
    size: "md",
  },
};

export const SingleDataPoint: Story = {
  args: {
    data: [99.5],
    type: "availability",
    size: "md",
  },
};

// SparklineWithLabel Stories
export const WithLabelAvailability: StoryObj<typeof SparklineWithLabel> = {
  render: () => (
    <SparklineWithLabel
      labelText="Uptime"
      data={healthyAvailability}
      type="availability"
      size="md"
      showValue
    />
  ),
};

export const WithLabelLatency: StoryObj<typeof SparklineWithLabel> = {
  render: () => (
    <SparklineWithLabel
      labelText="P95 Latency"
      data={goodLatency}
      type="latency"
      size="md"
      threshold={100}
      showValue
    />
  ),
};

// HealthScoreBadge Stories
export const BadgeHealthy: StoryObj<typeof HealthScoreBadge> = {
  render: () => <HealthScoreBadge score={92} size="md" />,
};

export const BadgeDegraded: StoryObj<typeof HealthScoreBadge> = {
  render: () => <HealthScoreBadge score={65} size="md" />,
};

export const BadgeCritical: StoryObj<typeof HealthScoreBadge> = {
  render: () => <HealthScoreBadge score={35} size="md" />,
};

export const BadgeSizes: StoryObj<typeof HealthScoreBadge> = {
  render: () => (
    <div className="flex items-center gap-4">
      <HealthScoreBadge score={92} size="sm" />
      <HealthScoreBadge score={92} size="md" />
      <HealthScoreBadge score={92} size="lg" />
    </div>
  ),
};

export const BadgeValueOnly: StoryObj<typeof HealthScoreBadge> = {
  render: () => (
    <div className="flex items-center gap-4">
      <HealthScoreBadge score={92} size="md" showValue />
      <HealthScoreBadge score={65} size="md" showValue />
      <HealthScoreBadge score={35} size="md" showValue />
    </div>
  ),
};

// Combined example showing all components together
export const CombinedExample: StoryObj<typeof Sparkline> = {
  render: () => (
    <div className="space-y-6 p-4 bg-bg-primary rounded-lg">
      <h3 className="text-text-primary font-semibold">Endpoint Health Overview</h3>

      <div className="space-y-4">
        {/* API Gateway */}
        <div className="flex items-center justify-between p-3 bg-bg-secondary rounded-md">
          <div className="flex items-center gap-3">
            <span className="text-text-primary font-medium">API Gateway</span>
            <HealthScoreBadge score={94} size="sm" />
          </div>
          <div className="flex items-center gap-4">
            <SparklineWithLabel
              labelText="Availability"
              data={healthyAvailability}
              type="availability"
              size="sm"
              showValue
            />
            <SparklineWithLabel
              labelText="Latency"
              data={goodLatency}
              type="latency"
              size="sm"
              threshold={100}
              showValue
            />
          </div>
        </div>

        {/* Database */}
        <div className="flex items-center justify-between p-3 bg-bg-secondary rounded-md">
          <div className="flex items-center gap-3">
            <span className="text-text-primary font-medium">Database</span>
            <HealthScoreBadge score={72} size="sm" />
          </div>
          <div className="flex items-center gap-4">
            <SparklineWithLabel
              labelText="Availability"
              data={degradedAvailability}
              type="availability"
              size="sm"
              showValue
            />
            <SparklineWithLabel
              labelText="Latency"
              data={warningLatency}
              type="latency"
              size="sm"
              threshold={100}
              showValue
            />
          </div>
        </div>

        {/* CDN */}
        <div className="flex items-center justify-between p-3 bg-bg-secondary rounded-md">
          <div className="flex items-center gap-3">
            <span className="text-text-primary font-medium">CDN</span>
            <HealthScoreBadge score={45} size="sm" />
          </div>
          <div className="flex items-center gap-4">
            <SparklineWithLabel
              labelText="Availability"
              data={criticalAvailability}
              type="availability"
              size="sm"
              showValue
            />
            <SparklineWithLabel
              labelText="Latency"
              data={criticalLatency}
              type="latency"
              size="sm"
              threshold={100}
              showValue
            />
          </div>
        </div>
      </div>
    </div>
  ),
};
