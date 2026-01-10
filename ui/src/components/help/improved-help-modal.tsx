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

import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn, icon as iconTokens, layout, modal, radius, spacing } from "../../styles/theme";
import {
  Activity,
  AlertTriangle,
  BookOpen,
  Cable,
  Heart,
  HeartPulse,
  Info,
  LayoutDashboard,
  Lightbulb,
  Monitor,
  Network,
  Search,
  Server,
  Shield,
  Signal,
  SlidersHorizontal,
  Wifi,
  Zap,
} from "../ui/icons";

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
export function ImprovedHelpModal({ isOpen, onClose, version = "dev" }: HelpModalProps) {
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
      content: <DnsSection />,
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
    {
      id: "healthChecks",
      title: t("sections.healthChecks"),
      icon: <Heart className={iconTokens.size.sm} />,
      content: <HealthChecksSection />,
    },
    {
      id: "security",
      title: t("sections.security"),
      icon: <Shield className={iconTokens.size.sm} />,
      content: <SecuritySection />,
    },
    {
      id: "troubleshooting",
      title: t("sections.troubleshooting"),
      icon: <AlertTriangle className={iconTokens.size.sm} />,
      content: <TroubleshootingSection />,
    },
    {
      id: "profiles",
      title: t("sections.profiles"),
      icon: <SlidersHorizontal className={iconTokens.size.sm} />,
      content: <ProfilesSection />,
    },
    {
      id: "wifiSurvey",
      title: t("sections.wifiSurvey"),
      icon: <Signal className={iconTokens.size.sm} />,
      content: <WiFiSurveySection />,
    },
    {
      id: "rtspChecks",
      title: t("sections.rtspChecks"),
      icon: <Monitor className={iconTokens.size.sm} />,
      content: <RtspChecksSection />,
    },
    {
      id: "dicomChecks",
      title: t("sections.dicomChecks"),
      icon: <HeartPulse className={iconTokens.size.sm} />,
      content: <DicomChecksSection />,
    },
    {
      id: "howTo",
      title: t("sections.howTo"),
      icon: <Lightbulb className={iconTokens.size.sm} />,
      content: <HowToSection />,
    },
    {
      id: "glossary",
      title: t("sections.glossary"),
      icon: <BookOpen className={iconTokens.size.sm} />,
      content: <GlossarySection />,
    },
  ];

  const filteredSections = sections.filter(
    (section) =>
      section.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      section.id.toLowerCase().includes(searchQuery.toLowerCase()),
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
          "flex flex-col overflow-hidden",
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
            "border-b border-surface-border shrink-0",
          )}
        >
          <h2 id="help-modal-title" className="heading-3">
            {t("modal.title")}
          </h2>
          <button
            type="button"
            onClick={onClose}
            className={cn(
              spacing.pad.xs,
              "text-text-muted hover:text-text-primary transition-colors",
              radius.default,
              "hover:bg-surface-hover",
            )}
            aria-label={t("modal.closeHelp")}
          >
            <svg
              className={iconTokens.size.md}
              viewBox="0 0 20 20"
              fill="currentColor"
              aria-hidden="true"
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
            <div className={cn(spacing.pad.sm, "border-b border-surface-border")}>
              <div className="relative">
                <Search
                  className={cn(
                    "absolute left-3 top-1/2 -translate-y-1/2",
                    iconTokens.size.sm,
                    "text-text-muted",
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
                    "border border-surface-border bg-surface-raised text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-brand-primary",
                  )}
                />
              </div>
            </div>

            {/* Table of Contents */}
            <nav className={cn(spacing.pad.xs, "stack-xs")}>
              <p className={cn("caption", spacing.chip.lg, "uppercase tracking-wider")}>
                {t("modal.contents")}
              </p>
              {filteredSections.map((section) => (
                <button
                  type="button"
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
                      : "text-text-secondary hover:bg-surface-hover hover:text-text-primary",
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
        <p className={cn("body leading-relaxed", spacing.margin.bottom.content)}>
          {t("content.about.description")}
        </p>
      </div>

      <div className={cn("grid md:grid-cols-2", spacing.gap.comfortable)}>
        <FeatureCard
          title={t("content.about.features.realTimeMonitoring.title")}
          description={t("content.about.features.realTimeMonitoring.description")}
        />
        <FeatureCard
          title={t("content.about.features.networkDiscovery.title")}
          description={t("content.about.features.networkDiscovery.description")}
        />
        <FeatureCard
          title={t("content.about.features.performanceTesting.title")}
          description={t("content.about.features.performanceTesting.description")}
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
          radius.default,
        )}
      >
        <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.inline)}>
          {t("content.about.licensing.title", "Commercial Software")}
        </h4>
        <p className="body-small text-text-secondary">
          {t(
            "content.about.licensing.description",
            "SEED is commercial software developed by Mustard Seed Networks. All rights reserved. Unauthorized distribution or modification is prohibited.",
          )}
        </p>
      </div>

      <div>
        <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.heading)}>
          {t("content.about.versionInfo.title")}
        </h4>
        <dl className="grid grid-cols-2 gap-x-4 gap-y-2 body-small">
          <dt className="text-text-muted">{t("content.about.versionInfo.currentVersion")}</dt>
          <dd className="font-mono text-text-primary">{version}</dd>
          <dt className="text-text-muted">{t("content.about.versionInfo.backend")}</dt>
          <dd className="text-text-primary">Go 1.25.5+</dd>
          <dt className="text-text-muted">{t("content.about.versionInfo.frontend")}</dt>
          <dd className="text-text-primary">React 19.2 + TypeScript</dd>
          <dt className="text-text-muted">{t("content.about.versionInfo.runtime", "Runtime")}</dt>
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
          description={t("content.gettingStarted.steps.exploreCards.description")}
        />
      </div>

      <div
        className={cn(
          "bg-surface-hover border border-surface-border",
          radius.default,
          spacing.pad.default,
          spacing.margin.top.section,
        )}
      >
        <h4
          className={cn(
            "font-semibold text-text-primary",
            spacing.margin.bottom.inline,
            "flex items-center",
            spacing.gap.compact,
          )}
        >
          <span className="text-status-info">💡</span>
          {t("content.gettingStarted.proTips.title")}
        </h4>
        <ul className={cn("body-small stack-sm", spacing.margin.left.spacious, "list-disc")}>
          {tips.map((tip) => (
            <li key={tip}>{tip}</li>
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
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
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
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
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
          spacing.pad.sm,
        )}
      >
        <p className="caption text-status-warning">
          <strong>{t("common:labels.note", "Note")}:</strong> {t("content.cableTest.note")}
        </p>
      </div>
    </HelpContentSection>
  );
}

function WiFiStatusSection() {
  const { t } = useTranslation("help");
  return (
    <HelpContentSection title={t("sections.wifi")}>
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
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
            description: "Basic Service Set Identifier - MAC address of the access point.",
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
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
        {t("content.networkDhcp.description")}
      </p>
      <HelpTermList
        items={[
          {
            term: "Lease Time",
            description: "Duration of current IP address assignment before renewal is needed.",
          },
          {
            term: "DHCP Server",
            description:
              "IP address of the DHCP server that issued the lease (usually your router).",
          },
          {
            term: "Gateway",
            description: "Default gateway assigned by DHCP for routing traffic off-subnet.",
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
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
        {t("content.gatewayHelp.description")}
      </p>
      <HelpTermList
        items={[
          {
            term: "IPv4 Gateway",
            description: "Default router for IPv4 traffic leaving your local network.",
          },
          {
            term: "IPv6 Gateway",
            description: "Default router for IPv6 traffic (may be link-local address).",
          },
          {
            term: "Reachability",
            description: "Whether the gateway responds to ICMP ping requests.",
          },
          {
            term: "Latency",
            description: "Round-trip time to gateway. Should be <1ms for local networks.",
          },
          {
            term: "Packet Loss",
            description: "Percentage of ping packets that didn't receive a response.",
          },
        ]}
      />
    </HelpContentSection>
  );
}

function DnsSection() {
  const { t } = useTranslation("help");
  return (
    <HelpContentSection title={t("sections.dns")}>
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
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
            description: "Time taken for the DNS query to complete. Good: <50ms for local DNS.",
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
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
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
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
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
            description: "Manufacturer identified from the MAC address OUI (first 3 bytes).",
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

function HealthChecksSection() {
  const { t } = useTranslation("help");
  const commonIssues = t("content.healthChecks.commonIssues", { returnObjects: true }) as {
    title: string;
    timeout: string;
    highLatency: string;
    packetLoss: string;
    connectionRefused: string;
  };
  return (
    <HelpContentSection title={t("sections.healthChecks")}>
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
        {t("content.healthChecks.description")}
      </p>
      <HelpTermList
        items={[
          {
            term: t("content.healthChecks.terms.pingTest.term"),
            description: t("content.healthChecks.terms.pingTest.description"),
          },
          {
            term: t("content.healthChecks.terms.tcpTest.term"),
            description: t("content.healthChecks.terms.tcpTest.description"),
          },
          {
            term: t("content.healthChecks.terms.httpTest.term"),
            description: t("content.healthChecks.terms.httpTest.description"),
          },
          {
            term: t("content.healthChecks.terms.customTargets.term"),
            description: t("content.healthChecks.terms.customTargets.description"),
          },
          {
            term: t("content.healthChecks.terms.thresholds.term"),
            description: t("content.healthChecks.terms.thresholds.description"),
          },
        ]}
      />
      <div
        className={cn(
          spacing.margin.top.content,
          "bg-status-info/10 border border-status-info/20",
          radius.default,
          spacing.pad.default,
        )}
      >
        <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.inline)}>
          {commonIssues.title}
        </h4>
        <ul
          className={cn(
            "body-small text-text-secondary stack-sm",
            spacing.margin.left.spacious,
            "list-disc",
          )}
        >
          <li>
            <strong>Timeout:</strong> {commonIssues.timeout}
          </li>
          <li>
            <strong>High Latency:</strong> {commonIssues.highLatency}
          </li>
          <li>
            <strong>Packet Loss:</strong> {commonIssues.packetLoss}
          </li>
          <li>
            <strong>Connection Refused:</strong> {commonIssues.connectionRefused}
          </li>
        </ul>
      </div>
    </HelpContentSection>
  );
}

function SecuritySection() {
  const { t } = useTranslation("help");
  const recovery = t("content.security.passwordRecovery", { returnObjects: true }) as {
    title: string;
    description: string;
    steps: string[];
    note: string;
  };
  const portDetails = t("content.security.portScanDetails", { returnObjects: true }) as {
    title: string;
    description: string;
    levels: Record<string, string>;
    commonPorts: Record<string, string>;
  };
  return (
    <HelpContentSection title={t("sections.security")}>
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
        {t("content.security.description")}
      </p>
      <HelpTermList
        items={[
          {
            term: t("content.security.terms.portScan.term"),
            description: t("content.security.terms.portScan.description"),
          },
          {
            term: t("content.security.terms.vulnScan.term"),
            description: t("content.security.terms.vulnScan.description"),
          },
          {
            term: t("content.security.terms.devicePosture.term"),
            description: t("content.security.terms.devicePosture.description"),
          },
          {
            term: t("content.security.terms.rogueDhcp.term"),
            description: t("content.security.terms.rogueDhcp.description"),
          },
        ]}
      />

      {/* Password Recovery Section */}
      <div
        className={cn(
          spacing.margin.top.section,
          "bg-status-warning/10 border border-status-warning/20",
          radius.default,
          spacing.pad.default,
        )}
      >
        <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.content)}>
          {recovery.title}
        </h4>
        <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
          {recovery.description}
        </p>
        <ol
          className={cn(
            "body-small text-text-secondary stack-sm",
            spacing.margin.left.spacious,
            "list-decimal",
          )}
        >
          {recovery.steps.map((step) => (
            <li
              key={step}
              className={
                step.startsWith("User mode:") || step.startsWith("System mode:")
                  ? "font-mono text-xs bg-surface-base px-2 py-1 rounded"
                  : ""
              }
            >
              {step}
            </li>
          ))}
        </ol>
        <p className={cn("caption text-status-warning", spacing.margin.top.content)}>
          <strong>Note:</strong> {recovery.note}
        </p>
      </div>

      {/* Port Scan Details */}
      <div className={cn(spacing.margin.top.section)}>
        <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.content)}>
          {portDetails.title}
        </h4>
        <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
          {portDetails.description}
        </p>
        <div className="grid grid-cols-2 gap-2 body-small">
          {Object.entries(portDetails.levels).map(([level, desc]) => (
            <div key={level} className={cn("border-l-2 border-surface-border", spacing.pad.sm)}>
              <dt className="font-semibold text-text-primary capitalize">{level}</dt>
              <dd className="text-text-secondary">{desc}</dd>
            </div>
          ))}
        </div>
      </div>

      {/* Common Ports Reference */}
      <div className={cn(spacing.margin.top.content)}>
        <h5 className={cn("font-semibold text-text-primary", spacing.margin.bottom.inline)}>
          Common Ports Reference
        </h5>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-2 body-small font-mono">
          {Object.entries(portDetails.commonPorts).map(([port, desc]) => (
            <div key={port} className="flex items-baseline gap-2">
              <span className="text-brand-primary font-bold">{port}</span>
              <span className="text-text-muted">{desc}</span>
            </div>
          ))}
        </div>
      </div>
    </HelpContentSection>
  );
}

function TroubleshootingSection() {
  const { t } = useTranslation("help");
  const categories = t("content.troubleshooting.categories", { returnObjects: true }) as Record<
    string,
    {
      title: string;
      [key: string]: unknown;
    }
  >;

  return (
    <HelpContentSection title={t("sections.troubleshooting")}>
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
        {t("content.troubleshooting.description")}
      </p>

      {/* Link Issues */}
      <TroubleshootingCategory
        title={categories.linkIssues.title}
        issues={[
          {
            symptom: (
              categories.linkIssues as {
                noCarrier: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).noCarrier.symptom,
            causes: (
              categories.linkIssues as {
                noCarrier: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).noCarrier.causes,
            solutions: (
              categories.linkIssues as {
                noCarrier: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).noCarrier.solutions,
          },
          {
            symptom: (
              categories.linkIssues as {
                slowSpeed: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowSpeed.symptom,
            causes: (
              categories.linkIssues as {
                slowSpeed: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowSpeed.causes,
            solutions: (
              categories.linkIssues as {
                slowSpeed: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowSpeed.solutions,
          },
        ]}
      />

      {/* Cable Issues */}
      <TroubleshootingCategory
        title={categories.cableIssues.title}
        issues={[
          {
            symptom: (
              categories.cableIssues as {
                open: { symptom: string; meaning: string; solutions: string[] };
              }
            ).open.symptom,
            causes: [
              (
                categories.cableIssues as {
                  open: { symptom: string; meaning: string; solutions: string[] };
                }
              ).open.meaning,
            ],
            solutions: (
              categories.cableIssues as {
                open: { symptom: string; meaning: string; solutions: string[] };
              }
            ).open.solutions,
          },
          {
            symptom: (
              categories.cableIssues as {
                short: { symptom: string; meaning: string; solutions: string[] };
              }
            ).short.symptom,
            causes: [
              (
                categories.cableIssues as {
                  short: { symptom: string; meaning: string; solutions: string[] };
                }
              ).short.meaning,
            ],
            solutions: (
              categories.cableIssues as {
                short: { symptom: string; meaning: string; solutions: string[] };
              }
            ).short.solutions,
          },
        ]}
      />

      {/* DNS Issues */}
      <TroubleshootingCategory
        title={categories.dnsIssues.title}
        issues={[
          {
            symptom: (
              categories.dnsIssues as {
                slowResolution: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowResolution.symptom,
            causes: (
              categories.dnsIssues as {
                slowResolution: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowResolution.causes,
            solutions: (
              categories.dnsIssues as {
                slowResolution: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowResolution.solutions,
          },
        ]}
      />

      {/* Gateway Issues */}
      <TroubleshootingCategory
        title={categories.gatewayIssues.title}
        issues={[
          {
            symptom: (
              categories.gatewayIssues as {
                unreachable: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).unreachable.symptom,
            causes: (
              categories.gatewayIssues as {
                unreachable: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).unreachable.causes,
            solutions: (
              categories.gatewayIssues as {
                unreachable: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).unreachable.solutions,
          },
          {
            symptom: (
              categories.gatewayIssues as {
                highLatency: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).highLatency.symptom,
            causes: (
              categories.gatewayIssues as {
                highLatency: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).highLatency.causes,
            solutions: (
              categories.gatewayIssues as {
                highLatency: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).highLatency.solutions,
          },
        ]}
      />

      {/* Performance Issues */}
      <TroubleshootingCategory
        title={categories.performanceIssues.title}
        issues={[
          {
            symptom: (
              categories.performanceIssues as {
                slowInternet: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowInternet.symptom,
            causes: (
              categories.performanceIssues as {
                slowInternet: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowInternet.causes,
            solutions: (
              categories.performanceIssues as {
                slowInternet: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowInternet.solutions,
          },
          {
            symptom: (
              categories.performanceIssues as {
                slowLan: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowLan.symptom,
            causes: (
              categories.performanceIssues as {
                slowLan: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowLan.causes,
            solutions: (
              categories.performanceIssues as {
                slowLan: { symptom: string; causes: string[]; solutions: string[] };
              }
            ).slowLan.solutions,
          },
        ]}
      />
    </HelpContentSection>
  );
}

interface TroubleshootingIssue {
  symptom: string;
  causes: string[];
  solutions: string[];
}

function TroubleshootingCategory({
  title,
  issues,
}: {
  title: string;
  issues: TroubleshootingIssue[];
}) {
  return (
    <div className={cn(spacing.margin.top.section)}>
      <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.content)}>
        {title}
      </h4>
      <div className="stack-lg">
        {issues.map((issue) => (
          <div
            key={issue.symptom}
            className={cn("border border-surface-border", radius.default, spacing.pad.default)}
          >
            <h5 className={cn("font-semibold text-status-warning", spacing.margin.bottom.inline)}>
              {issue.symptom}
            </h5>
            <div className="grid md:grid-cols-2 gap-4 body-small">
              <div>
                <p className="font-semibold text-text-primary mb-1">Possible Causes:</p>
                <ul
                  className={cn(
                    "text-text-secondary",
                    spacing.margin.left.comfortable,
                    "list-disc",
                  )}
                >
                  {issue.causes.map((cause) => (
                    <li key={cause}>{cause}</li>
                  ))}
                </ul>
              </div>
              <div>
                <p className="font-semibold text-text-primary mb-1">Solutions:</p>
                <ul
                  className={cn(
                    "text-text-secondary",
                    spacing.margin.left.comfortable,
                    "list-disc",
                  )}
                >
                  {issue.solutions.map((solution) => (
                    <li key={solution}>{solution}</li>
                  ))}
                </ul>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// ============================================================================
// HELPER COMPONENTS
// ============================================================================

function FeatureCard({ title, description }: { title: string; description: string }) {
  return (
    <div
      className={cn(
        "bg-surface-hover border border-surface-border",
        radius.lg,
        spacing.pad.default,
      )}
    >
      <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.inline)}>
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
          "font-semibold",
        )}
      >
        {number}
      </div>
      <div className="flex-1">
        <h4 className={cn("font-semibold", spacing.margin.bottom.inline)}>{title}</h4>
        <p className="body-small">{description}</p>
      </div>
    </div>
  );
}

function HelpContentSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="max-w-3xl">
      <h3 className={cn("heading-2", spacing.margin.bottom.content)}>{title}</h3>
      {children}
    </div>
  );
}

function HelpTermList({ items }: { items: Array<{ term: string; description: string }> }) {
  return (
    <dl className="stack-lg">
      {items.map((item) => (
        <div
          key={item.term}
          className={cn("border-l-2 border-surface-border", spacing.pad.default)}
        >
          <dt className={cn("font-semibold text-text-primary", spacing.margin.bottom.inline)}>
            {item.term}
          </dt>
          <dd className="body-small text-text-secondary">{item.description}</dd>
        </div>
      ))}
    </dl>
  );
}

// ============================================================================
// NEW FEATURE SECTIONS
// ============================================================================

function ProfilesSection() {
  const { t } = useTranslation("help");
  const capabilities = t("content.profiles.capabilities", { returnObjects: true }) as string[];
  const useCases = t("content.profiles.useCases.items", { returnObjects: true }) as Array<{
    name: string;
    description: string;
  }>;

  return (
    <HelpContentSection title={t("sections.profiles")}>
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
        {t("content.profiles.description")}
      </p>

      <div className={spacing.margin.bottom.section}>
        <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.content)}>
          {t("content.profiles.overview.title")}
        </h4>
        <p className="body-small text-text-secondary">{t("content.profiles.overview.content")}</p>
      </div>

      <div className={spacing.margin.bottom.section}>
        <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.content)}>
          {t("content.profiles.capabilities_title", "Profile Capabilities")}
        </h4>
        <ul
          className={cn(
            "body-small text-text-secondary stack-sm",
            spacing.margin.left.spacious,
            "list-disc",
          )}
        >
          {capabilities?.map((cap) => <li key={cap}>{cap}</li>)}
        </ul>
      </div>

      <div>
        <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.content)}>
          {t("content.profiles.useCases.title")}
        </h4>
        <div className="stack-lg">
          {useCases?.map((useCase) => (
            <div
              key={useCase.name}
              className={cn("border-l-2 border-brand-primary", spacing.pad.default)}
            >
              <dt className={cn("font-semibold text-text-primary", spacing.margin.bottom.inline)}>
                {useCase.name}
              </dt>
              <dd className="body-small text-text-secondary">{useCase.description}</dd>
            </div>
          ))}
        </div>
      </div>
    </HelpContentSection>
  );
}

function WiFiSurveySection() {
  const { t } = useTranslation("help");
  const visualizations = t("content.wifiSurvey.visualizations", { returnObjects: true }) as Array<{
    type: string;
    description: string;
  }>;
  const bestPractices = t("content.wifiSurvey.bestPractices.items", {
    returnObjects: true,
  }) as string[];

  return (
    <HelpContentSection title={t("sections.wifiSurvey")}>
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
        {t("content.wifiSurvey.description")}
      </p>

      <HelpTermList
        items={[
          {
            term: t("content.wifiSurvey.terms.floorPlan.term"),
            description: t("content.wifiSurvey.terms.floorPlan.description"),
          },
          {
            term: t("content.wifiSurvey.terms.heatmap.term"),
            description: t("content.wifiSurvey.terms.heatmap.description"),
          },
          {
            term: t("content.wifiSurvey.terms.surveyPoint.term"),
            description: t("content.wifiSurvey.terms.surveyPoint.description"),
          },
          {
            term: t("content.wifiSurvey.terms.dataRate.term"),
            description: t("content.wifiSurvey.terms.dataRate.description"),
          },
        ]}
      />

      <div className={spacing.margin.top.section}>
        <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.content)}>
          {t("content.wifiSurvey.visualizationsTitle", "Visualization Modes")}
        </h4>
        <div className="grid md:grid-cols-2 gap-4">
          {visualizations?.map((viz) => (
            <div
              key={viz.type}
              className={cn(
                "bg-surface-hover border border-surface-border",
                radius.default,
                spacing.pad.sm,
              )}
            >
              <h5 className="font-semibold text-text-primary">{viz.type}</h5>
              <p className="body-small text-text-secondary">{viz.description}</p>
            </div>
          ))}
        </div>
      </div>

      <div
        className={cn(
          spacing.margin.top.section,
          "bg-status-info/10 border border-status-info/20",
          radius.default,
          spacing.pad.default,
        )}
      >
        <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.inline)}>
          {t("content.wifiSurvey.bestPractices.title")}
        </h4>
        <ul
          className={cn(
            "body-small text-text-secondary stack-sm",
            spacing.margin.left.spacious,
            "list-disc",
          )}
        >
          {bestPractices?.map((practice) => <li key={practice}>{practice}</li>)}
        </ul>
      </div>
    </HelpContentSection>
  );
}

function RtspChecksSection() {
  const { t } = useTranslation("help");
  const configuration = t("content.rtspChecks.configuration", { returnObjects: true }) as Array<{
    field: string;
    description: string;
  }>;

  return (
    <HelpContentSection title={t("sections.rtspChecks")}>
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
        {t("content.rtspChecks.description")}
      </p>

      <HelpTermList
        items={[
          {
            term: t("content.rtspChecks.terms.rtsp.term"),
            description: t("content.rtspChecks.terms.rtsp.description"),
          },
          {
            term: t("content.rtspChecks.terms.options.term"),
            description: t("content.rtspChecks.terms.options.description"),
          },
          {
            term: t("content.rtspChecks.terms.describe.term"),
            description: t("content.rtspChecks.terms.describe.description"),
          },
          {
            term: t("content.rtspChecks.terms.authentication.term"),
            description: t("content.rtspChecks.terms.authentication.description"),
          },
        ]}
      />

      <div className={spacing.margin.top.section}>
        <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.content)}>
          {t("content.rtspChecks.configurationTitle", "Configuration Options")}
        </h4>
        <div className="stack-sm">
          {configuration?.map((config) => (
            <div
              key={config.field}
              className={cn("border-l-2 border-surface-border", spacing.pad.sm)}
            >
              <span className="font-mono text-brand-primary">{config.field}</span>
              <span className="body-small text-text-secondary ml-2">{config.description}</span>
            </div>
          ))}
        </div>
      </div>
    </HelpContentSection>
  );
}

function DicomChecksSection() {
  const { t } = useTranslation("help");
  const configuration = t("content.dicomChecks.configuration", { returnObjects: true }) as Array<{
    field: string;
    description: string;
  }>;
  const commonIssues = t("content.dicomChecks.commonIssues", { returnObjects: true }) as Array<{
    issue: string;
    solution: string;
  }>;

  return (
    <HelpContentSection title={t("sections.dicomChecks")}>
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
        {t("content.dicomChecks.description")}
      </p>

      <HelpTermList
        items={[
          {
            term: t("content.dicomChecks.terms.dicom.term"),
            description: t("content.dicomChecks.terms.dicom.description"),
          },
          {
            term: t("content.dicomChecks.terms.cEcho.term"),
            description: t("content.dicomChecks.terms.cEcho.description"),
          },
          {
            term: t("content.dicomChecks.terms.aeTitle.term"),
            description: t("content.dicomChecks.terms.aeTitle.description"),
          },
          {
            term: t("content.dicomChecks.terms.scp.term"),
            description: t("content.dicomChecks.terms.scp.description"),
          },
          {
            term: t("content.dicomChecks.terms.scu.term"),
            description: t("content.dicomChecks.terms.scu.description"),
          },
        ]}
      />

      <div className={spacing.margin.top.section}>
        <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.content)}>
          {t("content.dicomChecks.configurationTitle", "Configuration")}
        </h4>
        <div className="stack-sm">
          {configuration?.map((config) => (
            <div
              key={config.field}
              className={cn("border-l-2 border-surface-border", spacing.pad.sm)}
            >
              <span className="font-mono text-brand-primary">{config.field}</span>
              <span className="body-small text-text-secondary ml-2">{config.description}</span>
            </div>
          ))}
        </div>
      </div>

      <div
        className={cn(
          spacing.margin.top.section,
          "bg-status-warning/10 border border-status-warning/20",
          radius.default,
          spacing.pad.default,
        )}
      >
        <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.content)}>
          {t("content.dicomChecks.commonIssuesTitle", "Common Issues")}
        </h4>
        <div className="stack-lg">
          {commonIssues?.map((item) => (
            <div key={item.issue}>
              <p className="font-semibold text-status-warning">{item.issue}</p>
              <p className="body-small text-text-secondary">{item.solution}</p>
            </div>
          ))}
        </div>
      </div>
    </HelpContentSection>
  );
}

function HowToSection() {
  const { t } = useTranslation("help");
  const guides = t("content.howTo.guides", { returnObjects: true }) as Record<
    string,
    {
      title: string;
      description: string;
      steps: string[];
    }
  >;

  return (
    <HelpContentSection title={t("sections.howTo")}>
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
        {t("content.howTo.description")}
      </p>

      <div className="stack-xl">
        {guides &&
          Object.entries(guides).map(([key, guide]) => (
            <div
              key={key}
              className={cn("border border-surface-border", radius.lg, spacing.pad.default)}
            >
              <h4 className={cn("font-semibold text-text-primary", spacing.margin.bottom.inline)}>
                {guide.title}
              </h4>
              <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
                {guide.description}
              </p>
              <ol
                className={cn(
                  "body-small text-text-secondary stack-sm",
                  spacing.margin.left.spacious,
                  "list-decimal",
                )}
              >
                {guide.steps.map((step, index) => (
                  <li key={`${key}-step-${index}`}>{step}</li>
                ))}
              </ol>
            </div>
          ))}
      </div>
    </HelpContentSection>
  );
}

function GlossarySection() {
  const { t } = useTranslation("glossary");
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string>("all");

  const categories = t("categories", { returnObjects: true }) as Record<string, string>;
  const terms = t("terms", { returnObjects: true }) as Record<
    string,
    {
      term: string;
      fullName: string;
      definition: string;
      category: string;
    }
  >;

  const filteredTerms = terms
    ? Object.entries(terms).filter(([, termData]) => {
        const matchesSearch =
          searchTerm === "" ||
          termData.term.toLowerCase().includes(searchTerm.toLowerCase()) ||
          termData.fullName.toLowerCase().includes(searchTerm.toLowerCase()) ||
          termData.definition.toLowerCase().includes(searchTerm.toLowerCase());

        const matchesCategory =
          selectedCategory === "all" || termData.category === selectedCategory;

        return matchesSearch && matchesCategory;
      })
    : [];

  return (
    <div className="max-w-3xl">
      <h3 className={cn("heading-2", spacing.margin.bottom.content)}>{t("title")}</h3>
      <p className={cn("body-small text-text-secondary", spacing.margin.bottom.content)}>
        {t("description")}
      </p>

      {/* Search and Filter */}
      <div className={cn("flex flex-wrap gap-4", spacing.margin.bottom.section)}>
        <div className="flex-1 min-w-[200px]">
          <div className="relative">
            <Search
              className={cn(
                "absolute left-3 top-1/2 -translate-y-1/2",
                "w-4 h-4 text-text-muted",
              )}
            />
            <input
              type="text"
              placeholder="Search terms..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className={cn(
                "w-full pl-9 pr-3 py-2",
                "body-small",
                radius.default,
                "border border-surface-border bg-surface-raised text-text-primary placeholder-text-muted",
                "focus:outline-none focus:ring-2 focus:ring-brand-primary",
              )}
            />
          </div>
        </div>
        <select
          value={selectedCategory}
          onChange={(e) => setSelectedCategory(e.target.value)}
          className={cn(
            "px-3 py-2",
            "body-small",
            radius.default,
            "border border-surface-border bg-surface-raised text-text-primary",
            "focus:outline-none focus:ring-2 focus:ring-brand-primary",
          )}
        >
          <option value="all">All Categories</option>
          {categories &&
            Object.entries(categories).map(([key, label]) => (
              <option key={key} value={key}>
                {label}
              </option>
            ))}
        </select>
      </div>

      {/* Terms List */}
      <div className="stack-lg">
        {filteredTerms.map(([key, termData]) => (
          <div
            key={key}
            className={cn(
              "border border-surface-border",
              radius.default,
              spacing.pad.default,
              "hover:border-brand-primary/50 transition-colors",
            )}
          >
            <div className="flex items-start justify-between gap-4">
              <div className="flex-1">
                <div className="flex items-baseline gap-2 mb-1">
                  <span className="font-bold text-brand-primary">{termData.term}</span>
                  <span className="body-small text-text-muted">({termData.fullName})</span>
                </div>
                <p className="body-small text-text-secondary">{termData.definition}</p>
              </div>
              <span
                className={cn(
                  "px-2 py-0.5 text-xs font-medium",
                  radius.default,
                  "bg-surface-hover text-text-muted capitalize",
                )}
              >
                {categories?.[termData.category] || termData.category}
              </span>
            </div>
          </div>
        ))}

        {filteredTerms.length === 0 && (
          <div className="text-center py-8 text-text-muted">
            No terms found matching your search.
          </div>
        )}
      </div>
    </div>
  );
}
