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
import { type FC, lazy } from 'react';
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
  /**
   * Id of the HelpDrawer section this page's (?) opens. Omit to hide the
   * button. Kept explicit rather than derived from i18nKey so a page may
   * point at a shared section, and so additions are visible in review —
   * helpRouteCoverage.test.ts holds every route to having one.
   */
  help?: string;
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
  help?: string;
}

const staticPages: PageDef[] = [
  {
    path: '/link',
    i18nKey: 'link',
    icon: Network,
    iconColorClass: 'text-module-telemetry',
    component: LinkPage,
    help: 'link',
  },
  {
    path: '/network',
    i18nKey: 'network',
    icon: Server,
    iconColorClass: 'text-module-telemetry',
    component: NetworkPage,
    help: 'network',
  },
  {
    path: '/path',
    i18nKey: 'path',
    icon: Route,
    iconColorClass: 'text-module-path',
    component: PathAnalysisPage,
    help: 'path',
  },
  {
    path: '/wifi',
    i18nKey: 'wifi',
    icon: Wifi,
    iconColorClass: 'text-module-wifi',
    component: WifiPage,
    help: 'wifi',
  },
  {
    path: '/security',
    i18nKey: 'security',
    icon: Shield,
    iconColorClass: 'text-module-security',
    component: SecurityPage,
    help: 'security',
  },
  {
    path: '/performance',
    i18nKey: 'performance',
    icon: Activity,
    component: PerformancePage,
    help: 'performance',
  },
  {
    path: '/reports',
    i18nKey: 'reports',
    icon: BarChart3,
    iconColorClass: 'text-module-reporting',
    component: ReportsPage,
    help: 'reports',
  },
  {
    path: '/logs',
    i18nKey: 'logs',
    icon: ScrollText,
    iconColorClass: 'text-module-reporting',
    component: LogsPage,
    help: 'logs',
  },
  {
    path: '/polling-targets',
    i18nKey: 'pollingTargets',
    icon: ServerCog,
    iconColorClass: 'text-module-security',
    component: PollingTargetsPage,
    help: 'pollingTargets',
  },
  {
    path: '/topology',
    i18nKey: 'topology',
    icon: Network,
    iconColorClass: 'text-module-security',
    component: TopologyPage,
    help: 'topology',
  },
  {
    path: '/alerts',
    i18nKey: 'alerts',
    icon: Bell,
    iconColorClass: 'text-module-security',
    component: AlertsPage,
    help: 'alerts',
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
