import { Tooltip } from '../ui/Tooltip';
/**
 * DiscoveryModal - Full-screen modal for network device discovery.
 *
 * Opens as a large modal overlay for better device list viewing.
 * Provides table-based layout with sortable columns, search, and filtering.
 *
 * Features:
 * - Full-screen modal with backdrop
 * - Sortable table columns (IP, hostname, vendor, MAC, last seen)
 * - Search and filtering
 * - Device details expandable rows
 * - Lower-priority columns fold into the details panel on narrow widths, and
 *   long lists are virtualised (#461)
 * - Export to CSV/JSON
 * - Keyboard support (Escape to close)
 */

import type React from 'react';
import type { JSX } from 'react';
import { useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useFocusTrap } from '../../hooks/useFocusTrap';
import { button, cn, icon as iconTokens, modal, radius } from '../../styles/theme';
import { ArrowUpDown, ChevronDown, ChevronUp, Download, RefreshCw, Search, X } from '../ui/Icons';
import { column } from './DiscoveryModalCells';
import { DeviceRow } from './DiscoveryModalDeviceRow';
import type { DiscoveredDevice, NetworkDiscoveryData } from './NetworkDiscoveryCard';
import { useVirtualDeviceRows } from './useVirtualDeviceRows';
import { VulnerabilityDetailsModal } from './VulnerabilityDetailsModal';

interface DiscoveryModalProps {
  isOpen: boolean;
  onClose: () => void;
  data: NetworkDiscoveryData | null;
  onScan?: () => void;
  onDeepScan?: (ip: string) => Promise<void>;
}

type SortField = 'ip' | 'hostname' | 'vendor' | 'mac' | 'lastSeen';
type SortDirection = 'asc' | 'desc';

const COLUMN_COUNT = 9;

function deviceKey(device: DiscoveredDevice): string {
  return device.mac || `ip:${device.ip}`;
}

// Sort comparator
function compareDevices(
  a: DiscoveredDevice,
  b: DiscoveredDevice,
  field: SortField,
  direction: SortDirection,
): number {
  let cmp = 0;
  switch (field) {
    case 'ip': {
      // Sort IPs numerically
      const aParts = (a.ip || '').split('.').map(Number);
      const bParts = (b.ip || '').split('.').map(Number);
      for (let i = 0; i < 4; i++) {
        if ((aParts[i] || 0) !== (bParts[i] || 0)) {
          cmp = (aParts[i] || 0) - (bParts[i] || 0);
          break;
        }
      }
      break;
    }
    case 'hostname':
      cmp = (a.displayName || a.mdnsName || a.netbiosName || a.hostname || '').localeCompare(
        b.displayName || b.mdnsName || b.netbiosName || b.hostname || '',
      );
      break;
    case 'vendor':
      cmp = (a.vendor || '').localeCompare(b.vendor || '');
      break;
    case 'mac':
      cmp = (a.mac || '').localeCompare(b.mac || '');
      break;
    case 'lastSeen':
      cmp = new Date(a.lastSeen).getTime() - new Date(b.lastSeen).getTime();
      break;
    default:
      break;
  }
  return direction === 'asc' ? cmp : -cmp;
}

// Helper function to get sort icon (avoids nested ternary)
function getSortIcon(isActive: boolean, direction: SortDirection): JSX.Element {
  if (!isActive) {
    return <ArrowUpDown className="w-3 h-3 opacity-30" />;
  }
  if (direction === 'asc') {
    return <ChevronUp className="w-3 h-3" />;
  }
  return <ChevronDown className="w-3 h-3" />;
}

// Table header with sort indicator
function SortableHeader({
  label,
  field,
  currentField,
  direction,
  onSort,
  className,
}: {
  label: string;
  field: SortField;
  currentField: SortField | null;
  direction: SortDirection;
  onSort: (field: SortField) => void;
  className?: string;
}): JSX.Element {
  const isActive = currentField === field;
  let ariaSort: 'ascending' | 'descending' | 'none' = 'none';
  if (isActive) {
    ariaSort = direction === 'asc' ? 'ascending' : 'descending';
  }
  return (
    <th
      aria-sort={ariaSort}
      className={cn('text-left text-xs font-semibold uppercase tracking-wider', className)}
    >
      <button
        type="button"
        onClick={() => onSort(field)}
        className="flex w-full items-center gap-tight px-3 py-row uppercase tracking-wider hover:bg-surface-hover transition-colors select-none"
      >
        <span className="truncate">{label}</span>
        {getSortIcon(isActive, direction)}
      </button>
    </th>
  );
}

const headerCell = 'px-3 py-row text-left text-xs font-semibold uppercase tracking-wider';

// Stands in for the rows scrolled out of a virtualised list, so the scrollbar
// keeps the height of the whole list.
function Spacer({ height }: { height: number }): JSX.Element {
  return (
    <tbody aria-hidden="true">
      <tr>
        <td colSpan={COLUMN_COUNT} className="p-0" style={{ height }} />
      </tr>
    </tbody>
  );
}

/**
 * DiscoveryModal - Full-screen modal for device discovery viewing.
 */
export function DiscoveryModal({
  isOpen,
  onClose,
  data,
  onScan,
  onDeepScan,
}: DiscoveryModalProps): JSX.Element | null {
  const { t } = useTranslation(['cards', 'common']);

  const [searchQuery, setSearchQuery] = useState('');
  const [sortField, setSortField] = useState<SortField | null>('ip');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');
  const [expandedDevices, setExpandedDevices] = useState<Set<string>>(new Set());
  const [scanningDevices, setScanningDevices] = useState<Set<string>>(new Set());
  const [showLocalOnly, setShowLocalOnly] = useState(false);
  const [vulnDeviceIp, setVulnDeviceIp] = useState<string | null>(null);

  // Toggle sort
  const handleSort = useCallback((field: SortField) => {
    setSortField((prev) => {
      if (prev === field) {
        setSortDirection((d) => (d === 'asc' ? 'desc' : 'asc'));
        return field;
      }
      setSortDirection('asc');
      return field;
    });
  }, []);

  // Toggle device expansion
  const toggleDevice = useCallback((key: string) => {
    setExpandedDevices((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  }, []);

  // Deep scan handler
  const handleDeepScan = useCallback(
    async (ip: string): Promise<void> => {
      if (!onDeepScan) {
        return;
      }
      setScanningDevices((prev) => new Set(prev).add(ip));
      try {
        await onDeepScan(ip);
      } finally {
        setScanningDevices((prev) => {
          const next = new Set(prev);
          next.delete(ip);
          return next;
        });
      }
    },
    [onDeepScan],
  );

  // Filter and sort devices
  const filteredDevices = useMemo(() => {
    if (!data?.devices) {
      return [];
    }

    let devices = [...data.devices];

    // Filter by local/extended
    if (showLocalOnly) {
      devices = devices.filter((d) => d.isLocal);
    }

    // Search filter
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      devices = devices.filter(
        (d) =>
          d.ip?.toLowerCase().includes(q) ||
          d.hostname?.toLowerCase().includes(q) ||
          d.netbiosName?.toLowerCase().includes(q) ||
          d.mdnsName?.toLowerCase().includes(q) ||
          d.displayName?.toLowerCase().includes(q) ||
          d.mac?.toLowerCase().includes(q) ||
          d.vendor?.toLowerCase().includes(q),
      );
    }

    // Sort
    if (sortField) {
      devices.sort((a, b) => compareDevices(a, b, sortField, sortDirection));
    }

    return devices;
  }, [data?.devices, searchQuery, sortField, sortDirection, showLocalOnly]);

  // Export functions
  const exportJson = useCallback(() => {
    const blob = new Blob([JSON.stringify(filteredDevices, null, 2)], {
      type: 'application/json',
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `devices-${new Date().toISOString().split('T')[0]}.json`;
    link.click();
    URL.revokeObjectURL(url);
  }, [filteredDevices]);

  const exportCsv = useCallback((): void => {
    const escapeCsv = (val: unknown): string => {
      if (val === null || val === undefined) {
        return '';
      }
      const str = String(val);
      if (/[",\n]/.test(str)) {
        return `"${str.replace(/"/g, '""')}"`;
      }
      return str;
    };

    const rows = filteredDevices.map((d) =>
      [
        escapeCsv(d.ip),
        escapeCsv(d.displayName || d.mdnsName || d.netbiosName || d.hostname),
        escapeCsv(d.netbiosName),
        escapeCsv(d.mdnsName),
        escapeCsv(d.mac),
        escapeCsv(d.vendor),
        escapeCsv(d.discoveryMethod.join(';')),
        escapeCsv(d.lastSeen),
        escapeCsv(d.isLocal ? 'local' : 'extended'),
        escapeCsv(d.osGuess),
      ].join(','),
    );

    const header =
      'ip,name,netbios_name,mdns_name,mac,vendor,discovery_methods,last_seen,network,os_guess';
    const csv = [header, ...rows].join('\n');
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `devices-${new Date().toISOString().split('T')[0]}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  }, [filteredDevices]);

  const dialogRef = useFocusTrap<HTMLDivElement>({ isActive: isOpen, onEscape: onClose });

  const { scrollRef, rows, padTop, padBottom, measureRef, onKeyDown } = useVirtualDeviceRows(
    filteredDevices,
    deviceKey,
  );

  if (!isOpen) {
    return null;
  }

  const deviceCount = data?.devices?.length || 0;
  const localCount = data?.devices?.filter((d) => d.isLocal).length || 0;

  return (
    <div className={modal.overlay}>
      {/* Backdrop */}
      <div className={modal.backdrop} onClick={onClose} aria-hidden="true" />
      {/* Modal - full width */}
      <div
        ref={dialogRef}
        className={cn(
          'relative',
          modal.content,
          modal.size.full,
          modal.padding.lg,
          'flex flex-col',
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby="discovery-modal-title"
      >
        {/* Header */}
        <div className="flex flex-wrap items-center justify-between gap-default mb-content pb-4 border-b border-surface-border">
          <div>
            <h2 id="discovery-modal-title" className="heading-2 text-text-primary">
              {t('discovery.title')}
            </h2>
            <p className="text-sm text-text-muted mt-tight">
              {t('discovery.modalSubtitle', {
                total: deviceCount,
                local: localCount,
              })}
              {data?.status?.subnet ? ` - ${data.status.subnet}` : ''}
            </p>
          </div>

          <div className="flex items-center gap-default">
            {/* Scan button */}
            {onScan ? (
              <button
                type="button"
                onClick={onScan}
                disabled={data?.status?.scanning}
                className={cn(
                  button.base,
                  button.variant.secondary,
                  button.size.sm,
                  'flex items-center gap-compact',
                )}
              >
                <RefreshCw
                  className={cn(iconTokens.size.sm, data?.status?.scanning ? 'animate-spin' : '')}
                />
                {data?.status?.scanning ? t('discovery.scanning') : t('discovery.rescan')}
              </button>
            ) : null}

            {/* Export dropdown */}
            <div className="flex items-center gap-tight">
              <Tooltip text={t('discovery.exportCSV')}>
                <button
                  type="button"
                  onClick={exportCsv}
                  className={cn(
                    button.base,
                    button.variant.ghost,
                    button.size.sm,
                    'flex items-center gap-tight',
                  )}
                >
                  <Download className={iconTokens.size.sm} />
                  CSV
                </button>
              </Tooltip>
              <Tooltip text={t('discovery.exportJSON')}>
                <button
                  type="button"
                  onClick={exportJson}
                  className={cn(
                    button.base,
                    button.variant.ghost,
                    button.size.sm,
                    'flex items-center gap-tight',
                  )}
                >
                  <Download className={iconTokens.size.sm} />
                  JSON
                </button>
              </Tooltip>
            </div>

            {/* Close button */}
            <button
              type="button"
              onClick={onClose}
              className={cn(
                'pad-xs rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors',
              )}
              aria-label={t('common:buttons.close')}
            >
              <X className={iconTokens.size.md} />
            </button>
          </div>
        </div>

        {/* Search and filters */}
        <div className="flex items-center gap-comfortable mb-content">
          {/* Search input */}
          <div className="relative flex-1 max-w-md">
            <Search
              className={cn(
                'absolute left-3 top-1/2 -translate-y-1/2',
                iconTokens.size.sm,
                'text-text-muted',
              )}
            />
            <input
              type="text"
              value={searchQuery}
              onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
                setSearchQuery(e.target.value)
              }
              placeholder={t('discovery.searchPlaceholder')}
              className={cn(
                'w-full pl-10 pr-4 py-row',
                'text-sm bg-surface-base border border-surface-border',
                radius.md,
                'focus:outline-none focus:ring-1 focus:ring-brand-primary text-text-primary placeholder:text-text-muted',
              )}
            />
            {searchQuery ? (
              <Tooltip text={t('discovery.clearSearch')}>
                <button
                  type="button"
                  onClick={(): void => setSearchQuery('')}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
                  aria-label={t('discovery.clearSearch')}
                >
                  <X className={iconTokens.size.sm} />
                </button>
              </Tooltip>
            ) : null}
          </div>

          {/* Filter toggles */}
          <div className="flex items-center gap-compact">
            <button
              type="button"
              onClick={(): void => setShowLocalOnly(!showLocalOnly)}
              className={cn(
                'px-3 py-compact-md text-sm',
                radius.md,
                'transition-colors',
                showLocalOnly
                  ? 'bg-brand-primary text-on-brand'
                  : 'bg-surface-hover text-text-secondary hover:text-text-primary',
              )}
            >
              {t('discovery.localOnly')}
            </button>
          </div>

          {/* Results count */}
          <span className="text-sm text-text-muted">
            {t('discovery.filteredCount', { filtered: filteredDevices.length, total: deviceCount })}
          </span>
        </div>

        {/* Table. The scroll padding keeps a row that takes focus clear of the
            sticky header. The virtualiser keeps the scroll position itself;
            left to the browser's scroll anchoring, a long scroll moved the
            list a second time once the rows and spacers were swapped
            (30,000 px landed near 58,000 on Chromium, #2962). */}
        <div
          ref={scrollRef}
          className="@container flex-1 overflow-auto scroll-pt-12 [overflow-anchor:none]"
        >
          <table className="w-full table-fixed" onKeyDown={onKeyDown} data-testid="discovery-table">
            <thead className="bg-surface-base sticky top-0 z-10">
              <tr className="border-b border-surface-border">
                <SortableHeader
                  label={t('discovery.tableIp')}
                  field="ip"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  className="w-36"
                />
                <SortableHeader
                  label={t('discovery.tableHostname')}
                  field="hostname"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                />
                <SortableHeader
                  label={t('discovery.tableMac')}
                  field="mac"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  className={column.mac.cell}
                />
                <SortableHeader
                  label={t('discovery.tableVendor')}
                  field="vendor"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  className={column.vendor.cell}
                />
                <th className={cn(headerCell, column.methods.cell)}>
                  {t('discovery.tableDiscovery')}
                </th>
                <th className={cn(headerCell, column.ports.cell)}>{t('discovery.tablePorts')}</th>
                <th className={cn(headerCell, column.vulns.cell)}>{t('discovery.tableVulns')}</th>
                <SortableHeader
                  label={t('discovery.tableLastSeen')}
                  field="lastSeen"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  className={column.lastSeen.cell}
                />
                <th className={cn(headerCell, 'w-24')}>{t('discovery.tableActions')}</th>
              </tr>
            </thead>
            {padTop > 0 ? <Spacer height={padTop} /> : null}
            {rows.map(({ index, item: device }) => {
              const key = deviceKey(device);
              return (
                <DeviceRow
                  key={key}
                  index={index}
                  measureRef={measureRef}
                  device={device}
                  isExpanded={expandedDevices.has(key)}
                  onToggle={(): void => toggleDevice(key)}
                  onDeepScan={onDeepScan ? handleDeepScan : undefined}
                  onShowVulnerabilities={setVulnDeviceIp}
                  isScanning={scanningDevices.has(device.ip)}
                />
              );
            })}
            {padBottom > 0 ? <Spacer height={padBottom} /> : null}
          </table>

          {/* Empty state */}
          {filteredDevices.length === 0 ? (
            <div className="text-center py-centered text-text-muted">
              {searchQuery || showLocalOnly ? t('discovery.noResults') : t('discovery.noDevices')}
            </div>
          ) : null}
        </div>
      </div>
      {vulnDeviceIp ? (
        <VulnerabilityDetailsModal
          deviceIp={vulnDeviceIp}
          onClose={(): void => setVulnDeviceIp(null)}
        />
      ) : null}
    </div>
  );
}
