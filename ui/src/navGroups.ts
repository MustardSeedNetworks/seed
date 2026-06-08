/**
 * Sidebar navigation groups for The Seed.
 *
 * Groups follow the existing module taxonomy (Roots / Canopy / Shell /
 * Sap / Harvest). The botanical group *labels* are intentional brand (the
 * code-vs-brand split, R4b) — do not "fix" them to literal module names.
 * Non-module functional groups (Performance, Monitoring) are allowed where
 * pages do not map to a botanical module.
 *
 * Every routable page in pageRegistry must appear here so it is reachable
 * from the sidebar; navGroups.test.ts asserts that parity (guards H3 drift).
 * The sibling projects (niac, stem) ship the same shape via their own
 * navGroups files.
 */
import {
  Activity,
  BarChart3,
  Bell,
  Network,
  Route,
  ScrollText,
  Server,
  Share2,
  Shield,
  Target,
  Wifi,
} from 'lucide-react';
import type { SidebarNavGroup } from './ui/Sidebar';

export const navGroups: SidebarNavGroup[] = [
  {
    label: 'Sap',
    items: [
      { path: '/link', label: 'Link', icon: Network },
      { path: '/network', label: 'Network', icon: Server },
    ],
  },
  {
    label: 'Roots',
    items: [{ path: '/path', label: 'Path Analysis', icon: Route }],
  },
  {
    label: 'Canopy',
    items: [{ path: '/wifi', label: 'Wi-Fi', icon: Wifi }],
  },
  {
    label: 'Shell',
    items: [{ path: '/security', label: 'Security', icon: Shield }],
  },
  {
    label: 'Performance',
    items: [{ path: '/performance', label: 'Performance', icon: Activity }],
  },
  {
    // NMS pages (SNMP/LLDP/CDP topology, alerting, polling). Minimal
    // discoverable placement in a functional group; the broader nav IA
    // redesign is deferred (#1452).
    label: 'Monitoring',
    items: [
      { path: '/topology', label: 'Topology', icon: Share2 },
      { path: '/alerts', label: 'Alerts', icon: Bell },
      { path: '/polling-targets', label: 'Polling Targets', icon: Target },
    ],
  },
  {
    label: 'Harvest',
    items: [
      { path: '/reports', label: 'Reports', icon: BarChart3 },
      { path: '/logs', label: 'Logs', icon: ScrollText },
    ],
  },
];
