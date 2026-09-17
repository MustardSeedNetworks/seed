/**
 * TopologyGraph — the visual map of what discovery found.
 *
 * Drawn here rather than by a graph library: the shared map core (niac-go
 * R-D) is not published yet, and the interim the row sanctions is a
 * self-contained graph over /topology/nodes + /topology/links. Plain SVG over
 * a pure layout (topologyLayout.ts) also keeps the whole drawing assertable
 * in a unit test and adds no dependency to seed's bundle.
 *
 * Two rules from the design system decide how it looks:
 *   - Only status is saturated. A node's colour never encodes its role — the
 *     role is the tier it sits in. Selection is the one accent on the map.
 *   - An FDB edge is an inference from a forwarding table, not a neighbour
 *     protocol's statement, so it is dashed and the legend says which is which.
 *
 * Each node is a real <button> inside a <foreignObject> rather than a <g> with
 * role="button": the browser then gives it focus, Enter and Space for free,
 * and its hit area scales with the drawing instead of being a bare 18px dot.
 *
 * The map scales to its container through the viewBox, so there is no
 * horizontal scroll at 390px and no fixed pixel width to breach the fleet
 * phone-width gate.
 */

import { type JSX, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { TopologyLink, TopologyNode } from '../types/topology';
import { type GraphLinkInput, type GraphNodeInput, layoutTopology } from './topologyLayout';

/** The FDB reconciler's kind (internal/topology/edge_reconciler_fdb.go). Every
 * other kind — lldp, cdp, fdp — is a neighbour protocol and draws solid. */
const LEARNED_KIND = 'fdb';

/** The node's box in SVG user units: a dot with its label under it. Width sits
 * just inside the layout's 48-unit column so two neighbours never steal each
 * other's clicks, and the whole box is the target. */
const BOX_W = 44;
const BOX_H = 32;
const DOT = 18;
const LABEL_MAX = 9;

interface TopologyGraphProps {
  nodes: TopologyNode[];
  links: TopologyLink[];
  selectedId: string;
  onSelect: (id: string) => void;
}

/** labelOf matches what the list row shows, so the same device reads the same
 * on both halves of the page. */
function labelOf(node: TopologyNode): string {
  return node.displayName || node.sysName || node.primaryIp || node.id;
}

function truncate(label: string): string {
  return label.length > LABEL_MAX ? `${label.slice(0, LABEL_MAX - 1)}…` : label;
}

export function TopologyGraph({
  nodes,
  links,
  selectedId,
  onSelect,
}: TopologyGraphProps): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);

  const laid = useMemo(() => {
    const graphNodes: GraphNodeInput[] = nodes.map((n) => ({
      id: n.id,
      label: labelOf(n),
      deviceType: n.deviceType,
    }));
    const graphLinks: GraphLinkInput[] = links.map((l) => ({
      id: l.id,
      source: l.sourceNodeId,
      target: l.targetNodeId,
      learned: l.linkType === LEARNED_KIND,
    }));
    return layoutTopology(graphNodes, graphLinks);
  }, [nodes, links]);

  if (laid.nodes.length === 0) {
    return (
      <div
        data-testid="topology-graph-empty"
        className="rounded-2xl border border-surface-border bg-surface-raised pad-lg text-center body-small text-text-muted"
      >
        {t('topology.graphEmpty')}
      </div>
    );
  }

  return (
    <div className="rounded-2xl border border-surface-border bg-surface-raised">
      <div className="flex-between pad-sm border-b border-surface-border">
        <span className="kicker">{t('topology.graphTitle')}</span>
        <span
          className="flex items-center gap-tight body-small text-text-muted"
          data-testid="topology-graph-legend"
        >
          <Swatch dashed={false} />
          {t('topology.graphLegendNeighbor')}
          <Swatch dashed={true} />
          {t('topology.graphLegendLearned')}
        </span>
      </div>
      <div className="max-h-[32rem] overflow-y-auto pad-sm">
        <svg
          data-testid="topology-graph"
          viewBox={`0 0 ${laid.width} ${laid.height}`}
          preserveAspectRatio="xMidYMid meet"
          className="mx-auto block h-auto w-full"
          style={{ maxWidth: `${laid.width}px` }}
        >
          <title>{t('topology.graphAria')}</title>
          {laid.links.map((link) => (
            <line
              key={link.id}
              data-testid={`topology-graph-link-${link.id}`}
              x1={link.x1}
              y1={link.y1}
              x2={link.x2}
              y2={link.y2}
              strokeWidth={1}
              // An FDB edge is an inference, so it is dashed. The attribute is
              // absent (not "none") on a neighbour edge — the test reads it.
              {...(link.learned ? { strokeDasharray: '4 3' } : {})}
              className={
                link.source === selectedId || link.target === selectedId
                  ? 'stroke-brand-primary'
                  : 'stroke-hairline-strong'
              }
            />
          ))}
          {laid.nodes.map((node) => {
            const selected = node.id === selectedId;
            return (
              <foreignObject
                key={node.id}
                x={node.x - BOX_W / 2}
                y={node.y - DOT / 2}
                width={BOX_W}
                height={BOX_H}
              >
                <button
                  type="button"
                  data-testid={`topology-graph-node-${node.id}`}
                  aria-pressed={selected}
                  aria-label={t('topology.graphNodeAria', {
                    name: node.label,
                    type: node.deviceType || t('common:status.notApplicable'),
                  })}
                  onClick={(): void => onSelect(node.id)}
                  className="flex w-full flex-col items-center gap-0.5"
                >
                  <span
                    aria-hidden="true"
                    style={{ width: `${DOT}px`, height: `${DOT}px` }}
                    className={
                      selected
                        ? 'rounded-full border-[3px] border-brand-primary bg-surface-base'
                        : 'rounded-full border border-hairline-strong bg-surface-base'
                    }
                  />
                  <span
                    aria-hidden="true"
                    className={`text-[9px] leading-none ${
                      selected ? 'text-text-primary' : 'text-text-muted'
                    }`}
                  >
                    {truncate(node.label)}
                  </span>
                </button>
              </foreignObject>
            );
          })}
        </svg>
      </div>
    </div>
  );
}

/** Swatch is the legend's line sample — the same dash the map draws. */
function Swatch({ dashed }: { dashed: boolean }): JSX.Element {
  return (
    <svg width="22" height="8" aria-hidden="true" className="inline-block shrink-0 align-middle">
      <line
        x1={1}
        y1={4}
        x2={21}
        y2={4}
        strokeWidth={1}
        {...(dashed ? { strokeDasharray: '4 3' } : {})}
        className="stroke-hairline-strong"
      />
    </svg>
  );
}
