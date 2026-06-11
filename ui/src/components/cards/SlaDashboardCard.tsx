/**
 * SLADashboardCard Component
 *
 * Purpose: Surfaces the active anomalies detected by the active-monitoring probe
 * engine (source=probe, ADR-0025), read from the unified anomaly store
 * (ADR-0021). The legacy SLA/scoring/alerts sections were removed with the dead
 * health_check_results read-path (ADR-0026); SLA/scoring on probe_results, if
 * rebuilt, is a future feature and the component will be renamed then.
 *
 * Key Features:
 * - Active anomaly count with severity-tinted emphasis
 * - Periodic refresh
 *
 * Usage:
 * ```typescript
 * <SLADashboardCard />
 * ```
 *
 * Dependencies: BaseCard, Card UI components, Icons, theme utilities
 * State: Fetches the active anomaly count, updates periodically
 */

import { Shield } from 'lucide-react';
import type React from 'react';
import { memo, useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, icon as iconTokens, radius, spacing, status as statusColor } from '../../styles/theme';
import { Card } from '../ui/card';
import type { Status } from '../ui/StatusBadge';

interface SLADashboardCardProps {
  className?: string;
}

export const SLADashboardCard: React.NamedExoticComponent<SLADashboardCardProps> = memo(
  function slaDashboardCardInner({ className }: SLADashboardCardProps): React.ReactElement {
    const { t } = useTranslation('cards');
    const [anomalyCount, setAnomalyCount] = useState(0);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const fetchData = useCallback(async () => {
      setLoading(true);
      setError(null);
      try {
        const res = await fetch('/api/v1/telemetry/health-checks/anomalies', {
          credentials: 'include',
        });
        if (res.ok) {
          const data = await res.json();
          setAnomalyCount(data.activeCount ?? 0);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load anomalies');
      } finally {
        setLoading(false);
      }
    }, []);

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
      return anomalyCount > 0 ? 'warning' : 'success';
    };

    return (
      <Card
        title={t('slaDashboard.title', 'Active Anomalies')}
        subtitle={t('slaDashboard.subtitle', 'Detected by active monitoring')}
        icon={<Shield className={iconTokens.size.md} />}
        status={overallStatus()}
        className={className}
      >
        {loading ? (
          <div className={cn('animate-pulse stack-lg', spacing.pad.default)}>
            <div className="h-16 bg-surface-hover rounded-lg" />
          </div>
        ) : null}
        {error ? (
          <div className={cn('text-center text-status-error', spacing.pad.default)}>{error}</div>
        ) : null}
        {loading || error ? null : (
          <div className={cn('stack-lg', spacing.pad.default)}>
            <div>
              <h4 className="caption mb-2">{t('slaDashboard.anomalies', 'Anomalies')}</h4>
              <div className="flex items-center gap-compact">
                <span
                  className={cn(
                    'heading-1',
                    anomalyCount > 0 ? statusColor.text.warning : 'text-text-primary',
                  )}
                >
                  {anomalyCount}
                </span>
                {anomalyCount > 0 ? (
                  <span
                    className={cn(
                      'text-xs px-cell py-0.5 bg-status-warning/10 text-status-warning',
                      radius.full,
                    )}
                  >
                    {t('slaDashboard.detected', 'detected')}
                  </span>
                ) : null}
              </div>
            </div>
          </div>
        )}
      </Card>
    );
  },
);
