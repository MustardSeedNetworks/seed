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

import type React from "react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  button,
  cn,
  discoveryMethod as discoveryMethodTheme,
  icon as iconTokens,
  modal,
  radius,
  severity as severityTheme,
} from "../../styles/theme";
import {
  AlertTriangle,
  ArrowUpDown,
  ChevronDown,
  ChevronUp,
  Download,
  RefreshCw,
  Search,
  X,
} from "../ui/Icons";
import { Tooltip } from "../ui/Tooltip";
import type {
  DiscoveredDevice,
  DiscoveryMethod,
  NetworkDiscoveryData,
  OpenPort,
} from "./NetworkDiscoveryCard";

interface DiscoveryModalProps {
  isOpen: boolean;
  onClose: () => void;
  data: NetworkDiscoveryData | null;
  onScan?: () => void;
  onDeepScan?: (ip: string) => Promise<void>;
}

type SortField = "ip" | "hostname" | "vendor" | "mac" | "lastSeen";
type SortDirection = "asc" | "desc";

// Discovery method badge
function MethodBadge({ method }: { method: DiscoveryMethod }) {
  const theme = discoveryMethodTheme[method] || discoveryMethodTheme.arp;
  return (
    <span
      className={cn("px-1.5 py-0.5 text-xs font-medium uppercase", radius.md, theme.bg, theme.text)}
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

  if (diffSecs < 60) return "Just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString();
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
    case "ip": {
      // Sort IPs numerically
      const aParts = (a.ip || "").split(".").map(Number);
      const bParts = (b.ip || "").split(".").map(Number);
      for (let i = 0; i < 4; i++) {
        if ((aParts[i] || 0) !== (bParts[i] || 0)) {
          cmp = (aParts[i] || 0) - (bParts[i] || 0);
          break;
        }
      }
      break;
    }
    case "hostname":
      cmp = (a.hostname || "").localeCompare(b.hostname || "");
      break;
    case "vendor":
      cmp = (a.vendor || "").localeCompare(b.vendor || "");
      break;
    case "mac":
      cmp = (a.mac || "").localeCompare(b.mac || "");
      break;
    case "lastSeen":
      cmp = new Date(a.lastSeen).getTime() - new Date(b.lastSeen).getTime();
      break;
  }
  return direction === "asc" ? cmp : -cmp;
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
}) {
  const isActive = currentField === field;
  return (
    <th
      className={cn(
        "px-3 py-2 text-left text-xs font-semibold uppercase tracking-wider cursor-pointer hover:bg-surface-hover transition-colors select-none",
        className,
      )}
      onClick={() => onSort(field)}
    >
      <div className="flex items-center gap-1">
        <span>{label}</span>
        {isActive ? (
          direction === "asc" ? (
            <ChevronUp className="w-3 h-3" />
          ) : (
            <ChevronDown className="w-3 h-3" />
          )
        ) : (
          <ArrowUpDown className="w-3 h-3 opacity-30" />
        )}
      </div>
    </th>
  );
}

// Device row component
function DeviceRow({
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
}) {
  const { t } = useTranslation("cards");
  const openPorts = device.profile?.openPorts?.filter((p) => p.isOpen) || [];
  const hasDetails =
    device.lldpInfo ||
    device.cdpInfo ||
    device.edpInfo ||
    device.ndpInfo ||
    device.snmpData ||
    openPorts.length > 0;

  const handleScan = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onDeepScan && device.ip) {
      await onDeepScan(device.ip);
    }
  };

  return (
    <>
      <tr
        className={cn(
          "border-b border-surface-border hover:bg-surface-hover cursor-pointer transition-colors",
          isExpanded && "bg-surface-hover",
        )}
        onClick={onToggle}
      >
        {/* IP Address */}
        <td className="px-3 py-2">
          <div className="flex flex-col">
            <span className="font-mono text-sm font-medium text-text-primary">
              {device.ip || t("network.noIP")}
            </span>
            {device.ipv6 && (
              <span
                className="font-mono text-xs text-text-muted truncate max-w-40"
                title={device.ipv6}
              >
                {device.ipv6.length > 25 ? `${device.ipv6.substring(0, 25)}...` : device.ipv6}
              </span>
            )}
          </div>
        </td>

        {/* Hostname */}
        <td className="px-3 py-2">
          <span
            className="text-sm text-text-secondary truncate block max-w-40"
            title={device.hostname}
          >
            {device.hostname || "-"}
          </span>
        </td>

        {/* MAC Address */}
        <td className="px-3 py-2">
          <span className="font-mono text-xs text-text-muted">{device.mac || "-"}</span>
        </td>

        {/* Vendor */}
        <td className="px-3 py-2">
          {device.vendor === "LAA" ? (
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
              {device.vendor || "-"}
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
                "text-xs px-1.5 py-0.5 bg-status-success/20 text-status-success",
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
                "inline-flex items-center gap-1 text-xs px-1.5 py-0.5",
                radius.md,
                device.vulnerabilities.highestSeverity === "CRITICAL"
                  ? `${severityTheme.critical.bg} ${severityTheme.critical.text}`
                  : device.vulnerabilities.highestSeverity === "HIGH"
                    ? `${severityTheme.high.bg} ${severityTheme.high.text}`
                    : device.vulnerabilities.highestSeverity === "MEDIUM"
                      ? `${severityTheme.medium.bg} ${severityTheme.medium.text}`
                      : `${severityTheme.low.bg} ${severityTheme.low.text}`,
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
            {onDeepScan && device.ip && (
              <button
                type="button"
                type="button"
                onClick={handleScan}
                disabled={isScanning}
                className={cn(
                  "text-xs px-2 py-1 bg-brand-primary/20 text-brand-primary",
                  radius.md,
                  "hover:bg-brand-primary/30 transition-colors disabled:opacity-50",
                )}
              >
                {isScanning ? "..." : t("discovery.scan")}
              </button>
            )}
            <span className="text-xs text-text-muted">
              {hasDetails ? (isExpanded ? "▲" : "▼") : ""}
            </span>
          </div>
        </td>
      </tr>

      {/* Expanded details row */}
      {isExpanded && hasDetails && (
        <tr className="bg-surface-sunken">
          <td colSpan={9} className="px-4 py-3">
            <div className="space-y-3">
              {/* Open Ports */}
              {openPorts.length > 0 && (
                <div>
                  <h4 className="text-xs font-semibold text-text-secondary mb-1">Open Ports</h4>
                  <div className="flex flex-wrap gap-2">
                    {openPorts.map((port: OpenPort) => (
                      <span
                        key={port.port}
                        className={cn(
                          "px-2 py-1 text-xs font-mono",
                          radius.md,
                          "bg-surface-base text-text-primary",
                        )}
                      >
                        {port.port}/{port.protocol}{" "}
                        {port.service && <span className="text-text-muted">({port.service})</span>}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {/* LLDP Info */}
              {device.lldpInfo && (
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
                    {device.lldpInfo.managementAddress && (
                      <div>
                        <span className="text-text-muted">Mgmt IP:</span>{" "}
                        {device.lldpInfo.managementAddress}
                      </div>
                    )}
                    {device.lldpInfo.capabilities && (
                      <div>
                        <span className="text-text-muted">Capabilities:</span>{" "}
                        {device.lldpInfo.capabilities.join(", ")}
                      </div>
                    )}
                  </div>
                </div>
              )}

              {/* CDP Info */}
              {device.cdpInfo && (
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
                    {device.cdpInfo.nativeVlan && (
                      <div>
                        <span className="text-text-muted">Native VLAN:</span>{" "}
                        {device.cdpInfo.nativeVlan}
                      </div>
                    )}
                  </div>
                </div>
              )}

              {/* SNMP Data */}
              {device.snmpData && (
                <div className="space-y-2">
                  <h4 className="text-xs font-semibold text-text-secondary">
                    {t("discovery.snmpInfo", "SNMP Details")}
                  </h4>

                  {/* System Info */}
                  {device.snmpData.system && (
                    <div className="bg-surface-base p-2 rounded-md">
                      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-x-4 gap-y-1 text-xs">
                        {device.snmpData.system.sysName && (
                          <div>
                            <span className="text-text-muted">Name:</span>{" "}
                            <span className="text-text-primary font-medium">
                              {device.snmpData.system.sysName}
                            </span>
                          </div>
                        )}
                        {device.snmpData.system.sysDescr && (
                          <div className="col-span-2">
                            <span className="text-text-muted">Description:</span>{" "}
                            <span className="text-text-primary">
                              {device.snmpData.system.sysDescr.length > 80
                                ? `${device.snmpData.system.sysDescr.substring(0, 80)}...`
                                : device.snmpData.system.sysDescr}
                            </span>
                          </div>
                        )}
                        {device.snmpData.system.sysLocation && (
                          <div>
                            <span className="text-text-muted">Location:</span>{" "}
                            <span className="text-text-primary">
                              {device.snmpData.system.sysLocation}
                            </span>
                          </div>
                        )}
                        {device.snmpData.system.sysContact && (
                          <div>
                            <span className="text-text-muted">Contact:</span>{" "}
                            <span className="text-text-primary">
                              {device.snmpData.system.sysContact}
                            </span>
                          </div>
                        )}
                        {device.snmpData.system.sysUpTime !== undefined &&
                          device.snmpData.system.sysUpTime > 0 && (
                            <div>
                              <span className="text-text-muted">Uptime:</span>{" "}
                              <span className="text-text-primary">
                                {formatUptime(device.snmpData.system.sysUpTime)}
                              </span>
                            </div>
                          )}
                      </div>
                    </div>
                  )}

                  {/* Interfaces Summary */}
                  {device.snmpData.interfaces && device.snmpData.interfaces.length > 0 && (
                    <div>
                      <span className="text-xs text-text-muted">
                        Interfaces ({device.snmpData.interfaces.length}):
                      </span>
                      <div className="flex flex-wrap gap-1 mt-1">
                        {device.snmpData.interfaces.slice(0, 8).map((iface) => (
                          <span
                            key={iface.name}
                            className={cn(
                              "px-1.5 py-0.5 text-xs",
                              radius.sm,
                              iface.operStatus === "up"
                                ? "bg-status-success/20 text-status-success"
                                : "bg-surface-hover text-text-muted",
                            )}
                            title={`${iface.name} - ${iface.speed ? `${Math.round(iface.speed / 1000000)} Mbps` : "N/A"}`}
                          >
                            {iface.name}
                            {iface.speed && iface.speed > 0 && (
                              <span className="text-text-muted ml-1">
                                {iface.speed >= 1000000000
                                  ? `${Math.round(iface.speed / 1000000000)}G`
                                  : `${Math.round(iface.speed / 1000000)}M`}
                              </span>
                            )}
                          </span>
                        ))}
                        {device.snmpData.interfaces.length > 8 && (
                          <span className="text-xs text-text-muted">
                            +{device.snmpData.interfaces.length - 8} more
                          </span>
                        )}
                      </div>
                    </div>
                  )}

                  {/* VLANs Summary */}
                  {device.snmpData.vlans && device.snmpData.vlans.length > 0 && (
                    <div>
                      <span className="text-xs text-text-muted">
                        VLANs ({device.snmpData.vlans.length}):
                      </span>
                      <div className="flex flex-wrap gap-1 mt-1">
                        {device.snmpData.vlans.slice(0, 12).map((vlan) => (
                          <span
                            key={vlan.id}
                            className={cn(
                              "px-1.5 py-0.5 text-xs bg-brand-primary/10 text-brand-primary",
                              radius.sm,
                            )}
                            title={vlan.name || `VLAN ${vlan.id}`}
                          >
                            {vlan.id}
                            {vlan.name && vlan.name !== `VLAN${vlan.id}` && (
                              <span className="text-text-muted ml-1">
                                {vlan.name.length > 10
                                  ? `${vlan.name.substring(0, 10)}...`
                                  : vlan.name}
                              </span>
                            )}
                          </span>
                        ))}
                        {device.snmpData.vlans.length > 12 && (
                          <span className="text-xs text-text-muted">
                            +{device.snmpData.vlans.length - 12} more
                          </span>
                        )}
                      </div>
                    </div>
                  )}

                  {/* Hardware Inventory */}
                  {device.snmpData.entities && device.snmpData.entities.length > 0 && (
                    <div>
                      <span className="text-xs text-text-muted">Hardware:</span>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-1 mt-1 text-xs">
                        {device.snmpData.entities
                          .filter(
                            (e) =>
                              e.physicalClass === "chassis" ||
                              e.physicalClass === "module" ||
                              e.physicalClass === "powerSupply",
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
                              {entity.serialNum && (
                                <span className="text-text-muted ml-2">
                                  S/N: {entity.serialNum}
                                </span>
                              )}
                              {entity.modelName && (
                                <span className="text-text-muted ml-2">
                                  Model: {entity.modelName}
                                </span>
                              )}
                            </div>
                          ))}
                      </div>
                    </div>
                  )}
                </div>
              )}

              {/* OS Guess */}
              {device.osGuess && (
                <div>
                  <span className="text-xs text-text-muted">OS Guess:</span>{" "}
                  <span className="text-xs text-text-primary">{device.osGuess}</span>
                </div>
              )}
            </div>
          </td>
        </tr>
      )}
    </>
  );
}

/**
 * DiscoveryModal - Full-screen modal for device discovery viewing.
 */
export function DiscoveryModal({ isOpen, onClose, data, onScan, onDeepScan }: DiscoveryModalProps) {
  const { t } = useTranslation("cards");

  const [searchQuery, setSearchQuery] = useState("");
  const [sortField, setSortField] = useState<SortField | null>("ip");
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");
  const [expandedDevices, setExpandedDevices] = useState<Set<string>>(new Set());
  const [scanningDevices, setScanningDevices] = useState<Set<string>>(new Set());
  const [showLocalOnly, setShowLocalOnly] = useState(false);

  // Toggle sort
  const handleSort = useCallback((field: SortField) => {
    setSortField((prev) => {
      if (prev === field) {
        setSortDirection((d) => (d === "asc" ? "desc" : "asc"));
        return field;
      }
      setSortDirection("asc");
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
    async (ip: string) => {
      if (!onDeepScan) return;
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
    if (!data?.devices) return [];

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
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `devices-${new Date().toISOString().split("T")[0]}.json`;
    link.click();
    URL.revokeObjectURL(url);
  }, [filteredDevices]);

  const exportCsv = useCallback(() => {
    const escapeCsv = (val: unknown) => {
      if (val === null || val === undefined) return "";
      const str = String(val);
      if (/[",\n]/.test(str)) {
        return `"${str.replace(/"/g, '""')}"`;
      }
      return str;
    };

    const rows = filteredDevices.map((d) =>
      [
        escapeCsv(d.ip),
        escapeCsv(d.hostname),
        escapeCsv(d.mac),
        escapeCsv(d.vendor),
        escapeCsv(d.discoveryMethod.join(";")),
        escapeCsv(d.lastSeen),
        escapeCsv(d.isLocal ? "local" : "extended"),
        escapeCsv(d.osGuess),
      ].join(","),
    );

    const header = "ip,hostname,mac,vendor,discovery_methods,last_seen,network,os_guess";
    const csv = [header, ...rows].join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `devices-${new Date().toISOString().split("T")[0]}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  }, [filteredDevices]);

  // Keyboard handler for Escape
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const deviceCount = data?.devices?.length || 0;
  const localCount = data?.devices?.filter((d) => d.isLocal).length || 0;

  return (
    <div className={modal.overlay}>
      {/* Backdrop */}
      <div className={modal.backdrop} onClick={onClose} aria-hidden="true" />

      {/* Modal - full width */}
      <div
        className={cn(
          "relative",
          modal.content,
          modal.size.full,
          modal.padding.lg,
          "flex flex-col",
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby="discovery-modal-title"
      >
        {/* Header */}
        <div className="flex items-center justify-between mb-4 pb-4 border-b border-surface-border">
          <div>
            <h2 id="discovery-modal-title" className="text-xl font-semibold text-text-primary">
              {t("discovery.title", "Network Discovery")}
            </h2>
            <p className="text-sm text-text-muted mt-1">
              {t("discovery.modalSubtitle", "{{total}} devices ({{local}} local)", {
                total: deviceCount,
                local: localCount,
              })}
              {data?.status?.subnet && ` - ${data.status.subnet}`}
            </p>
          </div>

          <div className="flex items-center gap-3">
            {/* Scan button */}
            {onScan && (
              <button
                type="button"
                type="button"
                onClick={onScan}
                disabled={data?.status?.scanning}
                className={cn(
                  button.base,
                  button.variant.secondary,
                  button.size.sm,
                  "flex items-center gap-2",
                )}
              >
                <RefreshCw
                  className={cn(iconTokens.size.sm, data?.status?.scanning && "animate-spin")}
                />
                {data?.status?.scanning ? t("discovery.scanning") : t("discovery.rescan")}
              </button>
            )}

            {/* Export dropdown */}
            <div className="flex items-center gap-1">
              <button
                type="button"
                type="button"
                onClick={exportCsv}
                className={cn(
                  button.base,
                  button.variant.ghost,
                  button.size.sm,
                  "flex items-center gap-1",
                )}
                title="Export as CSV"
              >
                <Download className={iconTokens.size.sm} />
                CSV
              </button>
              <button
                type="button"
                type="button"
                onClick={exportJson}
                className={cn(
                  button.base,
                  button.variant.ghost,
                  button.size.sm,
                  "flex items-center gap-1",
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
              type="button"
              onClick={onClose}
              className={cn(
                "p-2 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors",
              )}
              aria-label="Close"
            >
              <X className={iconTokens.size.md} />
            </button>
          </div>
        </div>

        {/* Search and filters */}
        <div className="flex items-center gap-4 mb-4">
          {/* Search input */}
          <div className="relative flex-1 max-w-md">
            <Search
              className={cn(
                "absolute left-3 top-1/2 -translate-y-1/2",
                iconTokens.size.sm,
                "text-text-muted",
              )}
            />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder={t("discovery.searchPlaceholder", "Search IP, hostname, MAC, vendor...")}
              className={cn(
                "w-full pl-10 pr-4 py-2",
                "text-sm bg-surface-base border border-surface-border",
                radius.md,
                "focus:outline-none focus:ring-1 focus:ring-brand-primary text-text-primary placeholder:text-text-muted",
              )}
            />
            {searchQuery && (
              <button
                type="button"
                type="button"
                onClick={() => setSearchQuery("")}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
              >
                <X className={iconTokens.size.sm} />
              </button>
            )}
          </div>

          {/* Filter toggles */}
          <div className="flex items-center gap-2">
            <button
              type="button"
              type="button"
              onClick={() => setShowLocalOnly(!showLocalOnly)}
              className={cn(
                "px-3 py-1.5 text-sm",
                radius.md,
                "transition-colors",
                showLocalOnly
                  ? "bg-brand-primary text-text-inverse"
                  : "bg-surface-hover text-text-secondary hover:text-text-primary",
              )}
            >
              {t("discovery.localOnly", "Local Only")}
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
                  label={t("discovery.tableIp", "IP Address")}
                  field="ip"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  className="w-40"
                />
                <SortableHeader
                  label={t("discovery.tableHostname", "Hostname")}
                  field="hostname"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  className="w-40"
                />
                <SortableHeader
                  label={t("discovery.tableMac", "MAC")}
                  field="mac"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  className="w-36"
                />
                <SortableHeader
                  label={t("discovery.tableVendor", "Vendor")}
                  field="vendor"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  className="w-32"
                />
                <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wider w-28">
                  {t("discovery.tableDiscovery", "Discovery")}
                </th>
                <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wider w-20">
                  {t("discovery.tablePorts", "Ports")}
                </th>
                <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wider w-20">
                  {t("discovery.tableVulns", "CVEs")}
                </th>
                <SortableHeader
                  label={t("discovery.tableLastSeen", "Last Seen")}
                  field="lastSeen"
                  currentField={sortField}
                  direction={sortDirection}
                  onSort={handleSort}
                  className="w-24"
                />
                <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wider w-24">
                  {t("discovery.tableActions", "Actions")}
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
                    onToggle={() => toggleDevice(deviceKey)}
                    onDeepScan={onDeepScan ? handleDeepScan : undefined}
                    isScanning={scanningDevices.has(device.ip)}
                  />
                );
              })}
            </tbody>
          </table>

          {/* Empty state */}
          {filteredDevices.length === 0 && (
            <div className="text-center py-12 text-text-muted">
              {searchQuery || showLocalOnly
                ? t("discovery.noResults", "No devices match your filters")
                : t("discovery.noDevices", "No devices discovered yet")}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
