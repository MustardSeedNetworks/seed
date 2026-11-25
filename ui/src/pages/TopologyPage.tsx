/**
 * TopologyPage
 *
 * Renders the fat-Node graph the Stage A4 reconcilers maintain: a map of every
 * node and edge visible to this session, the list beneath it, and the selected
 * node's interfaces and links from /topology/nodes/{id}.
 *
 * The map is the page's primary answer and the list is the index into it — 200
 * discovered nodes as a scrollbar was the defect (seed#2700). The list is a
 * sortable, paged table beside the selected node's detail pane.
 *
 * Selection is in-page state, not router state: the rest of the app navigates
 * by path, and pushing a route per click would put node ids in history for a
 * pane the user is scanning.
 */

import type { TFunction } from 'i18next';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Activity, Cable, Network, RefreshCw } from '../components/ui/Icons';
import { useTableLabels } from '../hooks/useTableLabels';
import { useTopologyLinks, useTopologyNode, useTopologyNodes } from '../hooks/useTopology';
import { Alert } from '../shared/Alert';
import { Button } from '../shared/Button';
import { CategoryChip } from '../shared/CategoryChip';
import { DataTable, type DataTableColumn } from '../shared/DataTable';
import { EmptyState } from '../shared/EmptyState';
import { StatusPill, type StatusTone } from '../shared/StatusPill';
import type {
  TopologyInterface,
  TopologyLink,
  TopologyNode,
} from '../types/generated/topology-node-detail-response';
import { DetailEmpty, DetailFacts, DetailPane } from '../ui/ListDetail';
import { TopologyGraph } from '../ui/TopologyGraph';
import { type NodeFreshness, nodeFreshness } from './topologyFreshness';

export function TopologyPage(): JSX.Element {
  const [selectedID, setSelectedID] = useState<string>('');
  // One nodes read feeds both halves, so the map and the list can never
  // disagree about what was discovered.
  const { nodes, fetchedAt, loading, error, refresh } = useTopologyNodes();
  const { links, error: linksError } = useTopologyLinks();

  return (
    <div className="space-y-4">
      <TopologyGraph
        nodes={nodes}
        links={links}
        linksError={linksError}
        selectedId={selectedID}
        onSelect={setSelectedID}
      />
      <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <NodeList
          nodes={nodes}
          fetchedAt={fetchedAt}
          loading={loading}
          error={error}
          refresh={refresh}
          selectedID={selectedID}
          onSelect={setSelectedID}
        />
        <NodeDetail id={selectedID} onClear={(): void => setSelectedID('')} />
      </div>
    </div>
  );
}

interface NodeListProps {
  nodes: TopologyNode[];
  fetchedAt: number;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  selectedID: string;
  onSelect: (id: string) => void;
}

// Green only while the node is polled on schedule. A node that missed polls
// is a warning; one never measured is neutral, not healthy.
const FRESHNESS_TONE: Record<NodeFreshness, StatusTone> = {
  current: 'success',
  stale: 'warning',
  unknown: 'unknown',
};
const FRESHNESS_RANK: Record<NodeFreshness, number> = { current: 0, stale: 1, unknown: 2 };

function NodeList({
  nodes,
  fetchedAt,
  loading,
  error,
  refresh,
  selectedID,
  onSelect,
}: NodeListProps): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const labels = useTableLabels<TopologyNode>({ loading: t('common:status.loading') });
  const freshnessOf = (n: TopologyNode): NodeFreshness =>
    nodeFreshness(n.lastSeen, n.pollIntervalSeconds, fetchedAt);
  const freshnessLabel = (f: NodeFreshness): string => {
    switch (f) {
      case 'current':
        return t('topology.freshness.current');
      case 'stale':
        return t('topology.freshness.stale');
      case 'unknown':
        return t('topology.freshness.unknown');
    }
  };

  const columns: DataTableColumn<TopologyNode>[] = [
    {
      key: 'node',
      header: t('topology.colNode'),
      sortValue: (n) => n.displayName || n.sysName,
      cell: (n) => <span className="figure text-text-primary">{n.displayName || n.sysName}</span>,
    },
    {
      key: 'address',
      header: t('topology.colAddress'),
      sortValue: (n) => n.primaryIp,
      cell: (n) => <span className="figure">{n.primaryIp || '—'}</span>,
    },
    {
      key: 'type',
      header: t('topology.colType'),
      sortValue: (n) => n.deviceType,
      cell: (n) => <CategoryChip label={n.deviceType || t('common:status.notApplicable')} />,
    },
    {
      key: 'status',
      header: t('topology.colStatus'),
      sortValue: (n) => FRESHNESS_RANK[freshnessOf(n)],
      cell: (n) => {
        const f = freshnessOf(n);
        return <StatusPill tone={FRESHNESS_TONE[f]} label={freshnessLabel(f)} />;
      },
    },
  ];

  let body: JSX.Element;
  if (error) {
    body = <Alert tone="error">{error}</Alert>;
  } else {
    body = (
      <DataTable
        rows={nodes}
        columns={columns}
        getRowKey={(n) => n.id}
        caption={t('topology.nodesCaption')}
        labels={labels}
        loading={loading}
        pageSize={25}
        search={{
          label: t('topology.searchNodes'),
          clearLabel: t('topology.clearSearch'),
          text: (n) => [n.displayName, n.sysName, n.id, n.primaryIp].filter(Boolean).join(' '),
          noResults: <EmptyState title={t('topology.noMatchTitle')} />,
        }}
        defaultSort={{ key: 'node', direction: 'asc' }}
        onRowClick={(n): void => onSelect(n.id)}
        selectedRowKey={selectedID}
        empty={
          <EmptyState
            icon={<Network className="h-6 w-6" aria-hidden="true" />}
            title={t('topology.noNodesTitle')}
            hint={t('topology.noNodesHint')}
          />
        }
      />
    );
  }

  return (
    <section className="min-w-0 space-y-2">
      <div className="flex items-center justify-between gap-2">
        <span className="kicker">
          {loading ? t('common:status.loading') : t('topology.nodeCount', { count: nodes.length })}
        </span>
        <Button
          variant="ghost"
          iconOnly
          icon={<RefreshCw className="h-4 w-4" aria-hidden="true" />}
          aria-label={t('common:buttons.refresh')}
          onClick={(): void => {
            void refresh();
          }}
        />
      </div>
      {body}
    </section>
  );
}

interface NodeDetailProps {
  id: string;
  onClear: () => void;
}

// vendorOf reads the vendor the sysinfo reconciler derives from
// sysObjectID. It lives in metadata rather than in a column of its own
// because it is one of several scalars the poll carries; metadata is
// typed as unknown values, so the string check is the narrowing.
function vendorOf(node: TopologyNode): string {
  const vendor = node.metadata?.vendor;
  return typeof vendor === 'string' ? vendor : '';
}

function NodeDetail({ id, onClear }: NodeDetailProps): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const { detail, loading, error } = useTopologyNode(id);

  if (!id) {
    return <DetailEmpty>{t('topology.selectNode')}</DetailEmpty>;
  }
  if (loading) {
    return <DetailEmpty>{t('topology.loadingNode')}</DetailEmpty>;
  }
  if (error) {
    return <Alert tone="error">{error}</Alert>;
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
        <Button variant="secondary" size="sm" onClick={onClear}>
          {t('common:buttons.clear')}
        </Button>
      }
    >
      <DetailFacts
        items={[
          // Device type and sys name are names, not measurements — prose, so
          // they do not sit in the monospace column with the addresses.
          { label: t('topology.deviceType'), value: node.deviceType || na, prose: true },
          // The vendor stopped being the device type in seed#2456 — it is
          // still collected, so it keeps a row of its own rather than
          // disappearing from the screen.
          { label: t('topology.vendor'), value: vendorOf(node) || na, prose: true },
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

function IfStatusPair({ admin, oper }: { admin: number; oper: number }): JSX.Element {
  const { t } = useTranslation('pages');
  // RFC 2233 values: 1 is up and 2 is down; testing, dormant and the rest say
  // nothing about the port, so they are neutral, never green.
  const state = (value: number): { tone: StatusTone; word: string } => {
    if (value === 1) return { tone: 'success', word: t('topology.ifState.up') };
    if (value === 2) return { tone: 'error', word: t('topology.ifState.down') };
    return { tone: 'unknown', word: t('topology.ifState.other') };
  };
  const a = state(admin);
  const o = state(oper);
  return (
    <span className="flex flex-wrap items-center gap-1">
      <StatusPill
        tone={a.tone === 'error' ? 'unknown' : a.tone}
        label={t('topology.adminState', { state: a.word })}
      />
      <StatusPill tone={o.tone} label={t('topology.operState', { state: o.word })} />
    </span>
  );
}

function InterfacesPanel({ interfaces }: { interfaces: TopologyInterface[] }): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const labels = useTableLabels<TopologyInterface>({ loading: t('common:status.loading') });
  const columns: DataTableColumn<TopologyInterface>[] = [
    {
      key: 'index',
      header: t('topology.colIndex'),
      sortValue: (i) => i.ifIndex,
      cell: (i) => <span className="text-text-muted">{i.ifIndex}</span>,
    },
    {
      key: 'name',
      header: t('common:labels.name'),
      sortValue: (i) => i.ifName || i.ifDescr,
      cell: (i) => <span className="text-text-primary">{i.ifName || i.ifDescr}</span>,
    },
    {
      key: 'status',
      header: t('topology.adminOper'),
      cell: (i) => <IfStatusPair admin={i.ifAdminStatus} oper={i.ifOperStatus} />,
    },
    {
      key: 'speed',
      header: t('topology.colSpeed'),
      sortValue: (i) => i.speedBps,
      cell: (i) => fmtSpeed(i.speedBps),
    },
    {
      key: 'mac',
      header: 'MAC',
      cell: (i) => <span className="font-mono text-xs text-text-muted">{i.ifPhysAddr || '—'}</span>,
    },
  ];
  return (
    <div className="space-y-2">
      <h3 className="flex items-center gap-2 text-sm font-medium text-text-primary">
        <Activity className="h-4 w-4 text-text-muted" aria-hidden="true" />
        {t('topology.interfacesCount', { count: interfaces.length })}
      </h3>
      <DataTable
        rows={interfaces}
        columns={columns}
        getRowKey={(i) => String(i.id)}
        caption={t('topology.interfacesCaption')}
        labels={labels}
        pageSize={25}
        empty={
          <EmptyState
            icon={<Activity className="h-6 w-6" aria-hidden="true" />}
            title={t('topology.noInterfaceDataTitle')}
            hint={t('topology.noInterfaceDataHint')}
          />
        }
      />
    </div>
  );
}

function LinksPanel({ links, nodeID }: { links: TopologyLink[]; nodeID: string }): JSX.Element {
  const { t } = useTranslation(['pages', 'common']);
  const labels = useTableLabels<TopologyLink>({ loading: t('common:status.loading') });
  const columns: DataTableColumn<TopologyLink>[] = [
    {
      key: 'neighbor',
      header: t('topology.colNeighbor'),
      sortValue: (l) => (l.sourceNodeId === nodeID ? l.targetNodeId : l.sourceNodeId),
      cell: (l) => (
        <span className="text-text-primary">
          ↔ {l.sourceNodeId === nodeID ? l.targetNodeId : l.sourceNodeId}
        </span>
      ),
    },
    {
      key: 'type',
      header: t('topology.colLinkType'),
      sortValue: (l) => l.linkType,
      cell: (l) => <CategoryChip label={l.linkType} />,
    },
    {
      key: 'lastSeen',
      header: t('topology.colSeen'),
      cell: (l) => fmtTime(l.lastSeen, t),
    },
  ];
  return (
    <div className="space-y-2">
      <h3 className="flex items-center gap-2 text-sm font-medium text-text-primary">
        <Cable className="h-4 w-4 text-text-muted" aria-hidden="true" />
        {t('topology.neighborLinksCount', { count: links.length })}
      </h3>
      <DataTable
        rows={links}
        columns={columns}
        getRowKey={(l) => l.id}
        caption={t('topology.linksCaption')}
        labels={labels}
        pageSize={25}
        empty={
          <EmptyState
            icon={<Cable className="h-6 w-6" aria-hidden="true" />}
            title={t('topology.noEdgesTitle')}
            hint={t('topology.noEdgesHint')}
          />
        }
      />
    </div>
  );
}

/**
 * fmtTime takes `t` rather than calling useTranslation because it is used from
 * the row bodies of two panels; passing it keeps the "never" word in the
 * locale files without turning the helper into a component.
 */
function fmtTime(iso: string, t: TFunction<['pages', 'common']>): string {
  if (!iso) return t('common:status.never');
  return new Date(iso).toLocaleString();
}

function fmtSpeed(bps: number): string {
  if (!bps) return '—';
  if (bps >= 1_000_000_000) return `${(bps / 1_000_000_000).toFixed(1)} Gbps`;
  if (bps >= 1_000_000) return `${(bps / 1_000_000).toFixed(0)} Mbps`;
  return `${bps} bps`;
}
