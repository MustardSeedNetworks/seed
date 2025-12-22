/**
 * HealthCheckCard Component
 *
 * Purpose: Comprehensive health check monitoring for remote services via ping, TCP, UDP, and HTTP(S).
 * Tests end-to-end connectivity and provides detailed per-phase metrics (DNS, TCP, TLS, TTFB).
 *
 * Key Features:
 * - Multi-protocol testing: ICMP ping, TCP connect, UDP, HTTP/HTTPS requests
 * - Extended ping metrics: packet loss, jitter, min/max/avg latency
 * - HTTP timing breakdown: DNS resolution, TCP connection, TLS handshake, Time-To-First-Byte (TTFB)
 * - SSL/TLS certificate monitoring: expiry date, days remaining, issuer, common name, TLS version
 * - Per-test latency thresholds: warning/critical levels from settings
 * - CollapsibleSection for each test type to show detailed results
 * - Status indicators for each phase: DNS, TCP, TLS, TTFB with color-coding
 *
 * Usage:
 * ```typescript
 * <HealthCheckCard
 *   data={healthCheckResults}
 *   loading={isRunning}
 * />
 * ```
 *
 * Dependencies: Card UI components, StatusBadge, CollapsibleSection, Tooltip, useSettings hook,
 *              auth hooks for making secure test requests, Icons, theme utilities
 * State: Manages test result data, fetches results periodically, uses SettingsContext for thresholds
 */

import { useState, useEffect, useCallback, memo } from "react";
import { useTranslation } from "react-i18next";
import { Card, Status } from "../ui/Card";
import { StatusBadge } from "../ui/StatusBadge";
import { CollapsibleSection } from "../ui/CollapsibleSection";
import { Tooltip } from "../ui/Tooltip";
// Fix #669: Removed deprecated getAuthHeaders - using credentials: 'include' for cookie auth
import { HTTP_TIMING_HELP } from "../help/HelpContent";
import { useSettings } from "../../contexts/useSettings";
import { HeartPulse } from "../ui/Icons";
import {
  cn,
  timing,
  icon as iconTokens,
  layout,
  radius,
  spacing,
} from "../../styles/theme";

type StatusValue = "success" | "warning" | "error";

interface TestResult {
  name: string;
  host?: string;
  port?: number;
  url?: string;
  success: boolean;
  latency: number;
  error?: string;
  status?: number;
  testStatus?: StatusValue;
  // Extended ping fields
  packetLoss?: number;
  jitter?: number;
  minLatency?: number;
  maxLatency?: number;
  dnsLatency?: number;
  tcpConnect?: number;
  tlsLatency?: number;
  ttfbLatency?: number;
  // Per-phase status fields
  dnsStatus?: StatusValue;
  tcpStatus?: StatusValue;
  tlsStatus?: StatusValue;
  ttfbStatus?: StatusValue;
  // Certificate expiry fields
  certDaysLeft?: number;
  certStatus?: StatusValue;
  certExpiry?: string;
  certCommonName?: string;
  tlsVersion?: string;
  certIssuer?: string;
}

interface HealthCheckData {
  pingResults: TestResult[];
  tcpResults: TestResult[];
  udpResults: TestResult[];
  httpResults: TestResult[];
  hasTests: boolean;
}

interface HealthCheckCardProps {
  loading?: boolean;
}

export const HealthCheckCard = memo(function HealthCheckCard({
  loading,
}: HealthCheckCardProps) {
  const { t } = useTranslation("cards");
  const { cardSettings } = useSettings();
  const [data, setData] = useState<HealthCheckData | null>(null);
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTests = useCallback(async () => {
    setIsRunning(true);
    setError(null);
    try {
      const res = await fetch("/api/health-checks/run", {
        credentials: "include",
      });
      if (res.ok) {
        const result = await res.json();
        setData(result);
      } else {
        setError(t("health.failedToRun"));
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t("health.failedToRun"));
    } finally {
      setIsRunning(false);
    }
  }, [t]);

  // Initial fetch to check if tests are configured
  useEffect(() => {
    fetchTests();
  }, [fetchTests]);

  // Listen for settings changes (fired when settings drawer closes after test config changes)
  useEffect(() => {
    const handleHealthChecksUpdated = () => {
      // Re-run tests with new configuration
      fetchTests();
    };
    window.addEventListener("healthChecksUpdated", handleHealthChecksUpdated);
    return () => {
      window.removeEventListener(
        "healthChecksUpdated",
        handleHealthChecksUpdated
      );
    };
  }, [fetchTests]);

  // Listen for FAB "run all tests" event
  useEffect(() => {
    const handleRunAllTests = async () => {
      // Check per-card autoRunOnLink setting - skip if health checks disabled
      if (!cardSettings.healthChecks.autoRunOnLink) {
        return;
      }

      if (!isRunning) {
        await fetchTests();
        // Signal FAB that healthchecks are complete
        window.dispatchEvent(
          new CustomEvent("cardTestComplete", {
            detail: { test: "healthchecks" },
          })
        );
      }
    };
    window.addEventListener("runAllTests", handleRunAllTests);
    return () => {
      window.removeEventListener("runAllTests", handleRunAllTests);
    };
  }, [fetchTests, isRunning, cardSettings.healthChecks.autoRunOnLink]);

  // Don't render card if no tests are configured
  if (!data?.hasTests && !loading && !isRunning) {
    return null;
  }

  const getStatus = (): Status => {
    if (loading || isRunning) return "loading";
    if (error) return "error";
    if (!data) return "unknown";

    const allResults = [
      ...data.pingResults,
      ...data.tcpResults,
      ...(data.udpResults || []),
      ...data.httpResults,
    ];
    if (allResults.length === 0) return "unknown";

    // Priority: error > warning > success
    // Any failure (!success) or error status = card is error
    if (
      allResults.some(
        (r) =>
          !r.success || r.testStatus === "error" || r.certStatus === "error"
      )
    ) {
      return "error";
    }

    // Any warning status = card is warning
    if (
      allResults.some(
        (r) => r.testStatus === "warning" || r.certStatus === "warning"
      )
    ) {
      return "warning";
    }

    // All tests passed with no warnings
    return "success";
  };

  const formatLatency = (ms: number): string => {
    if (ms >= 1000) {
      return `${(ms / 1000).toFixed(1)}s`;
    }
    return `${Math.round(ms)}ms`;
  };

  const renderTestResult = (
    result: TestResult,
    type: "ping" | "tcp" | "udp" | "http"
  ) => {
    // Use testStatus for threshold-based coloring, fall back to success/error
    const statusLabel = result.success
      ? result.testStatus === "warning"
        ? "warning"
        : "success"
      : "error";
    const statusColor =
      statusLabel === "success"
        ? "text-status-success"
        : statusLabel === "warning"
          ? "text-status-warning"
          : "text-status-error";

    // Display name - backend already formats as host:port when name is empty
    // Only add HTTP status code, not ports (already in name)
    const displayName = result.name;
    let details = "";
    if (type === "http" && result.status) {
      details = ` (${result.status})`;
    }

    // Extended ping info
    const hasExtendedPing = type === "ping" && result.packetLoss !== undefined;
    const extendedInfo = hasExtendedPing
      ? `${result.packetLoss?.toFixed(0)}% loss${result.jitter !== undefined ? `, ${result.jitter.toFixed(1)}ms jitter` : ""}`
      : null;

    return (
      <div key={`${type}-${result.name}`} className={spacing.compact.py}>
        <div className={layout.flex.between}>
          <span
            className="body-small text-text-muted truncate flex-1"
            title={displayName}
          >
            {displayName}
            {details}
          </span>
          <span className={cn("inline-flex items-center", spacing.gap.compact)}>
            <StatusBadge status={statusLabel} size="sm" />
            <span className={cn("body-small font-medium", statusColor)}>
              {result.success ? formatLatency(result.latency) : "fail"}
            </span>
          </span>
        </div>
        {extendedInfo && (
          <div className={cn("caption text-text-muted", spacing.micro.mt)}>
            {extendedInfo}
          </div>
        )}
      </div>
    );
  };

  // Timing bar component for HTTP requests
  const TimingBar = ({ result }: { result: TestResult }) => {
    // Prefer total latency; fall back to sum of phases so we can still render on failures
    const safeNum = (v: number | undefined) =>
      v !== undefined && Number.isFinite(v) ? v : 0;
    const dns = safeNum(result.dnsLatency);
    const tcp = safeNum(result.tcpConnect);
    const tls = safeNum(result.tlsLatency);
    const ttfb = safeNum(result.ttfbLatency);
    const total =
      result.latency && Number.isFinite(result.latency) && result.latency > 0
        ? result.latency
        : dns + tcp + tls + ttfb;

    // Guard against NaN, Infinity, and zero/negative values
    if (!total || !Number.isFinite(total) || total <= 0) return null;

    // Download time is what's left after subtracting known phases
    const download = Math.max(0, total - dns - tcp - tls - ttfb);

    // Get status-based text color for legend (bar colors stay fixed for phase identification)
    const getStatusTextColor = (status?: StatusValue) => {
      if (status === "error") return "text-status-error";
      if (status === "warning") return "text-status-warning";
      return "text-text-muted";
    };

    // Segment colors are fixed per-phase for consistent identification
    // Using dark mode aware colors from theme
    // Status is indicated only via text color in the legend
    const segments = [
      {
        label: t("health.timingDns"),
        value: dns,
        color: timing.dns.bg,
        status: result.dnsStatus,
      },
      {
        label: t("health.timingTcp"),
        value: tcp,
        color: timing.tcp.bg,
        status: result.tcpStatus,
      },
      {
        label: t("health.timingTls"),
        value: tls,
        color: timing.tls.bg,
        status: result.tlsStatus,
      },
      {
        label: t("health.timingWait"),
        value: ttfb,
        color: timing.wait.bg,
        status: result.ttfbStatus,
      },
      {
        label: t("health.timingDownload"),
        value: download,
        color: timing.download.bg,
        status: undefined,
      },
    ].filter((s) => s.value > 0 && Number.isFinite(s.value));

    if (segments.length === 0) return null;

    const fmt = (ms: number) =>
      ms >= 1000 ? `${(ms / 1000).toFixed(1)}s` : `${Math.round(ms)}ms`;

    return (
      <div className={spacing.micro.mtCompactMd}>
        {/* Stacked bar */}
        <div
          className={cn(
            "h-2",
            radius.full,
            "overflow-hidden flex bg-bg-tertiary"
          )}
        >
          {segments.map((seg, i) => {
            const widthPercent = Math.min(
              100,
              Math.max(0, (seg.value / total) * 100)
            );
            const widthClass = `w-[${widthPercent}%]`;
            return (
              <div
                key={seg.label}
                className={cn(
                  seg.color,
                  widthClass,
                  i === 0 ? "rounded-l-full" : "",
                  i === segments.length - 1 ? "rounded-r-full" : ""
                )}
                title={`${seg.label}: ${fmt(seg.value)}${seg.status && seg.status !== "success" ? ` (${seg.status})` : ""}`}
              />
            );
          })}
        </div>
        {/* Legend with tooltips */}
        <div
          className={cn(
            "flex flex-wrap gap-x-3",
            spacing.margin.top.tight,
            "caption",
            spacing.micro.gap
          )}
        >
          {segments.map((seg) => (
            <Tooltip
              key={seg.label}
              content={HTTP_TIMING_HELP[seg.label] || seg.label}
              position="bottom"
            >
              <span
                className={cn(
                  "inline-flex items-center",
                  spacing.gap.tight,
                  getStatusTextColor(seg.status)
                )}
              >
                <span
                  className={cn("inline-block w-2 h-2", radius.full, seg.color)}
                />
                {seg.label} {fmt(seg.value)}
              </span>
            </Tooltip>
          ))}
        </div>
      </div>
    );
  };

  const renderHTTPResult = (result: TestResult) => {
    // Use testStatus for threshold-based coloring
    let statusColor = "text-status-error";
    if (result.success) {
      if (result.testStatus === "warning") {
        statusColor = "text-status-warning";
      } else if (result.testStatus === "error") {
        statusColor = "text-status-error";
      } else {
        statusColor = "text-status-success";
      }
    }

    // Certificate status coloring
    let certColor = "text-text-muted";
    if (result.certStatus === "error") {
      certColor = "text-status-error";
    } else if (result.certStatus === "warning") {
      certColor = "text-status-warning";
    } else if (result.certStatus === "success") {
      certColor = "text-status-success";
    }

    const hasCertInfo =
      result.certDaysLeft !== undefined && result.certDaysLeft >= 0;
    const hasTLS = result.tlsVersion && result.tlsVersion !== "Unknown";

    // Format cert expiry nicely
    const formatCertExpiry = () => {
      if (!hasCertInfo) return "";
      const days = result.certDaysLeft!;
      if (days <= 0) return t("health.expired");
      if (days === 1) return t("health.certExpiry1Day");
      if (days < 30) return t("health.certExpiryDays", { days });
      if (days < 365)
        return t("health.certExpiryMonths", { months: Math.floor(days / 30) });
      return t("health.certExpiryYears", { years: Math.floor(days / 365) });
    };

    // Check if we have timing breakdown data
    const hasTimingData =
      result.dnsLatency !== undefined ||
      result.tcpConnect !== undefined ||
      result.tlsLatency !== undefined ||
      result.ttfbLatency !== undefined;

    return (
      <div key={`http-${result.name}`} className={spacing.compact.pyMd}>
        <div className={layout.flex.between}>
          <span
            className="body-small text-text-muted truncate flex-1"
            title={result.name}
          >
            {result.name}
            {result.status ? ` (${result.status})` : ""}
          </span>
          <span className={cn("body-small font-medium", statusColor)}>
            {result.success ? formatLatency(result.latency) : "fail"}
          </span>
        </div>
        {hasTimingData && <TimingBar result={result} />}
        {!result.success && result.error && (
          <div
            className={cn(
              "caption text-status-error",
              spacing.margin.top.tight
            )}
          >
            {result.error}
          </div>
        )}
        {(hasTLS || hasCertInfo) && (
          <div
            className={cn(
              "caption",
              spacing.margin.top.tight,
              layout.inline.default
            )}
          >
            {hasTLS && (
              <span className="text-text-muted">{result.tlsVersion}</span>
            )}
            {hasTLS && hasCertInfo && (
              <span className="text-text-muted">·</span>
            )}
            {hasCertInfo && (
              <span
                className={certColor}
                title={`Expires: ${result.certExpiry}`}
              >
                {formatCertExpiry()}
              </span>
            )}
            {result.certIssuer && (
              <>
                <span className="text-text-muted">·</span>
                <span
                  className="text-text-muted truncate"
                  title={result.certIssuer}
                >
                  {result.certIssuer}
                </span>
              </>
            )}
          </div>
        )}
      </div>
    );
  };

  return (
    <Card
      title={t("health.title")}
      icon={<HeartPulse className={iconTokens.size.md} />}
      status={getStatus()}
    >
      {isRunning && (
        <p className="body-small text-text-muted">{t("health.runningTests")}</p>
      )}

      {!isRunning && data && (
        <>
          {/* Ping Results */}
          {data.pingResults && data.pingResults.length > 0 && (
            <CollapsibleSection
              title={t("health.ping")}
              count={data.pingResults.length}
              variant="compact"
              defaultOpen={true}
              status={
                data.pingResults.some(
                  (r) => !r.success || r.testStatus === "error"
                )
                  ? "error"
                  : data.pingResults.some((r) => r.testStatus === "warning")
                    ? "warning"
                    : "success"
              }
            >
              {data.pingResults.map((r) => renderTestResult(r, "ping"))}
            </CollapsibleSection>
          )}

          {/* TCP Results */}
          {data.tcpResults && data.tcpResults.length > 0 && (
            <CollapsibleSection
              title={t("health.tcpPorts")}
              count={data.tcpResults.length}
              variant="compact"
              defaultOpen={true}
              status={
                data.tcpResults.some(
                  (r) => !r.success || r.testStatus === "error"
                )
                  ? "error"
                  : data.tcpResults.some((r) => r.testStatus === "warning")
                    ? "warning"
                    : "success"
              }
            >
              {data.tcpResults.map((r) => renderTestResult(r, "tcp"))}
            </CollapsibleSection>
          )}

          {/* UDP Results */}
          {data.udpResults && data.udpResults.length > 0 && (
            <CollapsibleSection
              title={t("health.udpPorts")}
              count={data.udpResults.length}
              variant="compact"
              defaultOpen={true}
              status={
                data.udpResults.some(
                  (r) => !r.success || r.testStatus === "error"
                )
                  ? "error"
                  : data.udpResults.some((r) => r.testStatus === "warning")
                    ? "warning"
                    : "success"
              }
            >
              {data.udpResults.map((r) => renderTestResult(r, "udp"))}
            </CollapsibleSection>
          )}

          {/* HTTP Results */}
          {data.httpResults && data.httpResults.length > 0 && (
            <CollapsibleSection
              title={t("health.http")}
              count={data.httpResults.length}
              variant="compact"
              defaultOpen={true}
              status={
                data.httpResults.some(
                  (r) =>
                    !r.success ||
                    r.testStatus === "error" ||
                    r.certStatus === "error"
                )
                  ? "error"
                  : data.httpResults.some(
                        (r) =>
                          r.testStatus === "warning" ||
                          r.certStatus === "warning"
                      )
                    ? "warning"
                    : "success"
              }
            >
              {data.httpResults.map((r) => renderHTTPResult(r))}
            </CollapsibleSection>
          )}
        </>
      )}

      {error && <p className="body-small text-status-error">{error}</p>}
    </Card>
  );
});
