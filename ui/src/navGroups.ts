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
 *
 * Item labels resolve from the same pages.{i18nKey}.label keys the page
 * registry uses, so the rail and the page header cannot disagree and a
 * translator sees one canonical label per route. Group headings live at
 * pages.groups.* rather than common.sections.*, whose keys name a grouping
 * this file stopped using.
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
import { useTranslation } from 'react-i18next';
import type { SidebarNavGroup } from './ui/Sidebar';

export function useNavGroups(): SidebarNavGroup[] {
  const { t } = useTranslation('pages');
  return [
    {
      label: t('groups.liveTelemetry'),
      items: [
        { path: '/link', label: t('link.label'), icon: Network, accent: 'text-module-telemetry' },
        {
          path: '/network',
          label: t('network.label'),
          icon: Server,
          accent: 'text-module-telemetry',
        },
        {
          path: '/performance',
          label: t('performance.label'),
          icon: Activity,
          accent: 'text-module-telemetry',
        },
      ],
    },
    {
      label: t('groups.diagnostics'),
      items: [
        { path: '/path', label: t('path.label'), icon: Route, accent: 'text-module-path' },
        { path: '/wifi', label: t('wifi.label'), icon: Wifi, accent: 'text-module-wifi' },
        {
          path: '/security',
          label: t('security.label'),
          icon: Shield,
          accent: 'text-module-security',
        },
      ],
    },
    {
      // NMS pages have no botanical module; they keep the neutral colouring.
      label: t('groups.monitoring'),
      items: [
        { path: '/topology', label: t('topology.label'), icon: Share2 },
        { path: '/alerts', label: t('alerts.label'), icon: Bell },
        { path: '/polling-targets', label: t('pollingTargets.label'), icon: Target },
      ],
    },
    {
      label: t('groups.reporting'),
      items: [
        {
          path: '/reports',
          label: t('reports.label'),
          icon: BarChart3,
          accent: 'text-module-reporting',
        },
        {
          path: '/logs',
          label: t('logs.label'),
          icon: ScrollText,
          accent: 'text-module-reporting',
        },
      ],
    },
  ];
}
