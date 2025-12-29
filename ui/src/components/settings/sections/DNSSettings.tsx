/**
 * DnsSettings Component
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
 * <DnsSettings
 *   testsSettings={settings}
 *   setTestsSettings={updateSettings}
 *   testsStatus={saveStatus}
 * />
 * ```
 *
 * Dependencies: CollapsibleSection, AutoSaveIndicator, Globe icon, utilities for ID generation
 * State: Receives test settings and save status from parent, callbacks for updates
 */

import type React from "react";
import { memo, useCallback } from "react";
import { useTranslation } from "react-i18next";
import {
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
  spacing,
} from "../../../styles/theme";
import type { DnsServer, SaveStatus, TestsSettings } from "../../../types/settings";
import { generateId } from "../../../utils/id";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { Globe } from "../../ui/Icons";
import { AutoSaveIndicator } from "./AutoSaveIndicator";

interface DnsSettingsProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
  testsStatus: SaveStatus;
}

export const DnsSettings = memo(function DnsSettings({
  testsSettings,
  setTestsSettings,
  testsStatus,
}: DnsSettingsProps) {
  const { t } = useTranslation("settings");

  const addDnsServer = useCallback(() => {
    setTestsSettings((prev) => ({
      ...prev,
      dnsServers: [...prev.dnsServers, { id: generateId(), address: "", enabled: true }],
    }));
  }, [setTestsSettings]);

  const removeDnsServer = useCallback(
    (id: string) => {
      setTestsSettings((prev) => ({
        ...prev,
        dnsServers: prev.dnsServers.filter((s) => s.id !== id),
      }));
    },
    [setTestsSettings],
  );

  const updateDnsServer = useCallback(
    (id: string, field: keyof DnsServer, value: string | boolean) => {
      setTestsSettings((prev) => ({
        ...prev,
        dnsServers: prev.dnsServers.map((s) => (s.id === id ? { ...s, [field]: value } : s)),
      }));
    },
    [setTestsSettings],
  );

  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <Globe className={iconTokens.size.sm} />
          <span>{t("sections.dns")}</span>
          <AutoSaveIndicator status={testsStatus} />
        </div>
      }
    >
      <div className="stack">
        {/* DNS Hostname */}
        <div>
          <label htmlFor="dns-test-hostname" className="caption text-text-muted">
            {t("dns.testHostname")}
          </label>
          <input
            id="dns-test-hostname"
            type="text"
            value={testsSettings.dnsHostname}
            onChange={(e) =>
              setTestsSettings((prev) => ({
                ...prev,
                dnsHostname: e.target.value,
              }))
            }
            placeholder="google.com"
            className={cn(
              inputTokens.base,
              inputTokens.state.default,
              inputTokens.size.md,
              "w-full",
              spacing.margin.top.tight,
              "body-small",
            )}
          />
          <p className={cn("caption", "text-text-muted", spacing.margin.top.tight)}>
            {t("dns.testHostnameDesc")}
          </p>
        </div>

        {/* DNS Servers for per-server testing */}
        <div className={cn("border-t", "border-surface-border", spacing.padding.top.heading)}>
          <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
            <span className="caption text-text-muted font-medium">
              {t("dns.additionalServers")}
            </span>
            <button
              type="button"
              onClick={addDnsServer}
              className="caption text-brand-primary hover:text-brand-accent"
            >
              {t("common.add")}
            </button>
          </div>
          <p className={cn("caption", "text-text-muted", spacing.margin.bottom.inline)}>
            {t("dns.serversDescription")}
          </p>
          {testsSettings.dnsServers.map((server) => (
            <div
              key={server.id || server.address}
              className={cn("flex", spacing.gap.compact, spacing.margin.bottom.inline)}
            >
              <input
                type="text"
                value={server.address}
                onChange={(e) => updateDnsServer(server.id ?? "", "address", e.target.value)}
                placeholder={t("dns.serverIp")}
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.md,
                  "flex-1",
                  "caption",
                )}
              />
              <button
                type="button"
                onClick={() => removeDnsServer(server.id ?? "")}
                className={cn("text-status-error", "hover:text-status-error/80", spacing.actionBtn)}
                aria-label={t("common.remove")}
              >
                {t("common.remove")}
              </button>
            </div>
          ))}
        </div>
      </div>
    </CollapsibleSection>
  );
});
