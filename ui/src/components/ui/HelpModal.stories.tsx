import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { button, cn, radius, spacing } from '../../styles/theme';
import { ImprovedHelpModal } from '../help/ImprovedHelpModal';
import { HelpItem, HelpModal, HelpSection } from './HelpModal';

/**
 * HelpModal provides contextual help content in a modal dialog overlay.
 * Two variants are available:
 * - HelpModal: Simple reusable modal with custom content
 * - ImprovedHelpModal: Comprehensive help center with tabbed navigation and search
 */
const meta: Meta<typeof ImprovedHelpModal> = {
  title: 'UI/HelpModal',
  component: ImprovedHelpModal,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'Modal dialog for displaying help content with search, navigation, and rich documentation for all application features.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    isOpen: {
      control: 'boolean',
      description: 'Controls modal visibility',
    },
    onClose: {
      action: 'closed',
      description: 'Callback when modal is closed',
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default help modal in closed state
 */
export const Closed: Story = {
  args: {
    isOpen: false,
    onClose: () => {
      // intentionally empty
    },
  },
  parameters: {
    docs: {
      description: {
        story: 'Modal in closed state - nothing is rendered.',
      },
    },
  },
};

/**
 * Help modal open showing the About section
 */
export const OpenAbout: Story = {
  args: {
    isOpen: true,
    onClose: () => {
      // intentionally empty
    },
  },
  parameters: {
    docs: {
      description: {
        story:
          'Modal open displaying the About section with application overview and feature highlights.',
      },
    },
  },
};

/**
 * Interactive demo with open/close toggle
 */
export const Interactive: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);

    return (
      <div class={cn('min-h-screen bg-surface-base', spacing.pad.xl)}>
        <button
          type="button"
          onClick={() => setIsOpen(true)}
          class={cn(
            button.size.md,
            'bg-brand-primary text-text-inverse',
            radius.lg,
            'hover:bg-brand-primary/90 transition-colors',
          )}
        >
          Open Help Center
        </button>
        <ImprovedHelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          'Interactive demo allowing you to open and close the help modal by clicking the button.',
      },
    },
  },
};

/**
 * Help modal with search functionality demonstration
 */
export const WithSearch: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);

    return (
      <div class="min-h-screen bg-surface-base">
        <div class={cn(spacing.pad.default, 'bg-surface-raised border-b border-surface-border')}>
          <p class="body-small text-text-muted">
            Use the search box in the modal sidebar to filter help sections. Try searching for
            "wifi", "dns", or "gateway".
          </p>
        </div>
        <ImprovedHelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          'Demonstrates search functionality - use the search box to filter help sections by keyword.',
      },
    },
  },
};

/**
 * Help modal showing all navigation sections
 */
export const AllSections: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);

    return (
      <div class="min-h-screen bg-surface-base">
        <div class={cn(spacing.pad.default, 'bg-surface-raised border-b border-surface-border')}>
          <h2 class={cn('heading-3 text-text-primary', spacing.margin.bottom.inline)}>
            Help Modal - All Sections
          </h2>
          <p class="body-small text-text-muted">
            Navigate through all available help sections using the sidebar
          </p>
        </div>
        <ImprovedHelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          'Full help center with all sections: About, Getting Started, Link Status, Cable Test, WiFi, Network, Gateway, DNS, Performance Tests, and Discovery.',
      },
    },
  },
};

/**
 * Mobile viewport demonstration
 */
export const MobileView: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);

    return (
      <div class="min-h-screen bg-surface-base">
        <ImprovedHelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
      </div>
    );
  },
  parameters: {
    viewport: {
      defaultViewport: 'mobile1',
    },
    docs: {
      description: {
        story:
          'Help modal on mobile viewport showing responsive layout with sidebar and content stacking.',
      },
    },
  },
};

/**
 * Tablet viewport demonstration
 */
export const TabletView: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);

    return (
      <div class="min-h-screen bg-surface-base">
        <ImprovedHelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
      </div>
    );
  },
  parameters: {
    viewport: {
      defaultViewport: 'tablet',
    },
    docs: {
      description: {
        story: 'Help modal on tablet viewport showing medium-sized layout adaptation.',
      },
    },
  },
};

/**
 * Dark theme demonstration
 */
export const DarkTheme: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);

    return (
      <div class="min-h-screen bg-surface-base dark">
        <ImprovedHelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
      </div>
    );
  },
  parameters: {
    backgrounds: { default: 'dark' },
    docs: {
      description: {
        story: 'Help modal styled for dark theme using theme tokens.',
      },
    },
  },
};

// ============================================================================
// Simple HelpModal Stories
// ============================================================================

const SIMPLE_HELP_MODAL_META: Meta<typeof HelpModal> = {
  title: 'UI/HelpModal/Simple',
  component: HelpModal,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'Simple reusable help modal for custom content - lightweight alternative to the full Help Center.',
      },
    },
  },
  tags: ['autodocs'],
};

export const SimpleHelpModalStories: Meta<typeof HelpModal> = {
  ...SIMPLE_HELP_MODAL_META,
};

/**
 * Simple help modal with basic content
 */
export const SimpleBasic: StoryObj<typeof HelpModal> = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);

    return (
      <div class="min-h-screen bg-surface-base">
        <HelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} title="Link Status Help">
          <p class={cn('body text-text-secondary', spacing.margin.bottom.content)}>
            The Link Status card monitors the physical layer connection of your network interface.
          </p>
          <ul class={cn('list-disc list-inside body-small text-text-secondary', spacing.stack.sm)}>
            <li>Carrier: Shows if cable is physically connected</li>
            <li>Speed: Negotiated link speed (e.g., 1000 Mbps)</li>
            <li>Duplex: Full or Half duplex communication</li>
            <li>MTU: Maximum transmission unit size</li>
          </ul>
        </HelpModal>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: 'Simple help modal with basic text content and bullet points.',
      },
    },
  },
};

/**
 * Simple help modal with structured sections
 */
export const SimpleWithSections: StoryObj<typeof HelpModal> = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);

    return (
      <div class="min-h-screen bg-surface-base">
        <HelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} title="WiFi Survey Help">
          <HelpSection title="What is a WiFi Survey?">
            <p class="body-small text-text-secondary">
              A WiFi survey maps wireless signal coverage across different locations. Use it to
              identify dead zones and optimize access point placement.
            </p>
          </HelpSection>

          <HelpSection title="Signal Strength Levels">
            <HelpItem
              term="-30 dBm"
              description="Excellent - Maximum signal strength"
              color="bg-status-success"
            />
            <HelpItem
              term="-67 dBm"
              description="Good - Reliable connection"
              color="bg-status-success"
            />
            <HelpItem
              term="-70 dBm"
              description="Fair - Adequate for basic use"
              color="bg-status-warning"
            />
            <HelpItem
              term="-80 dBm"
              description="Weak - Unstable connection"
              color="bg-status-error"
            />
          </HelpSection>

          <HelpSection title="How to Use">
            <ol
              class={cn(
                'list-decimal list-inside body-small text-text-secondary',
                spacing.stack.sm,
              )}
            >
              <li>Click "Start Survey" to begin</li>
              <li>Upload a floor plan image (optional)</li>
              <li>Walk to different locations and click "Add Sample"</li>
              <li>View the heatmap to identify coverage gaps</li>
              <li>Click "Complete" when finished</li>
            </ol>
          </HelpSection>
        </HelpModal>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          'Simple help modal using HelpSection and HelpItem components for structured content.',
      },
    },
  },
};

/**
 * Simple help modal with custom styled content
 */
export const SimpleCustomContent: StoryObj<typeof HelpModal> = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);

    return (
      <div class="min-h-screen bg-surface-base">
        <HelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} title="Speed Test Results">
          <div class={cn(spacing.gap.comfortable, 'flex flex-col')}>
            <div
              class={cn(
                'bg-brand-primary/10 border border-brand-primary/20',
                radius.lg,
                spacing.pad.default,
              )}
            >
              <h4 class={cn('font-semibold text-brand-primary', spacing.margin.bottom.inline)}>
                Understanding Your Results
              </h4>
              <p class="body-small text-text-secondary">
                Speed test results show your connection's maximum throughput under ideal conditions.
                Real-world performance may vary based on network congestion, server load, and
                simultaneous connections.
              </p>
            </div>

            <div
              class={cn(
                'bg-status-warning/10 border border-status-warning/20',
                radius.lg,
                spacing.pad.default,
              )}
            >
              <h4 class={cn('font-semibold text-status-warning', spacing.margin.bottom.inline)}>
                Factors Affecting Speed
              </h4>
              <ul
                class={cn('list-disc list-inside body-small text-text-secondary', spacing.stack.xs)}
              >
                <li>Distance from router or access point</li>
                <li>Number of connected devices</li>
                <li>WiFi channel interference</li>
                <li>ISP throttling or congestion</li>
                <li>Time of day (peak usage hours)</li>
              </ul>
            </div>

            <div
              class={cn(
                'bg-status-success/10 border border-status-success/20',
                radius.lg,
                spacing.pad.default,
              )}
            >
              <h4 class={cn('font-semibold text-status-success', spacing.margin.bottom.inline)}>
                Improving Speed
              </h4>
              <p class="body-small text-text-secondary">
                Move closer to the router, switch to 5GHz band, reduce device count, or upgrade your
                internet plan for better performance.
              </p>
            </div>
          </div>
        </HelpModal>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: 'Simple help modal with custom styled callout boxes and formatted content.',
      },
    },
  },
};

/**
 * Simple help modal with interactive elements
 */
export const SimpleInteractive: StoryObj<typeof HelpModal> = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);

    return (
      <div class={cn('min-h-screen bg-surface-base', spacing.pad.xl)}>
        <div class={cn('max-w-2xl mx-auto flex flex-col', spacing.gap.comfortable)}>
          <h1 class="heading-2 text-text-primary">Network Dashboard</h1>
          <p class={cn('body text-text-secondary', spacing.margin.bottom.section)}>
            Click the help button below to learn more about this feature.
          </p>
          <button
            type="button"
            onClick={() => setIsOpen(true)}
            class={cn(
              button.size.md,
              'bg-surface-raised border border-surface-border hover:bg-surface-hover transition-colors text-text-primary',
              radius.lg,
            )}
          >
            Show Help
          </button>
        </div>

        <HelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} title="Network Dashboard">
          <p class={cn('body text-text-secondary', spacing.margin.bottom.content)}>
            The Network Dashboard provides a comprehensive overview of your network's health and
            performance.
          </p>
          <div class={spacing.stack.default}>
            <div class={cn('flex items-start', spacing.gap.default)}>
              <span class="inline-flex items-center justify-center w-6 h-6 rounded-full bg-brand-primary text-text-inverse text-sm font-semibold">
                1
              </span>
              <div class="flex-1">
                <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.tight)}>
                  Monitor Status
                </h4>
                <p class="body-small text-text-secondary">
                  View real-time status of all network interfaces and connections.
                </p>
              </div>
            </div>
            <div class={cn('flex items-start', spacing.gap.default)}>
              <span class="inline-flex items-center justify-center w-6 h-6 rounded-full bg-brand-primary text-text-inverse text-sm font-semibold">
                2
              </span>
              <div class="flex-1">
                <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.tight)}>
                  Run Tests
                </h4>
                <p class="body-small text-text-secondary">
                  Execute speed tests, cable diagnostics, and network discovery scans.
                </p>
              </div>
            </div>
            <div class={cn('flex items-start', spacing.gap.default)}>
              <span class="inline-flex items-center justify-center w-6 h-6 rounded-full bg-brand-primary text-text-inverse text-sm font-semibold">
                3
              </span>
              <div class="flex-1">
                <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.tight)}>
                  Analyze Results
                </h4>
                <p class="body-small text-text-secondary">
                  Review detailed metrics and identify potential issues or bottlenecks.
                </p>
              </div>
            </div>
          </div>
        </HelpModal>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          'Interactive demo with button to trigger help modal, showing numbered step-by-step guidance.',
      },
    },
  },
};

/**
 * Simple help modal - closed state
 */
export const SimpleClosed: StoryObj<typeof HelpModal> = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);

    return (
      <div class={cn('min-h-screen bg-surface-base', spacing.pad.xl)}>
        <p class="body text-text-secondary">Modal is closed - nothing rendered</p>
        <HelpModal isOpen={isOpen} onClose={() => setIsOpen(false)} title="Test">
          <p>This will not be visible</p>
        </HelpModal>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: 'Help modal in closed state returns null - no DOM elements rendered.',
      },
    },
  },
};

/**
 * Backdrop click to close demonstration
 */
export const BackdropClose: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);
    const [clickCount, setClickCount] = useState(0);

    return (
      <div class="min-h-screen bg-surface-base">
        {!isOpen && (
          <div class={spacing.pad.xl}>
            <p class={cn('body text-text-secondary', spacing.margin.bottom.content)}>
              Modal was closed {clickCount} time(s)
            </p>
            <button
              type="button"
              onClick={() => setIsOpen(true)}
              class={cn(
                button.size.md,
                'bg-brand-primary text-text-inverse hover:bg-brand-primary/90',
                radius.lg,
              )}
            >
              Reopen Modal
            </button>
          </div>
        )}
        <ImprovedHelpModal
          isOpen={isOpen}
          onClose={() => {
            setIsOpen(false);
            setClickCount((c) => c + 1);
          }}
        />
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          'Click the backdrop (dark area outside the modal) or the X button to close. Counter tracks how many times closed.',
      },
    },
  },
};
