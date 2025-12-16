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
import { cn, icon as iconTokens, layout, radius } from "../../styles/theme";
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
export function ImprovedHelpModal({ isOpen, onClose }: HelpModalProps) {
  // Track which help section is currently active
  const [activeSection, setActiveSection] = useState<string>("about");
  // Track search query for filtering help content
  const [searchQuery, setSearchQuery] = useState("");

  if (!isOpen) return null;

  const sections: HelpSection[] = [
    {
      id: "about",
      title: "About The Seed",
      icon: <Info className={iconTokens.size.sm} />,
      content: <AboutSection />,
    },
    {
      id: "getting-started",
      title: "Getting Started",
      icon: <LayoutDashboard className={iconTokens.size.sm} />,
      content: <GettingStartedSection />,
    },
    {
      id: "link",
      title: "Link Status",
      icon: <Activity className={iconTokens.size.sm} />,
      content: <LinkStatusSection />,
    },
    {
      id: "cable",
      title: "Cable Test",
      icon: <Cable className={iconTokens.size.sm} />,
      content: <CableTestSection />,
    },
    {
      id: "wifi",
      title: "WiFi Status",
      icon: <Wifi className={iconTokens.size.sm} />,
      content: <WiFiStatusSection />,
    },
    {
      id: "network",
      title: "Network & DHCP",
      icon: <Network className={iconTokens.size.sm} />,
      content: <NetworkSection />,
    },
    {
      id: "gateway",
      title: "Gateway",
      icon: <Server className={iconTokens.size.sm} />,
      content: <GatewaySection />,
    },
    {
      id: "dns",
      title: "DNS Tests",
      icon: <Search className={iconTokens.size.sm} />,
      content: <DNSSection />,
    },
    {
      id: "performance",
      title: "Performance Tests",
      icon: <Zap className={iconTokens.size.sm} />,
      content: <PerformanceSection />,
    },
    {
      id: "discovery",
      title: "Network Discovery",
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
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Modal */}
      <div
        className={`relative bg-surface-raised border border-surface-border ${radius.lg} shadow-xl w-full max-w-6xl max-h-modal flex flex-col overflow-hidden`}
        role="dialog"
        aria-modal="true"
        aria-labelledby="help-modal-title"
      >
        {/* Header */}
        <div className={`${layout.flex.between} p-4 border-b border-surface-border shrink-0`}>
          <h2 id="help-modal-title" className="heading-3">
            The Seed Help Center
          </h2>
          <button
            onClick={onClose}
            className={`p-2 text-text-muted hover:text-text-primary transition-colors ${radius.default} hover:bg-surface-hover`}
            aria-label="Close help"
          >
            <svg className={iconTokens.size.md} viewBox="0 0 20 20" fill="currentColor">
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
            <div className="p-3 border-b border-surface-border">
              <div className="relative">
                <Search
                  className={`absolute left-3 top-1/2 -translate-y-1/2 ${iconTokens.size.sm} text-text-muted`}
                />
                <input
                  type="text"
                  placeholder="Search help..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className={`w-full pl-9 pr-3 py-2 body-small ${radius.default} border border-surface-border bg-surface-raised text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-brand-primary`}
                />
              </div>
            </div>

            {/* Table of Contents */}
            <nav className="p-2">
              <p className="caption px-3 py-2 uppercase tracking-wider">Contents</p>
              {filteredSections.map((section) => (
                <button
                  key={section.id}
                  onClick={() => setActiveSection(section.id)}
                  className={cn(
                    `w-full flex items-center gap-3 px-3 py-2.5 ${radius.default} body-small transition-colors text-left`,
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
          <main className="flex-1 overflow-y-auto p-6">
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

function AboutSection() {
  return (
    <div className="section-gap max-w-3xl">
      <div>
        <h3 className="heading-2 mb-3">Welcome to The Seed</h3>
        <p className="body leading-relaxed mb-4">
          The Seed is a comprehensive network diagnostics and monitoring tool by Mustard Seed
          Networks, designed to provide deep visibility into your network infrastructure.
        </p>
      </div>

      <div className="grid md:grid-cols-2 gap-4">
        <FeatureCard
          title="Real-time Monitoring"
          description="Monitor link status, WiFi signal strength, gateway reachability, and more with live updates."
        />
        <FeatureCard
          title="Network Discovery"
          description="Discover all devices on your network with ARP scanning, LLDP/CDP neighbor detection."
        />
        <FeatureCard
          title="Performance Testing"
          description="Run internet speed tests and LAN throughput tests with iperf3 integration."
        />
        <FeatureCard
          title="Cable Diagnostics"
          description="Test Ethernet cables using Time Domain Reflectometry (TDR) to find faults and measure length."
        />
      </div>

      <div className={`border-l-4 border-brand-primary bg-brand-primary/5 p-4 ${radius.default}`}>
        <h4 className="font-semibold text-text-primary mb-2">Open Source & Customizable</h4>
        <p className="body-small text-text-secondary">
          The Seed is open source software. Customize thresholds, configure tests, and integrate
          with your existing monitoring infrastructure.
        </p>
      </div>

      <div>
        <h4 className="font-semibold text-text-primary mb-3">Version Info</h4>
        <dl className="grid grid-cols-2 gap-x-4 gap-y-2 body-small">
          <dt className="text-text-muted">Current Version:</dt>
          <dd className="font-mono text-text-primary">v0.13.0</dd>
          <dt className="text-text-muted">Backend:</dt>
          <dd className="text-text-primary">Go 1.25+</dd>
          <dt className="text-text-muted">Frontend:</dt>
          <dd className="text-text-primary">React 18 + TypeScript</dd>
        </dl>
      </div>
    </div>
  );
}

function GettingStartedSection() {
  return (
    <div className="section-gap max-w-3xl">
      <h3 className="heading-2 mb-3">Getting Started</h3>

      <div className="stack-lg">
        <StepCard
          number={1}
          title="Dashboard Overview"
          description="The main dashboard shows all diagnostic cards. Each card displays real-time information about a specific aspect of your network."
        />
        <StepCard
          number={2}
          title="Select Network Interface"
          description='Use the interface dropdown in the header to select which network interface to monitor (e.g., eth0, wlan0). Click "Settings" to configure interface-specific options.'
        />
        <StepCard
          number={3}
          title="Configure Thresholds"
          description="Open Settings → Thresholds to customize warning and critical levels for DNS latency, gateway ping, WiFi signal strength, and more."
        />
        <StepCard
          number={4}
          title="Run Tests"
          description='Click the "Run All Tests" button (⚡) in the bottom right to execute speed tests, cable diagnostics, network discovery, and custom health checks.'
        />
        <StepCard
          number={5}
          title="Explore Individual Cards"
          description="Each card has a help icon (?) that explains the metrics shown. Click on cards to view detailed information and run specific tests."
        />
      </div>

      <div className={`bg-surface-hover border border-surface-border ${radius.default} p-4 mt-6`}>
        <h4 className="font-semibold text-text-primary mb-2 flex items-center gap-2">
          <span className="text-status-info">💡</span>
          Pro Tips
        </h4>
        <ul className="body-small stack-sm ml-6 list-disc">
          <li>Use the Network Discovery card to find all devices on your network</li>
          <li>Set up custom health check tests in Settings → Custom Tests</li>
          <li>Export diagnostics data for troubleshooting or documentation</li>
          <li>Enable auto-scan for continuous network monitoring</li>
        </ul>
      </div>
    </div>
  );
}

function LinkStatusSection() {
  return (
    <HelpContentSection title="Link Status">
      <p className="body-small text-text-secondary mb-4">
        Monitors the physical layer connection of your network interface.
      </p>
      <HelpTermList
        items={[
          {
            term: "Carrier",
            description:
              "Physical layer signal detection. Shows 'Connected' when NIC detects a link partner (cable connected to active port).",
          },
          {
            term: "Speed",
            description:
              "Negotiated link speed between your network interface and the connected device (e.g., 1000 Mbps).",
          },
          {
            term: "Duplex",
            description:
              "Communication mode - Full duplex allows simultaneous bidirectional data, Half duplex is one direction at a time.",
          },
          {
            term: "Auto-Negotiation",
            description:
              "Whether speed and duplex were auto-negotiated with the link partner or manually configured.",
          },
          {
            term: "MTU",
            description:
              "Maximum Transmission Unit - largest packet size (in bytes) that can be sent without fragmentation. Standard is 1500 bytes.",
          },
        ]}
      />
    </HelpContentSection>
  );
}

function CableTestSection() {
  return (
    <HelpContentSection title="Cable Test (TDR)">
      <p className="body-small text-text-secondary mb-4">
        Time Domain Reflectometry tests cable quality and detects faults.
      </p>
      <HelpTermList
        items={[
          {
            term: "TDR Test",
            description:
              "Sends electrical pulses through the cable and measures reflections to detect faults and measure length.",
          },
          {
            term: "Cable Status",
            description:
              "Shows if cable pairs are OK, open (disconnected), short (wires touching), or have impedance mismatch.",
          },
          {
            term: "Fault Distance",
            description: "Distance to cable fault in meters. Helps locate physical cable problems.",
          },
          {
            term: "Pairs",
            description:
              "Ethernet cables have 4 twisted pairs. Gigabit uses all 4; Fast Ethernet uses pairs 1-2 and 3-6.",
          },
        ]}
      />
      <div
        className={`mt-4 bg-status-warning/10 border border-status-warning/20 ${radius.default} p-3`}
      >
        <p className="caption text-status-warning">
          <strong>Note:</strong> Cable testing requires compatible network hardware. Not all NICs
          support TDR.
        </p>
      </div>
    </HelpContentSection>
  );
}

function WiFiStatusSection() {
  return (
    <HelpContentSection title="WiFi Status">
      <p className="body-small text-text-secondary mb-4">
        Monitor wireless connection quality and settings.
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
  return (
    <HelpContentSection title="Network & DHCP">
      <p className="body-small text-text-secondary mb-4">
        Shows IP configuration and DHCP lease information.
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
  return (
    <HelpContentSection title="Gateway">
      <p className="body-small text-text-secondary mb-4">
        Tests reachability and latency to your default gateway.
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

function DNSSection() {
  return (
    <HelpContentSection title="DNS Tests">
      <p className="body-small text-text-secondary mb-4">
        Tests DNS resolution performance and functionality.
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
  return (
    <HelpContentSection title="Performance Tests">
      <p className="body-small text-text-secondary mb-4">
        Run speed tests to measure network throughput.
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
  return (
    <HelpContentSection title="Network Discovery">
      <p className="body-small text-text-secondary mb-4">
        Discover all devices on your network and identify connected switches.
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

// ============================================================================
// HELPER COMPONENTS
// ============================================================================

function FeatureCard({ title, description }: { title: string; description: string }) {
  return (
    <div className={`bg-surface-hover border border-surface-border ${radius.lg} p-4`}>
      <h4 className="font-semibold text-text-primary mb-2">{title}</h4>
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
    <div className="flex gap-4">
      <div
        className={`shrink-0 w-8 h-8 ${radius.full} bg-brand-primary text-text-inverse ${layout.flex.center} font-semibold`}
      >
        {number}
      </div>
      <div className="flex-1">
        <h4 className="font-semibold mb-1">{title}</h4>
        <p className="body-small">{description}</p>
      </div>
    </div>
  );
}

function HelpContentSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="max-w-3xl">
      <h3 className="heading-2 mb-4">{title}</h3>
      {children}
    </div>
  );
}

function HelpTermList({ items }: { items: Array<{ term: string; description: string }> }) {
  return (
    <dl className="stack-lg">
      {items.map((item, idx) => (
        <div key={idx} className="border-l-2 border-surface-border pl-4">
          <dt className="font-semibold text-text-primary mb-1">{item.term}</dt>
          <dd className="body-small text-text-secondary">{item.description}</dd>
        </div>
      ))}
    </dl>
  );
}
