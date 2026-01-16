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

import { Server } from "lucide-react";
import type React from "react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { formatBytes } from "../../lib/format";
import { cn, icon as iconTokens, radius, spacing } from "../../styles/theme";
import { CardDivider, CardRow } from "../ui/Card";
import type { Status } from "../ui/StatusBadge";
import { BaseCard } from "./BaseCard";

interface ProcessInfo {
  name: string;
  pid: number;
  cpuPercent: number;
  memoryMb: number;
}

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
  topCpuProcesses?: ProcessInfo[];
  topMemoryProcesses?: ProcessInfo[];
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
  if (percent >= 90) {
    return "error";
  }
  if (percent >= 75) {
    return "warning";
  }
  return "success";
}

/**
 * Returns contextual remediation suggestions based on resource type and usage level
 */
function getSuggestion(type: "cpu" | "memory" | "disk", usage: number): string {
  if (type === "cpu") {
    if (usage >= 90) {
      return "Check for runaway processes or consider upgrading CPU resources";
    }
    return "Consider closing unused applications or background tasks";
  }

  if (type === "memory") {
    if (usage >= 90) {
      return "Critical: Restart applications to free memory or add more RAM";
    }
    return "Consider increasing system memory or closing memory-intensive applications";
  }

  if (type === "disk") {
    if (usage >= 90) {
      return "Critical: Clear temporary files and archive old data immediately";
    }
    return "Clear temporary files, remove unused applications, or archive old data";
  }

  return "";
}

function _resourceBar({
  label,
  percent,
  used,
  total,
  topProcesses,
  type,
}: {
  label: string;
  percent: number;
  used: number;
  total: number;
  topProcesses?: ProcessInfo[];
  type: "cpu" | "memory" | "disk";
}): React.ReactElement {
  const status = getResourceStatus(percent);
  const barColor = ((): string => {
    switch (status) {
      case "success":
        return "bg-status-success";
      case "warning":
        return "bg-status-warning";
      case "error":
        return "bg-status-error";
      default:
        return "bg-text-muted";
    }
  })();

  const showConsumers = topProcesses && topProcesses.length > 0 && percent >= 75;

  return (
    <div class="stack-xs">
      <div class="flex justify-between caption">
        <span>{label}</span>
        <span class="text-text-primary font-medium">{percent.toFixed(0)}%</span>
      </div>
      <div class={cn("h-2 bg-surface-border overflow-hidden", radius.md)}>
        <div
          class={cn("h-full transition-all duration-300", barColor)}
          style={{ width: `${Math.min(percent, 100)}%` }}
        />
      </div>
      {used > 0 && total > 0 ? (
        <div class="caption">
          {formatBytes(used)} / {formatBytes(total)}
        </div>
      ) : null}
      {showConsumers ? (
        <div class="caption text-text-muted pl-3 mt-1">
          <div>Top consumers:</div>
          {topProcesses.slice(0, 3).map((proc) => (
            <div key={proc.pid} class="pl-2">
              - {proc.name} ({Math.round(proc.memoryMb)} MB)
            </div>
          ))}
        </div>
      ) : null}
      {percent >= 75 ? (
        <div class="mt-2 text-xs text-text-muted">
          <span class="font-medium">Tip:</span> {getSuggestion(type, percent)}
        </div>
      ) : null}
    </div>
  );
}

/**
 * Displays system resource usage with CPU, memory, and disk metrics.
 */
export function SystemHealthCard(): React.ReactElement {
  const { t } = useTranslation("cards");
  const [data, setData] = useState<SystemHealth | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchHealth = useCallback(async () => {
    try {
      const response = await fetch("/api/v1/sap/system/health", {
        credentials: "include",
      });
      if (response.status === 401) {
        // Trigger session refresh - dispatch custom event for app-level handling
        window.dispatchEvent(new CustomEvent("session-expired"));
        return; // Don't treat as error, let session refresh handle it
      }
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      // biome-ignore lint/nursery/useAwaitThenable: response.json() returns a Promise
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
    fetchHealth().catch(() => undefined);
    const interval = setInterval(() => {
      fetchHealth().catch(() => undefined);
    }, 5000);
    return (): void => clearInterval(interval);
  }, [fetchHealth]);

  const getStatus = (health: SystemHealth): Status => {
    const maxPercent = Math.max(
      health.cpuPercent ?? 0,
      health.memoryPercent ?? 0,
      health.diskPercent ?? 0,
    );
    return getResourceStatus(maxPercent);
  };

  return (
    <BaseCard
      title={t("system.title")}
      subtitle={data?.hostname}
      icon={<Server class={iconTokens.size.md} />}
      data={data}
      loading={loading}
      error={error}
      getStatus={getStatus}
    >
      {/* biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Resource bars require conditional rendering */}
      {(health: SystemHealth): React.ReactElement => (
        <div class="stack">
          <resourceBar
            label={t("system.cpu")}
            percent={health.cpuPercent ?? 0}
            used={0}
            total={0}
            topProcesses={health.topCpuProcesses}
            type="cpu"
          />
          <resourceBar
            label={t("system.memory")}
            percent={health.memoryPercent ?? 0}
            used={health.memoryUsed ?? 0}
            total={health.memoryTotal ?? 0}
            topProcesses={health.topMemoryProcesses}
            type="memory"
          />
          <resourceBar
            label={t("system.disk")}
            percent={health.diskPercent ?? 0}
            used={health.diskUsed ?? 0}
            total={health.diskTotal ?? 0}
            type="disk"
          />

          <CardDivider />

          <div class={cn("grid grid-cols-2", spacing.gap.compact)}>
            <CardRow
              label={t("system.uptime")}
              value={formatUptime(health.uptime ?? 0)}
              align="left"
            />
            <CardRow
              label={t("system.load1m")}
              value={(health.loadAvg1 ?? 0).toFixed(2)}
              align="left"
              status={(health.loadAvg1 ?? 0) > (health.numCpu ?? 1) ? "warning" : undefined}
            />
            <CardRow label={t("system.goroutines")} value={health.goroutines ?? 0} align="left" />
            <CardRow
              label={t("system.processMem")}
              value={formatBytes(health.processMemory ?? 0)}
              align="left"
            />
          </div>

          <div class={cn("caption text-center", spacing.padding.top.tight)}>
            {health.os ?? "Unknown"}/{health.arch ?? "Unknown"} - {health.numCpu ?? 0} CPUs
          </div>
        </div>
      )}
    </BaseCard>
  );
}
