/**
 * Sidebar navigation groups for The Seed.
 *
 * Groups follow the existing module taxonomy (Roots / Canopy / Shell /
 * Sap / Harvest). The sibling projects (niac, stem) ship the same
 * shape via their own navGroups files.
 */
import {
  Activity,
  BarChart3,
  Network,
  Route,
  ScrollText,
  Server,
  Shield,
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
    label: 'Harvest',
    items: [
      { path: '/reports', label: 'Reports', icon: BarChart3 },
      { path: '/logs', label: 'Logs', icon: ScrollText },
    ],
  },
];
