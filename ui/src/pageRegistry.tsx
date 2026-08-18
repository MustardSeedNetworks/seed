/**
 * Page registry — declarative route table for The Seed.
 *
 * Heavy pages are lazy-loaded so the initial chunk only carries the
 * Link landing page. The shape mirrors niac's pageRegistry; stem
 * exposes the same surface.
 *
 * The header a page wears is rendered centrally by AppShell from this
 * table, not by the page itself — one edit per route, and page bodies
 * can't drift from the nav label the way hand-rolled headers did.
 */
import type { LucideIcon } from 'lucide-react';
import {
  Activity,
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
import { type FC, lazy, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

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

/**
 * PageConfig is one entry in the route table, resolved at render time
 * via usePages() — label/title/description/eyebrow are translations of
 * the corresponding pages.{i18nKey}.* keys.
 */
export interface PageConfig {
  path: string;
  label: string;
  /** Kicker above the title naming the product domain. */
  eyebrow?: string;
  title: string;
  description: string;
  icon: LucideIcon;
  iconColorClass?: string;
  component: FC;
  /** Rich help shown in the header's side panel. Omit to hide the (?) button. */
  help?: ReactNode;
}

/**
 * PageI18nKey is the closed set of pages.* namespaces that carry a
 * matching {label,title,description} triple. Kept strict so adding a
 * new route forces a corresponding locale entry.
 */
type PageI18nKey =
  | 'link'
  | 'network'
  | 'path'
  | 'wifi'
  | 'security'
  | 'performance'
  | 'reports'
  | 'logs'
  | 'pollingTargets'
  | 'topology'
  | 'alerts';

/**
 * PageDef is the static, language-agnostic definition. The matching
 * translation lives at pages.{i18nKey}.{label,title,description} in
 * internal/i18n/locales/{en,es}/pages.json.
 */
interface PageDef {
  path: string;
  i18nKey: PageI18nKey;
  icon: LucideIcon;
  iconColorClass?: string;
  component: FC;
  help?: ReactNode;
}

const staticPages: PageDef[] = [
  {
    path: '/link',
    i18nKey: 'link',
    icon: Network,
    iconColorClass: 'text-module-telemetry',
    component: LinkPage,
  },
  {
    path: '/network',
    i18nKey: 'network',
    icon: Server,
    iconColorClass: 'text-module-telemetry',
    component: NetworkPage,
    help: (
      <div className="stack-md">
        <p>
          The Network page shows the diagnostic state of the upstream link: DHCP lease, default
          gateway, DNS resolvers, the public IP the gateway uses, and (when wired) the
          directly-attached switch and its VLAN configuration.
        </p>
        <p>
          Each card refreshes on its own schedule and reflects the active interface selected in the
          header. If a card shows "no data", either the interface has not yet been probed, or that
          piece of the upstream config is not present (e.g., no IPv6 gateway).
        </p>
      </div>
    ),
  },
  {
    path: '/path',
    i18nKey: 'path',
    icon: Route,
    iconColorClass: 'text-module-path',
    component: PathAnalysisPage,
  },
  {
    path: '/wifi',
    i18nKey: 'wifi',
    icon: Wifi,
    iconColorClass: 'text-module-wifi',
    component: WifiPage,
  },
  {
    path: '/security',
    i18nKey: 'security',
    icon: Shield,
    iconColorClass: 'text-module-security',
    component: SecurityPage,
  },
  { path: '/performance', i18nKey: 'performance', icon: Activity, component: PerformancePage },
  {
    path: '/reports',
    i18nKey: 'reports',
    icon: BarChart3,
    iconColorClass: 'text-module-reporting',
    component: ReportsPage,
  },
  {
    path: '/logs',
    i18nKey: 'logs',
    icon: ScrollText,
    iconColorClass: 'text-module-reporting',
    component: LogsPage,
  },
  {
    path: '/polling-targets',
    i18nKey: 'pollingTargets',
    icon: ServerCog,
    iconColorClass: 'text-module-security',
    component: PollingTargetsPage,
  },
  {
    path: '/topology',
    i18nKey: 'topology',
    icon: Network,
    iconColorClass: 'text-module-security',
    component: TopologyPage,
  },
  {
    path: '/alerts',
    i18nKey: 'alerts',
    icon: Bell,
    iconColorClass: 'text-module-security',
    component: AlertsPage,
  },
];

/**
 * usePages returns the route table with label/title/description/eyebrow
 * resolved against the active locale. A hook rather than a const so
 * react-i18next's languageChanged event re-renders consumers.
 */
export function usePages(): PageConfig[] {
  const { t } = useTranslation('pages');
  return staticPages.map((p) => ({
    path: p.path,
    label: t(`${p.i18nKey}.label`),
    // A page has an eyebrow when its locale namespace declares one, so the
    // copy lives in one place instead of being mirrored by a flag here.
    // Pages still awaiting their archetype pass have none.
    eyebrow: t(`${p.i18nKey}.eyebrow`, { defaultValue: '' }) || undefined,
    title: t(`${p.i18nKey}.title`),
    description: t(`${p.i18nKey}.description`),
    icon: p.icon,
    iconColorClass: p.iconColorClass,
    component: p.component,
    help: p.help,
  }));
}
