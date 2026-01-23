/**
 * GatewayCard Component
 *
 * Purpose: Monitors network gateway (default router) reachability via ping (ICMP).
 * Displays packet loss, latency statistics, and connection stability.
 *
 * Key Features:
 * - IPv4 and IPv6 gateway monitoring (dual-stack support)
 * - Latency statistics: min/max/avg time and last packet latency
 * - Packet loss percentage with color-coded status
 * - Latency thresholds from settings (warning/critical levels)
 * - Status derivation based on packet loss and latency
 * - Separate sections for IPv4 and IPv6 results (if available)
 *
 * Usage:
 * ```typescript
 * <GatewayCard
 *   data={gatewayData}
 *   loading={isPinging}
 * />
 * ```
 *
 * Dependencies: Card UI components, StatusBadge, useSettings hook, Router icon, theme utilities
 * State: Uses SettingsContext for threshold configuration, receives data from parent
 */

import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import { useSettings } from '../../contexts/useSettings';
import { formatTime, isValidNumber } from '../../lib/format';
import { cn, icon as iconTokens, layout, spacing } from '../../styles/theme';
import { Card, CardDivider, CardRow, CardValue, type Status } from '../ui/card';
import { Router } from '../ui/icons';
import { StatusBadge } from '../ui/StatusBadge';

export interface GatewayData {
  gateway: string;
  reachable: boolean;
  sent: number;
  received: number;
  lossPercent: number;
  minTime: number;
  maxTime: number;
  avgTime: number;
  lastTime: number;
  status: string;
  ipv6?: GatewayData;
}

interface GatewayCardProps {
  data: GatewayData | null;
  loading?: boolean;
}

function getLatencyStatus(
  value: number | undefined | null,
  thresholds: { warning: number; critical: number },
): Status {
  if (!isValidNumber(value)) {
    return 'unknown';
  }
  if (value >= thresholds.critical) {
    return 'error';
  }
  if (value >= thresholds.warning) {
    return 'warning';
  }
  return 'success';
}

// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Gateway card handles IPv4/IPv6 with multiple status conditions
export const GatewayCard: React.FC<GatewayCardProps> = memo(function gatewayCard({
  data,
  loading,
}: GatewayCardProps) {
  const { t: tr } = useTranslation('cards');
  const { thresholds } = useSettings();
  // Map context ThresholdPair (good/warning) to card format (warning/critical)
  // Use defaults if thresholds not yet loaded
  const th = {
    warning: thresholds?.gateway?.good ?? 20,
    critical: thresholds?.gateway?.warning ?? 50,
  };

  const getLatencyClassName = (time: number): string => {
    const status = getLatencyStatus(time, th);
    if (status === 'success') {
      return 'text-status-success';
    }
    if (status === 'warning') {
      return 'text-status-warning';
    }
    return 'text-status-error';
  };

  const getPacketLossStatus = (lossPercent: number): 'success' | 'warning' | 'error' => {
    if (lossPercent === 0) {
      return 'success';
    }
    if (lossPercent < 50) {
      return 'warning';
    }
    return 'error';
  };

  if (loading) {
    return (
      <Card
        title={tr('gateway.title')}
        icon={<Router class={iconTokens.size.md} />}
        status="loading"
      >
        <CardValue value={tr('gateway.pinging')} size="lg" />
      </Card>
    );
  }

  // Check if no gateways detected (neither IPv4 nor IPv6)
  const hasIpv4Gateway = data?.gateway;
  const hasIpv6Gateway = data?.ipv6?.gateway;

  if (!(data && (hasIpv4Gateway || hasIpv6Gateway))) {
    return (
      <Card
        title={tr('gateway.title')}
        icon={<Router class={iconTokens.size.md} />}
        status="unknown"
      >
        <CardValue value={tr('gateway.noGateway')} size="md" />
        <p class={cn('caption', spacing.margin.top.tight)}>{tr('gateway.unableToDetect')}</p>
      </Card>
    );
  }

  // Map API status to card status
  let status: Status = 'unknown';
  switch (data.status) {
    case 'success':
      status = 'success';
      break;
    case 'warning':
      status = 'warning';
      break;
    case 'error':
      status = 'error';
      break;
    default:
      status = data.reachable ? getLatencyStatus(data.avgTime, th) : 'error';
  }

  return (
    <Card title={tr('gateway.title')} icon={<Router class={iconTokens.size.md} />} status={status}>
      <div class={layout.flex.between}>
        <CardValue value={data.gateway} size="lg" />
        <StatusBadge status={data.reachable ? 'success' : 'error'} size="sm" />
      </div>
      <CardDivider />

      {/* Latency stats */}
      <div class={cn('grid grid-cols-3', spacing.gap.compact, spacing.margin.bottom.inline)}>
        <div class="text-center">
          <p class="caption">{tr('gateway.min')}</p>
          <p
            class={cn(
              'body-small font-medium',
              data.minTime > 0 ? getLatencyClassName(data.minTime) : 'text-text-muted',
            )}
          >
            {data.minTime > 0 ? formatTime(data.minTime) : '-'}
          </p>
        </div>
        <div class="text-center">
          <p class="caption">{tr('gateway.avg')}</p>
          <p
            class={cn(
              'body-small font-medium',
              data.avgTime > 0 ? getLatencyClassName(data.avgTime) : 'text-text-muted',
            )}
          >
            {data.avgTime > 0 ? formatTime(data.avgTime) : '-'}
          </p>
        </div>
        <div class="text-center">
          <p class="caption">{tr('gateway.max')}</p>
          <p
            class={cn(
              'body-small font-medium',
              data.maxTime > 0 ? getLatencyClassName(data.maxTime) : 'text-text-muted',
            )}
          >
            {data.maxTime > 0 ? formatTime(data.maxTime) : '-'}
          </p>
        </div>
      </div>

      <CardRow
        label={tr('gateway.packets')}
        value={`${data.received}/${data.sent}`}
        status={getPacketLossStatus(data.lossPercent)}
      />
      {data.lossPercent > 0 && (
        <CardRow
          label={tr('gateway.packetLoss')}
          value={`${Math.round(data.lossPercent)}%`}
          status={data.lossPercent >= 50 ? 'error' : 'warning'}
        />
      )}

      {/* IPv6 Gateway Section */}
      {data.ipv6?.gateway ? (
        <>
          <CardDivider />
          <p class={cn('caption', spacing.margin.bottom.tight, 'font-medium')}>
            {tr('gateway.ipv6Gateway')}
          </p>
          <CardValue value={data.ipv6.gateway} size="md" />
          <p class={cn('caption', spacing.margin.bottom.inline)}>
            {data.ipv6.reachable ? tr('gateway.reachable') : tr('gateway.unreachable')}
          </p>
          <div class={cn('grid grid-cols-3', spacing.gap.compact, spacing.margin.bottom.inline)}>
            <div class="text-center">
              <p class="caption">{tr('gateway.min')}</p>
              <p
                class={cn(
                  'body-small font-medium',
                  data.ipv6.minTime > 0
                    ? getLatencyClassName(data.ipv6.minTime)
                    : 'text-text-muted',
                )}
              >
                {data.ipv6.minTime > 0 ? formatTime(data.ipv6.minTime) : '-'}
              </p>
            </div>
            <div class="text-center">
              <p class="caption">{tr('gateway.avg')}</p>
              <p
                class={cn(
                  'body-small font-medium',
                  data.ipv6.avgTime > 0
                    ? getLatencyClassName(data.ipv6.avgTime)
                    : 'text-text-muted',
                )}
              >
                {data.ipv6.avgTime > 0 ? formatTime(data.ipv6.avgTime) : '-'}
              </p>
            </div>
            <div class="text-center">
              <p class="caption">{tr('gateway.max')}</p>
              <p
                class={cn(
                  'body-small font-medium',
                  data.ipv6.maxTime > 0
                    ? getLatencyClassName(data.ipv6.maxTime)
                    : 'text-text-muted',
                )}
              >
                {data.ipv6.maxTime > 0 ? formatTime(data.ipv6.maxTime) : '-'}
              </p>
            </div>
          </div>
          <CardRow
            label={tr('gateway.packets')}
            value={`${data.ipv6.received}/${data.ipv6.sent}`}
            status={getPacketLossStatus(data.ipv6.lossPercent)}
          />
        </>
      ) : null}
    </Card>
  );
});
