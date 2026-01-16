/**
 * WiFiSettings Storybook Stories
 *
 * Demonstrates the WiFi settings component for selecting and configuring
 * wireless network interfaces.
 *
 * Variants:
 * - With available interfaces: Dropdown selector
 * - No interfaces: Text input fallback
 * - Wireless detected: Active wireless interface
 * - No wireless: No wireless interface detected
 * - Interactive selection
 */

import type { Meta, StoryFn, StoryObj } from "@storybook/react-vite";
import type React from "react";
import { useState } from "react";
import { cn, spacing } from "../../../styles/theme";
import type { SaveStatus, WiFiSettings as WiFiSettingsType } from "../../../types/settings";
import { WiFiSettings } from "./WiFiSettings";

const meta: Meta<typeof WiFiSettings> = {
  title: "Settings/WiFiSettings",
  component: WiFiSettings,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "WiFi interface configuration panel. Displays available wireless interfaces as a dropdown or provides a text input if no interfaces are detected.",
      },
    },
  },
  tags: ["autodocs"],
  argTypes: {
    wifiStatus: {
      control: "select",
      options: ["idle", "saving", "saved", "error"],
      description: "Auto-save status indicator",
    },
  },
  decorators: [
    (StoryComponent: StoryFn): React.ReactElement => (
      <div class="w-[400px]">
        <StoryComponent />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * With available WiFi interfaces - shows dropdown selector
 */
export const WithAvailableInterfaces: Story = {
  args: {
    wifiSettings: {
      interface: "wlan0",
      availableWifi: ["wlan0", "wlan1", "wlp2s0"],
      isWireless: true,
    },
    setWifiSettings: () => {
      // intentionally empty
    },
    wifiStatus: "idle",
  },
};

/**
 * macOS WiFi interface - typical en0 naming
 */
export const MacosInterface: Story = {
  args: {
    wifiSettings: {
      interface: "en0",
      availableWifi: ["en0", "en1"],
      isWireless: true,
    },
    setWifiSettings: () => {
      // intentionally empty
    },
    wifiStatus: "idle",
  },
};

/**
 * No available interfaces - shows text input for manual entry
 */
export const NoAvailableInterfaces: Story = {
  args: {
    wifiSettings: {
      interface: "",
      availableWifi: [],
      isWireless: false,
    },
    setWifiSettings: () => {
      // intentionally empty
    },
    wifiStatus: "idle",
  },
};

/**
 * Manual interface entry - no auto-detection
 */
export const ManualEntry: Story = {
  args: {
    wifiSettings: {
      interface: "wlan0",
      availableWifi: [],
      isWireless: false,
    },
    setWifiSettings: () => {
      // intentionally empty
    },
    wifiStatus: "idle",
  },
};

/**
 * Single WiFi interface available
 */
export const SingleInterface: Story = {
  args: {
    wifiSettings: {
      interface: "wlan0",
      availableWifi: ["wlan0"],
      isWireless: true,
    },
    setWifiSettings: () => {
      // intentionally empty
    },
    wifiStatus: "idle",
  },
};

/**
 * Wireless interface not detected
 */
export const NoWirelessDetected: Story = {
  args: {
    wifiSettings: {
      interface: "eth0",
      availableWifi: ["eth0", "eth1"],
      isWireless: false,
    },
    setWifiSettings: () => {
      // intentionally empty
    },
    wifiStatus: "idle",
  },
};

/**
 * Saving state
 */
export const Saving: Story = {
  args: {
    wifiSettings: {
      interface: "wlan0",
      availableWifi: ["wlan0", "wlan1"],
      isWireless: true,
    },
    setWifiSettings: () => {
      // intentionally empty
    },
    wifiStatus: "saving",
  },
};

/**
 * Saved state
 */
export const Saved: Story = {
  args: {
    wifiSettings: {
      interface: "wlan0",
      availableWifi: ["wlan0", "wlan1"],
      isWireless: true,
    },
    setWifiSettings: () => {
      // intentionally empty
    },
    wifiStatus: "saved",
  },
};

/**
 * Interactive WiFi settings - fully functional
 */
export const Interactive: Story = {
  render: function interactiveStory() {
    const [wifiSettings, setWifiSettings] = useState<WiFiSettingsType>({
      interface: "wlan0",
      availableWifi: ["wlan0", "wlan1", "wlp2s0"],
      isWireless: true,
    });
    const [status, setStatus] = useState<SaveStatus>("idle");

    const handleSetWifiSettings = (updater: React.SetStateAction<WiFiSettingsType>) => {
      setWifiSettings(updater);
      setStatus("saving");

      setTimeout(() => {
        setStatus("saved");
        setTimeout(() => {
          setStatus("idle");
        }, 2000);
      }, 800);
    };

    return (
      <WiFiSettings
        wifiSettings={wifiSettings}
        setWifiSettings={handleSetWifiSettings}
        wifiStatus={status}
      />
    );
  },
};

/**
 * Comparison of interface states
 */
export const Comparison: Story = {
  render: () => (
    <div class={cn("stack-lg", spacing.pad.default)}>
      <div>
        <p class={cn("caption text-text-muted", spacing.margin.bottom.inline)}>
          Multiple interfaces available
        </p>
        <WiFiSettings
          wifiSettings={{
            interface: "wlan0",
            availableWifi: ["wlan0", "wlan1", "wlp2s0"],
            isWireless: true,
          }}
          setWifiSettings={() => {
            // intentionally empty
          }}
          wifiStatus="idle"
        />
      </div>
      <div>
        <p class={cn("caption text-text-muted", spacing.margin.bottom.inline)}>
          No interfaces (manual entry)
        </p>
        <WiFiSettings
          wifiSettings={{
            interface: "wlan0",
            availableWifi: [],
            isWireless: false,
          }}
          setWifiSettings={() => {
            // intentionally empty
          }}
          wifiStatus="idle"
        />
      </div>
      <div>
        <p class={cn("caption text-text-muted", spacing.margin.bottom.inline)}>Saving state</p>
        <WiFiSettings
          wifiSettings={{
            interface: "wlan0",
            availableWifi: ["wlan0"],
            isWireless: true,
          }}
          setWifiSettings={() => {
            // intentionally empty
          }}
          wifiStatus="saving"
        />
      </div>
    </div>
  ),
};
