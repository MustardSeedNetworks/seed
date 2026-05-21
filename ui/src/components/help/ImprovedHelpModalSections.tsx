/**
 * Help modal content sections.
 *
 * Each `_xxxSection` function renders the body for one help topic
 * (About, Network, Performance, Discovery, etc.). They were originally
 * declared inline inside ImprovedHelpModal.tsx; relocated here to keep
 * the modal shell file slim. The lowercase JSX usage (<aboutSection />)
 * in the parent file is preserved as-is.
 */

import type React from 'react';
import { useTranslation } from 'react-i18next';
import { cn, radius, spacing, status as statusColor } from '../../styles/theme';

// ============================================================================
// CONTENT SECTIONS
// ============================================================================

interface AboutSectionProps {
  version: string;
}

function _aboutSection({ version }: AboutSectionProps): React.JSX.Element {
  const { t } = useTranslation('help');
  return (
    <div className="section-gap max-w-3xl">
      <div>
        <h3 className={cn('heading-2', spacing.margin.bottom.heading)}>
          {t('content.about.welcome')}
        </h3>
        <p className={cn('body leading-relaxed', spacing.margin.bottom.content)}>
          {t('content.about.description')}
        </p>
      </div>
      <div className={cn('grid md:grid-cols-2', spacing.gap.comfortable)}>
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
        className={cn(
          'border-l-4 border-brand-primary bg-brand-primary/5',
          spacing.pad.default,
          radius.default,
        )}
      >
        <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>
          {t('content.about.licensing.title', 'Commercial Software')}
        </h4>
        <p className="body-small text-text-secondary">
          {t(
            'content.about.licensing.description',
            'SEED is commercial software developed by Mustard Seed Networks. All rights reserved. Unauthorized distribution or modification is prohibited.',
          )}
        </p>
      </div>
      <div>
        <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.heading)}>
          {t('content.about.versionInfo.title')}
        </h4>
        <dl className="grid grid-cols-2 gap-x-4 gap-y-2 body-small">
          <dt className="text-text-muted">{t('content.about.versionInfo.currentVersion')}</dt>
          <dd className="font-mono text-text-primary">{version}</dd>
          <dt className="text-text-muted">{t('content.about.versionInfo.backend')}</dt>
          <dd className="text-text-primary">Go 1.25.5+</dd>
          <dt className="text-text-muted">{t('content.about.versionInfo.frontend')}</dt>
          <dd className="text-text-primary">React 19.2 + TypeScript</dd>
          <dt className="text-text-muted">{t('content.about.versionInfo.runtime', 'Runtime')}</dt>
          <dd className="text-text-primary">Node.js 25.2</dd>
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
    <div className="section-gap max-w-3xl">
      <h3 className={cn('heading-2', spacing.margin.bottom.heading)}>
        {t('content.gettingStarted.title')}
      </h3>
      <div className="stack-lg">
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
        className={cn(
          'bg-surface-hover border border-surface-border',
          radius.default,
          spacing.pad.default,
          spacing.margin.top.section,
        )}
      >
        <h4
          className={cn(
            'font-semibold text-text-primary',
            spacing.margin.bottom.inline,
            'flex items-center',
            spacing.gap.compact,
          )}
        >
          <span className={statusColor.text.info}>💡</span>
          {t('content.gettingStarted.proTips.title')}
        </h4>
        <ul className={cn('body-small stack-sm', spacing.margin.left.spacious, 'list-disc')}>
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
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
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
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
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
        className={cn(
          spacing.margin.top.content,
          'bg-status-warning/10 border border-status-warning/20',
          radius.default,
          spacing.pad.sm,
        )}
      >
        <p className="caption text-status-warning">
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
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
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
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
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
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
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
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
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
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
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
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
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
