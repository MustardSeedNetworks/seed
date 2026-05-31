/**
 * TopologyPage
 *
 * Renders the fat-Node graph the Stage A4 reconcilers maintain.
 * Two views in one route — when no node is selected, the page shows
 * the list of every node visible to this session; clicking a node
 * loads /topology/nodes/{id} for the detail panel (interfaces +
 * links). State lives in this component (no router-state because
 * the rest of the app uses path-based navigation; node detail
 * is in-page state to avoid pushing a new route per click).
 */

import { Activity, Cable, Network, RefreshCw } from 'lucide-react';
import { type JSX, useState } from 'react';
import { useTopologyNode, useTopologyNodes } from '../hooks/useTopology';
import type { TopologyInterface, TopologyLink, TopologyNode } from '../types/topology';
import { Breadcrumbs } from '../ui/Breadcrumbs';
import { PageHeader } from '../ui/PageHeader';

export function TopologyPage(): JSX.Element {
  const [selectedID, setSelectedID] = useState<string>('');

  return (
    <section className="stack-xl">
      <Breadcrumbs />
      <PageHeader
        icon={Network}
        title="Topology"
        description="Devices polled and edges learned from LLDP/CDP/FDP neighbor discovery."
        iconColorClass="text-module-shell"
      />

      <div className="grid gap-4 lg:grid-cols-[1fr_2fr]">
        <NodeList selectedID={selectedID} onSelect={setSelectedID} />
        <NodeDetail id={selectedID} onClear={(): void => setSelectedID('')} />
      </div>
    </section>
  );
}

interface NodeListProps {
  selectedID: string;
  onSelect: (id: string) => void;
}

function NodeList({ selectedID, onSelect }: NodeListProps): JSX.Element {
  const { nodes, loading, error, refresh } = useTopologyNodes();

  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900/30">
      <div className="flex items-center justify-between border-b border-zinc-800 px-4 py-2">
        <span className="text-xs font-medium uppercase tracking-wide text-zinc-400">
          {loading ? 'Loading…' : `${nodes.length} node${nodes.length === 1 ? '' : 's'}`}
        </span>
        <button
          type="button"
          onClick={(): void => {
            void refresh();
          }}
          className="text-zinc-400 hover:text-zinc-200"
          aria-label="Refresh"
        >
          <RefreshCw className="h-4 w-4" />
        </button>
      </div>
      {error ? (
        <div className="p-3 text-sm text-rose-300">{error}</div>
      ) : nodes.length === 0 ? (
        <div className="p-4 text-sm text-zinc-500">
          No nodes yet. Add a polling target and wait a poll cycle.
        </div>
      ) : (
        <ul className="divide-y divide-zinc-800">
          {nodes.map((n) => (
            <li key={n.id}>
              <button
                type="button"
                onClick={(): void => onSelect(n.id)}
                data-testid={`node-row-${n.id}`}
                className={`flex w-full items-center justify-between gap-3 px-4 py-3 text-left text-sm transition hover:bg-zinc-800/50 ${
                  selectedID === n.id ? 'bg-zinc-800/70' : ''
                }`}
              >
                <span className="flex-1 truncate">
                  <span className="font-medium text-zinc-100">{n.displayName || n.sysName}</span>
                  {n.primaryIp ? (
                    <span className="ml-2 font-mono text-xs text-zinc-500">{n.primaryIp}</span>
                  ) : null}
                </span>
                <DeviceTypeBadge type={n.deviceType} />
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function DeviceTypeBadge({ type }: { type: string }): JSX.Element {
  // Coarse color hints for at-a-glance scanning — matches the
  // device_type values produced by topology.deviceTypeFromObjectID
  // (cisco / juniper / arista / linux / mikrotik / unknown).
  const palette: Record<string, string> = {
    cisco: 'bg-sky-500/20 text-sky-300',
    juniper: 'bg-emerald-500/20 text-emerald-300',
    arista: 'bg-orange-500/20 text-orange-300',
    linux: 'bg-zinc-500/20 text-zinc-300',
    mikrotik: 'bg-violet-500/20 text-violet-300',
  };
  const cls = palette[type] ?? 'bg-zinc-700 text-zinc-400';
  return (
    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${cls}`}>{type || 'n/a'}</span>
  );
}

interface NodeDetailProps {
  id: string;
  onClear: () => void;
}

function NodeDetail({ id, onClear }: NodeDetailProps): JSX.Element {
  const { detail, loading, error } = useTopologyNode(id);

  if (!id) {
    return (
      <div className="flex min-h-[300px] items-center justify-center rounded-lg border border-zinc-800 bg-zinc-900/30 text-sm text-zinc-500">
        Select a node to see interfaces and links.
      </div>
    );
  }
  if (loading) {
    return (
      <div className="rounded-lg border border-zinc-800 bg-zinc-900/30 p-6 text-sm text-zinc-500">
        Loading node…
      </div>
    );
  }
  if (error) {
    return (
      <div className="rounded-lg border border-rose-500/40 bg-rose-500/10 p-6 text-sm text-rose-300">
        {error}
      </div>
    );
  }
  if (!detail) {
    return (
      <div className="rounded-lg border border-zinc-800 bg-zinc-900/30 p-6 text-sm text-zinc-500">
        Node not found.
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <NodeSummary node={detail.node} onClear={onClear} />
      <InterfacesPanel interfaces={detail.interfaces} />
      <LinksPanel links={detail.links} nodeID={detail.node.id} />
    </div>
  );
}

function NodeSummary({ node, onClear }: { node: TopologyNode; onClear: () => void }): JSX.Element {
  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900/30 p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold text-zinc-100">
            {node.displayName || node.sysName}
          </h2>
          <p className="text-xs text-zinc-500">{node.id}</p>
        </div>
        <button
          type="button"
          onClick={onClear}
          className="text-xs text-zinc-400 hover:text-zinc-200"
        >
          Clear
        </button>
      </div>
      <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-1 text-sm text-zinc-300">
        <SummaryRow label="Device type" value={node.deviceType || 'n/a'} />
        <SummaryRow label="Sys name" value={node.sysName || 'n/a'} />
        <SummaryRow label="Primary MAC" value={node.primaryMac || 'n/a'} mono />
        <SummaryRow label="Primary IP" value={node.primaryIp || 'n/a'} mono />
        <SummaryRow label="First seen" value={fmtTime(node.firstSeen)} />
        <SummaryRow label="Last seen" value={fmtTime(node.lastSeen)} />
      </dl>
    </div>
  );
}

function SummaryRow({
  label,
  value,
  mono,
}: {
  label: string;
  value: string;
  mono?: boolean;
}): JSX.Element {
  return (
    <>
      <dt className="text-xs uppercase tracking-wide text-zinc-500">{label}</dt>
      <dd className={mono ? 'font-mono text-zinc-200' : 'text-zinc-200'}>{value}</dd>
    </>
  );
}

function InterfacesPanel({ interfaces }: { interfaces: TopologyInterface[] }): JSX.Element {
  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900/30">
      <div className="flex items-center gap-2 border-b border-zinc-800 px-4 py-2">
        <Activity className="h-4 w-4 text-emerald-400" />
        <span className="text-sm font-medium text-zinc-100">Interfaces ({interfaces.length})</span>
      </div>
      {interfaces.length === 0 ? (
        <div className="p-4 text-sm text-zinc-500">
          No interface data yet. The if_table reconciler folds these in on the next poll.
        </div>
      ) : (
        <table className="w-full text-sm">
          <thead className="text-left text-xs uppercase tracking-wide text-zinc-500">
            <tr>
              <th className="px-4 py-2">Index</th>
              <th className="px-4 py-2">Name</th>
              <th className="px-4 py-2">Admin / Oper</th>
              <th className="px-4 py-2">Speed</th>
              <th className="px-4 py-2">MAC</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-800">
            {interfaces.map((i) => (
              <tr key={i.id}>
                <td className="px-4 py-2 text-zinc-400">{i.ifIndex}</td>
                <td className="px-4 py-2 text-zinc-100">{i.ifName || i.ifDescr}</td>
                <td className="px-4 py-2">
                  <IfStatusPair admin={i.ifAdminStatus} oper={i.ifOperStatus} />
                </td>
                <td className="px-4 py-2 text-zinc-300">{fmtSpeed(i.speedBps)}</td>
                <td className="px-4 py-2 font-mono text-xs text-zinc-400">{i.ifPhysAddr || '—'}</td>
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
      <span className={admin === 1 ? 'text-emerald-400' : 'text-zinc-500'}>admin</span>
      <span className="text-zinc-600">/</span>
      <span
        className={oper === 1 ? 'text-emerald-400' : oper === 2 ? 'text-rose-400' : 'text-zinc-500'}
      >
        oper
      </span>
    </span>
  );
}

function LinksPanel({ links, nodeID }: { links: TopologyLink[]; nodeID: string }): JSX.Element {
  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900/30">
      <div className="flex items-center gap-2 border-b border-zinc-800 px-4 py-2">
        <Cable className="h-4 w-4 text-sky-400" />
        <span className="text-sm font-medium text-zinc-100">Neighbor links ({links.length})</span>
      </div>
      {links.length === 0 ? (
        <div className="p-4 text-sm text-zinc-500">
          No edges yet. LLDP/CDP/FDP needs both endpoints to be known nodes.
        </div>
      ) : (
        <ul className="divide-y divide-zinc-800">
          {links.map((l) => {
            const otherEnd = l.sourceNodeId === nodeID ? l.targetNodeId : l.sourceNodeId;
            return (
              <li key={l.id} className="flex items-center justify-between px-4 py-2 text-sm">
                <span className="text-zinc-100">↔ {otherEnd}</span>
                <span className="text-xs text-zinc-500">
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
