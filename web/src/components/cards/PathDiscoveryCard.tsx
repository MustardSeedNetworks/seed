/**
 * PathDiscoveryCard Component
 *
 * Purpose: Provides combined L2+L3 network path tracing functionality.
 * Displays hop-by-hop network path with latency, hostname resolution,
 * and L2 switch path with port details.
 *
 * Key Features:
 * - Traceroute (L3) with ICMP, UDP, or TCP protocols
 * - L2 switch path via LLDP/CDP/EDP + SNMP
 * - Device selector with discovered devices
 * - Quick target buttons for common destinations
 * - Visual RTT bar indicator for each hop
 * - L2 path diagram with port details
 * - Export results as JSON or CSV
 *
 * Usage:
 * ```typescript
 * <PathDiscoveryCard gateway="192.168.1.1" dnsServer="8.8.8.8" />
 * ```
 *
 * Dependencies: Card UI, DeviceSelector, theme utilities, path discovery API
 */

import { useState, useCallback, memo, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Card, CardValue, CardDivider, Status } from "../ui/Card";
import { Route, ChevronDown, ChevronUp } from "../ui/Icons";
import {
  cn,
  icon as iconTokens,
  layout,
  spacing,
  radius,
  input as inputTokens,
  button as buttonTokens,
} from "../../styles/theme";
import type {
  TracerouteResult,
  TracerouteHop,
  PathResponse,
  L2PathResult,
  L2Hop,
} from "../../types";

const API_BASE = import.meta.env.VITE_API_BASE || "";

type Protocol = "icmp" | "udp" | "tcp";

interface PathDiscoveryCardProps {
  gateway?: string;
  dnsServer?: string;
}

// Format RTT from nanoseconds to readable string
function formatRTT(ns: number): string {
  if (ns <= 0) return "---";
  const ms = ns / 1_000_000;
  if (ms < 1) return "<1ms";
  if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`;
  return `${ms.toFixed(1)}ms`;
}

// Calculate max RTT for scaling bars
function getMaxRTT(hops: TracerouteHop[]): number {
  const max = Math.max(...hops.filter((h) => h.rtt > 0).map((h) => h.rtt));
  return max > 0 ? max : 1;
}

export const PathDiscoveryCard = memo(function PathDiscoveryCard({
  gateway,
  dnsServer,
}: PathDiscoveryCardProps) {
  const { t } = useTranslation("cards");

  const [target, setTarget] = useState("");
  const [protocol, setProtocol] = useState<Protocol>("icmp");
  const [port, setPort] = useState<number>(80);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<PathResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [expandedL2Hop, setExpandedL2Hop] = useState<number | null>(null);

  // Run path discovery (always L2+L3 combined)
  const runTrace = useCallback(
    async (traceTarget: string) => {
      if (!traceTarget.trim()) return;

      setLoading(true);
      setError(null);
      setResult(null);
      setExpandedL2Hop(null);

      try {
        const response = await fetch(`${API_BASE}/api/discovery/path`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({
            source: "self",
            destination: traceTarget.trim(),
            method: "both", // Always do both L2+L3
            protocol,
            port: protocol !== "icmp" ? port : undefined,
          }),
        });

        if (!response.ok) {
          const errData = await response.json().catch(() => ({}));
          throw new Error(errData.message || "Path discovery failed");
        }

        const data: PathResponse = await response.json();
        setResult(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Path discovery failed");
      } finally {
        setLoading(false);
      }
    },
    [protocol, port]
  );

  // Handle form submit
  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      runTrace(target);
    },
    [target, runTrace]
  );

  // Quick target handlers
  const traceGateway = useCallback(() => {
    if (gateway) {
      setTarget(gateway);
      runTrace(gateway);
    }
  }, [gateway, runTrace]);

  const traceDNS = useCallback(() => {
    const dns = dnsServer || "8.8.8.8";
    setTarget(dns);
    runTrace(dns);
  }, [dnsServer, runTrace]);

  const traceInternet = useCallback(() => {
    const internetTarget = "8.8.8.8";
    setTarget(internetTarget);
    runTrace(internetTarget);
  }, [runTrace]);

  // Export as JSON
  const exportJSON = useCallback(() => {
    if (!result) return;
    const blob = new Blob([JSON.stringify(result, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `path-discovery-${target}-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }, [result, target]);

  // Export as CSV
  const exportCSV = useCallback(() => {
    if (!result) return;

    let csvContent = "";

    // L3 path section
    if (result.l3Path) {
      csvContent += "L3 Path\n";
      csvContent += "TTL,IP,Hostname,RTT (ms),State\n";
      csvContent += result.l3Path.hops
        .map(
          (h) =>
            `${h.ttl},${h.ip || "*"},${h.hostname || ""},${h.rtt > 0 ? (h.rtt / 1_000_000).toFixed(2) : ""},${h.state}`
        )
        .join("\n");
    }

    // L2 path section
    if (result.l2Path) {
      if (csvContent) csvContent += "\n\n";
      csvContent += "L2 Path\n";
      csvContent += "Device,Device IP,Ingress Port,Egress Port,Source\n";
      csvContent += result.l2Path.hops
        .map(
          (h) =>
            `${h.device},${h.deviceIp},${h.ingressPort?.name || ""},${h.egressPort?.name || ""},${h.source}`
        )
        .join("\n");
    }

    const blob = new Blob([csvContent], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `path-discovery-${target}-${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }, [result, target]);

  // Copy to clipboard
  const copyToClipboard = useCallback(() => {
    if (!result) return;
    navigator.clipboard.writeText(JSON.stringify(result, null, 2));
  }, [result]);

  // Determine card status based on worst hop result
  const cardStatus: Status = useMemo(() => {
    if (loading) return "loading";
    if (error) return "error";
    if (!result) return "unknown";

    // Check L3 path for issues
    const l3Hops = result.l3Path?.hops || [];
    const hasErrors = l3Hops.some((h) => h.state === "error" || h.state === "unreachable");
    const hasTimeouts = l3Hops.some((h) => h.state === "timeout");
    const hasHighLatency = l3Hops.some((h) => h.rtt > 100000000); // > 100ms

    if (hasErrors) return "error";
    if (hasTimeouts || hasHighLatency) return "warning";
    if (result.l3Path?.completed || result.l2Path) return "success";
    return "warning";
  }, [loading, error, result]);

  const maxRTT = result?.l3Path ? getMaxRTT(result.l3Path.hops) : 1;

  return (
    <Card
      title={t("pathDiscovery.title", "Path Discovery")}
      icon={<Route className={iconTokens.size.md} />}
      status={cardStatus}
    >
      {/* Target Input Form - Simplified: just enter IP/hostname and trace */}
      <form
        onSubmit={handleSubmit}
        className={cn("stack-sm", spacing.margin.bottom.content)}
      >
        {/* Target Input - Can type IP directly or select from discovered devices */}
        <div className={cn(layout.inline.default, spacing.gap.tight, "items-center")}>
          <input
            type="text"
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            placeholder={t("pathDiscovery.enterTarget", "Enter IP or hostname...")}
            disabled={loading}
            className={cn(
              "flex-1 min-w-0",
              inputTokens.base,
              inputTokens.state.default,
              inputTokens.size.sm,
              "body-small"
            )}
            onKeyDown={(e) => {
              if (e.key === "Enter" && target.trim()) {
                e.preventDefault();
                handleSubmit(e as unknown as React.FormEvent);
              }
            }}
          />

          {/* Protocol selector - compact */}
          <select
            value={protocol}
            onChange={(e) => setProtocol(e.target.value as Protocol)}
            disabled={loading}
            className={cn(
              inputTokens.base,
              inputTokens.state.default,
              "px-2 py-1 caption shrink-0"
            )}
            title={t("pathDiscovery.protocol", "Traceroute protocol")}
          >
            <option value="icmp">ICMP</option>
            <option value="udp">UDP</option>
            <option value="tcp">TCP</option>
          </select>

          {/* Port input (only for TCP/UDP) */}
          {protocol !== "icmp" && (
            <input
              type="number"
              value={port}
              onChange={(e) => setPort(parseInt(e.target.value) || 80)}
              placeholder="Port"
              min={1}
              max={65535}
              disabled={loading}
              className={cn(
                "w-16",
                inputTokens.base,
                inputTokens.state.default,
                "px-2 py-1 caption shrink-0"
              )}
            />
          )}

          <button
            type="submit"
            disabled={loading || !target.trim()}
            className={cn(
              buttonTokens.base,
              buttonTokens.variant.primary,
              buttonTokens.size.sm,
              "shrink-0"
            )}
          >
            {loading ? "..." : t("pathDiscovery.trace", "Trace")}
          </button>
        </div>

        {/* Quick Targets */}
        <div className={cn(layout.inline.default, spacing.gap.compact)}>
          <span className="caption text-text-muted">
            {t("pathDiscovery.quick", "Quick")}:
          </span>
          <button
            type="button"
            onClick={traceGateway}
            disabled={loading || !gateway}
            className={cn(
              buttonTokens.base,
              buttonTokens.variant.ghost,
              buttonTokens.size.xs,
              "caption"
            )}
          >
            {t("pathDiscovery.gateway", "Gateway")}
          </button>
          <button
            type="button"
            onClick={traceDNS}
            disabled={loading}
            className={cn(
              buttonTokens.base,
              buttonTokens.variant.ghost,
              buttonTokens.size.xs,
              "caption"
            )}
          >
            {t("pathDiscovery.dns", "DNS")}
          </button>
          <button
            type="button"
            onClick={traceInternet}
            disabled={loading}
            className={cn(
              buttonTokens.base,
              buttonTokens.variant.ghost,
              buttonTokens.size.xs,
              "caption"
            )}
          >
            {t("pathDiscovery.internet", "Internet")}
          </button>
        </div>
      </form>

      <CardDivider />

      {/* Loading State */}
      {loading && (
        <CardValue
          value={t("pathDiscovery.tracing", "Tracing path...")}
          size="lg"
        />
      )}

      {/* Error State */}
      {error && !loading && (
        <div
          className={cn(spacing.pad.sm, "bg-status-error/10", radius.default)}
        >
          <span className="body-small text-status-error">{error}</span>
        </div>
      )}

      {/* Results */}
      {result && !loading && (
        <div className="stack-md">
          {/* L3 Path Results */}
          {result.l3Path && (
            <L3PathDisplay result={result.l3Path} maxRTT={maxRTT} t={t} />
          )}

          {/* L2 Path Results */}
          {result.l2Path && (
            <L2PathDisplay
              result={result.l2Path}
              expandedHop={expandedL2Hop}
              onToggleHop={setExpandedL2Hop}
              t={t}
            />
          )}

          {/* Export Actions */}
          <div
            className={cn(
              layout.inline.default,
              spacing.gap.compact,
              spacing.margin.top.inline
            )}
          >
            <button
              type="button"
              onClick={exportJSON}
              className={cn(
                buttonTokens.base,
                buttonTokens.variant.ghost,
                buttonTokens.size.xs,
                "caption"
              )}
            >
              {t("pathDiscovery.exportJSON", "Export JSON")}
            </button>
            <button
              type="button"
              onClick={exportCSV}
              className={cn(
                buttonTokens.base,
                buttonTokens.variant.ghost,
                buttonTokens.size.xs,
                "caption"
              )}
            >
              {t("pathDiscovery.exportCSV", "Export CSV")}
            </button>
            <button
              type="button"
              onClick={copyToClipboard}
              className={cn(
                buttonTokens.base,
                buttonTokens.variant.ghost,
                buttonTokens.size.xs,
                "caption"
              )}
            >
              {t("pathDiscovery.copy", "Copy")}
            </button>
            <button
              type="button"
              onClick={() => runTrace(target)}
              disabled={loading}
              className={cn(
                buttonTokens.base,
                buttonTokens.variant.ghost,
                buttonTokens.size.xs,
                "caption"
              )}
            >
              {t("pathDiscovery.rerun", "Re-run")}
            </button>
          </div>
        </div>
      )}

      {/* Empty State */}
      {!result && !loading && !error && (
        <CardValue
          value={t("pathDiscovery.enterTarget", "Select a target to trace")}
          size="sm"
          className="text-text-muted"
        />
      )}
    </Card>
  );
});

// L3 Path Display Component
interface L3PathDisplayProps {
  result: TracerouteResult;
  maxRTT: number;
  t: (key: string, fallback: string) => string;
}

const L3PathDisplay = memo(function L3PathDisplay({
  result,
  maxRTT,
  t,
}: L3PathDisplayProps) {
  return (
    <div className="stack-sm">
      {/* L3 Header */}
      <div className={cn(layout.flex.between, "items-center")}>
        <div>
          <span className="body-small font-semibold text-brand-primary">
            L3 {t("pathDiscovery.path", "Path")}
          </span>
          <span className="body-small font-medium text-text-primary ml-2">
            {t("pathDiscovery.to", "to")} {result.target}
          </span>
          <span className="caption text-text-muted ml-2">
            ({result.hops.length} {t("pathDiscovery.hops", "hops")})
          </span>
        </div>
        {result.completed && (
          <span className="caption text-status-success">
            {t("pathDiscovery.completed", "Completed")}
          </span>
        )}
      </div>

      {/* Hop List */}
      <div className={cn("stack-xs", spacing.margin.top.inline)}>
        {result.hops.map((hop) => (
          <div
            key={hop.ttl}
            className={cn(
              layout.inline.default,
              spacing.gap.compact,
              spacing.pad.xs,
              radius.default,
              hop.state === "timeout" ? "bg-surface-base" : "bg-surface-raised",
              "border border-surface-border"
            )}
          >
            {/* TTL */}
            <span className="w-6 caption font-mono text-text-muted">
              {hop.ttl}
            </span>

            {/* IP and Hostname */}
            <div className="flex-1 min-w-0">
              {hop.state === "timeout" ? (
                <span className="caption text-text-muted">* * *</span>
              ) : (
                <>
                  <span className="body-small font-mono text-text-primary truncate">
                    {hop.ip || "?"}
                  </span>
                  {hop.hostname && hop.hostname !== hop.ip && (
                    <span className="caption text-text-muted ml-2 truncate">
                      {hop.hostname}
                    </span>
                  )}
                </>
              )}
            </div>

            {/* RTT */}
            <span
              className={cn(
                "w-16 text-right caption font-mono",
                hop.state === "timeout"
                  ? "text-text-muted"
                  : "text-text-primary"
              )}
            >
              {formatRTT(hop.rtt)}
            </span>

            {/* RTT Bar */}
            <div
              className={cn(
                "w-20 h-2",
                radius.full,
                "bg-surface-border overflow-hidden"
              )}
            >
              {hop.rtt > 0 && (
                <div
                  className={cn(
                    "h-full",
                    radius.full,
                    hop.state === "error"
                      ? "bg-status-error"
                      : hop.rtt / maxRTT > 0.7
                        ? "bg-status-warning"
                        : "bg-status-success"
                  )}
                  style={{
                    width: `${Math.min(100, (hop.rtt / maxRTT) * 100)}%`,
                  }}
                />
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
});

// L2 Path Display Component
interface L2PathDisplayProps {
  result: L2PathResult;
  expandedHop: number | null;
  onToggleHop: (index: number | null) => void;
  t: (key: string, fallback: string) => string;
}

const L2PathDisplay = memo(function L2PathDisplay({
  result,
  expandedHop,
  onToggleHop,
  t,
}: L2PathDisplayProps) {
  const toggleHop = useCallback(
    (index: number) => {
      onToggleHop(expandedHop === index ? null : index);
    },
    [expandedHop, onToggleHop]
  );

  if (result.hops.length === 0) {
    return (
      <div className="stack-sm">
        <div className="body-small font-semibold text-brand-primary">
          L2 {t("pathDiscovery.path", "Path")}
        </div>
        <div className={cn(spacing.pad.sm, "bg-surface-base", radius.default)}>
          <span className="caption text-text-muted">
            {t("pathDiscovery.noL2Path", "No L2 path information available")}
          </span>
        </div>
      </div>
    );
  }

  return (
    <div className="stack-sm">
      {/* L2 Header */}
      <div className={cn(layout.flex.between, "items-center")}>
        <div>
          <span className="body-small font-semibold text-brand-primary">
            L2 {t("pathDiscovery.path", "Path")}
          </span>
          <span className="caption text-text-muted ml-2">
            (via LLDP/CDP/SNMP)
          </span>
        </div>
        <span className="caption text-text-muted">
          {result.hops.length} {t("pathDiscovery.switches", "switches")}
        </span>
      </div>

      {/* Visual Path Diagram */}
      <div
        className={cn(
          "flex items-center overflow-x-auto",
          spacing.pad.sm,
          "bg-surface-base",
          radius.default,
          "border border-surface-border"
        )}
      >
        {result.hops.map((hop, index) => (
          <div key={index} className="flex items-center shrink-0">
            {/* Switch Box */}
            <div
              className={cn(
                "flex flex-col items-center",
                spacing.pad.sm,
                "bg-surface-raised",
                radius.md,
                "border border-surface-border",
                "min-w-28"
              )}
            >
              <span className="caption font-semibold text-text-primary truncate max-w-24">
                {hop.device || hop.deviceIp}
              </span>
              <span className="caption text-text-muted">{hop.deviceIp}</span>
              <span
                className={cn(
                  "caption",
                  hop.source === "lldp"
                    ? "text-brand-primary"
                    : hop.source === "cdp"
                      ? "text-status-success"
                      : "text-text-muted"
                )}
              >
                {hop.source.toUpperCase()}
              </span>
            </div>

            {/* Arrow with port names */}
            {index < result.hops.length - 1 && (
              <div className="flex items-center mx-2">
                <div className="flex flex-col items-end mr-1">
                  {hop.egressPort && (
                    <span className="caption text-text-muted">
                      {hop.egressPort.name}
                    </span>
                  )}
                </div>
                <div className="w-8 h-0.5 bg-brand-primary relative">
                  <div
                    className="absolute right-0 top-1/2 -translate-y-1/2 w-0 h-0"
                    style={{
                      borderTop: "4px solid transparent",
                      borderBottom: "4px solid transparent",
                      borderLeft: "6px solid var(--brand-primary)",
                    }}
                  />
                </div>
                <div className="flex flex-col items-start ml-1">
                  {result.hops[index + 1]?.ingressPort && (
                    <span className="caption text-text-muted">
                      {result.hops[index + 1].ingressPort?.name}
                    </span>
                  )}
                </div>
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Detailed Port Information */}
      <div className="stack-xs">
        {result.hops.map((hop, index) => (
          <L2HopDetail
            key={index}
            hop={hop}
            index={index}
            isExpanded={expandedHop === index}
            onToggle={() => toggleHop(index)}
            t={t}
          />
        ))}
      </div>
    </div>
  );
});

// L2 Hop Detail Component
interface L2HopDetailProps {
  hop: L2Hop;
  index: number;
  isExpanded: boolean;
  onToggle: () => void;
  t: (key: string, fallback: string) => string;
}

const L2HopDetail = memo(function L2HopDetail({
  hop,
  isExpanded,
  onToggle,
  t,
}: L2HopDetailProps) {
  return (
    <div
      className={cn(
        "border border-surface-border",
        radius.default,
        "overflow-hidden"
      )}
    >
      {/* Header */}
      <button
        type="button"
        onClick={onToggle}
        className={cn(
          "w-full flex items-center justify-between",
          spacing.pad.sm,
          "bg-surface-raised hover:bg-surface-hover transition-colors",
          "text-left"
        )}
      >
        <div className="flex items-center gap-2">
          <span className="body-small font-medium text-text-primary">
            {hop.device || hop.deviceIp}
          </span>
          <span className="caption text-text-muted">({hop.deviceIp})</span>
        </div>
        {isExpanded ? (
          <ChevronUp className={cn(iconTokens.size.sm, "text-text-muted")} />
        ) : (
          <ChevronDown className={cn(iconTokens.size.sm, "text-text-muted")} />
        )}
      </button>

      {/* Expanded Details */}
      {isExpanded && (
        <div
          className={cn(
            spacing.pad.sm,
            "bg-surface-base border-t border-surface-border"
          )}
        >
          <div className="grid grid-cols-2 gap-4">
            {/* Ingress Port */}
            <div>
              <div className="caption font-semibold text-text-muted uppercase tracking-wide mb-2">
                {t("pathDiscovery.ingressPort", "Ingress Port")}
              </div>
              {hop.ingressPort ? (
                <PortDetails port={hop.ingressPort} t={t} />
              ) : (
                <span className="caption text-text-muted">---</span>
              )}
            </div>

            {/* Egress Port */}
            <div>
              <div className="caption font-semibold text-text-muted uppercase tracking-wide mb-2">
                {t("pathDiscovery.egressPort", "Egress Port")}
              </div>
              {hop.egressPort ? (
                <PortDetails port={hop.egressPort} t={t} />
              ) : (
                <span className="caption text-text-muted">---</span>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
});

// Port Details Component
interface PortDetailsProps {
  port: L2Hop["ingressPort"];
  t: (key: string, fallback: string) => string;
}

const PortDetails = memo(function PortDetails({ port, t }: PortDetailsProps) {
  if (!port) return null;

  return (
    <div className="stack-xs">
      <div className="body-small font-mono text-text-primary">{port.name}</div>
      <div className="flex flex-wrap gap-2">
        {port.speed && (
          <span className="caption text-text-secondary">{port.speed}</span>
        )}
        {port.duplex && (
          <span className="caption text-text-muted">{port.duplex}</span>
        )}
        {port.isTrunk && (
          <span className="caption text-brand-primary">
            {t("pathDiscovery.trunk", "Trunk")}
          </span>
        )}
      </div>
      {port.vlans && port.vlans.length > 0 && (
        <div className="caption text-text-muted">
          VLANs: {port.vlans.slice(0, 5).join(", ")}
          {port.vlans.length > 5 && ` +${port.vlans.length - 5}`}
        </div>
      )}
      {port.connectedTo && (
        <div className="caption text-text-secondary">→ {port.connectedTo}</div>
      )}
    </div>
  );
});
