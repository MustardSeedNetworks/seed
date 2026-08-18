/**
 * reporting.tsx — Where results are recorded, watched, and escalated.
 *
 * Content module for the HelpDrawer. Section bodies are typed `HelpBlock`s
 * (see helpModel.ts) so one generic renderer presents every section. Content
 * is FACTUAL — drawn from Seed's real feature set. No invented features, no
 * banned vocabulary.
 *
 * @copyright 2026 Mustard Seed Networks. All rights reserved.
 */

import { AlertTriangle, BarChart3, Network, Route, ScrollText, Server } from '../../ui/icons';
import type { HelpSection } from '../helpModel';

const ICON = 'w-4 h-4';

export const reportingSections: HelpSection[] = [
  {
    id: 'path',
    titleKey: 'sections.path',
    icon: <Route className={ICON} />,
    keywords: ['path', 'traceroute', 'route', 'hops', 'arp', 'l2', 'l3', 'gateway'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Path Analysis traces how traffic leaves the local network and reaches a destination. It surfaces every L2 hop on the local segment, every L3 hop on the route off-link, and the on-link devices ARP/ND can see along the way.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'L2 path',
            description:
              'Hops within the same broadcast domain (switches, bridges), discovered via ARP and on-link MAC tables.',
          },
          {
            term: 'L3 path',
            description:
              'Per-hop IPv4/IPv6 traceroute with round-trip latency to each hop and any AS / reverse-DNS metadata.',
          },
          {
            term: 'Gateway hop',
            description:
              'The first L3 hop off the local subnet — usually the router that issued the DHCP lease.',
          },
          {
            term: 'On-link discovery',
            description:
              'ARP / ND sweep that surfaces neighbors visible without crossing a router.',
          },
        ],
      },
    ],
  },
  {
    id: 'reports',
    titleKey: 'sections.reports',
    icon: <BarChart3 className={ICON} />,
    keywords: ['reports', 'sla', 'compliance', 'history', 'export', 'csv', 'json', 'pdf'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Reports collects the results of Seed’s diagnostic tests over time and exports them as SLA dashboards, compliance summaries, and historical CSV/JSON.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'SLA dashboard',
            description:
              'Rolling availability and latency view derived from health-check probe results.',
          },
          {
            term: 'Compliance summary',
            description:
              'Snapshot of which checks pass against the active profile’s thresholds — useful for audits.',
          },
          {
            term: 'Scheduled reports',
            description: 'Pro-tier feature that produces a periodic PDF report on a cadence.',
          },
          {
            term: 'Export',
            description:
              'Download a slice of the underlying data as CSV or JSON for downstream tooling.',
          },
        ],
      },
    ],
  },
  {
    id: 'logs',
    titleKey: 'sections.logs',
    icon: <ScrollText className={ICON} />,
    keywords: ['logs', 'log', 'stream', 'tail', 'daemon', 'level', 'source', 'debug'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Logs streams the seed daemon’s structured log entries live as they are emitted, with filters by level, source, and free-text. Useful for confirming what the backend just did or diagnosing why a test failed.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'Level',
            description:
              'Standard severity (debug / info / warn / error). Filter to narrow the view.',
          },
          {
            term: 'Source',
            description:
              'The internal package emitting the entry (for example discovery, canopy, shell).',
          },
          {
            term: 'Live tail',
            description:
              'WebSocket stream of new entries as they are produced by the running seed process.',
          },
          {
            term: 'Daemon health',
            description:
              'Rotating-file usage, error counts, and uptime of the seed process itself.',
          },
        ],
      },
    ],
  },
  {
    id: 'alerts',
    titleKey: 'sections.alerts',
    icon: <AlertTriangle className={ICON} />,
    keywords: ['alerts', 'alert', 'acknowledge', 'resolve', 'severity', 'notification', 'incident'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Alerts lists the conditions the monitoring pipelines have raised — for example a polled device going unreachable or an interface changing state. Filter by severity, acknowledged, or resolved, then act on each row: acknowledge marks it seen, resolve marks it fixed. The person who clicks is recorded against the alert.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'Severity',
            description:
              'How urgent the condition is. Filter to focus on the most important alerts first.',
          },
          {
            term: 'Acknowledge',
            description:
              'Marks an alert as seen without closing it — signals someone is looking into it.',
          },
          {
            term: 'Resolve',
            description: 'Marks an alert as fixed and removes it from the default unresolved view.',
          },
          {
            term: 'Acknowledged by',
            description:
              'The operator who acknowledged the alert, taken from the signed-in identity.',
          },
        ],
      },
    ],
  },
  {
    id: 'pollingTargets',
    titleKey: 'sections.pollingTargets',
    icon: <Server className={ICON} />,
    keywords: ['polling', 'targets', 'snmp', 'monitor', 'device', 'collector', 'add', 'edit'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Polling targets is the list of devices Seed polls over SNMP. Add a target to start monitoring it, edit one to change its settings, or remove one you no longer track. A new target picks up the default collector chain and begins polling on the next cycle; the devices and links it discovers appear on the Topology page, and state changes surface as alerts.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'Target',
            description: 'A device (by address) that Seed polls on a recurring interval.',
          },
          {
            term: 'Collector chain',
            description:
              'The set of SNMP collectors run against a target (system info, interface table, LLDP, ARP, and forwarding-database neighbors).',
          },
          {
            term: 'SNMP',
            description:
              'Simple Network Management Protocol — the standard used to read device state and tables.',
          },
        ],
      },
    ],
  },
  {
    id: 'topology',
    titleKey: 'sections.topology',
    icon: <Network className={ICON} />,
    keywords: ['topology', 'graph', 'nodes', 'links', 'neighbors', 'map', 'interfaces'],
    blocks: [
      {
        kind: 'paragraph',
        text: 'Topology shows the network graph reconciled from discovery and SNMP polling: every node visible to your session, with its interfaces and the links between nodes. Select a node to open a detail panel listing its interfaces and discovered links. The graph is built from the same observations that drive polling targets and alerts.',
      },
      {
        kind: 'terms',
        heading: 'Terms',
        items: [
          {
            term: 'Node',
            description:
              'A discovered device in the graph — switch, router, host, or access point.',
          },
          {
            term: 'Link',
            description:
              'A neighbor relationship between two nodes, learned from LLDP/CDP or forwarding tables.',
          },
          {
            term: 'Interface',
            description: 'A port on a node, with its status and any links observed on it.',
          },
        ],
      },
    ],
  },
];
