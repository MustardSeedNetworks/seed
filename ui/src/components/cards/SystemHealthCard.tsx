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
 * - Process info: memory usage of the Seed process itself
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
 * State: Manages system health data, fetches from /api/v1/telemetry/system/health, updates periodically
 */

import { Server } from 'lucide-react';
import type React from 'react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { formatBytes } from '../../lib/format';
import { cn, icon as iconTokens, radius, spacing, status as statusColor } from '../../styles/theme';
import type {
  Health,
  ProcessInfo,
  SystemHealthResponse,
} from '../../types/generated/system-health-response';
import { CardDivider, CardRow } from '../ui/Card';
import type { Status } from '../ui/StatusBadge';
import { BaseCard } from './BaseCard';

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
    return 'error';
  }
  if (percent >= 75) {
    return 'warning';
  }
  return 'success';
}

/**
 * Returns contextual remediation suggestions based on resource type and usage level
 */
/** The suggestion for a resource under pressure; 90% is the critical line. */
const SUGGESTION_KEYS = {
  cpu: ['system.suggestion.cpuHigh', 'system.suggestion.cpuCritical'],
  memory: ['system.suggestion.memoryHigh', 'system.suggestion.memoryCritical'],
  disk: ['system.suggestion.diskHigh', 'system.suggestion.diskCritical'],
} as const;

function ResourceBar({
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
  type: 'cpu' | 'memory' | 'disk';
}): React.ReactElement {
  const { t } = useTranslation('cards');
  const status = getResourceStatus(percent);
  const barColor = ((): string => {
    switch (status) {
      case 'success':
        return statusColor.bg.success;
      case 'warning':
        return statusColor.bg.warning;
      case 'error':
        return statusColor.bg.error;
      default:
        return 'bg-text-muted';
    }
  })();

  const showConsumers = topProcesses && topProcesses.length > 0 && percent >= 75;

  return (
    <div className="stack-xs">
      <div className="flex justify-between caption">
        <span>{label}</span>
        <span className="text-text-primary font-medium">{percent.toFixed(0)}%</span>
      </div>
      <div className={cn('h-2 bg-surface-border overflow-hidden', radius.md)}>
        <div
          className={cn('h-full transition-all duration-300', barColor)}
          style={{ width: `${Math.min(percent, 100)}%` }}
        />
      </div>
      {used > 0 && total > 0 ? (
        <div className="caption">
          {formatBytes(used)} / {formatBytes(total)}
        </div>
      ) : null}
      {showConsumers ? (
        <div className="caption text-text-muted pl-3 mt-tight">
          <div>{t('systemHealth.topConsumers')}</div>
          {topProcesses.slice(0, 3).map((proc) => (
            // A process name is a command line, and a browser or a JVM can
            // produce one thousands of characters long with no space to break
            // at. Left to itself it pushes the whole page sideways — which is
            // what the 480px suite caught on CI, where chrome-headless-shell
            // runs with a 1600px argument list.
            <div key={proc.pid} className="pl-2 wrap-break-word">
              - {proc.name} ({Math.round(proc.memoryMb)} MB)
            </div>
          ))}
        </div>
      ) : null}
      {percent >= 75 ? (
        <div className="mt-inline text-xs text-text-muted">
          <span className="font-medium">{t('system.tip')}</span>{' '}
          {t(SUGGESTION_KEYS[type][percent >= 90 ? 1 : 0])}
        </div>
      ) : null}
    </div>
  );
}

/** The host's resource usage, or null on a 401 (left to the session refresh). */
async function fetchSystemHealth(): Promise<Health | null> {
  const response = await fetch('/api/v1/telemetry/system/health', {
    credentials: 'include',
  });
  if (response.status === 401) {
    // Trigger session refresh - dispatch custom event for app-level handling
    window.dispatchEvent(new CustomEvent('session-expired'));
    return null;
  }
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`);
  }
  const result: SystemHealthResponse = await response.json();
  return result.system;
}

/**
 * Displays system resource usage with CPU, memory, and disk metrics.
 */
export function SystemHealthCard(): React.ReactElement {
  const { t } = useTranslation('cards');
  const [data, setData] = useState<Health | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // A promise chain, not try/finally: the React Compiler cannot lower a
  // finally clause or a throw inside try, and would skip the whole component.
  const fetchHealth = (): Promise<void> =>
    fetchSystemHealth()
      .then((health) => {
        if (health) {
          setData(health);
          setError(null);
        }
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : t('system.loadFailed'));
      })
      .finally(() => setLoading(false));

  useEffect(() => {
    fetchHealth().catch(() => undefined);
    const interval = setInterval(() => {
      fetchHealth().catch(() => undefined);
    }, 5000);
    return (): void => clearInterval(interval);
  }, [fetchHealth]);

  const getStatus = (health: Health): Status => {
    const maxPercent = Math.max(health.cpuPercent, health.memoryPercent, health.diskPercent);
    return getResourceStatus(maxPercent);
  };

  return (
    <BaseCard
      title={t('system.title')}
      subtitle={data?.hostname}
      icon={<Server className={iconTokens.size.md} />}
      data={data}
      loading={loading}
      error={error}
      getStatus={getStatus}
    >
      {(health: Health): React.ReactElement => (
        <div className="stack">
          <ResourceBar
            label={t('system.cpu')}
            percent={health.cpuPercent}
            used={0}
            total={0}
            topProcesses={health.topCpuProcesses}
            type="cpu"
          />
          <ResourceBar
            label={t('system.memory')}
            percent={health.memoryPercent}
            used={health.memoryUsed}
            total={health.memoryTotal}
            topProcesses={health.topMemoryProcesses}
            type="memory"
          />
          <ResourceBar
            label={t('system.disk')}
            percent={health.diskPercent}
            used={health.diskUsed}
            total={health.diskTotal}
            type="disk"
          />

          <CardDivider />

          <div className={cn('grid grid-cols-2', spacing.gap.compact)}>
            <CardRow label={t('system.uptime')} value={formatUptime(health.uptime)} align="left" />
            <CardRow
              label={t('system.load1m')}
              value={health.loadAvg1.toFixed(2)}
              align="left"
              status={health.loadAvg1 > health.numCpu ? 'warning' : undefined}
            />
            <CardRow label={t('system.goroutines')} value={health.goroutines} align="left" />
            <CardRow
              label={t('system.processMem')}
              value={formatBytes(health.processMemory)}
              align="left"
            />
          </div>

          <div className={cn('caption text-center', spacing.padding.top.tight)}>
            {t('system.platform', {
              os: health.os,
              arch: health.arch,
              cpus: health.numCpu,
            })}
          </div>
        </div>
      )}
    </BaseCard>
  );
}
