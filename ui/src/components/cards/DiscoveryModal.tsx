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
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  button,
  cn,
  discoveryMethod as discoveryMethodTheme,
  icon as iconTokens,
  modal,
  radius,
  severity as severityTheme,
} from '../../styles/theme';
import {
  AlertTriangle,
  ArrowUpDown,
  ChevronDown,
  ChevronUp,
  Download,
  RefreshCw,
  Search,
  X,
} from '../ui/icons';
import { Tooltip } from '../ui/tooltip';
import type {
  DiscoveredDevice,
  DiscoveryMethod,
  NetworkDiscoveryData,
  OpenPort,
} from './NetworkDiscoveryCard';

interface DiscoveryModalProps {
  isOpen: boolean;
  onClose: () => void;
  data: NetworkDiscoveryData | null;
  onScan?: () => void;
  onDeepScan?: (ip: string) => Promise<void>;
}

type SortField = 'ip' | 'hostname' | 'vendor' | 'mac' | 'lastSeen';
type SortDirection = 'asc' | 'desc';

// Discovery method badge
function _methodBadge({ method }: { method: DiscoveryMethod }): JSX.Element {
  const theme = discoveryMethodTheme[method] || discoveryMethodTheme.arp;
  return (
    <span
      class={cn('px-1.5 py-0.5 text-xs font-medium uppercase', radius.md, theme.bg, theme.text)}
    >
      {method}
    </span>
  );
}

// Format SNMP sysUpTime (in hundredths of a second) to human-readable duration
function formatUptime(ticks: number): string {
  const seconds = Math.floor(ticks / 100);
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);

  if (days > 0) {
    return `${days}d ${hours}h ${minutes}m`;
  }
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  return `${minutes}m`;
}

// Format timestamp for display
function formatLastSeen(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSecs = Math.floor(diffMs / 1000);
  const diffMins = Math.floor(diffSecs / 60);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffSecs < 60) {
    return 'Just now';
  }
  if (diffMins < 60) {
    return `${diffMins}m ago`;
  }
  if (diffHours < 24) {
    return `${diffHours}h ago`;
  }
  if (diffDays < 7) {
    return `${diffDays}d ago`;
  }
  return date.toLocaleDateString();
}

// Sort comparator
// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Comparison function handles multiple sort fields
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

// Helper function to get expand icon (avoids nested ternary)
function getExpandIcon(hasDetails: boolean, isExpanded: boolean): string {
  if (!hasDetails) {
    return '';
  }
  if (isExpanded) {
    return '▲';
  }
  return '▼';
}

// Helper function to get severity theme classes (avoids nested ternary)
function getSeverityClasses(severity: string): string {
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

// Helper function to get sort icon (avoids nested ternary)
function getSortIcon(isActive: boolean, direction: SortDirection): JSX.Element {
  if (!isActive) {
    return <ArrowUpDown class="w-3 h-3 opacity-30" />;
  }
  if (direction === 'asc') {
    return <ChevronUp class="w-3 h-3" />;
  }
  return <ChevronDown class="w-3 h-3" />;
}

// Table header with sort indicator
function _sortableHeader({
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
      class={cn(
        'px-3 py-2 text-left text-xs font-semibold uppercase tracking-wider cursor-pointer hover:bg-surface-hover transition-colors select-none',
        className,
      )}
      onClick={() => onSort(field)}
    >
      <div class="flex items-center gap-1">
        <span>{label}</span>
        {getSortIcon(isActive, direction)}
      </div>
    </th>
  );
}

// Device row component
// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Device row handles many device types and states
function _deviceRow({
  device,
  isExpanded,
  onToggle,
  onDeepScan,
  isScanning,
}: {
  device: DiscoveredDevice;
  isExpanded: boolean;
  onToggle: () => void;
  onDeepScan?: (ip: string) => Promise<void>;
  isScanning: boolean;
}): JSX.Element {
  const { t } = useTranslation('cards');
  const openPorts = device.profile?.openPorts?.filter((p) => p.isOpen) || [];
  const hasDetails =
    device.lldpInfo ||
    device.cdpInfo ||
    device.edpInfo ||
    device.ndpInfo ||
    device.snmpData ||
    openPorts.length > 0;

  const handleScan = async (e: React.MouseEvent): Promise<void> => {
    e.stopPropagation();
    if (onDeepScan && device.ip) {
      // biome-ignore lint/nursery/useAwaitThenable: onDeepScan returns Promise<void>
      await onDeepScan(device.ip);
    }
  };

  return (
    <>
      <tr
        class={cn(
          'border-b border-surface-border hover:bg-surface-hover cursor-pointer transition-colors',
          isExpanded && 'bg-surface-hover',
        )}
        onClick={onToggle}
      >
        {/* IP Address */}
        <td class="px-3 py-2">
          <div class="flex flex-col">
            <span class="font-mono text-sm font-medium text-text-primary">
              {device.ip || t('network.noIP')}
            </span>
            {device.ipv6 ? (
              <span class="font-mono text-xs text-text-muted truncate max-w-40" title={device.ipv6}>
                {device.ipv6.length > 25 ? `${device.ipv6.substring(0, 25)}...` : device.ipv6}
              </span>
            ) : null}
          </div>
        </td>

        {/* Hostname - prefer displayName, fallback to mdnsName, netbiosName, hostname */}
        <td class="px-3 py-2">
          <span
            class="text-sm text-text-secondary truncate block max-w-40"
            title={device.displayName || device.mdnsName || device.netbiosName || device.hostname}
          >
            {device.displayName || device.mdnsName || device.netbiosName || device.hostname || '-'}
          </span>
        </td>

        {/* MAC Address */}
        <td class="px-3 py-2">
          <span class="font-mono text-xs text-text-muted">{device.mac || '-'}</span>
        </td>

        {/* Vendor */}
        <td class="px-3 py-2">
          {device.vendor === 'LAA' ? (
            <Tooltip
              content="Locally Administered Address - MAC assigned locally rather than by manufacturer"
              position="bottom"
            >
              <span class="text-xs text-text-muted underline decoration-dotted cursor-help">
                LAA
              </span>
            </Tooltip>
          ) : (
            <span class="text-xs text-text-muted truncate block max-w-28" title={device.vendor}>
              {device.vendor || '-'}
            </span>
          )}
        </td>

        {/* Discovery Methods */}
        <td class="px-3 py-2">
          <div class="flex items-center gap-1 flex-wrap">
            {device.discoveryMethod.map((method) => (
              <methodBadge key={method} method={method} />
            ))}
          </div>
        </td>

        {/* Open Ports */}
        <td class="px-3 py-2">
          {openPorts.length > 0 ? (
            <span
              class={cn(
                'text-xs px-1.5 py-0.5 bg-status-success/20 text-status-success',
                radius.md,
              )}
            >
              {openPorts.length} open
            </span>
          ) : (
            <span class="text-xs text-text-muted">-</span>
          )}
        </td>

        {/* Vulnerabilities */}
        <td class="px-3 py-2">
          {device.vulnerabilities && device.vulnerabilities.count > 0 ? (
            <span
              class={cn(
                'inline-flex items-center gap-1 text-xs px-1.5 py-0.5',
                radius.md,
                getSeverityClasses(device.vulnerabilities.highestSeverity),
              )}
            >
              <AlertTriangle class="w-3 h-3" />
              {device.vulnerabilities.count}
            </span>
          ) : (
            <span class="text-xs text-text-muted">-</span>
          )}
        </td>

        {/* Last Seen */}
        <td class="px-3 py-2">
          <span class="text-xs text-text-muted">{formatLastSeen(device.lastSeen)}</span>
        </td>

        {/* Actions */}
        <td class="px-3 py-2">
          <div class="flex items-center gap-2">
            {onDeepScan && device.ip ? (
              <button
                type="button"
                onClick={handleScan}
                disabled={isScanning}
                class={cn(
                  'text-xs px-2 py-1 bg-brand-primary/20 text-brand-primary',
                  radius.md,
                  'hover:bg-brand-primary/30 transition-colors disabled:opacity-50',
                )}
              >
                {isScanning ? '...' : t('discovery.scan')}
              </button>
            ) : null}
            <span class="text-xs text-text-muted">{getExpandIcon(hasDetails, isExpanded)}</span>
          </div>
        </td>
      </tr>

      {/* Expanded details row */}
      {isExpanded && hasDetails ? (
        <tr class="bg-surface-sunken">
          <td colSpan={9} class="px-4 py-3">
            <div class="space-y-3">
              {/* Open Ports */}
              {openPorts.length > 0 ? (
                <div>
                  <h4 class="text-xs font-semibold text-text-secondary mb-1">Open Ports</h4>
                  <div class="flex flex-wrap gap-2">
                    {openPorts.map((port: OpenPort) => (
                      <span
                        key={port.port}
                        class={cn(
                          'px-2 py-1 text-xs font-mono',
                          radius.md,
                          'bg-surface-base text-text-primary',
                        )}
                      >
                        {port.port}/{port.protocol}{' '}
                        {port.service ? (
                          <span class="text-text-muted">({port.service})</span>
                        ) : null}
                      </span>
                    ))}
                  </div>
                </div>
              ) : null}

              {/* LLDP Info */}
              {device.lldpInfo ? (
                <div>
                  <h4 class="text-xs font-semibold text-text-secondary mb-1">LLDP Information</h4>
                  <div class="grid grid-cols-2 md:grid-cols-4 gap-2 text-xs">
                    <div>
                      <span class="text-text-muted">System:</span> {device.lldpInfo.systemName}
                    </div>
                    <div>
                      <span class="text-text-muted">Port:</span> {device.lldpInfo.portId}
                    </div>
                    {device.lldpInfo.managementAddress ? (
                      <div>
                        <span class="text-text-muted">Mgmt IP:</span>{' '}
                        {device.lldpInfo.managementAddress}
                      </div>
                    ) : null}
                    {device.lldpInfo.capabilities ? (
                      <div>
                        <span class="text-text-muted">Capabilities:</span>{' '}
                        {device.lldpInfo.capabilities.join(', ')}
                      </div>
                    ) : null}
                  </div>
                </div>
              ) : null}

              {/* CDP Info */}
              {device.cdpInfo ? (
                <div>
                  <h4 class="text-xs font-semibold text-text-secondary mb-1">CDP Information</h4>
                  <div class="grid grid-cols-2 md:grid-cols-4 gap-2 text-xs">
                    <div>
                      <span class="text-text-muted">Device:</span> {device.cdpInfo.deviceId}
                    </div>
                    <div>
                      <span class="text-text-muted">Platform:</span> {device.cdpInfo.platform}
                    </div>
                    {device.cdpInfo.nativeVlan ? (
                      <div>
                        <span class="text-text-muted">Native VLAN:</span>{' '}
                        {device.cdpInfo.nativeVlan}
                      </div>
                    ) : null}
                  </div>
                </div>
              ) : null}

              {/* SNMP Data */}
              {device.snmpData ? (
                <div class="space-y-2">
                  <h4 class="text-xs font-semibold text-text-secondary">
                    {t('discovery.snmpInfo', 'SNMP Details')}
                  </h4>

                  {/* System Info */}
                  {device.snmpData.system ? (
                    <div class="bg-surface-base p-2 rounded-md">
                      <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-x-4 gap-y-1 text-xs">
                        {device.snmpData.system.sysName ? (
                          <div>
                            <span class="text-text-muted">Name:</span>{' '}
                            <span class="text-text-primary font-medium">
                              {device.snmpData.system.sysName}
                            </span>
                          </div>
                        ) : null}
                        {device.snmpData.system.sysDescr ? (
                          <div class="col-span-2">
                            <span class="text-text-muted">Description:</span>{' '}
                            <span class="text-text-primary">
                              {device.snmpData.system.sysDescr.length > 80
                                ? `${device.snmpData.system.sysDescr.substring(0, 80)}...`
                                : device.snmpData.system.sysDescr}
                            </span>
                          </div>
                        ) : null}
                        {device.snmpData.system.sysLocation ? (
                          <div>
                            <span class="text-text-muted">Location:</span>{' '}
                            <span class="text-text-primary">
                              {device.snmpData.system.sysLocation}
                            </span>
                          </div>
                        ) : null}
                        {device.snmpData.system.sysContact ? (
                          <div>
                            <span class="text-text-muted">Contact:</span>{' '}
                            <span class="text-text-primary">
                              {device.snmpData.system.sysContact}
                            </span>
                          </div>
                        ) : null}
                        {device.snmpData.system.sysUpTime !== undefined &&
                        device.snmpData.system.sysUpTime > 0 ? (
                          <div>
                            <span class="text-text-muted">Uptime:</span>{' '}
                            <span class="text-text-primary">
                              {formatUptime(device.snmpData.system.sysUpTime)}
                            </span>
                          </div>
                        ) : null}
                      </div>
                    </div>
                  ) : null}

                  {/* Interfaces Summary */}
                  {device.snmpData.interfaces && device.snmpData.interfaces.length > 0 ? (
                    <div>
                      <span class="text-xs text-text-muted">
                        Interfaces ({device.snmpData.interfaces.length}):
                      </span>
                      <div class="flex flex-wrap gap-1 mt-1">
                        {/* biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Complex interface status rendering */}
                        {device.snmpData.interfaces.slice(0, 8).map((iface) => (
                          <span
                            key={iface.name}
                            class={cn(
                              'px-1.5 py-0.5 text-xs',
                              radius.sm,
                              iface.operStatus === 'up'
                                ? 'bg-status-success/20 text-status-success'
                                : 'bg-surface-hover text-text-muted',
                            )}
                            title={`${iface.name} - ${iface.speed ? `${Math.round(iface.speed / 1000000)} Mbps` : 'N/A'}`}
                          >
                            {iface.name}
                            {iface.speed && iface.speed > 0 ? (
                              <span class="text-text-muted ml-1">
                                {iface.speed >= 1000000000
                                  ? `${Math.round(iface.speed / 1000000000)}G`
                                  : `${Math.round(iface.speed / 1000000)}M`}
                              </span>
                            ) : null}
                          </span>
                        ))}
                        {device.snmpData.interfaces.length > 8 ? (
                          <span class="text-xs text-text-muted">
                            +{device.snmpData.interfaces.length - 8} more
                          </span>
                        ) : null}
                      </div>
                    </div>
                  ) : null}

                  {/* VLANs Summary */}
                  {device.snmpData.vlans && device.snmpData.vlans.length > 0 ? (
                    <div>
                      <span class="text-xs text-text-muted">
                        VLANs ({device.snmpData.vlans.length}):
                      </span>
                      <div class="flex flex-wrap gap-1 mt-1">
                        {device.snmpData.vlans.slice(0, 12).map((vlan) => (
                          <span
                            key={vlan.id}
                            class={cn(
                              'px-1.5 py-0.5 text-xs bg-brand-primary/10 text-brand-primary',
                              radius.sm,
                            )}
                            title={vlan.name || `VLAN ${vlan.id}`}
                          >
                            {vlan.id}
                            {vlan.name && vlan.name !== `VLAN${vlan.id}` ? (
                              <span class="text-text-muted ml-1">
                                {vlan.name.length > 10
                                  ? `${vlan.name.substring(0, 10)}...`
                                  : vlan.name}
                              </span>
                            ) : null}
                          </span>
                        ))}
                        {device.snmpData.vlans.length > 12 ? (
                          <span class="text-xs text-text-muted">
                            +{device.snmpData.vlans.length - 12} more
                          </span>
                        ) : null}
                      </div>
                    </div>
                  ) : null}

                  {/* Hardware Inventory */}
                  {device.snmpData.entities && device.snmpData.entities.length > 0 ? (
                    <div>
                      <span class="text-xs text-text-muted">Hardware:</span>
                      <div class="grid grid-cols-1 md:grid-cols-2 gap-1 mt-1 text-xs">
                        {device.snmpData.entities
                          .filter(
                            (e) =>
                              e.physicalClass === 'chassis' ||
                              e.physicalClass === 'module' ||
                              e.physicalClass === 'powerSupply',
                          )
                          .slice(0, 4)
                          .map((entity) => (
                            <div
                              key={entity.serialNum || entity.name || entity.description}
                              class="bg-surface-hover px-2 py-1 rounded"
                            >
                              <span class="text-text-primary">
                                {entity.name || entity.description}
                              </span>
                              {entity.serialNum ? (
                                <span class="text-text-muted ml-2">S/N: {entity.serialNum}</span>
                              ) : null}
                              {entity.modelName ? (
                                <span class="text-text-muted ml-2">Model: {entity.modelName}</span>
                              ) : null}
                            </div>
                          ))}
                      </div>
                    </div>
                  ) : null}
                </div>
              ) : null}

              {/* OS Guess */}
              {device.osGuess ? (
                <div>
                  <span class="text-xs text-text-muted">OS Guess:</span>{' '}
                  <span class="text-xs text-text-primary">{device.osGuess}</span>
                </div>
              ) : null}
            </div>
          </td>
        </tr>
      ) : null}
    </>
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
        // biome-ignore lint/nursery/useAwaitThenable: onDeepScan returns Promise<void>
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
    <div class={modal.overlay}>
      {/* Backdrop */}
      <div class={modal.backdrop} onClick={onClose} aria-hidden="true" />

      {/* Modal - full width */}
      <div
        class={cn('relative', modal.content, modal.size.full, modal.padding.lg, 'flex flex-col')}
        role="dialog"
        aria-modal="true"
        aria-labelledby="discovery-modal-title"
      >
        {/* Header */}
        <div class="flex items-center justify-between mb-4 pb-4 border-b border-surface-border">
          <div>
            <h2 id="discovery-modal-title" class="text-xl font-semibold text-text-primary">
              {t('discovery.title', 'Network Discovery')}
            </h2>
            <p class="text-sm text-text-muted mt-1">
              {t('discovery.modalSubtitle', '{{total}} devices ({{local}} local)', {
                total: deviceCount,
                local: localCount,
              })}
              {data?.status?.subnet ? ` - ${data.status.subnet}` : ''}
            </p>
          </div>

          <div class="flex items-center gap-3">
            {/* Scan button */}
            {onScan ? (
              <button
                type="button"
                onClick={onScan}
                disabled={data?.status?.scanning}
                class={cn(
                  button.base,
                  button.variant.secondary,
                  button.size.sm,
                  'flex items-center gap-2',
                )}
              >
                <RefreshCw
                  class={cn(iconTokens.size.sm, data?.status?.scanning ? 'animate-spin' : '')}
                />
                {data?.status?.scanning ? t('discovery.scanning') : t('discovery.rescan')}
              </button>
            ) : null}

            {/* Export dropdown */}
            <div class="flex items-center gap-1">
              <button
                type="button"
                onClick={exportCsv}
                class={cn(
                  button.base,
                  button.variant.ghost,
                  button.size.sm,
                  'flex items-center gap-1',
                )}
                title="Export as CSV"
              >
                <Download class={iconTokens.size.sm} />
                CSV
              </button>
              <button
                type="button"
                onClick={exportJson}
                class={cn(
                  button.base,
                  button.variant.ghost,
                  button.size.sm,
                  'flex items-center gap-1',
                )}
                title="Export as JSON"
              >
                <Download class={iconTokens.size.sm} />
                JSON
              </button>
            </div>

            {/* Close button */}
            <button
              type="button"
              onClick={onClose}
              class={cn(
                'p-2 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors',
              )}
              aria-label="Close"
            >
              <X class={iconTokens.size.md} />
            </button>
          </div>
        </div>

        {/* Search and filters */}
        <div class="flex items-center gap-4 mb-4">
          {/* Search input */}
          <div class="relative flex-1 max-w-md">
            <Search
              class={cn(
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
              class={cn(
                'w-full pl-10 pr-4 py-2',
                'text-sm bg-surface-base border border-surface-border',
                radius.md,
                'focus:outline-none focus:ring-1 focus:ring-brand-primary text-text-primary placeholder:text-text-muted',
              )}
            />
            {searchQuery ? (
              <button
                type="button"
                onClick={(): void => setSearchQuery('')}
                class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
              >
                <X class={iconTokens.size.sm} />
              </button>
            ) : null}
          </div>

          {/* Filter toggles */}
          <div class="flex items-center gap-2">
            <button
              type="button"
              onClick={(): void => setShowLocalOnly(!showLocalOnly)}
              class={cn(
                'px-3 py-1.5 text-sm',
                radius.md,
                'transition-colors',
                showLocalOnly
                  ? 'bg-brand-primary text-text-inverse'
                  : 'bg-surface-hover text-text-secondary hover:text-text-primary',
              )}
            >
              {t('discovery.localOnly', 'Local Only')}
            </button>
          </div>

          {/* Results count */}
          <span class="text-sm text-text-muted">
            {filteredDevices.length} of {deviceCount} devices
          </span>
        </div>

        {/* Table */}
        <div class="flex-1 overflow-auto">
          <table class="w-full min-w-[900px]">
            <thead class="bg-surface-base sticky top-0">
              <tr class="border-b border-surface-border">
                <sortableHeader
                  label={t('discovery.tableIp', 'IP Address')}
                  field="ip"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  class="w-40"
                />
                <sortableHeader
                  label={t('discovery.tableHostname', 'Hostname')}
                  field="hostname"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  class="w-40"
                />
                <sortableHeader
                  label={t('discovery.tableMac', 'MAC')}
                  field="mac"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  class="w-36"
                />
                <sortableHeader
                  label={t('discovery.tableVendor', 'Vendor')}
                  field="vendor"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  class="w-32"
                />
                <th class="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wider w-28">
                  {t('discovery.tableDiscovery', 'Discovery')}
                </th>
                <th class="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wider w-20">
                  {t('discovery.tablePorts', 'Ports')}
                </th>
                <th class="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wider w-20">
                  {t('discovery.tableVulns', 'CVEs')}
                </th>
                <sortableHeader
                  label={t('discovery.tableLastSeen', 'Last Seen')}
                  field="lastSeen"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  class="w-24"
                />
                <th class="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wider w-24">
                  {t('discovery.tableActions', 'Actions')}
                </th>
              </tr>
            </thead>
            <tbody class="text-text-primary">
              {filteredDevices.map((device) => {
                const deviceKey = device.mac || `ip:${device.ip}`;
                return (
                  <deviceRow
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
            <div class="text-center py-12 text-text-muted">
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
