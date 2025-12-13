import { useEffect, useState, useCallback } from "react";
import { Server } from "lucide-react";
import { BaseCard } from "./BaseCard";
import { CardRow, CardDivider } from "../ui/Card";
import { Status } from "../ui/StatusBadge";
import { getAuthHeaders } from "../../hooks/useAuth";

interface SystemHealth {
  cpuPercent: number;
  memoryPercent: number;
  memoryUsed: number;
  memoryTotal: number;
  diskPercent: number;
  diskUsed: number;
  diskTotal: number;
  uptime: number;
  loadAvg1: number;
  loadAvg5: number;
  loadAvg15: number;
  goroutines: number;
  processMemory: number;
  hostname: string;
  os: string;
  arch: string;
  numCpu: number;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
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
  const barColor = {
    success: "bg-status-success",
    warning: "bg-status-warning",
    error: "bg-status-error",
    unknown: "bg-text-muted",
    loading: "bg-text-muted",
  }[status];

  return (
    <div className="space-y-1">
      <div className="flex justify-between text-xs">
        <span className="text-text-muted">{label}</span>
        <span className="text-text-primary font-medium">
          {percent.toFixed(0)}%
        </span>
      </div>
      <div className="h-2 bg-surface-border rounded-full overflow-hidden">
        <div
          className={`h-full ${barColor} transition-all duration-300`}
          style={{ width: `${Math.min(percent, 100)}%` }}
        />
      </div>
      <div className="text-xs text-text-muted">
        {formatBytes(used)} / {formatBytes(total)}
      </div>
    </div>
  );
}

export function SystemHealthCard() {
  const [data, setData] = useState<SystemHealth | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchHealth = useCallback(async () => {
    try {
      const response = await fetch("/api/system/health", {
        headers: getAuthHeaders(),
      });
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      const result = await response.json();
      setData(result);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch");
    } finally {
      setLoading(false);
    }
  }, [getAuthHeaders]);

  useEffect(() => {
    fetchHealth();
    const interval = setInterval(fetchHealth, 5000);
    return () => clearInterval(interval);
  }, [fetchHealth]);

  const getStatus = (health: SystemHealth): Status => {
    const maxPercent = Math.max(
      health.cpuPercent,
      health.memoryPercent,
      health.diskPercent,
    );
    return getResourceStatus(maxPercent);
  };

  return (
    <BaseCard
      title="System Health"
      subtitle={data?.hostname}
      icon={<Server className="w-5 h-5" />}
      data={data}
      loading={loading}
      error={error}
      getStatus={getStatus}
    >
      {(health) => (
        <div className="space-y-3">
          <ResourceBar
            label="CPU"
            percent={health.cpuPercent}
            used={0}
            total={0}
          />
          <ResourceBar
            label="Memory"
            percent={health.memoryPercent}
            used={health.memoryUsed}
            total={health.memoryTotal}
          />
          <ResourceBar
            label="Disk"
            percent={health.diskPercent}
            used={health.diskUsed}
            total={health.diskTotal}
          />

          <CardDivider />

          <div className="grid grid-cols-2 gap-2">
            <CardRow
              label="Uptime"
              value={formatUptime(health.uptime)}
              align="left"
            />
            <CardRow
              label="Load (1m)"
              value={health.loadAvg1.toFixed(2)}
              align="left"
              status={health.loadAvg1 > health.numCpu ? "warning" : undefined}
            />
            <CardRow
              label="Goroutines"
              value={health.goroutines}
              align="left"
            />
            <CardRow
              label="Process Mem"
              value={formatBytes(health.processMemory)}
              align="left"
            />
          </div>

          <div className="text-xs text-text-muted text-center pt-1">
            {health.os}/{health.arch} - {health.numCpu} CPUs
          </div>
        </div>
      )}
    </BaseCard>
  );
}
