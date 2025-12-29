import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { button, cn, radius, spacing } from "../../styles/theme";
import { ImprovedHelpModal } from "./ImprovedHelpModal";

/**
 * ImprovedHelpModal is a comprehensive help center with tabbed navigation,
 * search functionality, and rich documentation for all application features.
 *
 * Features:
 * - Multi-section help content (About, Getting Started, Link, WiFi, etc.)
 * - Search filtering across all sections
 * - Icon-based sidebar navigation
 * - Responsive design for mobile/tablet/desktop
 * - Keyboard support (ESC to close)
 * - Backdrop click to close
 *
 * This is the main help modal shown when users click the Help button in the header.
 */
const meta: Meta<typeof ImprovedHelpModal> = {
  title: "Help/ImprovedHelpModal",
  component: ImprovedHelpModal,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Full-featured help center modal with tabbed navigation, search, and comprehensive documentation for all The Seed features.",
      },
    },
  },
  tags: ["autodocs"],
  argTypes: {
    isOpen: {
      control: "boolean",
      description: "Controls modal visibility",
    },
    onClose: {
      action: "closed",
      description: "Callback when modal is closed",
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Modal in closed state - nothing is rendered.
 */
export const Closed: Story = {
  args: {
    isOpen: false,
    onClose: () => {
      // intentionally empty - story placeholder callback
    },
  },
};

/**
 * Modal open showing the default About section.
 */
export const OpenDefault: Story = {
  args: {
    isOpen: true,
    onClose: () => {
      // intentionally empty - story placeholder callback
    },
  },
};

/**
 * Interactive demo with open/close button.
 */
export const Interactive: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);

    return (
      <div className={cn("min-h-screen bg-surface-base", spacing.pad.xl)}>
        <div className="max-w-xl mx-auto text-center">
          <h1 className={cn("heading-2 text-text-primary", spacing.margin.bottom.content)}>
            Help Center Demo
          </h1>
          <p className={cn("body text-text-secondary", spacing.margin.bottom.section)}>
            Click the button below to open the help center. Use the sidebar to navigate between
            sections, or use the search to filter content.
          </p>
          <button
            type="button"
            onClick={() => setIsOpen(true)}
            className={cn(
              button.size.lg,
              "bg-brand-primary text-text-inverse",
              radius.lg,
              "hover:bg-brand-primary/90 transition-colors font-medium",
            )}
          >
            Open Help Center
          </button>
        </div>
        <ImprovedHelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: "Interactive demo - click the button to open the help modal.",
      },
    },
  },
};

/**
 * Search functionality demonstration.
 */
export const SearchDemo: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);

    return (
      <div className="min-h-screen bg-surface-base">
        <div
          className={cn(spacing.pad.default, "bg-surface-raised border-b border-surface-border")}
        >
          <h2 className={cn("heading-3 text-text-primary", spacing.margin.bottom.inline)}>
            Search Functionality
          </h2>
          <p className="body-small text-text-muted">
            Use the search box in the sidebar to filter help sections. Try searching for:
          </p>
          <ul
            className={cn(
              "list-disc list-inside body-small text-text-muted",
              spacing.margin.top.inline,
            )}
          >
            <li>"wifi" - Shows WiFi-related sections</li>
            <li>"dns" - Shows DNS test section</li>
            <li>"gateway" - Shows Gateway section</li>
            <li>"performance" - Shows Performance Tests section</li>
          </ul>
        </div>
        <ImprovedHelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: "Demonstrates the search functionality - type in the search box to filter sections.",
      },
    },
  },
};

/**
 * All sections overview.
 */
export const AllSections: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);

    return (
      <div className="min-h-screen bg-surface-base">
        <div
          className={cn(spacing.pad.default, "bg-surface-raised border-b border-surface-border")}
        >
          <h2 className={cn("heading-3 text-text-primary", spacing.margin.bottom.inline)}>
            All Help Sections
          </h2>
          <p className="body-small text-text-muted">
            Navigate through all available sections using the sidebar:
          </p>
          <div
            className={cn(
              "grid grid-cols-2 md:grid-cols-5",
              spacing.gap.compact,
              spacing.margin.top.heading,
            )}
          >
            <SectionBadge name="About" />
            <SectionBadge name="Getting Started" />
            <SectionBadge name="Link Status" />
            <SectionBadge name="Cable Test" />
            <SectionBadge name="WiFi Status" />
            <SectionBadge name="Network & DHCP" />
            <SectionBadge name="Gateway" />
            <SectionBadge name="DNS Tests" />
            <SectionBadge name="Performance" />
            <SectionBadge name="Discovery" />
          </div>
        </div>
        <ImprovedHelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: "Shows all available help sections in the navigation sidebar.",
      },
    },
  },
};

/**
 * Mobile viewport demonstration.
 */
export const MobileView: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);

    return (
      <div className="min-h-screen bg-surface-base">
        <ImprovedHelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
      </div>
    );
  },
  parameters: {
    viewport: {
      defaultViewport: "mobile1",
    },
    docs: {
      description: {
        story: "Help modal on mobile viewport - sidebar and content may stack vertically.",
      },
    },
  },
};

/**
 * Tablet viewport demonstration.
 */
export const TabletView: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);

    return (
      <div className="min-h-screen bg-surface-base">
        <ImprovedHelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
      </div>
    );
  },
  parameters: {
    viewport: {
      defaultViewport: "tablet",
    },
    docs: {
      description: {
        story: "Help modal on tablet viewport.",
      },
    },
  },
};

/**
 * Dark theme demonstration.
 */
export const DarkTheme: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);

    return (
      <div className="min-h-screen bg-surface-base dark">
        <ImprovedHelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
      </div>
    );
  },
  parameters: {
    backgrounds: { default: "dark" },
    docs: {
      description: {
        story: "Help modal with dark theme styling.",
      },
    },
  },
};

/**
 * Backdrop click to close demonstration.
 */
export const BackdropClose: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);
    const [closeCount, setCloseCount] = useState(0);

    return (
      <div className="min-h-screen bg-surface-base">
        {!isOpen && (
          <div className={cn(spacing.pad.xl, "text-center")}>
            <p className={cn("body text-text-secondary", spacing.margin.bottom.content)}>
              Modal closed {closeCount} time(s)
            </p>
            <button
              type="button"
              onClick={() => setIsOpen(true)}
              className={cn(button.size.md, "bg-brand-primary text-text-inverse", radius.lg)}
            >
              Reopen Modal
            </button>
          </div>
        )}
        <ImprovedHelpModal
          isOpen={isOpen}
          onClose={() => {
            setIsOpen(false);
            setCloseCount((c) => c + 1);
          }}
        />
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          "Click the dark backdrop area or the X button to close the modal. Counter tracks close events.",
      },
    },
  },
};

/**
 * Keyboard navigation demonstration.
 */
export const KeyboardNavigation: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);

    return (
      <div className="min-h-screen bg-surface-base">
        <div
          className={cn(spacing.pad.default, "bg-surface-raised border-b border-surface-border")}
        >
          <h2 className={cn("heading-3 text-text-primary", spacing.margin.bottom.inline)}>
            Keyboard Navigation
          </h2>
          <p className="body-small text-text-muted">The help modal supports keyboard navigation:</p>
          <ul
            className={cn(
              "list-disc list-inside body-small text-text-muted",
              spacing.margin.top.inline,
            )}
          >
            <li>
              Press <kbd className={cn(spacing.kbd, "bg-surface-hover rounded text-xs")}>ESC</kbd>{" "}
              to close
            </li>
            <li>
              Use <kbd className={cn(spacing.kbd, "bg-surface-hover rounded text-xs")}>Tab</kbd> to
              navigate between elements
            </li>
            <li>
              Press <kbd className={cn(spacing.kbd, "bg-surface-hover rounded text-xs")}>Enter</kbd>{" "}
              to select a section
            </li>
          </ul>
        </div>
        <ImprovedHelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: "Demonstrates keyboard accessibility features.",
      },
    },
  },
};

// Helper component
function SectionBadge({ name }: { name: string }) {
  return (
    <span
      className={cn(
        spacing.chip.sm,
        "bg-surface-base border border-surface-border rounded text-xs text-text-secondary",
      )}
    >
      {name}
    </span>
  );
}
