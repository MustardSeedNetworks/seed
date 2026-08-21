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
import { type JSX, useState } from 'react';
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
  const { nodes, loading, error, refresh } = useTopologyNodes();

  return (
    <RecordPane
      filter={
        <div className="flex items-center justify-between">
          <span className="kicker">
            {loading ? 'Loading…' : `${nodes.length} node${nodes.length === 1 ? '' : 's'}`}
          </span>
          <button
            type="button"
            onClick={(): void => {
              void refresh();
            }}
            className="text-text-muted hover:text-text-primary"
            aria-label="Refresh"
          >
            <RefreshCw className="h-4 w-4" />
          </button>
        </div>
      }
      empty={
        error ? (
          <span className="text-status-error">{error}</span>
        ) : (
          'No nodes yet. Add a polling target and wait a poll cycle.'
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
              value={n.deviceType || 'n/a'}
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
  const { detail, loading, error } = useTopologyNode(id);

  if (!id) {
    return <DetailEmpty>Select a node to see interfaces and links.</DetailEmpty>;
  }
  if (loading) {
    return <DetailEmpty>Loading node…</DetailEmpty>;
  }
  if (error) {
    return (
      <div className="rounded-2xl border border-status-error/40 bg-status-error/10 pad-lg text-sm text-status-error">
        {error}
      </div>
    );
  }
  if (!detail) {
    return <DetailEmpty>Node not found.</DetailEmpty>;
  }

  const { node } = detail;
  return (
    <DetailPane
      eyebrow="Selected node"
      title={node.displayName || node.sysName}
      meta={node.id}
      actions={
        <button
          type="button"
          onClick={onClear}
          className="text-xs text-text-muted hover:text-text-primary"
        >
          Clear
        </button>
      }
    >
      <DetailFacts
        items={[
          // Device type and sys name are names, not measurements — prose, so
          // they do not sit in the monospace column with the addresses.
          { label: 'Device type', value: node.deviceType || 'n/a', prose: true },
          { label: 'Sys name', value: node.sysName || 'n/a', prose: true },
          { label: 'Primary MAC', value: node.primaryMac || 'n/a' },
          { label: 'Primary IP', value: node.primaryIp || 'n/a' },
          { label: 'First seen', value: fmtTime(node.firstSeen) },
          { label: 'Last seen', value: fmtTime(node.lastSeen) },
        ]}
      />
      <InterfacesPanel interfaces={detail.interfaces} />
      <LinksPanel links={detail.links} nodeID={node.id} />
    </DetailPane>
  );
}

function InterfacesPanel({ interfaces }: { interfaces: TopologyInterface[] }): JSX.Element {
  return (
    <div className="rounded-lg border border-surface-border bg-surface-raised">
      <div className="flex items-center gap-2 border-b border-surface-border px-4 py-2">
        <Activity className="h-4 w-4 text-status-success" />
        <span className="text-sm font-medium text-text-primary">
          Interfaces ({interfaces.length})
        </span>
      </div>
      {interfaces.length === 0 ? (
        <div className="p-4 text-sm text-text-muted">
          No interface data yet. The if_table reconciler folds these in on the next poll.
        </div>
      ) : (
        <table className="w-full text-sm">
          <thead className="text-left text-xs uppercase tracking-wide text-text-muted">
            <tr>
              <th className="px-4 py-2">Index</th>
              <th className="px-4 py-2">Name</th>
              <th className="px-4 py-2">Admin / Oper</th>
              <th className="px-4 py-2">Speed</th>
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
    <span className="flex items-center gap-1 text-xs">
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
  return (
    <div className="rounded-lg border border-surface-border bg-surface-raised">
      <div className="flex items-center gap-2 border-b border-surface-border px-4 py-2">
        <Cable className="h-4 w-4 text-status-info" />
        <span className="text-sm font-medium text-text-primary">
          Neighbor links ({links.length})
        </span>
      </div>
      {links.length === 0 ? (
        <div className="p-4 text-sm text-text-muted">
          No edges yet. LLDP/CDP/FDP needs both endpoints to be known nodes.
        </div>
      ) : (
        <ul className="divide-y divide-surface-border">
          {links.map((l) => {
            const otherEnd = l.sourceNodeId === nodeID ? l.targetNodeId : l.sourceNodeId;
            return (
              <li key={l.id} className="flex items-center justify-between px-4 py-2 text-sm">
                <span className="text-text-primary">↔ {otherEnd}</span>
                <span className="text-xs text-text-muted">
                  {l.linkType} · {fmtTime(l.lastSeen)}
                </span>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

function fmtTime(iso: string): string {
  if (!iso) return 'never';
  return new Date(iso).toLocaleString();
}

function fmtSpeed(bps: number): string {
  if (!bps) return '—';
  if (bps >= 1_000_000_000) return `${(bps / 1_000_000_000).toFixed(1)} Gbps`;
  if (bps >= 1_000_000) return `${(bps / 1_000_000).toFixed(0)} Mbps`;
  return `${bps} bps`;
}
