/**
 * DNSCard Component
 *
 * Purpose: Comprehensive DNS diagnostic card showing forward/reverse lookups, DNS server testing,
 * and per-server resolution times. Detects DNS configuration issues and performance problems.
 *
 * Key Features:
 * - Tests forward (hostname→IP) and reverse (IP→hostname) lookups for IPv4 and IPv6
 * - Measures DNS query latency with configurable thresholds (warning/critical)
 * - Per-server testing: tests each configured DNS server independently
 * - Displays resolved IP addresses and error messages for failed lookups
 * - CollapsibleSection for each lookup type with detailed results
 * - Status color-coding based on response times and threshold settings
 *
 * Usage:
 * ```typescript
 * <DNSCard
 *   data={dnsTestData}
 *   loading={isRunning}
 * />
 * ```
 *
 * Dependencies: Card UI components, StatusBadge, CollapsibleSection, useSettings hook, Icons, theme utilities
 * State: Receives test data and thresholds from parent component and settings context
 */

import { memo } from "react";
import { useTranslation } from "react-i18next";
import { Card, CardValue, CardDivider, Status } from "../ui/Card";
import { StatusBadge } from "../ui/StatusBadge";
import { CollapsibleSection } from "../ui/CollapsibleSection";
import { Globe } from "../ui/Icons";
import { icon as iconTokens, layout } from "../../styles/theme";

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

function LookupRow({ label, lookup }: { label: string; lookup: LookupResult | null | undefined }) {
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
      <div className={layout.flex.between}>
        <span className="caption">{label}</span>
        <span className={layout.inline.default}>
          <StatusBadge status={statusBadge} size="sm" />
          <span className={`caption font-medium ${statusColor}`}>
            {formatTime(lookup.timeMs || lookup.time)}
          </span>
        </span>
      </div>
      <p className="body-small truncate" title={lookup.result}>
        {lookup.result}
      </p>
    </div>
  );
}

export const DNSCard = memo(function DNSCard({ data, loading }: DNSCardProps) {
  const { t } = useTranslation("cards");

  if (loading) {
    return (
      <Card title={t("dns.title")} icon={<Globe className={iconTokens.size.md} />} status="loading">
        <CardValue value={t("dns.testing")} size="lg" />
      </Card>
    );
  }

  if (!data) {
    return (
      <Card title={t("dns.title")} icon={<Globe className={iconTokens.size.md} />} status="unknown">
        <CardValue value={t("dns.noData")} size="md" />
      </Card>
    );
  }

  // Determine overall status based on forward/reverse lookups
  let overallStatus: Status = "success";
  const lookups = [data.forward, data.forwardIpv6, data.reverse, data.reverseIpv6];
  if (lookups.some((l) => l?.status === "error")) {
    overallStatus = "error";
  } else if (lookups.some((l) => l?.status === "warning")) {
    overallStatus = "warning";
  }

  // Show all DNS servers if available
  const servers = data.servers && data.servers.length > 0 ? data.servers : [data.server];

  return (
    <Card
      title={t("dns.title")}
      icon={<Globe className={iconTokens.size.md} />}
      status={overallStatus}
    >
      {/* DNS Servers */}
      <div className="mb-2">
        <p className="caption mb-1">{t("dns.dnsServers")}</p>
        <div className="stack-xs">
          {servers.map((server, idx) => (
            <p key={idx} className="body-small font-mono break-all" title={server}>
              {server}
            </p>
          ))}
        </div>
      </div>

      <p className="caption">{t("dns.testingHost", { hostname: data.testHostname })}</p>
      <CardDivider />

      {/* IPv4 Lookups */}
      {(data.forward || data.reverse) && (
        <div className="mb-2">
          <p className="caption font-medium mb-1">IPv4</p>
          <LookupRow label={t("dns.forwardA")} lookup={data.forward} />
          <LookupRow label={t("dns.reversePTR")} lookup={data.reverse} />
        </div>
      )}

      {/* IPv6 Lookups */}
      {(data.forwardIpv6 || data.reverseIpv6) && (
        <>
          <CardDivider />
          <div>
            <p className="caption font-medium mb-1">IPv6</p>
            <LookupRow label={t("dns.forwardAAAA")} lookup={data.forwardIpv6} />
            <LookupRow label={t("dns.reversePTR")} lookup={data.reverseIpv6} />
          </div>
        </>
      )}

      {/* Per-Server Results (collapsible) */}
      {data.perServerResults && data.perServerResults.length > 0 && (
        <>
          <CardDivider />
          <CollapsibleSection
            title={t("dns.serverTests")}
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
                <div className={`${layout.flex.between} mb-1`}>
                  <span className="caption font-mono">{server.server}</span>
                  <span
                    className={`caption font-medium ${
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
                  <div className={`${layout.flex.between} caption`}>
                    <span>A</span>
                    <span className={layout.inline.default}>
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
                  <div className={`${layout.flex.between} caption`}>
                    <span>AAAA</span>
                    <span className={layout.inline.default}>
                      <StatusBadge status={server.forwardIpv6.status} size="sm" />
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
