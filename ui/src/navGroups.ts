/**
 * Sidebar navigation groups for The Seed.
 *
 * Groups are labelled by FUNCTION, not by the botanical module metaphor
 * (Sap/Roots/Canopy/Shell/Harvest). This matches where the rest of the
 * product already landed after the strategy reset: code uses meaningful
 * tokens (`module-telemetry/path/wifi/security/reporting`), page copy never
 * used the metaphor, and the marketing site leads with the function and keeps
 * the botanical name only as a parenthetical aside ("Path analysis (Roots)").
 * The sidebar was the lone surface still leading with bare metaphors — and the
 * least self-evident one for a network engineer — so it now leads with the
 * function too (M1, #1452). The botanical names survive as the module accent
 * colour (`module-*` tokens) and in marketing; they are not sidebar headers.
 *
 * Grouping is by user intent:
 *   - Live Telemetry — real-time connection state + throughput.
 *   - Diagnostics    — on-demand investigations (path / Wi-Fi / security).
 *   - Monitoring     — NMS: SNMP/LLDP/CDP topology, alerting, polling.
 *   - Reporting      — outputs (reports, logs).
 *
 * Every routable page in pageRegistry must appear here so it is reachable
 * from the sidebar; navGroups.test.ts asserts that parity (guards H3 drift).
 * The sibling projects (niac, stem) ship the same shape via their own
 * navGroups files; mirror this function-first move there as separate PRs.
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
    label: 'Live Telemetry',
    items: [
      { path: '/link', label: 'Link', icon: Network },
      { path: '/network', label: 'Network', icon: Server },
      { path: '/performance', label: 'Performance', icon: Activity },
    ],
  },
  {
    label: 'Diagnostics',
    items: [
      { path: '/path', label: 'Path Analysis', icon: Route },
      { path: '/wifi', label: 'Wi-Fi', icon: Wifi },
      { path: '/security', label: 'Security', icon: Shield },
    ],
  },
  {
    label: 'Monitoring',
    items: [
      { path: '/topology', label: 'Topology', icon: Share2 },
      { path: '/alerts', label: 'Alerts', icon: Bell },
      { path: '/polling-targets', label: 'Polling Targets', icon: Target },
    ],
  },
  {
    label: 'Reporting',
    items: [
      { path: '/reports', label: 'Reports', icon: BarChart3 },
      { path: '/logs', label: 'Logs', icon: ScrollText },
    ],
  },
];
