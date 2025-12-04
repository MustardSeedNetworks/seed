import { useState, useEffect, useCallback } from 'react';
import { Card, CardValue, CardRow, CardDivider, Status } from '../ui/Card';
import { getAuthHeaders } from '../../hooks/useAuth';

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
  protocol: 'tcp' | 'udp';
  direction: 'upload' | 'download';
  duration: number;
  serverPort: number;
}

const speedtestPhaseLabels: Record<string, string> = {
  idle: 'Ready',
  finding_server: 'Finding server...',
  testing_latency: 'Testing latency...',
  testing_download: 'Testing download...',
  testing_upload: 'Testing upload...',
  complete: 'Complete',
};

const iperfPhaseLabels: Record<string, string> = {
  idle: 'Ready',
  connecting: 'Connecting...',
  testing: 'Testing...',
  complete: 'Complete',
};

export function PerformanceCard({ loading }: PerformanceCardProps) {
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

  // iperf3 settings (loaded from localStorage/Settings)
  const [iperfMode, setIperfMode] = useState<'client' | 'server'>('client');
  const [iperfSettings, setIperfSettings] = useState<IperfSettings>({
    server: '',
    port: 5201,
    protocol: 'tcp',
    direction: 'download',
    duration: 10,
    serverPort: 5201,
  });

  // Fetch initial status
  useEffect(() => {
    const fetchStatus = async () => {
      try {
        // Fetch speedtest status
        const speedRes = await fetch('/api/speedtest/status', {
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
        const iperfInfoRes = await fetch('/api/iperf/info', {
          headers: getAuthHeaders(),
        });
        if (iperfInfoRes.ok) {
          setIperfInfo(await iperfInfoRes.json());
        }

        // Fetch iperf3 client status
        const iperfClientRes = await fetch('/api/iperf/client/status', {
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
        const iperfServerRes = await fetch('/api/iperf/server/status', {
          headers: getAuthHeaders(),
        });
        if (iperfServerRes.ok) {
          setIperfServerStatus(await iperfServerRes.json());
        }
      } catch (err) {
        console.error('Failed to fetch performance status:', err);
      }
    };
    fetchStatus();
  }, []);

  // Load iperf settings from localStorage and listen for updates
  useEffect(() => {
    const loadSettings = () => {
      try {
        const saved = localStorage.getItem('netscope-iperf-settings');
        if (saved) {
          const parsed = JSON.parse(saved);
          setIperfSettings((prev) => ({ ...prev, ...parsed }));
        }
      } catch (err) {
        console.error('Failed to load iperf settings:', err);
      }
    };

    loadSettings();

    // Listen for settings updates from SettingsDrawer
    const handleSettingsUpdate = (e: CustomEvent<IperfSettings>) => {
      setIperfSettings((prev) => ({ ...prev, ...e.detail }));
    };

    window.addEventListener('iperfSettingsUpdated', handleSettingsUpdate as EventListener);
    return () => {
      window.removeEventListener('iperfSettingsUpdated', handleSettingsUpdate as EventListener);
    };
  }, []);

  // Poll speedtest status while running
  useEffect(() => {
    if (!speedtestRunning) return;

    const interval = setInterval(async () => {
      try {
        const res = await fetch('/api/speedtest/status', {
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
          }
        }
      } catch (err) {
        console.error('Failed to poll speedtest status:', err);
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [speedtestRunning]);

  // Poll iperf3 client status while running
  useEffect(() => {
    if (!iperfClientRunning) return;

    const interval = setInterval(async () => {
      try {
        const res = await fetch('/api/iperf/client/status', {
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
          }
        }
      } catch (err) {
        console.error('Failed to poll iperf status:', err);
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [iperfClientRunning]);

  const runSpeedtest = useCallback(async () => {
    setSpeedtestError(null);
    setSpeedtestRunning(true);
    setSpeedtestStatus({ running: true, phase: 'finding_server', progress: 0 });

    try {
      const res = await fetch('/api/speedtest', {
        method: 'POST',
        headers: getAuthHeaders(),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Speedtest failed');
      }
    } catch (err) {
      setSpeedtestError(err instanceof Error ? err.message : 'Speedtest failed');
      setSpeedtestStatus({ running: false, phase: 'idle', progress: 0 });
      setSpeedtestRunning(false);
    }
  }, []);

  const runIperfClient = useCallback(async () => {
    if (!iperfSettings.server) {
      setIperfError('Server address required - configure in Settings');
      return;
    }

    setIperfError(null);
    setIperfClientRunning(true);
    setIperfClientStatus({ running: true, phase: 'connecting', progress: 0 });

    try {
      const res = await fetch('/api/iperf/client', {
        method: 'POST',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          server: iperfSettings.server,
          port: iperfSettings.port,
          protocol: iperfSettings.protocol,
          reverse: iperfSettings.direction === 'download',
          duration: iperfSettings.duration,
          parallel: 1,
        }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'iperf3 test failed');
      }
    } catch (err) {
      setIperfError(err instanceof Error ? err.message : 'iperf3 test failed');
      setIperfClientStatus({ running: false, phase: 'idle', progress: 0 });
      setIperfClientRunning(false);
    }
  }, [iperfSettings]);

  const toggleIperfServer = useCallback(async () => {
    try {
      const action = iperfServerStatus?.running ? 'stop' : 'start';
      const res = await fetch('/api/iperf/server', {
        method: 'POST',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          action,
          port: iperfSettings.serverPort,
        }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text);
      }
      // Refresh server status
      const statusRes = await fetch('/api/iperf/server/status', {
        headers: getAuthHeaders(),
      });
      if (statusRes.ok) {
        setIperfServerStatus(await statusRes.json());
      }
    } catch (err) {
      setIperfError(err instanceof Error ? err.message : 'Server toggle failed');
    }
  }, [iperfServerStatus, iperfSettings.serverPort]);

  const formatSpeed = (mbps: number): string => {
    if (mbps >= 1000) {
      return `${(mbps / 1000).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} Gbps`;
    }
    return `${mbps.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} Mbps`;
  };

  const getStatus = (): Status => {
    if (loading || speedtestRunning || iperfClientRunning) return 'loading';
    if (speedtestError || iperfError) return 'error';
    if (speedtestResult || iperfResult) return 'success';
    return 'unknown';
  };

  const isAnyTestRunning = speedtestRunning || iperfClientRunning;

  return (
    <Card title="Performance" status={getStatus()}>
      {/* Internet Speed Section */}
      <p className="text-xs font-medium text-text-secondary mb-2">Internet Speed</p>

      {speedtestRunning && speedtestStatus && (
        <div className="space-y-2 mb-3">
          <p className="text-sm text-text-muted">{speedtestPhaseLabels[speedtestStatus.phase] || speedtestStatus.phase}</p>
          <div className="w-full bg-surface-hover rounded-full h-2">
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
            <CardValue label="Download" value={formatSpeed(speedtestResult.download)} size="md" status="success" />
            <CardValue label="Upload" value={formatSpeed(speedtestResult.upload)} size="md" status="success" />
          </div>
          <CardRow label="Latency" value={`${speedtestResult.latency.toFixed(0)} ms`} />
          <CardRow label="Server" value={speedtestResult.location} />
        </div>
      )}

      {!speedtestRunning && !speedtestResult && !speedtestError && (
        <p className="text-sm text-text-muted mb-2">No results yet</p>
      )}

      {speedtestError && (
        <p className="text-sm text-status-error mb-2">{speedtestError}</p>
      )}

      <button
        onClick={runSpeedtest}
        disabled={isAnyTestRunning}
        className={`w-full py-2 px-4 rounded-lg font-medium transition-colors mb-3 ${
          isAnyTestRunning
            ? 'bg-surface-hover text-text-muted cursor-not-allowed'
            : 'bg-brand-primary text-text-inverse hover:bg-brand-accent'
        }`}
      >
        {speedtestRunning ? 'Running...' : 'Run Again'}
      </button>

      <CardDivider />

      {/* LAN Speed (iperf3) Section */}
      <p className="text-xs font-medium text-text-secondary mb-2 mt-2">
        LAN Speed (iperf3)
        {iperfInfo?.version && (
          <span className="text-text-muted font-normal ml-2">{iperfInfo.version}</span>
        )}
      </p>

      {!iperfInfo?.installed && (
        <p className="text-sm text-status-warning mb-3">
          iperf3 not installed. Install it to enable LAN speed tests.
        </p>
      )}

      {iperfInfo?.installed && (
        <>
          {/* Mode Toggle */}
          <div className="flex gap-2 mb-3">
            <button
              onClick={() => setIperfMode('client')}
              className={`flex-1 py-1.5 px-3 rounded text-sm font-medium transition-colors ${
                iperfMode === 'client'
                  ? 'bg-brand-primary text-text-inverse'
                  : 'bg-surface-hover text-text-muted hover:text-text-primary'
              }`}
            >
              Client
            </button>
            <button
              onClick={() => setIperfMode('server')}
              className={`flex-1 py-1.5 px-3 rounded text-sm font-medium transition-colors ${
                iperfMode === 'server'
                  ? 'bg-brand-primary text-text-inverse'
                  : 'bg-surface-hover text-text-muted hover:text-text-primary'
              }`}
            >
              Server
            </button>
          </div>

          {iperfMode === 'client' && (
            <>
              {/* Client Config Summary */}
              {iperfSettings.server ? (
                <div className="text-xs text-text-muted mb-3 p-2 bg-surface-hover rounded">
                  <div className="flex justify-between">
                    <span>Server:</span>
                    <span className="text-text-primary">{iperfSettings.server}:{iperfSettings.port}</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Test:</span>
                    <span className="text-text-primary">{iperfSettings.protocol.toUpperCase()} {iperfSettings.direction}</span>
                  </div>
                </div>
              ) : (
                <p className="text-xs text-text-muted mb-3">
                  Configure server address in Settings
                </p>
              )}

              {/* Client Status/Results */}
              {iperfClientRunning && iperfClientStatus && (
                <div className="space-y-2 mb-3">
                  <p className="text-sm text-text-muted">{iperfPhaseLabels[iperfClientStatus.phase] || iperfClientStatus.phase}</p>
                  <div className="w-full bg-surface-hover rounded-full h-2">
                    <div
                      className="bg-brand-primary h-2 rounded-full transition-all duration-300"
                      style={{ width: `${iperfClientStatus.progress}%` }}
                    />
                  </div>
                </div>
              )}

              {!iperfClientRunning && iperfResult && (
                <div className="mb-3">
                  <CardValue
                    label={iperfResult.direction === 'download' ? 'Download' : 'Upload'}
                    value={formatSpeed(iperfResult.bandwidth)}
                    size="md"
                    status="success"
                  />
                  <CardRow label="Transfer" value={`${iperfResult.transfer.toFixed(1)} MB`} />
                  {iperfResult.protocol === 'tcp' && iperfResult.retransmits > 0 && (
                    <CardRow label="Retransmits" value={iperfResult.retransmits.toString()} />
                  )}
                  {iperfResult.protocol === 'udp' && (
                    <>
                      <CardRow label="Jitter" value={`${iperfResult.jitter.toFixed(2)} ms`} />
                      <CardRow label="Packet Loss" value={`${iperfResult.lostPercent.toFixed(2)}%`} />
                    </>
                  )}
                </div>
              )}

              {iperfError && (
                <p className="text-sm text-status-error mb-3">{iperfError}</p>
              )}

              <button
                onClick={runIperfClient}
                disabled={isAnyTestRunning}
                className={`w-full py-2 px-4 rounded-lg font-medium transition-colors ${
                  isAnyTestRunning
                    ? 'bg-surface-hover text-text-muted cursor-not-allowed'
                    : 'bg-brand-primary text-text-inverse hover:bg-brand-accent'
                }`}
              >
                {iperfClientRunning ? 'Testing...' : 'Run Test'}
              </button>
            </>
          )}

          {iperfMode === 'server' && (
            <>
              {/* Server Config */}
              <div className="space-y-2 mb-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-text-muted">Port: {iperfSettings.serverPort}</span>
                  <span className={`text-sm font-medium ${iperfServerStatus?.running ? 'text-status-success' : 'text-text-muted'}`}>
                    {iperfServerStatus?.running ? 'Running' : 'Stopped'}
                  </span>
                </div>
              </div>

              <button
                onClick={toggleIperfServer}
                className={`w-full py-2 px-4 rounded-lg font-medium transition-colors ${
                  iperfServerStatus?.running
                    ? 'bg-status-error text-text-inverse hover:opacity-90'
                    : 'bg-brand-primary text-text-inverse hover:bg-brand-accent'
                }`}
              >
                {iperfServerStatus?.running ? 'Stop Server' : 'Start Server'}
              </button>

              {iperfServerStatus?.running && (
                <p className="text-xs text-text-muted mt-2 text-center">
                  Listening on port {iperfServerStatus.port}
                </p>
              )}
            </>
          )}
        </>
      )}
    </Card>
  );
}
