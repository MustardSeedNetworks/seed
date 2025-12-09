import { useState, useEffect, useCallback } from "react";
import { Card, CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { getAuthHeaders } from "../../hooks/useAuth";

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
}

interface IperfSettings {
  server: string;
  port: number;
  protocol: "tcp" | "udp";
  direction: "upload" | "download" | "bidirectional";
  duration: number;
  serverPort: number;
  enableServer: boolean;
}

const speedtestPhaseLabels: Record<string, string> = {
  idle: "Ready",
  finding_server: "Finding server...",
  testing_latency: "Testing latency...",
  testing_download: "Testing download...",
  testing_upload: "Testing upload...",
  complete: "Complete",
};

const iperfPhaseLabels: Record<string, string> = {
  idle: "Ready",
  connecting: "Connecting...",
  testing: "Testing...",
  complete: "Complete",
};

export function PerformanceCard({ loading }: PerformanceCardProps) {
  // Speedtest state
  const [speedtestStatus, setSpeedtestStatus] =
    useState<SpeedtestStatus | null>(null);
  const [speedtestResult, setSpeedtestResult] = useState<SpeedtestData | null>(
    null,
  );
  const [speedtestError, setSpeedtestError] = useState<string | null>(null);
  const [speedtestRunning, setSpeedtestRunning] = useState(false);

  // iperf3 state
  const [iperfInfo, setIperfInfo] = useState<IperfInfo | null>(null);
  const [iperfClientStatus, setIperfClientStatus] =
    useState<IperfClientStatus | null>(null);
  const [iperfResult, setIperfResult] = useState<IperfResult | null>(null);
  const [iperfServerStatus, setIperfServerStatus] =
    useState<IperfServerStatus | null>(null);
  const [iperfError, setIperfError] = useState<string | null>(null);
  const [iperfClientRunning, setIperfClientRunning] = useState(false);
  const [runPerformanceEnabled, setRunPerformanceEnabled] = useState(true);

  // iperf3 settings (loaded from localStorage/Settings)
  const [iperfSettings, setIperfSettings] = useState<IperfSettings>({
    server: "",
    port: 5201,
    protocol: "tcp",
    direction: "download",
    duration: 10,
    serverPort: 5201,
    enableServer: false,
  });

  // Start/stop iperf server based on settings
  const manageIperfServer = useCallback(
    async (shouldRun: boolean, port: number) => {
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
    },
    [],
  );

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

  // Track master performance enable toggle from Settings
  useEffect(() => {
    try {
      const saved = localStorage.getItem("netscope-run-performance");
      if (saved !== null) {
        setRunPerformanceEnabled(JSON.parse(saved));
      }
    } catch (err) {
      console.error("Failed to read performance toggle:", err);
    }

    const handlePerfToggle = (e: Event) => {
      const detail = (e as CustomEvent<boolean>).detail;
      setRunPerformanceEnabled((prev) =>
        typeof detail === "boolean" ? detail : prev,
      );
    };

    window.addEventListener(
      "performanceToggleUpdated",
      handlePerfToggle as EventListener,
    );
    return () => {
      window.removeEventListener(
        "performanceToggleUpdated",
        handlePerfToggle as EventListener,
      );
    };
  }, []);

  // Load iperf settings from localStorage and listen for updates
  useEffect(() => {
    const loadSettings = () => {
      try {
        const saved = localStorage.getItem("netscope-iperf-settings");
        if (saved) {
          const parsed = JSON.parse(saved) as Partial<IperfSettings>;
          setIperfSettings((prev) => {
            const newSettings = { ...prev, ...parsed };
            // Auto-start server if enabled
            if (newSettings.enableServer && iperfInfo?.installed) {
              manageIperfServer(true, newSettings.serverPort);
            }
            return newSettings;
          });
        }
      } catch (err) {
        console.error("Failed to load iperf settings:", err);
      }
    };

    loadSettings();

    // Listen for settings updates from SettingsDrawer
    const handleSettingsUpdate = (e: CustomEvent<IperfSettings>) => {
      setIperfSettings((prev) => {
        const newSettings = { ...prev, ...e.detail };
        // Manage server based on enableServer setting
        if (
          newSettings.enableServer !== prev.enableServer ||
          newSettings.serverPort !== prev.serverPort
        ) {
          manageIperfServer(newSettings.enableServer, newSettings.serverPort);
        }
        return newSettings;
      });
    };

    window.addEventListener(
      "iperfSettingsUpdated",
      handleSettingsUpdate as EventListener,
    );
    return () => {
      window.removeEventListener(
        "iperfSettingsUpdated",
        handleSettingsUpdate as EventListener,
      );
    };
  }, [manageIperfServer, iperfInfo?.installed]);

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
              }),
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
              }),
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
    if (!runPerformanceEnabled) {
      setSpeedtestError("Performance tests are disabled in Settings");
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
        throw new Error(text || "Speedtest failed");
      }
    } catch (err) {
      setSpeedtestError(
        err instanceof Error ? err.message : "Speedtest failed",
      );
      setSpeedtestStatus({ running: false, phase: "idle", progress: 0 });
      setSpeedtestRunning(false);
    }
  }, [runPerformanceEnabled]);

  const runIperfClient = useCallback(async () => {
    if (!runPerformanceEnabled) {
      setIperfError("Performance tests are disabled in Settings");
      return;
    }

    if (!iperfSettings.server) {
      setIperfError("Server not configured");
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
  }, [iperfSettings, runPerformanceEnabled]);

  // Listen for FAB "run all tests" event
  useEffect(() => {
    const handleRunAllTests = () => {
      if (!runPerformanceEnabled) {
        return;
      }
      // Check FAB options from localStorage
      let fabOptions = {
        runSpeedtest: true, // Default ON
        runIperf: true, // Default ON
      };
      try {
        const saved = localStorage.getItem("netscope-fab-options");
        if (saved) {
          const parsed = JSON.parse(saved);
          fabOptions = { ...fabOptions, ...parsed };
        }
      } catch (err) {
        console.error("Failed to read FAB options:", err);
      }

      // Run speedtest if enabled
      if (fabOptions.runSpeedtest && !speedtestRunning) {
        runSpeedtest();
      }
      // Run iperf client test if enabled and configured
      if (
        fabOptions.runIperf &&
        !iperfClientRunning &&
        iperfSettings.server &&
        iperfInfo?.installed
      ) {
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
    runPerformanceEnabled,
  ]);

  const formatSpeed = (mbps: number): string => {
    if (mbps >= 1000) {
      return `${(mbps / 1000).toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })} Gbps`;
    }
    return `${mbps.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })} Mbps`;
  };

  const getStatus = (): Status => {
    if (!runPerformanceEnabled) return "unknown";
    if (loading || speedtestRunning || iperfClientRunning) return "loading";
    if (speedtestError || iperfError) return "error";
    if (speedtestResult || iperfResult) return "success";
    return "unknown";
  };

  return (
    <Card
      title="Performance Tests"
      subtitle="Speedtest & iPerf"
      status={getStatus()}
    >
      {!runPerformanceEnabled && (
        <p className="text-sm text-text-muted mb-3">
          Performance tests are disabled in Settings.
        </p>
      )}
      <div className={runPerformanceEnabled ? "" : "opacity-50"}>
        {/* Internet Speed Section */}
        <p className="text-xs font-medium text-text-secondary mb-2">
          Internet Speed
        </p>

        {speedtestRunning && speedtestStatus && (
          <div className="space-y-2 mb-3">
            <p
              className="text-sm text-text-muted"
              id="speedtest-progress-label"
            >
              {speedtestPhaseLabels[speedtestStatus.phase] ||
                speedtestStatus.phase}
            </p>
            <div
              role="progressbar"
              aria-valuenow={speedtestStatus.progress}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-labelledby="speedtest-progress-label"
              className="w-full bg-surface-hover rounded-full h-2"
            >
              <div
                className="bg-brand-primary h-2 rounded-full transition-all duration-300"
                style={{ width: `${speedtestStatus.progress}%` }}
              />
            </div>
          </div>
        )}

        {!speedtestRunning && speedtestResult && (
          <div className="mb-3">
            <div className="grid grid-cols-2 gap-4">
              <CardValue
                label="Download"
                value={formatSpeed(speedtestResult.download)}
                size="md"
                status="success"
              />
              <CardValue
                label="Upload"
                value={formatSpeed(speedtestResult.upload)}
                size="md"
                status="success"
              />
            </div>
            <CardRow
              label="Latency"
              value={`${speedtestResult.latency.toFixed(0)} ms`}
            />
            <CardRow label="Server" value={speedtestResult.location} />
          </div>
        )}

        {!speedtestRunning && !speedtestResult && !speedtestError && (
          <p className="text-sm text-text-muted mb-2">No results yet</p>
        )}

        {speedtestError && (
          <p className="text-sm text-status-error">{speedtestError}</p>
        )}

        <CardDivider />

        {/* LAN Speed (iperf3) Section */}
        <p className="text-xs font-medium text-text-secondary mb-2 mt-2">
          LAN Speed (iperf3)
          {iperfInfo?.version && (
            <span className="text-text-muted font-normal ml-2">
              {iperfInfo.version}
            </span>
          )}
        </p>

        {!iperfInfo?.installed && (
          <p className="text-sm text-status-warning mb-3">
            iperf3 not installed. Install it to enable LAN speed tests.
          </p>
        )}

        {iperfInfo?.installed && (
          <>
            {/* Config Summary */}
            {iperfSettings.server ? (
              <div className="text-xs text-text-muted mb-3 p-2 bg-surface-hover rounded">
                <div className="flex justify-between">
                  <span>Server:</span>
                  <span className="text-text-primary">
                    {iperfSettings.server}:{iperfSettings.port}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span>Test:</span>
                  <span className="text-text-primary">
                    {iperfSettings.protocol.toUpperCase()}{" "}
                    {iperfSettings.direction === "bidirectional"
                      ? "Both"
                      : iperfSettings.direction}
                  </span>
                </div>
              </div>
            ) : (
              <p className="text-xs text-text-muted mb-3">
                Configure server in Settings
              </p>
            )}

            {/* Client Status/Results */}
            {iperfClientRunning && iperfClientStatus && (
              <div className="space-y-2 mb-3">
                <p
                  className="text-sm text-text-muted"
                  id="iperf-progress-label"
                >
                  {iperfPhaseLabels[iperfClientStatus.phase] ||
                    iperfClientStatus.phase}
                </p>
                <div
                  role="progressbar"
                  aria-valuenow={iperfClientStatus.progress}
                  aria-valuemin={0}
                  aria-valuemax={100}
                  aria-labelledby="iperf-progress-label"
                  className="w-full bg-surface-hover rounded-full h-2"
                >
                  <div
                    className="bg-brand-primary h-2 rounded-full transition-all duration-300"
                    style={{ width: `${iperfClientStatus.progress}%` }}
                  />
                </div>
              </div>
            )}

            {!iperfClientRunning && iperfResult && (
              <div className="mb-3 space-y-2">
                {iperfResult.direction === "bidirectional" ? (
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <CardValue
                      label="Download"
                      value={formatSpeed(
                        iperfResult.downloadBandwidth ?? iperfResult.bandwidth,
                      )}
                      size="md"
                      status="success"
                    />
                    <CardValue
                      label="Upload"
                      value={formatSpeed(
                        iperfResult.uploadBandwidth ?? iperfResult.bandwidth,
                      )}
                      size="md"
                      status="success"
                    />
                  </div>
                ) : (
                  <CardValue
                    label={
                      iperfResult.direction === "download"
                        ? "Download"
                        : "Upload"
                    }
                    value={formatSpeed(iperfResult.bandwidth)}
                    size="md"
                    status="success"
                  />
                )}

                {iperfResult.direction === "bidirectional" ? (
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    {iperfResult.downloadTransfer !== undefined && (
                      <CardRow
                        label="Download Transfer"
                        value={`${iperfResult.downloadTransfer.toFixed(1)} MB`}
                      />
                    )}
                    {iperfResult.uploadTransfer !== undefined && (
                      <CardRow
                        label="Upload Transfer"
                        value={`${iperfResult.uploadTransfer.toFixed(1)} MB`}
                      />
                    )}
                  </div>
                ) : (
                  <CardRow
                    label="Transfer"
                    value={`${iperfResult.transfer.toFixed(1)} MB`}
                  />
                )}

                {iperfResult.protocol === "tcp" &&
                  iperfResult.retransmits > 0 && (
                    <CardRow
                      label="Retransmits"
                      value={iperfResult.retransmits.toString()}
                    />
                  )}
                {iperfResult.protocol === "udp" && (
                  <>
                    <CardRow
                      label="Jitter"
                      value={`${iperfResult.jitter.toFixed(2)} ms`}
                    />
                    <CardRow
                      label="Packet Loss"
                      value={`${iperfResult.lostPercent.toFixed(2)}%`}
                    />
                  </>
                )}
              </div>
            )}

            {iperfError && (
              <p className="text-sm text-status-error">{iperfError}</p>
            )}

            {/* Server status indicator (if enabled) */}
            {iperfSettings.enableServer && (
              <div className="text-xs text-text-muted flex items-center justify-between p-2 bg-surface-hover rounded">
                <span>Server Mode</span>
                <span
                  className={
                    iperfServerStatus?.running
                      ? "text-status-success"
                      : "text-text-muted"
                  }
                >
                  {iperfServerStatus?.running
                    ? `Listening :${iperfServerStatus.port}`
                    : "Stopped"}
                </span>
              </div>
            )}
          </>
        )}
      </div>
    </Card>
  );
}
