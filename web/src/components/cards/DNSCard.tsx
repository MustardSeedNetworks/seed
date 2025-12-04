import { Card, CardValue, CardDivider, Status } from '../ui/Card';

interface LookupResult {
  result: string;
  time: number; // ms
  timeMs: number;
  status: Status;
  error?: string;
  resolved?: string[];
}

export interface DNSData {
  server: string;
  servers?: string[]; // All configured DNS servers
  testHostname: string;
  forward: LookupResult | null;
  forwardIpv6?: LookupResult | null;
  reverse: LookupResult | null;
  reverseIpv6?: LookupResult | null;
}

interface DNSCardProps {
  data: DNSData | null;
  loading?: boolean;
}

function formatTime(ms: number): string {
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(1)}s`;
  }
  return `${Math.round(ms)}ms`;
}

function LookupRow({
  label,
  lookup,
}: {
  label: string;
  lookup: LookupResult | null | undefined;
}) {
  if (!lookup) return null;

  return (
    <div className="mb-2">
      <div className="flex items-center justify-between">
        <span className="text-xs text-text-muted">{label}</span>
        <span
          className={`text-xs font-medium ${
            lookup.status === 'success'
              ? 'text-status-success'
              : lookup.status === 'warning'
              ? 'text-status-warning'
              : 'text-status-error'
          }`}
        >
          {formatTime(lookup.timeMs || lookup.time)}
        </span>
      </div>
      <p className="text-sm truncate" title={lookup.result}>
        {lookup.result}
      </p>
    </div>
  );
}

export function DNSCard({ data, loading }: DNSCardProps) {
  if (loading) {
    return (
      <Card title="DNS" status="loading">
        <CardValue value="Testing..." size="lg" />
      </Card>
    );
  }

  if (!data) {
    return (
      <Card title="DNS" status="unknown">
        <CardValue value="No data" size="md" />
      </Card>
    );
  }

  // Determine overall status based on forward/reverse lookups
  let overallStatus: Status = 'success';
  const lookups = [data.forward, data.forwardIpv6, data.reverse, data.reverseIpv6];
  if (lookups.some((l) => l?.status === 'error')) {
    overallStatus = 'error';
  } else if (lookups.some((l) => l?.status === 'warning')) {
    overallStatus = 'warning';
  }

  // Show all DNS servers if available
  const servers = data.servers && data.servers.length > 0 ? data.servers : [data.server];

  return (
    <Card title="DNS" status={overallStatus}>
      {/* DNS Servers */}
      <div className="mb-2">
        <p className="text-xs text-text-muted mb-1">DNS Servers</p>
        <div className="space-y-0.5">
          {servers.map((server, idx) => (
            <p key={idx} className="text-sm truncate" title={server}>
              {server}
            </p>
          ))}
        </div>
      </div>

      <p className="text-xs text-text-muted">Testing: {data.testHostname}</p>
      <CardDivider />

      {/* IPv4 Lookups */}
      {(data.forward || data.reverse) && (
        <div className="mb-2">
          <p className="text-xs font-medium text-text-secondary mb-1">IPv4</p>
          <LookupRow label="Forward (A)" lookup={data.forward} />
          <LookupRow label="Reverse (PTR)" lookup={data.reverse} />
        </div>
      )}

      {/* IPv6 Lookups */}
      {(data.forwardIpv6 || data.reverseIpv6) && (
        <>
          <CardDivider />
          <div>
            <p className="text-xs font-medium text-text-secondary mb-1">IPv6</p>
            <LookupRow label="Forward (AAAA)" lookup={data.forwardIpv6} />
            <LookupRow label="Reverse (PTR)" lookup={data.reverseIpv6} />
          </div>
        </>
      )}
    </Card>
  );
}
