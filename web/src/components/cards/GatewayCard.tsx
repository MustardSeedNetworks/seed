/**
 * GatewayCard Component
 *
 * Purpose: Monitors network gateway (default router) reachability via ping (ICMP).
 * Displays packet loss, latency statistics, and connection stability.
 *
 * Key Features:
 * - IPv4 and IPv6 gateway monitoring (dual-stack support)
 * - Latency statistics: min/max/avg time and last packet latency
 * - Packet loss percentage with color-coded status
 * - Latency thresholds from settings (warning/critical levels)
 * - Status derivation based on packet loss and latency
 * - Separate sections for IPv4 and IPv6 results (if available)
 *
 * Usage:
 * ```typescript
 * <GatewayCard
 *   data={gatewayData}
 *   loading={isPinging}
 * />
 * ```
 *
 * Dependencies: Card UI components, StatusBadge, useSettings hook, Router icon, theme utilities
 * State: Uses SettingsContext for threshold configuration, receives data from parent
 */

import { memo } from "react";
import { useTranslation } from "react-i18next";
import { Card, CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { StatusBadge } from "../ui/StatusBadge";
import { useSettings } from "../../contexts/useSettings";
import { Router } from "../ui/Icons";
import { icon as iconTokens, layout, spacing } from "../../styles/theme";

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
  ipv6?: GatewayData;
}

interface GatewayCardProps {
  data: GatewayData | null;
  loading?: boolean;
}

function getLatencyStatus(
  value: number,
  thresholds: { warning: number; critical: number }
): Status {
  if (value >= thresholds.critical) return "error";
  if (value >= thresholds.warning) return "warning";
  return "success";
}

function formatTime(ms: number): string {
  if (ms < 1) return "<1ms";
  if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.round(ms * 10) / 10}ms`;
}

export const GatewayCard = memo(function GatewayCard({ data, loading }: GatewayCardProps) {
  const { t: tr } = useTranslation("cards");
  const { thresholds } = useSettings();
  // Map context ThresholdPair (good/warning) to card format (warning/critical)
  const th = {
    warning: thresholds.gateway.good,
    critical: thresholds.gateway.warning,
  };

  if (loading) {
    return (
      <Card
        title={tr("gateway.title")}
        icon={<Router className={iconTokens.size.md} />}
        status="loading"
      >
        <CardValue value={tr("gateway.pinging")} size="lg" />
      </Card>
    );
  }

  // Check if no gateways detected (neither IPv4 nor IPv6)
  const hasIPv4Gateway = data && data.gateway;
  const hasIPv6Gateway = data && data.ipv6 && data.ipv6.gateway;

  if (!data || (!hasIPv4Gateway && !hasIPv6Gateway)) {
    return (
      <Card
        title={tr("gateway.title")}
        icon={<Router className={iconTokens.size.md} />}
        status="unknown"
      >
        <CardValue value={tr("gateway.noGateway")} size="md" />
        <p className={`caption ${spacing.margin.top.tight}`}>{tr("gateway.unableToDetect")}</p>
      </Card>
    );
  }

  // Map API status to card status
  let status: Status = "unknown";
  switch (data.status) {
    case "success":
      status = "success";
      break;
    case "warning":
      status = "warning";
      break;
    case "error":
      status = "error";
      break;
    default:
      status = data.reachable ? getLatencyStatus(data.avgTime, th) : "error";
  }

  return (
    <Card
      title={tr("gateway.title")}
      icon={<Router className={iconTokens.size.md} />}
      status={status}
    >
      <div className={layout.flex.between}>
        <CardValue value={data.gateway} size="lg" />
        <StatusBadge status={data.reachable ? "success" : "error"} size="sm" />
      </div>
      <CardDivider />

      {/* Latency stats */}
      <div className={`grid grid-cols-3 ${spacing.gap.compact} ${spacing.margin.bottom.inline}`}>
        <div className="text-center">
          <p className="caption">{tr("gateway.min")}</p>
          <p
            className={`body-small font-medium ${
              data.minTime > 0
                ? getLatencyStatus(data.minTime, th) === "success"
                  ? "text-status-success"
                  : getLatencyStatus(data.minTime, th) === "warning"
                    ? "text-status-warning"
                    : "text-status-error"
                : "text-text-muted"
            }`}
          >
            {data.minTime > 0 ? formatTime(data.minTime) : "-"}
          </p>
        </div>
        <div className="text-center">
          <p className="caption">{tr("gateway.avg")}</p>
          <p
            className={`body-small font-medium ${
              data.avgTime > 0
                ? getLatencyStatus(data.avgTime, th) === "success"
                  ? "text-status-success"
                  : getLatencyStatus(data.avgTime, th) === "warning"
                    ? "text-status-warning"
                    : "text-status-error"
                : "text-text-muted"
            }`}
          >
            {data.avgTime > 0 ? formatTime(data.avgTime) : "-"}
          </p>
        </div>
        <div className="text-center">
          <p className="caption">{tr("gateway.max")}</p>
          <p
            className={`body-small font-medium ${
              data.maxTime > 0
                ? getLatencyStatus(data.maxTime, th) === "success"
                  ? "text-status-success"
                  : getLatencyStatus(data.maxTime, th) === "warning"
                    ? "text-status-warning"
                    : "text-status-error"
                : "text-text-muted"
            }`}
          >
            {data.maxTime > 0 ? formatTime(data.maxTime) : "-"}
          </p>
        </div>
      </div>

      <CardRow
        label={tr("gateway.packets")}
        value={`${data.received}/${data.sent}`}
        status={data.lossPercent === 0 ? "success" : data.lossPercent < 50 ? "warning" : "error"}
      />
      {data.lossPercent > 0 && (
        <CardRow
          label={tr("gateway.packetLoss")}
          value={`${Math.round(data.lossPercent)}%`}
          status={data.lossPercent >= 50 ? "error" : "warning"}
        />
      )}

      {/* IPv6 Gateway Section */}
      {data.ipv6 && data.ipv6.gateway && (
        <>
          <CardDivider />
          <p className={`caption ${spacing.margin.bottom.tight} font-medium`}>
            {tr("gateway.ipv6Gateway")}
          </p>
          <CardValue value={data.ipv6.gateway} size="md" />
          <p className={`caption ${spacing.margin.bottom.inline}`}>
            {data.ipv6.reachable ? tr("gateway.reachable") : tr("gateway.unreachable")}
          </p>
          <div
            className={`grid grid-cols-3 ${spacing.gap.compact} ${spacing.margin.bottom.inline}`}
          >
            <div className="text-center">
              <p className="caption">{tr("gateway.min")}</p>
              <p
                className={`body-small font-medium ${
                  data.ipv6.minTime > 0
                    ? getLatencyStatus(data.ipv6.minTime, th) === "success"
                      ? "text-status-success"
                      : getLatencyStatus(data.ipv6.minTime, th) === "warning"
                        ? "text-status-warning"
                        : "text-status-error"
                    : "text-text-muted"
                }`}
              >
                {data.ipv6.minTime > 0 ? formatTime(data.ipv6.minTime) : "-"}
              </p>
            </div>
            <div className="text-center">
              <p className="caption">{tr("gateway.avg")}</p>
              <p
                className={`body-small font-medium ${
                  data.ipv6.avgTime > 0
                    ? getLatencyStatus(data.ipv6.avgTime, th) === "success"
                      ? "text-status-success"
                      : getLatencyStatus(data.ipv6.avgTime, th) === "warning"
                        ? "text-status-warning"
                        : "text-status-error"
                    : "text-text-muted"
                }`}
              >
                {data.ipv6.avgTime > 0 ? formatTime(data.ipv6.avgTime) : "-"}
              </p>
            </div>
            <div className="text-center">
              <p className="caption">{tr("gateway.max")}</p>
              <p
                className={`body-small font-medium ${
                  data.ipv6.maxTime > 0
                    ? getLatencyStatus(data.ipv6.maxTime, th) === "success"
                      ? "text-status-success"
                      : getLatencyStatus(data.ipv6.maxTime, th) === "warning"
                        ? "text-status-warning"
                        : "text-status-error"
                    : "text-text-muted"
                }`}
              >
                {data.ipv6.maxTime > 0 ? formatTime(data.ipv6.maxTime) : "-"}
              </p>
            </div>
          </div>
          <CardRow
            label={tr("gateway.packets")}
            value={`${data.ipv6.received}/${data.ipv6.sent}`}
            status={
              data.ipv6.lossPercent === 0
                ? "success"
                : data.ipv6.lossPercent < 50
                  ? "warning"
                  : "error"
            }
          />
        </>
      )}
    </Card>
  );
});
