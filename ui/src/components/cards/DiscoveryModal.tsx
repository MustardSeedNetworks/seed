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
 * - Export to CSV/JSON
 * - Keyboard support (Escape to close)
 */

import type React from 'react';
import type { JSX } from 'react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { button, cn, icon as iconTokens, modal, radius } from '../../styles/theme';
import { ArrowUpDown, ChevronDown, ChevronUp, Download, RefreshCw, Search, X } from '../ui/icons';
import { DeviceRow } from './DiscoveryModalDeviceRow';
import type { DiscoveredDevice, NetworkDiscoveryData } from './NetworkDiscoveryCard';

interface DiscoveryModalProps {
  isOpen: boolean;
  onClose: () => void;
  data: NetworkDiscoveryData | null;
  onScan?: () => void;
  onDeepScan?: (ip: string) => Promise<void>;
}

type SortField = 'ip' | 'hostname' | 'vendor' | 'mac' | 'lastSeen';
type SortDirection = 'asc' | 'desc';

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
  return (
    <th
      className={cn(
        'px-3 py-row text-left text-xs font-semibold uppercase tracking-wider cursor-pointer hover:bg-surface-hover transition-colors select-none',
        className,
      )}
      onClick={() => onSort(field)}
    >
      <div className="flex items-center gap-tight">
        <span>{label}</span>
        {getSortIcon(isActive, direction)}
      </div>
    </th>
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
  const { t } = useTranslation('cards');

  const [searchQuery, setSearchQuery] = useState('');
  const [sortField, setSortField] = useState<SortField | null>('ip');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');
  const [expandedDevices, setExpandedDevices] = useState<Set<string>>(new Set());
  const [scanningDevices, setScanningDevices] = useState<Set<string>>(new Set());
  const [showLocalOnly, setShowLocalOnly] = useState(false);

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

  // Keyboard handler for Escape
  useEffect((): (() => void) | undefined => {
    if (!isOpen) {
      return;
    }

    const handleKeyDown = (e: KeyboardEvent): void => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return (): void => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

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
        <div className="flex-between mb-content pb-4 border-b border-surface-border">
          <div>
            <h2 id="discovery-modal-title" className="heading-2 text-text-primary">
              {t('discovery.title', 'Network Discovery')}
            </h2>
            <p className="text-sm text-text-muted mt-tight">
              {t('discovery.modalSubtitle', '{{total}} devices ({{local}} local)', {
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
              <button
                type="button"
                onClick={exportCsv}
                className={cn(
                  button.base,
                  button.variant.ghost,
                  button.size.sm,
                  'flex items-center gap-tight',
                )}
                title="Export as CSV"
              >
                <Download className={iconTokens.size.sm} />
                CSV
              </button>
              <button
                type="button"
                onClick={exportJson}
                className={cn(
                  button.base,
                  button.variant.ghost,
                  button.size.sm,
                  'flex items-center gap-tight',
                )}
                title="Export as JSON"
              >
                <Download className={iconTokens.size.sm} />
                JSON
              </button>
            </div>

            {/* Close button */}
            <button
              type="button"
              onClick={onClose}
              className={cn(
                'pad-xs rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors',
              )}
              aria-label="Close"
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
              placeholder={t('discovery.searchPlaceholder', 'Search IP, hostname, MAC, vendor...')}
              className={cn(
                'w-full pl-10 pr-4 py-row',
                'text-sm bg-surface-base border border-surface-border',
                radius.md,
                'focus:outline-none focus:ring-1 focus:ring-brand-primary text-text-primary placeholder:text-text-muted',
              )}
            />
            {searchQuery ? (
              <button
                type="button"
                onClick={(): void => setSearchQuery('')}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
                aria-label={t('discovery.clearSearch', 'Clear search')}
                title={t('discovery.clearSearch', 'Clear search')}
              >
                <X className={iconTokens.size.sm} />
              </button>
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
              {t('discovery.localOnly', 'Local Only')}
            </button>
          </div>

          {/* Results count */}
          <span className="text-sm text-text-muted">
            {filteredDevices.length} of {deviceCount} devices
          </span>
        </div>

        {/* Table */}
        <div className="flex-1 overflow-auto">
          <table className="w-full min-w-[900px]">
            <thead className="bg-surface-base sticky top-0">
              <tr className="border-b border-surface-border">
                <SortableHeader
                  label={t('discovery.tableIp', 'IP Address')}
                  field="ip"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  className="w-40"
                />
                <SortableHeader
                  label={t('discovery.tableHostname', 'Hostname')}
                  field="hostname"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  className="w-40"
                />
                <SortableHeader
                  label={t('discovery.tableMac', 'MAC')}
                  field="mac"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  className="w-36"
                />
                <SortableHeader
                  label={t('discovery.tableVendor', 'Vendor')}
                  field="vendor"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  className="w-32"
                />
                <th className="px-3 py-row text-left text-xs font-semibold uppercase tracking-wider w-28">
                  {t('discovery.tableDiscovery', 'Discovery')}
                </th>
                <th className="px-3 py-row text-left text-xs font-semibold uppercase tracking-wider w-20">
                  {t('discovery.tablePorts', 'Ports')}
                </th>
                <th className="px-3 py-row text-left text-xs font-semibold uppercase tracking-wider w-20">
                  {t('discovery.tableVulns', 'CVEs')}
                </th>
                <SortableHeader
                  label={t('discovery.tableLastSeen', 'Last Seen')}
                  field="lastSeen"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  className="w-24"
                />
                <th className="px-3 py-row text-left text-xs font-semibold uppercase tracking-wider w-24">
                  {t('discovery.tableActions', 'Actions')}
                </th>
              </tr>
            </thead>
            <tbody className="text-text-primary">
              {filteredDevices.map((device) => {
                const deviceKey = device.mac || `ip:${device.ip}`;
                return (
                  <DeviceRow
                    key={deviceKey}
                    device={device}
                    isExpanded={expandedDevices.has(deviceKey)}
                    onToggle={(): void => toggleDevice(deviceKey)}
                    onDeepScan={onDeepScan ? handleDeepScan : undefined}
                    isScanning={scanningDevices.has(device.ip)}
                  />
                );
              })}
            </tbody>
          </table>

          {/* Empty state */}
          {filteredDevices.length === 0 ? (
            <div className="text-center py-centered text-text-muted">
              {searchQuery || showLocalOnly
                ? t('discovery.noResults', 'No devices match your filters')
                : t('discovery.noDevices', 'No devices discovered yet')}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
