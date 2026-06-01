/**
 * Page registry — declarative route table for The Seed.
 *
 * Heavy pages are lazy-loaded so the initial chunk only carries the
 * Link landing page. The shape mirrors niac's pageRegistry; stem
 * exposes the same surface.
 */
import type { LucideIcon } from 'lucide-react';
import {
  Activity,
  AlertTriangle,
  BarChart3,
  Bell,
  Network,
  Route,
  ScrollText,
  Server,
  ServerCog,
  Shield,
  Wifi,
} from 'lucide-react';
import { type FC, lazy } from 'react';

// Eager — the default landing pages.
import { LinkPage } from './pages/LinkPage';
import { NetworkPage } from './pages/NetworkPage';

const PathAnalysisPage = lazy(() =>
  import('./pages/PathAnalysisPage').then((m) => ({ default: m.PathAnalysisPage })),
);
const WifiPage = lazy(() => import('./pages/WifiPage').then((m) => ({ default: m.WifiPage })));
const SecurityPage = lazy(() =>
  import('./pages/SecurityPage').then((m) => ({ default: m.SecurityPage })),
);
const PerformancePage = lazy(() =>
  import('./pages/PerformancePage').then((m) => ({ default: m.PerformancePage })),
);
const ReportsPage = lazy(() =>
  import('./pages/ReportsPage').then((m) => ({ default: m.ReportsPage })),
);
const LogsPage = lazy(() => import('./pages/LogsPage').then((m) => ({ default: m.LogsPage })));
const PollingTargetsPage = lazy(() =>
  import('./pages/PollingTargetsPage').then((m) => ({ default: m.PollingTargetsPage })),
);
const TopologyPage = lazy(() =>
  import('./pages/TopologyPage').then((m) => ({ default: m.TopologyPage })),
);
const AlertsPage = lazy(() =>
  import('./pages/AlertsPage').then((m) => ({ default: m.AlertsPage })),
);
const AlertRulesPage = lazy(() =>
  import('./pages/AlertRulesPage').then((m) => ({ default: m.AlertRulesPage })),
);

export interface PageConfig {
  path: string;
  label: string;
  title: string;
  description: string;
  icon: LucideIcon;
  component: FC;
}

export const pages: PageConfig[] = [
  {
    path: '/link',
    label: 'Link',
    title: 'Link',
    description: 'Physical link state, cable diagnostics, and Wi-Fi association.',
    icon: Network,
    component: LinkPage,
  },
  {
    path: '/network',
    label: 'Network',
    title: 'Network',
    description: 'DHCP, gateway, DNS, public IP, and switch detection.',
    icon: Server,
    component: NetworkPage,
  },
  {
    path: '/path',
    label: 'Path Analysis',
    title: 'Path Analysis',
    description: 'L2/L3 path discovery, traceroute hops, on-link device discovery.',
    icon: Route,
    component: PathAnalysisPage,
  },
  {
    path: '/wifi',
    label: 'Wi-Fi',
    title: 'Wi-Fi',
    description: 'Wi-Fi link, channel survey, and channel-overlap visualisation.',
    icon: Wifi,
    component: WifiPage,
  },
  {
    path: '/security',
    label: 'Security',
    title: 'Security',
    description: 'Guest network isolation audit and security posture checks.',
    icon: Shield,
    component: SecurityPage,
  },
  {
    path: '/performance',
    label: 'Performance',
    title: 'Performance',
    description: 'Active throughput tests and health-check probes.',
    icon: Activity,
    component: PerformancePage,
  },
  {
    path: '/reports',
    label: 'Reports',
    title: 'Reports',
    description: 'SLA dashboard, compliance tracking, and historical reporting.',
    icon: BarChart3,
    component: ReportsPage,
  },
  {
    path: '/logs',
    label: 'Logs',
    title: 'Logs',
    description: 'Live log stream and daemon system health.',
    icon: ScrollText,
    component: LogsPage,
  },
  {
    path: '/polling-targets',
    label: 'Polling targets',
    title: 'Polling targets',
    description: 'SNMP-polled devices and their collector chains.',
    icon: ServerCog,
    component: PollingTargetsPage,
  },
  {
    path: '/topology',
    label: 'Topology',
    title: 'Topology',
    description: 'Fat-Node graph reconciled from SNMP observations.',
    icon: Network,
    component: TopologyPage,
  },
  {
    path: '/alerts',
    label: 'Alerts',
    title: 'Alerts',
    description: 'Events emitted by the listener + observation pipelines.',
    icon: Bell,
    component: AlertsPage,
  },
  {
    path: '/alert-rules',
    label: 'Alert rules',
    title: 'Alert rules',
    description: 'Operator-defined rules feeding the listener pipeline.',
    icon: AlertTriangle,
    component: AlertRulesPage,
  },
];
