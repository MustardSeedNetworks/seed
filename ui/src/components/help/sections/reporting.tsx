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
        text: 'content.path.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.path.terms.hop.term',
            description: 'content.path.terms.hop.description',
          },
          {
            term: 'content.path.terms.protocol.term',
            description: 'content.path.terms.protocol.description',
          },
          {
            term: 'content.path.terms.local.term',
            description: 'content.path.terms.local.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:pathDiscovery.title',
            description: 'content.cardHelp.PathDiscoveryCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:discovery.title',
            description: 'content.cardHelp.NetworkDiscoveryCard.description',
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
        text: 'content.reports.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.reports.terms.sla.term',
            description: 'content.reports.terms.sla.description',
          },
          {
            term: 'content.reports.terms.export.term',
            description: 'content.reports.terms.export.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:slaDashboard.title',
            description: 'content.cardHelp.SlaDashboardCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:reports.title',
            description: 'content.cardHelp.ReportsCard.description',
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
        text: 'content.logs.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.logs.terms.level.term',
            description: 'content.logs.terms.level.description',
          },
          {
            term: 'content.logs.terms.source.term',
            description: 'content.logs.terms.source.description',
          },
          {
            term: 'content.logs.terms.stream.term',
            description: 'content.logs.terms.stream.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'cards:system.title',
            description: 'content.cardHelp.SystemHealthCard.description',
          },
        ],
      },
      {
        kind: 'terms',
        items: [
          {
            term: 'common:logs.title',
            description: 'content.cardHelp.LogViewerCard.description',
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
        text: 'content.alerts.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.alerts.terms.severity.term',
            description: 'content.alerts.terms.severity.description',
          },
          {
            term: 'content.alerts.terms.ack.term',
            description: 'content.alerts.terms.ack.description',
          },
          {
            term: 'content.alerts.terms.resolve.term',
            description: 'content.alerts.terms.resolve.description',
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
        text: 'content.pollingTargets.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.pollingTargets.terms.target.term',
            description: 'content.pollingTargets.terms.target.description',
          },
          {
            term: 'content.pollingTargets.terms.collectors.term',
            description: 'content.pollingTargets.terms.collectors.description',
          },
          {
            term: 'content.pollingTargets.terms.snmp.term',
            description: 'content.pollingTargets.terms.snmp.description',
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
        text: 'content.topology.description',
      },
      {
        kind: 'terms',
        heading: 'content.common.terms',
        items: [
          {
            term: 'content.topology.terms.node.term',
            description: 'content.topology.terms.node.description',
          },
          {
            term: 'content.topology.terms.link.term',
            description: 'content.topology.terms.link.description',
          },
          {
            term: 'content.topology.terms.interface.term',
            description: 'content.topology.terms.interface.description',
          },
        ],
      },
    ],
  },
];
