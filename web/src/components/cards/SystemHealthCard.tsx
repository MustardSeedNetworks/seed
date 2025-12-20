/**
 * SystemHealthCard Component
 *
 * Purpose: Monitors system resources (CPU, memory, disk usage) and OS information.
 * Displays real-time health metrics with status indicators and formatted values.
 *
 * Key Features:
 * - CPU monitoring: CPU percentage usage, load averages (1/5/15 min)
 * - Memory usage: percentage, used/total bytes with human-readable formatting
 * - Disk usage: percentage, used/total bytes with formatting
 * - System info: hostname, OS, architecture, CPU count, goroutines
 * - Uptime: displays in human-readable format (days + hours, hours + minutes, or minutes)
 * - Process info: memory usage of the The Seed process itself
 * - Threshold-based status: warning/critical levels from settings context
 * - Real-time updates: fetches metrics periodically from API
 *
 * Usage:
 * ```typescript
 * <SystemHealthCard
 *   data={systemHealth}
 *   loading={isFetching}
 * />
 * ```
 *
 * Dependencies: BaseCard, Card UI components, useSettings hook, auth hooks, Icons, theme utilities
 * State: Manages system health data, fetches from /api/status/system endpoint, updates periodically
 */

import { useEffect, useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Server } from "lucide-react";
import { radius, icon as iconTokens, spacing } from "../../styles/theme";
import { BaseCard } from "./BaseCard";
import { CardRow, CardDivider } from "../ui/Card";
import { Status } from "../ui/StatusBadge";
// Fix #669: Removed deprecated getAuthHeaders - using credentials: 'include' for cookie auth

interface SystemHealth {
  cpuPercent?: number;
  memoryPercent?: number;
  memoryUsed?: number;
  memoryTotal?: number;
  diskPercent?: number;
  diskUsed?: number;
  diskTotal?: number;
  uptime?: number;
  loadAvg1?: number;
  loadAvg5?: number;
  loadAvg15?: number;
  goroutines?: number;
  processMemory: number;
  hostname: string;
  os: string;
  arch: string;
  numCpu: number;
}

/**
 * Type-safe getter for byte size units
 */
function getSizeUnit(index: number): string {
  switch (index) {
    case 0:
      return "B";
    case 1:
      return "KB";
    case 2:
      return "MB";
    case 3:
      return "GB";
    case 4:
      return "TB";
    default:
      return index < 0 ? "B" : "TB";
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const unit = getSizeUnit(i);
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${unit}`;
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);

  if (days > 0) {
    return `${days}d ${hours}h`;
  }
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  return `${minutes}m`;
}

function getResourceStatus(percent: number): Status {
  if (percent >= 90) return "error";
  if (percent >= 75) return "warning";
  return "success";
}

function ResourceBar({
  label,
  percent,
  used,
  total,
}: {
  label: string;
  percent: number;
  used: number;
  total: number;
}) {
  const status = getResourceStatus(percent);
  const barColor = (() => {
    switch (status) {
      case "success":
        return "bg-status-success";
      case "warning":
        return "bg-status-warning";
      case "error":
        return "bg-status-error";
      case "unknown":
      case "loading":
        return "bg-text-muted";
    }
  })();

  return (
    <div className="stack-xs">
      <div className="flex justify-between caption">
        <span>{label}</span>
        <span className="text-text-primary font-medium">
          {percent.toFixed(0)}%
        </span>
      </div>
      <div className={`h-2 bg-surface-border ${radius.md} overflow-hidden`}>
        <div
          className={`h-full ${barColor} transition-all duration-300`}
          style={{ width: `${Math.min(percent, 100)}%` }}
        />
      </div>
      <div className="caption">
        {formatBytes(used)} / {formatBytes(total)}
      </div>
    </div>
  );
}

/**
 * Displays system resource usage with CPU, memory, and disk metrics.
 */
export function SystemHealthCard() {
  const { t } = useTranslation("cards");
  const [data, setData] = useState<SystemHealth | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchHealth = useCallback(async () => {
    try {
      const response = await fetch("/api/system/health", {
        credentials: "include",
      });
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      const result = await response.json();
      setData(result.system);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchHealth();
    const interval = setInterval(fetchHealth, 5000);
    return () => clearInterval(interval);
  }, [fetchHealth]);

  const getStatus = (health: SystemHealth): Status => {
    const maxPercent = Math.max(
      health.cpuPercent ?? 0,
      health.memoryPercent ?? 0,
      health.diskPercent ?? 0
    );
    return getResourceStatus(maxPercent);
  };

  return (
    <BaseCard
      title={t("system.title")}
      subtitle={data?.hostname}
      icon={<Server className={iconTokens.size.md} />}
      data={data}
      loading={loading}
      error={error}
      getStatus={getStatus}
    >
      {(health) => (
        <div className="stack">
          <ResourceBar
            label={t("system.cpu")}
            percent={health.cpuPercent ?? 0}
            used={0}
            total={0}
          />
          <ResourceBar
            label={t("system.memory")}
            percent={health.memoryPercent ?? 0}
            used={health.memoryUsed ?? 0}
            total={health.memoryTotal ?? 0}
          />
          <ResourceBar
            label={t("system.disk")}
            percent={health.diskPercent ?? 0}
            used={health.diskUsed ?? 0}
            total={health.diskTotal ?? 0}
          />

          <CardDivider />

          <div className={`grid grid-cols-2 ${spacing.gap.compact}`}>
            <CardRow
              label={t("system.uptime")}
              value={formatUptime(health.uptime ?? 0)}
              align="left"
            />
            <CardRow
              label={t("system.load1m")}
              value={(health.loadAvg1 ?? 0).toFixed(2)}
              align="left"
              status={
                (health.loadAvg1 ?? 0) > (health.numCpu ?? 1)
                  ? "warning"
                  : undefined
              }
            />
            <CardRow
              label={t("system.goroutines")}
              value={health.goroutines ?? 0}
              align="left"
            />
            <CardRow
              label={t("system.processMem")}
              value={formatBytes(health.processMemory ?? 0)}
              align="left"
            />
          </div>

          <div className={`caption text-center ${spacing.padding.top.tight}`}>
            {health.os ?? "Unknown"}/{health.arch ?? "Unknown"} -{" "}
            {health.numCpu ?? 0} CPUs
          </div>
        </div>
      )}
    </BaseCard>
  );
}
