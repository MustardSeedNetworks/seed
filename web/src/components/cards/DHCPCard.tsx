import { useMemo, useState } from "react";
import { Card, CardValue, CardRow, CardDivider, Status } from "../ui/Card";

export interface DHCPTiming {
  discover: number; // ms
  offer: number;
  request: number;
  ack: number;
  total: number;
}

export interface IPv4Info {
  address: string;
  subnet: string;
  gateway: string | null;
  dhcpServer: string | null;
  leaseTime: number | null;
}

export interface IPv6Info {
  address: string;
  prefix: number;
  scope: "global" | "link-local" | "unique-local";
  source: "slaac" | "dhcpv6" | "static" | "temporary";
}

export interface DHCPData {
  mac: string;
  mode: "dhcp" | "static" | "auto";
  ipv4: IPv4Info | null;
  ipv6: IPv6Info[];
  dns: string[];
  timing: DHCPTiming | null;
}

export interface PublicIPInfo {
  ipv4?: string;
  ipv6?: string;
  lastChecked: string;
  error?: string;
}

interface DHCPCardProps {
  data: DHCPData | null;
  publicip?: PublicIPInfo | null;
  loading?: boolean;
  showPublicIP?: boolean;
  thresholds?: {
    total: { warning: number; critical: number };
    perPhase: { warning: number; critical: number };
  };
}

function getTimingStatus(
  value: number,
  thresholds: { warning: number; critical: number },
): Status {
  if (value >= thresholds.critical) return "error";
  if (value >= thresholds.warning) return "warning";
  return "success";
}

function formatTime(ms: number): string {
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(1)}s`;
  }
  return `${Math.round(ms)}ms`;
}

function formatLeaseTime(seconds: number): string {
  if (seconds >= 86400) {
    const days = Math.floor(seconds / 86400);
    return `${days}d`;
  }
  if (seconds >= 3600) {
    const hours = Math.floor(seconds / 3600);
    return `${hours}h`;
  }
  if (seconds >= 60) {
    const mins = Math.floor(seconds / 60);
    return `${mins}m`;
  }
  return `${seconds}s`;
}

function getScopeLabel(scope: IPv6Info["scope"]): string {
  switch (scope) {
    case "global":
      return "Global";
    case "link-local":
      return "Link-Local";
    case "unique-local":
      return "ULA";
    default:
      return scope;
  }
}

// getSourceLabel for future use when displaying IPv6 source type
// function getSourceLabel(source: IPv6Info['source']): string {
//   switch (source) {
//     case 'slaac': return 'SLAAC';
//     case 'dhcpv6': return 'DHCPv6';
//     case 'static': return 'Static';
//     case 'temporary': return 'Temporary';
//     default: return source;
//   }
// }

// Compress IPv6 address by replacing longest run of zeros with ::
function compressIPv6(address: string): string {
  // Already compressed or not a valid IPv6
  if (address.includes("::") || !address.includes(":")) {
    return address;
  }

  // Split into groups and find longest run of zeros
  const groups = address.split(":");
  let longestStart = -1;
  let longestLength = 0;
  let currentStart = -1;
  let currentLength = 0;

  for (let i = 0; i < groups.length; i++) {
    if (groups[i] === "0" || groups[i] === "0000") {
      if (currentStart === -1) currentStart = i;
      currentLength++;
    } else {
      if (currentLength > longestLength) {
        longestStart = currentStart;
        longestLength = currentLength;
      }
      currentStart = -1;
      currentLength = 0;
    }
  }
  if (currentLength > longestLength) {
    longestStart = currentStart;
    longestLength = currentLength;
  }

  // Only compress if we have at least 2 consecutive zero groups
  if (longestLength < 2) {
    // Just remove leading zeros from each group
    return groups.map((g) => g.replace(/^0+/, "") || "0").join(":");
  }

  // Build compressed address
  const before = groups
    .slice(0, longestStart)
    .map((g) => g.replace(/^0+/, "") || "0");
  const after = groups
    .slice(longestStart + longestLength)
    .map((g) => g.replace(/^0+/, "") || "0");

  if (before.length === 0 && after.length === 0) return "::";
  if (before.length === 0) return "::" + after.join(":");
  if (after.length === 0) return before.join(":") + "::";
  return before.join(":") + "::" + after.join(":");
}

export function DHCPCard({
  data,
  publicip,
  loading,
  showPublicIP = true,
  thresholds,
}: DHCPCardProps) {
  const defaultThresholds = {
    total: { warning: 500, critical: 2000 },
    perPhase: { warning: 200, critical: 1000 },
  };
  const t = thresholds || defaultThresholds;
  const [showTiming, setShowTiming] = useState(false);

  // Keep hooks unconditional: derive safe fallbacks
  const hasData = !!data;
  const ipv4 = data?.ipv4 ?? null;
  const ipv6List = useMemo(() => data?.ipv6 ?? [], [data?.ipv6]);
  const timing = data?.timing ?? null;
  const hasIPv4 = ipv4 !== null;
  const hasIPv6 = ipv6List.length > 0;
  const globalIPv6 = ipv6List.filter((ip) => ip.scope === "global");

  const groupedIPv6 = useMemo(() => {
    const order: IPv6Info["scope"][] = ["global", "unique-local", "link-local"];
    return order
      .map((scope) => ({
        scope,
        label: getScopeLabel(scope),
        entries: ipv6List.filter((ip) => ip.scope === scope),
      }))
      .filter((group) => group.entries.length > 0);
  }, [ipv6List]);

  // Determine overall status using priority: error > warning > success
  const getOverallStatus = (): Status => {
    if (loading) return "loading";
    if (!hasData) return "unknown";

    // No IP at all is a warning (might be in progress)
    if (!hasIPv4 && globalIPv6.length === 0) {
      return "warning";
    }

    // If we have timing data, check for errors/warnings
    if (timing) {
      const timingStatuses = [
        getTimingStatus(timing.discover, t.perPhase),
        getTimingStatus(timing.offer, t.perPhase),
        getTimingStatus(timing.request, t.perPhase),
        getTimingStatus(timing.ack, t.perPhase),
        getTimingStatus(timing.total, t.total),
      ];

      // Any error = card is error
      if (timingStatuses.some((s) => s === "error")) {
        return "error";
      }

      // Any warning = card is warning
      if (timingStatuses.some((s) => s === "warning")) {
        return "warning";
      }
    }

    // All good
    return "success";
  };

  const status: Status = getOverallStatus();

  // Primary display value
  const primaryIPRaw =
    ipv4?.address ||
    globalIPv6[0]?.address ||
    (loading ? "Loading..." : hasData ? "No IP" : "No data");
  const primaryIP =
    primaryIPRaw && primaryIPRaw.includes(":")
      ? compressIPv6(primaryIPRaw)
      : primaryIPRaw;

  return (
    <Card title="IP Configuration" status={status}>
      <CardValue value={primaryIP} size="lg" mono allowWrap />

      <CardDivider />

      {!hasData && <CardValue value="No data available" size="md" />}

      {hasData && (
        <>
          {/* MAC Address */}
          <CardRow label="MAC" value={data!.mac} />
          <CardRow label="Mode" value={data!.mode.toUpperCase()} />

          {/* IPv4 Section */}
          {hasIPv4 && ipv4 && (
            <>
              <CardDivider />
              <p className="text-xs text-text-muted mb-1 font-medium">IPv4</p>
              <CardRow
                label="Address"
                value={`${ipv4.address}/${ipv4.subnet}`}
                wrap
                mono
              />
              {ipv4.gateway && (
                <CardRow label="Gateway" value={ipv4.gateway} wrap mono />
              )}
              {ipv4.dhcpServer && (
                <CardRow
                  label="DHCP Server"
                  value={ipv4.dhcpServer}
                  wrap
                  mono
                />
              )}
              {ipv4.leaseTime && (
                <CardRow
                  label="Lease"
                  value={formatLeaseTime(ipv4.leaseTime)}
                />
              )}
            </>
          )}

          {/* IPv6 Section */}
          {hasIPv6 && (
            <>
              <CardDivider />
              <p className="text-xs text-text-muted mb-1 font-medium">IPv6</p>
              <div className="space-y-2">
                {groupedIPv6.map((group, groupIdx) => (
                  <div key={groupIdx} className="space-y-1">
                    <p className="text-[11px] uppercase tracking-wide text-text-muted font-semibold">
                      {group.label}
                    </p>
                    {group.entries.map((ip, idx) => (
                      <CardRow
                        key={`${groupIdx}-${idx}`}
                        label="Address"
                        value={`${compressIPv6(ip.address)}/${ip.prefix}`}
                        wrap
                        mono
                        align="right"
                        status={ip.scope === "global" ? "success" : undefined}
                      />
                    ))}
                  </div>
                ))}
              </div>
            </>
          )}

          {/* DHCP Timing (if available) */}
          {timing && (
            <>
              <CardDivider />
              <div className="flex items-center justify-between mb-1">
                <p className="text-xs text-text-muted font-medium">
                  DHCP Timing
                </p>
                <button
                  type="button"
                  className="text-xs font-medium text-brand-primary hover:text-brand-primary/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary rounded px-1"
                  onClick={() => setShowTiming((v) => !v)}
                  aria-expanded={showTiming}
                >
                  {showTiming ? "Hide" : "Show"}
                </button>
              </div>
              {showTiming && (
                <div className="space-y-1">
                  <CardRow
                    label="Discover → Offer"
                    value={formatTime(timing.discover)}
                    status={getTimingStatus(timing.discover, t.perPhase)}
                  />
                  <CardRow
                    label="Offer → Request"
                    value={formatTime(timing.offer)}
                    status={getTimingStatus(timing.offer, t.perPhase)}
                  />
                  <CardRow
                    label="Request → Ack"
                    value={formatTime(timing.request)}
                    status={getTimingStatus(timing.request, t.perPhase)}
                  />
                  <div className="pt-1 border-t border-surface-border">
                    <CardRow
                      label="Total"
                      value={formatTime(timing.total)}
                      status={getTimingStatus(timing.total, t.total)}
                    />
                  </div>
                </div>
              )}
            </>
          )}

          {hasData && !timing && (
            <>
              <CardDivider />
              <p className="text-xs text-text-muted">
                DHCP timing not yet recorded
              </p>
            </>
          )}

          {/* Public IP Section */}
          {showPublicIP && publicip && (publicip.ipv4 || publicip.ipv6) && (
            <>
              <CardDivider />
              <p className="text-xs text-text-muted mb-1 font-medium">
                Public IP
              </p>
              {publicip.ipv4 && <CardRow label="IPv4" value={publicip.ipv4} />}
              {publicip.ipv6 && (
                <CardRow
                  label="IPv6"
                  value={compressIPv6(publicip.ipv6)}
                  wrap
                  mono
                  align="right"
                />
              )}
              {publicip.error && (
                <p className="text-xs text-status-error mt-1">
                  {publicip.error}
                </p>
              )}
            </>
          )}
        </>
      )}
    </Card>
  );
}
