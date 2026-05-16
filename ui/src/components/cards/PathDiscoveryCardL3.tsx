/**
 * L3 path display for PathDiscoveryCard.
 *
 * Renders the hop list returned by ICMP/UDP/TCP traceroute,
 * including TTL, IP/hostname, RTT, and a scaled latency bar.
 */

import type React from 'react';
import { memo } from 'react';
import { cn, layout, radius, spacing } from '../../styles/theme';
import type { TracerouteResult } from '../../types';
import { formatRtt, getRttBarColor } from './pathDiscoveryHelpers';

interface L3PathDisplayProps {
  result: TracerouteResult;
  maxRtt: number;
  t: (key: string, fallback: string) => string;
}

export const L3_PATH_DISPLAY: React.NamedExoticComponent<L3PathDisplayProps> = memo(
  function l3PathDisplay({ result, maxRtt, t }: L3PathDisplayProps): React.ReactElement {
    return (
      <div class="stack-sm">
        {/* L3 Header */}
        <div class={cn(layout.flex.between, 'items-center')}>
          <div>
            <span class="body-small font-semibold text-brand-primary">
              L3 {t('pathDiscovery.path', 'Path')}
            </span>
            <span class="body-small font-medium text-text-primary ml-2">
              {t('pathDiscovery.to', 'to')} {result.target}
            </span>
            <span class="caption text-text-muted ml-2">
              ({result.hops.length} {t('pathDiscovery.hops', 'hops')})
            </span>
          </div>
          {result.completed ? (
            <span class="caption text-status-success">
              {t('pathDiscovery.completed', 'Completed')}
            </span>
          ) : null}
        </div>

        {/* Hop List */}
        <div class={cn('stack-xs', spacing.margin.top.inline)}>
          {result.hops.map((hop) => (
            <div
              key={hop.ttl}
              class={cn(
                layout.inline.default,
                spacing.gap.compact,
                spacing.pad.xs,
                radius.default,
                hop.state === 'timeout' ? 'bg-surface-base' : 'bg-surface-raised',
                'border border-surface-border',
              )}
            >
              {/* TTL */}
              <span class="w-6 caption font-mono text-text-muted">{hop.ttl}</span>

              {/* IP and Hostname */}
              <div class="flex-1 min-w-0">
                {hop.state === 'timeout' ? (
                  <span class="caption text-text-muted">* * *</span>
                ) : (
                  <>
                    <span class="body-small font-mono text-text-primary truncate">
                      {hop.ip || '?'}
                    </span>
                    {hop.hostname && hop.hostname !== hop.ip ? (
                      <span class="caption text-text-muted ml-2 truncate">{hop.hostname}</span>
                    ) : null}
                  </>
                )}
              </div>

              {/* RTT */}
              <span
                class={cn(
                  'w-16 text-right caption font-mono',
                  hop.state === 'timeout' ? 'text-text-muted' : 'text-text-primary',
                )}
              >
                {formatRtt(hop.rtt)}
              </span>

              {/* RTT Bar */}
              <div class={cn('w-20 h-2', radius.full, 'bg-surface-border overflow-hidden')}>
                {hop.rtt > 0 ? (
                  <div
                    class={cn('h-full', radius.full, getRttBarColor(hop.state, hop.rtt, maxRtt))}
                    style={{
                      width: `${Math.min(100, (hop.rtt / maxRtt) * 100)}%`,
                    }}
                  />
                ) : null}
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  },
);
