/**
 * HealthCheckCard Component
 *
 * Purpose: Comprehensive health check monitoring for remote services via ping, TCP, UDP, and HTTP(S).
 * Tests end-to-end connectivity and provides detailed per-phase metrics (DNS, TCP, TLS, TTFB).
 *
 * Key Features:
 * - Multi-protocol testing: ICMP ping, TCP connect, UDP, HTTP/HTTPS requests
 * - Extended ping metrics: packet loss, jitter, min/max/avg latency
 * - HTTP timing breakdown: DNS resolution, TCP connection, TLS handshake, Time-To-First-Byte (TTFB)
 * - SSL/TLS certificate monitoring: expiry date, days remaining, issuer, common name, TLS version
 * - Per-test latency thresholds: warning/critical levels from settings
 * - CollapsibleSection for each test type to show detailed results
 * - Status indicators for each phase: DNS, TCP, TLS, TTFB with color-coding
 *
 * Usage:
 * ```typescript
 * <HealthCheckCard
 *   data={healthCheckResults}
 *   loading={isRunning}
 * />
 * ```
 *
 * Dependencies: Card UI components, StatusBadge, CollapsibleSection, Tooltip, useSettings hook,
 *              auth hooks for making secure test requests, Icons, theme utilities
 * State: the last run is React Query data, kept across navigation. A run is rate-limited and
 *        shares its budget with operator actions, so it never runs on mount (#2691).
 */

import { useQuery } from '@tanstack/react-query';
import { memo, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { api } from '../../api';
import { useSettings } from '../../contexts/useSettings';
import { useTestRunSignal, useTestRunStore } from '../../stores/testRunStore';
import {
  cn,
  icon as iconTokens,
  layout,
  radius,
  spacing,
  status as statusColor,
  timing,
} from '../../styles/theme';
import { Card, type Status } from '../ui/Card';
import { CollapsibleSection } from '../ui/CollapsibleSection';
import { HeartPulse } from '../ui/Icons';
import { StatusBadge } from '../ui/StatusBadge';
import { Tooltip } from '../ui/Tooltip';
import { HealthCheckCardProtocolSections } from './HealthCheckCardProtocolSections';
import type { HealthCheckData, StatusValue, TestResult } from './healthCheckCardTypes';
import { failuresFirst, testResultRank } from './healthCheckResultOrder';

/** One cache entry for the last run, shared by every mount of the card. */
const healthCheckRunKey = ['healthChecks', 'run'] as const;

interface HealthCheckCardProps {
  loading?: boolean;
}

export const HealthCheckCard: React.MemoExoticComponent<
  ({ loading }: HealthCheckCardProps) => React.JSX.Element | null
> = memo(function healthCheckCard({ loading }: HealthCheckCardProps): React.JSX.Element | null {
  const { t } = useTranslation('cards');
  const { cardSettings } = useSettings();
  const run = useQuery({
    queryKey: healthCheckRunKey,
    queryFn: () => api.get<HealthCheckData>('/api/v1/telemetry/probes/run'),
    enabled: false,
    staleTime: Number.POSITIVE_INFINITY,
    gcTime: Number.POSITIVE_INFINITY,
    retry: false,
  });
  const data = run.data ?? null;
  const isRunning = run.isFetching;
  const error = run.error ? t('health.failedToRun') : null;
  const fetchTests = async (): Promise<void> => {
    await run.refetch();
  };

  // Listen for settings changes (fired when settings drawer closes after test config changes)
  useEffect((): (() => void) => {
    const handleHealthChecksUpdated = (): void => {
      // Re-run tests with new configuration
      fetchTests().catch((): void => {
        /* Error handled in fetchTests */
      });
    };
    window.addEventListener('healthChecksUpdated', handleHealthChecksUpdated);
    return (): void => {
      window.removeEventListener('healthChecksUpdated', handleHealthChecksUpdated);
    };
  }, [fetchTests]);

  // React to a run start via the testRunStore (was the `window` runAllTests
  // event). Reports completion to the run orchestrator when the fetch settles.
  useTestRunSignal((): void => {
    const handleRunAllTests = async (): Promise<void> => {
      // Check per-card autoRunOnLink setting - skip if health checks disabled.
      // (The orchestrator only awaits 'healthchecks' under the same setting, so
      // an early return here keeps the run accounting consistent.)
      if (!cardSettings.healthChecks.autoRunOnLink) {
        return;
      }

      if (!isRunning) {
        await fetchTests();
        // Signal the run orchestrator that healthchecks are complete.
        useTestRunStore.getState().reportComplete('healthchecks');
      }
    };
    handleRunAllTests().catch((): void => {
      /* Error handled in fetchTests */
    });
  });

  // Hidden once a run finds no probes; shown before the first run so one can start.
  if (data && !(data.hasTests || loading || isRunning)) {
    return null;
  }

  const getStatus = (): Status => {
    if (loading || isRunning) {
      return 'loading';
    }
    if (error) {
      return 'error';
    }
    if (!data) {
      return 'unknown';
    }

    const allResults = [
      ...data.pingResults,
      ...data.tcpResults,
      ...(data.udpResults || []),
      ...data.httpResults,
    ];
    if (allResults.length === 0) {
      return 'unknown';
    }

    // Priority: error > warning > success
    // Any failure (!success) or error status = card is error
    if (
      allResults.some((r) => !r.success || r.testStatus === 'error' || r.certStatus === 'error')
    ) {
      return 'error';
    }

    // Any warning status = card is warning
    if (allResults.some((r) => r.testStatus === 'warning' || r.certStatus === 'warning')) {
      return 'warning';
    }

    // All tests passed with no warnings
    return 'success';
  };

  const formatLatency = (ms: number): string => {
    if (ms >= 1000) {
      return `${(ms / 1000).toFixed(1)}s`;
    }
    return `${Math.round(ms)}ms`;
  };

  // Helper to determine status label for a test result
  const getStatusLabel = (result: TestResult): 'success' | 'warning' | 'error' => {
    if (!result.success) {
      return 'error';
    }
    if (result.testStatus === 'warning') {
      return 'warning';
    }
    return 'success';
  };

  // Helper to get status color class
  const getStatusColor = (statusLabel: 'success' | 'warning' | 'error'): string => {
    if (statusLabel === 'success') {
      return statusColor.text.success;
    }
    if (statusLabel === 'warning') {
      return statusColor.text.warning;
    }
    return statusColor.text.error;
  };

  const renderTestResult = (
    result: TestResult,
    type: 'ping' | 'tcp' | 'udp' | 'http',
  ): React.JSX.Element => {
    // Use testStatus for threshold-based coloring, fall back to success/error
    const statusLabel: 'success' | 'warning' | 'error' = getStatusLabel(result);
    const statusClass: string = getStatusColor(statusLabel);

    // Display name - backend already formats as host:port when name is empty
    // Only add HTTP status code, not ports (already in name)
    const displayName = result.name;
    let details = '';
    if (type === 'http' && result.status) {
      details = ` (${result.status})`;
    }

    // Extended ping info
    const hasExtendedPing = type === 'ping' && result.packetLoss !== undefined;
    const extendedInfo = hasExtendedPing
      ? [
          t('health.packetLoss', { percent: result.packetLoss?.toFixed(0) }),
          result.jitter !== undefined ? t('health.jitter', { ms: result.jitter.toFixed(1) }) : null,
        ]
          .filter(Boolean)
          .join(', ')
      : null;

    return (
      <div key={`${type}-${result.name}`} className={spacing.compact.py}>
        <div className={layout.flex.between}>
          <Tooltip text={displayName}>
            <span className="body-small text-text-muted truncate flex-1">
              {displayName}
              {details}
            </span>
          </Tooltip>
          <span className={cn('inline-flex items-center', spacing.gap.compact)}>
            <StatusBadge status={statusLabel} size="sm" />
            <span className={cn('body-small font-medium', statusClass)}>
              {result.success ? formatLatency(result.latency) : t('health.fail')}
            </span>
          </span>
        </div>
        {extendedInfo ? (
          <div className={cn('caption text-text-muted', spacing.micro.mt)}>{extendedInfo}</div>
        ) : null}
      </div>
    );
  };

  // Timing bar component for HTTP requests
  const TIMING_BAR = ({ result }: { result: TestResult }): React.JSX.Element | null => {
    // Prefer total latency; fall back to sum of phases so we can still render on failures
    const safeNum = (v: number | undefined): number =>
      v !== undefined && Number.isFinite(v) ? v : 0;
    const dns = safeNum(result.dnsLatency);
    const tcp = safeNum(result.tcpConnect);
    const tls = safeNum(result.tlsLatency);
    const ttfb = safeNum(result.ttfbLatency);
    const total =
      result.latency && Number.isFinite(result.latency) && result.latency > 0
        ? result.latency
        : dns + tcp + tls + ttfb;

    // Guard against NaN, Infinity, and zero/negative values
    if (!(total && Number.isFinite(total)) || total <= 0) {
      return null;
    }

    // Download time is what's left after subtracting known phases
    const download = Math.max(0, total - dns - tcp - tls - ttfb);

    // Get status-based text color for legend (bar colors stay fixed for phase identification)
    const getStatusTextColor = (status?: StatusValue): string => {
      if (status === 'error') {
        return statusColor.text.error;
      }
      if (status === 'warning') {
        return statusColor.text.warning;
      }
      return 'text-text-muted';
    };

    // Segment colors are fixed per-phase for consistent identification
    // Using dark mode aware colors from theme
    // Status is indicated only via text color in the legend
    const segments = (
      [
        {
          helpKey: 'health.timingHelp.dns',
          label: t('health.timingDns'),
          value: dns,
          color: timing.dns.bg,
          status: result.dnsStatus,
        },
        {
          helpKey: 'health.timingHelp.tcp',
          label: t('health.timingTcp'),
          value: tcp,
          color: timing.tcp.bg,
          status: result.tcpStatus,
        },
        {
          helpKey: 'health.timingHelp.tls',
          label: t('health.timingTls'),
          value: tls,
          color: timing.tls.bg,
          status: result.tlsStatus,
        },
        {
          helpKey: 'health.timingHelp.wait',
          label: t('health.timingWait'),
          value: ttfb,
          color: timing.wait.bg,
          status: result.ttfbStatus,
        },
        {
          helpKey: 'health.timingHelp.download',
          label: t('health.timingDownload'),
          value: download,
          color: timing.download.bg,
          status: undefined,
        },
      ] as const
    ).filter((s) => s.value > 0 && Number.isFinite(s.value));

    if (segments.length === 0) {
      return null;
    }

    const fmt = (ms: number): string =>
      ms >= 1000 ? `${(ms / 1000).toFixed(1)}s` : `${Math.round(ms)}ms`;

    return (
      <div className={spacing.micro.mtCompactMd}>
        {/* Stacked bar */}
        <div className={cn('h-2', radius.full, 'overflow-hidden flex bg-surface-sunken')}>
          {segments.map((seg, i) => {
            const widthPercent = Math.min(100, Math.max(0, (seg.value / total) * 100));
            return (
              <div
                aria-hidden="true"
                key={seg.label}
                className={cn(
                  'h-full',
                  seg.color,
                  i === 0 ? 'rounded-l-full' : '',
                  i === segments.length - 1 ? 'rounded-r-full' : '',
                )}
                style={{ width: `${widthPercent}%` }}
              />
            );
          })}
        </div>
        {/* Legend with tooltips */}
        <div
          className={cn(
            'flex flex-wrap gap-x-3',
            spacing.margin.top.tight,
            'caption',
            spacing.micro.gap,
          )}
        >
          {segments.map((seg) => (
            <Tooltip key={seg.label} text={t(seg.helpKey)} side="bottom">
              <button
                type="button"
                data-testid="http-timing-segment"
                className={cn(
                  'inline-flex items-center',
                  spacing.gap.tight,
                  getStatusTextColor(seg.status),
                )}
              >
                <span className={cn('inline-block w-2 h-2', radius.full, seg.color)} />
                {seg.label} {fmt(seg.value)}
              </button>
            </Tooltip>
          ))}
        </div>
      </div>
    );
  };

  // Helper to get HTTP result status color
  const getHttpStatusColor = (result: TestResult): string => {
    if (!result.success) {
      return statusColor.text.error;
    }
    if (result.testStatus === 'warning') {
      return statusColor.text.warning;
    }
    if (result.testStatus === 'error') {
      return statusColor.text.error;
    }
    return statusColor.text.success;
  };

  // Helper to determine section status from test results
  const getSectionStatus = (
    results: TestResult[],
    checkCertStatus: boolean = false,
  ): 'success' | 'warning' | 'error' => {
    const hasError = results.some(
      (r) =>
        !r.success || r.testStatus === 'error' || (checkCertStatus && r.certStatus === 'error'),
    );
    if (hasError) {
      return 'error';
    }
    const hasWarning = results.some(
      (r) => r.testStatus === 'warning' || (checkCertStatus && r.certStatus === 'warning'),
    );
    if (hasWarning) {
      return 'warning';
    }
    return 'success';
  };

  // Helper to get cert status color
  const getCertStatusColor = (status?: StatusValue): string => {
    if (status === 'error') {
      return statusColor.text.error;
    }
    if (status === 'warning') {
      return statusColor.text.warning;
    }
    if (status === 'success') {
      return statusColor.text.success;
    }
    return 'text-text-muted';
  };

  const renderHttpResult = (result: TestResult): React.JSX.Element => {
    // Use testStatus for threshold-based coloring
    const statusClass: string = getHttpStatusColor(result);

    // Certificate status coloring
    const certColor: string = getCertStatusColor(result.certStatus);

    const hasCertInfo = result.certDaysLeft !== undefined && result.certDaysLeft >= 0;
    const hasTls = result.tlsVersion && result.tlsVersion !== 'Unknown';

    // Format cert expiry nicely
    const formatCertExpiry = (): string => {
      if (!hasCertInfo || result.certDaysLeft === undefined) {
        return '';
      }
      const days: number = result.certDaysLeft;
      if (days <= 0) {
        return t('health.expired');
      }
      if (days === 1) {
        return t('health.certExpiry1Day');
      }
      if (days < 30) {
        return t('health.certExpiryDays', { days });
      }
      if (days < 365) {
        return t('health.certExpiryMonths', { months: Math.floor(days / 30) });
      }
      return t('health.certExpiryYears', { years: Math.floor(days / 365) });
    };

    // Check if we have timing breakdown data
    const hasTimingData =
      result.dnsLatency !== undefined ||
      result.tcpConnect !== undefined ||
      result.tlsLatency !== undefined ||
      result.ttfbLatency !== undefined;

    return (
      <div key={`http-${result.name}`} className={spacing.compact.pyMd}>
        <div className={layout.flex.between}>
          <Tooltip text={result.name}>
            <span className="body-small text-text-muted truncate flex-1">
              {result.name}
              {result.status ? ` (${result.status})` : ''}
            </span>
          </Tooltip>
          <span className={cn('body-small font-medium', statusClass)}>
            {result.success ? formatLatency(result.latency) : t('health.fail')}
          </span>
        </div>
        {hasTimingData ? <TIMING_BAR result={result} /> : null}
        {!result.success && result.error ? (
          <div className={cn('caption text-status-error', spacing.margin.top.tight)}>
            {result.error}
          </div>
        ) : null}
        {hasTls || hasCertInfo ? (
          <div className={cn('caption', spacing.margin.top.tight, layout.inline.default)}>
            {hasTls ? (
              <span className="text-text-muted whitespace-nowrap">{result.tlsVersion}</span>
            ) : null}
            {hasTls && hasCertInfo ? <span className="text-text-muted">·</span> : null}
            {hasCertInfo ? (
              <Tooltip text={t('health.detail.expires', { date: result.certExpiry })}>
                <span className={cn(certColor, 'whitespace-nowrap')}>{formatCertExpiry()}</span>
              </Tooltip>
            ) : null}
            {result.certIssuer ? (
              <>
                <span className="text-text-muted">·</span>
                <Tooltip text={result.certIssuer}>
                  <span className="text-text-muted truncate min-w-0">{result.certIssuer}</span>
                </Tooltip>
              </>
            ) : null}
          </div>
        ) : null}
      </div>
    );
  };

  return (
    <Card
      title={t('health.title')}
      icon={<HeartPulse className={iconTokens.size.md} />}
      status={getStatus()}
      headerAction={
        <button
          type="button"
          onClick={(): void => {
            fetchTests().catch((): void => {
              /* surfaced through the query error */
            });
          }}
          disabled={isRunning}
          data-testid="health-check-run"
          className={cn(
            spacing.chip.sm,
            'bg-brand-primary text-on-brand',
            radius.md,
            'hover:bg-brand-primary/90 transition-colors font-medium caption disabled:opacity-50 disabled:cursor-not-allowed',
          )}
        >
          {t('health.run')}
        </button>
      }
    >
      {isRunning ? <p className="body-small text-text-muted">{t('health.runningTests')}</p> : null}
      {!(isRunning || data || error) ? (
        <p className="body-small text-text-muted">{t('health.notRun')}</p>
      ) : null}
      {!isRunning && data ? (
        <>
          {/* Ping Results */}
          {data.pingResults && data.pingResults.length > 0 ? (
            <CollapsibleSection
              title={t('health.ping')}
              count={data.pingResults.length}
              variant="compact"
              defaultOpen={true}
              status={getSectionStatus(data.pingResults)}
            >
              {failuresFirst(data.pingResults, testResultRank).map((r) =>
                renderTestResult(r, 'ping'),
              )}
            </CollapsibleSection>
          ) : null}

          {/* TCP Results */}
          {data.tcpResults && data.tcpResults.length > 0 ? (
            <CollapsibleSection
              title={t('health.tcpPorts')}
              count={data.tcpResults.length}
              variant="compact"
              defaultOpen={true}
              status={getSectionStatus(data.tcpResults)}
            >
              {failuresFirst(data.tcpResults, testResultRank).map((r) =>
                renderTestResult(r, 'tcp'),
              )}
            </CollapsibleSection>
          ) : null}

          {/* UDP Results */}
          {data.udpResults && data.udpResults.length > 0 ? (
            <CollapsibleSection
              title={t('health.udpPorts')}
              count={data.udpResults.length}
              variant="compact"
              defaultOpen={true}
              status={getSectionStatus(data.udpResults)}
            >
              {failuresFirst(data.udpResults, testResultRank).map((r) =>
                renderTestResult(r, 'udp'),
              )}
            </CollapsibleSection>
          ) : null}

          {/* HTTP Results */}
          {data.httpResults && data.httpResults.length > 0 ? (
            <CollapsibleSection
              title={t('health.http')}
              count={data.httpResults.length}
              variant="compact"
              defaultOpen={true}
              status={getSectionStatus(data.httpResults, true)}
            >
              {failuresFirst(data.httpResults, testResultRank).map((r) => renderHttpResult(r))}
            </CollapsibleSection>
          ) : null}

          <HealthCheckCardProtocolSections data={data} t={t} />
        </>
      ) : null}
      {error ? <p className="body-small text-status-error">{error}</p> : null}
    </Card>
  );
});
