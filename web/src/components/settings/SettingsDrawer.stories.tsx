/**
 * SettingsDrawer Storybook Stories
 *
 * Demonstrates the master settings drawer component containing all settings sections.
 * Shows drawer behavior, section organization, and responsive layouts.
 *
 * Variants:
 * - Closed: Drawer is hidden
 * - Open: Drawer is visible with all sections
 * - Mobile viewport: Full-screen drawer on small screens
 * - Desktop viewport: Side drawer on larger screens
 *
 * Note: SettingsProvider and I18nextProvider are provided by global decorators
 * in .storybook/preview.tsx. For individual section testing, see section-specific stories.
 */

import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { SettingsDrawer } from "./SettingsDrawer";

const meta: Meta<typeof SettingsDrawer> = {
  title: "Settings/SettingsDrawer",
  component: SettingsDrawer,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Master settings drawer containing all configuration sections: Network, WiFi, DNS, Health Checks, Performance, Discovery, SNMP, Thresholds, and Appearance. Provides comprehensive application configuration with auto-save, validation, and responsive design.",
      },
    },
  },
  tags: ["autodocs"],
  argTypes: {
    isOpen: {
      control: "boolean",
      description: "Whether drawer is open",
    },
    version: {
      control: "text",
      description: "Application version string",
    },
  },
  decorators: [
    (Story) => (
      <div className="h-screen bg-surface-base">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Drawer closed - hidden state
 */
export const Closed: Story = {
  args: {
    isOpen: false,
    onClose: () => {},
    version: "1.0.0",
  },
};

/**
 * Drawer open - all sections visible
 */
export const Open: Story = {
  args: {
    isOpen: true,
    onClose: () => {},
    version: "1.0.0",
  },
};

/**
 * Development version
 */
export const DevVersion: Story = {
  args: {
    isOpen: true,
    onClose: () => {},
    version: "dev",
  },
};

/**
 * Production version
 */
export const ProductionVersion: Story = {
  args: {
    isOpen: true,
    onClose: () => {},
    version: "1.2.3",
  },
};

/**
 * Interactive drawer - can be opened and closed
 */
export const Interactive: Story = {
  render: function InteractiveStory() {
    const [isOpen, setIsOpen] = useState(false);

    return (
      <div className="h-screen bg-surface-base p-4">
        <button
          onClick={() => setIsOpen(true)}
          className="px-4 py-2 bg-brand-primary text-text-inverse rounded-lg hover:bg-brand-accent"
        >
          Open Settings
        </button>
        <SettingsDrawer isOpen={isOpen} onClose={() => setIsOpen(false)} version="1.0.0" />
      </div>
    );
  },
};

/**
 * Mobile viewport - full-screen drawer
 */
export const MobileViewport: Story = {
  args: {
    isOpen: true,
    onClose: () => {},
    version: "1.0.0",
  },
  parameters: {
    viewport: {
      defaultViewport: "mobile1",
    },
  },
};

/**
 * Tablet viewport - side drawer
 */
export const TabletViewport: Story = {
  args: {
    isOpen: true,
    onClose: () => {},
    version: "1.0.0",
  },
  parameters: {
    viewport: {
      defaultViewport: "tablet",
    },
  },
};

/**
 * Desktop viewport - side drawer
 */
export const DesktopViewport: Story = {
  args: {
    isOpen: true,
    onClose: () => {},
    version: "1.0.0",
  },
  parameters: {
    viewport: {
      defaultViewport: "desktop",
    },
  },
};

/**
 * With backdrop click - demonstrates close behavior
 */
export const WithBackdrop: Story = {
  render: function BackdropStory() {
    const [isOpen, setIsOpen] = useState(true);
    const [clickCount, setClickCount] = useState(0);

    const handleClose = () => {
      setIsOpen(false);
      setClickCount((c) => c + 1);
      // Reopen after a brief delay to show the interaction
      setTimeout(() => setIsOpen(true), 500);
    };

    return (
      <div className="h-screen bg-surface-base p-4">
        <div className="mb-4 p-4 bg-surface-raised rounded-lg">
          <p className="body-small text-text-primary">
            Click the dark backdrop to close the drawer.
          </p>
          <p className="caption text-text-muted mt-1">Backdrop clicks: {clickCount}</p>
        </div>
        <SettingsDrawer isOpen={isOpen} onClose={handleClose} version="1.0.0" />
      </div>
    );
  },
};

/**
 * Dark theme - shows drawer in dark mode
 * Note: Dark theme is the default via global decorators
 */
export const DarkTheme: Story = {
  args: {
    isOpen: true,
    onClose: () => {},
    version: "1.0.0",
  },
};

/**
 * With sample content behind drawer
 */
export const WithContent: Story = {
  render: function ContentStory() {
    const [isOpen, setIsOpen] = useState(true);

    return (
      <div className="h-screen bg-surface-base">
        {/* Sample page content */}
        <div className="p-8">
          <h1 className="heading-1 mb-4">Network Dashboard</h1>
          <div className="grid grid-cols-3 gap-4">
            <div className="p-6 bg-surface-raised rounded-lg">
              <h2 className="heading-3 mb-2">Network Status</h2>
              <p className="body-small text-text-muted">Connected</p>
            </div>
            <div className="p-6 bg-surface-raised rounded-lg">
              <h2 className="heading-3 mb-2">Speed Test</h2>
              <p className="body-small text-text-muted">150 Mbps</p>
            </div>
            <div className="p-6 bg-surface-raised rounded-lg">
              <h2 className="heading-3 mb-2">Devices</h2>
              <p className="body-small text-text-muted">12 found</p>
            </div>
          </div>
          <button
            onClick={() => setIsOpen(true)}
            className="mt-4 px-4 py-2 bg-brand-primary text-text-inverse rounded-lg hover:bg-brand-accent"
          >
            Open Settings
          </button>
        </div>

        <SettingsDrawer isOpen={isOpen} onClose={() => setIsOpen(false)} version="1.0.0" />
      </div>
    );
  },
};

/**
 * Scrollable content - shows drawer with many sections
 */
export const ScrollableContent: Story = {
  args: {
    isOpen: true,
    onClose: () => {},
    version: "1.0.0",
  },
  parameters: {
    docs: {
      description: {
        story:
          "Demonstrates drawer scrolling behavior when content exceeds viewport height. All sections are expanded to show scroll functionality.",
      },
    },
  },
};
