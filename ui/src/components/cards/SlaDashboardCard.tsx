/**
 * SLADashboardCard Component
 *
 * Purpose: Displays SLA compliance metrics, health scores, and alerts for monitored endpoints.
 * Provides an overview of service health and SLA adherence across all configured endpoints.
 *
 * Key Features:
 * - SLA compliance summary: percentage of endpoints meeting SLA targets
 * - Health score distribution: healthy, degraded, critical endpoint counts
 * - Active alerts: count and severity breakdown
 * - Anomaly detection: active anomalies with severity indicators
 * - Period selection: daily, weekly, monthly views
 * - Click-through to detailed endpoint views
 *
 * Usage:
 * ```typescript
 * <SLADashboardCard />
 * ```
 *
 * Dependencies: BaseCard, Card UI components, Icons, theme utilities
 * State: Fetches from SLA and scores API endpoints, updates periodically
 */

import { AlertTriangle, CheckCircle2, Shield, TrendingUp, XCircle } from 'lucide-react';
import type React from 'react';
import { memo, useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, icon as iconTokens, radius, spacing } from '../../styles/theme';
import { Card, CardDivider } from '../ui/Card';
import type { Status } from '../ui/StatusBadge';
import { StatusBadge } from '../ui/StatusBadge';

// biome-ignore lint/style/useNamingConvention: SLA is a standard acronym
interface SLASummary {
  period: string;
  periodStart: string;
  periodEnd: string;
  totalEndpoints: number;
  endpointsMet: number;
  endpointsMissed: number;
  complianceRate: number;
  generatedAt: string;
}

interface ScoresSummary {
  totalEndpoints: number;
  healthy: number;
  degraded: number;
  critical: number;
  unknown: number;
}

interface AlertStats {
  total: number;
  active: number;
  acknowledged: number;
  resolved: number;
  critical: number;
  warning: number;
  info: number;
}

interface DashboardData {
  sla: SLASummary | null;
  scores: ScoresSummary | null;
  alerts: AlertStats | null;
  anomalyCount: number;
}

// biome-ignore lint/style/useNamingConvention: SLA is a standard acronym
interface SLADashboardCardProps {
  className?: string;
}

function getComplianceStatus(rate: number): Status {
  if (rate >= 99) {
    return 'success';
  }
  if (rate >= 95) {
    return 'warning';
  }
  return 'error';
}

/** Helper to get stroke color based on status */
function getStrokeColor(status: Status): string {
  if (status === 'success') {
    return 'var(--color-status-success)';
  }
  if (status === 'warning') {
    return 'var(--color-status-warning)';
  }
  return 'var(--color-status-error)';
}

function _complianceRing({ rate, size = 80 }: { rate: number; size?: number }): React.ReactElement {
  const strokeWidth = 8;
  const normalizedRadius = (size - strokeWidth) / 2;
  const circumference = normalizedRadius * 2 * Math.PI;
  const strokeDashoffset = circumference - (rate / 100) * circumference;

  const status = getComplianceStatus(rate);
  const strokeColor = getStrokeColor(status);

  return (
    <div class="relative inline-flex items-center justify-center">
      <svg
        height={size}
        width={size}
        class="-rotate-90"
        role="img"
        aria-labelledby="compliance-title"
      >
        <title id="compliance-title">SLA Compliance {rate.toFixed(1)}%</title>
        <circle
          stroke="var(--color-border)"
          fill="transparent"
          strokeWidth={strokeWidth}
          r={normalizedRadius}
          cx={size / 2}
          cy={size / 2}
        />
        <circle
          stroke={strokeColor}
          fill="transparent"
          strokeWidth={strokeWidth}
          strokeDasharray={`${circumference} ${circumference}`}
          style={{ strokeDashoffset }}
          strokeLinecap="round"
          r={normalizedRadius}
          cx={size / 2}
          cy={size / 2}
          class="transition-all duration-500"
        />
      </svg>
      <div class="absolute inset-0 flex items-center justify-center">
        <span class="text-lg font-bold text-text-primary">{rate.toFixed(1)}%</span>
      </div>
    </div>
  );
}

function _statBlock({
  icon: ICON,
  label,
  value,
  status,
  className,
}: {
  icon: React.ElementType;
  label: string;
  value: number | string;
  status?: Status;
  className?: string;
}): React.ReactElement {
  return (
    <div class={cn('flex items-center gap-2', className)}>
      <div
        class={cn(
          'flex items-center justify-center rounded-md',
          radius.md,
          spacing.p2,
          status === 'success' && 'bg-status-success/10 text-status-success',
          status === 'warning' && 'bg-status-warning/10 text-status-warning',
          status === 'error' && 'bg-status-error/10 text-status-error',
          !status && 'bg-surface-secondary text-text-muted',
        )}
      >
        <ICON class={iconTokens.sm} />
      </div>
      <div class="flex flex-col">
        <span class="text-xs text-text-muted">{label}</span>
        <span class="text-sm font-medium text-text-primary">{value}</span>
      </div>
    </div>
  );
}

// biome-ignore lint/style/useNamingConvention: SLA is a standard acronym
export const SLADashboardCard: React.NamedExoticComponent<SLADashboardCardProps> = memo(
  // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Dashboard cards require multiple conditional UI sections
  function slaDashboardCardInner({ className }: SLADashboardCardProps): React.ReactElement {
    const { t } = useTranslation('cards');
    const [data, setData] = useState<DashboardData>({
      sla: null,
      scores: null,
      alerts: null,
      anomalyCount: 0,
    });
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [period, setPeriod] = useState<'daily' | 'weekly' | 'monthly'>('daily');

    const fetchData = useCallback(async () => {
      setLoading(true);
      setError(null);

      try {
        const [slaRes, scoresRes, alertsRes, anomaliesRes] = await Promise.all([
          fetch(`/api/v1/sap/health-checks/sla?period=${period}`, { credentials: 'include' }),
          fetch('/api/v1/sap/health-checks/scores', { credentials: 'include' }),
          fetch('/api/v1/sap/health-checks/alerts', { credentials: 'include' }),
          fetch('/api/v1/sap/health-checks/anomalies', { credentials: 'include' }),
        ]);

        const newData: DashboardData = {
          sla: null,
          scores: null,
          alerts: null,
          anomalyCount: 0,
        };

        if (slaRes.ok) {
          // biome-ignore lint/nursery/useAwaitThenable: response.json() returns a Promise
          newData.sla = await slaRes.json();
        }
        if (scoresRes.ok) {
          // biome-ignore lint/nursery/useAwaitThenable: response.json() returns a Promise
          const scoresData = await scoresRes.json();
          newData.scores = scoresData.summary;
        }
        if (alertsRes.ok) {
          // biome-ignore lint/nursery/useAwaitThenable: response.json() returns a Promise
          const alertsData = await alertsRes.json();
          newData.alerts = alertsData.stats;
        }
        if (anomaliesRes.ok) {
          // biome-ignore lint/nursery/useAwaitThenable: response.json() returns a Promise
          const anomaliesData = await anomaliesRes.json();
          newData.anomalyCount = anomaliesData.activeCount ?? 0;
        }

        setData(newData);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load SLA data');
      } finally {
        setLoading(false);
      }
    }, [period]);

    useEffect(() => {
      fetchData().catch(() => undefined);
      // Refresh every 60 seconds
      const interval = setInterval(() => {
        fetchData().catch(() => undefined);
      }, 60000);
      return (): void => clearInterval(interval);
    }, [fetchData]);

    const overallStatus = (): Status => {
      if (loading) {
        return 'loading';
      }
      if (error) {
        return 'error';
      }
      if (!data.scores) {
        return 'unknown';
      }

      const { healthy, degraded, critical, totalEndpoints } = data.scores;
      if (critical > 0) {
        return 'error';
      }
      if (degraded > 0) {
        return 'warning';
      }
      if (healthy === totalEndpoints && totalEndpoints > 0) {
        return 'success';
      }
      return 'unknown';
    };

    return (
      <Card
        title={t('slaDashboard.title', 'SLA Dashboard')}
        subtitle={t('slaDashboard.subtitle', 'Service health and compliance')}
        icon={<Shield class={iconTokens.md} />}
        status={overallStatus()}
        class={className}
      >
        {loading ? (
          <div class={cn('animate-pulse space-y-4', spacing.p4)}>
            <div class="h-20 bg-surface-secondary rounded-lg" />
            <div class="h-16 bg-surface-secondary rounded-lg" />
          </div>
        ) : null}

        {error ? <div class={cn('text-center text-status-error', spacing.p4)}>{error}</div> : null}

        {loading || error ? null : (
          <div class={cn('space-y-4', spacing.p4)}>
            {/* Period selector */}
            <div class="flex justify-end gap-1">
              {(['daily', 'weekly', 'monthly'] as const).map((p) => (
                <button
                  type="button"
                  key={p}
                  onClick={(): void => setPeriod(p)}
                  class={cn(
                    'px-2 py-1 text-xs rounded transition-colors',
                    period === p
                      ? 'bg-accent text-white'
                      : 'bg-surface-secondary text-text-muted hover:text-text-primary',
                  )}
                >
                  {t(`slaDashboard.period.${p}`, p.charAt(0).toUpperCase() + p.slice(1))}
                </button>
              ))}
            </div>

            {/* SLA Compliance */}
            {data.sla ? (
              <div class="flex items-center justify-between">
                <div>
                  <h4 class="text-sm font-medium text-text-primary mb-1">
                    {t('slaDashboard.compliance', 'SLA Compliance')}
                  </h4>
                  <p class="text-xs text-text-muted">
                    {data.sla.endpointsMet} / {data.sla.totalEndpoints}{' '}
                    {t('slaDashboard.endpointsMet', 'endpoints meeting SLA')}
                  </p>
                  {data.sla.endpointsMissed > 0 ? (
                    <p class="text-xs text-status-error mt-1">
                      {data.sla.endpointsMissed}{' '}
                      {t('slaDashboard.endpointsMissed', 'endpoints missing SLA')}
                    </p>
                  ) : null}
                </div>
                <complianceRing rate={data.sla.complianceRate} />
              </div>
            ) : null}

            <CardDivider />

            {/* Health Score Distribution */}
            {data.scores ? (
              <div>
                <h4 class="text-sm font-medium text-text-primary mb-3">
                  {t('slaDashboard.healthScores', 'Health Scores')}
                </h4>
                <div class="grid grid-cols-2 gap-3">
                  <statBlock
                    icon={CheckCircle2}
                    label={t('slaDashboard.healthy', 'Healthy')}
                    value={data.scores.healthy}
                    status="success"
                  />
                  <statBlock
                    icon={AlertTriangle}
                    label={t('slaDashboard.degraded', 'Degraded')}
                    value={data.scores.degraded}
                    status={data.scores.degraded > 0 ? 'warning' : undefined}
                  />
                  <statBlock
                    icon={XCircle}
                    label={t('slaDashboard.critical', 'Critical')}
                    value={data.scores.critical}
                    status={data.scores.critical > 0 ? 'error' : undefined}
                  />
                  <statBlock
                    icon={TrendingUp}
                    label={t('slaDashboard.total', 'Total')}
                    value={data.scores.totalEndpoints}
                  />
                </div>
              </div>
            ) : null}

            <CardDivider />

            {/* Alerts and Anomalies */}
            <div class="grid grid-cols-2 gap-4">
              {data.alerts ? (
                <div>
                  <h4 class="text-xs text-text-muted mb-2">
                    {t('slaDashboard.activeAlerts', 'Active Alerts')}
                  </h4>
                  <div class="flex items-center gap-2">
                    <span
                      class={cn(
                        'text-2xl font-bold',
                        data.alerts.active > 0 ? 'text-status-error' : 'text-text-primary',
                      )}
                    >
                      {data.alerts.active}
                    </span>
                    {data.alerts.critical > 0 ? (
                      <StatusBadge status="error" size="sm">
                        {data.alerts.critical} {t('slaDashboard.criticalLabel', 'critical')}
                      </StatusBadge>
                    ) : null}
                  </div>
                </div>
              ) : null}

              <div>
                <h4 class="text-xs text-text-muted mb-2">
                  {t('slaDashboard.anomalies', 'Anomalies')}
                </h4>
                <div class="flex items-center gap-2">
                  <span
                    class={cn(
                      'text-2xl font-bold',
                      data.anomalyCount > 0 ? 'text-status-warning' : 'text-text-primary',
                    )}
                  >
                    {data.anomalyCount}
                  </span>
                  {data.anomalyCount > 0 ? (
                    <StatusBadge status="warning" size="sm">
                      {t('slaDashboard.detected', 'detected')}
                    </StatusBadge>
                  ) : null}
                </div>
              </div>
            </div>
          </div>
        )}
      </Card>
    );
  },
);
