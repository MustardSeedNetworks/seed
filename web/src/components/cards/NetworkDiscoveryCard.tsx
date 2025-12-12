import { useState, memo, useCallback } from "react";
import { Card, CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { CollapsibleSection } from "../ui/CollapsibleSection";
import { getAuthHeaders } from "../../hooks/useAuth";

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

export type DiscoveryMethod = "arp" | "lldp" | "cdp" | "edp" | "mdns" | "ping";

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
  mac: string;
  hostname?: string;
  vendor?: string;
  osGuess?: string;
  ttl?: number;
  discoveryMethod: DiscoveryMethod[];
  lastSeen: string;
  isLocal: boolean; // true if on local subnet, false for extended networks
  lldpInfo?: LLDPInfo;
  cdpInfo?: CDPInfo;
  edpInfo?: EDPInfo;
  profile?: DeviceProfile;
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

function MethodBadge({ method }: { method: DiscoveryMethod }) {
  const colors: Record<DiscoveryMethod, string> = {
    arp: "bg-blue-500/20 text-blue-400",
    lldp: "bg-green-500/20 text-green-400",
    cdp: "bg-orange-500/20 text-orange-400",
    edp: "bg-purple-500/20 text-purple-400",
    mdns: "bg-teal-500/20 text-teal-400",
    ping: "bg-cyan-500/20 text-cyan-400",
  };

  return (
    <span
      className={`px-1.5 py-0.5 rounded text-xs font-medium uppercase ${colors[method]}`}
    >
      {method}
    </span>
  );
}

// Icon mapping for device profile icons
const ICON_SYMBOLS: Record<string, string> = {
  ssh: "S",
  telnet: "T",
  web: "W",
  "web-secure": "Ws",
  ftp: "F",
  mail: "M",
  dns: "D",
  snmp: "N",
  database: "DB",
  cache: "C",
  printer: "P",
  router: "R",
  switch: "Sw",
  firewall: "Fw",
  storage: "St",
  server: "Sv",
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
      {icons.slice(0, 5).map((icon) => (
        <span
          key={icon}
          className="px-1 py-0.5 rounded text-[10px] font-medium bg-indigo-500/20 text-indigo-400"
          title={`${icon}${deviceType ? ` (${deviceType})` : ""}`}
        >
          {ICON_SYMBOLS[icon] || icon[0]?.toUpperCase()}
        </span>
      ))}
      {icons.length > 5 && (
        <span className="text-[10px] text-text-muted">+{icons.length - 5}</span>
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
}: {
  device: DiscoveredDevice;
  isExpanded: boolean;
  onToggle: () => void;
  onDeepScan?: (ip: string) => void;
  isScanning?: boolean;
  scanResult?: DeepScanResult;
}) {
  const hasDetails =
    device.lldpInfo || device.cdpInfo || device.edpInfo || device.profile;
  const openPorts = scanResult?.results.filter((r) => r.state === "open") || [];
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
              {device.hostname && (
                <span
                  className="text-xs text-text-muted truncate max-w-[120px]"
                  title={device.hostname}
                >
                  ({device.hostname})
                </span>
              )}
              {openPorts.length > 0 && (
                <span className="text-xs bg-green-500/20 text-green-400 px-1.5 py-0.5 rounded">
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
                className="px-2 py-1 text-xs bg-blue-500/20 text-blue-400 rounded hover:bg-blue-500/30 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
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
                        <span className="text-green-400">
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
                          className="px-1.5 py-0.5 rounded text-[10px] bg-green-500/20 text-green-400"
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

  if (loading) {
    return (
      <Card title="Network Discovery" status="loading">
        <CardValue value="Scanning..." size="lg" />
      </Card>
    );
  }

  if (!data) {
    return (
      <Card title="Network Discovery" status="unknown">
        <CardValue value="No data" size="md" />
        {onScan && (
          <button
            type="button"
            onClick={onScan}
            className="mt-3 w-full py-2 px-4 bg-brand-primary text-white rounded-lg hover:bg-brand-primary/90 transition-colors font-medium text-sm"
          >
            Start Scan
          </button>
        )}
      </Card>
    );
  }

  const { devices: rawDevices, status } = data;
  // Ensure devices is an array (defensive check for malformed API responses)
  const devices = Array.isArray(rawDevices) ? rawDevices : [];
  const deviceCount = devices.length;

  const getOverallStatus = (): Status => {
    if (status.scanning) return "loading";
    if (deviceCount === 0) return "warning";
    return "success";
  };

  const cardStatus = getOverallStatus();

  // Sort devices: local first, then by discovery methods, then by IP
  const sortedDevices = [...devices].sort((a, b) => {
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

  // Separate into local and extended for display
  const localDevices = sortedDevices.filter((d) => d.isLocal);
  const extendedDevices = sortedDevices.filter((d) => !d.isLocal);

  return (
    <Card title="Network Discovery" status={cardStatus}>
      <div className="flex items-center justify-between gap-2">
        <CardValue
          value={`${deviceCount} device${deviceCount !== 1 ? "s" : ""}`}
          size="lg"
        />
        {onScan && (
          <button
            type="button"
            onClick={onScan}
            disabled={status.scanning}
            className="py-1.5 px-3 bg-brand-primary text-white rounded-lg hover:bg-brand-primary/90 transition-colors font-medium text-sm disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5"
          >
            {status.scanning ? (
              <>
                <span className="animate-spin">◐</span>
                Scanning...
              </>
            ) : (
              "Scan"
            )}
          </button>
        )}
      </div>

      <CardDivider />

      {/* Network Info - Collapsible */}
      <CollapsibleSection
        title="Network Info"
        variant="compact"
        defaultOpen={false}
      >
        <div className="space-y-1 text-xs">
          {status.subnet && <CardRow label="Subnet" value={status.subnet} />}
          {status.localIP && (
            <CardRow label="Local IP" value={status.localIP} />
          )}
          {status.interface && (
            <CardRow label="Interface" value={status.interface} />
          )}
          <CardRow label="Last Scan" value={formatLastSeen(status.lastScan)} />
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
    </Card>
  );
});
