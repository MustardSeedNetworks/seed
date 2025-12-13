import { memo } from "react";
import { Card, CardValue, CardDivider, Status } from "../ui/Card";
import { StatusBadge } from "../ui/StatusBadge";
import { CollapsibleSection } from "../ui/CollapsibleSection";
import { Globe } from "../ui/Icons";

interface LookupResult {
  result: string;
  time: number; // ms
  timeMs: number;
  status: Status;
  error?: string;
  resolved?: string[];
}

interface ServerTestResult {
  server: string;
  forward: LookupResult | null;
  forwardIpv6: LookupResult | null;
  status: Status;
  avgTimeMs: number;
}

export interface DNSData {
  server: string;
  servers?: string[]; // All configured DNS servers
  testHostname: string;
  forward: LookupResult | null;
  forwardIpv6?: LookupResult | null;
  reverse: LookupResult | null;
  reverseIpv6?: LookupResult | null;
  perServerResults?: ServerTestResult[];
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

  const statusBadge = lookup.status;
  const statusColor =
    statusBadge === "success"
      ? "text-status-success"
      : statusBadge === "warning"
        ? "text-status-warning"
        : "text-status-error";

  return (
    <div className="mb-2">
      <div className="flex items-center justify-between">
        <span className="text-xs text-text-muted">{label}</span>
        <span className="inline-flex items-center gap-2">
          <StatusBadge status={statusBadge} size="sm" />
          <span className={`text-xs font-medium ${statusColor}`}>
            {formatTime(lookup.timeMs || lookup.time)}
          </span>
        </span>
      </div>
      <p className="text-sm truncate" title={lookup.result}>
        {lookup.result}
      </p>
    </div>
  );
}

export const DNSCard = memo(function DNSCard({ data, loading }: DNSCardProps) {
  if (loading) {
    return (
      <Card title="DNS" icon={<Globe className="w-5 h-5" />} status="loading">
        <CardValue value="Testing..." size="lg" />
      </Card>
    );
  }

  if (!data) {
    return (
      <Card title="DNS" icon={<Globe className="w-5 h-5" />} status="unknown">
        <CardValue value="No data" size="md" />
      </Card>
    );
  }

  // Determine overall status based on forward/reverse lookups
  let overallStatus: Status = "success";
  const lookups = [
    data.forward,
    data.forwardIpv6,
    data.reverse,
    data.reverseIpv6,
  ];
  if (lookups.some((l) => l?.status === "error")) {
    overallStatus = "error";
  } else if (lookups.some((l) => l?.status === "warning")) {
    overallStatus = "warning";
  }

  // Show all DNS servers if available
  const servers =
    data.servers && data.servers.length > 0 ? data.servers : [data.server];

  return (
    <Card
      title="DNS"
      icon={<Globe className="w-5 h-5" />}
      status={overallStatus}
    >
      {/* DNS Servers */}
      <div className="mb-2">
        <p className="text-xs text-text-muted mb-1">DNS Servers</p>
        <div className="space-y-0.5">
          {servers.map((server, idx) => (
            <p key={idx} className="text-sm font-mono break-all" title={server}>
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

      {/* Per-Server Results (collapsible) */}
      {data.perServerResults && data.perServerResults.length > 0 && (
        <>
          <CardDivider />
          <CollapsibleSection
            title="Server Tests"
            count={data.perServerResults.length}
            variant="compact"
            status={
              data.perServerResults.some((s) => s.status === "error")
                ? "error"
                : data.perServerResults.some((s) => s.status === "warning")
                  ? "warning"
                  : "success"
            }
          >
            {data.perServerResults.map((server) => (
              <div key={server.server} className="py-1">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs font-mono text-text-primary">
                    {server.server}
                  </span>
                  <span
                    className={`text-xs font-medium ${
                      server.status === "success"
                        ? "text-status-success"
                        : server.status === "warning"
                          ? "text-status-warning"
                          : "text-status-error"
                    }`}
                  >
                    {formatTime(server.avgTimeMs)}
                  </span>
                </div>
                {server.forward && (
                  <div className="flex justify-between text-xs text-text-muted">
                    <span>A</span>
                    <span className="inline-flex items-center gap-2">
                      <StatusBadge status={server.forward.status} size="sm" />
                      <span
                        className={
                          server.forward.status === "success"
                            ? "text-status-success"
                            : server.forward.status === "warning"
                              ? "text-status-warning"
                              : "text-status-error"
                        }
                      >
                        {server.forward.result === "No A record"
                          ? "N/A"
                          : formatTime(server.forward.timeMs)}
                      </span>
                    </span>
                  </div>
                )}
                {server.forwardIpv6 && (
                  <div className="flex justify-between text-xs text-text-muted">
                    <span>AAAA</span>
                    <span className="inline-flex items-center gap-2">
                      <StatusBadge
                        status={server.forwardIpv6.status}
                        size="sm"
                      />
                      <span
                        className={
                          server.forwardIpv6.status === "success"
                            ? "text-status-success"
                            : server.forwardIpv6.status === "warning"
                              ? "text-status-warning"
                              : "text-status-error"
                        }
                      >
                        {server.forwardIpv6.result === "No AAAA record"
                          ? "N/A"
                          : formatTime(server.forwardIpv6.timeMs)}
                      </span>
                    </span>
                  </div>
                )}
              </div>
            ))}
          </CollapsibleSection>
        </>
      )}
    </Card>
  );
});
