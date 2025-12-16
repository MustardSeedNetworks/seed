/**
 * PerformanceCard Component
 *
 * Purpose: Internet speed and network performance testing with speedtest.net integration and
 * iperf3 support. Displays download/upload speeds, latency, jitter, and packet loss metrics.
 * (~670 lines - complex testing orchestration)
 *
 * Key Features:
 * - Speedtest.net integration: download/upload speeds, latency, server info, distance
 * - iperf3 support: UDP and TCP throughput testing, bandwidth, jitter, packet loss
 * - Real-time progress: shows test phase and progress percentage
 * - Multiple test result history: displays last completed test results
 * - Visual gauges: SpeedGauge components for visual speed representation
 * - Protocol metrics: separate IPv4/IPv6 results (when available)
 * - Test controls: start/stop test, clear results
 * - Latency thresholds: warning/critical levels from settings
 * - System info: iperf3 installation status and version
 *
 * Usage:
 * ```typescript
 * <PerformanceCard
 *   data={performanceData}
 *   loading={isRunningTest}
 * />
 * ```
 *
 * Dependencies: Card UI components, SpeedGauge, useSettings hook, auth hooks,
 *              Icons, theme utilities
 * State: Manages test state, results history, current phase, progress tracking
 */

import { useState, useEffect, useCallback, memo } from "react";
import { useTranslation } from "react-i18next";
import { Card, CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { getAuthHeaders } from "../../hooks/useAuth";
import { useSettings } from "../../contexts/useSettings";
import { Gauge } from "../ui/Icons";
import { SpeedGauge, ProgressRing, PulsingDot } from "../ui/SpeedGauge";
import { icon as iconTokens, layout, radius, spacing } from "../../styles/theme";

// Speedtest types
interface SpeedtestData {
  download: number;
  upload: number;
  latency: number;
  server: string;
  location: string;
  host: string;
  distance: number;
  timestamp: string;
  testDuration: number;
}

interface SpeedtestStatus {
  running: boolean;
  phase: string;
  progress: number;
  last?: SpeedtestData;
}

// iperf3 types
interface IperfInfo {
  installed: boolean;
  version?: string;
  error?: string;
}

interface IperfResult {
  bandwidth: number;
  transfer: number;
  retransmits: number;
  jitter: number;
  lostPackets: number;
  lostPercent: number;
  protocol: string;
  direction: string;
  duration: number;
  server: string;
  port: number;
  timestamp: string;
  downloadBandwidth?: number;
  uploadBandwidth?: number;
  downloadTransfer?: number;
  uploadTransfer?: number;
}

interface IperfClientStatus {
  running: boolean;
  phase: string;
  progress: number;
  last?: IperfResult;
}

interface IperfServerStatus {
  running: boolean;
  port: number;
  pid: number;
  error?: string;
}

interface PerformanceCardProps {
  loading?: boolean;
  runSpeedtestEnabled?: boolean;
  runIperfEnabled?: boolean;
}

// Phase label translation keys
type SpeedtestPhase =
  | "idle"
  | "finding_server"
  | "testing_latency"
  | "testing_download"
  | "testing_upload"
  | "complete";
type IperfPhase = "idle" | "connecting" | "testing" | "complete";

export const PerformanceCard = memo(function PerformanceCard({
  loading,
  runSpeedtestEnabled = true,
  runIperfEnabled = true,
}: PerformanceCardProps) {
  const { t } = useTranslation("cards");
  // Get iperf settings from context
  const { iperfSettings } = useSettings();

  // Helper to get speedtest phase label
  const getSpeedtestPhaseLabel = (phase: string): string => {
    switch (phase as SpeedtestPhase) {
      case "idle":
        return t("performance.phaseReady");
      case "finding_server":
        return t("performance.phaseFindingServer");
      case "testing_latency":
        return t("performance.phaseTestingLatency");
      case "testing_download":
        return t("performance.phaseTestingDownload");
      case "testing_upload":
        return t("performance.phaseTestingUpload");
      case "complete":
        return t("performance.phaseComplete");
      default:
        return phase;
    }
  };

  // Helper to get iperf phase label
  const getIperfPhaseLabel = (phase: string): string => {
    switch (phase as IperfPhase) {
      case "idle":
        return t("performance.phaseReady");
      case "connecting":
        return t("performance.phaseConnecting");
      case "testing":
        return t("performance.phaseTesting");
      case "complete":
        return t("performance.phaseComplete");
      default:
        return phase;
    }
  };

  // Speedtest state
  const [speedtestStatus, setSpeedtestStatus] = useState<SpeedtestStatus | null>(null);
  const [speedtestResult, setSpeedtestResult] = useState<SpeedtestData | null>(null);
  const [speedtestError, setSpeedtestError] = useState<string | null>(null);
  const [speedtestRunning, setSpeedtestRunning] = useState(false);

  // iperf3 state
  const [iperfInfo, setIperfInfo] = useState<IperfInfo | null>(null);
  const [iperfClientStatus, setIperfClientStatus] = useState<IperfClientStatus | null>(null);
  const [iperfResult, setIperfResult] = useState<IperfResult | null>(null);
  const [iperfServerStatus, setIperfServerStatus] = useState<IperfServerStatus | null>(null);
  const [iperfError, setIperfError] = useState<string | null>(null);
  const [iperfClientRunning, setIperfClientRunning] = useState(false);

  // Start/stop iperf server based on settings
  const manageIperfServer = useCallback(async (shouldRun: boolean, port: number) => {
    try {
      const action = shouldRun ? "start" : "stop";
      const res = await fetch("/api/iperf/server", {
        method: "POST",
        headers: {
          ...getAuthHeaders(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ action, port }),
      });
      if (res.ok) {
        const statusRes = await fetch("/api/iperf/server/status", {
          headers: getAuthHeaders(),
        });
        if (statusRes.ok) {
          setIperfServerStatus(await statusRes.json());
        }
      }
    } catch (err) {
      console.error("Failed to manage iperf server:", err);
    }
  }, []);

  // Fetch initial status
  useEffect(() => {
    const fetchStatus = async () => {
      try {
        // Fetch speedtest status
        const speedRes = await fetch("/api/speedtest/status", {
          headers: getAuthHeaders(),
        });
        if (speedRes.ok) {
          const data = await speedRes.json();
          setSpeedtestStatus(data);
          if (data.last) {
            setSpeedtestResult(data.last);
          }
          setSpeedtestRunning(data.running);
        }

        // Fetch iperf3 info
        const iperfInfoRes = await fetch("/api/iperf/info", {
          headers: getAuthHeaders(),
        });
        if (iperfInfoRes.ok) {
          setIperfInfo(await iperfInfoRes.json());
        }

        // Fetch iperf3 client status
        const iperfClientRes = await fetch("/api/iperf/client/status", {
          headers: getAuthHeaders(),
        });
        if (iperfClientRes.ok) {
          const data = await iperfClientRes.json();
          setIperfClientStatus(data);
          if (data.last) {
            setIperfResult(data.last);
          }
          setIperfClientRunning(data.running);
        }

        // Fetch iperf3 server status
        const iperfServerRes = await fetch("/api/iperf/server/status", {
          headers: getAuthHeaders(),
        });
        if (iperfServerRes.ok) {
          setIperfServerStatus(await iperfServerRes.json());
        }
      } catch (err) {
        console.error("Failed to fetch performance status:", err);
      }
    };
    fetchStatus();
  }, []);

  // Track previous server settings to detect changes
  const prevServerSettings = useState({
    enableServer: iperfSettings.enableServer,
    serverPort: iperfSettings.serverPort,
  })[0];

  // Manage iperf server based on settings from context
  useEffect(() => {
    // Only manage server on settings changes after iperf3 is confirmed installed
    if (!iperfInfo?.installed) return;

    // Check if server settings changed
    if (
      iperfSettings.enableServer !== prevServerSettings.enableServer ||
      iperfSettings.serverPort !== prevServerSettings.serverPort
    ) {
      manageIperfServer(iperfSettings.enableServer, iperfSettings.serverPort);
      prevServerSettings.enableServer = iperfSettings.enableServer;
      prevServerSettings.serverPort = iperfSettings.serverPort;
    }
  }, [
    iperfSettings.enableServer,
    iperfSettings.serverPort,
    iperfInfo?.installed,
    manageIperfServer,
    prevServerSettings,
  ]);

  // Poll speedtest status while running
  useEffect(() => {
    if (!speedtestRunning) return;

    const interval = setInterval(async () => {
      try {
        const res = await fetch("/api/speedtest/status", {
          headers: getAuthHeaders(),
        });
        if (res.ok) {
          const data = await res.json();
          setSpeedtestStatus(data);
          if (!data.running) {
            setSpeedtestRunning(false);
            if (data.last) {
              setSpeedtestResult(data.last);
            }
            // Signal FAB that speedtest is complete
            window.dispatchEvent(
              new CustomEvent("cardTestComplete", {
                detail: { test: "speedtest" },
              })
            );
          }
        }
      } catch (err) {
        console.error("Failed to poll speedtest status:", err);
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [speedtestRunning]);

  // Poll iperf3 client status while running
  useEffect(() => {
    if (!iperfClientRunning) return;

    const interval = setInterval(async () => {
      try {
        const res = await fetch("/api/iperf/client/status", {
          headers: getAuthHeaders(),
        });
        if (res.ok) {
          const data = await res.json();
          setIperfClientStatus(data);
          if (!data.running) {
            setIperfClientRunning(false);
            if (data.last) {
              setIperfResult(data.last);
            }
            // Signal FAB that iperf is complete
            window.dispatchEvent(
              new CustomEvent("cardTestComplete", {
                detail: { test: "iperf" },
              })
            );
          }
        }
      } catch (err) {
        console.error("Failed to poll iperf status:", err);
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [iperfClientRunning]);

  const runSpeedtest = useCallback(async () => {
    if (!runSpeedtestEnabled) {
      setSpeedtestError(t("performance.testsDisabled"));
      return;
    }

    setSpeedtestError(null);
    setSpeedtestRunning(true);
    setSpeedtestStatus({ running: true, phase: "finding_server", progress: 0 });

    try {
      const res = await fetch("/api/speedtest", {
        method: "POST",
        headers: getAuthHeaders(),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || t("performance.speedtestFailed"));
      }
    } catch (err) {
      setSpeedtestError(err instanceof Error ? err.message : t("performance.speedtestFailed"));
      setSpeedtestStatus({ running: false, phase: "idle", progress: 0 });
      setSpeedtestRunning(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- t is stable from react-i18next
  }, [runSpeedtestEnabled]);

  const runIperfClient = useCallback(async () => {
    if (!runIperfEnabled) {
      setIperfError(t("performance.testsDisabled"));
      return;
    }

    if (!iperfSettings.server) {
      setIperfError(t("performance.serverNotConfigured"));
      return;
    }

    setIperfError(null);
    setIperfClientRunning(true);
    setIperfClientStatus({ running: true, phase: "connecting", progress: 0 });

    try {
      const res = await fetch("/api/iperf/client", {
        method: "POST",
        headers: {
          ...getAuthHeaders(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          server: iperfSettings.server,
          port: iperfSettings.port,
          protocol: iperfSettings.protocol,
          direction: iperfSettings.direction,
          reverse: iperfSettings.direction === "download",
          duration: iperfSettings.duration,
          parallel: 1,
        }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || "iperf3 test failed");
      }
    } catch (err) {
      setIperfError(err instanceof Error ? err.message : "iperf3 test failed");
      setIperfClientStatus({ running: false, phase: "idle", progress: 0 });
      setIperfClientRunning(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- t is stable from react-i18next
  }, [iperfSettings, runIperfEnabled]);

  // Listen for FAB "run all tests" event
  useEffect(() => {
    const handleRunAllTests = () => {
      // Run speedtest if enabled
      if (runSpeedtestEnabled && !speedtestRunning) {
        runSpeedtest();
      }
      // Run iperf client test if enabled and configured
      if (runIperfEnabled && !iperfClientRunning && iperfSettings.server && iperfInfo?.installed) {
        // Delay slightly so tests don't all hammer at once
        setTimeout(() => runIperfClient(), 500);
      }
    };

    window.addEventListener("runAllTests", handleRunAllTests);
    return () => {
      window.removeEventListener("runAllTests", handleRunAllTests);
    };
  }, [
    runSpeedtest,
    runIperfClient,
    speedtestRunning,
    iperfClientRunning,
    iperfSettings.server,
    iperfInfo?.installed,
    runSpeedtestEnabled,
    runIperfEnabled,
  ]);

  const formatSpeed = (mbps: number): string => {
    if (mbps >= 1000) {
      return `${(mbps / 1000).toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })} Gbps`;
    }
    return `${mbps.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })} Mbps`;
  };

  const getStatus = (): Status => {
    if (!runSpeedtestEnabled && !runIperfEnabled) return "unknown";
    if (loading || speedtestRunning || iperfClientRunning) return "loading";
    if (speedtestError || iperfError) return "error";
    if (speedtestResult || iperfResult) return "success";
    return "unknown";
  };

  return (
    <Card
      title={t("performance.title")}
      subtitle={t("performance.subtitle")}
      icon={<Gauge className={iconTokens.size.md} />}
      status={getStatus()}
    >
      <div>
        {/* Internet Speed Section */}
        <p className="caption font-medium mb-2">{t("performance.internetSpeed")}</p>

        {speedtestRunning && speedtestStatus && (
          <div className={`${layout.inline.spacious} mb-3 pad-sm bg-surface-hover ${radius.lg}`}>
            <ProgressRing progress={speedtestStatus.progress} size={56} strokeWidth={5} />
            <div className="flex-1">
              <div className={layout.inline.default}>
                <PulsingDot color="primary" size="sm" />
                <span className="body-small font-medium">
                  {getSpeedtestPhaseLabel(speedtestStatus.phase)}
                </span>
              </div>
              {(() => {
                const sp = Math.min(Math.max(speedtestStatus.progress, 0), 100);
                return (
                  <progress
                    value={sp}
                    max={100}
                    aria-label={t("performance.progress")}
                    className={`mt-2 w-full ${radius.full}`}
                  />
                );
              })()}
            </div>
          </div>
        )}

        {!speedtestRunning && speedtestResult && (
          <div className="mb-3">
            <div className={`${layout.flex.center} ${spacing.gap.spacious} py-2`}>
              <SpeedGauge
                value={speedtestResult.download}
                label={t("performance.download")}
                size="md"
              />
              <SpeedGauge
                value={speedtestResult.upload}
                label={t("performance.upload")}
                size="md"
              />
            </div>
            <CardRow
              label={t("performance.latency")}
              value={`${speedtestResult.latency.toFixed(0)} ms`}
            />
            <CardRow label={t("performance.server")} value={speedtestResult.location} />
          </div>
        )}

        {!speedtestRunning && !speedtestResult && !speedtestError && (
          <p className="body-small mb-2">{t("performance.noResults")}</p>
        )}

        {speedtestError && <p className="body-small text-status-error">{speedtestError}</p>}

        <CardDivider />

        {/* LAN Speed (iperf3) Section */}
        <p className="caption font-medium mb-2 mt-2">
          {t("performance.lanSpeed")}
          {iperfInfo?.version && (
            <span className="text-text-muted font-normal ml-2">{iperfInfo.version}</span>
          )}
        </p>

        {!iperfInfo?.installed && (
          <p className="body-small text-status-warning mb-3">
            {t("performance.iperfNotInstalled")}
          </p>
        )}

        {iperfInfo?.installed && (
          <>
            {/* Config Summary */}
            {iperfSettings.server ? (
              <div className={`caption mb-3 pad-sm bg-surface-hover ${radius.default}`}>
                <div className={layout.flex.between}>
                  <span>{t("performance.server")}:</span>
                  <span className="text-text-primary">
                    {iperfSettings.server}:{iperfSettings.port}
                  </span>
                </div>
                <div className={layout.flex.between}>
                  <span>{t("performance.test")}:</span>
                  <span className="text-text-primary">
                    {iperfSettings.protocol.toUpperCase()}{" "}
                    {iperfSettings.direction === "bidirectional"
                      ? t("performance.both")
                      : iperfSettings.direction}
                  </span>
                </div>
              </div>
            ) : (
              <p className="caption mb-3">{t("performance.configureServer")}</p>
            )}

            {/* Client Status/Results */}
            {iperfClientRunning && iperfClientStatus && (
              <div
                className={`${layout.inline.spacious} mb-3 pad-sm bg-surface-hover ${radius.lg}`}
              >
                <ProgressRing progress={iperfClientStatus.progress} size={56} strokeWidth={5} />
                <div className="flex-1">
                  <div className={layout.inline.default}>
                    <PulsingDot color="primary" size="sm" />
                    <span className="body-small font-medium">
                      {getIperfPhaseLabel(iperfClientStatus.phase)}
                    </span>
                  </div>
                  {(() => {
                    const pp = Math.min(Math.max(iperfClientStatus.progress, 0), 100);
                    return (
                      <progress
                        value={pp}
                        max={100}
                        aria-label="iPerf progress"
                        className={`mt-2 w-full ${radius.full}`}
                      />
                    );
                  })()}
                </div>
              </div>
            )}

            {!iperfClientRunning && iperfResult && (
              <div className="mb-3 stack-sm">
                {iperfResult.direction === "bidirectional" ? (
                  <div className={`grid grid-cols-1 sm:grid-cols-2 ${spacing.gap.default}`}>
                    <CardValue
                      label={t("performance.download")}
                      value={formatSpeed(iperfResult.downloadBandwidth ?? iperfResult.bandwidth)}
                      size="md"
                      status="success"
                    />
                    <CardValue
                      label={t("performance.upload")}
                      value={formatSpeed(iperfResult.uploadBandwidth ?? iperfResult.bandwidth)}
                      size="md"
                      status="success"
                    />
                  </div>
                ) : (
                  <CardValue
                    label={
                      iperfResult.direction === "download"
                        ? t("performance.download")
                        : t("performance.upload")
                    }
                    value={formatSpeed(iperfResult.bandwidth)}
                    size="md"
                    status="success"
                  />
                )}

                {iperfResult.direction === "bidirectional" ? (
                  <div className={`grid grid-cols-1 sm:grid-cols-2 ${spacing.gap.default}`}>
                    {iperfResult.downloadTransfer !== undefined && (
                      <CardRow
                        label={t("performance.downloadTransfer")}
                        value={`${iperfResult.downloadTransfer.toFixed(1)} MB`}
                      />
                    )}
                    {iperfResult.uploadTransfer !== undefined && (
                      <CardRow
                        label={t("performance.uploadTransfer")}
                        value={`${iperfResult.uploadTransfer.toFixed(1)} MB`}
                      />
                    )}
                  </div>
                ) : (
                  <CardRow
                    label={t("performance.transfer")}
                    value={`${iperfResult.transfer.toFixed(1)} MB`}
                  />
                )}

                {iperfResult.protocol === "tcp" && iperfResult.retransmits > 0 && (
                  <CardRow
                    label={t("performance.retransmits")}
                    value={iperfResult.retransmits.toString()}
                  />
                )}
                {iperfResult.protocol === "udp" && (
                  <>
                    <CardRow
                      label={t("performance.jitter")}
                      value={`${iperfResult.jitter.toFixed(2)} ms`}
                    />
                    <CardRow
                      label={t("performance.packetLoss")}
                      value={`${iperfResult.lostPercent.toFixed(2)}%`}
                    />
                  </>
                )}
              </div>
            )}

            {iperfError && <p className="body-small text-status-error">{iperfError}</p>}

            {/* Server status indicator (if enabled) */}
            {iperfSettings.enableServer && (
              <div
                className={`caption ${layout.flex.between} pad-sm bg-surface-hover ${radius.default}`}
              >
                <span>{t("performance.serverMode")}</span>
                <span
                  className={iperfServerStatus?.running ? "text-status-success" : "text-text-muted"}
                >
                  {iperfServerStatus?.running
                    ? t("performance.listening", { port: iperfServerStatus.port })
                    : t("performance.stopped")}
                </span>
              </div>
            )}
          </>
        )}
      </div>
    </Card>
  );
});
