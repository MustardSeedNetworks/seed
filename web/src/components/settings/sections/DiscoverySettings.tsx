import { useState, useEffect, useCallback } from "react";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import {
  NetworkDiscoverySettings as NetworkDiscoverySettingsType,
  SubnetConfig,
  SaveStatus,
  DiscoveryProfile,
  DiscoveryServiceStatus,
} from "../../../types/settings";

interface DiscoverySettingsProps {
  networkDiscoverySettings: NetworkDiscoverySettingsType;
  setNetworkDiscoverySettings: React.Dispatch<
    React.SetStateAction<NetworkDiscoverySettingsType>
  >;
  networkDiscoveryStatus: SaveStatus;
  subnets: SubnetConfig[];
  subnetsStatus: SaveStatus;
  newSubnetCidr: string;
  setNewSubnetCidr: React.Dispatch<React.SetStateAction<string>>;
  newSubnetName: string;
  setNewSubnetName: React.Dispatch<React.SetStateAction<string>>;
  subnetError: string | null;
  setSubnetError: React.Dispatch<React.SetStateAction<string | null>>;
  addSubnet: () => void;
  toggleSubnet: (cidr: string, enabled: boolean) => void;
  deleteSubnet: (cidr: string) => void;
}

const PROFILE_DESCRIPTIONS: Record<DiscoveryProfile, string> = {
  stealth: "Passive only - LLDP/CDP/EDP listeners, no active scanning",
  standard: "Passive + ARP/ICMP scanning on local subnet",
  full_scan: "All methods including port scanning and additional subnets",
  custom: "Fine-grained control over individual discovery methods",
};

const PROFILE_LABELS: Record<DiscoveryProfile, string> = {
  stealth: "Stealth",
  standard: "Standard",
  full_scan: "Full Scan",
  custom: "Custom",
};

export function DiscoverySettings({
  networkDiscoverySettings,
  setNetworkDiscoverySettings,
  networkDiscoveryStatus,
  subnets,
  subnetsStatus,
  newSubnetCidr,
  setNewSubnetCidr,
  newSubnetName,
  setNewSubnetName,
  subnetError,
  setSubnetError,
  addSubnet,
  toggleSubnet,
  deleteSubnet,
}: DiscoverySettingsProps) {
  const [serviceStatus, setServiceStatus] =
    useState<DiscoveryServiceStatus | null>(null);
  const [statusLoading, setStatusLoading] = useState(false);

  // Fetch service status
  const fetchServiceStatus = useCallback(async () => {
    setStatusLoading(true);
    try {
      const response = await fetch("/api/discovery/service/status");
      if (response.ok) {
        const data = await response.json();
        setServiceStatus(data);
      }
    } catch {
      // Silently fail - status display is informational
    } finally {
      setStatusLoading(false);
    }
  }, []);

  // Fetch status on mount and periodically
  useEffect(() => {
    fetchServiceStatus();
    const interval = setInterval(fetchServiceStatus, 10000);
    return () => clearInterval(interval);
  }, [fetchServiceStatus]);

  // Update profile via API
  const handleProfileChange = async (profile: DiscoveryProfile) => {
    setNetworkDiscoverySettings((prev) => ({
      ...prev,
      profile,
    }));

    try {
      await fetch("/api/discovery/profile", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ profile }),
      });
      // Refresh status after profile change
      setTimeout(fetchServiceStatus, 500);
    } catch {
      // Settings auto-save will handle persistence
    }
  };

  const currentProfile = networkDiscoverySettings.profile || "standard";
  const showCustomOptions = currentProfile === "custom";
  const showSubnets =
    currentProfile === "full_scan" || currentProfile === "custom";

  return (
    <CollapsibleSection
      title={
        <>
          Network Discovery
          <AutoSaveIndicator status={networkDiscoveryStatus} />
        </>
      }
    >
      <div className="space-y-4">
        {/* Service Status Banner */}
        {serviceStatus && (
          <div
            className={`p-3 rounded-lg border ${
              serviceStatus.running
                ? "bg-status-success/10 border-status-success/30"
                : "bg-status-error/10 border-status-error/30"
            }`}
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div
                  className={`w-2 h-2 rounded-full ${
                    serviceStatus.running
                      ? serviceStatus.scanning
                        ? "bg-status-warning animate-pulse"
                        : "bg-status-success"
                      : "bg-status-error"
                  }`}
                />
                <span className="text-sm font-medium text-text-primary">
                  {serviceStatus.running
                    ? serviceStatus.scanning
                      ? "Scanning..."
                      : "Running"
                    : "Stopped"}
                </span>
              </div>
              <button
                onClick={fetchServiceStatus}
                disabled={statusLoading}
                className="text-xs text-text-muted hover:text-text-primary"
              >
                {statusLoading ? "..." : "Refresh"}
              </button>
            </div>
            {serviceStatus.running && (
              <div className="mt-2 grid grid-cols-2 gap-2 text-xs text-text-muted">
                <div>
                  <span className="font-medium">Devices:</span>{" "}
                  {serviceStatus.deviceCount}
                </div>
                <div>
                  <span className="font-medium">Interface:</span>{" "}
                  {serviceStatus.interface || "auto"}
                </div>
                <div>
                  <span className="font-medium">Subnet:</span>{" "}
                  {serviceStatus.subnet || "detecting..."}
                </div>
                <div>
                  <span className="font-medium">Local IP:</span>{" "}
                  {serviceStatus.localIP || "detecting..."}
                </div>
              </div>
            )}
            {serviceStatus.activeMethods &&
              serviceStatus.activeMethods.length > 0 && (
                <div className="mt-2 flex flex-wrap gap-1">
                  {serviceStatus.activeMethods.map((method) => (
                    <span
                      key={method}
                      className="px-1.5 py-0.5 bg-surface-base rounded text-xs text-text-muted"
                    >
                      {method}
                    </span>
                  ))}
                </div>
              )}
          </div>
        )}

        {/* Discovery Profile Selector */}
        <div>
          <label className="text-xs text-text-muted font-medium">
            Discovery Profile
          </label>
          <div className="mt-2 space-y-2">
            {(
              [
                "stealth",
                "standard",
                "full_scan",
                "custom",
              ] as DiscoveryProfile[]
            ).map((profile) => (
              <label
                key={profile}
                className={`flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-colors ${
                  currentProfile === profile
                    ? "border-brand-primary bg-brand-primary/5"
                    : "border-surface-border hover:border-brand-primary/50"
                }`}
              >
                <input
                  type="radio"
                  name="discovery-profile"
                  value={profile}
                  checked={currentProfile === profile}
                  onChange={() => handleProfileChange(profile)}
                  className="mt-0.5"
                />
                <div className="flex-1">
                  <div className="text-sm font-medium text-text-primary">
                    {PROFILE_LABELS[profile]}
                  </div>
                  <div className="text-xs text-text-muted mt-0.5">
                    {PROFILE_DESCRIPTIONS[profile]}
                  </div>
                </div>
              </label>
            ))}
          </div>
        </div>

        {/* Custom Options (only shown when Custom profile is selected) */}
        {showCustomOptions && (
          <div className="border-t border-surface-border pt-3">
            <span className="text-xs text-text-muted font-medium">
              Custom Discovery Options
            </span>
            <div className="mt-2 space-y-2">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={
                    networkDiscoverySettings.customOptions?.passiveListen ??
                    true
                  }
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        passiveListen: e.target.checked,
                      },
                    }))
                  }
                  className="w-4 h-4"
                />
                <span className="text-sm text-text-primary">
                  Passive Protocol Listeners (LLDP, CDP, EDP)
                </span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={
                    networkDiscoverySettings.customOptions?.arpScan ?? true
                  }
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        arpScan: e.target.checked,
                      },
                    }))
                  }
                  className="w-4 h-4"
                />
                <span className="text-sm text-text-primary">ARP Scanning</span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={
                    networkDiscoverySettings.customOptions?.icmpScan ?? true
                  }
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        icmpScan: e.target.checked,
                      },
                    }))
                  }
                  className="w-4 h-4"
                />
                <span className="text-sm text-text-primary">
                  ICMP Ping Sweep
                </span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={
                    networkDiscoverySettings.customOptions?.portScan?.enabled ??
                    false
                  }
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        portScan: {
                          ...prev.customOptions?.portScan,
                          enabled: e.target.checked,
                          ports: prev.customOptions?.portScan?.ports ?? [],
                          topPorts:
                            prev.customOptions?.portScan?.topPorts ?? 100,
                        },
                      },
                    }))
                  }
                  className="w-4 h-4"
                />
                <span className="text-sm text-text-primary">Port Scanning</span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={
                    networkDiscoverySettings.customOptions?.traceroute ?? false
                  }
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        traceroute: e.target.checked,
                      },
                    }))
                  }
                  className="w-4 h-4"
                />
                <span className="text-sm text-text-primary">Traceroute</span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={
                    networkDiscoverySettings.customOptions?.snmpQuery ?? false
                  }
                  onChange={(e) =>
                    setNetworkDiscoverySettings((prev) => ({
                      ...prev,
                      customOptions: {
                        ...prev.customOptions,
                        snmpQuery: e.target.checked,
                      },
                    }))
                  }
                  className="w-4 h-4"
                />
                <span className="text-sm text-text-primary">SNMP Queries</span>
              </label>
            </div>
          </div>
        )}

        {/* Timing Settings */}
        <div className="border-t border-surface-border pt-3">
          <span className="text-xs text-text-muted font-medium">
            Timing Settings
          </span>

          {/* Scan Workers */}
          <div className="mt-2">
            <label className="text-xs text-text-muted">
              Concurrent Scan Workers
            </label>
            <input
              type="number"
              value={networkDiscoverySettings.arpScanWorkers}
              onChange={(e) =>
                setNetworkDiscoverySettings((prev) => ({
                  ...prev,
                  arpScanWorkers: parseInt(e.target.value) || 50,
                }))
              }
              min={1}
              max={100}
              className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
            />
            <p className="text-xs text-text-muted mt-1">
              More workers = faster scan (default: 50)
            </p>
          </div>

          {/* Ping Timeout */}
          <div className="mt-3">
            <label className="text-xs text-text-muted">Ping Timeout (ms)</label>
            <input
              type="number"
              value={networkDiscoverySettings.pingTimeoutMs}
              onChange={(e) =>
                setNetworkDiscoverySettings((prev) => ({
                  ...prev,
                  pingTimeoutMs: parseInt(e.target.value) || 500,
                }))
              }
              min={100}
              max={5000}
              className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
            />
          </div>

          {/* Scan Timeout */}
          <div className="mt-3">
            <label className="text-xs text-text-muted">
              Total Scan Timeout (ms)
            </label>
            <input
              type="number"
              value={networkDiscoverySettings.scanTimeoutMs}
              onChange={(e) =>
                setNetworkDiscoverySettings((prev) => ({
                  ...prev,
                  scanTimeoutMs: parseInt(e.target.value) || 30000,
                }))
              }
              min={5000}
              max={120000}
              className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
            />
          </div>

          {/* Rescan Interval */}
          <div className="mt-3">
            <label className="text-xs text-text-muted">
              Auto-Rescan Interval (ms)
            </label>
            <input
              type="number"
              value={networkDiscoverySettings.scanIntervalMs}
              onChange={(e) =>
                setNetworkDiscoverySettings((prev) => ({
                  ...prev,
                  scanIntervalMs: parseInt(e.target.value) || 0,
                }))
              }
              min={0}
              className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
            />
            <p className="text-xs text-text-muted mt-1">
              0 = disabled, otherwise interval between automatic rescans
            </p>
          </div>
        </div>

        {/* OUI File Path */}
        <div className="border-t border-surface-border pt-3">
          <label className="text-xs text-text-muted font-medium">
            OUI Database File Path
          </label>
          <input
            type="text"
            value={networkDiscoverySettings.ouiFilePath}
            onChange={(e) =>
              setNetworkDiscoverySettings((prev) => ({
                ...prev,
                ouiFilePath: e.target.value,
              }))
            }
            placeholder="oui.txt"
            className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
          />
          <p className="text-xs text-text-muted mt-1">
            Path to IEEE OUI file for vendor lookup (download from{" "}
            <a
              href="https://standards-oui.ieee.org/oui/oui.txt"
              target="_blank"
              rel="noopener noreferrer"
              className="text-brand-primary hover:underline"
            >
              IEEE
            </a>
            )
          </p>
        </div>

        {/* Additional Subnets (only for full_scan or custom profile) */}
        {showSubnets && (
          <div className="border-t border-surface-border pt-3">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs text-text-muted font-medium">
                Additional Subnets <AutoSaveIndicator status={subnetsStatus} />
              </span>
            </div>
            <p className="text-xs text-text-muted mb-2">
              Add subnets beyond the local interface to scan for devices (e.g.,
              server VLANs, remote networks).
            </p>

            {/* List of configured subnets */}
            {subnets.length > 0 && (
              <div className="space-y-2 mb-3">
                {subnets.map((subnet) => (
                  <div
                    key={subnet.cidr}
                    className="flex items-center justify-between p-2 bg-surface-base rounded border border-surface-border"
                  >
                    <div className="flex-1 min-w-0">
                      <div className="text-sm text-text-primary truncate">
                        {subnet.name || subnet.cidr}
                      </div>
                      <div className="text-xs text-text-muted">
                        {subnet.cidr}
                      </div>
                    </div>
                    <div className="flex items-center gap-2 ml-2">
                      <input
                        type="checkbox"
                        checked={subnet.enabled}
                        onChange={(e) =>
                          toggleSubnet(subnet.cidr, e.target.checked)
                        }
                        className="w-4 h-4"
                        title={
                          subnet.enabled ? "Disable subnet" : "Enable subnet"
                        }
                      />
                      <button
                        onClick={() => deleteSubnet(subnet.cidr)}
                        className="text-status-error hover:text-red-400 text-sm"
                        title="Remove subnet"
                      >
                        X
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* Add new subnet form */}
            <div className="space-y-2">
              <input
                type="text"
                value={newSubnetCidr}
                onChange={(e) => {
                  setNewSubnetCidr(e.target.value);
                  setSubnetError(null);
                }}
                placeholder="CIDR (e.g., 10.0.0.0/24)"
                className="w-full px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
              />
              <input
                type="text"
                value={newSubnetName}
                onChange={(e) => setNewSubnetName(e.target.value)}
                placeholder="Name (optional, e.g., Server VLAN)"
                className="w-full px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
              />
              {subnetError && (
                <p className="text-xs text-status-error">{subnetError}</p>
              )}
              <button
                onClick={addSubnet}
                className="w-full px-3 py-2 bg-brand-primary hover:bg-brand-accent text-white rounded text-sm"
              >
                + Add Subnet
              </button>
            </div>
          </div>
        )}
      </div>
    </CollapsibleSection>
  );
}
