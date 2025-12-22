/**
 * ImprovedHelpModal Component (~681 lines)
 *
 * Purpose: Comprehensive application help modal providing user guidance across multiple topics.
 * Features tabbed navigation, search functionality, and rich content for all major features.
 *
 * Key Features:
 * - Multi-section help: About, Network Discovery, WiFi, Cable/Link, Performance, etc.
 * - Search functionality: Filter help content by keyword
 * - Icon-based navigation: Visual section selector with icons
 * - Rich content: Markdown-like formatting for help text
 * - Modal overlay: Centered help dialog with close button
 * - Responsive design: Adapts to different screen sizes
 * - Keyboard support: ESC key closes modal
 * - Scrollable sections: Long help content in scrollable containers
 *
 * Usage:
 * ```typescript
 * <ImprovedHelpModal isOpen={showHelp} onClose={() => setShowHelp(false)} />
 * ```
 *
 * Dependencies: Icons, theme utilities, useState for tab/search state management
 * State: activeSection (current tab), searchQuery (help search text)
 */

import { ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  cn,
  icon as iconTokens,
  layout,
  radius,
  modal,
  spacing,
} from "../../styles/theme";
import {
  Activity,
  Wifi,
  Cable,
  Network,
  Server,
  Search,
  Info,
  LayoutDashboard,
  Zap,
} from "../ui/Icons";

interface HelpModalProps {
  isOpen: boolean;
  onClose: () => void;
  /** Application version from backend */
  version?: string;
}

interface HelpSection {
  id: string;
  title: string;
  icon: ReactNode;
  content: ReactNode;
}

/**
 * ImprovedHelpModal Component
 * Renders a modal dialog with tabbed help content and search functionality
 */
export function ImprovedHelpModal({
  isOpen,
  onClose,
  version = "dev",
}: HelpModalProps) {
  const { t } = useTranslation("help");
  // Track which help section is currently active
  const [activeSection, setActiveSection] = useState<string>("about");
  // Track search query for filtering help content
  const [searchQuery, setSearchQuery] = useState("");

  if (!isOpen) return null;

  const sections: HelpSection[] = [
    {
      id: "about",
      title: t("sections.about"),
      icon: <Info className={iconTokens.size.sm} />,
      content: <AboutSection version={version} />,
    },
    {
      id: "getting-started",
      title: t("sections.gettingStarted"),
      icon: <LayoutDashboard className={iconTokens.size.sm} />,
      content: <GettingStartedSection />,
    },
    {
      id: "link",
      title: t("sections.link"),
      icon: <Activity className={iconTokens.size.sm} />,
      content: <LinkStatusSection />,
    },
    {
      id: "cable",
      title: t("sections.cable"),
      icon: <Cable className={iconTokens.size.sm} />,
      content: <CableTestSection />,
    },
    {
      id: "wifi",
      title: t("sections.wifi"),
      icon: <Wifi className={iconTokens.size.sm} />,
      content: <WiFiStatusSection />,
    },
    {
      id: "network",
      title: t("sections.network"),
      icon: <Network className={iconTokens.size.sm} />,
      content: <NetworkSection />,
    },
    {
      id: "gateway",
      title: t("sections.gateway"),
      icon: <Server className={iconTokens.size.sm} />,
      content: <GatewaySection />,
    },
    {
      id: "dns",
      title: t("sections.dns"),
      icon: <Search className={iconTokens.size.sm} />,
      content: <DNSSection />,
    },
    {
      id: "performance",
      title: t("sections.performance"),
      icon: <Zap className={iconTokens.size.sm} />,
      content: <PerformanceSection />,
    },
    {
      id: "discovery",
      title: t("sections.discovery"),
      icon: <Search className={iconTokens.size.sm} />,
      content: <DiscoverySection />,
    },
  ];

  const filteredSections = sections.filter(
    (section) =>
      section.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      section.id.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const currentSection = sections.find((s) => s.id === activeSection);

  return (
    <div className={modal.overlay}>
      {/* Backdrop */}
      <div className={modal.backdrop} onClick={onClose} aria-hidden="true" />

      {/* Modal */}
      <div
        className={cn(
          "relative",
          modal.content,
          modal.size.xl,
          radius.lg,
          "flex flex-col overflow-hidden"
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby="help-modal-title"
      >
        {/* Header */}
        <div
          className={cn(
            layout.flex.between,
            spacing.pad.default,
            "border-b border-surface-border shrink-0"
          )}
        >
          <h2 id="help-modal-title" className="heading-3">
            {t("modal.title")}
          </h2>
          <button
            onClick={onClose}
            className={cn(
              spacing.pad.xs,
              "text-text-muted hover:text-text-primary transition-colors",
              radius.default,
              "hover:bg-surface-hover"
            )}
            aria-label={t("modal.closeHelp")}
          >
            <svg
              className={iconTokens.size.md}
              viewBox="0 0 20 20"
              fill="currentColor"
            >
              <path
                fillRule="evenodd"
                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        </div>

        {/* Content area with sidebar */}
        <div className="flex flex-1 overflow-hidden">
          {/* Sidebar / TOC */}
          <aside className="w-64 border-r border-surface-border bg-surface-base overflow-y-auto shrink-0">
            {/* Search */}
            <div
              className={cn(spacing.pad.sm, "border-b border-surface-border")}
            >
              <div className="relative">
                <Search
                  className={cn(
                    "absolute left-3 top-1/2 -translate-y-1/2",
                    iconTokens.size.sm,
                    "text-text-muted"
                  )}
                />
                <input
                  type="text"
                  placeholder={t("modal.searchPlaceholder")}
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className={cn(
                    "w-full pl-9",
                    spacing.chip.lg,
                    "body-small",
                    radius.default,
                    "border border-surface-border bg-surface-raised text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-brand-primary"
                  )}
                />
              </div>
            </div>

            {/* Table of Contents */}
            <nav className={cn(spacing.pad.xs, "stack-xs")}>
              <p
                className={cn(
                  "caption",
                  spacing.chip.lg,
                  "uppercase tracking-wider"
                )}
              >
                {t("modal.contents")}
              </p>
              {filteredSections.map((section) => (
                <button
                  key={section.id}
                  onClick={() => setActiveSection(section.id)}
                  className={cn(
                    "w-full flex items-center",
                    spacing.gap.default,
                    spacing.tab,
                    radius.default,
                    "body-small transition-colors text-left",
                    activeSection === section.id
                      ? "bg-brand-primary/10 text-brand-primary font-medium"
                      : "text-text-secondary hover:bg-surface-hover hover:text-text-primary"
                  )}
                >
                  {section.icon}
                  <span>{section.title}</span>
                </button>
              ))}
            </nav>
          </aside>

          {/* Main content */}
          <main className={cn("flex-1 overflow-y-auto", spacing.pad.lg)}>
            {currentSection && <div>{currentSection.content}</div>}
          </main>
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// CONTENT SECTIONS
// ============================================================================

interface AboutSectionProps {
  version: string;
}

function AboutSection({ version }: AboutSectionProps) {
  const { t } = useTranslation("help");
  return (
    <div className="section-gap max-w-3xl">
      <div>
        <h3 className={cn("heading-2", spacing.margin.bottom.heading)}>
          {t("content.about.welcome")}
        </h3>
        <p
          className={cn("body leading-relaxed", spacing.margin.bottom.content)}
        >
          {t("content.about.description")}
        </p>
      </div>

      <div className={cn("grid md:grid-cols-2", spacing.gap.comfortable)}>
        <FeatureCard
          title={t("content.about.features.realTimeMonitoring.title")}
          description={t(
            "content.about.features.realTimeMonitoring.description"
          )}
        />
        <FeatureCard
          title={t("content.about.features.networkDiscovery.title")}
          description={t("content.about.features.networkDiscovery.description")}
        />
        <FeatureCard
          title={t("content.about.features.performanceTesting.title")}
          description={t(
            "content.about.features.performanceTesting.description"
          )}
        />
        <FeatureCard
          title={t("content.about.features.cableDiagnostics.title")}
          description={t("content.about.features.cableDiagnostics.description")}
        />
      </div>

      <div
        className={cn(
          "border-l-4 border-brand-primary bg-brand-primary/5",
          spacing.pad.default,
          radius.default
        )}
      >
        <h4
          className={cn(
            "font-semibold text-text-primary",
            spacing.margin.bottom.inline
          )}
        >
          {t("content.about.licensing.title", "Commercial Software")}
        </h4>
        <p className="body-small text-text-secondary">
          {t(
            "content.about.licensing.description",
            "SEED is commercial software developed by Mustard Seed Networks. All rights reserved. Unauthorized distribution or modification is prohibited."
          )}
        </p>
      </div>

      <div>
        <h4
          className={cn(
            "font-semibold text-text-primary",
            spacing.margin.bottom.heading
          )}
        >
          {t("content.about.versionInfo.title")}
        </h4>
        <dl className="grid grid-cols-2 gap-x-4 gap-y-2 body-small">
          <dt className="text-text-muted">
            {t("content.about.versionInfo.currentVersion")}
          </dt>
          <dd className="font-mono text-text-primary">{version}</dd>
          <dt className="text-text-muted">
            {t("content.about.versionInfo.backend")}
          </dt>
          <dd className="text-text-primary">Go 1.25.5+</dd>
          <dt className="text-text-muted">
            {t("content.about.versionInfo.frontend")}
          </dt>
          <dd className="text-text-primary">React 19.2 + TypeScript</dd>
          <dt className="text-text-muted">
            {t("content.about.versionInfo.runtime", "Runtime")}
          </dt>
          <dd className="text-text-primary">Node.js 25.2</dd>
        </dl>
      </div>
    </div>
  );
}

function GettingStartedSection() {
  const { t } = useTranslation("help");
  const tips = t("content.gettingStarted.proTips.tips", {
    returnObjects: true,
  }) as string[];
  return (
    <div className="section-gap max-w-3xl">
      <h3 className={cn("heading-2", spacing.margin.bottom.heading)}>
        {t("content.gettingStarted.title")}
      </h3>

      <div className="stack-lg">
        <StepCard
          number={1}
          title={t("content.gettingStarted.steps.dashboard.title")}
          description={t("content.gettingStarted.steps.dashboard.description")}
        />
        <StepCard
          number={2}
          title={t("content.gettingStarted.steps.interface.title")}
          description={t("content.gettingStarted.steps.interface.description")}
        />
        <StepCard
          number={3}
          title={t("content.gettingStarted.steps.thresholds.title")}
          description={t("content.gettingStarted.steps.thresholds.description")}
        />
        <StepCard
          number={4}
          title={t("content.gettingStarted.steps.runTests.title")}
          description={t("content.gettingStarted.steps.runTests.description")}
        />
        <StepCard
          number={5}
          title={t("content.gettingStarted.steps.exploreCards.title")}
          description={t(
            "content.gettingStarted.steps.exploreCards.description"
          )}
        />
      </div>

      <div
        className={cn(
          "bg-surface-hover border border-surface-border",
          radius.default,
          spacing.pad.default,
          spacing.margin.top.section
        )}
      >
        <h4
          className={cn(
            "font-semibold text-text-primary",
            spacing.margin.bottom.inline,
            "flex items-center",
            spacing.gap.compact
          )}
        >
          <span className="text-status-info">💡</span>
          {t("content.gettingStarted.proTips.title")}
        </h4>
        <ul
          className={cn(
            "body-small stack-sm",
            spacing.margin.left.spacious,
            "list-disc"
          )}
        >
          {tips.map((tip, index) => (
            <li key={index}>{tip}</li>
          ))}
        </ul>
      </div>
    </div>
  );
}

function LinkStatusSection() {
  const { t } = useTranslation("help");
  return (
    <HelpContentSection title={t("sections.link")}>
      <p
        className={cn(
          "body-small text-text-secondary",
          spacing.margin.bottom.content
        )}
      >
        {t("content.linkStatus.description")}
      </p>
      <HelpTermList
        items={[
          {
            term: t("content.linkStatus.terms.carrier.term"),
            description: t("content.linkStatus.terms.carrier.description"),
          },
          {
            term: t("content.linkStatus.terms.speed.term"),
            description: t("content.linkStatus.terms.speed.description"),
          },
          {
            term: t("content.linkStatus.terms.duplex.term"),
            description: t("content.linkStatus.terms.duplex.description"),
          },
          {
            term: t("content.linkStatus.terms.autoNeg.term"),
            description: t("content.linkStatus.terms.autoNeg.description"),
          },
          {
            term: t("content.linkStatus.terms.mtu.term"),
            description: t("content.linkStatus.terms.mtu.description"),
          },
        ]}
      />
    </HelpContentSection>
  );
}

function CableTestSection() {
  const { t } = useTranslation("help");
  return (
    <HelpContentSection title={t("sections.cable")}>
      <p
        className={cn(
          "body-small text-text-secondary",
          spacing.margin.bottom.content
        )}
      >
        {t("content.cableTest.description")}
      </p>
      <HelpTermList
        items={[
          {
            term: t("content.cableTest.terms.tdrTest.term"),
            description: t("content.cableTest.terms.tdrTest.description"),
          },
          {
            term: t("content.cableTest.terms.cableStatus.term"),
            description: t("content.cableTest.terms.cableStatus.description"),
          },
          {
            term: t("content.cableTest.terms.faultDistance.term"),
            description: t("content.cableTest.terms.faultDistance.description"),
          },
          {
            term: t("content.cableTest.terms.pairs.term"),
            description: t("content.cableTest.terms.pairs.description"),
          },
        ]}
      />
      <div
        className={cn(
          spacing.margin.top.content,
          "bg-status-warning/10 border border-status-warning/20",
          radius.default,
          spacing.pad.sm
        )}
      >
        <p className="caption text-status-warning">
          <strong>{t("common:labels.note", "Note")}:</strong>{" "}
          {t("content.cableTest.note")}
        </p>
      </div>
    </HelpContentSection>
  );
}

function WiFiStatusSection() {
  const { t } = useTranslation("help");
  return (
    <HelpContentSection title={t("sections.wifi")}>
      <p
        className={cn(
          "body-small text-text-secondary",
          spacing.margin.bottom.content
        )}
      >
        {t("content.wifiStatus.description")}
      </p>
      <HelpTermList
        items={[
          {
            term: "SSID",
            description:
              "Service Set Identifier - the name of the wireless network you're connected to.",
          },
          {
            term: "BSSID",
            description:
              "Basic Service Set Identifier - MAC address of the access point.",
          },
          {
            term: "Signal Strength",
            description:
              "Signal strength in dBm. -30 is excellent, -67 is good, -70 is fair, -80 is weak. Higher (less negative) is better.",
          },
          {
            term: "Channel",
            description:
              "WiFi channel number (1-14 for 2.4GHz, 36-165 for 5GHz). Overlapping channels cause interference.",
          },
          {
            term: "Security",
            description:
              "Encryption protocol protecting the connection (WPA2, WPA3, WEP, or Open).",
          },
          {
            term: "Frequency",
            description:
              "Radio band - 2.4GHz has better range, 5GHz has better speed and less interference.",
          },
        ]}
      />
    </HelpContentSection>
  );
}

function NetworkSection() {
  const { t } = useTranslation("help");
  return (
    <HelpContentSection title={t("sections.network")}>
      <p
        className={cn(
          "body-small text-text-secondary",
          spacing.margin.bottom.content
        )}
      >
        {t("content.networkDhcp.description")}
      </p>
      <HelpTermList
        items={[
          {
            term: "Lease Time",
            description:
              "Duration of current IP address assignment before renewal is needed.",
          },
          {
            term: "DHCP Server",
            description:
              "IP address of the DHCP server that issued the lease (usually your router).",
          },
          {
            term: "Gateway",
            description:
              "Default gateway assigned by DHCP for routing traffic off-subnet.",
          },
          {
            term: "DNS Servers",
            description: "DNS servers assigned by DHCP for name resolution.",
          },
          {
            term: "Subnet Mask",
            description: "Network mask defining the local subnet size.",
          },
        ]}
      />
    </HelpContentSection>
  );
}

function GatewaySection() {
  const { t } = useTranslation("help");
  return (
    <HelpContentSection title={t("sections.gateway")}>
      <p
        className={cn(
          "body-small text-text-secondary",
          spacing.margin.bottom.content
        )}
      >
        {t("content.gatewayHelp.description")}
      </p>
      <HelpTermList
        items={[
          {
            term: "IPv4 Gateway",
            description:
              "Default router for IPv4 traffic leaving your local network.",
          },
          {
            term: "IPv6 Gateway",
            description:
              "Default router for IPv6 traffic (may be link-local address).",
          },
          {
            term: "Reachability",
            description: "Whether the gateway responds to ICMP ping requests.",
          },
          {
            term: "Latency",
            description:
              "Round-trip time to gateway. Should be <1ms for local networks.",
          },
          {
            term: "Packet Loss",
            description:
              "Percentage of ping packets that didn't receive a response.",
          },
        ]}
      />
    </HelpContentSection>
  );
}

function DNSSection() {
  const { t } = useTranslation("help");
  return (
    <HelpContentSection title={t("sections.dns")}>
      <p
        className={cn(
          "body-small text-text-secondary",
          spacing.margin.bottom.content
        )}
      >
        {t("content.dnsTests.description")}
      </p>
      <HelpTermList
        items={[
          {
            term: "Forward Lookup",
            description: "Resolves hostname to IPv4 address (A record).",
          },
          {
            term: "Reverse Lookup",
            description: "Resolves IP address back to hostname (PTR record).",
          },
          {
            term: "IPv6 Lookup",
            description: "Resolves hostname to IPv6 address (AAAA record).",
          },
          {
            term: "Latency",
            description:
              "Time taken for the DNS query to complete. Good: <50ms for local DNS.",
          },
        ]}
      />
    </HelpContentSection>
  );
}

function PerformanceSection() {
  const { t } = useTranslation("help");
  return (
    <HelpContentSection title={t("sections.performance")}>
      <p
        className={cn(
          "body-small text-text-secondary",
          spacing.margin.bottom.content
        )}
      >
        {t("content.performanceTests.description")}
      </p>
      <HelpTermList
        items={[
          {
            term: "Internet Speed Test",
            description:
              "Tests download/upload speeds to public speedtest servers. Measures your connection to the internet.",
          },
          {
            term: "LAN Speed (iperf3)",
            description:
              "Tests throughput on your local network using iperf3 to a configured server.",
          },
          {
            term: "Download",
            description: "Maximum download speed achieved during the test.",
          },
          {
            term: "Upload",
            description: "Maximum upload speed achieved during the test.",
          },
          {
            term: "Latency",
            description: "Round-trip time (ping) to the test server.",
          },
          {
            term: "Jitter",
            description:
              "Variation in latency over time. Lower is better for real-time applications like VoIP and gaming.",
          },
        ]}
      />
    </HelpContentSection>
  );
}

function DiscoverySection() {
  const { t } = useTranslation("help");
  return (
    <HelpContentSection title={t("sections.discovery")}>
      <p
        className={cn(
          "body-small text-text-secondary",
          spacing.margin.bottom.content
        )}
      >
        {t("content.networkDiscovery.description")}
      </p>
      <HelpTermList
        items={[
          {
            term: "Network Scan",
            description:
              "Discovers active devices on your network subnet using ARP/ICMP ping sweeps.",
          },
          {
            term: "MAC Address",
            description:
              "Hardware address of the network interface - unique identifier for the device.",
          },
          {
            term: "Vendor",
            description:
              "Manufacturer identified from the MAC address OUI (first 3 bytes).",
          },
          {
            term: "Hostname",
            description: "DNS hostname if reverse lookup succeeded.",
          },
          {
            term: "LLDP/CDP",
            description:
              "Link Layer Discovery Protocol (standard) or Cisco Discovery Protocol - provides information about directly connected network switches.",
          },
        ]}
      />
    </HelpContentSection>
  );
}

// ============================================================================
// HELPER COMPONENTS
// ============================================================================

function FeatureCard({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div
      className={cn(
        "bg-surface-hover border border-surface-border",
        radius.lg,
        spacing.pad.default
      )}
    >
      <h4
        className={cn(
          "font-semibold text-text-primary",
          spacing.margin.bottom.inline
        )}
      >
        {title}
      </h4>
      <p className="body-small text-text-secondary">{description}</p>
    </div>
  );
}

function StepCard({
  number,
  title,
  description,
}: {
  number: number;
  title: string;
  description: string;
}) {
  return (
    <div className={cn("flex", spacing.gap.comfortable)}>
      <div
        className={cn(
          "shrink-0 w-8 h-8",
          radius.full,
          "bg-brand-primary text-text-inverse",
          layout.flex.center,
          "font-semibold"
        )}
      >
        {number}
      </div>
      <div className="flex-1">
        <h4 className={cn("font-semibold", spacing.margin.bottom.inline)}>
          {title}
        </h4>
        <p className="body-small">{description}</p>
      </div>
    </div>
  );
}

function HelpContentSection({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <div className="max-w-3xl">
      <h3 className={cn("heading-2", spacing.margin.bottom.content)}>
        {title}
      </h3>
      {children}
    </div>
  );
}

function HelpTermList({
  items,
}: {
  items: Array<{ term: string; description: string }>;
}) {
  return (
    <dl className="stack-lg">
      {items.map((item, idx) => (
        <div
          key={idx}
          className={cn(
            "border-l-2 border-surface-border",
            spacing.pad.default
          )}
        >
          <dt
            className={cn(
              "font-semibold text-text-primary",
              spacing.margin.bottom.inline
            )}
          >
            {item.term}
          </dt>
          <dd className="body-small text-text-secondary">{item.description}</dd>
        </div>
      ))}
    </dl>
  );
}
