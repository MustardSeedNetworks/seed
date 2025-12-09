import { useState } from "react";
import { Card, CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { CollapsibleSection } from "../ui/CollapsibleSection";

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

function DeviceRow({
  device,
  isExpanded,
  onToggle,
}: {
  device: DiscoveredDevice;
  isExpanded: boolean;
  onToggle: () => void;
}) {
  const hasDetails = device.lldpInfo || device.cdpInfo || device.edpInfo;

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
            <span
              className={`text-lg transition-transform ${isExpanded ? "rotate-180" : ""}`}
            >
              {hasDetails ? "▼" : "○"}
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

export function NetworkDiscoveryCard({
  data,
  loading,
  onScan,
}: NetworkDiscoveryCardProps) {
  const [expandedDevices, setExpandedDevices] = useState<Set<string>>(
    new Set(),
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
}
