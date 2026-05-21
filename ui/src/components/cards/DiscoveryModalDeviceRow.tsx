/**
 * DiscoveryModal device row and inline badges.
 *
 * Renders a single device row in the discovery modal table, plus the
 * expandable detail panel (open ports, LLDP/CDP, SNMP, OS guess).
 */

import type React from 'react';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import {
  cn,
  discoveryMethod as discoveryMethodTheme,
  radius,
  severity as severityTheme,
} from '../../styles/theme';
import { AlertTriangle } from '../ui/icons';
import { Tooltip } from '../ui/tooltip';
import type { DiscoveredDevice, DiscoveryMethod, OpenPort } from './NetworkDiscoveryCard';

// Discovery method badge
export function MethodBadge({ method }: { method: DiscoveryMethod }): JSX.Element {
  const themeClass = discoveryMethodTheme[method] ?? discoveryMethodTheme.arp;
  return (
    <span className={cn('px-1.5 py-0.5 text-xs font-medium uppercase', radius.md, themeClass)}>
      {method}
    </span>
  );
}

// Format SNMP sysUpTime (in hundredths of a second) to human-readable duration
export function formatUptime(ticks: number): string {
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
export function formatLastSeen(timestamp: string): string {
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

// Helper function to get expand icon (avoids nested ternary)
export function getExpandIcon(hasDetails: boolean, isExpanded: boolean): string {
  if (!hasDetails) {
    return '';
  }
  if (isExpanded) {
    return '▲';
  }
  return '▼';
}

// Helper function to get severity theme classes (avoids nested ternary)
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

// Device row component
// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Device row handles many device types and states
export function DeviceRow({
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
  const hasDetails = Boolean(
    device.lldpInfo ||
      device.cdpInfo ||
      device.edpInfo ||
      device.ndpInfo ||
      device.snmpData ||
      openPorts.length > 0,
  );

  const handleScan = async (e: React.MouseEvent): Promise<void> => {
    e.stopPropagation();
    if (onDeepScan && device.ip) {
      await onDeepScan(device.ip);
    }
  };

  return (
    <>
      <tr
        className={cn(
          'border-b border-surface-border hover:bg-surface-hover cursor-pointer transition-colors',
          isExpanded && 'bg-surface-hover',
        )}
        onClick={onToggle}
      >
        {/* IP Address */}
        <td className="px-3 py-2">
          <div className="flex flex-col">
            <span className="font-mono text-sm font-medium text-text-primary">
              {device.ip || t('network.noIP')}
            </span>
            {device.ipv6 ? (
              <span
                className="font-mono text-xs text-text-muted truncate max-w-40"
                title={device.ipv6}
              >
                {device.ipv6.length > 25 ? `${device.ipv6.substring(0, 25)}...` : device.ipv6}
              </span>
            ) : null}
          </div>
        </td>

        {/* Hostname - prefer displayName, fallback to mdnsName, netbiosName, hostname */}
        <td className="px-3 py-2">
          <span
            className="text-sm text-text-secondary truncate block max-w-40"
            title={device.displayName || device.mdnsName || device.netbiosName || device.hostname}
          >
            {device.displayName || device.mdnsName || device.netbiosName || device.hostname || '-'}
          </span>
        </td>

        {/* MAC Address */}
        <td className="px-3 py-2">
          <span className="font-mono text-xs text-text-muted">{device.mac || '-'}</span>
        </td>

        {/* Vendor */}
        <td className="px-3 py-2">
          {device.vendor === 'LAA' ? (
            <Tooltip
              content="Locally Administered Address - MAC assigned locally rather than by manufacturer"
              position="bottom"
            >
              <span className="text-xs text-text-muted underline decoration-dotted cursor-help">
                LAA
              </span>
            </Tooltip>
          ) : (
            <span className="text-xs text-text-muted truncate block max-w-28" title={device.vendor}>
              {device.vendor || '-'}
            </span>
          )}
        </td>

        {/* Discovery Methods */}
        <td className="px-3 py-2">
          <div className="flex items-center gap-1 flex-wrap">
            {device.discoveryMethod.map((method) => (
              <MethodBadge key={method} method={method} />
            ))}
          </div>
        </td>

        {/* Open Ports */}
        <td className="px-3 py-2">
          {openPorts.length > 0 ? (
            <span
              className={cn(
                'text-xs px-1.5 py-0.5 bg-status-success/20 text-status-success',
                radius.md,
              )}
            >
              {openPorts.length} open
            </span>
          ) : (
            <span className="text-xs text-text-muted">-</span>
          )}
        </td>

        {/* Vulnerabilities */}
        <td className="px-3 py-2">
          {device.vulnerabilities && device.vulnerabilities.count > 0 ? (
            <span
              className={cn(
                'inline-flex items-center gap-1 text-xs px-1.5 py-0.5',
                radius.md,
                getSeverityClasses(device.vulnerabilities.highestSeverity),
              )}
            >
              <AlertTriangle className="w-3 h-3" />
              {device.vulnerabilities.count}
            </span>
          ) : (
            <span className="text-xs text-text-muted">-</span>
          )}
        </td>

        {/* Last Seen */}
        <td className="px-3 py-2">
          <span className="text-xs text-text-muted">{formatLastSeen(device.lastSeen)}</span>
        </td>

        {/* Actions */}
        <td className="px-3 py-2">
          <div className="flex items-center gap-2">
            {onDeepScan && device.ip ? (
              <button
                type="button"
                onClick={handleScan}
                disabled={isScanning}
                className={cn(
                  'text-xs px-2 py-1 bg-brand-primary/20 text-brand-primary',
                  radius.md,
                  'hover:bg-brand-primary/30 transition-colors disabled:opacity-50',
                )}
              >
                {isScanning ? '...' : t('discovery.scan')}
              </button>
            ) : null}
            <span className="text-xs text-text-muted">{getExpandIcon(hasDetails, isExpanded)}</span>
          </div>
        </td>
      </tr>
      {/* Expanded details row */}
      {isExpanded && hasDetails ? (
        <tr className="bg-surface-sunken">
          <td colSpan={9} className="px-4 py-3">
            <div className="space-y-3">
              {/* Open Ports */}
              {openPorts.length > 0 ? (
                <div>
                  <h4 className="text-xs font-semibold text-text-secondary mb-1">Open Ports</h4>
                  <div className="flex flex-wrap gap-2">
                    {openPorts.map((port: OpenPort) => (
                      <span
                        key={port.port}
                        className={cn(
                          'px-2 py-1 text-xs font-mono',
                          radius.md,
                          'bg-surface-base text-text-primary',
                        )}
                      >
                        {port.port}/{port.protocol}{' '}
                        {port.service ? (
                          <span className="text-text-muted">({port.service})</span>
                        ) : null}
                      </span>
                    ))}
                  </div>
                </div>
              ) : null}

              {/* LLDP Info */}
              {device.lldpInfo ? (
                <div>
                  <h4 className="text-xs font-semibold text-text-secondary mb-1">
                    LLDP Information
                  </h4>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-2 text-xs">
                    <div>
                      <span className="text-text-muted">System:</span> {device.lldpInfo.systemName}
                    </div>
                    <div>
                      <span className="text-text-muted">Port:</span> {device.lldpInfo.portId}
                    </div>
                    {device.lldpInfo.managementAddress ? (
                      <div>
                        <span className="text-text-muted">Mgmt IP:</span>{' '}
                        {device.lldpInfo.managementAddress}
                      </div>
                    ) : null}
                    {device.lldpInfo.capabilities ? (
                      <div>
                        <span className="text-text-muted">Capabilities:</span>{' '}
                        {device.lldpInfo.capabilities.join(', ')}
                      </div>
                    ) : null}
                  </div>
                </div>
              ) : null}

              {/* CDP Info */}
              {device.cdpInfo ? (
                <div>
                  <h4 className="text-xs font-semibold text-text-secondary mb-1">
                    CDP Information
                  </h4>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-2 text-xs">
                    <div>
                      <span className="text-text-muted">Device:</span> {device.cdpInfo.deviceId}
                    </div>
                    <div>
                      <span className="text-text-muted">Platform:</span> {device.cdpInfo.platform}
                    </div>
                    {device.cdpInfo.nativeVlan ? (
                      <div>
                        <span className="text-text-muted">Native VLAN:</span>{' '}
                        {device.cdpInfo.nativeVlan}
                      </div>
                    ) : null}
                  </div>
                </div>
              ) : null}

              {/* SNMP Data */}
              {device.snmpData ? (
                <div className="space-y-2">
                  <h4 className="text-xs font-semibold text-text-secondary">
                    {t('discovery.snmpInfo', 'SNMP Details')}
                  </h4>

                  {/* System Info */}
                  {device.snmpData.system ? (
                    <div className="bg-surface-base p-2 rounded-md">
                      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-x-4 gap-y-1 text-xs">
                        {device.snmpData.system.sysName ? (
                          <div>
                            <span className="text-text-muted">Name:</span>{' '}
                            <span className="text-text-primary font-medium">
                              {device.snmpData.system.sysName}
                            </span>
                          </div>
                        ) : null}
                        {device.snmpData.system.sysDescr ? (
                          <div className="col-span-2">
                            <span className="text-text-muted">Description:</span>{' '}
                            <span className="text-text-primary">
                              {device.snmpData.system.sysDescr.length > 80
                                ? `${device.snmpData.system.sysDescr.substring(0, 80)}...`
                                : device.snmpData.system.sysDescr}
                            </span>
                          </div>
                        ) : null}
                        {device.snmpData.system.sysLocation ? (
                          <div>
                            <span className="text-text-muted">Location:</span>{' '}
                            <span className="text-text-primary">
                              {device.snmpData.system.sysLocation}
                            </span>
                          </div>
                        ) : null}
                        {device.snmpData.system.sysContact ? (
                          <div>
                            <span className="text-text-muted">Contact:</span>{' '}
                            <span className="text-text-primary">
                              {device.snmpData.system.sysContact}
                            </span>
                          </div>
                        ) : null}
                        {device.snmpData.system.sysUpTime !== undefined &&
                        device.snmpData.system.sysUpTime > 0 ? (
                          <div>
                            <span className="text-text-muted">Uptime:</span>{' '}
                            <span className="text-text-primary">
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
                      <span className="text-xs text-text-muted">
                        Interfaces ({device.snmpData.interfaces.length}):
                      </span>
                      <div className="flex flex-wrap gap-1 mt-1">
                        {/* biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Complex interface status rendering */}
                        {device.snmpData.interfaces.slice(0, 8).map((iface) => (
                          <span
                            key={iface.name}
                            className={cn(
                              'px-1.5 py-0.5 text-xs',
                              radius.sm,
                              iface.operStatus === 'up'
                                ? 'bg-status-success/20 text-status-success'
                                : 'bg-surface-hover text-text-muted',
                            )}
                            title={`${iface.name} - ${iface.speedMbps ? `${iface.speedMbps} Mbps` : 'N/A'}`}
                          >
                            {iface.name}
                            {iface.speedMbps && iface.speedMbps > 0 ? (
                              <span className="text-text-muted ml-1">
                                {iface.speedMbps >= 1000
                                  ? `${Math.round(iface.speedMbps / 1000)}G`
                                  : `${iface.speedMbps}M`}
                              </span>
                            ) : null}
                          </span>
                        ))}
                        {device.snmpData.interfaces.length > 8 ? (
                          <span className="text-xs text-text-muted">
                            +{device.snmpData.interfaces.length - 8} more
                          </span>
                        ) : null}
                      </div>
                    </div>
                  ) : null}

                  {/* VLANs Summary */}
                  {device.snmpData.vlans && device.snmpData.vlans.length > 0 ? (
                    <div>
                      <span className="text-xs text-text-muted">
                        VLANs ({device.snmpData.vlans.length}):
                      </span>
                      <div className="flex flex-wrap gap-1 mt-1">
                        {device.snmpData.vlans.slice(0, 12).map((vlan) => (
                          <span
                            key={vlan.id}
                            className={cn(
                              'px-1.5 py-0.5 text-xs bg-brand-primary/10 text-brand-primary',
                              radius.sm,
                            )}
                            title={vlan.name || `VLAN ${vlan.id}`}
                          >
                            {vlan.id}
                            {vlan.name && vlan.name !== `VLAN${vlan.id}` ? (
                              <span className="text-text-muted ml-1">
                                {vlan.name.length > 10
                                  ? `${vlan.name.substring(0, 10)}...`
                                  : vlan.name}
                              </span>
                            ) : null}
                          </span>
                        ))}
                        {device.snmpData.vlans.length > 12 ? (
                          <span className="text-xs text-text-muted">
                            +{device.snmpData.vlans.length - 12} more
                          </span>
                        ) : null}
                      </div>
                    </div>
                  ) : null}

                  {/* Hardware Inventory */}
                  {device.snmpData.inventory && device.snmpData.inventory.length > 0 ? (
                    <div>
                      <span className="text-xs text-text-muted">Hardware:</span>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-1 mt-1 text-xs">
                        {device.snmpData.inventory
                          .filter(
                            (e) =>
                              e.className === 'chassis' ||
                              e.className === 'module' ||
                              e.className === 'powerSupply',
                          )
                          .slice(0, 4)
                          .map((entity) => (
                            <div
                              key={entity.serialNum || entity.name || entity.description}
                              className="bg-surface-hover px-2 py-1 rounded"
                            >
                              <span className="text-text-primary">
                                {entity.name || entity.description}
                              </span>
                              {entity.serialNum ? (
                                <span className="text-text-muted ml-2">
                                  S/N: {entity.serialNum}
                                </span>
                              ) : null}
                              {entity.modelName ? (
                                <span className="text-text-muted ml-2">
                                  Model: {entity.modelName}
                                </span>
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
                  <span className="text-xs text-text-muted">OS Guess:</span>{' '}
                  <span className="text-xs text-text-primary">{device.osGuess}</span>
                </div>
              ) : null}
            </div>
          </td>
        </tr>
      ) : null}
    </>
  );
}
