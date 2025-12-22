/**
 * Network Configuration Card Component
 *
 * Displays DHCP/network configuration information including:
 * - IPv4 and IPv6 addresses and configuration
 * - MAC address and DHCP mode
 * - DHCP timing breakdown (discover, offer, request, ACK phases)
 * - Public IP information (if available)
 * - DNS servers
 * - Lease time information
 *
 * Features:
 * - Color-coded status based on DHCP timing thresholds
 * - Automatic unit formatting (ms vs seconds for times)
 * - Lease time human-readable formatting (days, hours)
 * - IPv6 scope and source type display
 * - Public IP information with last check timestamp
 *
 * The card is part of the main network monitoring dashboard and provides
 * insight into network layer 3 configuration and DHCP performance.
 */

import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { SimpleBaseCard } from "./BaseCard";
import { Network } from "../ui/Icons";
import {
  cn,
  layout,
  radius,
  icon as iconTokens,
  border,
  spacing,
} from "../../styles/theme";

/**
 * DHCP timing information for each phase of address assignment
 */
export interface DHCPTiming {
  discover: number; // Time for DISCOVER phase (ms)
  offer: number; // Time for OFFER phase (ms)
  request: number; // Time for REQUEST phase (ms)
  ack: number; // Time for ACK phase (ms)
  total: number; // Total DHCP negotiation time (ms)
}

/**
 * IPv4 address and configuration information
 */
export interface IPv4Info {
  address: string; // IPv4 address assigned
  subnet: string; // Subnet mask (CIDR notation)
  gateway: string | null; // Default gateway IP
  dhcpServer: string | null; // DHCP server IP that assigned address
  leaseTime: number | null; // Lease duration in seconds
}

/**
 * IPv6 address and configuration information
 */
export interface IPv6Info {
  address: string; // IPv6 address
  prefix: number; // Prefix length (0-128)
  scope: "global" | "link-local" | "unique-local"; // Address scope
  source: "slaac" | "dhcpv6" | "static" | "temporary"; // How address was configured
}

/**
 * DHCP configuration data from the backend
 */
export interface DHCPData {
  mac: string; // MAC address of interface
  mode: "dhcp" | "static" | "auto"; // Address assignment mode
  ipv4: IPv4Info | null; // IPv4 configuration (or null if not configured)
  ipv6: IPv6Info[]; // Array of IPv6 addresses
  dns: string[]; // DNS servers in use
  timing: DHCPTiming | null; // DHCP timing info (or null if not DHCP)
}

/**
 * Public IP and geolocation information
 */
export interface PublicIPInfo {
  ipv4?: string; // Public IPv4 address
  ipv6?: string; // Public IPv6 address
  lastChecked: string; // ISO 8601 timestamp of last check
  error?: string; // Error message if check failed
}

/**
 * Props for the DHCP/Network Card
 */
interface DHCPCardProps {
  data: DHCPData | null; // DHCP/network configuration data
  publicip?: PublicIPInfo | null; // Optional public IP information
  loading?: boolean; // True while loading data
  showPublicIP?: boolean; // Whether to display public IP info
  thresholds?: {
    total: { warning: number; critical: number };
    perPhase: { warning: number; critical: number };
  };
}

/**
 * Determines status indicator color based on DHCP timing thresholds.
 * Higher timing = more degradation.
 *
 * @param value - Timing value in milliseconds
 * @param thresholds - Warning and critical thresholds in ms
 * @returns Status color ('success', 'warning', 'error')
 */
function getTimingStatus(
  value: number,
  thresholds: { warning: number; critical: number }
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

// Scope labels are handled in the component using i18n

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

  for (const [i, group] of groups.entries()) {
    if (group === "0" || group === "0000") {
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

/**
 * Displays network interface information with IP addresses and connection status.
 */
export function NetworkCard({
  data,
  publicip,
  loading,
  showPublicIP = true,
  thresholds,
}: DHCPCardProps) {
  const { t: tr } = useTranslation("cards");
  const { t: tc } = useTranslation("common");

  const defaultThresholds = {
    total: { warning: 500, critical: 2000 },
    perPhase: { warning: 200, critical: 1000 },
  };
  const th = thresholds || defaultThresholds;
  const [showTiming, setShowTiming] = useState(false);

  // Keep hooks unconditional: derive safe fallbacks
  const hasData = !!data;
  const ipv4 = data?.ipv4 ?? null;
  const ipv6List = useMemo(() => data?.ipv6 ?? [], [data?.ipv6]);
  const timing = data?.timing ?? null;
  const hasIPv4 = ipv4 !== null;
  const hasIPv6 = ipv6List.length > 0;
  const globalIPv6 = ipv6List.filter((ip) => ip.scope === "global");

  const getScopeLabel = (scope: IPv6Info["scope"]): string => {
    switch (scope) {
      case "global":
        return tr("network.global");
      case "link-local":
        return tr("network.linkLocal");
      case "unique-local":
        return tr("network.ula");
      default:
        return scope;
    }
  };

  const groupedIPv6 = useMemo(() => {
    const order: IPv6Info["scope"][] = ["global", "unique-local", "link-local"];
    return order
      .map((scope) => ({
        scope,
        label: getScopeLabel(scope),
        entries: ipv6List.filter((ip) => ip.scope === scope),
      }))
      .filter((group) => group.entries.length > 0);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ipv6List, tr]);

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
        getTimingStatus(timing.discover, th.perPhase),
        getTimingStatus(timing.offer, th.perPhase),
        getTimingStatus(timing.request, th.perPhase),
        getTimingStatus(timing.ack, th.perPhase),
        getTimingStatus(timing.total, th.total),
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
    (loading
      ? tc("status.loading")
      : hasData
        ? tr("network.noIP")
        : tr("network.noData"));
  const primaryIP =
    primaryIPRaw && primaryIPRaw.includes(":")
      ? compressIPv6(primaryIPRaw)
      : primaryIPRaw;

  return (
    <SimpleBaseCard
      title={tr("network.title")}
      icon={<Network className={iconTokens.size.md} />}
      status={status}
      loading={loading}
    >
      <CardValue value={primaryIP} size="lg" mono allowWrap />

      <CardDivider />

      {!hasData && (
        <CardValue value={tr("network.noDataAvailable")} size="md" />
      )}

      {hasData && (
        <>
          {/* MAC Address */}
          <CardRow label={tr("network.mac")} value={data!.mac} />
          <CardRow
            label={tr("network.mode")}
            value={data!.mode.toUpperCase()}
          />

          {/* IPv4 Section */}
          {hasIPv4 && ipv4 && (
            <>
              <CardDivider />
              <p
                className={cn(
                  "caption font-medium",
                  spacing.margin.bottom.tight
                )}
              >
                {tr("network.ipv4")}
              </p>
              <CardRow
                label={tr("network.address")}
                value={`${ipv4.address}/${ipv4.subnet}`}
                wrap
                mono
              />
              {ipv4.gateway && (
                <CardRow
                  label={tr("network.gateway")}
                  value={ipv4.gateway}
                  wrap
                  mono
                />
              )}
              {ipv4.dhcpServer && (
                <CardRow
                  label={tr("network.dhcpServer")}
                  value={ipv4.dhcpServer}
                  wrap
                  mono
                />
              )}
              {ipv4.leaseTime && (
                <CardRow
                  label={tr("network.lease")}
                  value={formatLeaseTime(ipv4.leaseTime)}
                />
              )}
            </>
          )}

          {/* IPv6 Section */}
          {hasIPv6 && (
            <>
              <CardDivider />
              <p
                className={cn(
                  "caption font-medium",
                  spacing.margin.bottom.tight
                )}
              >
                {tr("network.ipv6")}
              </p>
              <div className="stack-sm">
                {groupedIPv6.map((group, groupIdx) => (
                  <div key={groupIdx} className="stack-xs">
                    <p className="text-2xs uppercase tracking-wide text-text-muted font-semibold">
                      {group.label}
                    </p>
                    {group.entries.map((ip, idx) => (
                      <CardRow
                        key={`${groupIdx}-${idx}`}
                        label={tr("network.address")}
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
              <div
                className={cn(layout.flex.between, spacing.margin.bottom.tight)}
              >
                <p className="caption font-medium">
                  {tr("network.dhcpTiming")}
                </p>
                {showTiming ? (
                  <button
                    type="button"
                    className={cn(
                      "caption font-medium text-brand-primary hover:text-brand-primary/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary",
                      spacing.actionBtn,
                      radius.default
                    )}
                    onClick={() => setShowTiming(false)}
                    aria-expanded="true"
                  >
                    {tc("buttons.hide")}
                  </button>
                ) : (
                  <button
                    type="button"
                    className={cn(
                      "caption font-medium text-brand-primary hover:text-brand-primary/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary",
                      spacing.actionBtn,
                      radius.default
                    )}
                    onClick={() => setShowTiming(true)}
                    aria-expanded="false"
                  >
                    {tc("buttons.show")}
                  </button>
                )}
              </div>
              {showTiming && (
                <div className="stack-xs">
                  <CardRow
                    label={tr("network.discoverOffer")}
                    value={formatTime(timing.discover)}
                    status={getTimingStatus(timing.discover, th.perPhase)}
                  />
                  <CardRow
                    label={tr("network.offerRequest")}
                    value={formatTime(timing.offer)}
                    status={getTimingStatus(timing.offer, th.perPhase)}
                  />
                  <CardRow
                    label={tr("network.requestAck")}
                    value={formatTime(timing.request)}
                    status={getTimingStatus(timing.request, th.perPhase)}
                  />
                  <div
                    className={cn(spacing.padding.top.tight, border.divider)}
                  >
                    <CardRow
                      label={tr("network.total")}
                      value={formatTime(timing.total)}
                      status={getTimingStatus(timing.total, th.total)}
                    />
                  </div>
                </div>
              )}
            </>
          )}

          {hasData && !timing && (
            <>
              <CardDivider />
              <p className="caption">{tr("network.notRecorded")}</p>
            </>
          )}

          {/* Public IP Section */}
          {showPublicIP && publicip && (publicip.ipv4 || publicip.ipv6) && (
            <>
              <CardDivider />
              <p
                className={cn(
                  "caption font-medium",
                  spacing.margin.bottom.tight
                )}
              >
                {tr("network.publicIp")}
              </p>
              {publicip.ipv4 && (
                <CardRow label={tr("network.ipv4")} value={publicip.ipv4} />
              )}
              {publicip.ipv6 && (
                <CardRow
                  label={tr("network.ipv6")}
                  value={compressIPv6(publicip.ipv6)}
                  wrap
                  mono
                  align="right"
                />
              )}
              {publicip.error && (
                <p
                  className={cn(
                    "caption text-status-error",
                    spacing.margin.top.tight
                  )}
                >
                  {publicip.error}
                </p>
              )}
            </>
          )}
        </>
      )}
    </SimpleBaseCard>
  );
}
