import { memo, useCallback } from "react";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { Globe } from "../../ui/Icons";
import { TestsSettings, SaveStatus, DNSServer } from "../../../types/settings";
import { generateId } from "../../../utils/id";

interface DNSSettingsProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
  testsStatus: SaveStatus;
}

export const DNSSettings = memo(function DNSSettings({
  testsSettings,
  setTestsSettings,
  testsStatus,
}: DNSSettingsProps) {
  const addDNSServer = useCallback(() => {
    setTestsSettings((prev) => ({
      ...prev,
      dnsServers: [
        ...prev.dnsServers,
        { id: generateId(), address: "", enabled: true },
      ],
    }));
  }, [setTestsSettings]);

  const removeDNSServer = useCallback(
    (id: string) => {
      setTestsSettings((prev) => ({
        ...prev,
        dnsServers: prev.dnsServers.filter((s) => s.id !== id),
      }));
    },
    [setTestsSettings],
  );

  const updateDNSServer = useCallback(
    (id: string, field: keyof DNSServer, value: string | boolean) => {
      setTestsSettings((prev) => ({
        ...prev,
        dnsServers: prev.dnsServers.map((s) =>
          s.id === id ? { ...s, [field]: value } : s,
        ),
      }));
    },
    [setTestsSettings],
  );

  return (
    <CollapsibleSection
      title={
        <div className="flex items-center gap-2">
          <Globe className="w-4 h-4" />
          <span>DNS</span>
          <AutoSaveIndicator status={testsStatus} />
        </div>
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
          {testsSettings.dnsServers.map((server) => (
            <div key={server.id || server.address} className="flex gap-2 mb-2">
              <input
                type="text"
                value={server.address}
                onChange={(e) =>
                  updateDNSServer(server.id!, "address", e.target.value)
                }
                placeholder="DNS Server IP"
                className="flex-1 px-2.5 py-2 bg-surface-base border border-surface-border rounded text-xs text-text-primary"
              />
              <button
                onClick={() => removeDNSServer(server.id!)}
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
});
