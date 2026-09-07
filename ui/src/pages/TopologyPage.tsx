/**
 * TopologyPage
 *
 * Renders the fat-Node graph the Stage A4 reconcilers maintain: the list of
 * every node visible to this session, and the selected node's interfaces and
 * links from /topology/nodes/{id}.
 *
 * Despite the route name this is not the Topology archetype — there is no
 * graph here, and seed carries no graph library. It is List + detail, which is
 * what the shared parts in ui/ListDetail.tsx model, so it uses those rather
 * than a second hand-rolled two-pane layout.
 *
 * Selection is in-page state, not router state: the rest of the app navigates
 * by path, and pushing a route per click would put node ids in history for a
 * pane the user is scanning.
 */

import { Activity, Cable, RefreshCw } from 'lucide-react';
import type { TFunction } from 'i18next';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useTopologyNode, useTopologyNodes } from '../hooks/useTopology';
import type { TopologyInterface, TopologyLink } from '../types/topology';
import {
  DetailEmpty,
  DetailFacts,
  DetailPane,
  ListDetail,
  RecordPane,
  RecordRow,
} from '../ui/ListDetail';

export function TopologyPage(): JSX.Element {
  const [selectedID, setSelectedID] = useState<string>('');

  return (
    <ListDetail>
      <NodeList selectedID={selectedID} onSelect={setSelectedID} />
      <NodeDetail id={selectedID} onClear={(): void => setSelectedID('')} />
    </ListDetail>
  );
}

interface NodeListProps {
  selectedID: string;
  onSelect: (id: string) => void;
}

function NodeList({ selectedID, onSelect }: NodeListProps): JSX.Element {
  const { t } = useTranslation('pages');
  const { nodes, loading, error, refresh } = useTopologyNodes();

  return (
    <RecordPane
      filter={
        <div className="flex-between">
          <span className="kicker">
            {loading ? t('common:status.loading') : t('topology.nodeCount', { count: nodes.length })}
          </span>
          <button
            type="button"
            onClick={(): void => {
              void refresh();
            }}
            // Icon-only, so the icon's 16px was the whole target (#244).
            className="target flex-center text-text-muted hover:text-text-primary"
            aria-label={t('common:buttons.refresh')}
          >
            <RefreshCw className="h-4 w-4" />
          </button>
        </div>
      }
      empty={
        error ? (
          <span className="text-status-error">{error}</span>
        ) : (
          t('topology.noNodes')
        )
      }
    >
      {error || nodes.length === 0
        ? null
        : nodes.map((n) => (
            <RecordRow
              key={n.id}
              data-testid={`node-row-${n.id}`}
              // A hostname is a figure, not prose — it is an identifier the
              // operator matches against other screens character by character.
              name={n.displayName || n.sysName}
              meta={n.primaryIp || undefined}
              value={n.deviceType || t('common:status.notApplicable')}
              // A node the reconcilers have never dated is not healthy or
              // unhealthy; it is unmeasured, and the bar says so rather than
              // showing the green that "seen" would imply.
              state={n.lastSeen ? 'ok' : 'unknown'}
              selected={selectedID === n.id}
              onSelect={(): void => onSelect(n.id)}
            />
          ))}
    </RecordPane>
  );
}

interface NodeDetailProps {
  id: string;
  onClear: () => void;
}

function NodeDetail({ id, onClear }: NodeDetailProps): JSX.Element {
  const { t } = useTranslation('pages');
  const { detail, loading, error } = useTopologyNode(id);

  if (!id) {
    return <DetailEmpty>{t('topology.selectNode')}</DetailEmpty>;
  }
  if (loading) {
    return <DetailEmpty>{t('topology.loadingNode')}</DetailEmpty>;
  }
  if (error) {
    return (
      <div className="rounded-2xl border border-status-error/40 bg-status-error/10 pad-lg text-sm text-status-error">
        {error}
      </div>
    );
  }
  if (!detail) {
    return <DetailEmpty>{t('topology.nodeNotFound')}</DetailEmpty>;
  }

  const { node } = detail;
  const na = t('common:status.notApplicable');
  return (
    <DetailPane
      eyebrow={t('topology.selectedNode')}
      title={node.displayName || node.sysName}
      meta={node.id}
      actions={
        <button
          type="button"
          onClick={onClear}
          className="text-xs text-text-muted hover:text-text-primary"
        >
          {t('common:buttons.clear')}
        </button>
      }
    >
      <DetailFacts
        items={[
          // Device type and sys name are names, not measurements — prose, so
          // they do not sit in the monospace column with the addresses.
          { label: t('topology.deviceType'), value: node.deviceType || na, prose: true },
          { label: t('topology.sysName'), value: node.sysName || na, prose: true },
          { label: t('topology.primaryMac'), value: node.primaryMac || na },
          { label: t('topology.primaryIp'), value: node.primaryIp || na },
          { label: t('topology.firstSeen'), value: fmtTime(node.firstSeen, t) },
          { label: t('topology.lastSeen'), value: fmtTime(node.lastSeen, t) },
        ]}
      />
      <InterfacesPanel interfaces={detail.interfaces} />
      <LinksPanel links={detail.links} nodeID={node.id} />
    </DetailPane>
  );
}

function InterfacesPanel({ interfaces }: { interfaces: TopologyInterface[] }): JSX.Element {
  const { t } = useTranslation('pages');
  return (
    <div className="rounded-lg border border-surface-border bg-surface-raised">
      <div className="flex items-center gap-compact border-b border-surface-border px-4 py-2">
        <Activity className="h-4 w-4 text-status-success" />
        <span className="text-sm font-medium text-text-primary">
          {t('topology.interfacesCount', { count: interfaces.length })}
        </span>
      </div>
      {interfaces.length === 0 ? (
        <div className="pad text-sm text-text-muted">{t('topology.noInterfaceData')}</div>
      ) : (
        <table className="w-full text-sm">
          <thead className="text-left text-xs uppercase tracking-wide text-text-muted">
            <tr>
              <th className="px-4 py-2">{t('topology.colIndex')}</th>
              <th className="px-4 py-2">{t('common:labels.name')}</th>
              <th className="px-4 py-2">{t('topology.adminOper')}</th>
              <th className="px-4 py-2">{t('topology.colSpeed')}</th>
              <th className="px-4 py-2">MAC</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-surface-border">
            {interfaces.map((i) => (
              <tr key={i.id}>
                <td className="px-4 py-2 text-text-muted">{i.ifIndex}</td>
                <td className="px-4 py-2 text-text-primary">{i.ifName || i.ifDescr}</td>
                <td className="px-4 py-2">
                  <IfStatusPair admin={i.ifAdminStatus} oper={i.ifOperStatus} />
                </td>
                <td className="px-4 py-2 text-text-secondary">{fmtSpeed(i.speedBps)}</td>
                <td className="px-4 py-2 font-mono text-xs text-text-muted">
                  {i.ifPhysAddr || '—'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

function IfStatusPair({ admin, oper }: { admin: number; oper: number }): JSX.Element {
  // RFC 2233 values — 1=up, 2=down, anything else lands in the
  // catch-all dim color.
  return (
    <span className="flex items-center gap-tight text-xs">
      <span className={admin === 1 ? 'text-status-success' : 'text-text-muted'}>admin</span>
      <span className="text-text-muted">/</span>
      <span
        className={
          oper === 1 ? 'text-status-success' : oper === 2 ? 'text-status-error' : 'text-text-muted'
        }
      >
        oper
      </span>
    </span>
  );
}

function LinksPanel({ links, nodeID }: { links: TopologyLink[]; nodeID: string }): JSX.Element {
  const { t } = useTranslation('pages');
  return (
    <div className="rounded-lg border border-surface-border bg-surface-raised">
      <div className="flex items-center gap-compact border-b border-surface-border px-4 py-2">
        <Cable className="h-4 w-4 text-status-info" />
        <span className="text-sm font-medium text-text-primary">
          {t('topology.neighborLinksCount', { count: links.length })}
        </span>
      </div>
      {links.length === 0 ? (
        <div className="pad text-sm text-text-muted">{t('topology.noEdges')}</div>
      ) : (
        <ul className="divide-y divide-surface-border">
          {links.map((l) => {
            const otherEnd = l.sourceNodeId === nodeID ? l.targetNodeId : l.sourceNodeId;
            return (
              <li key={l.id} className="flex-between px-4 py-2 text-sm">
                <span className="text-text-primary">↔ {otherEnd}</span>
                <span className="text-xs text-text-muted">
                  {l.linkType} · {fmtTime(l.lastSeen, t)}
                </span>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

/**
 * fmtTime takes `t` rather than calling useTranslation because it is used from
 * the row bodies of two panels; passing it keeps the "never" word in the
 * locale files without turning the helper into a component.
 */
function fmtTime(iso: string, t: TFunction<'pages'>): string {
  if (!iso) return t('common:status.never');
  return new Date(iso).toLocaleString();
}

function fmtSpeed(bps: number): string {
  if (!bps) return '—';
  if (bps >= 1_000_000_000) return `${(bps / 1_000_000_000).toFixed(1)} Gbps`;
  if (bps >= 1_000_000) return `${(bps / 1_000_000).toFixed(0)} Mbps`;
  return `${bps} bps`;
}
