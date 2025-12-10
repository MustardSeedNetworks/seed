import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import {
  NetworkDiscoverySettings as NetworkDiscoverySettingsType,
  SubnetConfig,
  SaveStatus,
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
        <p className="text-xs text-text-muted">
          Configure ARP-based device discovery for finding devices on the local
          network.
        </p>

        {/* Scan Workers */}
        <div>
          <label className="text-xs text-text-muted font-medium">
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
        <div>
          <label className="text-xs text-text-muted font-medium">
            Ping Timeout (ms)
          </label>
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
          <p className="text-xs text-text-muted mt-1">
            Timeout per host ping (default: 500ms)
          </p>
        </div>

        {/* Scan Timeout */}
        <div>
          <label className="text-xs text-text-muted font-medium">
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
          <p className="text-xs text-text-muted mt-1">
            Max time for entire scan (default: 30s)
          </p>
        </div>

        {/* Scan Interval */}
        <div>
          <label className="text-xs text-text-muted font-medium">
            Auto-Scan Interval (ms)
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
            0 = disabled, otherwise interval between automatic scans
          </p>
        </div>

        {/* OUI File Path */}
        <div>
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

        {/* Additional Subnets */}
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
                    <div className="text-xs text-text-muted">{subnet.cidr}</div>
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
      </div>
    </CollapsibleSection>
  );
}
