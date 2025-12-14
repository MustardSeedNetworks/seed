/**
 * DNSSettings Component
 *
 * Purpose: Allows users to configure custom DNS servers for testing and specify
 * test hostnames and other DNS test parameters.
 *
 * Key Features:
 * - Multiple DNS servers: add/remove custom DNS server addresses
 * - Enable/disable per-server: toggle which servers to test
 * - Test hostname: configurable hostname for DNS resolution testing
 * - IPv6 support: separate options for IPv4 and IPv6 queries
 * - CRUD operations: add new servers, remove existing, update addresses
 * - AutoSaveIndicator: shows save status while persisting changes
 * - Globe icon: visual indicator in settings menu
 * - ID generation: unique IDs for server entries
 *
 * Usage:
 * ```typescript
 * <DNSSettings
 *   testsSettings={settings}
 *   setTestsSettings={updateSettings}
 *   testsStatus={saveStatus}
 * />
 * ```
 *
 * Dependencies: CollapsibleSection, AutoSaveIndicator, Globe icon, utilities for ID generation
 * State: Receives test settings and save status from parent, callbacks for updates
 */

import { memo, useCallback } from "react";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { Globe } from "../../ui/Icons";
import { TestsSettings, SaveStatus, DNSServer } from "../../../types/settings";
import { generateId } from "../../../utils/id";
import { icon as iconTokens, layout, radius } from "../../../styles/theme";

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
        <div className={layout.inline.default}>
          <Globe className={iconTokens.size.sm} />
          <span>DNS</span>
          <AutoSaveIndicator status={testsStatus} />
        </div>
      }
    >
      <div className="stack">
        {/* DNS Hostname */}
        <div>
          <label className="caption text-text-muted">Test Hostname</label>
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
            className={`w-full mt-1 px-2.5 py-2 bg-surface-base border border-surface-border ${radius.default} body-small text-text-primary`}
          />
          <p className="caption text-text-muted mt-1">
            Hostname used for DNS forward/reverse lookups
          </p>
        </div>

        {/* DNS Servers for per-server testing */}
        <div className="border-t border-surface-border pt-3">
          <div className={`${layout.flex.between} mb-2`}>
            <span className="caption text-text-muted font-medium">
              Additional DNS Servers
            </span>
            <button
              onClick={addDNSServer}
              className="caption text-brand-primary hover:text-brand-accent"
            >
              + Add
            </button>
          </div>
          <p className="caption text-text-muted mb-2">
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
                className={`flex-1 px-2.5 py-2 bg-surface-base border border-surface-border ${radius.default} caption text-text-primary`}
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
