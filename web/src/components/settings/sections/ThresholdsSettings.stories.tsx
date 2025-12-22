/**
 * ThresholdsSettings Storybook Stories
 *
 * Demonstrates the thresholds settings component for configuring performance
 * and health check thresholds for various network tests.
 *
 * Variants:
 * - Default values: Standard threshold configuration
 * - Custom values: User-modified thresholds
 * - All sections expanded: Shows all threshold categories
 * - Interactive with auto-save: Demonstrates save status feedback
 */

import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { ThresholdsSettings } from "./ThresholdsSettings";
import type { SettingsThresholds, SaveStatus } from "../../../types/settings";
import { spacing, cn } from "../../../styles/theme";

const defaultThresholds: SettingsThresholds = {
  dns: { good: 50, warning: 100 },
  gateway: { good: 20, warning: 50 },
  wifi: { good: -50, warning: -70 },
  customPing: { good: 50, warning: 100 },
  customTcp: { good: 100, warning: 500 },
  customHttp: { good: 500, warning: 2000 },
  httpTimings: {
    dns: { good: 100, warning: 500 },
    tcp: { good: 100, warning: 500 },
    tls: { good: 150, warning: 500 },
    ttfb: { good: 500, warning: 2000 },
  },
};

const customThresholds: SettingsThresholds = {
  dns: { good: 30, warning: 80 },
  gateway: { good: 10, warning: 30 },
  wifi: { good: -40, warning: -60 },
  customPing: { good: 40, warning: 80 },
  customTcp: { good: 80, warning: 400 },
  customHttp: { good: 400, warning: 1500 },
  httpTimings: {
    dns: { good: 80, warning: 400 },
    tcp: { good: 80, warning: 400 },
    tls: { good: 120, warning: 400 },
    ttfb: { good: 400, warning: 1500 },
  },
};

const meta: Meta<typeof ThresholdsSettings> = {
  title: "Settings/ThresholdsSettings",
  component: ThresholdsSettings,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Configuration panel for performance and health check thresholds. Allows users to customize 'good' and 'warning' thresholds for DNS, Gateway, WiFi, Ping, TCP, and HTTP tests including per-phase HTTP timings.",
      },
    },
  },
  tags: ["autodocs"],
  argTypes: {
    thresholdsStatus: {
      control: "select",
      options: ["idle", "saving", "saved", "error"],
      description: "Auto-save status indicator",
    },
  },
  decorators: [
    (Story) => (
      <div className="w-[500px] max-h-[600px] overflow-y-auto">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default threshold values - standard configuration
 */
export const Default: Story = {
  args: {
    thresholds: defaultThresholds,
    setThresholds: () => {},
    thresholdsStatus: "idle",
  },
};

/**
 * Custom threshold values - user has modified settings
 */
export const CustomValues: Story = {
  args: {
    thresholds: customThresholds,
    setThresholds: () => {},
    thresholdsStatus: "idle",
  },
};

/**
 * Saving state - shows auto-save in progress
 */
export const Saving: Story = {
  args: {
    thresholds: defaultThresholds,
    setThresholds: () => {},
    thresholdsStatus: "saving",
  },
};

/**
 * Saved state - confirms successful save
 */
export const Saved: Story = {
  args: {
    thresholds: defaultThresholds,
    setThresholds: () => {},
    thresholdsStatus: "saved",
  },
};

/**
 * Error state - save failed
 */
export const SaveError: Story = {
  args: {
    thresholds: defaultThresholds,
    setThresholds: () => {},
    thresholdsStatus: "error",
  },
};

/**
 * Strict thresholds - very tight performance requirements
 */
export const StrictThresholds: Story = {
  args: {
    thresholds: {
      dns: { good: 20, warning: 40 },
      gateway: { good: 5, warning: 15 },
      wifi: { good: -30, warning: -50 },
      customPing: { good: 20, warning: 50 },
      customTcp: { good: 50, warning: 200 },
      customHttp: { good: 200, warning: 800 },
      httpTimings: {
        dns: { good: 50, warning: 200 },
        tcp: { good: 50, warning: 200 },
        tls: { good: 80, warning: 300 },
        ttfb: { good: 200, warning: 800 },
      },
    },
    setThresholds: () => {},
    thresholdsStatus: "idle",
  },
};

/**
 * Relaxed thresholds - lenient performance requirements
 */
export const RelaxedThresholds: Story = {
  args: {
    thresholds: {
      dns: { good: 100, warning: 200 },
      gateway: { good: 50, warning: 100 },
      wifi: { good: -60, warning: -80 },
      customPing: { good: 100, warning: 200 },
      customTcp: { good: 200, warning: 1000 },
      customHttp: { good: 1000, warning: 3000 },
      httpTimings: {
        dns: { good: 200, warning: 800 },
        tcp: { good: 200, warning: 800 },
        tls: { good: 300, warning: 1000 },
        ttfb: { good: 1000, warning: 3000 },
      },
    },
    setThresholds: () => {},
    thresholdsStatus: "idle",
  },
};

/**
 * Interactive thresholds - fully functional with auto-save simulation
 */
export const Interactive: Story = {
  render: function InteractiveStory() {
    const [thresholds, setThresholds] =
      useState<SettingsThresholds>(defaultThresholds);
    const [status, setStatus] = useState<SaveStatus>("idle");

    // Simulate auto-save behavior
    const handleSetThresholds = (
      updater: React.SetStateAction<SettingsThresholds>
    ) => {
      setThresholds(updater);
      setStatus("saving");

      // Simulate save delay
      setTimeout(() => {
        setStatus("saved");
        setTimeout(() => {
          setStatus("idle");
        }, 2000);
      }, 800);
    };

    return (
      <ThresholdsSettings
        thresholds={thresholds}
        setThresholds={handleSetThresholds}
        thresholdsStatus={status}
      />
    );
  },
};

/**
 * All save states demonstrated side by side
 */
export const SaveStates: Story = {
  render: () => (
    <div className={cn("stack-lg", spacing.pad.default)}>
      <div>
        <p
          className={cn(
            "caption text-text-muted",
            spacing.margin.bottom.inline
          )}
        >
          Idle (no changes)
        </p>
        <div className="w-[400px]">
          <ThresholdsSettings
            thresholds={defaultThresholds}
            setThresholds={() => {}}
            thresholdsStatus="idle"
          />
        </div>
      </div>
      <div>
        <p
          className={cn(
            "caption text-text-muted",
            spacing.margin.bottom.inline
          )}
        >
          Saving
        </p>
        <div className="w-[400px]">
          <ThresholdsSettings
            thresholds={defaultThresholds}
            setThresholds={() => {}}
            thresholdsStatus="saving"
          />
        </div>
      </div>
      <div>
        <p
          className={cn(
            "caption text-text-muted",
            spacing.margin.bottom.inline
          )}
        >
          Saved
        </p>
        <div className="w-[400px]">
          <ThresholdsSettings
            thresholds={defaultThresholds}
            setThresholds={() => {}}
            thresholdsStatus="saved"
          />
        </div>
      </div>
      <div>
        <p
          className={cn(
            "caption text-text-muted",
            spacing.margin.bottom.inline
          )}
        >
          Error
        </p>
        <div className="w-[400px]">
          <ThresholdsSettings
            thresholds={defaultThresholds}
            setThresholds={() => {}}
            thresholdsStatus="error"
          />
        </div>
      </div>
    </div>
  ),
};
