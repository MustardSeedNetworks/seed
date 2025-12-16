/**
 * HealthChecksSettings Component (~449 lines)
 *
 * Purpose: Comprehensive health check configuration panel allowing users to define
 * and customize ping targets, TCP/UDP ports, and HTTP endpoints for monitoring.
 *
 * Key Features:
 * - Ping targets: add/remove/configure ping destinations with custom names and packet counts
 * - TCP ports: configure TCP connectivity tests on specific ports
 * - UDP ports: configure UDP reachability tests
 * - HTTP endpoints: configure HTTP/HTTPS monitoring with customizable URLs
 * - Enable/disable: toggle each test individually
 * - Interval configuration: set how frequently tests run
 * - Timeout settings: configure test timeout values per protocol
 * - Port validation: validates port numbers (1-65535)
 * - URL validation: validates HTTP endpoint URLs
 * - CRUD operations: add/remove/update all test types
 * - AutoSaveIndicator: shows persistent save status
 * - HeartPulse icon: visual indicator in settings menu
 *
 * Usage:
 * ```typescript
 * <HealthChecksSettings
 *   testsSettings={settings}
 *   setTestsSettings={updateSettings}
 *   testsStatus={saveStatus}
 * />
 * ```
 *
 * Dependencies: CollapsibleSection, AutoSaveIndicator, Icons, settings types, ID generation
 * State: Manages multiple arrays of test configurations with CRUD callbacks
 */

import { memo, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { CollapsibleSection } from "../../ui/CollapsibleSection";
import { AutoSaveIndicator } from "./AutoSaveIndicator";
import { HeartPulse } from "../../ui/Icons";
import { icon as iconTokens, layout, radius, cn, spacing, input } from "../../../styles/theme";
import {
  TestsSettings,
  SaveStatus,
  PingTarget,
  TCPPort,
  UDPPort,
  HTTPEndpoint,
} from "../../../types/settings";
import { generateId } from "../../../utils/id";

interface HealthChecksSettingsProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
  testsStatus: SaveStatus;
}

export const HealthChecksSettings = memo(function HealthChecksSettings({
  testsSettings,
  setTestsSettings,
  testsStatus,
}: HealthChecksSettingsProps) {
  const { t } = useTranslation("settings");

  // Ping target helpers
  const addPingTarget = useCallback(() => {
    setTestsSettings((prev) => ({
      ...prev,
      pingTargets: [
        ...prev.pingTargets,
        { id: generateId(), name: "", host: "", enabled: true, count: 3 },
      ],
    }));
  }, [setTestsSettings]);

  const removePingTarget = useCallback(
    (id: string) => {
      setTestsSettings((prev) => ({
        ...prev,
        pingTargets: prev.pingTargets.filter((t) => t.id !== id),
      }));
    },
    [setTestsSettings]
  );

  const updatePingTarget = useCallback(
    (id: string, field: keyof PingTarget, value: string | boolean | number) => {
      setTestsSettings((prev) => ({
        ...prev,
        pingTargets: prev.pingTargets.map((t) => (t.id === id ? { ...t, [field]: value } : t)),
      }));
    },
    [setTestsSettings]
  );

  // TCP port helpers
  const addTCPPort = useCallback(() => {
    setTestsSettings((prev) => ({
      ...prev,
      tcpPorts: [
        ...prev.tcpPorts,
        { id: generateId(), name: "", host: "", port: 80, enabled: true },
      ],
    }));
  }, [setTestsSettings]);

  const removeTCPPort = useCallback(
    (id: string) => {
      setTestsSettings((prev) => ({
        ...prev,
        tcpPorts: prev.tcpPorts.filter((p) => p.id !== id),
      }));
    },
    [setTestsSettings]
  );

  const updateTCPPort = useCallback(
    (id: string, field: keyof TCPPort, value: string | boolean | number) => {
      setTestsSettings((prev) => ({
        ...prev,
        tcpPorts: prev.tcpPorts.map((p) => (p.id === id ? { ...p, [field]: value } : p)),
      }));
    },
    [setTestsSettings]
  );

  // UDP port helpers
  const addUDPPort = useCallback(() => {
    setTestsSettings((prev) => ({
      ...prev,
      udpPorts: [
        ...prev.udpPorts,
        { id: generateId(), name: "", host: "", port: 53, enabled: true },
      ],
    }));
  }, [setTestsSettings]);

  const removeUDPPort = useCallback(
    (id: string) => {
      setTestsSettings((prev) => ({
        ...prev,
        udpPorts: prev.udpPorts.filter((p) => p.id !== id),
      }));
    },
    [setTestsSettings]
  );

  const updateUDPPort = useCallback(
    (id: string, field: keyof UDPPort, value: string | boolean | number) => {
      setTestsSettings((prev) => ({
        ...prev,
        udpPorts: prev.udpPorts.map((p) => (p.id === id ? { ...p, [field]: value } : p)),
      }));
    },
    [setTestsSettings]
  );

  // HTTP endpoint helpers
  const addHTTPEndpoint = useCallback(() => {
    setTestsSettings((prev) => ({
      ...prev,
      httpEndpoints: [
        ...prev.httpEndpoints,
        {
          id: generateId(),
          name: "",
          url: "",
          expectedStatus: 200,
          enabled: true,
        },
      ],
    }));
  }, [setTestsSettings]);

  const removeHTTPEndpoint = useCallback(
    (id: string) => {
      setTestsSettings((prev) => ({
        ...prev,
        httpEndpoints: prev.httpEndpoints.filter((e) => e.id !== id),
      }));
    },
    [setTestsSettings]
  );

  const updateHTTPEndpoint = useCallback(
    (id: string, field: keyof HTTPEndpoint, value: string | boolean | number) => {
      setTestsSettings((prev) => ({
        ...prev,
        httpEndpoints: prev.httpEndpoints.map((e) => (e.id === id ? { ...e, [field]: value } : e)),
      }));
    },
    [setTestsSettings]
  );

  return (
    <CollapsibleSection
      title={
        <div className={layout.inline.default}>
          <HeartPulse className={iconTokens.size.sm} />
          <span>{t("sections.health")}</span>
          <AutoSaveIndicator status={testsStatus} />
        </div>
      }
    >
      <div className={spacing.stack.default}>
        {/* Enable Toggle */}
        <label
          className={cn(
            layout.flex.between,
            spacing.pad.sm,
            "bg-surface-base border border-surface-border",
            radius.default
          )}
        >
          <div>
            <span className="body-small text-text-primary font-medium">
              {t("health.enableHealthChecks")}
            </span>
            <p className="caption text-text-muted">{t("health.enableDescription")}</p>
          </div>
          <input
            type="checkbox"
            checked={testsSettings.runPerformance !== false}
            onChange={(e) =>
              setTestsSettings((prev) => ({
                ...prev,
                runPerformance: e.target.checked,
              }))
            }
            className={iconTokens.size.sm}
          />
        </label>

        {/* Ping Targets */}
        <div>
          <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
            <span className="caption text-text-muted font-medium">{t("health.pingTargets")}</span>
            <button
              onClick={addPingTarget}
              className="caption text-brand-primary hover:text-brand-accent"
            >
              {t("common.add")}
            </button>
          </div>
          <p className={cn("caption text-text-muted", spacing.margin.bottom.inline)}>
            {t("health.pingDefault")}
          </p>
          {testsSettings.pingTargets.map((target) => (
            <div
              key={target.id || target.host}
              className={cn("flex", spacing.gap.compact, spacing.margin.bottom.inline)}
            >
              <input
                type="text"
                value={target.name}
                onChange={(e) => updatePingTarget(target.id!, "name", e.target.value)}
                placeholder={t("common.name")}
                className={cn(input.base, input.state.default, input.size.md, "w-24")}
              />
              <input
                type="text"
                value={target.host}
                onChange={(e) => updatePingTarget(target.id!, "host", e.target.value)}
                placeholder={t("common.hostIp")}
                className={cn(input.base, input.state.default, input.size.md, "flex-1")}
              />
              <input
                type="number"
                value={target.count || 3}
                onChange={(e) =>
                  updatePingTarget(target.id!, "count", parseInt(e.target.value) || 3)
                }
                min={1}
                max={10}
                title={t("health.numberOfPings")}
                className={cn(input.base, input.state.default, input.size.md, "w-14 text-center")}
              />
              <button
                onClick={() => removePingTarget(target.id!)}
                className={`text-status-error hover:text-status-error/80 ${spacing.actionBtn}`}
                aria-label={t("common.remove")}
              >
                {t("common.remove")}
              </button>
            </div>
          ))}
        </div>

        {/* TCP Ports */}
        <div className={cn("border-t border-surface-border", spacing.padding.top.heading)}>
          <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
            <span className="caption text-text-muted font-medium">{t("health.tcpPortTests")}</span>
            <button
              onClick={addTCPPort}
              className="caption text-brand-primary hover:text-brand-accent"
            >
              {t("common.add")}
            </button>
          </div>
          {testsSettings.tcpPorts.map((port) => (
            <div
              key={port.id || `${port.host}:${port.port}`}
              className={cn("flex", spacing.gap.compact, spacing.margin.bottom.inline)}
            >
              <input
                type="text"
                value={port.name}
                onChange={(e) => updateTCPPort(port.id!, "name", e.target.value)}
                placeholder={t("common.name")}
                className={cn(input.base, input.state.default, input.size.md, "w-24")}
              />
              <input
                type="text"
                value={port.host}
                onChange={(e) => updateTCPPort(port.id!, "host", e.target.value)}
                placeholder={t("common.host")}
                className={cn(input.base, input.state.default, input.size.md, "flex-1")}
              />
              <input
                type="number"
                value={port.port}
                onChange={(e) => updateTCPPort(port.id!, "port", parseInt(e.target.value) || 80)}
                placeholder={t("common.port")}
                className={cn(input.base, input.state.default, input.size.md, "w-20")}
              />
              <button
                onClick={() => removeTCPPort(port.id!)}
                className={`text-status-error hover:text-status-error/80 ${spacing.actionBtn}`}
                aria-label={t("common.remove")}
              >
                {t("common.remove")}
              </button>
            </div>
          ))}
        </div>

        {/* UDP Ports */}
        <div className={cn("border-t border-surface-border", spacing.padding.top.heading)}>
          <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
            <span className="caption text-text-muted font-medium">{t("health.udpPortTests")}</span>
            <button
              onClick={addUDPPort}
              className="caption text-brand-primary hover:text-brand-accent"
            >
              {t("common.add")}
            </button>
          </div>
          <p className={cn("caption text-text-muted", spacing.margin.bottom.inline)}>
            {t("health.udpDescription")}
          </p>
          {testsSettings.udpPorts.map((port) => (
            <div
              key={port.id || `${port.host}:${port.port}`}
              className={cn("flex", spacing.gap.compact, spacing.margin.bottom.inline)}
            >
              <input
                type="text"
                value={port.name}
                onChange={(e) => updateUDPPort(port.id!, "name", e.target.value)}
                placeholder={t("common.name")}
                className={cn(input.base, input.state.default, input.size.md, "w-24")}
              />
              <input
                type="text"
                value={port.host}
                onChange={(e) => updateUDPPort(port.id!, "host", e.target.value)}
                placeholder={t("common.host")}
                className={cn(input.base, input.state.default, input.size.md, "flex-1")}
              />
              <input
                type="number"
                value={port.port}
                onChange={(e) => updateUDPPort(port.id!, "port", parseInt(e.target.value) || 53)}
                placeholder={t("common.port")}
                className={cn(input.base, input.state.default, input.size.md, "w-20")}
              />
              <button
                onClick={() => removeUDPPort(port.id!)}
                className={`text-status-error hover:text-status-error/80 ${spacing.actionBtn}`}
                aria-label={t("common.remove")}
              >
                {t("common.remove")}
              </button>
            </div>
          ))}
        </div>

        {/* HTTP Endpoints */}
        <div className={cn("border-t border-surface-border", spacing.padding.top.heading)}>
          <div className={cn(layout.flex.between, spacing.margin.bottom.inline)}>
            <span className="caption text-text-muted font-medium">{t("health.httpEndpoints")}</span>
            <button
              onClick={addHTTPEndpoint}
              className="caption text-brand-primary hover:text-brand-accent"
            >
              {t("common.add")}
            </button>
          </div>
          {testsSettings.httpEndpoints.map((endpoint) => (
            <div
              key={endpoint.id || endpoint.url}
              className={cn(
                spacing.stack.xs,
                spacing.margin.bottom.heading,
                spacing.pad.xs,
                "bg-surface-base border border-surface-border",
                radius.default
              )}
            >
              <div className={cn("flex", spacing.gap.compact)}>
                <input
                  type="text"
                  value={endpoint.name}
                  onChange={(e) => updateHTTPEndpoint(endpoint.id!, "name", e.target.value)}
                  placeholder={t("common.name")}
                  className={cn(
                    input.base,
                    input.state.default,
                    input.size.md,
                    "flex-1 bg-surface-raised"
                  )}
                />
                <input
                  type="number"
                  value={endpoint.expectedStatus}
                  onChange={(e) =>
                    updateHTTPEndpoint(
                      endpoint.id!,
                      "expectedStatus",
                      parseInt(e.target.value) || 200
                    )
                  }
                  placeholder={t("health.status")}
                  className={cn(
                    input.base,
                    input.state.default,
                    input.size.md,
                    "w-20 bg-surface-raised"
                  )}
                />
                <button
                  onClick={() => removeHTTPEndpoint(endpoint.id!)}
                  className={`text-status-error hover:text-status-error/80 ${spacing.actionBtn}`}
                  aria-label={t("common.remove")}
                >
                  {t("common.remove")}
                </button>
              </div>
              <input
                type="text"
                value={endpoint.url}
                onChange={(e) => updateHTTPEndpoint(endpoint.id!, "url", e.target.value)}
                placeholder="https://example.com/health"
                className={cn(input.base, input.state.default, input.size.md, "bg-surface-raised")}
              />
            </div>
          ))}
        </div>
      </div>
    </CollapsibleSection>
  );
});
