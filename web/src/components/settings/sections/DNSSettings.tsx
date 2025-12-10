import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { TestsSettings, SaveStatus } from "../../../types/settings";

interface DNSSettingsProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
  testsStatus: SaveStatus;
}

export function DNSSettings({
  testsSettings,
  setTestsSettings,
  testsStatus,
}: DNSSettingsProps) {
  const addDNSServer = () => {
    setTestsSettings((prev) => ({
      ...prev,
      dnsServers: [...prev.dnsServers, { address: "", enabled: true }],
    }));
  };

  const removeDNSServer = (index: number) => {
    setTestsSettings((prev) => ({
      ...prev,
      dnsServers: prev.dnsServers.filter((_, i) => i !== index),
    }));
  };

  const updateDNSServer = (
    index: number,
    field: "address" | "enabled",
    value: string | boolean,
  ) => {
    setTestsSettings((prev) => ({
      ...prev,
      dnsServers: prev.dnsServers.map((s, i) =>
        i === index ? { ...s, [field]: value } : s,
      ),
    }));
  };

  return (
    <CollapsibleSection
      title={
        <>
          DNS
          <AutoSaveIndicator status={testsStatus} />
        </>
      }
    >
      <div className="space-y-4">
        {/* DNS Hostname */}
        <div>
          <label className="text-xs text-text-muted">Test Hostname</label>
          <input
            type="text"
            value={testsSettings.dnsHostname}
            onChange={(e) =>
              setTestsSettings((prev) => ({
                ...prev,
                dnsHostname: e.target.value,
              }))
            }
            placeholder="google.com"
            className="w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-sm text-text-primary"
          />
          <p className="text-xs text-text-muted mt-1">
            Hostname used for DNS forward/reverse lookups
          </p>
        </div>

        {/* DNS Servers for per-server testing */}
        <div className="border-t border-surface-border pt-3">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-text-muted font-medium">
              Additional DNS Servers
            </span>
            <button
              onClick={addDNSServer}
              className="text-xs text-brand-primary hover:text-brand-accent"
            >
              + Add
            </button>
          </div>
          <p className="text-xs text-text-muted mb-2">
            Add servers to compare DNS response times (e.g., 8.8.8.8, 1.1.1.1)
          </p>
          {testsSettings.dnsServers.map((server, idx) => (
            <div key={idx} className="flex gap-2 mb-2">
              <input
                type="text"
                value={server.address}
                onChange={(e) =>
                  updateDNSServer(idx, "address", e.target.value)
                }
                placeholder="DNS Server IP"
                className="flex-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <button
                onClick={() => removeDNSServer(idx)}
                className="text-status-error hover:text-status-error/80 px-1"
              >
                x
              </button>
            </div>
          ))}
        </div>
      </div>
    </CollapsibleSection>
  );
}
