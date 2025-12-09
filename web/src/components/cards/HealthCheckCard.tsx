import { useState, useEffect, useCallback } from "react";
import { Card, Status } from "../ui/Card";
import { StatusBadge } from "../ui/StatusBadge";
import { CollapsibleSection } from "../ui/CollapsibleSection";
import { Tooltip } from "../ui/Tooltip";
import { getAuthHeaders } from "../../hooks/useAuth";
import { HTTP_TIMING_HELP } from "../help/HelpContent";

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

export function HealthCheckCard({ loading }: HealthCheckCardProps) {
  const [data, setData] = useState<HealthCheckData | null>(null);
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTests = useCallback(async () => {
    setIsRunning(true);
    setError(null);
    try {
      const res = await fetch("/api/tests/run", {
        headers: getAuthHeaders(),
      });
      if (res.ok) {
        const result = await res.json();
        setData(result);
      } else {
        setError("Failed to run tests");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to run tests");
    } finally {
      setIsRunning(false);
    }
  }, []);

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
        handleHealthChecksUpdated,
      );
    };
  }, [fetchTests]);

  // Listen for FAB "run all tests" event
  useEffect(() => {
    const handleRunAllTests = async () => {
      // Check FAB options from localStorage
      try {
        const saved = localStorage.getItem("netscope-fab-options");
        if (saved) {
          const fabOptions = JSON.parse(saved);
          if (fabOptions.runHealthChecks === false) {
            return; // Skip health checks if disabled
          }
        }
      } catch (err) {
        console.error("Failed to read FAB options:", err);
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
  }, [fetchTests, isRunning]);

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
          !r.success || r.testStatus === "error" || r.certStatus === "error",
      )
    ) {
      return "error";
    }

    // Any warning status = card is warning
    if (
      allResults.some(
        (r) => r.testStatus === "warning" || r.certStatus === "warning",
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
    type: "ping" | "tcp" | "udp" | "http",
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
      <div key={`${type}-${result.name}`} className="py-1">
        <div className="flex items-center justify-between">
          <span
            className="text-sm text-text-muted truncate flex-1"
            title={displayName}
          >
            {displayName}
            {details}
          </span>
          <span className="inline-flex items-center gap-2">
            <StatusBadge status={statusLabel} size="sm" />
            <span className={`text-sm font-medium ${statusColor}`}>
              {result.success ? formatLatency(result.latency) : "fail"}
            </span>
          </span>
        </div>
        {extendedInfo && (
          <div className="text-xs text-text-muted mt-0.5">{extendedInfo}</div>
        )}
      </div>
    );
  };

  // Timing bar component for HTTP requests
  const TimingBar = ({ result }: { result: TestResult }) => {
    const total = result.latency || 0;
    // Guard against NaN, Infinity, and zero/negative values
    if (!total || !Number.isFinite(total) || total <= 0) return null;

    // Safely extract timing values, treating NaN/undefined as 0
    const safeNum = (v: number | undefined) =>
      v !== undefined && Number.isFinite(v) ? v : 0;
    const dns = safeNum(result.dnsLatency);
    const tcp = safeNum(result.tcpConnect);
    const tls = safeNum(result.tlsLatency);
    const ttfb = safeNum(result.ttfbLatency);
    // Download time is what's left after subtracting known phases
    const download = Math.max(0, total - dns - tcp - tls - ttfb);

    // Get status-based colors for each segment
    const getStatusColor = (status?: StatusValue, defaultColor?: string) => {
      if (status === "error") return "bg-status-error";
      if (status === "warning") return "bg-status-warning";
      return defaultColor || "bg-status-success";
    };

    const getStatusTextColor = (status?: StatusValue) => {
      if (status === "error") return "text-status-error";
      if (status === "warning") return "text-status-warning";
      return "text-text-muted";
    };

    const segments = [
      {
        label: "DNS",
        value: dns,
        color: getStatusColor(result.dnsStatus, "bg-blue-400"),
        status: result.dnsStatus,
      },
      {
        label: "TCP",
        value: tcp,
        color: getStatusColor(result.tcpStatus, "bg-cyan-400"),
        status: result.tcpStatus,
      },
      {
        label: "TLS",
        value: tls,
        color: getStatusColor(result.tlsStatus, "bg-purple-400"),
        status: result.tlsStatus,
      },
      {
        label: "Wait",
        value: ttfb,
        color: getStatusColor(result.ttfbStatus, "bg-amber-400"),
        status: result.ttfbStatus,
      },
      {
        label: "Download",
        value: download,
        color: "bg-green-400",
        status: undefined,
      },
    ].filter((s) => s.value > 0 && Number.isFinite(s.value));

    if (segments.length === 0) return null;

    const fmt = (ms: number) =>
      ms >= 1000 ? `${(ms / 1000).toFixed(1)}s` : `${Math.round(ms)}ms`;

    return (
      <div className="mt-1.5">
        {/* Stacked bar */}
        <div className="h-2 rounded-full overflow-hidden flex bg-bg-tertiary">
          {segments.map((seg, i) => (
            <div
              key={seg.label}
              className={`${seg.color} ${i === 0 ? "rounded-l-full" : ""} ${i === segments.length - 1 ? "rounded-r-full" : ""}`}
              style={{
                width: `${Math.min(100, Math.max(0, (seg.value / total) * 100))}%`,
              }}
              title={`${seg.label}: ${fmt(seg.value)}${seg.status && seg.status !== "success" ? ` (${seg.status})` : ""}`}
            />
          ))}
        </div>
        {/* Legend with tooltips */}
        <div className="flex flex-wrap gap-x-3 gap-y-0.5 mt-1 text-xs">
          {segments.map((seg) => (
            <Tooltip
              key={seg.label}
              content={HTTP_TIMING_HELP[seg.label] || seg.label}
              position="bottom"
            >
              <span
                className={`inline-flex items-center gap-1 ${getStatusTextColor(seg.status)}`}
              >
                <span
                  className={`inline-block w-2 h-2 rounded-full ${seg.color}`}
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
      if (days <= 0) return "EXPIRED";
      if (days === 1) return "1 day";
      if (days < 30) return `${days} days`;
      if (days < 365) return `${Math.floor(days / 30)}mo`;
      return `${Math.floor(days / 365)}y`;
    };

    // Check if we have timing breakdown data
    const hasTimingData =
      result.dnsLatency !== undefined ||
      result.tcpConnect !== undefined ||
      result.tlsLatency !== undefined ||
      result.ttfbLatency !== undefined;

    return (
      <div key={`http-${result.name}`} className="py-1.5">
        <div className="flex items-center justify-between">
          <span
            className="text-sm text-text-muted truncate flex-1"
            title={result.name}
          >
            {result.name}
            {result.status ? ` (${result.status})` : ""}
          </span>
          <span className={`text-sm font-medium ${statusColor}`}>
            {result.success ? formatLatency(result.latency) : "fail"}
          </span>
        </div>
        {hasTimingData && result.success && <TimingBar result={result} />}
        {(hasTLS || hasCertInfo) && (
          <div className="text-xs mt-1 flex items-center gap-2">
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
    <Card title="Health Checks" status={getStatus()}>
      {isRunning && <p className="text-sm text-text-muted">Running tests...</p>}

      {!isRunning && data && (
        <>
          {/* Ping Results */}
          {data.pingResults.length > 0 && (
            <CollapsibleSection
              title="Ping"
              count={data.pingResults.length}
              variant="compact"
              defaultOpen={true}
              status={
                data.pingResults.some(
                  (r) => !r.success || r.testStatus === "error",
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
          {data.tcpResults.length > 0 && (
            <CollapsibleSection
              title="TCP Ports"
              count={data.tcpResults.length}
              variant="compact"
              defaultOpen={true}
              status={
                data.tcpResults.some(
                  (r) => !r.success || r.testStatus === "error",
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
              title="UDP Ports"
              count={data.udpResults.length}
              variant="compact"
              defaultOpen={true}
              status={
                data.udpResults.some(
                  (r) => !r.success || r.testStatus === "error",
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
          {data.httpResults.length > 0 && (
            <CollapsibleSection
              title="HTTP"
              count={data.httpResults.length}
              variant="compact"
              defaultOpen={true}
              status={
                data.httpResults.some(
                  (r) =>
                    !r.success ||
                    r.testStatus === "error" ||
                    r.certStatus === "error",
                )
                  ? "error"
                  : data.httpResults.some(
                        (r) =>
                          r.testStatus === "warning" ||
                          r.certStatus === "warning",
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

      {error && <p className="text-sm text-status-error">{error}</p>}
    </Card>
  );
}
