import { Card, CardValue, CardRow, CardDivider, Status } from '../ui/Card';

export interface GatewayData {
  gateway: string;
  reachable: boolean;
  sent: number;
  received: number;
  lossPercent: number;
  minTime: number;
  maxTime: number;
  avgTime: number;
  lastTime: number;
  status: string;
}

interface GatewayCardProps {
  data: GatewayData | null;
  loading?: boolean;
  thresholds?: { warning: number; critical: number };
}

function getLatencyStatus(
  value: number,
  thresholds: { warning: number; critical: number }
): Status {
  if (value >= thresholds.critical) return 'error';
  if (value >= thresholds.warning) return 'warning';
  return 'success';
}

function formatTime(ms: number): string {
  if (ms < 1) return '<1ms';
  if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.round(ms * 10) / 10}ms`;
}

export function GatewayCard({ data, loading, thresholds }: GatewayCardProps) {
  const t = thresholds || { warning: 50, critical: 200 };

  if (loading) {
    return (
      <Card title="Gateway" status="loading">
        <CardValue value="Pinging..." size="lg" />
      </Card>
    );
  }

  if (!data || !data.gateway) {
    return (
      <Card title="Gateway" status="unknown">
        <CardValue value="No gateway" size="md" />
        <p className="text-xs text-text-muted mt-1">Unable to detect default gateway</p>
      </Card>
    );
  }

  // Map API status to card status
  let status: Status = 'unknown';
  switch (data.status) {
    case 'success':
      status = 'success';
      break;
    case 'warning':
      status = 'warning';
      break;
    case 'error':
      status = 'error';
      break;
    default:
      status = data.reachable ? getLatencyStatus(data.avgTime, t) : 'error';
  }

  return (
    <Card title="Gateway" status={status}>
      <CardValue value={data.gateway} size="lg" />
      <p className="text-xs text-text-muted">
        {data.reachable ? 'Reachable' : 'Unreachable'}
      </p>
      <CardDivider />

      {/* Latency stats */}
      <div className="grid grid-cols-3 gap-2 mb-2">
        <div className="text-center">
          <p className="text-xs text-text-muted">Min</p>
          <p className={`text-sm font-medium ${
            data.minTime > 0 ? getLatencyStatus(data.minTime, t) === 'success'
              ? 'text-status-success'
              : getLatencyStatus(data.minTime, t) === 'warning'
              ? 'text-status-warning'
              : 'text-status-error'
            : 'text-text-muted'
          }`}>
            {data.minTime > 0 ? formatTime(data.minTime) : '-'}
          </p>
        </div>
        <div className="text-center">
          <p className="text-xs text-text-muted">Avg</p>
          <p className={`text-sm font-medium ${
            data.avgTime > 0 ? getLatencyStatus(data.avgTime, t) === 'success'
              ? 'text-status-success'
              : getLatencyStatus(data.avgTime, t) === 'warning'
              ? 'text-status-warning'
              : 'text-status-error'
            : 'text-text-muted'
          }`}>
            {data.avgTime > 0 ? formatTime(data.avgTime) : '-'}
          </p>
        </div>
        <div className="text-center">
          <p className="text-xs text-text-muted">Max</p>
          <p className={`text-sm font-medium ${
            data.maxTime > 0 ? getLatencyStatus(data.maxTime, t) === 'success'
              ? 'text-status-success'
              : getLatencyStatus(data.maxTime, t) === 'warning'
              ? 'text-status-warning'
              : 'text-status-error'
            : 'text-text-muted'
          }`}>
            {data.maxTime > 0 ? formatTime(data.maxTime) : '-'}
          </p>
        </div>
      </div>

      <CardRow
        label="Packets"
        value={`${data.received}/${data.sent}`}
        status={data.lossPercent === 0 ? 'success' : data.lossPercent < 50 ? 'warning' : 'error'}
      />
      {data.lossPercent > 0 && (
        <CardRow
          label="Packet Loss"
          value={`${Math.round(data.lossPercent)}%`}
          status={data.lossPercent >= 50 ? 'error' : 'warning'}
        />
      )}
    </Card>
  );
}
