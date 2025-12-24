import { useState, memo, useCallback, useMemo, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { Card, CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { CollapsibleSection } from "../ui/CollapsibleSection";
import { Tooltip } from "../ui/Tooltip";
// Fix #669: Removed deprecated getAuthHeaders - using credentials: 'include' for cookie auth
import { logger, LogComponents } from "../../lib/logger";
import {
  ScanSearch,
  Terminal,
  Globe,
  Lock,
  FileText,
  Mail,
  Database,
  Printer,
  Router,
  Server,
  Shield,
  HardDrive,
  Container,
  Monitor,
  Smartphone,
  Wifi,
  Clock,
  CheckCircle,
  RefreshCw,
  Search,
  X,
  ChevronUp,
  ChevronDown,
  AlertTriangle,
} from "../ui/Icons";
import type { LucideIcon } from "lucide-react";
import { VulnerabilityDetailsModal } from "./VulnerabilityDetailsModal";
import {
  discoveryMethod as discoveryMethodTheme,
  category as categoryTheme,
  severity as severityTheme,
  radius,
  icon as iconTokens,
  cn,
  spacing,
  button,
} from "../../styles/theme";

export interface LLDPInfo {
  chassisId: string;
  portId: string;
  portDescription?: string;
  systemName?: string;
  systemDescription?: string;
  capabilities?: string[];
  managementAddress?: string;
}

export interface CDPInfo {
  deviceId: string;
  portId: string;
  platform?: string;
  softwareVersion?: string;
  capabilities?: string[];
  managementAddress?: string;
  nativeVlan?: number;
  voiceVlan?: number;
}

export interface EDPInfo {
  deviceId: string;
  displayName?: string;
  portId: string;
  platform?: string;
  softwareVersion?: string;
  vlan?: number;
}

export interface NDPInfo {
  linkLayerAddress: string;
  isRouter: boolean;
  reachableTime?: number;
  retransTimer?: number;
  flags?: number;
  lastAdvertisement?: string;
}

export type DiscoveryMethod =
  | "arp"
  | "ndp"
  | "lldp"
  | "cdp"
  | "edp"
  | "mdns"
  | "ping";

// Auto-profiling types from backend
export interface OpenPort {
  port: number;
  protocol: string;
  service?: string;
  banner?: string;
  isOpen: boolean;
}

export interface HTTPInfo {
  port: number;
  statusCode: number;
  title?: string;
  server?: string;
  isHttps: boolean;
}

export interface DeviceProfile {
  profiledAt: string;
  openPorts?: OpenPort[];
  httpInfo?: HTTPInfo;
  deviceType?: string;
  deviceIcons?: string[];
}

export interface DiscoveredDevice {
  ip: string;
  ipv6?: string;
  ipv6Addresses?: string[];
  mac: string;
  hostname?: string;
  vendor?: string;
  osGuess?: string;
  ttl?: number;
  discoveryMethod: DiscoveryMethod[];
  lastSeen: string;
  isLocal: boolean; // true if on local subnet, false for extended networks
  isRouter?: boolean;
  lldpInfo?: LLDPInfo;
  cdpInfo?: CDPInfo;
  edpInfo?: EDPInfo;
  ndpInfo?: NDPInfo;
  profile?: DeviceProfile;
  vulnerabilities?: {
    count: number;
    highestSeverity: "CRITICAL" | "HIGH" | "MEDIUM" | "LOW";
  };
}

export interface DiscoveryStatus {
  scanning: boolean;
  deviceCount: number;
  lastScan: string;
  subnet: string;
  localIP: string;
  interface: string;
}

export interface NetworkDiscoveryData {
  devices: DiscoveredDevice[];
  status: DiscoveryStatus;
}

// Deep Scan (Port Scan) Types - matches backend discovery.ServiceInfo
export interface ServiceInfo {
  port: number;
  state: "open" | "closed" | "filtered";
  service: string;
  banner?: string;
  version?: string;
  protocol?: string;
}

// PortScanResult for display - normalized from backend response
export interface PortScanResult {
  port: number;
  state: "open" | "closed" | "filtered";
  service: string;
  banner?: string;
  version?: string;
  rtt: number; // nanoseconds (0 if not available from backend)
}

// Backend API response structure
interface PortScanAPIResponse {
  ip: string;
  hostname?: string;
  services: ServiceInfo[];
  scanTime: number;
  error?: string;
}

export interface DeepScanResult {
  target: string;
  results: PortScanResult[];
  osGuess?: string;
  scannedAt: Date;
}

interface DiscoverySettingsForAutoScan {
  portScanEnabled?: boolean;
  vulnScanEnabled?: boolean;
  vulnAutoScan?: boolean;
}

interface NetworkDiscoveryCardProps {
  data: NetworkDiscoveryData | null;
  loading?: boolean;
  onScan?: () => void;
}

// Sorting types
type SortField = "ip" | "hostname" | "vendor" | "lastSeen" | null;
type SortDirection = "asc" | "desc";

// Search bar component
function DeviceSearchBar({
  searchQuery,
  onSearchChange,
  sortField,
  sortDirection,
  onSortChange,
  deviceCount,
  filteredCount,
  t,
}: {
  searchQuery: string;
  onSearchChange: (query: string) => void;
  sortField: SortField;
  sortDirection: SortDirection;
  onSortChange: (field: SortField) => void;
  deviceCount: number;
  filteredCount: number;
  t: ReturnType<typeof useTranslation<"cards">>["t"];
}) {
  return (
    <div className="stack-sm">
      {/* Search input */}
      <div className="relative">
        <Search
          className={cn(
            "absolute left-2.5 top-1/2 -translate-y-1/2",
            iconTokens.size.sm,
            "text-text-muted pointer-events-none"
          )}
        />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder={t("discovery.searchPlaceholder")}
          className={cn(
            "w-full pl-9 pr-8",
            spacing.chip.sm,
            "body-small bg-surface-base border border-surface-border",
            radius.md,
            "focus:outline-none focus:ring-1 focus:ring-brand-primary text-text-primary placeholder:text-text-muted"
          )}
        />
        {searchQuery && (
          <button
            type="button"
            onClick={() => onSearchChange("")}
            className="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
            aria-label={t("discovery.clearSearch")}
          >
            <X className={iconTokens.size.sm} />
          </button>
        )}
      </div>

      {/* Sort buttons row */}
      <div
        className={cn(
          "flex items-center justify-between",
          spacing.gap.compact,
          "flex-wrap"
        )}
      >
        <div className={cn("flex items-center", spacing.gap.tight)}>
          <span className="caption text-text-muted">
            {t("discovery.sort")}:
          </span>
          {(["ip", "hostname", "vendor", "lastSeen"] as SortField[]).map(
            (field) => (
              <button
                key={field}
                type="button"
                onClick={() => onSortChange(field)}
                className={cn(
                  spacing.chip.sm,
                  "caption",
                  radius.md,
                  "transition-colors flex items-center",
                  spacing.gap.tight,
                  sortField === field
                    ? "bg-brand-primary/20 text-brand-primary"
                    : "bg-surface-hover text-text-muted hover:text-text-primary"
                )}
              >
                {field === "ip"
                  ? t("discovery.sortIp")
                  : field === "hostname"
                    ? t("discovery.sortName")
                    : field === "vendor"
                      ? t("discovery.sortVendor")
                      : t("discovery.sortSeen")}
                {sortField === field &&
                  (sortDirection === "asc" ? (
                    <ChevronUp className={iconTokens.size.xs} />
                  ) : (
                    <ChevronDown className={iconTokens.size.xs} />
                  ))}
              </button>
            )
          )}
        </div>
        {searchQuery && (
          <span className="caption text-text-muted">
            {t("discovery.filteredCount", {
              filtered: filteredCount,
              total: deviceCount,
            })}
          </span>
        )}
      </div>
    </div>
  );
}

function formatLastSeen(
  dateStr: string,
  t: ReturnType<typeof useTranslation<"cards">>["t"]
): string {
  if (!dateStr) return t("discovery.never");
  const date = new Date(dateStr);
  // Check for invalid date or Go's zero time (year 1 or epoch)
  if (isNaN(date.getTime()) || date.getFullYear() < 2000)
    return t("discovery.never");
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 0) return t("discovery.never"); // Future date = invalid
  if (diffSec < 60) return t("discovery.justNow");
  if (diffSec < 3600)
    return t("discovery.mAgo", { min: Math.floor(diffSec / 60) });
  if (diffSec < 86400)
    return t("discovery.hAgo", { hour: Math.floor(diffSec / 3600) });
  return t("discovery.dAgo", { day: Math.floor(diffSec / 86400) });
}

/**
 * Convert host IP/CIDR to network address (fixes #738)
 * e.g., "192.168.64.7/24" -> "192.168.64.0/24"
 */
function calculateNetworkAddress(cidr: string): string {
  const [ip, maskStr] = cidr.split("/");
  if (!ip || !maskStr) return cidr;

  const mask = parseInt(maskStr, 10);
  if (isNaN(mask) || mask < 0 || mask > 32) return cidr;

  const octets = ip.split(".").map(Number);
  if (octets.length !== 4 || octets.some(isNaN)) return cidr;

  // Calculate network mask and apply to IP
  const netmask = (0xffffffff << (32 - mask)) >>> 0;
  const ipInt =
    ((octets[0] << 24) | (octets[1] << 16) | (octets[2] << 8) | octets[3]) >>>
    0;
  const networkInt = (ipInt & netmask) >>> 0;

  const networkOctets = [
    (networkInt >>> 24) & 0xff,
    (networkInt >>> 16) & 0xff,
    (networkInt >>> 8) & 0xff,
    networkInt & 0xff,
  ];

  return `${networkOctets.join(".")}/${mask}`;
}

// Discovery method colors - from theme tokens (dark mode aware)
// These use colored backgrounds for visual distinction between methods

/** Type-safe getter for discovery method colors */
function getMethodColor(method: DiscoveryMethod): string {
  switch (method) {
    case "arp":
      return discoveryMethodTheme.arp;
    case "ndp":
      return discoveryMethodTheme.ndp;
    case "lldp":
      return discoveryMethodTheme.lldp;
    case "cdp":
      return discoveryMethodTheme.cdp;
    case "edp":
      return discoveryMethodTheme.edp;
    case "mdns":
      return discoveryMethodTheme.mdns;
    case "ping":
      return discoveryMethodTheme.icmp;
  }
}

function MethodBadge({ method }: { method: DiscoveryMethod }) {
  return (
    <span
      className={cn(
        spacing.chip.sm,
        radius.md,
        "caption font-medium uppercase",
        getMethodColor(method)
      )}
    >
      {method}
    </span>
  );
}

// Icon mapping for device profile icons - maps service names to Lucide icons
// Using Map for type-safe dynamic key lookups
const SERVICE_ICONS_MAP = new Map<string, LucideIcon>([
  ["ssh", Terminal],
  ["telnet", Terminal],
  ["web", Globe],
  ["web-secure", Lock],
  ["ftp", FileText],
  ["mail", Mail],
  ["dns", Globe],
  ["snmp", Server],
  ["database", Database],
  ["cache", Container],
  ["printer", Printer],
  ["router", Router],
  ["switch", Server],
  ["firewall", Shield],
  ["storage", HardDrive],
  ["server", Server],
]);

function ProfileIcons({
  icons,
  deviceType,
}: {
  icons?: string[];
  deviceType?: string;
}) {
  if (!icons || icons.length === 0) return null;

  return (
    <div className={cn("flex items-center", spacing.micro.gap, "flex-wrap")}>
      {icons.slice(0, 5).map((icon) => {
        const IconComponent = SERVICE_ICONS_MAP.get(icon);
        return (
          <span
            key={icon}
            className={cn(
              spacing.pad.xs,
              radius.md,
              "bg-brand-primary/20 text-brand-primary flex items-center justify-center"
            )}
            title={`${icon}${deviceType ? ` (${deviceType})` : ""}`}
          >
            {IconComponent ? (
              <IconComponent className={iconTokens.size.xs} />
            ) : (
              <span className="text-[10px] font-medium">
                {icon[0]?.toUpperCase()}
              </span>
            )}
          </span>
        );
      })}
      {icons.length > 5 && (
        <span className="text-[10px] text-text-muted">+{icons.length - 5}</span>
      )}
    </div>
  );
}

// Device type categorization based on profile icons and device type
function categorizeDevices(devices: DiscoveredDevice[]) {
  const categories = {
    routers: 0,
    servers: 0,
    workstations: 0,
    printers: 0,
    mobile: 0,
    network: 0, // switches, APs
    other: 0,
  };

  devices.forEach((device) => {
    const deviceType = device.profile?.deviceType?.toLowerCase() || "";
    const icons = device.profile?.deviceIcons || [];

    if (
      icons.includes("router") ||
      deviceType.includes("router") ||
      device.cdpInfo?.capabilities?.some((c) =>
        c.toLowerCase().includes("router")
      ) ||
      device.lldpInfo?.capabilities?.some((c) =>
        c.toLowerCase().includes("router")
      )
    ) {
      categories.routers++;
    } else if (
      icons.includes("switch") ||
      deviceType.includes("switch") ||
      device.cdpInfo?.capabilities?.some((c) =>
        c.toLowerCase().includes("switch")
      ) ||
      device.lldpInfo?.capabilities?.some((c) =>
        c.toLowerCase().includes("bridge")
      )
    ) {
      categories.network++;
    } else if (icons.includes("printer") || deviceType.includes("printer")) {
      categories.printers++;
    } else if (
      icons.includes("server") ||
      deviceType.includes("server") ||
      icons.includes("database") ||
      icons.includes("dns") ||
      icons.includes("mail")
    ) {
      categories.servers++;
    } else if (
      deviceType.includes("phone") ||
      deviceType.includes("mobile") ||
      device.vendor?.toLowerCase().includes("apple") ||
      device.vendor?.toLowerCase().includes("samsung")
    ) {
      categories.mobile++;
    } else if (
      device.osGuess?.toLowerCase().includes("windows") ||
      device.osGuess?.toLowerCase().includes("linux") ||
      device.osGuess?.toLowerCase().includes("macos")
    ) {
      categories.workstations++;
    } else {
      categories.other++;
    }
  });

  return categories;
}

// Summary bar component
function DiscoverySummary({
  status,
  deviceCount,
  categories,
  t,
}: {
  status: DiscoveryStatus;
  deviceCount: number;
  categories: ReturnType<typeof categorizeDevices>;
  t: ReturnType<typeof useTranslation<"cards">>["t"];
}) {
  // Build stat items with non-zero counts
  // Using theme tokens for device category colors (dark mode aware)
  const stats = [
    {
      icon: Router,
      label: t("discovery.routers"),
      count: categories.routers,
      color: categoryTheme.router,
    },
    {
      icon: Server,
      label: t("discovery.servers"),
      count: categories.servers,
      color: categoryTheme.server,
    },
    {
      icon: Monitor,
      label: t("discovery.workstations"),
      count: categories.workstations,
      color: categoryTheme.workstation,
    },
    {
      icon: Printer,
      label: t("discovery.printers"),
      count: categories.printers,
      color: categoryTheme.printer,
    },
    {
      icon: Smartphone,
      label: t("discovery.mobile"),
      count: categories.mobile,
      color: categoryTheme.mobile,
    },
    {
      icon: Wifi,
      label: t("discovery.networkDevices"),
      count: categories.network,
      color: categoryTheme.network,
    },
  ].filter((s) => s.count > 0);

  return (
    <div
      className={cn("bg-surface-hover", radius.md, spacing.pad.sm, "stack-sm")}
    >
      {/* Status row */}
      <div className="flex items-center justify-between body-small">
        <div className={cn("flex items-center", spacing.gap.compact)}>
          {status.scanning ? (
            <>
              <RefreshCw
                className={cn(
                  iconTokens.size.sm,
                  "text-status-info animate-spin"
                )}
              />
              <span className="text-status-info font-medium">
                {t("discovery.scanning")}
              </span>
            </>
          ) : (
            <>
              <CheckCircle
                className={cn(iconTokens.size.sm, "text-status-success")}
              />
              <span className="text-status-success font-medium">
                {t("discovery.complete")}
              </span>
            </>
          )}
        </div>
        <div
          className={cn(
            "flex items-center",
            spacing.inline.sm,
            "text-text-muted"
          )}
        >
          <Clock className={iconTokens.size.sm} />
          <span className="caption">{formatLastSeen(status.lastScan, t)}</span>
        </div>
      </div>

      {/* Network info row - fixes #738: show network address, not host IP */}
      <div className="flex items-center justify-between caption text-text-muted">
        <span className="font-mono">
          {status.subnet
            ? calculateNetworkAddress(status.subnet)
            : t("discovery.unknownSubnet")}
        </span>
        <span>
          {deviceCount === 1
            ? t("discovery.deviceFound", { count: deviceCount })
            : t("discovery.devicesFound", { count: deviceCount })}
        </span>
      </div>

      {(status.localIP || status.interface) && (
        <div className="flex items-center justify-between caption text-text-muted mt-1">
          <span className="font-mono">
            {status.localIP
              ? t("discovery.localAddress", { ip: status.localIP })
              : t("discovery.localIp")}
          </span>
          <span className="text-right truncate">
            {t("discovery.listeningInterface", {
              iface: status.interface || t("discovery.unknownInterface"),
            })}
          </span>
        </div>
      )}

      {/* Category stats row */}
      {stats.length > 0 && (
        <div
          className={cn(
            "flex items-center",
            spacing.gap.default,
            "flex-wrap",
            spacing.padding.top.heading
          )}
        >
          {stats.map(({ icon: Icon, label, count, color }) => (
            <div
              key={label}
              className={cn("flex items-center", spacing.gap.tight)}
              title={`${count} ${label}`}
            >
              <Icon className={cn(iconTokens.size.sm, color)} />
              <span className="caption text-text-secondary">{count}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// Common port to service name mapping - using Map for type-safe lookups
const PORT_SERVICES_MAP = new Map<number, string>([
  [21, "FTP"],
  [22, "SSH"],
  [23, "Telnet"],
  [25, "SMTP"],
  [53, "DNS"],
  [80, "HTTP"],
  [110, "POP3"],
  [143, "IMAP"],
  [443, "HTTPS"],
  [445, "SMB"],
  [993, "IMAPS"],
  [995, "POP3S"],
  [3306, "MySQL"],
  [3389, "RDP"],
  [5432, "PostgreSQL"],
  [5900, "VNC"],
  [6379, "Redis"],
  [8080, "HTTP-Alt"],
  [8443, "HTTPS-Alt"],
  [27017, "MongoDB"],
]);

function getServiceName(port: number): string {
  return PORT_SERVICES_MAP.get(port) || `Port ${port}`;
}

function DeviceRow({
  device,
  isExpanded,
  onToggle,
  onDeepScan,
  isScanning,
  scanResult,
  onVulnerabilityClick,
  t,
}: {
  device: DiscoveredDevice;
  isExpanded: boolean;
  onToggle: () => void;
  onDeepScan?: (ip: string) => void;
  isScanning?: boolean;
  scanResult?: DeepScanResult;
  onVulnerabilityClick?: (ip: string) => void;
  t: ReturnType<typeof useTranslation<"cards">>["t"];
}) {
  const hasDetails =
    device.lldpInfo || device.cdpInfo || device.edpInfo || device.profile;
  const openPorts =
    scanResult?.results?.filter((r) => r.state === "open") || [];
  const profileOpenPorts =
    device.profile?.openPorts?.filter((p) => p.isOpen) || [];

  const handleDeepScan = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onDeepScan && device.ip) {
      onDeepScan(device.ip);
    }
  };

  return (
    <div
      className={cn(
        "border border-surface-border",
        radius.md,
        "overflow-hidden"
      )}
    >
      <button
        type="button"
        onClick={onToggle}
        className={cn(
          "w-full",
          spacing.pad.xs,
          `sm:${spacing.pad.sm}`,
          "text-left hover:bg-surface-hover transition-colors focus:outline-none focus:ring-1 focus:ring-brand-primary"
        )}
      >
        <div
          className={cn(
            "flex items-center justify-between",
            spacing.gap.compact
          )}
        >
          <div className="flex-1 min-w-0">
            <div
              className={cn(
                "flex items-center",
                spacing.gap.compact,
                "flex-wrap"
              )}
            >
              <span className="font-mono body-small text-text-primary">
                {device.ip || t("network.noIP")}
              </span>
              {device.ipv6 && (
                <span
                  className="font-mono caption text-text-accent"
                  title={device.ipv6}
                >
                  {device.ipv6.length > 20
                    ? device.ipv6.substring(0, 20) + "..."
                    : device.ipv6}
                </span>
              )}
              {device.hostname && (
                <span
                  className="caption text-text-muted truncate max-w-30"
                  title={device.hostname}
                >
                  ({device.hostname})
                </span>
              )}
              {openPorts.length > 0 && (
                <span
                  className={cn(
                    "caption bg-status-success/20 text-status-success",
                    spacing.chip.sm,
                    radius.md
                  )}
                >
                  {t("discovery.open", { count: openPorts.length })}
                </span>
              )}
              {device.profile?.deviceIcons &&
                device.profile.deviceIcons.length > 0 && (
                  <ProfileIcons
                    icons={device.profile.deviceIcons}
                    deviceType={device.profile.deviceType}
                  />
                )}
              {device.vulnerabilities && device.vulnerabilities.count > 0 && (
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    onVulnerabilityClick?.(device.ip);
                  }}
                  className={cn(
                    "inline-flex items-center",
                    spacing.gap.tight,
                    "caption",
                    spacing.chip.sm,
                    radius.md,
                    "cursor-pointer hover:opacity-80 transition-opacity",
                    device.vulnerabilities.highestSeverity === "CRITICAL"
                      ? `${severityTheme.critical.bg} ${severityTheme.critical.text}`
                      : device.vulnerabilities.highestSeverity === "HIGH"
                        ? `${severityTheme.high.bg} ${severityTheme.high.text}`
                        : device.vulnerabilities.highestSeverity === "MEDIUM"
                          ? `${severityTheme.medium.bg} ${severityTheme.medium.text}`
                          : `${severityTheme.low.bg} ${severityTheme.low.text}`
                  )}
                  title={t("discovery.clickViewVulnerabilities")}
                >
                  <AlertTriangle className={iconTokens.size.xs} />
                  {device.vulnerabilities.count} CVE
                </button>
              )}
            </div>
            <div
              className={cn(
                "flex items-center",
                spacing.inline.sm,
                spacing.margin.top.inline,
                "flex-wrap"
              )}
            >
              {device.discoveryMethod.map((method) => (
                <MethodBadge key={method} method={method} />
              ))}
              {device.vendor && device.vendor !== "Unknown" && (
                device.vendor === "LAA" ? (
                  <Tooltip
                    content="Locally Administered Address - A MAC address that was locally assigned rather than by the manufacturer. Common in virtual machines, containers, and devices with MAC randomization enabled for privacy."
                    position="bottom"
                  >
                    <span className="caption text-text-muted truncate max-w-25 underline decoration-dotted cursor-help">
                      {device.vendor}
                    </span>
                  </Tooltip>
                ) : (
                  <span
                    className="caption text-text-muted truncate max-w-25"
                    title={device.vendor}
                  >
                    {device.vendor}
                  </span>
                )
              )}
            </div>
          </div>
          <div
            className={cn("flex items-center", spacing.gap.compact, "shrink-0")}
          >
            {device.osGuess && (
              <span className="caption text-text-muted hidden sm:inline">
                {device.osGuess}
              </span>
            )}
            {onDeepScan && device.ip && (
              <button
                type="button"
                onClick={handleDeepScan}
                disabled={isScanning}
                className={cn(
                  spacing.chip.sm,
                  "caption bg-status-info/20 text-status-info",
                  radius.md,
                  "hover:bg-status-info/30 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                )}
                title={t("discovery.deepScan")}
              >
                {isScanning ? (
                  <span className={cn("flex items-center", spacing.gap.tight)}>
                    <span className="animate-spin">◐</span>
                  </span>
                ) : (
                  t("discovery.scan")
                )}
              </button>
            )}
            <span
              className={cn(
                "text-lg transition-transform",
                isExpanded ? "rotate-180" : ""
              )}
            >
              {hasDetails || scanResult ? "▼" : "○"}
            </span>
          </div>
        </div>
      </button>

      {isExpanded && (
        <div
          className={cn(
            spacing.pad.xs,
            `sm:${spacing.pad.sm}`,
            spacing.padding.top.heading,
            "border-t border-surface-border bg-surface-base"
          )}
        >
          <div className="stack-xs caption">
            <CardRow label={t("discovery.mac")} value={device.mac} />
            {device.ipv6 && (
              <CardRow label={t("discovery.ipv6")} value={device.ipv6} />
            )}
            {device.ipv6Addresses && device.ipv6Addresses.length > 1 && (
              <CardRow
                label={t("discovery.allIpv6")}
                value={device.ipv6Addresses.join(", ")}
              />
            )}
            {device.isRouter && (
              <CardRow
                label={t("discovery.router")}
                value={t("discovery.yesNdp")}
              />
            )}
            {device.vendor && (
              <div
                className={cn(
                  "flex justify-between",
                  spacing.compact.py,
                  "items-center"
                )}
              >
                <span className="body-small shrink-0">
                  {t("discovery.vendor")}
                </span>
                {device.vendor === "LAA" ? (
                  <Tooltip
                    content="Locally Administered Address - A MAC address that was locally assigned rather than by the manufacturer. Common in virtual machines, containers, and devices with MAC randomization enabled for privacy."
                    position="top"
                  >
                    <span className="body-small font-medium text-text-secondary text-right underline decoration-dotted cursor-help">
                      {device.vendor}
                    </span>
                  </Tooltip>
                ) : (
                  <span className="body-small font-medium text-text-secondary text-right">
                    {device.vendor}
                  </span>
                )}
              </div>
            )}
            {device.osGuess && (
              <CardRow label={t("discovery.osGuess")} value={device.osGuess} />
            )}
            {device.ttl && (
              <CardRow label={t("discovery.ttl")} value={device.ttl} />
            )}
            <CardRow
              label={t("discovery.lastSeen")}
              value={formatLastSeen(device.lastSeen, t)}
            />

            {/* Deep Scan Results */}
            {scanResult && (
              <>
                <CardDivider />
                <p
                  className={cn(
                    "font-medium text-text-primary",
                    spacing.margin.bottom.inline
                  )}
                >
                  {t("discovery.portScanResults")}
                </p>
                {openPorts.length > 0 ? (
                  <div className="stack-xs">
                    {openPorts.map((result) => (
                      <div
                        key={result.port}
                        className={cn(
                          "flex flex-col",
                          spacing.pad.xs,
                          "bg-surface-hover",
                          radius.sm
                        )}
                      >
                        <div className="flex items-center justify-between">
                          <span className="text-status-success font-mono body-small">
                            {result.port}/
                            {result.service && result.service !== "unknown"
                              ? result.service
                              : getServiceName(result.port)}
                          </span>
                          {result.version && (
                            <span className="caption text-text-muted">
                              {result.version}
                            </span>
                          )}
                        </div>
                        {result.banner && (
                          <span
                            className="caption text-text-secondary font-mono mt-0.5 break-all"
                            title={result.banner}
                          >
                            {result.banner.length > 80
                              ? result.banner.substring(0, 80) + "..."
                              : result.banner}
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-text-muted">
                    {t("discovery.noOpenPorts")}
                  </p>
                )}
              </>
            )}

            {/* Auto-Profile Results */}
            {device.profile && (
              <>
                <CardDivider />
                <p
                  className={cn(
                    "font-medium text-text-primary",
                    spacing.margin.bottom.inline
                  )}
                >
                  {t("discovery.autoProfile")}
                  {device.profile.deviceType &&
                    device.profile.deviceType !== "unknown" && (
                      <span
                        className={cn(
                          spacing.margin.left.inline,
                          "caption font-normal text-text-muted"
                        )}
                      >
                        ({device.profile.deviceType})
                      </span>
                    )}
                </p>
                {device.profile.httpInfo && (
                  <div
                    className={cn("gap-y-0.5", spacing.margin.bottom.inline)}
                  >
                    <CardRow
                      label={device.profile.httpInfo.isHttps ? "HTTPS" : "HTTP"}
                      value={`Port ${device.profile.httpInfo.port} (${device.profile.httpInfo.statusCode})`}
                    />
                    {device.profile.httpInfo.title && (
                      <CardRow
                        label={t("discovery.title2")}
                        value={device.profile.httpInfo.title}
                      />
                    )}
                    {device.profile.httpInfo.server && (
                      <CardRow
                        label={t("discovery.httpServer")}
                        value={device.profile.httpInfo.server}
                      />
                    )}
                  </div>
                )}
                {profileOpenPorts.length > 0 && (
                  <div className="gap-y-0.5">
                    <p
                      className={cn(
                        "text-text-muted text-[10px] uppercase tracking-wide",
                        spacing.margin.bottom.inline
                      )}
                    >
                      {t("discovery.openPorts")}
                    </p>
                    <div className={cn("flex flex-wrap", spacing.gap.tight)}>
                      {profileOpenPorts.map((port) => (
                        <span
                          key={port.port}
                          className={cn(
                            spacing.chip.sm,
                            radius.md,
                            "text-[10px] bg-status-success/20 text-status-success"
                          )}
                          title={port.banner || port.service || undefined}
                        >
                          {port.port}
                          {port.service && `/${port.service}`}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </>
            )}

            {device.lldpInfo && (
              <>
                <CardDivider />
                <p
                  className={cn(
                    "font-medium text-text-primary",
                    spacing.margin.bottom.inline
                  )}
                >
                  {t("discovery.lldpInfo")}
                </p>
                <CardRow
                  label={t("discovery.chassisId")}
                  value={device.lldpInfo.chassisId}
                />
                <CardRow
                  label={t("discovery.portId")}
                  value={device.lldpInfo.portId}
                />
                {device.lldpInfo.systemName && (
                  <CardRow
                    label={t("discovery.systemName")}
                    value={device.lldpInfo.systemName}
                  />
                )}
                {device.lldpInfo.portDescription && (
                  <CardRow
                    label={t("discovery.portDesc")}
                    value={device.lldpInfo.portDescription}
                  />
                )}
                {device.lldpInfo.systemDescription && (
                  <CardRow
                    label={t("discovery.systemDesc")}
                    value={device.lldpInfo.systemDescription}
                  />
                )}
                {device.lldpInfo.capabilities &&
                  device.lldpInfo.capabilities.length > 0 && (
                    <CardRow
                      label={t("discovery.capabilities")}
                      value={device.lldpInfo.capabilities.join(", ")}
                    />
                  )}
                {device.lldpInfo.managementAddress && (
                  <CardRow
                    label={t("discovery.mgmtIp")}
                    value={device.lldpInfo.managementAddress}
                  />
                )}
              </>
            )}

            {device.cdpInfo && (
              <>
                <CardDivider />
                <p
                  className={cn(
                    "font-medium text-text-primary",
                    spacing.margin.bottom.inline
                  )}
                >
                  {t("discovery.cdpInfo")}
                </p>
                <CardRow
                  label={t("discovery.deviceId")}
                  value={device.cdpInfo.deviceId}
                />
                <CardRow
                  label={t("discovery.portId")}
                  value={device.cdpInfo.portId}
                />
                {device.cdpInfo.platform && (
                  <CardRow
                    label={t("discovery.platform")}
                    value={device.cdpInfo.platform}
                  />
                )}
                {device.cdpInfo.softwareVersion && (
                  <CardRow
                    label={t("discovery.software")}
                    value={device.cdpInfo.softwareVersion}
                  />
                )}
                {device.cdpInfo.capabilities &&
                  device.cdpInfo.capabilities.length > 0 && (
                    <CardRow
                      label={t("discovery.capabilities")}
                      value={device.cdpInfo.capabilities.join(", ")}
                    />
                  )}
                {device.cdpInfo.nativeVlan && (
                  <CardRow
                    label={t("discovery.nativeVlan")}
                    value={device.cdpInfo.nativeVlan}
                  />
                )}
                {device.cdpInfo.managementAddress && (
                  <CardRow
                    label={t("discovery.mgmtIp")}
                    value={device.cdpInfo.managementAddress}
                  />
                )}
              </>
            )}

            {device.edpInfo && (
              <>
                <CardDivider />
                <p
                  className={cn(
                    "font-medium text-text-primary",
                    spacing.margin.bottom.inline
                  )}
                >
                  {t("discovery.edpInfo")}
                </p>
                <CardRow
                  label={t("discovery.deviceId")}
                  value={device.edpInfo.deviceId}
                />
                {device.edpInfo.displayName && (
                  <CardRow
                    label={t("discovery.displayName")}
                    value={device.edpInfo.displayName}
                  />
                )}
                <CardRow
                  label={t("discovery.portId")}
                  value={device.edpInfo.portId}
                />
                {device.edpInfo.platform && (
                  <CardRow
                    label={t("discovery.platform")}
                    value={device.edpInfo.platform}
                  />
                )}
                {device.edpInfo.softwareVersion && (
                  <CardRow
                    label={t("discovery.software")}
                    value={device.edpInfo.softwareVersion}
                  />
                )}
                {device.edpInfo.vlan && (
                  <CardRow
                    label={t("discovery.vlan")}
                    value={device.edpInfo.vlan}
                  />
                )}
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// Common ports to scan for Deep Scan
const COMMON_PORTS = [
  21, 22, 23, 25, 53, 80, 110, 143, 443, 445, 993, 995, 3306, 3389, 5432, 5900,
  6379, 8080, 8443, 27017,
];

export const NetworkDiscoveryCard = memo(function NetworkDiscoveryCard({
  data,
  loading,
  onScan,
}: NetworkDiscoveryCardProps) {
  const { t } = useTranslation("cards");
  const [expandedDevices, setExpandedDevices] = useState<Set<string>>(
    new Set()
  );
  const [scanningDevices, setScanningDevices] = useState<Set<string>>(
    new Set()
  );
  const [scanResults, setScanResults] = useState<Map<string, DeepScanResult>>(
    new Map()
  );
  // Search and sort state
  const [searchQuery, setSearchQuery] = useState("");
  const [sortField, setSortField] = useState<SortField>(null);
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");

  // Settings for auto-scan behavior - fetched from API
  const [autoScanSettings, setAutoScanSettings] = useState<DiscoverySettingsForAutoScan>({
    portScanEnabled: false,
    vulnScanEnabled: false,
    vulnAutoScan: false,
  });

  // Vulnerability modal state
  const [selectedDeviceForVuln, setSelectedDeviceForVuln] = useState<
    string | null
  >(null);

  // Fetch settings for auto-scan behavior on mount
  useEffect(() => {
    const fetchSettings = async () => {
      const apiBase = import.meta.env.VITE_API_BASE || "";
      try {
        // Fetch discovery options from correct endpoint
        const discoveryResponse = await fetch(`${apiBase}/api/discovery/options`, {
          credentials: "include",
        });
        if (discoveryResponse.ok) {
          const discoveryData = await discoveryResponse.json();
          // Backend returns { options: { PortScan: { Enabled: true, ... } } }
          const portScanEnabled = discoveryData?.options?.PortScan?.Enabled ?? false;

          // Fetch vulnerability settings from correct endpoint
          const vulnResponse = await fetch(`${apiBase}/api/vulnerabilities/settings`, {
            credentials: "include",
          });
          let vulnEnabled = false;
          let vulnAutoScan = false;
          if (vulnResponse.ok) {
            const vulnData = await vulnResponse.json();
            // Backend returns { Enabled: false, AutoScan: false, ... }
            vulnEnabled = vulnData?.Enabled ?? false;
            vulnAutoScan = vulnData?.AutoScan ?? false;
          }

          setAutoScanSettings({
            portScanEnabled,
            vulnScanEnabled: vulnEnabled,
            vulnAutoScan,
          });
        }
      } catch (error) {
        logger.debug(LogComponents.DISCOVERY, "Failed to fetch auto-scan settings", error);
      }
    };

    fetchSettings();
  }, []);

  // Toggle sort field/direction
  const handleSortChange = useCallback(
    (field: SortField) => {
      if (sortField === field) {
        // Toggle direction or clear
        if (sortDirection === "asc") {
          setSortDirection("desc");
        } else {
          setSortField(null);
          setSortDirection("asc");
        }
      } else {
        setSortField(field);
        setSortDirection("asc");
      }
    },
    [sortField, sortDirection]
  );

  const toggleDevice = (mac: string) => {
    setExpandedDevices((prev) => {
      const next = new Set(prev);
      if (next.has(mac)) {
        next.delete(mac);
      } else {
        next.add(mac);
      }
      return next;
    });
  };

  // Trigger vulnerability scan for a device based on any good info we have
  const triggerVulnScan = useCallback(async (ip: string, device?: DiscoveredDevice, services?: ServiceInfo[]) => {
    if (!autoScanSettings.vulnScanEnabled || !autoScanSettings.vulnAutoScan) {
      return;
    }

    // Check if we have any good info to run vuln scan against:
    // - Port scan results with services/banners/versions
    // - Device OS guess
    // - SNMP info (system description, software version)
    // - LLDP/CDP info (capabilities, software version)
    // - Device profile with open ports

    let hasGoodInfo = false;
    const reasons: string[] = [];

    // Check port scan services
    if (services && services.length > 0) {
      const openServices = services.filter(
        (s) => s.state === "open" && (s.banner || s.version || s.service !== "unknown")
      );
      if (openServices.length > 0) {
        hasGoodInfo = true;
        reasons.push(`${openServices.length} services`);
      }
    }

    // Check device info if provided
    if (device) {
      // OS guess
      if (device.osGuess) {
        hasGoodInfo = true;
        reasons.push("OS guess");
      }

      // LLDP info
      if (device.lldpInfo?.systemDescription) {
        hasGoodInfo = true;
        reasons.push("LLDP system info");
      }

      // CDP info
      if (device.cdpInfo?.platform || device.cdpInfo?.softwareVersion) {
        hasGoodInfo = true;
        reasons.push("CDP info");
      }

      // Device profile with open ports
      if (device.profile?.openPorts && device.profile.openPorts.some((p) => p.isOpen)) {
        hasGoodInfo = true;
        reasons.push("profile ports");
      }

      // HTTP info from profile
      if (device.profile?.httpInfo?.server) {
        hasGoodInfo = true;
        reasons.push("HTTP server");
      }
    }

    if (!hasGoodInfo) return;

    try {
      const apiBase = import.meta.env.VITE_API_BASE || "";
      logger.info(LogComponents.DISCOVERY, "Triggering auto vulnerability scan", {
        ip,
        reasons: reasons.join(", "),
      });
      await fetch(`${apiBase}/api/vulnerabilities/scan`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          targets: [ip],
        }),
      });
    } catch (error) {
      logger.debug(LogComponents.DISCOVERY, "Failed to trigger vulnerability scan", error);
    }
  }, [autoScanSettings.vulnScanEnabled, autoScanSettings.vulnAutoScan]);

  const handleDeepScan = useCallback(async (ip: string) => {
    setScanningDevices((prev) => new Set(prev).add(ip));

    try {
      const apiBase = import.meta.env.VITE_API_BASE || "";
      const response = await fetch(`${apiBase}/api/discovery/portscan`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          target: ip,
          ports: COMMON_PORTS,
          timeout: 2000,
        }),
      });

      if (response.ok) {
        const apiResponse = (await response.json()) as PortScanAPIResponse;
        // Transform backend response to frontend format
        const results: PortScanResult[] = apiResponse.services.map((svc) => ({
          port: svc.port,
          state: svc.state,
          service: svc.service,
          banner: svc.banner,
          version: svc.version,
          rtt: 0, // Backend doesn't return individual RTT per port
        }));
        setScanResults((prev) => {
          const next = new Map(prev);
          next.set(ip, {
            target: apiResponse.ip,
            results: results,
            scannedAt: new Date(),
          });
          return next;
        });

        // If vulnerability scanning is enabled with auto-scan, trigger vuln scan
        // Find the device from data to pass additional info
        const device = data?.devices.find((d) => d.ip === ip);
        if (apiResponse.services && apiResponse.services.length > 0) {
          triggerVulnScan(ip, device, apiResponse.services);
        }
      }
    } catch (error) {
      logger.error(LogComponents.DISCOVERY, "Deep scan failed", error);
    } finally {
      setScanningDevices((prev) => {
        const next = new Set(prev);
        next.delete(ip);
        return next;
      });
    }
  }, [triggerVulnScan, data?.devices]);

  // Track devices we've already auto-scanned to avoid duplicates
  const autoScannedDevices = useRef<Set<string>>(new Set());

  // Auto-scan devices after discovery completes (only if port scanning is enabled)
  // Triggers when new devices appear and discovery is not actively scanning
  useEffect(() => {
    // Only auto-scan if port scanning is enabled in settings
    if (!autoScanSettings.portScanEnabled) return;

    // Don't auto-scan while discovery is still in progress
    if (!data?.status || data.status.scanning) return;
    if (!data.devices || data.devices.length === 0) return;

    // Find devices we haven't auto-scanned yet
    const devicesToScan = data.devices.filter((device) => {
      if (!device.ip) return false;
      // Skip if already scanned or currently scanning
      if (autoScannedDevices.current.has(device.ip)) return false;
      if (scanningDevices.has(device.ip)) return false;
      if (scanResults.has(device.ip)) return false;
      return true;
    });

    if (devicesToScan.length === 0) return;

    // Mark these devices as queued for auto-scan
    devicesToScan.forEach((device) => {
      autoScannedDevices.current.add(device.ip);
    });

    logger.info(LogComponents.DISCOVERY, "Auto-scanning devices for open ports", {
      count: devicesToScan.length,
      portScanEnabled: autoScanSettings.portScanEnabled,
    });

    // Scan devices with a small delay between each to avoid overwhelming the network
    // Limit concurrent scans to 3 at a time
    const MAX_CONCURRENT_SCANS = 3;
    let scanIndex = 0;

    const scanNextBatch = () => {
      const batch = devicesToScan.slice(scanIndex, scanIndex + MAX_CONCURRENT_SCANS);
      if (batch.length === 0) return;

      batch.forEach((device) => {
        handleDeepScan(device.ip);
      });

      scanIndex += MAX_CONCURRENT_SCANS;

      // Schedule next batch after a delay
      if (scanIndex < devicesToScan.length) {
        setTimeout(scanNextBatch, 1000);
      }
    };

    // Start scanning with a small initial delay
    const timeoutId = setTimeout(scanNextBatch, 500);

    return () => clearTimeout(timeoutId);
  }, [data?.status?.scanning, data?.devices, handleDeepScan, scanningDevices, scanResults, autoScanSettings.portScanEnabled]);

  // Track devices we've already queued for vuln scan to avoid duplicates
  const vulnScannedDevices = useRef<Set<string>>(new Set());

  // Auto-trigger vulnerability scans based on device discovery info
  // This runs independently of port scanning - any device with good info gets vuln scanned
  useEffect(() => {
    // Only run if vuln scanning is enabled with auto-scan
    if (!autoScanSettings.vulnScanEnabled || !autoScanSettings.vulnAutoScan) return;

    // Don't run while discovery is still in progress
    if (!data?.status || data.status.scanning) return;
    if (!data.devices || data.devices.length === 0) return;

    // Find devices with good info that we haven't vuln-scanned yet
    const devicesToVulnScan = data.devices.filter((device) => {
      if (!device.ip) return false;
      // Skip if already queued for vuln scan
      if (vulnScannedDevices.current.has(device.ip)) return false;

      // Check if device has any good info for vulnerability scanning
      const hasGoodInfo =
        device.osGuess ||
        device.lldpInfo?.systemDescription ||
        device.cdpInfo?.platform ||
        device.cdpInfo?.softwareVersion ||
        device.profile?.httpInfo?.server ||
        (device.profile?.openPorts && device.profile.openPorts.some((p) => p.isOpen));

      return hasGoodInfo;
    });

    if (devicesToVulnScan.length === 0) return;

    // Mark devices as queued
    devicesToVulnScan.forEach((device) => {
      vulnScannedDevices.current.add(device.ip);
    });

    logger.info(LogComponents.DISCOVERY, "Auto-triggering vulnerability scans for devices with discovery info", {
      count: devicesToVulnScan.length,
    });

    // Trigger vuln scans with a small delay between each
    let index = 0;
    const triggerNext = () => {
      if (index >= devicesToVulnScan.length) return;
      const device = devicesToVulnScan[index];
      triggerVulnScan(device.ip, device);
      index++;
      if (index < devicesToVulnScan.length) {
        setTimeout(triggerNext, 200);
      }
    };

    const timeoutId = setTimeout(triggerNext, 300);
    return () => clearTimeout(timeoutId);
  }, [data?.status?.scanning, data?.devices, autoScanSettings.vulnScanEnabled, autoScanSettings.vulnAutoScan, triggerVulnScan]);

  // Extract data with safe defaults (must come before any hooks to avoid conditional hook calls)
  const rawDevices = data?.devices;
  const status = data?.status;
  // Ensure devices is an array (defensive check for malformed API responses)
  const devices = useMemo(
    () => (Array.isArray(rawDevices) ? rawDevices : []),
    [rawDevices]
  );
  const deviceCount = devices.length;

  // Helper function for IP to numeric conversion
  const ipToNum = useCallback((ip: string) => {
    const parts = ip.split(".").map(Number);
    return parts[0] * 16777216 + parts[1] * 65536 + parts[2] * 256 + parts[3];
  }, []);

  // Filter and sort devices
  const filteredDevices = useMemo(() => {
    let result = [...devices];

    // Apply search filter
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter((device) => {
        return (
          device.ip?.toLowerCase().includes(query) ||
          device.hostname?.toLowerCase().includes(query) ||
          device.vendor?.toLowerCase().includes(query) ||
          device.mac?.toLowerCase().includes(query) ||
          device.osGuess?.toLowerCase().includes(query)
        );
      });
    }

    // Apply sorting
    if (sortField) {
      result.sort((a, b) => {
        let aVal: string | number | null = null;
        let bVal: string | number | null = null;

        switch (sortField) {
          case "ip":
            // Sort IP numerically
            aVal = a.ip ? ipToNum(a.ip) : 0;
            bVal = b.ip ? ipToNum(b.ip) : 0;
            break;
          case "hostname":
            aVal = a.hostname?.toLowerCase() || "";
            bVal = b.hostname?.toLowerCase() || "";
            break;
          case "vendor":
            aVal = a.vendor?.toLowerCase() || "";
            bVal = b.vendor?.toLowerCase() || "";
            break;
          case "lastSeen":
            aVal = a.lastSeen ? new Date(a.lastSeen).getTime() : 0;
            bVal = b.lastSeen ? new Date(b.lastSeen).getTime() : 0;
            break;
        }

        if (aVal === null && bVal === null) return 0;
        if (aVal === null) return 1;
        if (bVal === null) return -1;

        let comparison = 0;
        if (typeof aVal === "number" && typeof bVal === "number") {
          comparison = aVal - bVal;
        } else {
          comparison = String(aVal).localeCompare(String(bVal));
        }

        return sortDirection === "asc" ? comparison : -comparison;
      });
    }

    return result;
  }, [devices, searchQuery, sortField, sortDirection, ipToNum]);

  const filteredCount = filteredDevices.length;

  // If no user sort applied, use default sorting: local first, then by discovery methods, then by IP
  const sortedDevices = useMemo(() => {
    // If user has applied search/sort, use filtered devices
    if (searchQuery.trim() || sortField) {
      return filteredDevices;
    }

    // Default sorting when no user filters applied
    return [...devices].sort((a, b) => {
      // Local devices first
      if (a.isLocal !== b.isLocal) {
        return a.isLocal ? -1 : 1;
      }
      // Then by discovery method count
      if (b.discoveryMethod.length !== a.discoveryMethod.length) {
        return b.discoveryMethod.length - a.discoveryMethod.length;
      }
      // Then by IP numerically - compare each octet
      const ipA = a.ip.split(".").map(Number);
      const ipB = b.ip.split(".").map(Number);
      // Compare octets using zip iterator pattern
      const ipIterA = ipA[Symbol.iterator]();
      const ipIterB = ipB[Symbol.iterator]();
      let resultA = ipIterA.next();
      let resultB = ipIterB.next();
      while (!resultA.done && !resultB.done) {
        if (resultA.value !== resultB.value)
          return resultA.value - resultB.value;
        resultA = ipIterA.next();
        resultB = ipIterB.next();
      }
      return 0;
    });
  }, [devices, filteredDevices, searchQuery, sortField]);

  // Early returns for loading/error states (after all hooks)
  // Fixes #674: Enable live regions for dynamic content updates
  if (loading) {
    return (
      <Card
        title={t("discovery.title")}
        icon={<ScanSearch className={iconTokens.size.md} />}
        status="loading"
        enableLiveRegion={true}
        ariaLabel="Network discovery scanning in progress"
      >
        <CardValue value={t("discovery.scanning")} size="lg" />
      </Card>
    );
  }

  if (!data || !status) {
    return (
      <Card
        title={t("discovery.title")}
        icon={<ScanSearch className={iconTokens.size.md} />}
        status="unknown"
        enableLiveRegion={true}
        ariaLabel="Network discovery - no data available"
      >
        <CardValue value={t("discovery.noData")} size="md" />
        {onScan && (
          <button
            type="button"
            onClick={onScan}
            className={cn(
              spacing.margin.top.heading,
              "w-full",
              button.size.md,
              "bg-brand-primary text-text-inverse",
              radius.md,
              "hover:bg-brand-primary/90 transition-colors font-medium body-small"
            )}
            aria-label="Start network discovery scan"
          >
            {t("discovery.startScan")}
          </button>
        )}
      </Card>
    );
  }

  // Categorize devices for summary
  const categories = categorizeDevices(devices);

  const getOverallStatus = (): Status => {
    if (status.scanning) return "loading";
    if (deviceCount === 0) return "warning";
    return "success";
  };

  const cardStatus = getOverallStatus();

  // Separate into local and extended for display
  const localDevices = sortedDevices.filter((d) => d.isLocal);
  const extendedDevices = sortedDevices.filter((d) => !d.isLocal);

  return (
    <Card
      title={t("discovery.title")}
      icon={<ScanSearch className={iconTokens.size.md} />}
      status={cardStatus}
      enableLiveRegion={true}
      ariaLabel={`Network discovery - ${deviceCount} devices found`}
      headerAction={
        onScan && (
          <button
            type="button"
            onClick={onScan}
            disabled={status.scanning}
            className={cn(
              spacing.chip.sm,
              "bg-brand-primary text-text-inverse",
              radius.md,
              "hover:bg-brand-primary/90 transition-colors font-medium caption disabled:opacity-50 disabled:cursor-not-allowed flex items-center",
              spacing.inline.sm
            )}
            aria-label={
              status.scanning ? "Scanning network" : "Start network scan"
            }
          >
            {status.scanning ? (
              <>
                <RefreshCw
                  className={cn(iconTokens.size.xs, "animate-spin")}
                  aria-hidden="true"
                />
                {t("discovery.scan")}
              </>
            ) : (
              t("discovery.scan")
            )}
          </button>
        )
      }
    >
      {/* Discovery Summary */}
      <DiscoverySummary
        status={status}
        deviceCount={deviceCount}
        categories={categories}
        t={t}
      />

      {/* Device Search/Sort Bar - show when there are devices */}
      {deviceCount > 0 && (
        <DeviceSearchBar
          searchQuery={searchQuery}
          onSearchChange={setSearchQuery}
          sortField={sortField}
          sortDirection={sortDirection}
          onSortChange={handleSortChange}
          deviceCount={deviceCount}
          filteredCount={filteredCount}
          t={t}
        />
      )}

      {/* Fixes #739: Removed redundant "Network Info" section - subnet is already in summary */}

      {/* Local Devices - Collapsible */}
      {localDevices.length > 0 && (
        <CollapsibleSection
          title={t("discovery.localNetwork")}
          variant="compact"
          defaultOpen={true}
          count={localDevices.length}
        >
          <div className="stack-sm max-h-60 overflow-y-auto">
            {localDevices.map((device) => {
              const deviceKey = device.mac || `ip:${device.ip}`;
              return (
                <DeviceRow
                  key={deviceKey}
                  device={device}
                  isExpanded={expandedDevices.has(deviceKey)}
                  onToggle={() => toggleDevice(deviceKey)}
                  onDeepScan={handleDeepScan}
                  isScanning={scanningDevices.has(device.ip)}
                  scanResult={scanResults.get(device.ip)}
                  onVulnerabilityClick={setSelectedDeviceForVuln}
                  t={t}
                />
              );
            })}
          </div>
        </CollapsibleSection>
      )}

      {/* Extended Networks - Collapsible */}
      {extendedDevices.length > 0 && (
        <CollapsibleSection
          title={t("discovery.extendedNetworks")}
          variant="compact"
          defaultOpen={false}
          count={extendedDevices.length}
        >
          <div className="stack-sm max-h-60 overflow-y-auto">
            {extendedDevices.map((device) => {
              const deviceKey = device.mac || `ip:${device.ip}`;
              return (
                <DeviceRow
                  key={deviceKey}
                  device={device}
                  isExpanded={expandedDevices.has(deviceKey)}
                  onToggle={() => toggleDevice(deviceKey)}
                  onDeepScan={handleDeepScan}
                  isScanning={scanningDevices.has(device.ip)}
                  scanResult={scanResults.get(device.ip)}
                  onVulnerabilityClick={setSelectedDeviceForVuln}
                  t={t}
                />
              );
            })}
          </div>
        </CollapsibleSection>
      )}

      {deviceCount === 0 && !status.scanning && (
        <p
          className={cn(
            "body-small text-text-muted text-center",
            spacing.pad.default
          )}
        >
          {t("discovery.noDevices")}
        </p>
      )}

      {/* Vulnerability Details Modal */}
      {selectedDeviceForVuln && (
        <VulnerabilityDetailsModal
          deviceIp={selectedDeviceForVuln}
          onClose={() => setSelectedDeviceForVuln(null)}
        />
      )}
    </Card>
  );
});
