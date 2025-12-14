import { useState, memo, useCallback, useMemo } from "react";
import { Card, CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { CollapsibleSection } from "../ui/CollapsibleSection";
import { getAuthHeaders } from "../../hooks/useAuth";
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

// Deep Scan (Port Scan) Types
export interface PortScanResult {
  ip: string;
  port: number;
  state: "open" | "closed" | "filtered";
  ttl: number;
  rtt: number; // nanoseconds
}

export interface DeepScanResult {
  target: string;
  results: PortScanResult[];
  osGuess?: string;
  scannedAt: Date;
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
}: {
  searchQuery: string;
  onSearchChange: (query: string) => void;
  sortField: SortField;
  sortDirection: SortDirection;
  onSortChange: (field: SortField) => void;
  deviceCount: number;
  filteredCount: number;
}) {
  return (
    <div className="space-y-2">
      {/* Search input */}
      <div className="relative">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted pointer-events-none" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search devices by IP, hostname, vendor, MAC..."
          className="w-full pl-9 pr-8 py-1.5 text-sm bg-surface-base border border-surface-border rounded-lg focus:outline-none focus:ring-1 focus:ring-brand-primary text-text-primary placeholder:text-text-muted"
        />
        {searchQuery && (
          <button
            type="button"
            onClick={() => onSearchChange("")}
            className="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
          >
            <X className="w-4 h-4" />
          </button>
        )}
      </div>

      {/* Sort buttons row */}
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <div className="flex items-center gap-1">
          <span className="text-xs text-text-muted">Sort:</span>
          {(["ip", "hostname", "vendor", "lastSeen"] as SortField[]).map(
            (field) => (
              <button
                key={field}
                type="button"
                onClick={() => onSortChange(field)}
                className={`px-2 py-0.5 text-xs rounded transition-colors flex items-center gap-1 ${
                  sortField === field
                    ? "bg-brand-primary/20 text-brand-primary"
                    : "bg-surface-hover text-text-muted hover:text-text-primary"
                }`}
              >
                {field === "ip"
                  ? "IP"
                  : field === "hostname"
                    ? "Name"
                    : field === "vendor"
                      ? "Vendor"
                      : "Seen"}
                {sortField === field &&
                  (sortDirection === "asc" ? (
                    <ChevronUp className="w-3 h-3" />
                  ) : (
                    <ChevronDown className="w-3 h-3" />
                  ))}
              </button>
            ),
          )}
        </div>
        {searchQuery && (
          <span className="text-xs text-text-muted">
            {filteredCount} of {deviceCount}
          </span>
        )}
      </div>
    </div>
  );
}

function formatLastSeen(dateStr: string): string {
  if (!dateStr) return "Never";
  const date = new Date(dateStr);
  // Check for invalid date or Go's zero time (year 1 or epoch)
  if (isNaN(date.getTime()) || date.getFullYear() < 2000) return "Never";
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 0) return "Never"; // Future date = invalid
  if (diffSec < 60) return "Just now";
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
  return `${Math.floor(diffSec / 86400)}d ago`;
}

// Discovery method colors - dark mode aware
// These use colored backgrounds for visual distinction between methods
const discoveryMethodColors: Record<DiscoveryMethod, string> = {
  arp: "bg-blue-500/20 text-blue-600 dark:text-blue-400",
  ndp: "bg-indigo-500/20 text-indigo-600 dark:text-indigo-400",
  lldp: "bg-green-500/20 text-green-600 dark:text-green-400",
  cdp: "bg-orange-500/20 text-orange-600 dark:text-orange-400",
  edp: "bg-purple-500/20 text-purple-600 dark:text-purple-400",
  mdns: "bg-teal-500/20 text-teal-600 dark:text-teal-400",
  ping: "bg-cyan-500/20 text-cyan-600 dark:text-cyan-400",
};

function MethodBadge({ method }: { method: DiscoveryMethod }) {
  return (
    <span
      className={`px-1.5 py-0.5 rounded text-xs font-medium uppercase ${discoveryMethodColors[method]}`}
    >
      {method}
    </span>
  );
}

// Icon mapping for device profile icons - maps service names to Lucide icons
const SERVICE_ICONS: Record<string, LucideIcon> = {
  ssh: Terminal,
  telnet: Terminal,
  web: Globe,
  "web-secure": Lock,
  ftp: FileText,
  mail: Mail,
  dns: Globe,
  snmp: Server,
  database: Database,
  cache: Container,
  printer: Printer,
  router: Router,
  switch: Server,
  firewall: Shield,
  storage: HardDrive,
  server: Server,
};

function ProfileIcons({
  icons,
  deviceType,
}: {
  icons?: string[];
  deviceType?: string;
}) {
  if (!icons || icons.length === 0) return null;

  return (
    <div className="flex items-center gap-0.5 flex-wrap">
      {icons.slice(0, 5).map((icon) => {
        const IconComponent = SERVICE_ICONS[icon];
        return (
          <span
            key={icon}
            className="p-1 rounded bg-indigo-500/20 text-indigo-400 flex items-center justify-center"
            title={`${icon}${deviceType ? ` (${deviceType})` : ""}`}
          >
            {IconComponent ? (
              <IconComponent className="w-3 h-3" />
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
        c.toLowerCase().includes("router"),
      ) ||
      device.lldpInfo?.capabilities?.some((c) =>
        c.toLowerCase().includes("router"),
      )
    ) {
      categories.routers++;
    } else if (
      icons.includes("switch") ||
      deviceType.includes("switch") ||
      device.cdpInfo?.capabilities?.some((c) =>
        c.toLowerCase().includes("switch"),
      ) ||
      device.lldpInfo?.capabilities?.some((c) =>
        c.toLowerCase().includes("bridge"),
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
}: {
  status: DiscoveryStatus;
  deviceCount: number;
  categories: ReturnType<typeof categorizeDevices>;
}) {
  // Build stat items with non-zero counts
  // Using dark mode aware colors for device categories
  const stats = [
    {
      icon: Router,
      label: "Routers",
      count: categories.routers,
      color: "text-blue-600 dark:text-blue-400",
    },
    {
      icon: Server,
      label: "Servers",
      count: categories.servers,
      color: "text-purple-600 dark:text-purple-400",
    },
    {
      icon: Monitor,
      label: "Workstations",
      count: categories.workstations,
      color: "text-green-600 dark:text-green-400",
    },
    {
      icon: Printer,
      label: "Printers",
      count: categories.printers,
      color: "text-orange-600 dark:text-orange-400",
    },
    {
      icon: Smartphone,
      label: "Mobile",
      count: categories.mobile,
      color: "text-cyan-600 dark:text-cyan-400",
    },
    {
      icon: Wifi,
      label: "Network",
      count: categories.network,
      color: "text-teal-600 dark:text-teal-400",
    },
  ].filter((s) => s.count > 0);

  return (
    <div className="bg-surface-hover rounded-lg p-3 space-y-2">
      {/* Status row */}
      <div className="flex items-center justify-between text-sm">
        <div className="flex items-center gap-2">
          {status.scanning ? (
            <>
              <RefreshCw className="w-4 h-4 text-status-info animate-spin" />
              <span className="text-status-info font-medium">Scanning...</span>
            </>
          ) : (
            <>
              <CheckCircle className="w-4 h-4 text-status-success" />
              <span className="text-status-success font-medium">Complete</span>
            </>
          )}
        </div>
        <div className="flex items-center gap-1.5 text-text-muted">
          <Clock className="w-3.5 h-3.5" />
          <span className="text-xs">{formatLastSeen(status.lastScan)}</span>
        </div>
      </div>

      {/* Network info row */}
      <div className="flex items-center justify-between text-xs text-text-muted">
        <span className="font-mono">{status.subnet || "Unknown subnet"}</span>
        <span>
          {deviceCount} device{deviceCount !== 1 ? "s" : ""} found
        </span>
      </div>

      {/* Category stats row */}
      {stats.length > 0 && (
        <div className="flex items-center gap-3 flex-wrap pt-1">
          {stats.map(({ icon: Icon, label, count, color }) => (
            <div
              key={label}
              className="flex items-center gap-1"
              title={`${count} ${label}`}
            >
              <Icon className={`w-3.5 h-3.5 ${color}`} />
              <span className="text-xs text-text-secondary">{count}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// Common port to service name mapping
const PORT_SERVICES: Record<number, string> = {
  21: "FTP",
  22: "SSH",
  23: "Telnet",
  25: "SMTP",
  53: "DNS",
  80: "HTTP",
  110: "POP3",
  143: "IMAP",
  443: "HTTPS",
  445: "SMB",
  993: "IMAPS",
  995: "POP3S",
  3306: "MySQL",
  3389: "RDP",
  5432: "PostgreSQL",
  5900: "VNC",
  6379: "Redis",
  8080: "HTTP-Alt",
  8443: "HTTPS-Alt",
  27017: "MongoDB",
};

function getServiceName(port: number): string {
  return PORT_SERVICES[port] || `Port ${port}`;
}

function DeviceRow({
  device,
  isExpanded,
  onToggle,
  onDeepScan,
  isScanning,
  scanResult,
  onVulnerabilityClick,
}: {
  device: DiscoveredDevice;
  isExpanded: boolean;
  onToggle: () => void;
  onDeepScan?: (ip: string) => void;
  isScanning?: boolean;
  scanResult?: DeepScanResult;
  onVulnerabilityClick?: (ip: string) => void;
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
    <div className="border border-surface-border rounded-lg overflow-hidden">
      <button
        type="button"
        onClick={onToggle}
        className="w-full p-2 sm:p-3 text-left hover:bg-surface-hover transition-colors focus:outline-none focus:ring-1 focus:ring-brand-primary"
      >
        <div className="flex items-center justify-between gap-2">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="font-mono text-sm text-text-primary">
                {device.ip || "No IP"}
              </span>
              {device.ipv6 && (
                <span
                  className="font-mono text-xs text-text-accent"
                  title={device.ipv6}
                >
                  {device.ipv6.length > 20
                    ? device.ipv6.substring(0, 20) + "..."
                    : device.ipv6}
                </span>
              )}
              {device.hostname && (
                <span
                  className="text-xs text-text-muted truncate max-w-[120px]"
                  title={device.hostname}
                >
                  ({device.hostname})
                </span>
              )}
              {openPorts.length > 0 && (
                <span className="text-xs bg-status-success/20 text-status-success px-1.5 py-0.5 rounded">
                  {openPorts.length} open
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
                  className={`inline-flex items-center gap-1 text-xs px-1.5 py-0.5 rounded cursor-pointer hover:opacity-80 transition-opacity ${
                    device.vulnerabilities.highestSeverity === "CRITICAL"
                      ? "bg-status-error/20 text-status-error"
                      : device.vulnerabilities.highestSeverity === "HIGH"
                        ? "bg-orange-500/20 text-orange-600 dark:text-orange-400" // High severity = orange (industry standard)
                        : device.vulnerabilities.highestSeverity === "MEDIUM"
                          ? "bg-status-warning/20 text-status-warning"
                          : "bg-status-info/20 text-status-info"
                  }`}
                  title="Click to view vulnerability details"
                >
                  <AlertTriangle className="w-3 h-3" />
                  {device.vulnerabilities.count} CVE
                </button>
              )}
            </div>
            <div className="flex items-center gap-1.5 mt-1 flex-wrap">
              {device.discoveryMethod.map((method) => (
                <MethodBadge key={method} method={method} />
              ))}
              {device.vendor && device.vendor !== "Unknown" && (
                <span
                  className="text-xs text-text-muted truncate max-w-[100px]"
                  title={device.vendor}
                >
                  {device.vendor}
                </span>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2 flex-shrink-0">
            {device.osGuess && (
              <span className="text-xs text-text-muted hidden sm:inline">
                {device.osGuess}
              </span>
            )}
            {onDeepScan && device.ip && (
              <button
                type="button"
                onClick={handleDeepScan}
                disabled={isScanning}
                className="px-2 py-1 text-xs bg-status-info/20 text-status-info rounded hover:bg-status-info/30 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                title="Deep Scan - scan common ports"
              >
                {isScanning ? (
                  <span className="flex items-center gap-1">
                    <span className="animate-spin">◐</span>
                  </span>
                ) : (
                  "Scan"
                )}
              </button>
            )}
            <span
              className={`text-lg transition-transform ${isExpanded ? "rotate-180" : ""}`}
            >
              {hasDetails || scanResult ? "▼" : "○"}
            </span>
          </div>
        </div>
      </button>

      {isExpanded && (
        <div className="px-2 sm:px-3 pb-2 sm:pb-3 pt-1 border-t border-surface-border bg-surface-base">
          <div className="space-y-1 text-xs">
            <CardRow label="MAC" value={device.mac} />
            {device.ipv6 && <CardRow label="IPv6" value={device.ipv6} />}
            {device.ipv6Addresses && device.ipv6Addresses.length > 1 && (
              <CardRow
                label="All IPv6"
                value={device.ipv6Addresses.join(", ")}
              />
            )}
            {device.isRouter && (
              <CardRow label="Router" value="Yes (IPv6 NDP)" />
            )}
            {device.vendor && <CardRow label="Vendor" value={device.vendor} />}
            {device.osGuess && (
              <CardRow label="OS Guess" value={device.osGuess} />
            )}
            {device.ttl && <CardRow label="TTL" value={device.ttl} />}
            <CardRow
              label="Last Seen"
              value={formatLastSeen(device.lastSeen)}
            />

            {/* Deep Scan Results */}
            {scanResult && (
              <>
                <CardDivider />
                <p className="font-medium text-text-primary mb-1">
                  Port Scan Results
                </p>
                {openPorts.length > 0 ? (
                  <div className="space-y-0.5">
                    {openPorts.map((result) => (
                      <div
                        key={result.port}
                        className="flex items-center justify-between py-0.5"
                      >
                        <span className="text-status-success">
                          {result.port}/{getServiceName(result.port)}
                        </span>
                        <span className="text-text-muted">
                          {(result.rtt / 1000000).toFixed(1)}ms
                        </span>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-text-muted">No open ports found</p>
                )}
              </>
            )}

            {/* Auto-Profile Results */}
            {device.profile && (
              <>
                <CardDivider />
                <p className="font-medium text-text-primary mb-1">
                  Auto-Profile
                  {device.profile.deviceType &&
                    device.profile.deviceType !== "unknown" && (
                      <span className="ml-2 text-xs font-normal text-text-muted">
                        ({device.profile.deviceType})
                      </span>
                    )}
                </p>
                {device.profile.httpInfo && (
                  <div className="space-y-0.5 mb-1">
                    <CardRow
                      label={device.profile.httpInfo.isHttps ? "HTTPS" : "HTTP"}
                      value={`Port ${device.profile.httpInfo.port} (${device.profile.httpInfo.statusCode})`}
                    />
                    {device.profile.httpInfo.title && (
                      <CardRow
                        label="Title"
                        value={device.profile.httpInfo.title}
                      />
                    )}
                    {device.profile.httpInfo.server && (
                      <CardRow
                        label="Server"
                        value={device.profile.httpInfo.server}
                      />
                    )}
                  </div>
                )}
                {profileOpenPorts.length > 0 && (
                  <div className="space-y-0.5">
                    <p className="text-text-muted text-[10px] uppercase tracking-wide mb-0.5">
                      Open Ports
                    </p>
                    <div className="flex flex-wrap gap-1">
                      {profileOpenPorts.map((port) => (
                        <span
                          key={port.port}
                          className="px-1.5 py-0.5 rounded text-[10px] bg-status-success/20 text-status-success"
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
                <p className="font-medium text-text-primary mb-1">LLDP Info</p>
                <CardRow label="Chassis ID" value={device.lldpInfo.chassisId} />
                <CardRow label="Port ID" value={device.lldpInfo.portId} />
                {device.lldpInfo.systemName && (
                  <CardRow
                    label="System Name"
                    value={device.lldpInfo.systemName}
                  />
                )}
                {device.lldpInfo.portDescription && (
                  <CardRow
                    label="Port Desc"
                    value={device.lldpInfo.portDescription}
                  />
                )}
                {device.lldpInfo.systemDescription && (
                  <CardRow
                    label="System Desc"
                    value={device.lldpInfo.systemDescription}
                  />
                )}
                {device.lldpInfo.capabilities &&
                  device.lldpInfo.capabilities.length > 0 && (
                    <CardRow
                      label="Capabilities"
                      value={device.lldpInfo.capabilities.join(", ")}
                    />
                  )}
                {device.lldpInfo.managementAddress && (
                  <CardRow
                    label="Mgmt IP"
                    value={device.lldpInfo.managementAddress}
                  />
                )}
              </>
            )}

            {device.cdpInfo && (
              <>
                <CardDivider />
                <p className="font-medium text-text-primary mb-1">CDP Info</p>
                <CardRow label="Device ID" value={device.cdpInfo.deviceId} />
                <CardRow label="Port ID" value={device.cdpInfo.portId} />
                {device.cdpInfo.platform && (
                  <CardRow label="Platform" value={device.cdpInfo.platform} />
                )}
                {device.cdpInfo.softwareVersion && (
                  <CardRow
                    label="Software"
                    value={device.cdpInfo.softwareVersion}
                  />
                )}
                {device.cdpInfo.capabilities &&
                  device.cdpInfo.capabilities.length > 0 && (
                    <CardRow
                      label="Capabilities"
                      value={device.cdpInfo.capabilities.join(", ")}
                    />
                  )}
                {device.cdpInfo.nativeVlan && (
                  <CardRow
                    label="Native VLAN"
                    value={device.cdpInfo.nativeVlan}
                  />
                )}
                {device.cdpInfo.managementAddress && (
                  <CardRow
                    label="Mgmt IP"
                    value={device.cdpInfo.managementAddress}
                  />
                )}
              </>
            )}

            {device.edpInfo && (
              <>
                <CardDivider />
                <p className="font-medium text-text-primary mb-1">EDP Info</p>
                <CardRow label="Device ID" value={device.edpInfo.deviceId} />
                {device.edpInfo.displayName && (
                  <CardRow
                    label="Display Name"
                    value={device.edpInfo.displayName}
                  />
                )}
                <CardRow label="Port ID" value={device.edpInfo.portId} />
                {device.edpInfo.platform && (
                  <CardRow label="Platform" value={device.edpInfo.platform} />
                )}
                {device.edpInfo.softwareVersion && (
                  <CardRow
                    label="Software"
                    value={device.edpInfo.softwareVersion}
                  />
                )}
                {device.edpInfo.vlan && (
                  <CardRow label="VLAN" value={device.edpInfo.vlan} />
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
  const [expandedDevices, setExpandedDevices] = useState<Set<string>>(
    new Set(),
  );
  const [scanningDevices, setScanningDevices] = useState<Set<string>>(
    new Set(),
  );
  const [scanResults, setScanResults] = useState<Map<string, DeepScanResult>>(
    new Map(),
  );
  // Search and sort state
  const [searchQuery, setSearchQuery] = useState("");
  const [sortField, setSortField] = useState<SortField>(null);
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");

  // Vulnerability modal state
  const [selectedDeviceForVuln, setSelectedDeviceForVuln] = useState<
    string | null
  >(null);

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
    [sortField, sortDirection],
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

  const handleDeepScan = useCallback(async (ip: string) => {
    setScanningDevices((prev) => new Set(prev).add(ip));

    try {
      const apiBase = import.meta.env.VITE_API_BASE || "";
      const response = await fetch(`${apiBase}/api/discovery/portscan`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...getAuthHeaders(),
        },
        body: JSON.stringify({
          target: ip,
          ports: COMMON_PORTS,
          timeout: 2000,
        }),
      });

      if (response.ok) {
        const data = (await response.json()) as {
          target: string;
          results: PortScanResult[];
        };
        setScanResults((prev) => {
          const next = new Map(prev);
          next.set(ip, {
            target: data.target,
            results: data.results,
            scannedAt: new Date(),
          });
          return next;
        });
      }
    } catch (error) {
      console.error("Deep scan failed:", error);
    } finally {
      setScanningDevices((prev) => {
        const next = new Set(prev);
        next.delete(ip);
        return next;
      });
    }
  }, []);

  // Extract data with safe defaults (must come before any hooks to avoid conditional hook calls)
  const rawDevices = data?.devices;
  const status = data?.status;
  // Ensure devices is an array (defensive check for malformed API responses)
  const devices = useMemo(
    () => (Array.isArray(rawDevices) ? rawDevices : []),
    [rawDevices],
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
      // Then by IP numerically
      const ipA = a.ip.split(".").map(Number);
      const ipB = b.ip.split(".").map(Number);
      for (let i = 0; i < 4; i++) {
        if (ipA[i] !== ipB[i]) return (ipA[i] || 0) - (ipB[i] || 0);
      }
      return 0;
    });
  }, [devices, filteredDevices, searchQuery, sortField]);

  // Early returns for loading/error states (after all hooks)
  if (loading) {
    return (
      <Card
        title="Network Discovery"
        icon={<ScanSearch className="w-5 h-5" />}
        status="loading"
      >
        <CardValue value="Scanning..." size="lg" />
      </Card>
    );
  }

  if (!data || !status) {
    return (
      <Card
        title="Network Discovery"
        icon={<ScanSearch className="w-5 h-5" />}
        status="unknown"
      >
        <CardValue value="No data" size="md" />
        {onScan && (
          <button
            type="button"
            onClick={onScan}
            className="mt-3 w-full py-2 px-4 bg-brand-primary text-text-inverse rounded-lg hover:bg-brand-primary/90 transition-colors font-medium text-sm"
          >
            Start Scan
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
      title="Network Discovery"
      icon={<ScanSearch className="w-5 h-5" />}
      status={cardStatus}
      headerAction={
        onScan && (
          <button
            type="button"
            onClick={onScan}
            disabled={status.scanning}
            className="py-1 px-2.5 bg-brand-primary text-text-inverse rounded-lg hover:bg-brand-primary/90 transition-colors font-medium text-xs disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5"
          >
            {status.scanning ? (
              <>
                <RefreshCw className="w-3 h-3 animate-spin" />
                Scan
              </>
            ) : (
              "Scan"
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
        />
      )}

      {/* Network Info - Collapsible */}
      <CollapsibleSection
        title="Network Info"
        variant="compact"
        defaultOpen={false}
      >
        <div className="space-y-1 text-xs">
          {status.localIP && (
            <CardRow label="Local IP" value={status.localIP} />
          )}
          {status.interface && (
            <CardRow label="Interface" value={status.interface} />
          )}
        </div>
      </CollapsibleSection>

      {/* Local Devices - Collapsible */}
      {localDevices.length > 0 && (
        <CollapsibleSection
          title="Local Network"
          variant="compact"
          defaultOpen={true}
          count={localDevices.length}
        >
          <div className="space-y-2 max-h-60 overflow-y-auto">
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
                />
              );
            })}
          </div>
        </CollapsibleSection>
      )}

      {/* Extended Networks - Collapsible */}
      {extendedDevices.length > 0 && (
        <CollapsibleSection
          title="Extended Networks"
          variant="compact"
          defaultOpen={false}
          count={extendedDevices.length}
        >
          <div className="space-y-2 max-h-60 overflow-y-auto">
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
                />
              );
            })}
          </div>
        </CollapsibleSection>
      )}

      {deviceCount === 0 && !status.scanning && (
        <p className="text-sm text-text-muted text-center py-4">
          No devices discovered. Click Scan to discover network devices.
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
