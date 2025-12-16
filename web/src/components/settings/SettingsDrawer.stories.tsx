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
import { spacing, button, radius } from "../../styles/theme";

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
      <div className={`h-screen bg-surface-base ${spacing.pad.default}`}>
        <button
          onClick={() => setIsOpen(true)}
          className={`${button.size.md} bg-brand-primary text-text-inverse ${radius.lg} hover:bg-brand-accent`}
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
      <div className={`h-screen bg-surface-base ${spacing.pad.default}`}>
        <div
          className={`${spacing.margin.bottom.content} ${spacing.pad.default} bg-surface-raised ${radius.lg}`}
        >
          <p className="body-small text-text-primary">
            Click the dark backdrop to close the drawer.
          </p>
          <p className={`caption text-text-muted ${spacing.margin.top.inline}`}>
            Backdrop clicks: {clickCount}
          </p>
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
        <div className={spacing.pad.xl}>
          <h1 className={`heading-1 ${spacing.margin.bottom.content}`}>Network Dashboard</h1>
          <div className={`grid grid-cols-3 ${spacing.gap.comfortable}`}>
            <div className={`${spacing.pad.lg} bg-surface-raised ${radius.lg}`}>
              <h2 className={`heading-3 ${spacing.margin.bottom.inline}`}>Network Status</h2>
              <p className="body-small text-text-muted">Connected</p>
            </div>
            <div className={`${spacing.pad.lg} bg-surface-raised ${radius.lg}`}>
              <h2 className={`heading-3 ${spacing.margin.bottom.inline}`}>Speed Test</h2>
              <p className="body-small text-text-muted">150 Mbps</p>
            </div>
            <div className={`${spacing.pad.lg} bg-surface-raised ${radius.lg}`}>
              <h2 className={`heading-3 ${spacing.margin.bottom.inline}`}>Devices</h2>
              <p className="body-small text-text-muted">12 found</p>
            </div>
          </div>
          <button
            onClick={() => setIsOpen(true)}
            className={`${spacing.margin.top.content} ${button.size.md} bg-brand-primary text-text-inverse ${radius.lg} hover:bg-brand-accent`}
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
