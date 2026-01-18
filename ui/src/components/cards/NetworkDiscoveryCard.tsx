// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Complex component
import type React from "react";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { usePipelineStatus } from "../../hooks/usePipelineStatus";
import { api } from "../../api";
import { LogComponents, logger } from "../../lib/logger";
import {
  button,
  category as categoryTheme,
  cn,
  icon as iconTokens,
  radius,
  spacing,
} from "../../styles/theme";
import { Card, CardValue, type Status } from "../ui/Card";
import {
  CheckCircle,
  ChevronDown,
  ChevronUp,
  Clock,
  Maximize2,
  Monitor,
  Printer,
  RefreshCw,
  Router,
  ScanSearch,
  Server,
  Smartphone,
  Wifi,
} from "../ui/Icons";
import { DiscoveryModal } from "./DiscoveryModal";
import { PipelineProgress } from "./PipelineProgress";
import { VulnerabilityDetailsModal } from "./VulnerabilityDetailsModal";

export interface LldpInfo {
  chassisId: string;
  portId: string;
  portDescription?: string;
  systemName?: string;
  systemDescription?: string;
  capabilities?: string[];
  managementAddress?: string;
}

export interface CdpInfo {
  deviceId: string;
  portId: string;
  platform?: string;
  softwareVersion?: string;
  capabilities?: string[];
  managementAddress?: string;
  nativeVlan?: number;
  voiceVlan?: number;
}

export interface EdpInfo {
  deviceId: string;
  displayName?: string;
  portId: string;
  platform?: string;
  softwareVersion?: string;
  vlan?: number;
}

export interface NdpInfo {
  linkLayerAddress: string;
  isRouter: boolean;
  reachableTime?: number;
  retransTimer?: number;
  flags?: number;
  lastAdvertisement?: string;
}

export type DiscoveryMethod = "arp" | "ndp" | "lldp" | "cdp" | "edp" | "mdns" | "ping" | "snmp";

// Auto-profiling types from backend
export interface OpenPort {
  port: number;
  protocol: string;
  service?: string;
  banner?: string;
  isOpen: boolean;
}

export interface HttpInfo {
  port: number;
  statusCode: number;
  title?: string;
  server?: string;
  isHttps: boolean;
}

export interface DeviceProfile {
  profiledAt: string;
  openPorts?: OpenPort[];
  httpInfo?: HttpInfo;
  deviceType?: string;
  deviceIcons?: string[];
}

// SNMP-related interfaces for extended device data
export interface SnmpSystemInfo {
  sysDescr?: string;
  sysObjectId?: string;
  sysName?: string;
  sysContact?: string;
  sysLocation?: string;
  sysUpTime?: number;
}

export interface SnmpInterface {
  index: number;
  name?: string;
  description?: string;
  alias?: string;
  type?: number;
  mtu?: number;
  speedMbps?: number;
  mac?: string;
  adminStatus?: string;
  operStatus?: string;
}

export interface SnmpIpAddress {
  address: string;
  prefix?: number;
  ifIndex: number;
  type?: string;
}

export interface SnmpVlan {
  id: number;
  name?: string;
  status?: string;
  egressPorts?: number[];
}

export interface SnmpEntity {
  index: number;
  description?: string;
  class?: string;
  name?: string;
  hardwareRev?: string;
  firmwareRev?: string;
  softwareRev?: string;
  serialNum?: string;
  modelName?: string;
}

export interface SnmpFullData {
  collectedAt?: string;
  system?: SnmpSystemInfo;
  interfaces?: SnmpInterface[];
  ipAddresses?: SnmpIpAddress[];
  vlans?: SnmpVlan[];
  inventory?: SnmpEntity[];
  errors?: string[];
}

export interface DiscoveredDevice {
  ip: string;
  ipv6?: string;
  ipv6Addresses?: string[];
  mac: string;
  hostname?: string; // DNS PTR resolved name
  netbiosName?: string; // Windows NetBIOS name (UDP 137)
  mdnsName?: string; // mDNS/Bonjour .local name
  displayName?: string; // Best available name for UI display
  vendor?: string;
  osGuess?: string;
  ttl?: number;
  discoveryMethod: DiscoveryMethod[];
  lastSeen: string;
  isLocal: boolean; // true if on local subnet, false for extended networks
  isRouter?: boolean;
  lldpInfo?: LldpInfo;
  cdpInfo?: CdpInfo;
  edpInfo?: EdpInfo;
  ndpInfo?: NdpInfo;
  profile?: DeviceProfile;
  snmpData?: SnmpFullData; // Extended SNMP data from Phase 3 scanning
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
  subnets?: string[]; // All subnets being scanned (I3)
  // biome-ignore lint/style/useNamingConvention: Matches backend API
  localIP: string; // Matches backend json:"localIP"
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
interface PortScanApiResponse {
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

// Sorting types for device list
type SortField = "ip" | "hostname" | "vendor" | "lastSeen" | null;
type SortDirection = "asc" | "desc";

// Format last seen timestamp to human-readable relative time
function formatLastSeen(
  dateStr: string,
  t: ReturnType<typeof useTranslation<"cards">>["t"],
): string {
  if (!dateStr) {
    return t("discovery.never");
  }
  const date = new Date(dateStr);
  // Check for invalid date or Go's zero time (year 1 or epoch)
  if (Number.isNaN(date.getTime()) || date.getFullYear() < 2000) {
    return t("discovery.never");
  }
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 0) {
    return t("discovery.never"); // Future date = invalid
  }
  if (diffSec < 60) {
    return t("discovery.justNow");
  }
  if (diffSec < 3600) {
    return t("discovery.mAgo", { min: Math.floor(diffSec / 60) });
  }
  if (diffSec < 86400) {
    return t("discovery.hAgo", { hour: Math.floor(diffSec / 3600) });
  }
  return t("discovery.dAgo", { day: Math.floor(diffSec / 86400) });
}

/**
 * Convert host IP/CIDR to network address (fixes #738)
 * e.g., "192.168.64.7/24" -> "192.168.64.0/24"
 */
function calculateNetworkAddress(cidr: string): string {
  const [ip, maskStr] = cidr.split("/");
  if (!(ip && maskStr)) {
    return cidr;
  }

  const mask = Number.parseInt(maskStr, 10);
  if (Number.isNaN(mask) || mask < 0 || mask > 32) {
    return cidr;
  }

  const octets = ip.split(".").map(Number);
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

  return `${networkOctets.join(".")}/${mask}`;
}

/**
 * SubnetList component for I3 - displays subnets with smart rollup.
 * - Inline display for ≤5 subnets
 * - Expandable dropdown for >5 subnets
 */
function _subnetList({
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
    return <span class="font-mono">{unknownLabel}</span>;
  }

  // Single subnet - simple display
  if (allSubnets.length === 1) {
    return <span class="font-mono">{allSubnets[0]}</span>;
  }

  // ≤5 subnets - inline display
  if (allSubnets.length <= 5) {
    return <span class="font-mono">{allSubnets.join(", ")}</span>;
  }

  // >5 subnets - collapsible display
  if (!expanded) {
    return (
      <button
        type="button"
        onClick={(): void => setExpanded(true)}
        class="font-mono text-text-muted hover:text-text-primary flex items-center gap-1"
      >
        <span>{allSubnets.length} subnets</span>
        <ChevronDown class={iconTokens.size.xs} />
      </button>
    );
  }

  return (
    <div class="flex flex-col gap-1">
      <button
        type="button"
        onClick={(): void => setExpanded(false)}
        class="font-mono text-text-muted hover:text-text-primary flex items-center gap-1"
      >
        <span>{allSubnets.length} subnets</span>
        <ChevronUp class={iconTokens.size.xs} />
      </button>
      <div class="flex flex-wrap gap-1">
        {allSubnets.map((subnet) => (
          <span key={subnet} class="font-mono text-xs">
            {subnet}
          </span>
        ))}
      </div>
    </div>
  );
}

interface CategoryCounts {
  routers: number;
  servers: number;
  workstations: number;
  printers: number;
  mobile: number;
  network: number;
  other: number;
}

// Device type categorization based on profile icons and device type
function categorizeDevices(devices: DiscoveredDevice[]): CategoryCounts {
  const categories = {
    routers: 0,
    servers: 0,
    workstations: 0,
    printers: 0,
    mobile: 0,
    network: 0, // switches, APs
    other: 0,
  };

  for (const device of devices) {
    const deviceType = device.profile?.deviceType?.toLowerCase() || "";
    const icons = device.profile?.deviceIcons || [];

    if (
      icons.includes("router") ||
      deviceType.includes("router") ||
      device.cdpInfo?.capabilities?.some((c) => c.toLowerCase().includes("router")) ||
      device.lldpInfo?.capabilities?.some((c) => c.toLowerCase().includes("router"))
    ) {
      categories.routers++;
    } else if (
      icons.includes("switch") ||
      deviceType.includes("switch") ||
      device.cdpInfo?.capabilities?.some((c) => c.toLowerCase().includes("switch")) ||
      device.lldpInfo?.capabilities?.some((c) => c.toLowerCase().includes("bridge"))
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
  }

  return categories;
}

// Summary bar component
function _discoverySummary({
  status,
  deviceCount,
  categories,
  pipelineStatus,
  onCancelPipeline,
  t,
}: {
  status: DiscoveryStatus;
  deviceCount: number;
  categories: ReturnType<typeof categorizeDevices>;
  pipelineStatus?: ReturnType<typeof usePipelineStatus>["status"];
  onCancelPipeline?: () => void;
  t: ReturnType<typeof useTranslation<"cards">>["t"];
}): React.ReactElement {
  // Check if pipeline is actively running
  const isPipelineRunning =
    pipelineStatus &&
    pipelineStatus.state !== "idle" &&
    pipelineStatus.state !== "complete" &&
    pipelineStatus.state !== "failed" &&
    pipelineStatus.state !== "canceled";

  // Show pipeline progress when running
  if (isPipelineRunning && pipelineStatus) {
    return (
      <div class="stack-sm">
        <PipelineProgress status={pipelineStatus} onCancel={onCancelPipeline} />
      </div>
    );
  }

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
    <div class="stack-sm">
      {/* Status row */}
      <div class="flex items-center justify-between body-small">
        <div class={cn("flex items-center", spacing.gap.compact)}>
          {status.scanning ? (
            <>
              <RefreshCw class={cn(iconTokens.size.sm, "text-status-info animate-spin")} />
              <span class="text-status-info font-medium">{t("discovery.scanning")}</span>
            </>
          ) : (
            <>
              <CheckCircle class={cn(iconTokens.size.sm, "text-status-success")} />
              <span class="text-status-success font-medium">{t("discovery.complete")}</span>
            </>
          )}
        </div>
        <div class={cn("flex items-center", spacing.inline.sm, "text-text-muted")}>
          <Clock class={iconTokens.size.sm} />
          <span class="caption">{formatLastSeen(status.lastScan, t)}</span>
        </div>
      </div>

      {/* Simplified network info row - I3: Uses SubnetList for multi-subnet display */}
      <div class="flex items-center justify-between caption text-text-muted">
        <subnetList
          subnets={status.subnets}
          fallbackSubnet={status.subnet}
          unknownLabel={t("discovery.unknownSubnet")}
        />
        <span>
          {deviceCount === 1
            ? t("discovery.deviceFound", { count: deviceCount })
            : t("discovery.devicesFound", { count: deviceCount })}
        </span>
      </div>

      {/* Category stats row */}
      {stats.length > 0 && (
        <div
          class={cn(
            "flex items-center",
            spacing.gap.default,
            "flex-wrap",
            spacing.padding.top.heading,
          )}
        >
          {stats.map(({ icon: ICON, label, count, color }) => (
            <div
              key={label}
              class={cn("flex items-center", spacing.gap.tight)}
              title={`${count} ${label}`}
            >
              <ICON class={cn(iconTokens.size.sm, color)} />
              <span class="caption text-text-secondary">{count}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// Common ports to scan for Deep Scan
const COMMON_PORTS: number[] = [
  21, 22, 23, 25, 53, 80, 110, 143, 443, 445, 993, 995, 3306, 3389, 5432, 5900, 6379, 8080, 8443,
  27017,
];

export const NetworkDiscoveryCard: React.NamedExoticComponent<NetworkDiscoveryCardProps> = memo(
  function networkDiscoveryCard({
    data,
    loading,
    onScan,
  }: NetworkDiscoveryCardProps): React.ReactElement | null {
    const { t } = useTranslation("cards");
    const [_expandedDevices, SET_EXPANDED_DEVICES] = useState<Set<string>>(new Set());
    const [scanningDevices, setScanningDevices] = useState<Set<string>>(new Set());
    const [scanResults, setScanResults] = useState<Map<string, DeepScanResult>>(new Map());
    // Search and sort state (kept for modal use)
    const [searchQuery, _setSearchQuery] = useState("");
    const [sortField, setSortField] = useState<SortField>(null);
    const [sortDirection, setSortDirection] = useState<SortDirection>("asc");

    // Pipeline status hook for multi-phase progress display
    const { status: pipelineStatus, startPipeline, cancelPipeline } = usePipelineStatus();

    // Check if pipeline is actively running
    const isPipelineRunning =
      pipelineStatus.state !== "idle" &&
      pipelineStatus.state !== "complete" &&
      pipelineStatus.state !== "failed" &&
      pipelineStatus.state !== "canceled";

    // Settings for auto-scan behavior - fetched from API
    const [autoScanSettings, setAutoScanSettings] = useState<DiscoverySettingsForAutoScan>({
      portScanEnabled: false,
      vulnScanEnabled: false,
      vulnAutoScan: false,
    });

    // Vulnerability modal state
    const [selectedDeviceForVuln, setSelectedDeviceForVuln] = useState<string | null>(null);

    // Full-screen modal state
    const [isModalOpen, setIsModalOpen] = useState(false);

    // Fetch settings for auto-scan behavior on mount
    useEffect(() => {
      const fetchSettings = async (): Promise<void> => {
        const apiBase = import.meta.env.VITE_API_BASE || "";
        try {
          // Fetch discovery options from correct endpoint
          const discoveryResponse = await fetch(`${apiBase}/api/v1/shell/discovery/options`, {
            credentials: "include",
          });
          if (discoveryResponse.ok) {
            // biome-ignore lint/nursery/useAwaitThenable: Response.json() returns Promise
            const discoveryData = await discoveryResponse.json();
            // Backend returns { options: { PortScan: { Enabled: true, ... } } }
            const portScanEnabled = discoveryData?.options?.PortScan?.Enabled ?? false;

            // Fetch vulnerability settings from correct endpoint
            const vulnResponse = await fetch(`${apiBase}/api/v1/shell/vulnerabilities/settings`, {
              credentials: "include",
            });
            let vulnEnabled = false;
            let vulnAutoScan = false;
            if (vulnResponse.ok) {
              // biome-ignore lint/nursery/useAwaitThenable: Response.json() returns Promise
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
          logger.debug(LogComponents.Discovery, "Failed to fetch auto-scan settings", error);
        }
      };

      fetchSettings().catch(() => {
        // Error already logged in fetchSettings
      });
    }, []);

    // Toggle sort field/direction (kept for modal use)
    const _handleSortChange = useCallback(
      (field: SortField): void => {
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

    const _toggleDevice = (mac: string): void => {
      SET_EXPANDED_DEVICES((prev) => {
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
    const triggerVulnScan = useCallback(
      async (ip: string, device?: DiscoveredDevice, services?: ServiceInfo[]) => {
        if (!(autoScanSettings.vulnScanEnabled && autoScanSettings.vulnAutoScan)) {
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
            (s) => s.state === "open" && (s.banner || s.version || s.service !== "unknown"),
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
          if (device.profile?.openPorts?.some((p) => p.isOpen)) {
            hasGoodInfo = true;
            reasons.push("profile ports");
          }

          // HTTP info from profile
          if (device.profile?.httpInfo?.server) {
            hasGoodInfo = true;
            reasons.push("HTTP server");
          }
        }

        if (!hasGoodInfo) {
          return;
        }

        try {
          logger.info(LogComponents.Discovery, "Triggering auto vulnerability scan", {
            ip,
            reasons: reasons.join(", "),
          });
          await api.post("/api/v1/shell/vulnerabilities/scan", { targets: [ip] });
        } catch (error) {
          logger.debug(LogComponents.Discovery, "Failed to trigger vulnerability scan", error);
        }
      },
      [autoScanSettings.vulnScanEnabled, autoScanSettings.vulnAutoScan],
    );

    const handleDeepScan = useCallback(
      async (ip: string) => {
        setScanningDevices((prev) => new Set(prev).add(ip));

        try {
          const apiResponse = await api.post<PortScanApiResponse>(
            "/api/v1/shell/discovery/portscan",
            {
              target: ip,
              ports: COMMON_PORTS,
              timeout: 2000,
            },
          );

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
            // Fixes #904: Limit stored scan results to prevent unbounded memory growth
            const MAX_SCAN_RESULTS = 100;
            if (next.size > MAX_SCAN_RESULTS) {
              // Remove oldest entries
              const entries = [...next.entries()].sort(
                (a, b) => a[1].scannedAt.getTime() - b[1].scannedAt.getTime(),
              );
              while (next.size > MAX_SCAN_RESULTS && entries.length > 0) {
                const oldest = entries.shift();
                if (oldest) {
                  next.delete(oldest[0]);
                }
              }
            }
            return next;
          });

          // If vulnerability scanning is enabled with auto-scan, trigger vuln scan
          // Find the device from data to pass additional info
          const device = data?.devices.find((d) => d.ip === ip);
          if (apiResponse.services && apiResponse.services.length > 0) {
            await triggerVulnScan(ip, device, apiResponse.services);
          }
        } catch (error) {
          logger.error(LogComponents.Discovery, "Deep scan failed", error);
        } finally {
          setScanningDevices((prev) => {
            const next = new Set(prev);
            next.delete(ip);
            return next;
          });
        }
      },
      [triggerVulnScan, data?.devices],
    );

    // Track devices we've already auto-scanned to avoid duplicates
    const autoScannedDevices = useRef<Set<string>>(new Set());

    // Fixes #905: Clear auto-scanned tracking when a new scan cycle starts
    useEffect(() => {
      if (data?.status?.scanning) {
        autoScannedDevices.current.clear();
      }
    }, [data?.status?.scanning]);

    // Auto-scan devices after discovery completes (only if port scanning is enabled)
    // Triggers when new devices appear and discovery is not actively scanning
    useEffect(() => {
      // Only auto-scan if port scanning is enabled in settings
      if (!autoScanSettings.portScanEnabled) {
        return;
      }

      // Don't auto-scan while discovery is still in progress
      if (!data?.status || data.status.scanning) {
        return;
      }
      if (!data.devices || data.devices.length === 0) {
        return;
      }

      // Find devices we haven't auto-scanned yet
      const devicesToScan = data.devices.filter((device) => {
        if (!device.ip) {
          return false;
        }
        // Skip if already scanned or currently scanning
        if (autoScannedDevices.current.has(device.ip)) {
          return false;
        }
        if (scanningDevices.has(device.ip)) {
          return false;
        }
        if (scanResults.has(device.ip)) {
          return false;
        }
        return true;
      });

      if (devicesToScan.length === 0) {
        return;
      }

      // Mark these devices as queued for auto-scan
      for (const device of devicesToScan) {
        autoScannedDevices.current.add(device.ip);
      }

      logger.info(LogComponents.Discovery, "Auto-scanning devices for open ports", {
        count: devicesToScan.length,
        portScanEnabled: autoScanSettings.portScanEnabled,
      });

      // Fixes #906: Track all timeout IDs for proper cleanup
      const timeoutIds: ReturnType<typeof setTimeout>[] = [];

      // Scan devices with a small delay between each to avoid overwhelming the network
      // Limit concurrent scans to 3 at a time
      const MAX_CONCURRENT_SCANS = 3;
      let scanIndex = 0;

      const scanNextBatch = (): void => {
        const batch = devicesToScan.slice(scanIndex, scanIndex + MAX_CONCURRENT_SCANS);
        if (batch.length === 0) {
          return;
        }

        for (const device of batch) {
          handleDeepScan(device.ip).catch(() => {
            // Errors handled in handleDeepScan
          });
        }

        scanIndex += MAX_CONCURRENT_SCANS;

        // Schedule next batch after a delay
        if (scanIndex < devicesToScan.length) {
          const tid = setTimeout(scanNextBatch, 1000);
          timeoutIds.push(tid);
        }
      };

      // Start scanning with a small initial delay
      const initialTimeoutId = setTimeout(scanNextBatch, 500);
      timeoutIds.push(initialTimeoutId);

      // Fixes #906: Clean up all scheduled timeouts on unmount/re-render
      return (): void => {
        for (const tid of timeoutIds) {
          clearTimeout(tid);
        }
      };
    }, [
      data?.status,
      data?.devices,
      handleDeepScan,
      scanningDevices,
      scanResults,
      autoScanSettings.portScanEnabled,
    ]);

    // Track devices we've already queued for vuln scan to avoid duplicates
    const vulnScannedDevices = useRef<Set<string>>(new Set());

    // Fixes #905: Clear vuln-scanned tracking when a new scan cycle starts
    useEffect(() => {
      if (data?.status?.scanning) {
        vulnScannedDevices.current.clear();
      }
    }, [data?.status?.scanning]);

    // Auto-trigger vulnerability scans based on device discovery info
    // This runs independently of port scanning - any device with good info gets vuln scanned
    useEffect(() => {
      // Only run if vuln scanning is enabled with auto-scan
      if (!(autoScanSettings.vulnScanEnabled && autoScanSettings.vulnAutoScan)) {
        return;
      }

      // Don't run while discovery is still in progress
      if (!data?.status || data.status.scanning) {
        return;
      }
      if (!data.devices || data.devices.length === 0) {
        return;
      }

      // Find devices with good info that we haven't vuln-scanned yet
      const devicesToVulnScan = data.devices.filter((device) => {
        if (!device.ip) {
          return false;
        }
        // Skip if already queued for vuln scan
        if (vulnScannedDevices.current.has(device.ip)) {
          return false;
        }

        // Check if device has any good info for vulnerability scanning
        const hasGoodInfo =
          device.osGuess ||
          device.lldpInfo?.systemDescription ||
          device.cdpInfo?.platform ||
          device.cdpInfo?.softwareVersion ||
          device.profile?.httpInfo?.server ||
          device.profile?.openPorts?.some((p) => p.isOpen);

        return hasGoodInfo;
      });

      if (devicesToVulnScan.length === 0) {
        return;
      }

      // Mark devices as queued
      for (const device of devicesToVulnScan) {
        vulnScannedDevices.current.add(device.ip);
      }

      logger.info(
        LogComponents.Discovery,
        "Auto-triggering vulnerability scans for devices with discovery info",
        {
          count: devicesToVulnScan.length,
        },
      );

      // Trigger vuln scans with a small delay between each
      // Fixes #928: Track ALL timeout IDs to prevent orphaned recursive timeouts
      const timeoutIds: ReturnType<typeof setTimeout>[] = [];
      let index = 0;

      const triggerNext = (): void => {
        if (index >= devicesToVulnScan.length) {
          return;
        }
        const device = devicesToVulnScan[index];
        triggerVulnScan(device.ip, device).catch(() => {
          // Errors handled in triggerVulnScan
        });
        index++;
        if (index < devicesToVulnScan.length) {
          const tid = setTimeout(triggerNext, 200);
          timeoutIds.push(tid);
        }
      };

      const initialId = setTimeout(triggerNext, 300);
      timeoutIds.push(initialId);
      return (): void => {
        for (const tid of timeoutIds) {
          clearTimeout(tid);
        }
      };
    }, [
      data?.status,
      data?.devices,
      autoScanSettings.vulnScanEnabled,
      autoScanSettings.vulnAutoScan,
      triggerVulnScan,
    ]);

    // Extract data with safe defaults (must come before any hooks to avoid conditional hook calls)
    const rawDevices = data?.devices;
    const status = data?.status;
    // Ensure devices is an array (defensive check for malformed API responses)
    const devices = useMemo(() => (Array.isArray(rawDevices) ? rawDevices : []), [rawDevices]);
    const deviceCount = devices.length;

    // Helper function for IP to numeric conversion
    // Fixes #953: Handle malformed IPs that would produce NaN
    const ipToNum = useCallback((ip: string) => {
      const parts = ip.split(".").map((s) => Number.parseInt(s, 10) || 0);
      return parts[0] * 16777216 + parts[1] * 65536 + parts[2] * 256 + parts[3];
    }, []);

    // Filter and sort devices
    const filteredDevices = useMemo(() => {
      let result = [...devices];

      // Apply search filter
      if (searchQuery.trim()) {
        const query = searchQuery.toLowerCase();
        result = result.filter(
          (device) =>
            device.ip?.toLowerCase().includes(query) ||
            device.hostname?.toLowerCase().includes(query) ||
            device.vendor?.toLowerCase().includes(query) ||
            device.mac?.toLowerCase().includes(query) ||
            device.osGuess?.toLowerCase().includes(query),
        );
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
            default:
              break;
          }

          if (aVal === null && bVal === null) {
            return 0;
          }
          if (aVal === null) {
            return 1;
          }
          if (bVal === null) {
            return -1;
          }

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

    const _filteredCount = filteredDevices.length;

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
        while (!(resultA.done || resultB.done)) {
          if (resultA.value !== resultB.value) {
            return resultA.value - resultB.value;
          }
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
          icon={<ScanSearch class={iconTokens.size.md} />}
          status="loading"
          enableLiveRegion={true}
          ariaLabel="Network discovery scanning in progress"
        >
          <CardValue value={t("discovery.scanning")} size="lg" />
        </Card>
      );
    }

    if (!(data && status)) {
      return (
        <Card
          title={t("discovery.title")}
          icon={<ScanSearch class={iconTokens.size.md} />}
          status="unknown"
          enableLiveRegion={true}
          ariaLabel="Network discovery - no data available"
        >
          <CardValue value={t("discovery.noData")} size="md" />
          {onScan ? (
            <button
              type="button"
              onClick={onScan}
              class={cn(
                spacing.margin.top.heading,
                "w-full",
                button.size.md,
                "bg-brand-primary text-text-inverse",
                radius.md,
                "hover:bg-brand-primary/90 transition-colors font-medium body-small",
              )}
              aria-label="Start network discovery scan"
            >
              {t("discovery.startScan")}
            </button>
          ) : null}
        </Card>
      );
    }

    // Categorize devices for summary
    const categories = categorizeDevices(devices);

    const getOverallStatus = (): Status => {
      if (status.scanning || isPipelineRunning) {
        return "loading";
      }
      if (deviceCount === 0) {
        return "warning";
      }
      return "success";
    };

    const cardStatus = getOverallStatus();

    // Separate into local and extended for display (kept for modal use)
    const _localDevices = sortedDevices.filter((d) => d.isLocal);
    const _extendedDevices = sortedDevices.filter((d) => !d.isLocal);

    return (
      <Card
        title={t("discovery.title")}
        icon={<ScanSearch class={iconTokens.size.md} />}
        status={cardStatus}
        enableLiveRegion={true}
        ariaLabel={`Network discovery - ${deviceCount} devices found`}
        headerAction={
          <div class="flex items-center gap-2">
            {/* Full Screen button */}
            <button
              type="button"
              onClick={(): void => setIsModalOpen(true)}
              class={cn(
                "p-1.5",
                "bg-surface-hover text-text-secondary",
                radius.md,
                "hover:bg-surface-border hover:text-text-primary transition-colors flex items-center justify-center cursor-pointer",
              )}
              aria-label="Open full screen view"
              title={t("discovery.fullScreen", "Full Screen")}
            >
              <Maximize2 class={iconTokens.size.sm} aria-hidden="true" />
            </button>

            {/* Scan button */}
            {onScan || startPipeline ? (
              <button
                type="button"
                onClick={(): void => {
                  // Use pipeline start with port scanning enabled
                  // This enables the serviceDiscovery phase with quick port scan
                  startPipeline({
                    phases: {
                      enumeration: true,
                      nameResolution: true,
                      serviceDiscovery: true,
                      vulnAssessment: false,
                    },
                    portScan: {
                      intensity: "quick",
                      bannerGrab: true,
                      connectTimeout: 2000,
                    },
                  }).catch(() => {
                    // Errors handled in usePipelineStatus
                  });
                  // Also call onScan for backwards compatibility
                  onScan?.();
                }}
                disabled={status.scanning || isPipelineRunning}
                class={cn(
                  spacing.chip.sm,
                  "bg-brand-primary text-text-inverse",
                  radius.md,
                  "hover:bg-brand-primary/90 transition-colors font-medium caption disabled:opacity-50 disabled:cursor-not-allowed flex items-center",
                  spacing.inline.sm,
                )}
                aria-label={
                  status.scanning || isPipelineRunning ? "Scanning network" : "Start network scan"
                }
              >
                {status.scanning || isPipelineRunning ? (
                  <>
                    <RefreshCw class={cn(iconTokens.size.xs, "animate-spin")} aria-hidden="true" />
                    {t("discovery.scan")}
                  </>
                ) : (
                  t("discovery.scan")
                )}
              </button>
            ) : null}
          </div>
        }
      >
        {/* Discovery Summary - Minimal view showing status, subnet, device count, and categories */}
        <discoverySummary
          status={status}
          deviceCount={deviceCount}
          categories={categories}
          pipelineStatus={pipelineStatus}
          onCancelPipeline={cancelPipeline}
          t={t}
        />

        {deviceCount === 0 && !status.scanning && !isPipelineRunning ? (
          <p class={cn("body-small text-text-muted text-center", spacing.pad.default)}>
            {t("discovery.noDevices")}
          </p>
        ) : null}

        {/* Vulnerability Details Modal */}
        {selectedDeviceForVuln ? (
          <VulnerabilityDetailsModal
            deviceIp={selectedDeviceForVuln}
            onClose={(): void => setSelectedDeviceForVuln(null)}
          />
        ) : null}

        {/* Full Screen Discovery Modal */}
        <DiscoveryModal
          isOpen={isModalOpen}
          onClose={(): void => setIsModalOpen(false)}
          data={data}
          onScan={onScan}
          onDeepScan={handleDeepScan}
        />
      </Card>
    );
  },
);
