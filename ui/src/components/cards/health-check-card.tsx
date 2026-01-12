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

import { memo, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSettings } from "../../contexts/useSettings";
import { cn, icon as iconTokens, layout, radius, spacing, timing } from "../../styles/theme";
import { HTTP_TIMING_HELP } from "../help/help-content";
import { Card, type Status } from "../ui/card";
import { CollapsibleSection } from "../ui/collapsible-section";
import { HeartPulse } from "../ui/icons";
import { StatusBadge } from "../ui/status-badge";
import { Tooltip } from "../ui/tooltip";

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

interface SqlTestResult {
  name: string;
  driver: string;
  host: string;
  port: number;
  database: string;
  success: boolean;
  connectTimeMs: number;
  queryTimeMs?: number;
  totalTimeMs: number;
  serverVersion?: string;
  error?: string;
}

interface FileShareTestResult {
  name: string;
  protocol: string;
  host: string;
  share: string;
  success: boolean;
  connectTimeMs: number;
  readSpeedMbps?: number;
  writeSpeedMbps?: number;
  readLatencyMs?: number;
  writeLatencyMs?: number;
  testFileSizeMb?: number;
  totalTimeMs: number;
  error?: string;
}

interface LdapTestResult {
  name: string;
  host: string;
  port: number;
  useTls: boolean;
  success: boolean;
  connectTimeMs: number;
  bindTimeMs?: number;
  searchTimeMs?: number;
  totalTimeMs: number;
  entriesFound?: number;
  serverInfo?: string;
  error?: string;
}

// Video protocol results
interface RtspTestResult {
  name: string;
  url: string;
  success: boolean;
  connectTimeMs: number;
  streamInfo?: string;
  codec?: string;
  resolution?: string;
  error?: string;
}

// Medical protocol results
interface DicomTestResult {
  name: string;
  host: string;
  port: number;
  aeTitle: string;
  success: boolean;
  connectTimeMs: number;
  echoTimeMs?: number;
  totalTimeMs: number;
  serverAeTitle?: string;
  error?: string;
}

interface Hl7TestResult {
  name: string;
  host: string;
  port: number;
  success: boolean;
  connectTimeMs: number;
  responseTimeMs?: number;
  totalTimeMs: number;
  ackCode?: string; // AA, AE, AR
  serverVersion?: string;
  error?: string;
}

interface FhirTestResult {
  name: string;
  baseUrl: string;
  success: boolean;
  connectTimeMs: number;
  responseTimeMs?: number;
  totalTimeMs: number;
  fhirVersion?: string;
  resourceCount?: number;
  serverName?: string;
  error?: string;
}

// Education protocol results
interface LtiTestResult {
  name: string;
  launchUrl: string;
  success: boolean;
  connectTimeMs: number;
  responseTimeMs?: number;
  totalTimeMs: number;
  ltiVersion?: string;
  error?: string;
}

// Industrial protocol results
interface OpcuaTestResult {
  name: string;
  endpointUrl: string;
  success: boolean;
  connectTimeMs: number;
  browseTimeMs?: number;
  totalTimeMs: number;
  securityMode?: string;
  serverState?: string;
  productName?: string;
  error?: string;
}

interface ModbusTestResult {
  name: string;
  host: string;
  port: number;
  unitId: number;
  success: boolean;
  connectTimeMs: number;
  readTimeMs?: number;
  totalTimeMs: number;
  registerValue?: number;
  error?: string;
}

interface EnterpriseResults {
  sqlResults?: SqlTestResult[];
  fileShareResults?: FileShareTestResult[];
  ldapResults?: LdapTestResult[];
}

interface VideoResults {
  rtspResults?: RtspTestResult[];
}

interface MedicalResults {
  dicomResults?: DicomTestResult[];
  hl7Results?: Hl7TestResult[];
  fhirResults?: FhirTestResult[];
}

interface EducationResults {
  ltiResults?: LtiTestResult[];
}

interface IndustrialResults {
  opcuaResults?: OpcuaTestResult[];
  modbusResults?: ModbusTestResult[];
}

interface HealthCheckData {
  pingResults: TestResult[];
  tcpResults: TestResult[];
  udpResults: TestResult[];
  httpResults: TestResult[];
  enterpriseResults?: EnterpriseResults;
  videoResults?: VideoResults;
  medicalResults?: MedicalResults;
  educationResults?: EducationResults;
  industrialResults?: IndustrialResults;
  hasTests: boolean;
}

interface HealthCheckCardProps {
  loading?: boolean;
}

// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Health check orchestrates multiple test types
export const HealthCheckCard = memo(function HealthCheckCard({ loading }: HealthCheckCardProps) {
  const { t } = useTranslation("cards");
  const { cardSettings } = useSettings();
  const [data, setData] = useState<HealthCheckData | null>(null);
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTests = useCallback(async () => {
    setIsRunning(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/sap/health-checks/run", {
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
      window.removeEventListener("healthChecksUpdated", handleHealthChecksUpdated);
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
          }),
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
      allResults.some((r) => !r.success || r.testStatus === "error" || r.certStatus === "error")
    ) {
      return "error";
    }

    // Any warning status = card is warning
    if (allResults.some((r) => r.testStatus === "warning" || r.certStatus === "warning")) {
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

  // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Renders multiple test result types
  const renderTestResult = (result: TestResult, type: "ping" | "tcp" | "udp" | "http") => {
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
          <span className="body-small text-text-muted truncate flex-1" title={displayName}>
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
          <div className={cn("caption text-text-muted", spacing.micro.mt)}>{extendedInfo}</div>
        )}
      </div>
    );
  };

  // Timing bar component for HTTP requests
  const TimingBar = ({ result }: { result: TestResult }) => {
    // Prefer total latency; fall back to sum of phases so we can still render on failures
    const safeNum = (v: number | undefined) => (v !== undefined && Number.isFinite(v) ? v : 0);
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

    const fmt = (ms: number) => (ms >= 1000 ? `${(ms / 1000).toFixed(1)}s` : `${Math.round(ms)}ms`);

    return (
      <div className={spacing.micro.mtCompactMd}>
        {/* Stacked bar */}
        <div className={cn("h-2", radius.full, "overflow-hidden flex bg-bg-tertiary")}>
          {segments.map((seg, i) => {
            const widthPercent = Math.min(100, Math.max(0, (seg.value / total) * 100));
            return (
              <div
                key={seg.label}
                className={cn(
                  "h-full",
                  seg.color,
                  i === 0 ? "rounded-l-full" : "",
                  i === segments.length - 1 ? "rounded-r-full" : "",
                )}
                style={{ width: `${widthPercent}%` }}
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
            spacing.micro.gap,
          )}
        >
          {segments.map((seg) => (
            <Tooltip
              key={seg.label}
              content={HTTP_TIMING_HELP[seg.label.toLowerCase()] || seg.label}
              position="bottom"
            >
              <span
                className={cn(
                  "inline-flex items-center",
                  spacing.gap.tight,
                  getStatusTextColor(seg.status),
                )}
              >
                <span className={cn("inline-block w-2 h-2", radius.full, seg.color)} />
                {seg.label} {fmt(seg.value)}
              </span>
            </Tooltip>
          ))}
        </div>
      </div>
    );
  };

  // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: HTTP results have multiple status conditions
  const renderHttpResult = (result: TestResult) => {
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

    const hasCertInfo = result.certDaysLeft !== undefined && result.certDaysLeft >= 0;
    const hasTls = result.tlsVersion && result.tlsVersion !== "Unknown";

    // Format cert expiry nicely
    const formatCertExpiry = () => {
      if (!hasCertInfo || result.certDaysLeft === undefined) return "";
      const days = result.certDaysLeft;
      if (days <= 0) return t("health.expired");
      if (days === 1) return t("health.certExpiry1Day");
      if (days < 30) return t("health.certExpiryDays", { days });
      if (days < 365) return t("health.certExpiryMonths", { months: Math.floor(days / 30) });
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
          <span className="body-small text-text-muted truncate flex-1" title={result.name}>
            {result.name}
            {result.status ? ` (${result.status})` : ""}
          </span>
          <span className={cn("body-small font-medium", statusColor)}>
            {result.success ? formatLatency(result.latency) : "fail"}
          </span>
        </div>
        {hasTimingData && <TimingBar result={result} />}
        {!result.success && result.error && (
          <div className={cn("caption text-status-error", spacing.margin.top.tight)}>
            {result.error}
          </div>
        )}
        {(hasTls || hasCertInfo) && (
          <div className={cn("caption", spacing.margin.top.tight, layout.inline.default)}>
            {hasTls && <span className="text-text-muted">{result.tlsVersion}</span>}
            {hasTls && hasCertInfo && <span className="text-text-muted">·</span>}
            {hasCertInfo && (
              <span className={certColor} title={`Expires: ${result.certExpiry}`}>
                {formatCertExpiry()}
              </span>
            )}
            {result.certIssuer && (
              <>
                <span className="text-text-muted">·</span>
                <span className="text-text-muted truncate" title={result.certIssuer}>
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
      {isRunning && <p className="body-small text-text-muted">{t("health.runningTests")}</p>}

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
                data.pingResults.some((r) => !r.success || r.testStatus === "error")
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
                data.tcpResults.some((r) => !r.success || r.testStatus === "error")
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
                data.udpResults.some((r) => !r.success || r.testStatus === "error")
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
                  (r) => !r.success || r.testStatus === "error" || r.certStatus === "error",
                )
                  ? "error"
                  : data.httpResults.some(
                        (r) => r.testStatus === "warning" || r.certStatus === "warning",
                      )
                    ? "warning"
                    : "success"
              }
            >
              {data.httpResults.map((r) => renderHttpResult(r))}
            </CollapsibleSection>
          )}

          {/* SQL/Database Results */}
          {data.enterpriseResults?.sqlResults && data.enterpriseResults.sqlResults.length > 0 && (
            <CollapsibleSection
              title={t("health.database")}
              count={data.enterpriseResults.sqlResults.length}
              variant="compact"
              defaultOpen={true}
              status={
                data.enterpriseResults.sqlResults.some((r) => !r.success) ? "error" : "success"
              }
            >
              {data.enterpriseResults.sqlResults.map((r) => (
                <div
                  key={`sql-${r.name}`}
                  className={cn(
                    layout.flex.between,
                    spacing.pad.xs,
                    radius.default,
                    !r.success ? "bg-status-error/10" : "bg-surface-raised",
                  )}
                >
                  <div className={layout.stack.compact}>
                    <div className="flex items-center gap-2">
                      <StatusBadge status={r.success ? "success" : "error"} />
                      <span className="body-small font-medium">{r.name}</span>
                      <span className="caption text-text-muted">
                        {r.driver} • {r.host}:{r.port}
                      </span>
                    </div>
                    {r.success && r.serverVersion && (
                      <span className="caption text-text-muted ml-6">{r.serverVersion}</span>
                    )}
                    {r.error && <span className="caption text-status-error ml-6">{r.error}</span>}
                  </div>
                  <div className="text-right">
                    <div className="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
                    {r.success && (
                      <div className="caption text-text-muted">
                        Connect: {r.connectTimeMs.toFixed(1)}ms
                        {r.queryTimeMs !== undefined && ` • Query: ${r.queryTimeMs.toFixed(1)}ms`}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </CollapsibleSection>
          )}

          {/* File Share Results (SMB/NFS) */}
          {data.enterpriseResults?.fileShareResults &&
            data.enterpriseResults.fileShareResults.length > 0 && (
              <CollapsibleSection
                title={t("health.fileShares")}
                count={data.enterpriseResults.fileShareResults.length}
                variant="compact"
                defaultOpen={true}
                status={
                  data.enterpriseResults.fileShareResults.some((r) => !r.success)
                    ? "error"
                    : "success"
                }
              >
                {data.enterpriseResults.fileShareResults.map((r) => (
                  <div
                    key={`fileshare-${r.name}`}
                    className={cn(
                      layout.flex.between,
                      spacing.pad.xs,
                      radius.default,
                      !r.success ? "bg-status-error/10" : "bg-surface-raised",
                    )}
                  >
                    <div className={layout.stack.compact}>
                      <div className="flex items-center gap-2">
                        <StatusBadge status={r.success ? "success" : "error"} />
                        <span className="body-small font-medium">{r.name}</span>
                        <span className="caption text-text-muted">
                          {r.protocol.toUpperCase()} • {"//"}
                          {r.host}/{r.share}
                        </span>
                      </div>
                      {r.error && <span className="caption text-status-error ml-6">{r.error}</span>}
                    </div>
                    <div className="text-right">
                      <div className="body-small font-mono">{r.connectTimeMs.toFixed(1)}ms</div>
                      {r.success &&
                        (r.readSpeedMbps !== undefined || r.writeSpeedMbps !== undefined) && (
                          <div className="caption text-text-muted">
                            {r.readSpeedMbps !== undefined &&
                              `R: ${r.readSpeedMbps.toFixed(1)} MB/s`}
                            {r.readSpeedMbps !== undefined &&
                              r.writeSpeedMbps !== undefined &&
                              " • "}
                            {r.writeSpeedMbps !== undefined &&
                              `W: ${r.writeSpeedMbps.toFixed(1)} MB/s`}
                          </div>
                        )}
                    </div>
                  </div>
                ))}
              </CollapsibleSection>
            )}

          {/* LDAP Results */}
          {data.enterpriseResults?.ldapResults && data.enterpriseResults.ldapResults.length > 0 && (
            <CollapsibleSection
              title="LDAP"
              count={data.enterpriseResults.ldapResults.length}
              variant="compact"
              defaultOpen={true}
              status={
                data.enterpriseResults.ldapResults.some((r) => !r.success) ? "error" : "success"
              }
            >
              {data.enterpriseResults.ldapResults.map((r) => (
                <div
                  key={`ldap-${r.name}`}
                  className={cn(
                    layout.flex.between,
                    spacing.pad.xs,
                    radius.default,
                    !r.success ? "bg-status-error/10" : "bg-surface-raised",
                  )}
                >
                  <div className={layout.stack.compact}>
                    <div className="flex items-center gap-2">
                      <StatusBadge status={r.success ? "success" : "error"} />
                      <span className="body-small font-medium">{r.name}</span>
                      <span className="caption text-text-muted">
                        {r.useTls ? "LDAPS" : "LDAP"} • {r.host}:{r.port}
                      </span>
                    </div>
                    {r.success && r.serverInfo && (
                      <span className="caption text-text-muted ml-6">{r.serverInfo}</span>
                    )}
                    {r.error && <span className="caption text-status-error ml-6">{r.error}</span>}
                  </div>
                  <div className="text-right">
                    <div className="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
                    {r.success && (
                      <div className="caption text-text-muted">
                        Connect: {r.connectTimeMs.toFixed(1)}ms
                        {r.bindTimeMs !== undefined && ` • Bind: ${r.bindTimeMs.toFixed(1)}ms`}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </CollapsibleSection>
          )}

          {/* RTSP Video Results */}
          {data.videoResults?.rtspResults && data.videoResults.rtspResults.length > 0 && (
            <CollapsibleSection
              title={t("health.rtsp")}
              count={data.videoResults.rtspResults.length}
              variant="compact"
              defaultOpen={true}
              status={data.videoResults.rtspResults.some((r) => !r.success) ? "error" : "success"}
            >
              {data.videoResults.rtspResults.map((r) => (
                <div
                  key={`rtsp-${r.name}`}
                  className={cn(
                    layout.flex.between,
                    spacing.pad.xs,
                    radius.default,
                    !r.success ? "bg-status-error/10" : "bg-surface-raised",
                  )}
                >
                  <div className={layout.stack.compact}>
                    <div className="flex items-center gap-2">
                      <StatusBadge status={r.success ? "success" : "error"} />
                      <span className="body-small font-medium">{r.name}</span>
                      <span className="caption text-text-muted truncate max-w-48" title={r.url}>
                        {r.url}
                      </span>
                    </div>
                    {r.success && (r.codec || r.resolution) && (
                      <span className="caption text-text-muted ml-6">
                        {r.codec && r.codec}
                        {r.codec && r.resolution && " • "}
                        {r.resolution && r.resolution}
                      </span>
                    )}
                    {r.error && <span className="caption text-status-error ml-6">{r.error}</span>}
                  </div>
                  <div className="text-right">
                    <div className="body-small font-mono">{r.connectTimeMs.toFixed(1)}ms</div>
                  </div>
                </div>
              ))}
            </CollapsibleSection>
          )}

          {/* DICOM Results */}
          {data.medicalResults?.dicomResults && data.medicalResults.dicomResults.length > 0 && (
            <CollapsibleSection
              title="DICOM"
              count={data.medicalResults.dicomResults.length}
              variant="compact"
              defaultOpen={true}
              status={
                data.medicalResults.dicomResults.some((r) => !r.success) ? "error" : "success"
              }
            >
              {data.medicalResults.dicomResults.map((r) => (
                <div
                  key={`dicom-${r.name}`}
                  className={cn(
                    layout.flex.between,
                    spacing.pad.xs,
                    radius.default,
                    !r.success ? "bg-status-error/10" : "bg-surface-raised",
                  )}
                >
                  <div className={layout.stack.compact}>
                    <div className="flex items-center gap-2">
                      <StatusBadge status={r.success ? "success" : "error"} />
                      <span className="body-small font-medium">{r.name}</span>
                      <span className="caption text-text-muted">
                        {r.host}:{r.port} • AE: {r.aeTitle}
                      </span>
                    </div>
                    {r.success && r.serverAeTitle && (
                      <span className="caption text-text-muted ml-6">
                        Server AE: {r.serverAeTitle}
                      </span>
                    )}
                    {r.error && <span className="caption text-status-error ml-6">{r.error}</span>}
                  </div>
                  <div className="text-right">
                    <div className="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
                    {r.success && r.echoTimeMs !== undefined && (
                      <div className="caption text-text-muted">
                        C-ECHO: {r.echoTimeMs.toFixed(1)}ms
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </CollapsibleSection>
          )}

          {/* HL7 MLLP Results */}
          {data.medicalResults?.hl7Results && data.medicalResults.hl7Results.length > 0 && (
            <CollapsibleSection
              title="HL7 MLLP"
              count={data.medicalResults.hl7Results.length}
              variant="compact"
              defaultOpen={true}
              status={data.medicalResults.hl7Results.some((r) => !r.success) ? "error" : "success"}
            >
              {data.medicalResults.hl7Results.map((r) => (
                <div
                  key={`hl7-${r.name}`}
                  className={cn(
                    layout.flex.between,
                    spacing.pad.xs,
                    radius.default,
                    !r.success ? "bg-status-error/10" : "bg-surface-raised",
                  )}
                >
                  <div className={layout.stack.compact}>
                    <div className="flex items-center gap-2">
                      <StatusBadge status={r.success ? "success" : "error"} />
                      <span className="body-small font-medium">{r.name}</span>
                      <span className="caption text-text-muted">
                        {r.host}:{r.port}
                      </span>
                    </div>
                    {r.success && (r.ackCode || r.serverVersion) && (
                      <span className="caption text-text-muted ml-6">
                        {r.ackCode && `ACK: ${r.ackCode}`}
                        {r.ackCode && r.serverVersion && " • "}
                        {r.serverVersion && r.serverVersion}
                      </span>
                    )}
                    {r.error && <span className="caption text-status-error ml-6">{r.error}</span>}
                  </div>
                  <div className="text-right">
                    <div className="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
                    {r.success && r.responseTimeMs !== undefined && (
                      <div className="caption text-text-muted">
                        Response: {r.responseTimeMs.toFixed(1)}ms
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </CollapsibleSection>
          )}

          {/* FHIR Results */}
          {data.medicalResults?.fhirResults && data.medicalResults.fhirResults.length > 0 && (
            <CollapsibleSection
              title="FHIR R4"
              count={data.medicalResults.fhirResults.length}
              variant="compact"
              defaultOpen={true}
              status={data.medicalResults.fhirResults.some((r) => !r.success) ? "error" : "success"}
            >
              {data.medicalResults.fhirResults.map((r) => (
                <div
                  key={`fhir-${r.name}`}
                  className={cn(
                    layout.flex.between,
                    spacing.pad.xs,
                    radius.default,
                    !r.success ? "bg-status-error/10" : "bg-surface-raised",
                  )}
                >
                  <div className={layout.stack.compact}>
                    <div className="flex items-center gap-2">
                      <StatusBadge status={r.success ? "success" : "error"} />
                      <span className="body-small font-medium">{r.name}</span>
                      <span className="caption text-text-muted truncate max-w-48" title={r.baseUrl}>
                        {r.baseUrl}
                      </span>
                    </div>
                    {r.success && (r.fhirVersion || r.serverName) && (
                      <span className="caption text-text-muted ml-6">
                        {r.fhirVersion && `v${r.fhirVersion}`}
                        {r.fhirVersion && r.serverName && " • "}
                        {r.serverName && r.serverName}
                      </span>
                    )}
                    {r.error && <span className="caption text-status-error ml-6">{r.error}</span>}
                  </div>
                  <div className="text-right">
                    <div className="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
                    {r.success && r.resourceCount !== undefined && (
                      <div className="caption text-text-muted">{r.resourceCount} resources</div>
                    )}
                  </div>
                </div>
              ))}
            </CollapsibleSection>
          )}

          {/* LTI/LMS Results */}
          {data.educationResults?.ltiResults && data.educationResults.ltiResults.length > 0 && (
            <CollapsibleSection
              title="LTI/LMS"
              count={data.educationResults.ltiResults.length}
              variant="compact"
              defaultOpen={true}
              status={
                data.educationResults.ltiResults.some((r) => !r.success) ? "error" : "success"
              }
            >
              {data.educationResults.ltiResults.map((r) => (
                <div
                  key={`lti-${r.name}`}
                  className={cn(
                    layout.flex.between,
                    spacing.pad.xs,
                    radius.default,
                    !r.success ? "bg-status-error/10" : "bg-surface-raised",
                  )}
                >
                  <div className={layout.stack.compact}>
                    <div className="flex items-center gap-2">
                      <StatusBadge status={r.success ? "success" : "error"} />
                      <span className="body-small font-medium">{r.name}</span>
                      <span
                        className="caption text-text-muted truncate max-w-48"
                        title={r.launchUrl}
                      >
                        {r.launchUrl}
                      </span>
                    </div>
                    {r.success && r.ltiVersion && (
                      <span className="caption text-text-muted ml-6">LTI {r.ltiVersion}</span>
                    )}
                    {r.error && <span className="caption text-status-error ml-6">{r.error}</span>}
                  </div>
                  <div className="text-right">
                    <div className="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
                  </div>
                </div>
              ))}
            </CollapsibleSection>
          )}

          {/* OPC-UA Results */}
          {data.industrialResults?.opcuaResults &&
            data.industrialResults.opcuaResults.length > 0 && (
              <CollapsibleSection
                title="OPC-UA"
                count={data.industrialResults.opcuaResults.length}
                variant="compact"
                defaultOpen={true}
                status={
                  data.industrialResults.opcuaResults.some((r) => !r.success) ? "error" : "success"
                }
              >
                {data.industrialResults.opcuaResults.map((r) => (
                  <div
                    key={`opcua-${r.name}`}
                    className={cn(
                      layout.flex.between,
                      spacing.pad.xs,
                      radius.default,
                      !r.success ? "bg-status-error/10" : "bg-surface-raised",
                    )}
                  >
                    <div className={layout.stack.compact}>
                      <div className="flex items-center gap-2">
                        <StatusBadge status={r.success ? "success" : "error"} />
                        <span className="body-small font-medium">{r.name}</span>
                        <span
                          className="caption text-text-muted truncate max-w-48"
                          title={r.endpointUrl}
                        >
                          {r.endpointUrl}
                        </span>
                      </div>
                      {r.success && (r.securityMode || r.productName) && (
                        <span className="caption text-text-muted ml-6">
                          {r.securityMode && r.securityMode}
                          {r.securityMode && r.productName && " • "}
                          {r.productName && r.productName}
                        </span>
                      )}
                      {r.error && <span className="caption text-status-error ml-6">{r.error}</span>}
                    </div>
                    <div className="text-right">
                      <div className="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
                      {r.success && r.serverState && (
                        <div className="caption text-text-muted">{r.serverState}</div>
                      )}
                    </div>
                  </div>
                ))}
              </CollapsibleSection>
            )}

          {/* Modbus TCP Results */}
          {data.industrialResults?.modbusResults &&
            data.industrialResults.modbusResults.length > 0 && (
              <CollapsibleSection
                title="Modbus TCP"
                count={data.industrialResults.modbusResults.length}
                variant="compact"
                defaultOpen={true}
                status={
                  data.industrialResults.modbusResults.some((r) => !r.success) ? "error" : "success"
                }
              >
                {data.industrialResults.modbusResults.map((r) => (
                  <div
                    key={`modbus-${r.name}`}
                    className={cn(
                      layout.flex.between,
                      spacing.pad.xs,
                      radius.default,
                      !r.success ? "bg-status-error/10" : "bg-surface-raised",
                    )}
                  >
                    <div className={layout.stack.compact}>
                      <div className="flex items-center gap-2">
                        <StatusBadge status={r.success ? "success" : "error"} />
                        <span className="body-small font-medium">{r.name}</span>
                        <span className="caption text-text-muted">
                          {r.host}:{r.port} • Unit {r.unitId}
                        </span>
                      </div>
                      {r.error && <span className="caption text-status-error ml-6">{r.error}</span>}
                    </div>
                    <div className="text-right">
                      <div className="body-small font-mono">{r.totalTimeMs.toFixed(1)}ms</div>
                      {r.success && r.registerValue !== undefined && (
                        <div className="caption text-text-muted">
                          Reg: 0x{r.registerValue.toString(16).toUpperCase().padStart(4, "0")}
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </CollapsibleSection>
            )}
        </>
      )}

      {error && <p className="body-small text-status-error">{error}</p>}
    </Card>
  );
});
