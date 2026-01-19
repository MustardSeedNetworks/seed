/**
 * DNSCard Component
 *
 * Purpose: Comprehensive DNS diagnostic card showing forward/reverse lookups, DNS server testing,
 * and per-server resolution times. Detects DNS configuration issues and performance problems.
 *
 * Key Features:
 * - Tests forward (hostname→IP) and reverse (IP→hostname) lookups for IPv4 and IPv6
 * - Measures DNS query latency with configurable thresholds (warning/critical)
 * - Per-server testing: tests each configured DNS server independently
 * - Displays resolved IP addresses and error messages for failed lookups
 * - CollapsibleSection for each lookup type with detailed results
 * - Status color-coding based on response times and threshold settings
 *
 * Usage:
 * ```typescript
 * <DNSCard
 *   data={dnsTestData}
 *   loading={isRunning}
 * />
 * ```
 *
 * Dependencies: Card UI components, StatusBadge, CollapsibleSection, useSettings hook, Icons, theme utilities
 * State: Receives test data and thresholds from parent component and settings context
 */

import type React from 'react';
import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { formatTime } from '../../lib/format';
import { cn, icon as iconTokens, layout, spacing } from '../../styles/theme';
import { Card, CardDivider, CardValue, type Status } from '../ui/Card';
import { CollapsibleSection } from '../ui/CollapsibleSection';
import { Globe } from '../ui/Icons';
import { StatusBadge } from '../ui/StatusBadge';

interface LookupResult {
  result: string;
  time: number; // ms
  timeMs: number;
  status: Status;
  error?: string;
  resolved?: string[];
}

interface ServerTestResult {
  server: string;
  forward: LookupResult | null;
  forwardIpv6: LookupResult | null;
  status: Status;
  avgTimeMs: number;
}

export interface DnsData {
  server: string;
  servers?: string[]; // All configured DNS servers
  testHostname: string;
  forward: LookupResult | null;
  forwardIpv6?: LookupResult | null;
  reverse: LookupResult | null;
  reverseIpv6?: LookupResult | null;
  perServerResults?: ServerTestResult[];
}

interface DnsCardProps {
  data: DnsData | null;
  loading?: boolean;
}

function getStatusColorClass(status: string): string {
  if (status === 'success') {
    return 'text-status-success';
  }
  if (status === 'warning') {
    return 'text-status-warning';
  }
  return 'text-status-error';
}

// Helper to get aggregated status from server results (avoids nested ternary)
function getAggregatedStatus(results: ServerTestResult[]): 'error' | 'warning' | 'success' {
  if (results.some((s) => s.status === 'error')) {
    return 'error';
  }
  if (results.some((s) => s.status === 'warning')) {
    return 'warning';
  }
  return 'success';
}

function _lookupRow({
  label,
  lookup,
}: {
  label: string;
  lookup: LookupResult | null | undefined;
}): JSX.Element | null {
  if (!lookup) {
    return null;
  }

  const statusBadge = lookup.status;
  const statusColor = getStatusColorClass(statusBadge);

  return (
    <div class={spacing.margin.bottom.inline}>
      <div class={layout.flex.between}>
        <span class="caption">{label}</span>
        <span class={layout.inline.default}>
          <StatusBadge status={statusBadge} size="sm" />
          <span class={cn('caption font-medium', statusColor)}>
            {formatTime(lookup.timeMs || lookup.time)}
          </span>
        </span>
      </div>
      <p class="body-small truncate" title={lookup.result}>
        {lookup.result}
      </p>
    </div>
  );
}

export const DnsCard: React.MemoExoticComponent<(props: DnsCardProps) => JSX.Element> = memo(
  function dnsCard({ data, loading }: DnsCardProps): JSX.Element {
    const { t } = useTranslation('cards');

    if (loading) {
      return (
        <Card title={t('dns.title')} icon={<Globe class={iconTokens.size.md} />} status="loading">
          <CardValue value={t('dns.testing')} size="lg" />
        </Card>
      );
    }

    if (!data) {
      return (
        <Card title={t('dns.title')} icon={<Globe class={iconTokens.size.md} />} status="unknown">
          <CardValue value={t('dns.noData')} size="md" />
        </Card>
      );
    }

    // Determine overall status based on forward/reverse lookups
    let overallStatus: Status = 'success';
    const lookups = [data.forward, data.forwardIpv6, data.reverse, data.reverseIpv6];
    if (lookups.some((l) => l?.status === 'error')) {
      overallStatus = 'error';
    } else if (lookups.some((l) => l?.status === 'warning')) {
      overallStatus = 'warning';
    }

    // Show all DNS servers if available
    const servers = data.servers && data.servers.length > 0 ? data.servers : [data.server];

    return (
      <Card
        title={t('dns.title')}
        icon={<Globe class={iconTokens.size.md} />}
        status={overallStatus}
      >
        {/* DNS Servers */}
        <div class={spacing.margin.bottom.inline}>
          <p class={cn('caption', spacing.margin.bottom.tight)}>{t('dns.dnsServers')}</p>
          <div class="stack-xs">
            {servers.map((server) => (
              <p key={server} class="body-small font-mono break-all" title={server}>
                {server}
              </p>
            ))}
          </div>
        </div>

        <p class="caption">{t('dns.testingHost', { hostname: data.testHostname })}</p>
        <CardDivider />

        {/* IPv4 Lookups */}
        {data.forward || data.reverse ? (
          <div class={spacing.margin.bottom.inline}>
            <p class={cn('caption font-medium', spacing.margin.bottom.tight)}>IPv4</p>
            <lookupRow label={t('dns.forwardA')} lookup={data.forward} />
            <lookupRow label={t('dns.reversePTR')} lookup={data.reverse} />
          </div>
        ) : null}

        {/* IPv6 Lookups */}
        {data.forwardIpv6 || data.reverseIpv6 ? (
          <>
            <CardDivider />
            <div>
              <p class={cn('caption font-medium', spacing.margin.bottom.tight)}>IPv6</p>
              <lookupRow label={t('dns.forwardAAAA')} lookup={data.forwardIpv6} />
              <lookupRow label={t('dns.reversePTR')} lookup={data.reverseIpv6} />
            </div>
          </>
        ) : null}

        {/* Per-Server Results (collapsible) */}
        {data.perServerResults && data.perServerResults.length > 0 && (
          <>
            <CardDivider />
            <CollapsibleSection
              title={t('dns.serverTests')}
              count={data.perServerResults.length}
              variant="compact"
              status={getAggregatedStatus(data.perServerResults)}
            >
              {data.perServerResults.map((server) => (
                <div key={server.server} class={spacing.chip.sm}>
                  <div class={cn(layout.flex.between, spacing.margin.bottom.tight)}>
                    <span class="caption font-mono">{server.server}</span>
                    <span class={cn('caption font-medium', getStatusColorClass(server.status))}>
                      {formatTime(server.avgTimeMs)}
                    </span>
                  </div>
                  {server.forward ? (
                    <div class={cn(layout.flex.between, 'caption')}>
                      <span>A</span>
                      <span class={layout.inline.default}>
                        <StatusBadge status={server.forward.status} size="sm" />
                        <span class={getStatusColorClass(server.forward.status)}>
                          {server.forward.result === 'No A record'
                            ? 'N/A'
                            : formatTime(server.forward.timeMs)}
                        </span>
                      </span>
                    </div>
                  ) : null}
                  {server.forwardIpv6 ? (
                    <div class={cn(layout.flex.between, 'caption')}>
                      <span>AAAA</span>
                      <span class={layout.inline.default}>
                        <StatusBadge status={server.forwardIpv6.status} size="sm" />
                        <span class={getStatusColorClass(server.forwardIpv6.status)}>
                          {server.forwardIpv6.result === 'No AAAA record'
                            ? 'N/A'
                            : formatTime(server.forwardIpv6.timeMs)}
                        </span>
                      </span>
                    </div>
                  ) : null}
                </div>
              ))}
            </CollapsibleSection>
          </>
        )}
      </Card>
    );
  },
);
