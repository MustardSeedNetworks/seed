/**
 * L2 path display for PathDiscoveryCard.
 *
 * Renders the L2 switch hop chain discovered via LLDP/CDP/SNMP,
 * including a visual diagram and per-hop port detail expansion.
 */

import type React from 'react';
import { memo, useCallback } from 'react';
import { cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';
import type { L2Hop, L2PathResult } from '../../types';
import { ChevronDown, ChevronUp } from '../ui/icons';
import { getSourceColor } from './pathDiscoveryHelpers';

interface L2PathDisplayProps {
  result: L2PathResult;
  expandedHop: number | null;
  onToggleHop: (index: number | null) => void;
  t: (key: string, fallback: string) => string;
}

export const L2_PATH_DISPLAY: React.NamedExoticComponent<L2PathDisplayProps> = memo(
  function l2PathDisplay({
    result,
    expandedHop,
    onToggleHop,
    t,
  }: L2PathDisplayProps): React.ReactElement {
    const toggleHop = useCallback(
      (index: number) => {
        onToggleHop(expandedHop === index ? null : index);
      },
      [expandedHop, onToggleHop],
    );

    if (result.hops.length === 0) {
      return (
        <div class="stack-sm">
          <div class="body-small font-semibold text-brand-primary">
            L2 {t('pathDiscovery.path', 'Path')}
          </div>
          <div class={cn(spacing.pad.sm, 'bg-surface-base', radius.default)}>
            <span class="caption text-text-muted">
              {t('pathDiscovery.noL2Path', 'No L2 path information available')}
            </span>
          </div>
        </div>
      );
    }

    return (
      <div class="stack-sm">
        {/* L2 Header */}
        <div class={cn(layout.flex.between, 'items-center')}>
          <div>
            <span class="body-small font-semibold text-brand-primary">
              L2 {t('pathDiscovery.path', 'Path')}
            </span>
            <span class="caption text-text-muted ml-2">(via LLDP/CDP/SNMP)</span>
          </div>
          <span class="caption text-text-muted">
            {result.hops.length} {t('pathDiscovery.switches', 'switches')}
          </span>
        </div>

        {/* Visual Path Diagram */}
        <div
          class={cn(
            'flex items-center overflow-x-auto',
            spacing.pad.sm,
            'bg-surface-base',
            radius.default,
            'border border-surface-border',
          )}
        >
          {result.hops.map((hop, hopIndex) => (
            <div
              key={`${hop.deviceIp}-${hop.ingressPort?.name || 'start'}`}
              class="flex items-center shrink-0"
            >
              {/* Switch Box */}
              <div
                class={cn(
                  'flex flex-col items-center',
                  spacing.pad.sm,
                  'bg-surface-raised',
                  radius.md,
                  'border border-surface-border',
                  'min-w-28',
                )}
              >
                <span class="caption font-semibold text-text-primary truncate max-w-24">
                  {hop.device || hop.deviceIp}
                </span>
                <span class="caption text-text-muted">{hop.deviceIp}</span>
                <span class={cn('caption', getSourceColor(hop.source))}>
                  {hop.source.toUpperCase()}
                </span>
              </div>

              {/* Arrow with port names */}
              {hopIndex < result.hops.length - 1 ? (
                <div class="flex items-center mx-2">
                  <div class="flex flex-col items-end mr-1">
                    {hop.egressPort ? (
                      <span class="caption text-text-muted">{hop.egressPort.name}</span>
                    ) : null}
                  </div>
                  <div class="w-8 h-0.5 bg-brand-primary relative">
                    <div
                      class="absolute right-0 top-1/2 -translate-y-1/2 w-0 h-0"
                      style={{
                        borderTop: '4px solid transparent',
                        borderBottom: '4px solid transparent',
                        borderLeft: '6px solid var(--brand-primary)',
                      }}
                    />
                  </div>
                  <div class="flex flex-col items-start ml-1">
                    {result.hops[hopIndex + 1]?.ingressPort ? (
                      <span class="caption text-text-muted">
                        {result.hops[hopIndex + 1]?.ingressPort?.name}
                      </span>
                    ) : null}
                  </div>
                </div>
              ) : null}
            </div>
          ))}
        </div>

        {/* Detailed Port Information */}
        <div class="stack-xs">
          {result.hops.map((hop, index) => (
            <L2_HOP_DETAIL
              key={`${hop.deviceIp}-${hop.ingressPort?.name || 'start'}-detail`}
              hop={hop}
              index={index}
              isExpanded={expandedHop === index}
              onToggle={(): void => toggleHop(index)}
              t={t}
            />
          ))}
        </div>
      </div>
    );
  },
);

interface L2HopDetailProps {
  hop: L2Hop;
  index: number;
  isExpanded: boolean;
  onToggle: () => void;
  t: (key: string, fallback: string) => string;
}

const L2_HOP_DETAIL: React.NamedExoticComponent<L2HopDetailProps> = memo(function l2HopDetail({
  hop,
  isExpanded,
  onToggle,
  t,
}: L2HopDetailProps): React.ReactElement {
  return (
    <div class={cn('border border-surface-border', radius.default, 'overflow-hidden')}>
      {/* Header */}
      <button
        type="button"
        onClick={onToggle}
        class={cn(
          'w-full flex items-center justify-between',
          spacing.pad.sm,
          'bg-surface-raised hover:bg-surface-hover transition-colors',
          'text-left',
        )}
      >
        <div class="flex items-center gap-2">
          <span class="body-small font-medium text-text-primary">{hop.device || hop.deviceIp}</span>
          <span class="caption text-text-muted">({hop.deviceIp})</span>
        </div>
        {isExpanded ? (
          <ChevronUp class={cn(iconTokens.size.sm, 'text-text-muted')} />
        ) : (
          <ChevronDown class={cn(iconTokens.size.sm, 'text-text-muted')} />
        )}
      </button>

      {/* Expanded Details */}
      {isExpanded ? (
        <div class={cn(spacing.pad.sm, 'bg-surface-base border-t border-surface-border')}>
          <div class="grid grid-cols-2 gap-4">
            {/* Ingress Port */}
            <div>
              <div class="caption font-semibold text-text-muted uppercase tracking-wide mb-2">
                {t('pathDiscovery.ingressPort', 'Ingress Port')}
              </div>
              {hop.ingressPort ? (
                <PORT_DETAILS port={hop.ingressPort} t={t} />
              ) : (
                <span class="caption text-text-muted">---</span>
              )}
            </div>

            {/* Egress Port */}
            <div>
              <div class="caption font-semibold text-text-muted uppercase tracking-wide mb-2">
                {t('pathDiscovery.egressPort', 'Egress Port')}
              </div>
              {hop.egressPort ? (
                <PORT_DETAILS port={hop.egressPort} t={t} />
              ) : (
                <span class="caption text-text-muted">---</span>
              )}
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
});

interface PortDetailsProps {
  port: L2Hop['ingressPort'];
  t: (key: string, fallback: string) => string;
}

const PORT_DETAILS: React.NamedExoticComponent<PortDetailsProps> = memo(function portDetails({
  port,
  t,
}: PortDetailsProps): React.ReactElement | null {
  if (!port) {
    return null;
  }

  return (
    <div class="stack-xs">
      <div class="body-small font-mono text-text-primary">{port.name}</div>
      <div class="flex flex-wrap gap-2">
        {port.speed ? <span class="caption text-text-secondary">{port.speed}</span> : null}
        {port.duplex ? <span class="caption text-text-muted">{port.duplex}</span> : null}
        {port.isTrunk ? (
          <span class="caption text-brand-primary">{t('pathDiscovery.trunk', 'Trunk')}</span>
        ) : null}
      </div>
      {port.vlans && port.vlans.length > 0 ? (
        <div class="caption text-text-muted">
          VLANs: {port.vlans.slice(0, 5).join(', ')}
          {port.vlans.length > 5 ? ` +${port.vlans.length - 5}` : null}
        </div>
      ) : null}
      {port.connectedTo ? (
        <div class="caption text-text-secondary">→ {port.connectedTo}</div>
      ) : null}
    </div>
  );
});
