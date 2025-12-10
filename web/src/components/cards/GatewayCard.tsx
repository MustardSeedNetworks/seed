import { Card, CardValue, CardRow, CardDivider, Status } from "../ui/Card";
import { StatusBadge } from "../ui/StatusBadge";
import { useSettings } from "../../contexts/SettingsContext";

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
  thresholds: { warning: number; critical: number },
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

export function GatewayCard({ data, loading }: GatewayCardProps) {
  const { thresholds } = useSettings();
  // Map context ThresholdPair (good/warning) to card format (warning/critical)
  const t = {
    warning: thresholds.gateway.good,
    critical: thresholds.gateway.warning,
  };

  if (loading) {
    return (
      <Card title="Gateway" status="loading">
        <CardValue value="Pinging..." size="lg" />
      </Card>
    );
  }

  // Check if no gateways detected (neither IPv4 nor IPv6)
  const hasIPv4Gateway = data && data.gateway;
  const hasIPv6Gateway = data && data.ipv6 && data.ipv6.gateway;

  if (!data || (!hasIPv4Gateway && !hasIPv6Gateway)) {
    return (
      <Card title="Gateway" status="unknown">
        <CardValue value="No gateway" size="md" />
        <p className="text-xs text-text-muted mt-1">
          Unable to detect default gateway
        </p>
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
      status = data.reachable ? getLatencyStatus(data.avgTime, t) : "error";
  }

  return (
    <Card title="Gateway" status={status}>
      <div className="flex items-center justify-between gap-2">
        <CardValue value={data.gateway} size="lg" />
        <StatusBadge status={data.reachable ? "success" : "error"} size="sm" />
      </div>
      <CardDivider />

      {/* Latency stats */}
      <div className="grid grid-cols-3 gap-2 mb-2">
        <div className="text-center">
          <p className="text-xs text-text-muted">Min</p>
          <p
            className={`text-sm font-medium ${
              data.minTime > 0
                ? getLatencyStatus(data.minTime, t) === "success"
                  ? "text-status-success"
                  : getLatencyStatus(data.minTime, t) === "warning"
                    ? "text-status-warning"
                    : "text-status-error"
                : "text-text-muted"
            }`}
          >
            {data.minTime > 0 ? formatTime(data.minTime) : "-"}
          </p>
        </div>
        <div className="text-center">
          <p className="text-xs text-text-muted">Avg</p>
          <p
            className={`text-sm font-medium ${
              data.avgTime > 0
                ? getLatencyStatus(data.avgTime, t) === "success"
                  ? "text-status-success"
                  : getLatencyStatus(data.avgTime, t) === "warning"
                    ? "text-status-warning"
                    : "text-status-error"
                : "text-text-muted"
            }`}
          >
            {data.avgTime > 0 ? formatTime(data.avgTime) : "-"}
          </p>
        </div>
        <div className="text-center">
          <p className="text-xs text-text-muted">Max</p>
          <p
            className={`text-sm font-medium ${
              data.maxTime > 0
                ? getLatencyStatus(data.maxTime, t) === "success"
                  ? "text-status-success"
                  : getLatencyStatus(data.maxTime, t) === "warning"
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
        label="Packets"
        value={`${data.received}/${data.sent}`}
        status={
          data.lossPercent === 0
            ? "success"
            : data.lossPercent < 50
              ? "warning"
              : "error"
        }
      />
      {data.lossPercent > 0 && (
        <CardRow
          label="Packet Loss"
          value={`${Math.round(data.lossPercent)}%`}
          status={data.lossPercent >= 50 ? "error" : "warning"}
        />
      )}

      {/* IPv6 Gateway Section */}
      {data.ipv6 && data.ipv6.gateway && (
        <>
          <CardDivider />
          <p className="text-xs text-text-muted mb-1 font-medium">
            IPv6 Gateway
          </p>
          <CardValue value={data.ipv6.gateway} size="md" />
          <p className="text-xs text-text-muted mb-2">
            {data.ipv6.reachable ? "Reachable" : "Unreachable"}
          </p>
          <div className="grid grid-cols-3 gap-2 mb-2">
            <div className="text-center">
              <p className="text-xs text-text-muted">Min</p>
              <p
                className={`text-sm font-medium ${
                  data.ipv6.minTime > 0
                    ? getLatencyStatus(data.ipv6.minTime, t) === "success"
                      ? "text-status-success"
                      : getLatencyStatus(data.ipv6.minTime, t) === "warning"
                        ? "text-status-warning"
                        : "text-status-error"
                    : "text-text-muted"
                }`}
              >
                {data.ipv6.minTime > 0 ? formatTime(data.ipv6.minTime) : "-"}
              </p>
            </div>
            <div className="text-center">
              <p className="text-xs text-text-muted">Avg</p>
              <p
                className={`text-sm font-medium ${
                  data.ipv6.avgTime > 0
                    ? getLatencyStatus(data.ipv6.avgTime, t) === "success"
                      ? "text-status-success"
                      : getLatencyStatus(data.ipv6.avgTime, t) === "warning"
                        ? "text-status-warning"
                        : "text-status-error"
                    : "text-text-muted"
                }`}
              >
                {data.ipv6.avgTime > 0 ? formatTime(data.ipv6.avgTime) : "-"}
              </p>
            </div>
            <div className="text-center">
              <p className="text-xs text-text-muted">Max</p>
              <p
                className={`text-sm font-medium ${
                  data.ipv6.maxTime > 0
                    ? getLatencyStatus(data.ipv6.maxTime, t) === "success"
                      ? "text-status-success"
                      : getLatencyStatus(data.ipv6.maxTime, t) === "warning"
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
            label="Packets"
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
}
