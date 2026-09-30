/**
 * DiscoveryModal cell content and column priority (#461).
 *
 * Each low-priority column joins the table at a container width and, below
 * it, moves into the row's details panel. The same pieces render in both
 * places, so a field is never shown one way in the column and another way in
 * the panel. Container queries, not viewport breakpoints: the table sits in a
 * modal whose width is the viewport's minus the modal's own padding.
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import {
  cn,
  discoveryMethod as discoveryMethodTheme,
  radius,
  severity as severityTheme,
} from '../../styles/theme';
import type {
  DiscoveredDevice,
  Vulnerability,
} from '../../types/generated/engine-discovery-response';
import { AlertTriangle } from '../ui/Icons';
import { Tooltip } from '../ui/Tooltip';
import type { DiscoveryMethod } from './NetworkDiscoveryCard';
import { formatLastSeen } from './NetworkDiscoveryCardHelpers';

/**
 * `cell` shows the column from its width up; `summary` hides the panel copy
 * from the same width. Literal strings so Tailwind generates them. IP, name and
 * actions are always columns.
 */
export const column = {
  vendor: { cell: 'hidden @xl:table-cell w-40', summary: '@xl:hidden' },
  lastSeen: { cell: 'hidden @2xl:table-cell w-24', summary: '@2xl:hidden' },
  mac: { cell: 'hidden @3xl:table-cell w-40', summary: '@3xl:hidden' },
  methods: { cell: 'hidden @4xl:table-cell w-32', summary: '@4xl:hidden' },
  ports: { cell: 'hidden @6xl:table-cell w-20', summary: '@6xl:hidden' },
  vulns: { cell: 'hidden @6xl:table-cell w-20', summary: '@6xl:hidden' },
} as const;

/** Hides what exists only to carry hidden columns once every column shows. */
export const ALL_COLUMNS_SHOWN_HIDDEN = '@6xl:hidden';

export function deviceName(device: DiscoveredDevice): string | undefined {
  return device.displayName || device.mdnsName || device.netbiosName || device.hostname;
}

// `discoveryMethod` is an open string set on the wire — the Go type has no enum
// and a collector may add one — so the badge takes whatever arrives and falls
// back to the ARP styling for a method it has no colour for.
export function MethodBadge({ method }: { method: string }): JSX.Element {
  const themeClass = discoveryMethodTheme[method as DiscoveryMethod] ?? discoveryMethodTheme.arp;
  return (
    <span className={cn('px-1.5 py-0.5 text-xs font-medium uppercase', radius.md, themeClass)}>
      {method}
    </span>
  );
}

export function getSeverityClasses(severity: string): string {
  if (severity === 'CRITICAL') {
    return `${severityTheme.critical.bg} ${severityTheme.critical.text}`;
  }
  if (severity === 'HIGH') {
    return `${severityTheme.high.bg} ${severityTheme.high.text}`;
  }
  if (severity === 'MEDIUM') {
    return `${severityTheme.medium.bg} ${severityTheme.medium.text}`;
  }
  return `${severityTheme.low.bg} ${severityTheme.low.text}`;
}

// Severity order, worst first, matching getSeverityClasses' branches.
const SEVERITY_ORDER = ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW'] as const;

// The wire carries the findings themselves (DeviceVulnerabilities.vulnerabilities);
// the count and the worst severity are the row's own summary of them. The
// hand-typed mirror this file used to carry declared `count`/`highestSeverity`
// instead, which the daemon has never sent, so the badge read undefined and the
// column was always "-" (seed#2393).
export function highestSeverity(vulnerabilities: Vulnerability[] | undefined): string {
  // `severity` is NVD's `baseSeverity` passed through verbatim (internal/discovery/vuln/cve_nvd.go),
  // so its casing is the feed's, not ours.
  const severities = new Set((vulnerabilities ?? []).map((v) => v.severity.toUpperCase()));
  return SEVERITY_ORDER.find((s) => severities.has(s)) ?? 'LOW';
}

const none = <span className="text-xs text-text-muted">-</span>;

export function VendorLabel({ vendor }: { vendor: string | undefined }): JSX.Element {
  const { t } = useTranslation('cards');
  if (vendor === 'LAA') {
    return (
      <Tooltip text={t('discovery.localMacHint')} side="bottom">
        <span className="text-xs text-text-muted underline decoration-dotted cursor-help">LAA</span>
      </Tooltip>
    );
  }
  return (
    <Tooltip text={vendor}>
      <span className="text-xs text-text-muted truncate block">{vendor || '-'}</span>
    </Tooltip>
  );
}

export function MethodList({ methods }: { methods: string[] }): JSX.Element {
  return (
    <div className="flex items-center gap-tight flex-wrap">
      {methods.map((method) => (
        <MethodBadge key={method} method={method} />
      ))}
    </div>
  );
}

export function PortsBadge({ count }: { count: number }): JSX.Element {
  const { t } = useTranslation('cards');
  if (count === 0) {
    return none;
  }
  return (
    <span
      className={cn(
        'text-xs px-1.5 py-0.5 bg-status-success/15 text-status-success-strong',
        radius.md,
      )}
    >
      {t('discovery.open', { count })}
    </span>
  );
}

export function VulnBadge({
  vulnerabilities,
}: {
  vulnerabilities: Vulnerability[] | undefined;
}): JSX.Element {
  const count = vulnerabilities?.length ?? 0;
  if (count === 0) {
    return none;
  }
  return (
    <span
      className={cn(
        'inline-flex items-center gap-tight text-xs px-1.5 py-0.5',
        radius.md,
        getSeverityClasses(highestSeverity(vulnerabilities)),
      )}
    >
      <AlertTriangle className="w-3 h-3" />
      {count}
    </span>
  );
}

/** The panel copy of every priority column, each hidden once its column shows. */
export function ColumnSummary({
  device,
  openPortCount,
}: {
  device: DiscoveredDevice;
  openPortCount: number;
}): JSX.Element {
  const { t } = useTranslation('cards');
  const items = [
    {
      key: 'vendor',
      label: t('discovery.tableVendor'),
      value: <VendorLabel vendor={device.vendor} />,
    },
    {
      key: 'lastSeen',
      label: t('discovery.tableLastSeen'),
      value: <span className="text-xs text-text-muted">{formatLastSeen(device.lastSeen, t)}</span>,
    },
    {
      key: 'mac',
      label: t('discovery.tableMac'),
      value: <span className="font-mono text-xs text-text-muted">{device.mac || '-'}</span>,
    },
    {
      key: 'methods',
      label: t('discovery.tableDiscovery'),
      value: <MethodList methods={device.discoveryMethod} />,
    },
    { key: 'ports', label: t('discovery.tablePorts'), value: <PortsBadge count={openPortCount} /> },
    {
      key: 'vulns',
      label: t('discovery.tableVulns'),
      value: <VulnBadge vulnerabilities={device.vulnerabilities?.vulnerabilities} />,
    },
  ] as const;
  return (
    <dl
      className={cn('grid grid-cols-2 gap-compact text-xs', ALL_COLUMNS_SHOWN_HIDDEN)}
      data-testid="discovery-row-summary"
    >
      {items.map((item) => (
        <div key={item.key} className={cn('min-w-0', column[item.key].summary)}>
          <dt className="text-text-muted">{item.label}</dt>
          <dd>{item.value}</dd>
        </div>
      ))}
    </dl>
  );
}
