/**
 * AppDashboard - the four dashboard sections shown to authenticated users.
 *
 * Connectivity, Network Services, Testing & Discovery, and System sections.
 * Pulled out of App.tsx as pure presentation so the orchestration component
 * only wires hook state into props.
 */

import { useTranslation } from 'react-i18next';
import type { CardState } from '../../hooks/useCardState';
import { cn, layout, spacing } from '../../styles/theme';
import type { ChannelGraphResponse } from '../../types';
import type { CardSettings, DisplayOptions } from '../../types/settings';
import { CableCard } from '../cards/CableCard';
import { DnsCard } from '../cards/DnsCard';
import { GatewayCard } from '../cards/GatewayCard';
import { GuestNetworkAuditCard } from '../cards/GuestNetworkAuditCard';
import { HealthCheckCard } from '../cards/HealthCheckCard';
import { LinkCard } from '../cards/LinkCard';
import { LogViewerCard } from '../cards/LogViewerCard';
import { NetworkCard } from '../cards/NetworkCard';
import { NetworkDiscoveryCard, type NetworkDiscoveryData } from '../cards/NetworkDiscoveryCard';
import { PathDiscoveryCard, type TraceHopMessage } from '../cards/PathDiscoveryCard';
import { PerformanceCard } from '../cards/PerformanceCard';
import { PublicIpCard } from '../cards/PublicIpCard';
import { SLADashboardCard } from '../cards/SlaDashboardCard';
import { SwitchCard } from '../cards/SwitchCard';
import { SystemHealthCard } from '../cards/SystemHealthCard';
import { WiFiCard } from '../cards/WiFiCard';
import { WifiChannelGraph } from '../cards/WiFiChannelGraph';
import { WiFiSurveyCard } from '../cards/WiFiSurveyCard';

interface AppDashboardProps {
  cards: CardState;
  loading: boolean;
  isWifi: boolean;
  currentInterface: string;
  cardSettings: CardSettings;
  displayOptions: DisplayOptions;
  networkDiscovery: NetworkDiscoveryData | null;
  triggerDeviceScan: () => Promise<void>;
  registerTraceHopHandler: (handler: (msg: TraceHopMessage) => void) => () => void;
  channelGraphData: ChannelGraphResponse | null;
  channelGraphLoading: boolean;
}

// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Dashboard renders many conditional cards; the gating logic mirrors the previous inline App.tsx render
export function AppDashboard({
  cards,
  loading,
  isWifi,
  currentInterface,
  cardSettings,
  displayOptions,
  networkDiscovery,
  triggerDeviceScan,
  registerTraceHopHandler,
  channelGraphData,
  channelGraphLoading,
}: AppDashboardProps): JSX.Element {
  const { t } = useTranslation('common');

  return (
    <>
      {/* Section: Primary Connectivity - cards differ by interface type */}
      <section aria-labelledby="connectivity-heading" class={spacing.margin.bottom.section}>
        <h2 id="connectivity-heading" class={cn('section-title', spacing.margin.bottom.heading)}>
          {t('sections.connectivity')}
        </h2>
        <div class={layout.grid.cards}>
          {/* WiFi-only cards */}
          {isWifi ? <WiFiCard data={cards.wifi} loading={loading} visible={true} /> : null}

          {/* Ethernet-only cards */}
          {!isWifi && (
            <>
              <LinkCard data={cards.link} loading={loading} />
              {/* Cable Test card: only render when the link is DOWN
                  (cable plugged in + working = nothing to diagnose).
                  The card itself handles the supported/not-supported
                  branches when rendered. Fixes #740. */}
              {cards.link && cards.link.linkUp === false ? (
                <CableCard
                  data={cards.cable}
                  loading={loading}
                  unitSystem={displayOptions.unitSystem}
                />
              ) : null}
              <SwitchCard data={cards.switch} vlanData={cards.vlan} loading={loading} />
            </>
          )}
        </div>
      </section>

      {/* Section: Network Services */}
      <section aria-labelledby="network-heading" class={spacing.margin.bottom.section}>
        <h2 id="network-heading" class={cn('section-title', spacing.margin.bottom.heading)}>
          {t('sections.network')}
        </h2>
        <div class={layout.grid.cards}>
          {/* Network info cards - hide when in WiFi mode without WiFi connection */}
          {/* Prevents showing wired interface data when user selected WiFi mode */}
          {(!isWifi || cards.wifi) && (
            <>
              <NetworkCard
                data={cards.dhcp}
                publicip={cards.publicip}
                loading={loading}
                showPublicIp={displayOptions.showPublicIp}
              />
              <GatewayCard data={cards.gateway} loading={loading} />
              <DnsCard data={cards.dns} loading={loading} />
              {/* Public IP Card - shows geolocation, ISP/ASN, and IP history */}
              <PublicIpCard data={cards.publicip} loading={loading} />
            </>
          )}
        </div>
      </section>

      {/* Section: Testing & Discovery - cards differ by interface type */}
      <section aria-labelledby="performance-heading" class={spacing.margin.bottom.section}>
        <h2 id="performance-heading" class={cn('section-title', spacing.margin.bottom.heading)}>
          {t('sections.testingDiscovery')}
        </h2>
        <div class={layout.grid.cards}>
          {/* Test cards - only show when connected to the selected interface type */}
          {/* Fix: Don't show test results from wired when in WiFi mode but disconnected */}
          {(!isWifi || cards.wifi) && (
            <>
              <HealthCheckCard loading={loading} />
              {/* #397: Guest Network isolation audit. Self-hides when disabled. */}
              <GuestNetworkAuditCard />
              {/* SLA Dashboard - aggregates health scores, SLA compliance, and alerts */}
              <SLADashboardCard />
              {cardSettings.performance.enabled ? (
                <PerformanceCard
                  loading={loading}
                  runSpeedtestEnabled={
                    cardSettings.performance.speedtest.enabled &&
                    cardSettings.performance.speedtest.autoRunOnLink
                  }
                  runIperfEnabled={
                    cardSettings.performance.iperf.enabled &&
                    cardSettings.performance.iperf.autoRunOnLink
                  }
                />
              ) : null}
            </>
          )}

          {/* Ethernet-only: Network Discovery (ARP/LLDP/SNMP) */}
          {!isWifi && cardSettings.networkDiscovery.enabled && (
            <NetworkDiscoveryCard
              data={networkDiscovery}
              loading={loading}
              onScan={triggerDeviceScan}
            />
          )}

          {/* Path Discovery - only show when connected */}
          {(!isWifi || cards.wifi) && (
            <PathDiscoveryCard
              gateway={cards.gateway?.gateway}
              dnsServer={cards.dns?.servers?.[0]?.address}
              onRegisterTraceHandler={registerTraceHopHandler}
            />
          )}

          {/* WiFi-only: WiFi Survey for heatmaps and site surveys */}
          {/* Fix #572: Pass current interface to avoid hardcoded "wlan0" */}
          {isWifi ? <WiFiSurveyCard isWifi={isWifi} currentInterface={currentInterface} /> : null}

          {/* WiFi-only: Channel Graph for visualizing channel overlap */}
          {isWifi ? (
            <WifiChannelGraph
              data={channelGraphData}
              loading={channelGraphLoading}
              visible={isWifi}
            />
          ) : null}
        </div>
      </section>

      {/* Section: System */}
      <section aria-labelledby="system-heading" class={spacing.margin.bottom.section}>
        <h2 id="system-heading" class={cn('section-title', spacing.margin.bottom.heading)}>
          {t('sections.system')}
        </h2>
        <div class={layout.grid.cards}>
          <SystemHealthCard />
          <LogViewerCard maxHeight="400px" />
        </div>
      </section>
    </>
  );
}
