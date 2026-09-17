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
  /** Which sidebar group this route belongs to. */
  group: NavGroupKey;
  /**
   * Kicker above the title. Always the label of `group`, so the kicker and
   * the rail heading are one string resolved once (#2645).
   */
  eyebrow: string;
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
 * The sidebar groups, as `pages.groups.*` names them. A route declares its
 * group here and nowhere else: the rail reads it (navGroups.ts), and the
 * page's eyebrow is the group's own label, so the kicker over a page title
 * cannot contradict the heading the rail files it under. Before this,
 * `network` carried a hand-written eyebrow of "Diagnostics" while the rail
 * filed it under Live Telemetry (#2645).
 */
export type NavGroupKey = 'liveTelemetry' | 'diagnostics' | 'monitoring' | 'reporting';

/**
 * PageDef is the static, language-agnostic definition. The matching
 * translation lives at pages.{i18nKey}.{label,title,description} in
 * internal/i18n/locales/{en,es}/pages.json.
 */
interface PageDef {
  path: string;
  i18nKey: PageI18nKey;
  /** Which sidebar group this route belongs to; also its eyebrow. */
  group: NavGroupKey;
  icon: LucideIcon;
  iconColorClass?: string;
  component: FC;
  help?: string;
}

const staticPages: PageDef[] = [
  {
    path: '/link',
    group: 'liveTelemetry',
    i18nKey: 'link',
    icon: Network,
    iconColorClass: 'text-module-telemetry',
    component: LinkPage,
    help: 'link',
  },
  {
    path: '/network',
    group: 'liveTelemetry',
    i18nKey: 'network',
    icon: Server,
    iconColorClass: 'text-module-telemetry',
    component: NetworkPage,
    help: 'network',
  },
  {
    path: '/path',
    group: 'diagnostics',
    i18nKey: 'path',
    icon: Route,
    iconColorClass: 'text-module-path',
    component: PathAnalysisPage,
    help: 'path',
  },
  {
    path: '/wifi',
    group: 'diagnostics',
    i18nKey: 'wifi',
    icon: Wifi,
    iconColorClass: 'text-module-wifi',
    component: WifiPage,
    help: 'wifi',
  },
  {
    path: '/security',
    group: 'diagnostics',
    i18nKey: 'security',
    icon: Shield,
    iconColorClass: 'text-module-security',
    component: SecurityPage,
    help: 'security',
  },
  {
    path: '/performance',
    group: 'liveTelemetry',
    i18nKey: 'performance',
    icon: Activity,
    component: PerformancePage,
    help: 'performance',
  },
  {
    path: '/reports',
    group: 'reporting',
    i18nKey: 'reports',
    icon: BarChart3,
    iconColorClass: 'text-module-reporting',
    component: ReportsPage,
    help: 'reports',
  },
  {
    path: '/logs',
    group: 'reporting',
    i18nKey: 'logs',
    icon: ScrollText,
    iconColorClass: 'text-module-reporting',
    component: LogsPage,
    help: 'logs',
  },
  {
    path: '/polling-targets',
    group: 'monitoring',
    i18nKey: 'pollingTargets',
    icon: ServerCog,
    iconColorClass: 'text-module-security',
    component: PollingTargetsPage,
    help: 'pollingTargets',
  },
  {
    path: '/topology',
    group: 'monitoring',
    i18nKey: 'topology',
    icon: Network,
    iconColorClass: 'text-module-security',
    component: TopologyPage,
    help: 'topology',
  },
  {
    path: '/alerts',
    group: 'monitoring',
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
    group: p.group,
    // The eyebrow IS the group's label, not a second string a page can get
    // wrong: `network` used to declare "Diagnostics" while the rail filed it
    // under Live Telemetry (#2645). One key, one answer.
    eyebrow: t(`groups.${p.group}`),
    title: t(`${p.i18nKey}.title`),
    description: t(`${p.i18nKey}.description`),
    icon: p.icon,
    iconColorClass: p.iconColorClass,
    component: p.component,
    help: p.help,
  }));
}
