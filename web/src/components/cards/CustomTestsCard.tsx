import { useState, useEffect, useCallback } from 'react';
import { Card, CardDivider, Status } from '../ui/Card';
import { getAuthHeaders } from '../../hooks/useAuth';

interface TestResult {
  name: string;
  host?: string;
  port?: number;
  url?: string;
  success: boolean;
  latency: number;
  error?: string;
  status?: number;
  testStatus?: 'success' | 'warning' | 'error';
  // Extended ping fields
  packetLoss?: number;
  jitter?: number;
  minLatency?: number;
  maxLatency?: number;
  // Certificate expiry fields
  certDaysLeft?: number;
  certStatus?: 'success' | 'warning' | 'error';
  certExpiry?: string;
  certCommonName?: string;
}

interface CustomTestsData {
  pingResults: TestResult[];
  tcpResults: TestResult[];
  udpResults: TestResult[];
  httpResults: TestResult[];
  hasTests: boolean;
}

interface CustomTestsCardProps {
  loading?: boolean;
}

export function CustomTestsCard({ loading }: CustomTestsCardProps) {
  const [data, setData] = useState<CustomTestsData | null>(null);
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTests = useCallback(async () => {
    setIsRunning(true);
    setError(null);
    try {
      const res = await fetch('/api/tests/run', {
        headers: getAuthHeaders(),
      });
      if (res.ok) {
        const result = await res.json();
        setData(result);
      } else {
        setError('Failed to run tests');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to run tests');
    } finally {
      setIsRunning(false);
    }
  }, []);

  // Initial fetch to check if tests are configured
  useEffect(() => {
    fetchTests();
  }, [fetchTests]);

  // Don't render card if no tests are configured
  if (!data?.hasTests && !loading && !isRunning) {
    return null;
  }

  const getStatus = (): Status => {
    if (loading || isRunning) return 'loading';
    if (error) return 'error';
    if (!data) return 'unknown';

    const allResults = [
      ...data.pingResults,
      ...data.tcpResults,
      ...(data.udpResults || []),
      ...data.httpResults,
    ];
    if (allResults.length === 0) return 'unknown';

    // Check for any failures first
    if (allResults.some((r) => !r.success)) {
      if (allResults.every((r) => !r.success)) return 'error';
      return 'warning';
    }

    // All succeeded - check for threshold warnings (including cert expiry)
    if (allResults.some((r) => r.testStatus === 'error' || r.certStatus === 'error')) return 'error';
    if (allResults.some((r) => r.testStatus === 'warning' || r.certStatus === 'warning')) return 'warning';
    return 'success';
  };

  const formatLatency = (ms: number): string => {
    if (ms >= 1000) {
      return `${(ms / 1000).toFixed(1)}s`;
    }
    return `${Math.round(ms)}ms`;
  };

  const renderTestResult = (result: TestResult, type: 'ping' | 'tcp' | 'udp' | 'http') => {
    // Use testStatus for threshold-based coloring, fall back to success/error
    let statusColor = 'text-status-error';
    if (result.success) {
      if (result.testStatus === 'warning') {
        statusColor = 'text-status-warning';
      } else if (result.testStatus === 'error') {
        statusColor = 'text-status-error';
      } else {
        statusColor = 'text-status-success';
      }
    }

    let details = '';
    if ((type === 'tcp' || type === 'udp') && result.port) {
      details = `:${result.port}`;
    } else if (type === 'http' && result.status) {
      details = ` (${result.status})`;
    }

    // Extended ping info
    const hasExtendedPing = type === 'ping' && result.packetLoss !== undefined;
    const extendedInfo = hasExtendedPing
      ? `${result.packetLoss?.toFixed(0)}% loss${result.jitter !== undefined ? `, ${result.jitter.toFixed(1)}ms jitter` : ''}`
      : null;

    return (
      <div key={`${type}-${result.name}`} className="py-1">
        <div className="flex items-center justify-between">
          <span className="text-sm text-text-muted truncate flex-1" title={result.name}>
            {result.name}{details}
          </span>
          <span className={`text-sm font-medium ${statusColor}`}>
            {result.success ? formatLatency(result.latency) : 'fail'}
          </span>
        </div>
        {extendedInfo && (
          <div className="text-xs text-text-muted mt-0.5">
            {extendedInfo}
          </div>
        )}
      </div>
    );
  };

  const renderHTTPResult = (result: TestResult) => {
    // Use testStatus for threshold-based coloring
    let statusColor = 'text-status-error';
    if (result.success) {
      if (result.testStatus === 'warning') {
        statusColor = 'text-status-warning';
      } else if (result.testStatus === 'error') {
        statusColor = 'text-status-error';
      } else {
        statusColor = 'text-status-success';
      }
    }

    // Certificate status coloring
    let certColor = 'text-text-muted';
    if (result.certStatus === 'error') {
      certColor = 'text-status-error';
    } else if (result.certStatus === 'warning') {
      certColor = 'text-status-warning';
    } else if (result.certStatus === 'success') {
      certColor = 'text-status-success';
    }

    const hasCertInfo = result.certDaysLeft !== undefined && result.certDaysLeft >= 0;

    return (
      <div key={`http-${result.name}`} className="py-1">
        <div className="flex items-center justify-between">
          <span className="text-sm text-text-muted truncate flex-1" title={result.name}>
            {result.name}{result.status ? ` (${result.status})` : ''}
          </span>
          <span className={`text-sm font-medium ${statusColor}`}>
            {result.success ? formatLatency(result.latency) : 'fail'}
          </span>
        </div>
        {hasCertInfo && (
          <div className={`text-xs mt-0.5 ${certColor}`}>
            Cert: {result.certDaysLeft}d left
          </div>
        )}
      </div>
    );
  };

  return (
    <Card title="Health Checks" status={getStatus()}>
      {isRunning && (
        <p className="text-sm text-text-muted">Running tests...</p>
      )}

      {!isRunning && data && (
        <>
          {/* Ping Results */}
          {data.pingResults.length > 0 && (
            <div className="mb-2">
              <p className="text-xs font-medium text-text-secondary mb-1">Ping</p>
              {data.pingResults.map((r) => renderTestResult(r, 'ping'))}
            </div>
          )}

          {/* TCP Results */}
          {data.tcpResults.length > 0 && (
            <>
              {data.pingResults.length > 0 && <CardDivider />}
              <div className="mb-2">
                <p className="text-xs font-medium text-text-secondary mb-1">TCP Ports</p>
                {data.tcpResults.map((r) => renderTestResult(r, 'tcp'))}
              </div>
            </>
          )}

          {/* UDP Results */}
          {data.udpResults && data.udpResults.length > 0 && (
            <>
              {(data.pingResults.length > 0 || data.tcpResults.length > 0) && <CardDivider />}
              <div className="mb-2">
                <p className="text-xs font-medium text-text-secondary mb-1">UDP Ports</p>
                {data.udpResults.map((r) => renderTestResult(r, 'udp'))}
              </div>
            </>
          )}

          {/* HTTP Results */}
          {data.httpResults.length > 0 && (
            <>
              {(data.pingResults.length > 0 || data.tcpResults.length > 0 || (data.udpResults && data.udpResults.length > 0)) && <CardDivider />}
              <div className="mb-2">
                <p className="text-xs font-medium text-text-secondary mb-1">HTTP</p>
                {data.httpResults.map((r) => renderHTTPResult(r))}
              </div>
            </>
          )}

          <CardDivider />
        </>
      )}

      {error && (
        <p className="text-sm text-status-error mb-3">{error}</p>
      )}

      <button
        onClick={fetchTests}
        disabled={isRunning}
        className={`w-full py-2 px-4 rounded-lg font-medium transition-colors ${
          isRunning
            ? 'bg-surface-hover text-text-muted cursor-not-allowed'
            : 'bg-brand-primary text-text-inverse hover:bg-brand-accent'
        }`}
      >
        {isRunning ? 'Running...' : 'Run Tests'}
      </button>
    </Card>
  );
}
