/**
 * Pre-render helpers shared between NetworkDiscoveryCard and its summary.
 *
 * Contains:
 * - formatLastSeen / calculateNetworkAddress pure helpers
 * - SubnetList: smart inline / collapsible subnet display
 * - categorizeDevices: bucketise discovered devices for the stat row
 * - DiscoverySummary: the top-of-card status + subnet + category panel
 * - COMMON_PORTS: deep-scan port list
 */

import type React from 'react';
import { useMemo, useState } from 'react';
import type { useTranslation } from 'react-i18next';
import {
  category as categoryTheme,
  cn,
  icon as iconTokens,
  spacing,
  status as statusColor,
} from '../../styles/theme';
import {
  CheckCircle,
  ChevronDown,
  ChevronUp,
  Clock,
  Monitor,
  Printer,
  RefreshCw,
  Router,
  Server,
  Smartphone,
  Wifi,
} from '../ui/icons';
import type { DiscoveredDevice, DiscoveryStatus } from './networkDiscoveryCardTypes';
import { ScanProgress } from './ScanProgress';

type CardsT = ReturnType<typeof useTranslation<'cards'>>['t'];

// Common ports to scan for Deep Scan
export const COMMON_PORTS: number[] = [
  21, 22, 23, 25, 53, 80, 110, 143, 443, 445, 993, 995, 3306, 3389, 5432, 5900, 6379, 8080, 8443,
  27017,
];

// Format last seen timestamp to human-readable relative time
export function formatLastSeen(dateStr: string, t: CardsT): string {
  if (!dateStr) {
    return t('discovery.never');
  }
  const date = new Date(dateStr);
  // Check for invalid date or Go's zero time (year 1 or epoch)
  if (Number.isNaN(date.getTime()) || date.getFullYear() < 2000) {
    return t('discovery.never');
  }
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 0) {
    return t('discovery.never'); // Future date = invalid
  }
  if (diffSec < 60) {
    return t('discovery.justNow');
  }
  if (diffSec < 3600) {
    return t('discovery.mAgo', { min: Math.floor(diffSec / 60) });
  }
  if (diffSec < 86400) {
    return t('discovery.hAgo', { hour: Math.floor(diffSec / 3600) });
  }
  return t('discovery.dAgo', { day: Math.floor(diffSec / 86400) });
}

/**
 * Convert host IP/CIDR to network address (fixes #738)
 * e.g., "192.168.64.7/24" -> "192.168.64.0/24"
 */
export function calculateNetworkAddress(cidr: string): string {
  const [ip, maskStr] = cidr.split('/');
  if (!(ip && maskStr)) {
    return cidr;
  }

  const mask = Number.parseInt(maskStr, 10);
  if (Number.isNaN(mask) || mask < 0 || mask > 32) {
    return cidr;
  }

  const octets = ip.split('.').map(Number);
  if (octets.length !== 4 || octets.some(Number.isNaN)) {
    return cidr;
  }

  // Calculate network mask and apply to IP
  const netmask = (0xffffffff << (32 - mask)) >>> 0;
  const ipInt = ((octets[0] << 24) | (octets[1] << 16) | (octets[2] << 8) | octets[3]) >>> 0;
  const networkInt = (ipInt & netmask) >>> 0;

  const networkOctets = [
    (networkInt >>> 24) & 0xff,
    (networkInt >>> 16) & 0xff,
    (networkInt >>> 8) & 0xff,
    networkInt & 0xff,
  ];

  return `${networkOctets.join('.')}/${mask}`;
}

/**
 * SubnetList component for I3 - displays subnets with smart rollup.
 * - Inline display for <=5 subnets
 * - Expandable dropdown for >5 subnets
 */
export function SubnetList({
  subnets,
  fallbackSubnet,
  unknownLabel,
}: {
  subnets?: string[];
  fallbackSubnet?: string;
  unknownLabel: string;
}): React.ReactElement {
  const [expanded, setExpanded] = useState(false);

  // Use subnets array if available, otherwise fall back to single subnet
  const allSubnets = useMemo(() => {
    if (subnets && subnets.length > 0) {
      return subnets.map(calculateNetworkAddress);
    }
    if (fallbackSubnet) {
      return [calculateNetworkAddress(fallbackSubnet)];
    }
    return [];
  }, [subnets, fallbackSubnet]);

  if (allSubnets.length === 0) {
    return <span className="font-mono">{unknownLabel}</span>;
  }

  // Single subnet - simple display
  if (allSubnets.length === 1) {
    return <span className="font-mono">{allSubnets[0]}</span>;
  }

  // <=5 subnets - inline display
  if (allSubnets.length <= 5) {
    return <span className="font-mono">{allSubnets.join(', ')}</span>;
  }

  // >5 subnets - collapsible display
  if (!expanded) {
    return (
      <button
        type="button"
        onClick={(): void => setExpanded(true)}
        className="font-mono text-text-muted hover:text-text-primary flex items-center gap-tight"
      >
        <span>{allSubnets.length} subnets</span>
        <ChevronDown className={iconTokens.size.xs} />
      </button>
    );
  }

  return (
    <div className="flex flex-col gap-tight">
      <button
        type="button"
        onClick={(): void => setExpanded(false)}
        className="font-mono text-text-muted hover:text-text-primary flex items-center gap-tight"
      >
        <span>{allSubnets.length} subnets</span>
        <ChevronUp className={iconTokens.size.xs} />
      </button>
      <div className="flex flex-wrap gap-tight">
        {allSubnets.map((subnet) => (
          <span key={subnet} className="font-mono text-xs">
            {subnet}
          </span>
        ))}
      </div>
    </div>
  );
}

export interface CategoryCounts {
  routers: number;
  servers: number;
  workstations: number;
  printers: number;
  mobile: number;
  network: number;
  other: number;
}

// Device type categorization based on profile icons and device type
export function categorizeDevices(devices: DiscoveredDevice[]): CategoryCounts {
  const categories: CategoryCounts = {
    routers: 0,
    servers: 0,
    workstations: 0,
    printers: 0,
    mobile: 0,
    network: 0, // switches, APs
    other: 0,
  };

  for (const device of devices) {
    const deviceType = device.profile?.deviceType?.toLowerCase() || '';
    const icons = device.profile?.deviceIcons || [];

    if (
      icons.includes('router') ||
      deviceType.includes('router') ||
      device.cdpInfo?.capabilities?.some((c) => c.toLowerCase().includes('router')) ||
      device.lldpInfo?.capabilities?.some((c) => c.toLowerCase().includes('router'))
    ) {
      categories.routers++;
    } else if (
      icons.includes('switch') ||
      deviceType.includes('switch') ||
      device.cdpInfo?.capabilities?.some((c) => c.toLowerCase().includes('switch')) ||
      device.lldpInfo?.capabilities?.some((c) => c.toLowerCase().includes('bridge'))
    ) {
      categories.network++;
    } else if (icons.includes('printer') || deviceType.includes('printer')) {
      categories.printers++;
    } else if (
      icons.includes('server') ||
      deviceType.includes('server') ||
      icons.includes('database') ||
      icons.includes('dns') ||
      icons.includes('mail')
    ) {
      categories.servers++;
    } else if (
      deviceType.includes('phone') ||
      deviceType.includes('mobile') ||
      device.vendor?.toLowerCase().includes('apple') ||
      device.vendor?.toLowerCase().includes('samsung')
    ) {
      categories.mobile++;
    } else if (
      device.osGuess?.toLowerCase().includes('windows') ||
      device.osGuess?.toLowerCase().includes('linux') ||
      device.osGuess?.toLowerCase().includes('macos')
    ) {
      categories.workstations++;
    } else {
      categories.other++;
    }
  }

  return categories;
}

// Summary bar component
export function DiscoverySummary({
  status,
  deviceCount,
  categories,
  scanRunning,
  scanPercent,
  scanPhase,
  onCancelScan,
  t,
}: {
  status: DiscoveryStatus;
  deviceCount: number;
  categories: CategoryCounts;
  scanRunning?: boolean;
  scanPercent?: number;
  scanPhase?: string;
  onCancelScan?: () => void;
  t: CardsT;
}): React.ReactElement {
  // Show scan progress while the engine-scan job is running (S3).
  if (scanRunning) {
    return (
      <div className="stack-sm">
        <ScanProgress percent={scanPercent ?? 0} phase={scanPhase ?? ''} onCancel={onCancelScan} />
      </div>
    );
  }

  // Build stat items with non-zero counts
  // Using theme tokens for device category colors (dark mode aware)
  const stats = [
    {
      icon: Router,
      label: t('discovery.routers'),
      count: categories.routers,
      color: categoryTheme.router,
    },
    {
      icon: Server,
      label: t('discovery.servers'),
      count: categories.servers,
      color: categoryTheme.server,
    },
    {
      icon: Monitor,
      label: t('discovery.workstations'),
      count: categories.workstations,
      color: categoryTheme.workstation,
    },
    {
      icon: Printer,
      label: t('discovery.printers'),
      count: categories.printers,
      color: categoryTheme.printer,
    },
    {
      icon: Smartphone,
      label: t('discovery.mobile'),
      count: categories.mobile,
      color: categoryTheme.mobile,
    },
    {
      icon: Wifi,
      label: t('discovery.networkDevices'),
      count: categories.network,
      color: categoryTheme.network,
    },
  ].filter((s) => s.count > 0);

  return (
    <div className="stack-sm">
      {/* Status row */}
      <div className="flex-between body-small">
        <div className={cn('flex items-center', spacing.gap.compact)}>
          {status.scanning ? (
            <>
              <RefreshCw className={cn(iconTokens.size.sm, 'text-status-info animate-spin')} />
              <span className="text-status-info font-medium">{t('discovery.scanning')}</span>
            </>
          ) : (
            <>
              <CheckCircle className={cn(iconTokens.size.sm, statusColor.text.success)} />
              <span className="text-status-success font-medium">{t('discovery.complete')}</span>
            </>
          )}
        </div>
        <div className={cn('flex items-center', spacing.inline.sm, 'text-text-muted')}>
          <Clock className={iconTokens.size.sm} />
          <span className="caption">{formatLastSeen(status.lastScan, t)}</span>
        </div>
      </div>
      {/* Simplified network info row - I3: Uses SubnetList for multi-subnet display */}
      <div className="flex-between caption text-text-muted">
        <SubnetList
          subnets={status.subnets}
          fallbackSubnet={status.subnet}
          unknownLabel={t('discovery.unknownSubnet')}
        />
        <span>
          {deviceCount === 1
            ? t('discovery.deviceFound', { count: deviceCount })
            : t('discovery.devicesFound', { count: deviceCount })}
        </span>
      </div>
      {/* Category stats row */}
      {stats.length > 0 && (
        <div
          className={cn(
            'flex items-center',
            spacing.gap.default,
            'flex-wrap',
            spacing.padding.top.heading,
          )}
        >
          {stats.map(({ icon: ICON, label, count, color }) => (
            <div
              key={label}
              className={cn('flex items-center', spacing.gap.tight)}
              title={`${count} ${label}`}
            >
              <ICON className={cn(iconTokens.size.sm, color)} />
              <span className="caption text-text-secondary">{count}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
