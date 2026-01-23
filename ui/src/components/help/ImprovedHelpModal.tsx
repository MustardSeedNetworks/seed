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

import type React from 'react';
import { type ReactNode, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, icon as iconTokens, layout, modal, radius, spacing } from '../../styles/theme';
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
} from '../ui/icons';

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
  version = 'dev',
}: HelpModalProps): React.JSX.Element | null {
  const { t } = useTranslation('help');
  // Track which help section is currently active
  const [activeSection, setActiveSection] = useState<string>('about');
  // Track search query for filtering help content
  const [searchQuery, setSearchQuery] = useState('');

  if (!isOpen) {
    return null;
  }

  const sections: HelpSection[] = [
    {
      id: 'about',
      title: t('sections.about'),
      icon: <Info class={iconTokens.size.sm} />,
      content: <aboutSection version={version} />,
    },
    {
      id: 'getting-started',
      title: t('sections.gettingStarted'),
      icon: <LayoutDashboard class={iconTokens.size.sm} />,
      content: <gettingStartedSection />,
    },
    {
      id: 'link',
      title: t('sections.link'),
      icon: <Activity class={iconTokens.size.sm} />,
      content: <linkStatusSection />,
    },
    {
      id: 'cable',
      title: t('sections.cable'),
      icon: <Cable class={iconTokens.size.sm} />,
      content: <cableTestSection />,
    },
    {
      id: 'wifi',
      title: t('sections.wifi'),
      icon: <Wifi class={iconTokens.size.sm} />,
      content: <wiFiStatusSection />,
    },
    {
      id: 'network',
      title: t('sections.network'),
      icon: <Network class={iconTokens.size.sm} />,
      content: <networkSection />,
    },
    {
      id: 'gateway',
      title: t('sections.gateway'),
      icon: <Server class={iconTokens.size.sm} />,
      content: <gatewaySection />,
    },
    {
      id: 'dns',
      title: t('sections.dns'),
      icon: <Search class={iconTokens.size.sm} />,
      content: <dnsSection />,
    },
    {
      id: 'performance',
      title: t('sections.performance'),
      icon: <Zap class={iconTokens.size.sm} />,
      content: <performanceSection />,
    },
    {
      id: 'discovery',
      title: t('sections.discovery'),
      icon: <Search class={iconTokens.size.sm} />,
      content: <discoverySection />,
    },
    {
      id: 'healthChecks',
      title: t('sections.healthChecks'),
      icon: <Heart class={iconTokens.size.sm} />,
      content: <healthChecksSection />,
    },
    {
      id: 'security',
      title: t('sections.security'),
      icon: <Shield class={iconTokens.size.sm} />,
      content: <securitySection />,
    },
    {
      id: 'troubleshooting',
      title: t('sections.troubleshooting'),
      icon: <AlertTriangle class={iconTokens.size.sm} />,
      content: <troubleshootingSection />,
    },
    {
      id: 'profiles',
      title: t('sections.profiles'),
      icon: <SlidersHorizontal class={iconTokens.size.sm} />,
      content: <profilesSection />,
    },
    {
      id: 'wifiSurvey',
      title: t('sections.wifiSurvey'),
      icon: <Signal class={iconTokens.size.sm} />,
      content: <wiFiSurveySection />,
    },
    {
      id: 'rtspChecks',
      title: t('sections.rtspChecks'),
      icon: <Monitor class={iconTokens.size.sm} />,
      content: <rtspChecksSection />,
    },
    {
      id: 'dicomChecks',
      title: t('sections.dicomChecks'),
      icon: <HeartPulse class={iconTokens.size.sm} />,
      content: <dicomChecksSection />,
    },
    {
      id: 'howTo',
      title: t('sections.howTo'),
      icon: <Lightbulb class={iconTokens.size.sm} />,
      content: <howToSection />,
    },
    {
      id: 'glossary',
      title: t('sections.glossary'),
      icon: <BookOpen class={iconTokens.size.sm} />,
      content: <glossarySection />,
    },
  ];

  const filteredSections = sections.filter(
    (section) =>
      section.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      section.id.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  const currentSection = sections.find((s) => s.id === activeSection);

  return (
    <div class={modal.overlay}>
      {/* Backdrop */}
      <div class={modal.backdrop} onClick={onClose} aria-hidden="true" />

      {/* Modal */}
      <div
        class={cn(
          'relative',
          modal.content,
          modal.size.xl,
          radius.lg,
          'flex flex-col overflow-hidden',
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby="help-modal-title"
      >
        {/* Header */}
        <div
          class={cn(
            layout.flex.between,
            spacing.pad.default,
            'border-b border-surface-border shrink-0',
          )}
        >
          <h2 id="help-modal-title" class="heading-3">
            {t('modal.title')}
          </h2>
          <button
            type="button"
            onClick={onClose}
            class={cn(
              spacing.pad.xs,
              'text-text-muted hover:text-text-primary transition-colors',
              radius.default,
              'hover:bg-surface-hover',
            )}
            aria-label={t('modal.closeHelp')}
          >
            <svg
              class={iconTokens.size.md}
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
        <div class="flex flex-1 overflow-hidden">
          {/* Sidebar / TOC */}
          <aside class="w-64 border-r border-surface-border bg-surface-base overflow-y-auto shrink-0">
            {/* Search */}
            <div class={cn(spacing.pad.sm, 'border-b border-surface-border')}>
              <div class="relative">
                <Search
                  class={cn(
                    'absolute left-3 top-1/2 -translate-y-1/2',
                    iconTokens.size.sm,
                    'text-text-muted',
                  )}
                />
                <input
                  type="text"
                  placeholder={t('modal.searchPlaceholder')}
                  value={searchQuery}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setSearchQuery(e.target.value)
                  }
                  class={cn(
                    'w-full pl-9',
                    spacing.chip.lg,
                    'body-small',
                    radius.default,
                    'border border-surface-border bg-surface-raised text-text-primary placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-brand-primary',
                  )}
                />
              </div>
            </div>

            {/* Table of Contents */}
            <nav class={cn(spacing.pad.xs, 'stack-xs')}>
              <p class={cn('caption', spacing.chip.lg, 'uppercase tracking-wider')}>
                {t('modal.contents')}
              </p>
              {filteredSections.map((section) => (
                <button
                  type="button"
                  key={section.id}
                  onClick={(): void => setActiveSection(section.id)}
                  class={cn(
                    'w-full flex items-center',
                    spacing.gap.default,
                    spacing.tab,
                    radius.default,
                    'body-small transition-colors text-left',
                    activeSection === section.id
                      ? 'bg-brand-primary/10 text-brand-primary font-medium'
                      : 'text-text-secondary hover:bg-surface-hover hover:text-text-primary',
                  )}
                >
                  {section.icon}
                  <span>{section.title}</span>
                </button>
              ))}
            </nav>
          </aside>

          {/* Main content */}
          <main class={cn('flex-1 overflow-y-auto', spacing.pad.lg)}>
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

function _aboutSection({ version }: AboutSectionProps): React.JSX.Element {
  const { t } = useTranslation('help');
  return (
    <div class="section-gap max-w-3xl">
      <div>
        <h3 class={cn('heading-2', spacing.margin.bottom.heading)}>{t('content.about.welcome')}</h3>
        <p class={cn('body leading-relaxed', spacing.margin.bottom.content)}>
          {t('content.about.description')}
        </p>
      </div>

      <div class={cn('grid md:grid-cols-2', spacing.gap.comfortable)}>
        <featureCard
          title={t('content.about.features.realTimeMonitoring.title')}
          description={t('content.about.features.realTimeMonitoring.description')}
        />
        <featureCard
          title={t('content.about.features.networkDiscovery.title')}
          description={t('content.about.features.networkDiscovery.description')}
        />
        <featureCard
          title={t('content.about.features.performanceTesting.title')}
          description={t('content.about.features.performanceTesting.description')}
        />
        <featureCard
          title={t('content.about.features.cableDiagnostics.title')}
          description={t('content.about.features.cableDiagnostics.description')}
        />
      </div>

      <div
        class={cn(
          'border-l-4 border-brand-primary bg-brand-primary/5',
          spacing.pad.default,
          radius.default,
        )}
      >
        <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>
          {t('content.about.licensing.title', 'Commercial Software')}
        </h4>
        <p class="body-small text-text-secondary">
          {t(
            'content.about.licensing.description',
            'SEED is commercial software developed by Mustard Seed Networks. All rights reserved. Unauthorized distribution or modification is prohibited.',
          )}
        </p>
      </div>

      <div>
        <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.heading)}>
          {t('content.about.versionInfo.title')}
        </h4>
        <dl class="grid grid-cols-2 gap-x-4 gap-y-2 body-small">
          <dt class="text-text-muted">{t('content.about.versionInfo.currentVersion')}</dt>
          <dd class="font-mono text-text-primary">{version}</dd>
          <dt class="text-text-muted">{t('content.about.versionInfo.backend')}</dt>
          <dd class="text-text-primary">Go 1.25.5+</dd>
          <dt class="text-text-muted">{t('content.about.versionInfo.frontend')}</dt>
          <dd class="text-text-primary">React 19.2 + TypeScript</dd>
          <dt class="text-text-muted">{t('content.about.versionInfo.runtime', 'Runtime')}</dt>
          <dd class="text-text-primary">Node.js 25.2</dd>
        </dl>
      </div>
    </div>
  );
}

function _gettingStartedSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const tips = t('content.gettingStarted.proTips.tips', {
    returnObjects: true,
  }) as string[];
  return (
    <div class="section-gap max-w-3xl">
      <h3 class={cn('heading-2', spacing.margin.bottom.heading)}>
        {t('content.gettingStarted.title')}
      </h3>

      <div class="stack-lg">
        <stepCard
          number={1}
          title={t('content.gettingStarted.steps.dashboard.title')}
          description={t('content.gettingStarted.steps.dashboard.description')}
        />
        <stepCard
          number={2}
          title={t('content.gettingStarted.steps.interface.title')}
          description={t('content.gettingStarted.steps.interface.description')}
        />
        <stepCard
          number={3}
          title={t('content.gettingStarted.steps.thresholds.title')}
          description={t('content.gettingStarted.steps.thresholds.description')}
        />
        <stepCard
          number={4}
          title={t('content.gettingStarted.steps.runTests.title')}
          description={t('content.gettingStarted.steps.runTests.description')}
        />
        <stepCard
          number={5}
          title={t('content.gettingStarted.steps.exploreCards.title')}
          description={t('content.gettingStarted.steps.exploreCards.description')}
        />
      </div>

      <div
        class={cn(
          'bg-surface-hover border border-surface-border',
          radius.default,
          spacing.pad.default,
          spacing.margin.top.section,
        )}
      >
        <h4
          class={cn(
            'font-semibold text-text-primary',
            spacing.margin.bottom.inline,
            'flex items-center',
            spacing.gap.compact,
          )}
        >
          <span class="text-status-info">💡</span>
          {t('content.gettingStarted.proTips.title')}
        </h4>
        <ul class={cn('body-small stack-sm', spacing.margin.left.spacious, 'list-disc')}>
          {tips.map((tip) => (
            <li key={tip}>{tip}</li>
          ))}
        </ul>
      </div>
    </div>
  );
}

function _linkStatusSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  return (
    <helpContentSection title={t('sections.link')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.linkStatus.description')}
      </p>
      <helpTermList
        items={[
          {
            term: t('content.linkStatus.terms.carrier.term'),
            description: t('content.linkStatus.terms.carrier.description'),
          },
          {
            term: t('content.linkStatus.terms.speed.term'),
            description: t('content.linkStatus.terms.speed.description'),
          },
          {
            term: t('content.linkStatus.terms.duplex.term'),
            description: t('content.linkStatus.terms.duplex.description'),
          },
          {
            term: t('content.linkStatus.terms.autoNeg.term'),
            description: t('content.linkStatus.terms.autoNeg.description'),
          },
          {
            term: t('content.linkStatus.terms.mtu.term'),
            description: t('content.linkStatus.terms.mtu.description'),
          },
        ]}
      />
    </helpContentSection>
  );
}

function _cableTestSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  return (
    <helpContentSection title={t('sections.cable')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.cableTest.description')}
      </p>
      <helpTermList
        items={[
          {
            term: t('content.cableTest.terms.tdrTest.term'),
            description: t('content.cableTest.terms.tdrTest.description'),
          },
          {
            term: t('content.cableTest.terms.cableStatus.term'),
            description: t('content.cableTest.terms.cableStatus.description'),
          },
          {
            term: t('content.cableTest.terms.faultDistance.term'),
            description: t('content.cableTest.terms.faultDistance.description'),
          },
          {
            term: t('content.cableTest.terms.pairs.term'),
            description: t('content.cableTest.terms.pairs.description'),
          },
        ]}
      />
      <div
        class={cn(
          spacing.margin.top.content,
          'bg-status-warning/10 border border-status-warning/20',
          radius.default,
          spacing.pad.sm,
        )}
      >
        <p class="caption text-status-warning">
          <strong>{t('common:labels.note', 'Note')}:</strong> {t('content.cableTest.note')}
        </p>
      </div>
    </helpContentSection>
  );
}

function _wiFiStatusSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  return (
    <helpContentSection title={t('sections.wifi')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.wifiStatus.description')}
      </p>
      <helpTermList
        items={[
          {
            term: 'SSID',
            description:
              "Service Set Identifier - the name of the wireless network you're connected to.",
          },
          {
            term: 'BSSID',
            description: 'Basic Service Set Identifier - MAC address of the access point.',
          },
          {
            term: 'Signal Strength',
            description:
              'Signal strength in dBm. -30 is excellent, -67 is good, -70 is fair, -80 is weak. Higher (less negative) is better.',
          },
          {
            term: 'Channel',
            description:
              'WiFi channel number (1-14 for 2.4GHz, 36-165 for 5GHz). Overlapping channels cause interference.',
          },
          {
            term: 'Security',
            description:
              'Encryption protocol protecting the connection (WPA2, WPA3, WEP, or Open).',
          },
          {
            term: 'Frequency',
            description:
              'Radio band - 2.4GHz has better range, 5GHz has better speed and less interference.',
          },
        ]}
      />
    </helpContentSection>
  );
}

function _networkSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  return (
    <helpContentSection title={t('sections.network')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.networkDhcp.description')}
      </p>
      <helpTermList
        items={[
          {
            term: 'Lease Time',
            description: 'Duration of current IP address assignment before renewal is needed.',
          },
          {
            term: 'DHCP Server',
            description:
              'IP address of the DHCP server that issued the lease (usually your router).',
          },
          {
            term: 'Gateway',
            description: 'Default gateway assigned by DHCP for routing traffic off-subnet.',
          },
          {
            term: 'DNS Servers',
            description: 'DNS servers assigned by DHCP for name resolution.',
          },
          {
            term: 'Subnet Mask',
            description: 'Network mask defining the local subnet size.',
          },
        ]}
      />
    </helpContentSection>
  );
}

function _gatewaySection(): React.JSX.Element {
  const { t } = useTranslation('help');
  return (
    <helpContentSection title={t('sections.gateway')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.gatewayHelp.description')}
      </p>
      <helpTermList
        items={[
          {
            term: 'IPv4 Gateway',
            description: 'Default router for IPv4 traffic leaving your local network.',
          },
          {
            term: 'IPv6 Gateway',
            description: 'Default router for IPv6 traffic (may be link-local address).',
          },
          {
            term: 'Reachability',
            description: 'Whether the gateway responds to ICMP ping requests.',
          },
          {
            term: 'Latency',
            description: 'Round-trip time to gateway. Should be <1ms for local networks.',
          },
          {
            term: 'Packet Loss',
            description: "Percentage of ping packets that didn't receive a response.",
          },
        ]}
      />
    </helpContentSection>
  );
}

function _dnsSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  return (
    <helpContentSection title={t('sections.dns')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.dnsTests.description')}
      </p>
      <helpTermList
        items={[
          {
            term: 'Forward Lookup',
            description: 'Resolves hostname to IPv4 address (A record).',
          },
          {
            term: 'Reverse Lookup',
            description: 'Resolves IP address back to hostname (PTR record).',
          },
          {
            term: 'IPv6 Lookup',
            description: 'Resolves hostname to IPv6 address (AAAA record).',
          },
          {
            term: 'Latency',
            description: 'Time taken for the DNS query to complete. Good: <50ms for local DNS.',
          },
        ]}
      />
    </helpContentSection>
  );
}

function _performanceSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  return (
    <helpContentSection title={t('sections.performance')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.performanceTests.description')}
      </p>
      <helpTermList
        items={[
          {
            term: 'Internet Speed Test',
            description:
              'Tests download/upload speeds to public speedtest servers. Measures your connection to the internet.',
          },
          {
            term: 'LAN Speed (iperf3)',
            description:
              'Tests throughput on your local network using iperf3 to a configured server.',
          },
          {
            term: 'Download',
            description: 'Maximum download speed achieved during the test.',
          },
          {
            term: 'Upload',
            description: 'Maximum upload speed achieved during the test.',
          },
          {
            term: 'Latency',
            description: 'Round-trip time (ping) to the test server.',
          },
          {
            term: 'Jitter',
            description:
              'Variation in latency over time. Lower is better for real-time applications like VoIP and gaming.',
          },
        ]}
      />
    </helpContentSection>
  );
}

function _discoverySection(): React.JSX.Element {
  const { t } = useTranslation('help');
  return (
    <helpContentSection title={t('sections.discovery')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.networkDiscovery.description')}
      </p>
      <helpTermList
        items={[
          {
            term: 'Network Scan',
            description:
              'Discovers active devices on your network subnet using ARP/ICMP ping sweeps.',
          },
          {
            term: 'MAC Address',
            description:
              'Hardware address of the network interface - unique identifier for the device.',
          },
          {
            term: 'Vendor',
            description: 'Manufacturer identified from the MAC address OUI (first 3 bytes).',
          },
          {
            term: 'Hostname',
            description: 'DNS hostname if reverse lookup succeeded.',
          },
          {
            term: 'LLDP/CDP',
            description:
              'Link Layer Discovery Protocol (standard) or Cisco Discovery Protocol - provides information about directly connected network switches.',
          },
        ]}
      />
    </helpContentSection>
  );
}

function _healthChecksSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const commonIssues = t('content.healthChecks.commonIssues', { returnObjects: true }) as {
    title: string;
    timeout: string;
    highLatency: string;
    packetLoss: string;
    connectionRefused: string;
  };
  return (
    <helpContentSection title={t('sections.healthChecks')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.healthChecks.description')}
      </p>
      <helpTermList
        items={[
          {
            term: t('content.healthChecks.terms.pingTest.term'),
            description: t('content.healthChecks.terms.pingTest.description'),
          },
          {
            term: t('content.healthChecks.terms.tcpTest.term'),
            description: t('content.healthChecks.terms.tcpTest.description'),
          },
          {
            term: t('content.healthChecks.terms.httpTest.term'),
            description: t('content.healthChecks.terms.httpTest.description'),
          },
          {
            term: t('content.healthChecks.terms.customTargets.term'),
            description: t('content.healthChecks.terms.customTargets.description'),
          },
          {
            term: t('content.healthChecks.terms.thresholds.term'),
            description: t('content.healthChecks.terms.thresholds.description'),
          },
        ]}
      />
      <div
        class={cn(
          spacing.margin.top.content,
          'bg-status-info/10 border border-status-info/20',
          radius.default,
          spacing.pad.default,
        )}
      >
        <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>
          {commonIssues.title}
        </h4>
        <ul
          class={cn(
            'body-small text-text-secondary stack-sm',
            spacing.margin.left.spacious,
            'list-disc',
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
    </helpContentSection>
  );
}

function _securitySection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const recovery = t('content.security.passwordRecovery', { returnObjects: true }) as {
    title: string;
    description: string;
    steps: string[];
    note: string;
  };
  const portDetails = t('content.security.portScanDetails', { returnObjects: true }) as {
    title: string;
    description: string;
    levels: Record<string, string>;
    commonPorts: Record<string, string>;
  };
  return (
    <helpContentSection title={t('sections.security')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.security.description')}
      </p>
      <helpTermList
        items={[
          {
            term: t('content.security.terms.portScan.term'),
            description: t('content.security.terms.portScan.description'),
          },
          {
            term: t('content.security.terms.vulnScan.term'),
            description: t('content.security.terms.vulnScan.description'),
          },
          {
            term: t('content.security.terms.devicePosture.term'),
            description: t('content.security.terms.devicePosture.description'),
          },
          {
            term: t('content.security.terms.rogueDhcp.term'),
            description: t('content.security.terms.rogueDhcp.description'),
          },
        ]}
      />

      {/* Password Recovery Section */}
      <div
        class={cn(
          spacing.margin.top.section,
          'bg-status-warning/10 border border-status-warning/20',
          radius.default,
          spacing.pad.default,
        )}
      >
        <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {recovery.title}
        </h4>
        <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
          {recovery.description}
        </p>
        <ol
          class={cn(
            'body-small text-text-secondary stack-sm',
            spacing.margin.left.spacious,
            'list-decimal',
          )}
        >
          {recovery.steps.map((step) => (
            <li
              key={step}
              class={
                step.startsWith('User mode:') || step.startsWith('System mode:')
                  ? 'font-mono text-xs bg-surface-base px-2 py-1 rounded'
                  : ''
              }
            >
              {step}
            </li>
          ))}
        </ol>
        <p class={cn('caption text-status-warning', spacing.margin.top.content)}>
          <strong>Note:</strong> {recovery.note}
        </p>
      </div>

      {/* Port Scan Details */}
      <div class={cn(spacing.margin.top.section)}>
        <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {portDetails.title}
        </h4>
        <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
          {portDetails.description}
        </p>
        <div class="grid grid-cols-2 gap-2 body-small">
          {Object.entries(portDetails.levels).map(([level, desc]) => (
            <div key={level} class={cn('border-l-2 border-surface-border', spacing.pad.sm)}>
              <dt class="font-semibold text-text-primary capitalize">{level}</dt>
              <dd class="text-text-secondary">{desc}</dd>
            </div>
          ))}
        </div>
      </div>

      {/* Common Ports Reference */}
      <div class={cn(spacing.margin.top.content)}>
        <h5 class={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>
          Common Ports Reference
        </h5>
        <div class="grid grid-cols-2 md:grid-cols-3 gap-2 body-small font-mono">
          {Object.entries(portDetails.commonPorts).map(([port, desc]) => (
            <div key={port} class="flex items-baseline gap-2">
              <span class="text-brand-primary font-bold">{port}</span>
              <span class="text-text-muted">{desc}</span>
            </div>
          ))}
        </div>
      </div>
    </helpContentSection>
  );
}

function _troubleshootingSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const categories = t('content.troubleshooting.categories', { returnObjects: true }) as Record<
    string,
    {
      title: string;
      [key: string]: unknown;
    }
  >;

  return (
    <helpContentSection title={t('sections.troubleshooting')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.troubleshooting.description')}
      </p>

      {/* Link Issues */}
      <troubleshootingCategory
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
      <troubleshootingCategory
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
      <troubleshootingCategory
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
      <troubleshootingCategory
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
      <troubleshootingCategory
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
    </helpContentSection>
  );
}

interface TroubleshootingIssue {
  symptom: string;
  causes: string[];
  solutions: string[];
}

function _troubleshootingCategory({
  title,
  issues,
}: {
  title: string;
  issues: TroubleshootingIssue[];
}): React.JSX.Element {
  return (
    <div class={cn(spacing.margin.top.section)}>
      <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>{title}</h4>
      <div class="stack-lg">
        {issues.map((issue) => (
          <div
            key={issue.symptom}
            class={cn('border border-surface-border', radius.default, spacing.pad.default)}
          >
            <h5 class={cn('font-semibold text-status-warning', spacing.margin.bottom.inline)}>
              {issue.symptom}
            </h5>
            <div class="grid md:grid-cols-2 gap-4 body-small">
              <div>
                <p class="font-semibold text-text-primary mb-1">Possible Causes:</p>
                <ul class={cn('text-text-secondary', spacing.margin.left.comfortable, 'list-disc')}>
                  {issue.causes.map((cause) => (
                    <li key={cause}>{cause}</li>
                  ))}
                </ul>
              </div>
              <div>
                <p class="font-semibold text-text-primary mb-1">Solutions:</p>
                <ul class={cn('text-text-secondary', spacing.margin.left.comfortable, 'list-disc')}>
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

function _featureCard({
  title,
  description,
}: {
  title: string;
  description: string;
}): React.JSX.Element {
  return (
    <div
      class={cn('bg-surface-hover border border-surface-border', radius.lg, spacing.pad.default)}
    >
      <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>{title}</h4>
      <p class="body-small text-text-secondary">{description}</p>
    </div>
  );
}

function _stepCard({
  number,
  title,
  description,
}: {
  number: number;
  title: string;
  description: string;
}): React.JSX.Element {
  return (
    <div class={cn('flex', spacing.gap.comfortable)}>
      <div
        class={cn(
          'shrink-0 w-8 h-8',
          radius.full,
          'bg-brand-primary text-text-inverse',
          layout.flex.center,
          'font-semibold',
        )}
      >
        {number}
      </div>
      <div class="flex-1">
        <h4 class={cn('font-semibold', spacing.margin.bottom.inline)}>{title}</h4>
        <p class="body-small">{description}</p>
      </div>
    </div>
  );
}

function _helpContentSection({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}): React.JSX.Element {
  return (
    <div class="max-w-3xl">
      <h3 class={cn('heading-2', spacing.margin.bottom.content)}>{title}</h3>
      {children}
    </div>
  );
}

function _helpTermList({
  items,
}: {
  items: Array<{ term: string; description: string }>;
}): React.JSX.Element {
  return (
    <dl class="stack-lg">
      {items.map((item) => (
        <div key={item.term} class={cn('border-l-2 border-surface-border', spacing.pad.default)}>
          <dt class={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>
            {item.term}
          </dt>
          <dd class="body-small text-text-secondary">{item.description}</dd>
        </div>
      ))}
    </dl>
  );
}

// ============================================================================
// NEW FEATURE SECTIONS
// ============================================================================

function _profilesSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const capabilities = t('content.profiles.capabilities', { returnObjects: true }) as string[];
  const useCases = t('content.profiles.useCases.items', { returnObjects: true }) as Array<{
    name: string;
    description: string;
  }>;

  return (
    <helpContentSection title={t('sections.profiles')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.profiles.description')}
      </p>

      <div class={spacing.margin.bottom.section}>
        <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {t('content.profiles.overview.title')}
        </h4>
        <p class="body-small text-text-secondary">{t('content.profiles.overview.content')}</p>
      </div>

      <div class={spacing.margin.bottom.section}>
        <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {t('content.profiles.capabilities_title', 'Profile Capabilities')}
        </h4>
        <ul
          class={cn(
            'body-small text-text-secondary stack-sm',
            spacing.margin.left.spacious,
            'list-disc',
          )}
        >
          {capabilities?.map((cap) => (
            <li key={cap}>{cap}</li>
          ))}
        </ul>
      </div>

      <div>
        <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {t('content.profiles.useCases.title')}
        </h4>
        <div class="stack-lg">
          {useCases?.map((useCase) => (
            <div
              key={useCase.name}
              class={cn('border-l-2 border-brand-primary', spacing.pad.default)}
            >
              <dt class={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>
                {useCase.name}
              </dt>
              <dd class="body-small text-text-secondary">{useCase.description}</dd>
            </div>
          ))}
        </div>
      </div>
    </helpContentSection>
  );
}

function _wiFiSurveySection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const visualizations = t('content.wifiSurvey.visualizations', { returnObjects: true }) as Array<{
    type: string;
    description: string;
  }>;
  const bestPractices = t('content.wifiSurvey.bestPractices.items', {
    returnObjects: true,
  }) as string[];

  return (
    <helpContentSection title={t('sections.wifiSurvey')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.wifiSurvey.description')}
      </p>

      <helpTermList
        items={[
          {
            term: t('content.wifiSurvey.terms.floorPlan.term'),
            description: t('content.wifiSurvey.terms.floorPlan.description'),
          },
          {
            term: t('content.wifiSurvey.terms.heatmap.term'),
            description: t('content.wifiSurvey.terms.heatmap.description'),
          },
          {
            term: t('content.wifiSurvey.terms.surveyPoint.term'),
            description: t('content.wifiSurvey.terms.surveyPoint.description'),
          },
          {
            term: t('content.wifiSurvey.terms.dataRate.term'),
            description: t('content.wifiSurvey.terms.dataRate.description'),
          },
        ]}
      />

      <div class={spacing.margin.top.section}>
        <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {t('content.wifiSurvey.visualizationsTitle', 'Visualization Modes')}
        </h4>
        <div class="grid md:grid-cols-2 gap-4">
          {visualizations?.map((viz) => (
            <div
              key={viz.type}
              class={cn(
                'bg-surface-hover border border-surface-border',
                radius.default,
                spacing.pad.sm,
              )}
            >
              <h5 class="font-semibold text-text-primary">{viz.type}</h5>
              <p class="body-small text-text-secondary">{viz.description}</p>
            </div>
          ))}
        </div>
      </div>

      <div
        class={cn(
          spacing.margin.top.section,
          'bg-status-info/10 border border-status-info/20',
          radius.default,
          spacing.pad.default,
        )}
      >
        <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>
          {t('content.wifiSurvey.bestPractices.title')}
        </h4>
        <ul
          class={cn(
            'body-small text-text-secondary stack-sm',
            spacing.margin.left.spacious,
            'list-disc',
          )}
        >
          {bestPractices?.map((practice) => (
            <li key={practice}>{practice}</li>
          ))}
        </ul>
      </div>
    </helpContentSection>
  );
}

function _rtspChecksSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const configuration = t('content.rtspChecks.configuration', { returnObjects: true }) as Array<{
    field: string;
    description: string;
  }>;

  return (
    <helpContentSection title={t('sections.rtspChecks')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.rtspChecks.description')}
      </p>

      <helpTermList
        items={[
          {
            term: t('content.rtspChecks.terms.rtsp.term'),
            description: t('content.rtspChecks.terms.rtsp.description'),
          },
          {
            term: t('content.rtspChecks.terms.options.term'),
            description: t('content.rtspChecks.terms.options.description'),
          },
          {
            term: t('content.rtspChecks.terms.describe.term'),
            description: t('content.rtspChecks.terms.describe.description'),
          },
          {
            term: t('content.rtspChecks.terms.authentication.term'),
            description: t('content.rtspChecks.terms.authentication.description'),
          },
        ]}
      />

      <div class={spacing.margin.top.section}>
        <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {t('content.rtspChecks.configurationTitle', 'Configuration Options')}
        </h4>
        <div class="stack-sm">
          {configuration?.map((config) => (
            <div key={config.field} class={cn('border-l-2 border-surface-border', spacing.pad.sm)}>
              <span class="font-mono text-brand-primary">{config.field}</span>
              <span class="body-small text-text-secondary ml-2">{config.description}</span>
            </div>
          ))}
        </div>
      </div>
    </helpContentSection>
  );
}

function _dicomChecksSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const configuration = t('content.dicomChecks.configuration', { returnObjects: true }) as Array<{
    field: string;
    description: string;
  }>;
  const commonIssues = t('content.dicomChecks.commonIssues', { returnObjects: true }) as Array<{
    issue: string;
    solution: string;
  }>;

  return (
    <helpContentSection title={t('sections.dicomChecks')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.dicomChecks.description')}
      </p>

      <helpTermList
        items={[
          {
            term: t('content.dicomChecks.terms.dicom.term'),
            description: t('content.dicomChecks.terms.dicom.description'),
          },
          {
            term: t('content.dicomChecks.terms.cEcho.term'),
            description: t('content.dicomChecks.terms.cEcho.description'),
          },
          {
            term: t('content.dicomChecks.terms.aeTitle.term'),
            description: t('content.dicomChecks.terms.aeTitle.description'),
          },
          {
            term: t('content.dicomChecks.terms.scp.term'),
            description: t('content.dicomChecks.terms.scp.description'),
          },
          {
            term: t('content.dicomChecks.terms.scu.term'),
            description: t('content.dicomChecks.terms.scu.description'),
          },
        ]}
      />

      <div class={spacing.margin.top.section}>
        <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {t('content.dicomChecks.configurationTitle', 'Configuration')}
        </h4>
        <div class="stack-sm">
          {configuration?.map((config) => (
            <div key={config.field} class={cn('border-l-2 border-surface-border', spacing.pad.sm)}>
              <span class="font-mono text-brand-primary">{config.field}</span>
              <span class="body-small text-text-secondary ml-2">{config.description}</span>
            </div>
          ))}
        </div>
      </div>

      <div
        class={cn(
          spacing.margin.top.section,
          'bg-status-warning/10 border border-status-warning/20',
          radius.default,
          spacing.pad.default,
        )}
      >
        <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {t('content.dicomChecks.commonIssuesTitle', 'Common Issues')}
        </h4>
        <div class="stack-lg">
          {commonIssues?.map((item) => (
            <div key={item.issue}>
              <p class="font-semibold text-status-warning">{item.issue}</p>
              <p class="body-small text-text-secondary">{item.solution}</p>
            </div>
          ))}
        </div>
      </div>
    </helpContentSection>
  );
}

function _howToSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const guides = t('content.howTo.guides', { returnObjects: true }) as Record<
    string,
    {
      title: string;
      description: string;
      steps: string[];
    }
  >;

  return (
    <helpContentSection title={t('sections.howTo')}>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.howTo.description')}
      </p>

      <div class="stack-xl">
        {guides
          ? Object.entries(guides).map(([key, guide]) => (
              <div
                key={key}
                class={cn('border border-surface-border', radius.lg, spacing.pad.default)}
              >
                <h4 class={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>
                  {guide.title}
                </h4>
                <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
                  {guide.description}
                </p>
                <ol
                  class={cn(
                    'body-small text-text-secondary stack-sm',
                    spacing.margin.left.spacious,
                    'list-decimal',
                  )}
                >
                  {guide.steps.map((step) => (
                    <li key={`${key}-${step.slice(0, 50)}`}>{step}</li>
                  ))}
                </ol>
              </div>
            ))
          : null}
      </div>
    </helpContentSection>
  );
}

function _glossarySection(): React.JSX.Element {
  const { t } = useTranslation('glossary');
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('all');

  const categories = t('categories', { returnObjects: true }) as Record<string, string>;
  const terms = t('terms', { returnObjects: true }) as Record<
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
          searchTerm === '' ||
          termData.term.toLowerCase().includes(searchTerm.toLowerCase()) ||
          termData.fullName.toLowerCase().includes(searchTerm.toLowerCase()) ||
          termData.definition.toLowerCase().includes(searchTerm.toLowerCase());

        const matchesCategory =
          selectedCategory === 'all' || termData.category === selectedCategory;

        return matchesSearch && matchesCategory;
      })
    : [];

  return (
    <div class="max-w-3xl">
      <h3 class={cn('heading-2', spacing.margin.bottom.content)}>{t('title')}</h3>
      <p class={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('description')}
      </p>

      {/* Search and Filter */}
      <div class={cn('flex flex-wrap gap-4', spacing.margin.bottom.section)}>
        <div class="flex-1 min-w-[200px]">
          <div class="relative">
            <Search
              class={cn('absolute left-3 top-1/2 -translate-y-1/2', 'w-4 h-4 text-text-muted')}
            />
            <input
              type="text"
              placeholder="Search terms..."
              value={searchTerm}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setSearchTerm(e.target.value)
              }
              class={cn(
                'w-full pl-9 pr-3 py-2',
                'body-small',
                radius.default,
                'border border-surface-border bg-surface-raised text-text-primary placeholder-text-muted',
                'focus:outline-none focus:ring-2 focus:ring-brand-primary',
              )}
            />
          </div>
        </div>
        <select
          value={selectedCategory}
          onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
            setSelectedCategory(e.target.value)
          }
          class={cn(
            'px-3 py-2',
            'body-small',
            radius.default,
            'border border-surface-border bg-surface-raised text-text-primary',
            'focus:outline-none focus:ring-2 focus:ring-brand-primary',
          )}
        >
          <option value="all">All Categories</option>
          {categories
            ? Object.entries(categories).map(([key, label]) => (
                <option key={key} value={key}>
                  {label}
                </option>
              ))
            : null}
        </select>
      </div>

      {/* Terms List */}
      <div class="stack-lg">
        {filteredTerms.map(([key, termData]) => (
          <div
            key={key}
            class={cn(
              'border border-surface-border',
              radius.default,
              spacing.pad.default,
              'hover:border-brand-primary/50 transition-colors',
            )}
          >
            <div class="flex items-start justify-between gap-4">
              <div class="flex-1">
                <div class="flex items-baseline gap-2 mb-1">
                  <span class="font-bold text-brand-primary">{termData.term}</span>
                  <span class="body-small text-text-muted">({termData.fullName})</span>
                </div>
                <p class="body-small text-text-secondary">{termData.definition}</p>
              </div>
              <span
                class={cn(
                  'px-2 py-0.5 text-xs font-medium',
                  radius.default,
                  'bg-surface-hover text-text-muted capitalize',
                )}
              >
                {categories?.[termData.category] || termData.category}
              </span>
            </div>
          </div>
        ))}

        {filteredTerms.length === 0 && (
          <div class="text-center py-8 text-text-muted">No terms found matching your search.</div>
        )}
      </div>
    </div>
  );
}
