/**
 * DriverStatsCard — the NIC driver's error counters, and what each one means
 * (#416).
 *
 * A link can be up, negotiated at the right speed, and still dropping frames.
 * These counters say which problem it is, so each row carries its meaning
 * rather than a number the operator has to look up: CRC errors are cabling,
 * receive drops are the host, pause frames are congestion downstream.
 *
 * The meanings come from the backend, next to the counter names they explain,
 * so the API and the card cannot drift.
 *
 * Linux only. <FeatureUnavailable> shows why on the platforms where ethtool has
 * no equivalent, instead of an empty table an operator would read as "no
 * errors" (#750).
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

import { useDriverStats } from '../../hooks/useDriverStats';
import { cn, radius, spacing, status as statusColor } from '../../styles/theme';
import { Button } from '../ui/Button';
import { Card, type Status } from '../ui/card';
import { FeatureUnavailable } from '../ui/FeatureUnavailable';
import { Activity } from '../ui/icons';

interface DriverStatsCardProps {
  interfaceName?: string;
}

export function DriverStatsCard({ interfaceName }: DriverStatsCardProps): JSX.Element {
  const { t } = useTranslation('cards');
  const { counters, total, loading, error, refresh } = useDriverStats(interfaceName);

  // Any non-zero curated counter is worth flagging: every one of them counts
  // something that should be zero on a healthy link.
  const unhealthy = counters.some((counter) => counter.value > 0);

  let cardStatus: Status = 'success';
  if (loading) {
    cardStatus = 'loading';
  } else if (error) {
    cardStatus = 'error';
  } else if (unhealthy) {
    cardStatus = 'warning';
  }

  return (
    <FeatureUnavailable capability="driver_statistics">
      <Card
        title={t('driverStats.title')}
        icon={<Activity className="w-4 h-4" />}
        status={cardStatus}
        ariaLabel={t('driverStats.title')}
      >
        <div className="stack-sm">
          <p className="caption text-text-muted">{t('driverStats.description')}</p>

          {error ? <p className="caption text-status-error">{error}</p> : null}

          {!loading && !error && counters.length === 0 ? (
            <p className="caption text-text-muted">{t('driverStats.noCounters')}</p>
          ) : null}

          {counters.map((counter) => (
            <div
              key={counter.key}
              className={cn(
                spacing.pad.xs,
                'bg-surface-base',
                radius.default,
                'border border-surface-border',
              )}
            >
              <div className="flex-between">
                <span className="body-small text-text-primary">{counter.label}</span>
                <span
                  className={cn(
                    'body-small font-mono',
                    counter.value > 0 ? statusColor.text.warning : 'text-text-muted',
                  )}
                >
                  {counter.value}
                </span>
              </div>
              <div className="caption text-text-secondary">{counter.meaning}</div>
              {/* The driver's own name, so an operator can correlate with
                  `ethtool -S` output rather than guessing which counter this is. */}
              <div className="caption text-text-muted font-mono">{counter.key}</div>
            </div>
          ))}

          {total > counters.length ? (
            <p className="caption text-text-muted">
              {t('driverStats.curated', { shown: counters.length, total })}
            </p>
          ) : null}

          <Button
            variant="ghost"
            size="sm"
            onClick={(): void => {
              refresh().catch(() => undefined);
            }}
            loading={loading}
            className="self-start"
          >
            {t('driverStats.refresh')}
          </Button>
        </div>
      </Card>
    </FeatureUnavailable>
  );
}
