/**
 * Unified L2+L3 path timeline for PathDiscoveryCard.
 *
 * Renders the whole path as one ordered host -> switches -> routers ->
 * destination vertical timeline: the L2 switch segment (LLDP/CDP/SNMP) first,
 * then the L3 router segment (traceroute), anchored by source and destination
 * endpoints. Each step sits on a connector rail with a layer chip so the two
 * layers read as a single end-to-end path.
 */

import type React from 'react';
import { memo, useCallback } from 'react';
import { cn, icon as iconTokens, radius } from '../../styles/theme';
import type { L2Hop, PathResponse, PortInfo, TracerouteHop } from '../../types';
import { ChevronDown, ChevronUp, Globe, HardDrive, Network, Router } from '../ui/icons';
import { formatRtt, getRttBarColor, getSourceColor } from './pathDiscoveryHelpers';

type Translate = (key: string, fallback: string) => string;

interface PathTimelineProps {
  result: PathResponse;
  maxRtt: number;
  expandedL2Hop: number | null;
  onToggleL2Hop: (index: number | null) => void;
  t: Translate;
}

/** Layer chip — filled for L3, outlined for L2, sharing the roots brand hue. */
const LAYER_CHIP: Record<'l2' | 'l3', string> = {
  l2: 'border border-brand-primary text-brand-primary',
  l3: 'bg-brand-primary text-text-inverse',
};

export const PATH_TIMELINE: React.NamedExoticComponent<PathTimelineProps> = memo(
  function pathTimeline({
    result,
    maxRtt,
    expandedL2Hop,
    onToggleL2Hop,
    t,
  }: PathTimelineProps): React.ReactElement {
    const l2Path = result.l2Path;
    const l2Hops = l2Path?.hops ?? [];
    const l3Hops = result.l3Path?.hops ?? [];
    const destination = result.l3Path?.target ?? '';

    const toggleHop = useCallback(
      (index: number): void => {
        onToggleL2Hop(expandedL2Hop === index ? null : index);
      },
      [expandedL2Hop, onToggleL2Hop],
    );

    // The bottom rail must reach the destination endpoint when one is rendered.
    const lastIsEndpoint = destination !== '';

    return (
      <div data-testid="path-timeline" className="stack-xs">
        {/* Source endpoint */}
        <TIMELINE_ROW isFirst dotKind="endpoint">
          <div className="flex items-center gap-compact">
            <HardDrive className={cn(iconTokens.size.sm, 'text-text-secondary')} />
            <span className="body-small font-medium text-text-primary">
              {t('pathDiscovery.thisDevice', 'This device')}
            </span>
            <span className="caption text-text-muted">{t('pathDiscovery.source', 'Source')}</span>
          </div>
        </TIMELINE_ROW>

        {/* L2 switch segment */}
        {l2Path ? (
          l2Hops.length > 0 ? (
            l2Hops.map((hop, index) => (
              <TIMELINE_ROW
                key={`l2-${hop.deviceIp}-${hop.ingressPort?.name ?? index}`}
                dotKind="l2"
              >
                <L2_TIMELINE_HOP
                  hop={hop}
                  index={index}
                  isExpanded={expandedL2Hop === index}
                  onToggle={(): void => toggleHop(index)}
                  t={t}
                />
              </TIMELINE_ROW>
            ))
          ) : (
            <TIMELINE_ROW dotKind="l2-empty">
              <span data-testid="l2-empty" className="caption text-text-muted">
                {t(
                  'pathDiscovery.noL2Hops',
                  'No L2 switch hops — needs LLDP/CDP neighbors or SNMP',
                )}
              </span>
            </TIMELINE_ROW>
          )
        ) : null}

        {/* L3 router segment */}
        {l3Hops.map((hop) => (
          <TIMELINE_ROW
            key={`l3-${hop.ttl}`}
            dotKind={hop.state === 'timeout' ? 'l3-timeout' : 'l3'}
          >
            <L3_TIMELINE_HOP hop={hop} maxRtt={maxRtt} t={t} />
          </TIMELINE_ROW>
        ))}

        {/* Destination endpoint */}
        {lastIsEndpoint ? (
          <TIMELINE_ROW isLast dotKind="endpoint">
            <div className="flex items-center gap-compact" data-testid="timeline-dest">
              <Globe className={cn(iconTokens.size.sm, 'text-text-secondary')} />
              <span className="body-small font-medium text-text-primary font-mono truncate">
                {destination}
              </span>
              <span className="caption text-text-muted">
                {t('pathDiscovery.destination', 'Destination')}
              </span>
            </div>
          </TIMELINE_ROW>
        ) : null}
      </div>
    );
  },
);

type DotKind = 'endpoint' | 'l2' | 'l2-empty' | 'l3' | 'l3-timeout';

interface TimelineRowProps {
  children: React.ReactNode;
  dotKind: DotKind;
  isFirst?: boolean;
  isLast?: boolean;
}

const DOT_CLASS: Record<DotKind, string> = {
  endpoint: 'h-3 w-3 bg-text-primary',
  l2: 'h-2.5 w-2.5 border-2 border-brand-primary bg-surface-raised',
  'l2-empty': 'h-2 w-2 border border-dashed border-surface-border bg-surface-base',
  l3: 'h-2.5 w-2.5 bg-brand-primary',
  'l3-timeout': 'h-2.5 w-2.5 bg-surface-border',
};

/** One timeline step: a connector rail (line + node dot) plus its content. */
const TIMELINE_ROW: React.NamedExoticComponent<TimelineRowProps> = memo(function timelineRow({
  children,
  dotKind,
  isFirst = false,
  isLast = false,
}: TimelineRowProps): React.ReactElement {
  return (
    <div className="flex items-stretch gap-compact">
      {/* Connector rail */}
      <div className="flex w-4 shrink-0 flex-col items-center">
        <div
          className={cn('w-px flex-none h-2', isFirst ? 'bg-transparent' : 'bg-surface-border')}
        />
        <div className={cn('shrink-0', radius.full, DOT_CLASS[dotKind])} />
        <div className={cn('w-px flex-1', isLast ? 'bg-transparent' : 'bg-surface-border')} />
      </div>
      {/* Content */}
      <div className="min-w-0 flex-1 py-tight">{children}</div>
    </div>
  );
});

interface L3HopProps {
  hop: TracerouteHop;
  maxRtt: number;
  t: Translate;
}

const L3_TIMELINE_HOP: React.NamedExoticComponent<L3HopProps> = memo(function l3TimelineHop({
  hop,
  maxRtt,
  t,
}: L3HopProps): React.ReactElement {
  const isTimeout = hop.state === 'timeout';
  return (
    <div
      data-testid={`l3-hop-${hop.ttl}`}
      className={cn('flex items-center gap-compact', isTimeout && 'opacity-60')}
    >
      <span className={cn('px-1 caption font-semibold', radius.sm, LAYER_CHIP.l3)}>L3</span>
      <Router className={cn(iconTokens.size.sm, 'text-text-muted shrink-0')} />
      <span className="w-6 caption font-mono text-text-muted">{hop.ttl}</span>
      <div className="flex-1 min-w-0">
        {isTimeout ? (
          <span className="caption text-text-muted">* * *</span>
        ) : (
          <>
            <span className="body-small font-mono text-text-primary truncate">{hop.ip || '?'}</span>
            {hop.hostname && hop.hostname !== hop.ip ? (
              <span className="caption text-text-muted ml-inline truncate">{hop.hostname}</span>
            ) : null}
          </>
        )}
      </div>
      <span
        className={cn(
          'w-16 text-right caption font-mono',
          isTimeout ? 'text-text-muted' : 'text-text-primary',
        )}
      >
        {formatRtt(hop.rtt)}
      </span>
      <div
        className={cn('w-16 h-2 hidden sm:block', radius.full, 'bg-surface-border overflow-hidden')}
      >
        {hop.rtt > 0 ? (
          <div
            className={cn('h-full', radius.full, getRttBarColor(hop.state, hop.rtt, maxRtt))}
            style={{ width: `${Math.min(100, (hop.rtt / maxRtt) * 100)}%` }}
          />
        ) : null}
      </div>
      <span className="sr-only">{t('pathDiscovery.layerL3', 'Layer 3 hop')}</span>
    </div>
  );
});

interface L2HopProps {
  hop: L2Hop;
  index: number;
  isExpanded: boolean;
  onToggle: () => void;
  t: Translate;
}

const L2_TIMELINE_HOP: React.NamedExoticComponent<L2HopProps> = memo(function l2TimelineHop({
  hop,
  index,
  isExpanded,
  onToggle,
  t,
}: L2HopProps): React.ReactElement {
  const ports = [hop.ingressPort?.name, hop.egressPort?.name].filter(Boolean).join(' → ');
  return (
    <div
      data-testid={`l2-hop-${index}`}
      className={cn('border border-surface-border', radius.default)}
    >
      <button
        type="button"
        onClick={onToggle}
        className="w-full flex items-center gap-compact px-tight py-tight text-left hover:bg-surface-hover transition-colors"
      >
        <span className={cn('px-1 caption font-semibold', radius.sm, LAYER_CHIP.l2)}>L2</span>
        <Network className={cn(iconTokens.size.sm, 'text-text-muted shrink-0')} />
        <span className="body-small font-medium text-text-primary truncate">
          {hop.device || hop.deviceIp}
        </span>
        <span className="caption text-text-muted font-mono truncate">{hop.deviceIp}</span>
        <span className={cn('caption', getSourceColor(hop.source))}>
          {hop.source.toUpperCase()}
        </span>
        {ports ? (
          <span className="caption text-text-muted font-mono ml-auto truncate">{ports}</span>
        ) : null}
        {isExpanded ? (
          <ChevronUp className={cn(iconTokens.size.sm, 'text-text-muted shrink-0')} />
        ) : (
          <ChevronDown className={cn(iconTokens.size.sm, 'text-text-muted shrink-0')} />
        )}
      </button>
      {isExpanded ? (
        <div className="px-tight pb-tight bg-surface-base border-t border-surface-border">
          <div className="grid grid-cols-2 gap-comfortable pt-tight">
            <PORT_DETAIL
              label={t('pathDiscovery.ingressPort', 'Ingress Port')}
              port={hop.ingressPort}
              t={t}
            />
            <PORT_DETAIL
              label={t('pathDiscovery.egressPort', 'Egress Port')}
              port={hop.egressPort}
              t={t}
            />
          </div>
        </div>
      ) : null}
    </div>
  );
});

interface PortDetailProps {
  label: string;
  port: PortInfo | null;
  t: Translate;
}

const PORT_DETAIL: React.NamedExoticComponent<PortDetailProps> = memo(function portDetail({
  label,
  port,
  t,
}: PortDetailProps): React.ReactElement {
  return (
    <div>
      <div className="caption font-semibold text-text-muted uppercase tracking-wide mb-tight">
        {label}
      </div>
      {port ? (
        <div className="stack-xs">
          <div className="body-small font-mono text-text-primary">{port.name}</div>
          <div className="flex flex-wrap gap-compact">
            {port.speed ? <span className="caption text-text-secondary">{port.speed}</span> : null}
            {port.duplex ? <span className="caption text-text-muted">{port.duplex}</span> : null}
            {port.isTrunk ? (
              <span className="caption text-brand-primary">
                {t('pathDiscovery.trunk', 'Trunk')}
              </span>
            ) : null}
          </div>
          {port.vlans && port.vlans.length > 0 ? (
            <div className="caption text-text-muted">
              VLANs: {port.vlans.slice(0, 5).join(', ')}
              {port.vlans.length > 5 ? ` +${port.vlans.length - 5}` : null}
            </div>
          ) : null}
          {port.connectedTo ? (
            <div className="caption text-text-secondary">
              {'→ '}
              {port.connectedTo}
            </div>
          ) : null}
        </div>
      ) : (
        <span className="caption text-text-muted">---</span>
      )}
    </div>
  );
});
