// biome-ignore-all lint/nursery/useAwaitThenable: response.json() returns a Promise
/**
 * Log Viewer Hook
 *
 * Manages log fetching, filtering, and real-time streaming for the log viewer component.
 *
 * Features:
 * - Real-time log streaming via WebSocket
 * - Filtering by level, layer, component, and search text
 * - Sorting by timestamp
 * - Pagination support
 * - Log statistics
 *
 * Usage:
 * ```typescript
 * const { logs, filters, setFilters, stats, isStreaming } = useLogs({
 *   maxLogs: 500,
 *   onMessage: handleWebSocketMessage
 * });
 * ```
 */

import { useCallback, useEffect, useMemo, useState } from "react";

/** Log severity levels */
export type LogLevel = "DEBUG" | "INFO" | "WARN" | "ERROR";

/** Log source layers */
export type LogLayer = "backend" | "api" | "frontend";

/** Log entry structure matching backend LogEntry */
export interface LogEntry {
  timestamp: string;
  level: LogLevel;
  layer: LogLayer;
  // biome-ignore lint/style/useNamingConvention: API response property names match backend schema
  request_id?: string;
  // biome-ignore lint/style/useNamingConvention: API response property names match backend schema
  session_id?: string;
  message: string;
  component?: string;
  // biome-ignore lint/style/useNamingConvention: API response property names match backend schema
  duration_ms?: number;
  metadata?: Record<string, unknown>;
  stack?: string;
}

/** Filter configuration for logs */
export interface LogFilters {
  levels: LogLevel[];
  layers: LogLayer[];
  components: string[];
  search: string;
}

/** Log statistics from backend */
export interface LogStats {
  // biome-ignore lint/style/useNamingConvention: API response property names match backend schema
  total_count: number;
  // biome-ignore lint/style/useNamingConvention: API response property names match backend schema
  by_level: Record<string, number>;
  // biome-ignore lint/style/useNamingConvention: API response property names match backend schema
  by_layer: Record<string, number>;
  // biome-ignore lint/style/useNamingConvention: API response property names match backend schema
  by_component: Record<string, number>;
  // biome-ignore lint/style/useNamingConvention: API response property names match backend schema
  errors_last_hour: number;
  // biome-ignore lint/style/useNamingConvention: API response property names match backend schema
  warnings_last_hour: number;
}

/** Configuration options for useLogs hook */
interface UseLogsOptions {
  /** Maximum number of logs to keep in memory (default: 1000) */
  maxLogs?: number;
  /** Initial filter configuration */
  initialFilters?: Partial<LogFilters>;
  /** Callback for WebSocket messages (to integrate with existing WebSocket) */
  onMessage?: (message: { type: string; payload: unknown }) => void;
}

/** Return value from useLogs hook */
interface UseLogsReturn {
  /** Filtered and sorted log entries */
  logs: LogEntry[];
  /** All unfiltered log entries */
  allLogs: LogEntry[];
  /** Current filter configuration */
  filters: LogFilters;
  /** Update filter configuration */
  setFilters: (filters: Partial<LogFilters>) => void;
  /** Reset filters to defaults */
  resetFilters: () => void;
  /** Log statistics */
  stats: LogStats | null;
  /** Whether real-time streaming is active */
  isStreaming: boolean;
  /** Toggle streaming on/off */
  setIsStreaming: (streaming: boolean) => void;
  /** Loading state */
  isLoading: boolean;
  /** Error state */
  error: string | null;
  /** Fetch logs from backend */
  fetchLogs: (limit?: number) => Promise<void>;
  /** Fetch log statistics */
  fetchStats: () => Promise<void>;
  /** Clear all logs from state */
  clearLogs: () => void;
  /** Add a log entry (used by WebSocket handler) */
  addLog: (entry: LogEntry) => void;
}

const DEFAULT_FILTERS: LogFilters = {
  levels: [],
  layers: [],
  components: [],
  search: "",
};

const DEFAULT_STATS: LogStats = {
  // biome-ignore lint/style/useNamingConvention: Mock data uses API response property names
  total_count: 0,
  // biome-ignore lint/style/useNamingConvention: Mock data uses API response property names
  by_level: {},
  // biome-ignore lint/style/useNamingConvention: Mock data uses API response property names
  by_layer: {},
  // biome-ignore lint/style/useNamingConvention: Mock data uses API response property names
  by_component: {},
  // biome-ignore lint/style/useNamingConvention: Mock data uses API response property names
  errors_last_hour: 0,
  // biome-ignore lint/style/useNamingConvention: Mock data uses API response property names
  warnings_last_hour: 0,
};

/**
 * Custom hook for managing log viewing with filtering and real-time updates.
 */
export function useLogs({
  maxLogs = 1000,
  initialFilters = {},
}: UseLogsOptions = {}): UseLogsReturn {
  const [allLogs, setAllLogs] = useState<LogEntry[]>([]);
  const [filters, setFiltersState] = useState<LogFilters>({
    ...DEFAULT_FILTERS,
    ...initialFilters,
  });
  const [stats, setStats] = useState<LogStats | null>(null);
  const [isStreaming, setIsStreaming] = useState(true);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  /**
   * Add a single log entry to the state.
   */
  const addLog = useCallback(
    (entry: LogEntry) => {
      if (!isStreaming) {
        return;
      }

      setAllLogs((prev) => {
        const newLogs = [...prev, entry];
        // Trim to maxLogs
        if (newLogs.length > maxLogs) {
          return newLogs.slice(-maxLogs);
        }
        return newLogs;
      });
    },
    [isStreaming, maxLogs],
  );

  /**
   * Fetch logs from the backend API.
   */
  const fetchLogs = useCallback(async (limit = 200) => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/harvest/logs/recent?limit=${limit}`);
      if (!response.ok) {
        throw new Error(`Failed to fetch logs: ${response.statusText}`);
      }

      const data = await response.json();
      setAllLogs(data.logs || []);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to fetch logs";
      setError(message);
    } finally {
      setIsLoading(false);
    }
  }, []);

  /**
   * Fetch log statistics from the backend.
   */
  const fetchStats = useCallback(async () => {
    try {
      const response = await fetch("/api/v1/harvest/logs/stats");
      if (!response.ok) {
        throw new Error(`Failed to fetch stats: ${response.statusText}`);
      }

      const data = await response.json();
      setStats(data);
    } catch {
      // Silently fail to avoid infinite loops in logging system
      // fixes #681 - removed console.error statement
      setStats(DEFAULT_STATS);
    }
  }, []);

  /**
   * Update filter configuration.
   */
  const setFilters = useCallback((newFilters: Partial<LogFilters>) => {
    setFiltersState((prev) => ({ ...prev, ...newFilters }));
  }, []);

  /**
   * Reset filters to defaults.
   */
  const resetFilters = useCallback(() => {
    setFiltersState(DEFAULT_FILTERS);
  }, []);

  /**
   * Clear all logs from state.
   */
  const clearLogs = useCallback(() => {
    setAllLogs([]);
  }, []);

  /**
   * Apply filters to logs and sort by timestamp (newest first).
   */
  const filteredLogs = useMemo(() => {
    let result = [...allLogs];

    // Filter by level
    if (filters.levels.length > 0) {
      result = result.filter((log) => filters.levels.includes(log.level));
    }

    // Filter by layer
    if (filters.layers.length > 0) {
      result = result.filter((log) => filters.layers.includes(log.layer));
    }

    // Filter by component
    if (filters.components.length > 0) {
      result = result.filter((log) => log.component && filters.components.includes(log.component));
    }

    // Filter by search text
    if (filters.search) {
      const searchLower = filters.search.toLowerCase();
      result = result.filter(
        (log) =>
          log.message.toLowerCase().includes(searchLower) ||
          log.component?.toLowerCase().includes(searchLower) ||
          log.request_id?.toLowerCase().includes(searchLower),
      );
    }

    // Sort by timestamp (newest first)
    result.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

    return result;
  }, [allLogs, filters]);

  // Fetch initial logs and stats on mount
  useEffect(() => {
    fetchLogs().catch(() => undefined);
    fetchStats().catch(() => undefined);
  }, [fetchLogs, fetchStats]);

  // Refresh stats periodically
  useEffect(() => {
    const interval = setInterval(fetchStats, 30000); // Every 30 seconds
    return () => clearInterval(interval);
  }, [fetchStats]);

  return {
    logs: filteredLogs,
    allLogs,
    filters,
    setFilters,
    resetFilters,
    stats,
    isStreaming,
    setIsStreaming,
    isLoading,
    error,
    fetchLogs,
    fetchStats,
    clearLogs,
    addLog,
  };
}

/**
 * Color configuration for log levels using design system tokens.
 */
export const LOG_LEVEL_COLORS = {
  // biome-ignore lint/style/useNamingConvention: LogLevel enum-like keys for color mapping
  ERROR: {
    bg: "bg-red-50 dark:bg-red-950",
    text: "text-red-700 dark:text-red-300",
    badge: "bg-status-error text-text-inverse",
    border: "border-l-4 border-status-error",
  },
  // biome-ignore lint/style/useNamingConvention: LogLevel enum-like keys for color mapping
  WARN: {
    bg: "bg-yellow-50 dark:bg-yellow-950",
    text: "text-yellow-700 dark:text-yellow-300",
    badge: "bg-status-warning text-text-inverse",
    border: "border-l-4 border-status-warning",
  },
  // biome-ignore lint/style/useNamingConvention: LogLevel enum-like keys for color mapping
  INFO: {
    bg: "bg-blue-50 dark:bg-blue-950",
    text: "text-blue-700 dark:text-blue-300",
    badge: "bg-status-info text-text-inverse",
    border: "border-l-4 border-status-info",
  },
  // biome-ignore lint/style/useNamingConvention: LogLevel enum-like keys for color mapping
  DEBUG: {
    bg: "bg-surface-secondary dark:bg-dark-surface-secondary",
    text: "text-content-secondary dark:text-dark-content-secondary",
    badge: "bg-surface-tertiary text-text-inverse",
    border: "border-l-4 border-border",
  },
} as const;

/**
 * Format a timestamp for display.
 */
export function formatLogTimestamp(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleTimeString("en-US", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    fractionalSecondDigits: 3,
  });
}

/**
 * Format a full timestamp with date.
 */
export function formatLogDateTime(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
}
