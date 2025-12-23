/**
 * PathDiscoveryCard Component
 *
 * Purpose: Provides network path tracing (traceroute) functionality.
 * Displays hop-by-hop network path with latency and hostname resolution.
 *
 * Key Features:
 * - Traceroute with ICMP, UDP, or TCP protocols
 * - Quick target buttons for common destinations (Gateway, DNS, Internet)
 * - Hop-by-hop display with IP, hostname, RTT, and status
 * - Visual RTT bar indicator for each hop
 * - Export results as JSON or CSV
 *
 * Usage:
 * ```typescript
 * <PathDiscoveryCard gateway="192.168.1.1" dnsServer="8.8.8.8" />
 * ```
 *
 * Dependencies: Card UI components, theme utilities, traceroute API
 */

import { useState, useCallback, memo } from "react";
import { useTranslation } from "react-i18next";
import { Card, CardValue, CardDivider, Status } from "../ui/Card";
import { Route } from "../ui/Icons";
import {
  cn,
  icon as iconTokens,
  layout,
  spacing,
  radius,
  input as inputTokens,
  button as buttonTokens,
} from "../../styles/theme";
import type { TracerouteResult, TracerouteHop } from "../../types";

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
  const [result, setResult] = useState<TracerouteResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Run traceroute
  const runTrace = useCallback(
    async (traceTarget: string) => {
      if (!traceTarget.trim()) return;

      setLoading(true);
      setError(null);
      setResult(null);

      try {
        const response = await fetch(`${API_BASE}/api/discovery/traceroute`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({
            target: traceTarget.trim(),
            protocol,
            port: protocol !== "icmp" ? port : undefined,
            maxHops: 30,
            timeout: 3000,
          }),
        });

        if (!response.ok) {
          const errData = await response.json().catch(() => ({}));
          throw new Error(errData.message || "Traceroute failed");
        }

        const data = await response.json();
        setResult(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Traceroute failed");
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
    a.download = `traceroute-${result.target}-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }, [result]);

  // Export as CSV
  const exportCSV = useCallback(() => {
    if (!result) return;
    const headers = "TTL,IP,Hostname,RTT (ms),State\n";
    const rows = result.hops
      .map(
        (h) =>
          `${h.ttl},${h.ip || "*"},${h.hostname || ""},${h.rtt > 0 ? (h.rtt / 1_000_000).toFixed(2) : ""},${h.state}`
      )
      .join("\n");
    const blob = new Blob([headers + rows], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `traceroute-${result.target}-${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }, [result]);

  // Determine card status
  const cardStatus: Status = loading
    ? "loading"
    : error
      ? "error"
      : result?.completed
        ? "success"
        : result
          ? "warning"
          : "unknown";

  const maxRTT = result ? getMaxRTT(result.hops) : 1;

  return (
    <Card
      title={t("pathDiscovery.title", "Path Discovery")}
      icon={<Route className={iconTokens.size.md} />}
      status={cardStatus}
    >
      {/* Target Input Form */}
      <form
        onSubmit={handleSubmit}
        className={cn("stack-sm", spacing.margin.bottom.content)}
      >
        <div className={cn(layout.inline.default, spacing.gap.compact)}>
          <input
            type="text"
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            placeholder={t("pathDiscovery.targetPlaceholder", "IP or hostname")}
            disabled={loading}
            className={cn(
              "flex-1",
              inputTokens.base,
              inputTokens.state.default,
              inputTokens.size.sm,
              "body-small"
            )}
          />
          <select
            value={protocol}
            onChange={(e) => setProtocol(e.target.value as Protocol)}
            disabled={loading}
            className={cn(
              inputTokens.base,
              inputTokens.state.default,
              inputTokens.size.sm,
              "body-small"
            )}
          >
            <option value="icmp">ICMP</option>
            <option value="udp">UDP</option>
            <option value="tcp">TCP</option>
          </select>
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
                "w-20",
                inputTokens.base,
                inputTokens.state.default,
                inputTokens.size.sm,
                "body-small"
              )}
            />
          )}
          <button
            type="submit"
            disabled={loading || !target.trim()}
            className={cn(
              buttonTokens.base,
              buttonTokens.variant.primary,
              buttonTokens.size.sm
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
        <div className="stack-sm">
          {/* Summary Header */}
          <div className={cn(layout.flex.between, "items-center")}>
            <div>
              <span className="body-small font-medium text-text-primary">
                {t("pathDiscovery.pathTo", "Path to")} {result.target}
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
                  hop.state === "timeout"
                    ? "bg-surface-base"
                    : "bg-surface-raised",
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
              onClick={() => runTrace(result.target)}
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
          value={t("pathDiscovery.enterTarget", "Enter a target to trace")}
          size="sm"
          className="text-text-muted"
        />
      )}
    </Card>
  );
});
