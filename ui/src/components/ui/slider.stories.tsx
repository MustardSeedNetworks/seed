import type { Meta, StoryObj } from "@storybook/react-vite";
import type React from "react";
import { useState } from "react";
import { Slider } from "./Slider";

/**
 * Slider component for numeric input with visual feedback.
 *
 * Use cases:
 * - Timing configurations (probe intervals, timeouts)
 * - Performance tuning (worker threads, batch sizes)
 * - Threshold adjustments (retry counts, buffer sizes)
 *
 * Key features:
 * - Visual track with filled progress indicator
 * - Custom value formatters (ms, seconds, minutes, counts)
 * - Optional end labels for context (e.g., "Slower ◄────► Faster")
 * - Keyboard accessible (arrows, Page Up/Down, Home/End)
 * - Touch-friendly for mobile devices
 */
const meta: Meta<typeof Slider> = {
  title: "UI/Slider",
  component: Slider,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "A customizable range slider for numeric input with visual feedback and formatting options.",
      },
    },
  },
  tags: ["autodocs"],
  argTypes: {
    value: {
      control: { type: "number" },
      description: "Current slider value",
    },
    min: {
      control: { type: "number" },
      description: "Minimum value",
    },
    max: {
      control: { type: "number" },
      description: "Maximum value",
    },
    step: {
      control: { type: "number" },
      description: "Step increment",
    },
    label: {
      control: { type: "text" },
      description: "Label displayed above slider",
    },
    leftLabel: {
      control: { type: "text" },
      description: "Label at left end (e.g., 'Slower')",
    },
    rightLabel: {
      control: { type: "text" },
      description: "Label at right end (e.g., 'Faster')",
    },
    disabled: {
      control: { type: "boolean" },
      description: "Disable slider interaction",
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Interactive wrapper for stories
 */
function _sliderWrapper(
  props: Omit<React.ComponentProps<typeof Slider>, "value" | "onChange">,
): React.JSX.Element {
  const [value, setValue] = useState(props.min + (props.max - props.min) / 2);
  return <Slider {...props} value={value} onChange={setValue} />;
}

/**
 * Default slider - simple numeric range
 */
export const Default: Story = {
  render: () => <sliderWrapper min={0} max={100} step={1} label="Volume" />,
};

/**
 * Probe Interval - milliseconds formatter, 25ms to 500ms
 * Real-world use: Configure how often to probe network devices
 */
export const ProbeInterval: Story = {
  render: () => (
    <sliderWrapper
      min={25}
      max={500}
      step={25}
      label="Probe Interval"
      leftLabel="Faster"
      rightLabel="Slower"
      formatValue={(v: number): string => `${v}ms`}
    />
  ),
};

/**
 * Scan Timeout - milliseconds to seconds conversion
 * Real-world use: Configure scan timeout duration
 */
export const ScanTimeout: Story = {
  render: () => (
    <sliderWrapper
      min={500}
      max={10000}
      step={500}
      label="Scan Timeout"
      leftLabel="Quick"
      rightLabel="Patient"
      formatValue={(v: number): string => (v >= 1000 ? `${v / 1000}s` : `${v}ms`)}
    />
  ),
};

/**
 * Worker Threads - discrete count values
 * Real-world use: Configure parallel worker threads
 */
export const Workers: Story = {
  render: () => (
    <sliderWrapper
      min={5}
      max={100}
      step={5}
      label="Worker Threads"
      leftLabel="Conservative"
      rightLabel="Aggressive"
      formatValue={(v: number): string => `${v} workers`}
    />
  ),
};

/**
 * Rescan Interval - minutes formatter
 * Real-world use: Configure automatic rescan interval
 */
export const RescanInterval: Story = {
  render: () => (
    <sliderWrapper
      min={1}
      max={60}
      step={1}
      label="Rescan Interval"
      leftLabel="Frequent"
      rightLabel="Rare"
      formatValue={(v: number): string => `${v} min`}
    />
  ),
};

/**
 * Banner Timeout - milliseconds formatter
 * Real-world use: Configure banner grab timeout
 */
export const BannerTimeout: Story = {
  render: () => (
    <sliderWrapper
      min={500}
      max={10000}
      step={500}
      label="Banner Timeout"
      formatValue={(v: number): string => `${v}ms`}
    />
  ),
};

/**
 * Without labels - minimal slider
 */
export const NoLabels: Story = {
  render: () => <sliderWrapper min={0} max={100} step={10} />,
};

/**
 * With label only - no end labels
 */
export const LabelOnly: Story = {
  render: () => (
    <sliderWrapper
      min={0}
      max={100}
      step={5}
      label="Brightness"
      formatValue={(v: number): string => `${v}%`}
    />
  ),
};

/**
 * With end labels only - no top label
 */
export const EndLabelsOnly: Story = {
  render: () => (
    <sliderWrapper min={0} max={100} step={10} leftLabel="Gentler" rightLabel="Aggressive" />
  ),
};

/**
 * Disabled state
 */
export const Disabled: Story = {
  render: () => (
    <sliderWrapper
      min={0}
      max={100}
      step={10}
      label="Disabled Slider"
      leftLabel="Min"
      rightLabel="Max"
      disabled={true}
    />
  ),
};

/**
 * Fine granularity - small steps
 */
export const FineGrained: Story = {
  render: () => (
    <sliderWrapper
      min={0}
      max={10}
      step={0.1}
      label="Precision"
      formatValue={(v: number): string => v.toFixed(1)}
    />
  ),
};

/**
 * Large range - big numbers
 */
export const LargeRange: Story = {
  render: () => (
    <sliderWrapper
      min={0}
      max={10000}
      step={100}
      label="Buffer Size"
      formatValue={(v: number): string => `${(v / 1000).toFixed(1)}KB`}
    />
  ),
};

/**
 * All scanner settings together - demonstrates real-world configuration panel
 */
export const ScannerSettings: Story = {
  render: () => {
    const [probeInterval, setProbeInterval] = useState(75);
    const [scanTimeout, setScanTimeout] = useState(2000);
    const [workers, setWorkers] = useState(20);
    const [rescanInterval, setRescanInterval] = useState(10);
    const [bannerTimeout, setBannerTimeout] = useState(2000);

    return (
      <div class="space-y-6 max-w-2xl">
        <h3 class="heading-3">Network Scanner Settings</h3>

        <Slider
          value={probeInterval}
          onChange={setProbeInterval}
          min={25}
          max={500}
          step={25}
          label="Probe Interval"
          leftLabel="Faster"
          rightLabel="Slower"
          formatValue={(v: number): string => `${v}ms`}
        />

        <Slider
          value={scanTimeout}
          onChange={setScanTimeout}
          min={500}
          max={10000}
          step={500}
          label="Scan Timeout"
          leftLabel="Quick"
          rightLabel="Patient"
          formatValue={(v: number): string => (v >= 1000 ? `${v / 1000}s` : `${v}ms`)}
        />

        <Slider
          value={workers}
          onChange={setWorkers}
          min={5}
          max={100}
          step={5}
          label="Worker Threads"
          leftLabel="Conservative"
          rightLabel="Aggressive"
          formatValue={(v: number): string => `${v} workers`}
        />

        <Slider
          value={rescanInterval}
          onChange={setRescanInterval}
          min={1}
          max={60}
          step={1}
          label="Rescan Interval"
          leftLabel="Frequent"
          rightLabel="Rare"
          formatValue={(v: number): string => `${v} min`}
        />

        <Slider
          value={bannerTimeout}
          onChange={setBannerTimeout}
          min={500}
          max={10000}
          step={500}
          label="Banner Timeout"
          leftLabel="Quick"
          rightLabel="Patient"
          formatValue={(v: number): string => `${v}ms`}
        />

        <div class="mt-8 p-4 bg-surface-raised rounded-lg border border-surface-border">
          <h4 class="heading-4 mb-2">Current Configuration</h4>
          <div class="body-small space-y-1 font-mono">
            <div>Probe Interval: {probeInterval}ms</div>
            <div>Scan Timeout: {scanTimeout}ms</div>
            <div>Workers: {workers}</div>
            <div>Rescan Interval: {rescanInterval} min</div>
            <div>Banner Timeout: {bannerTimeout}ms</div>
          </div>
        </div>
      </div>
    );
  },
};
