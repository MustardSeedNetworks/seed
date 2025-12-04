import { useState, useEffect, useCallback } from 'react';
import { Card, CardValue, CardRow, CardDivider, Status } from '../ui/Card';
import { getAuthHeaders } from '../../hooks/useAuth';

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

interface SpeedtestCardProps {
  loading?: boolean;
}

const phaseLabels: Record<string, string> = {
  idle: 'Ready',
  finding_server: 'Finding server...',
  testing_latency: 'Testing latency...',
  testing_download: 'Testing download...',
  testing_upload: 'Testing upload...',
  complete: 'Complete',
};

export function SpeedtestCard({ loading }: SpeedtestCardProps) {
  const [status, setStatus] = useState<SpeedtestStatus | null>(null);
  const [result, setResult] = useState<SpeedtestData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isRunning, setIsRunning] = useState(false);

  // Fetch initial status
  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const res = await fetch('/api/speedtest/status', {
          headers: getAuthHeaders(),
        });
        if (res.ok) {
          const data = await res.json();
          setStatus(data);
          if (data.last) {
            setResult(data.last);
          }
          setIsRunning(data.running);
        }
      } catch (err) {
        console.error('Failed to fetch speedtest status:', err);
      }
    };
    fetchStatus();
  }, []);

  // Poll status while running
  useEffect(() => {
    if (!isRunning) return;

    const interval = setInterval(async () => {
      try {
        const res = await fetch('/api/speedtest/status', {
          headers: getAuthHeaders(),
        });
        if (res.ok) {
          const data = await res.json();
          setStatus(data);
          if (!data.running) {
            setIsRunning(false);
            if (data.last) {
              setResult(data.last);
            }
          }
        }
      } catch (err) {
        console.error('Failed to poll speedtest status:', err);
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [isRunning]);

  const runSpeedtest = useCallback(async () => {
    setError(null);
    setIsRunning(true);
    setStatus({ running: true, phase: 'finding_server', progress: 0 });

    try {
      // Start the speedtest (returns immediately, runs in background)
      const res = await fetch('/api/speedtest', {
        method: 'POST',
        headers: getAuthHeaders(),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Speedtest failed');
      }
      // The test is now running in background - polling will pick up results
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Speedtest failed');
      setStatus({ running: false, phase: 'idle', progress: 0 });
      setIsRunning(false);
    }
  }, []);

  const formatSpeed = (mbps: number): string => {
    if (mbps >= 1000) {
      return `${(mbps / 1000).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} Gbps`;
    }
    return `${mbps.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} Mbps`;
  };

  const getStatus = (): Status => {
    if (loading || isRunning) return 'loading';
    if (error) return 'error';
    if (result) return 'success';
    return 'unknown';
  };

  return (
    <Card title="Speedtest" status={getStatus()}>
      {isRunning && status && (
        <div className="space-y-2">
          <p className="text-sm text-text-muted">{phaseLabels[status.phase] || status.phase}</p>
          <div className="w-full bg-surface-hover rounded-full h-2">
            <div
              className="bg-brand-primary h-2 rounded-full transition-all duration-300"
              style={{ width: `${status.progress}%` }}
            />
          </div>
          <p className="text-xs text-text-muted text-right">{status.progress.toFixed(0)}%</p>
        </div>
      )}

      {!isRunning && result && (
        <>
          <div className="grid grid-cols-2 gap-4">
            <CardValue label="Download" value={formatSpeed(result.download)} size="lg" status="success" />
            <CardValue label="Upload" value={formatSpeed(result.upload)} size="lg" status="success" />
          </div>
          <CardDivider />
          <CardRow label="Latency" value={`${result.latency.toFixed(0)} ms`} />
          <CardRow label="Server" value={result.server} />
          <CardRow label="Location" value={result.location} />
          {result.distance > 0 && (
            <CardRow label="Distance" value={`${result.distance.toFixed(0)} km`} />
          )}
          <CardDivider />
        </>
      )}

      {!isRunning && !result && !error && (
        <p className="text-sm text-text-muted mb-3">
          Click the button below to run a speed test.
        </p>
      )}

      {error && (
        <p className="text-sm text-status-error mb-3">{error}</p>
      )}

      <button
        onClick={runSpeedtest}
        disabled={isRunning}
        className={`w-full py-2 px-4 rounded-lg font-medium transition-colors ${
          isRunning
            ? 'bg-surface-hover text-text-muted cursor-not-allowed'
            : 'bg-brand-primary text-text-inverse hover:bg-brand-accent'
        }`}
      >
        {isRunning ? 'Running...' : result ? 'Run Again' : 'Start Speedtest'}
      </button>
    </Card>
  );
}
