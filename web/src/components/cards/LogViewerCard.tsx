/**
 * LogViewerCard - Real-time log viewer with color-coded severity levels.
 *
 * Features:
 * - Real-time log streaming via WebSocket
 * - Color-coded severity levels (ERROR=red, WARN=yellow, INFO=blue, DEBUG=gray)
 * - Filterable by level, layer, component
 * - Searchable log text
 * - Expandable log entries for metadata/stack traces
 * - Auto-scroll to latest logs
 *
 * Usage:
 * ```tsx
 * <LogViewerCard
 *   logs={logs}
 *   filters={filters}
 *   onFilterChange={setFilters}
 *   isStreaming={true}
 * />
 * ```
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  useLogs,
  LogEntry,
  LogFilters,
  LogLevel,
  LogLayer,
  LOG_LEVEL_COLORS,
  formatLogTimestamp,
} from "../../hooks/useLogs";
import { spacing, button as buttonTokens } from "../../styles/theme";

// Filter badge component
interface FilterBadgeProps {
  label: string;
  active: boolean;
  onClick: () => void;
  color?: string;
}

/**
 * FilterBadge renders a clickable badge for filtering logs.
 */
function FilterBadge({ label, active, onClick, color }: FilterBadgeProps) {
  const baseClasses = "px-2 py-1 text-xs font-medium rounded-md cursor-pointer transition-all";
  const activeClasses = color || "bg-primary-600 text-text-inverse";
  const inactiveClasses =
    "bg-surface-secondary dark:bg-dark-surface-secondary text-content-secondary dark:text-dark-content-secondary hover:bg-surface-tertiary dark:hover:bg-dark-surface-tertiary";

  return (
    <button
      type="button"
      className={`${baseClasses} ${active ? activeClasses : inactiveClasses}`}
      onClick={onClick}
    >
      {label}
    </button>
  );
}

// Single log entry row component
interface LogEntryRowProps {
  entry: LogEntry;
  expanded: boolean;
  onToggle: () => void;
}

function LogEntryRow({ entry, expanded, onToggle }: LogEntryRowProps) {
  const colors = LOG_LEVEL_COLORS[entry.level];

  return (
    <div
      className={`${colors.bg} ${colors.border} p-2 mb-1 rounded cursor-pointer transition-colors hover:brightness-95`}
      onClick={onToggle}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onToggle();
        }
      }}
    >
      <div className="flex items-center gap-2 flex-wrap">
        {/* Level badge */}
        <span
          className={`${colors.badge} px-2 py-0.5 text-xs rounded font-mono font-bold min-w-[50px] text-center`}
        >
          {entry.level}
        </span>

        {/* Timestamp */}
        <span className="text-xs text-content-tertiary dark:text-dark-content-tertiary font-mono">
          {formatLogTimestamp(entry.timestamp)}
        </span>

        {/* Layer badge */}
        <span className="px-1.5 py-0.5 bg-surface-secondary dark:bg-dark-surface-secondary text-xs rounded">
          {entry.layer}
        </span>

        {/* Component badge */}
        {entry.component && (
          <span className="px-1.5 py-0.5 bg-purple-100 dark:bg-purple-900 text-purple-700 dark:text-purple-300 text-xs rounded">
            {entry.component}
          </span>
        )}

        {/* Request ID badge */}
        {entry.request_id && (
          <span className="px-1.5 py-0.5 bg-cyan-100 dark:bg-cyan-900 text-cyan-700 dark:text-cyan-300 text-xs rounded font-mono">
            {entry.request_id.substring(0, 8)}
          </span>
        )}

        {/* Message */}
        <span className={`${colors.text} flex-1 truncate text-sm`} title={entry.message}>
          {entry.message}
        </span>

        {/* Duration badge */}
        {entry.duration_ms !== undefined && entry.duration_ms > 0 && (
          <span className="px-1.5 py-0.5 bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300 text-xs rounded">
            {entry.duration_ms}ms
          </span>
        )}

        {/* Expand indicator */}
        <span className="text-xs text-content-tertiary">{expanded ? "▼" : "▶"}</span>
      </div>

      {/* Expanded content */}
      {expanded && (
        <div className="mt-2 space-y-2">
          {/* Full message */}
          <div className="text-sm text-content-primary dark:text-dark-content-primary break-words whitespace-pre-wrap">
            {entry.message}
          </div>

          {/* Metadata */}
          {entry.metadata && Object.keys(entry.metadata).length > 0 && (
            <pre className="text-xs bg-black/10 dark:bg-white/10 p-2 rounded overflow-x-auto font-mono whitespace-pre-wrap break-words">
              {JSON.stringify(entry.metadata, null, 2)}
            </pre>
          )}

          {/* Stack trace */}
          {entry.stack && (
            <pre className="text-xs text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-950 p-2 rounded overflow-x-auto font-mono whitespace-pre-wrap">
              {entry.stack}
            </pre>
          )}

          {/* Full details */}
          <div className="grid grid-cols-2 gap-2 text-xs text-content-secondary dark:text-dark-content-secondary">
            <div>
              <strong>Timestamp:</strong> {new Date(entry.timestamp).toISOString()}
            </div>
            {entry.session_id && (
              <div>
                <strong>Session:</strong> {entry.session_id}
              </div>
            )}
            {entry.request_id && (
              <div>
                <strong>Request ID:</strong> {entry.request_id}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// Filter bar component
interface LogFiltersBarProps {
  filters: LogFilters;
  onFilterChange: (filters: Partial<LogFilters>) => void;
  onReset: () => void;
  availableComponents: string[];
}

function LogFiltersBar({
  filters,
  onFilterChange,
  onReset,
  availableComponents,
}: LogFiltersBarProps) {
  const { t } = useTranslation("common");
  const levels: LogLevel[] = ["ERROR", "WARN", "INFO", "DEBUG"];
  const layers: LogLayer[] = ["backend", "api", "frontend"];

  const toggleLevel = (level: LogLevel) => {
    const newLevels = filters.levels.includes(level)
      ? filters.levels.filter((l) => l !== level)
      : [...filters.levels, level];
    onFilterChange({ levels: newLevels });
  };

  const toggleLayer = (layer: LogLayer) => {
    const newLayers = filters.layers.includes(layer)
      ? filters.layers.filter((l) => l !== layer)
      : [...filters.layers, layer];
    onFilterChange({ layers: newLayers });
  };

  const toggleComponent = (component: string) => {
    const newComponents = filters.components.includes(component)
      ? filters.components.filter((c) => c !== component)
      : [...filters.components, component];
    onFilterChange({ components: newComponents });
  };

  const hasActiveFilters =
    filters.levels.length > 0 ||
    filters.layers.length > 0 ||
    filters.components.length > 0 ||
    filters.search !== "";

  return (
    <div className="space-y-2 p-3 bg-surface-secondary dark:bg-dark-surface-secondary rounded-lg">
      {/* Search bar */}
      <div className="flex items-center gap-2">
        <input
          type="text"
          placeholder={t("logs.searchPlaceholder", "Search logs...")}
          value={filters.search}
          onChange={(e) => onFilterChange({ search: e.target.value })}
          className="flex-1 px-3 py-1.5 text-sm rounded-md border border-border dark:border-dark-border bg-surface-primary dark:bg-dark-surface-primary focus:outline-none focus:ring-2 focus:ring-primary-500"
        />
        {hasActiveFilters && (
          <button
            type="button"
            onClick={onReset}
            className="px-3 py-1.5 text-sm text-content-secondary hover:text-content-primary dark:text-dark-content-secondary dark:hover:text-dark-content-primary"
          >
            {t("logs.clearFilters", "Clear")}
          </button>
        )}
      </div>

      {/* Level filters */}
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-xs text-content-secondary dark:text-dark-content-secondary font-medium">
          {t("logs.level", "Level")}:
        </span>
        {levels.map((level) => {
          // eslint-disable-next-line security/detect-object-injection -- level is typed LogLevel
          const badgeColor = LOG_LEVEL_COLORS[level].badge;
          return (
            <FilterBadge
              key={level}
              label={level}
              active={filters.levels.includes(level)}
              onClick={() => toggleLevel(level)}
              color={filters.levels.includes(level) ? badgeColor : undefined}
            />
          );
        })}
      </div>

      {/* Layer filters */}
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-xs text-content-secondary dark:text-dark-content-secondary font-medium">
          {t("logs.layer", "Layer")}:
        </span>
        {layers.map((layer) => (
          <FilterBadge
            key={layer}
            label={layer}
            active={filters.layers.includes(layer)}
            onClick={() => toggleLayer(layer)}
          />
        ))}
      </div>

      {/* Component filters (if any available) */}
      {availableComponents.length > 0 && (
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs text-content-secondary dark:text-dark-content-secondary font-medium">
            {t("logs.component", "Component")}:
          </span>
          {availableComponents.slice(0, 10).map((component) => (
            <FilterBadge
              key={component}
              label={component}
              active={filters.components.includes(component)}
              onClick={() => toggleComponent(component)}
            />
          ))}
        </div>
      )}
    </div>
  );
}

/** Props for the log statistics bar. */
interface LogStatsBarProps {
  /** Log statistics data or null if not loaded. */
  stats: {
    total_count: number;
    by_level: Record<string, number>;
    errors_last_hour: number;
    warnings_last_hour: number;
  } | null;
  /** Whether real-time streaming is enabled. */
  isStreaming: boolean;
  /** Callback to toggle streaming state. */
  onToggleStreaming: () => void;
}

/**
 * LogStatsBar displays log statistics and streaming toggle.
 */
function LogStatsBar({ stats, isStreaming, onToggleStreaming }: LogStatsBarProps) {
  const { t } = useTranslation("common");

  return (
    <div className="flex items-center justify-between p-2 bg-surface-tertiary dark:bg-dark-surface-tertiary rounded-lg text-sm">
      <div className="flex items-center gap-4">
        {stats && (
          <>
            <span>
              <strong>{stats.total_count}</strong> {t("logs.totalLogs", "logs")}
            </span>
            <span className="text-red-600 dark:text-red-400">
              <strong>
                {stats.by_level && "ERROR" in stats.by_level ? stats.by_level.ERROR : 0}
              </strong>{" "}
              {t("logs.errors", "errors")}
            </span>
            <span className="text-yellow-600 dark:text-yellow-400">
              <strong>
                {stats.by_level && "WARN" in stats.by_level ? stats.by_level.WARN : 0}
              </strong>{" "}
              {t("logs.warnings", "warnings")}
            </span>
            {stats.errors_last_hour > 0 && (
              <span className="text-red-600 dark:text-red-400 text-xs">
                ({stats.errors_last_hour} {t("logs.lastHour", "last hour")})
              </span>
            )}
          </>
        )}
      </div>

      <button
        type="button"
        onClick={onToggleStreaming}
        className={`px-3 py-1 rounded-md text-xs font-medium transition-colors ${
          isStreaming
            ? "bg-status-success text-text-inverse hover:brightness-90"
            : "bg-surface-tertiary text-text-inverse hover:brightness-90"
        }`}
      >
        {isStreaming ? t("logs.streaming", "Live") : t("logs.paused", "Paused")}
      </button>
    </div>
  );
}

/** Props for the LogViewerCard component. */
export interface LogViewerCardProps {
  /** Maximum height of the log viewer container. */
  maxHeight?: string;
  /** Additional CSS classes. */
  className?: string;
}

/**
 * LogViewerCard displays real-time logs with color-coded severity levels.
 * Supports filtering by level, layer, component, and search text.
 */
export function LogViewerCard({ maxHeight = "500px", className = "" }: LogViewerCardProps) {
  const { t } = useTranslation("common");
  const {
    logs,
    allLogs,
    filters,
    setFilters,
    resetFilters,
    stats,
    isStreaming,
    setIsStreaming,
    isLoading,
    error,
    addLog,
    clearLogs,
  } = useLogs({ maxLogs: 1000 });

  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const [autoScroll, setAutoScroll] = useState(true);
  const [collapsed, setCollapsed] = useState(false);
  const logContainerRef = useRef<HTMLDivElement>(null);

  // Get unique components from logs
  const availableComponents = Array.from(
    new Set(allLogs.map((log) => log.component).filter(Boolean) as string[])
  ).sort();

  // Toggle log expansion
  const toggleExpand = useCallback((timestamp: string) => {
    setExpandedIds((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(timestamp)) {
        newSet.delete(timestamp);
      } else {
        newSet.add(timestamp);
      }
      return newSet;
    });
  }, []);

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (autoScroll && logContainerRef.current && isStreaming) {
      logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight;
    }
  }, [logs, autoScroll, isStreaming]);

  // Handle scroll to detect if user scrolled up
  const handleScroll = useCallback(() => {
    if (!logContainerRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = logContainerRef.current;
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 50;
    setAutoScroll(isAtBottom);
  }, []);

  // Expose addLog for WebSocket integration
  useEffect(() => {
    // This could be connected to a global WebSocket handler
    // For now, it's exposed via the hook return value
  }, [addLog]);

  const exportJSON = useCallback(() => {
    const blob = new Blob([JSON.stringify(logs, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "logs.json";
    link.click();
    URL.revokeObjectURL(url);
  }, [logs]);

  const exportCSV = useCallback(() => {
    const escape = (val: unknown) => {
      if (val === null || val === undefined) return "";
      const str = String(val);
      if (/[",\n]/.test(str)) {
        return `"${str.replace(/"/g, '""')}"`;
      }
      return str;
    };

    const rows = logs.map((entry) => {
      const metadata = entry.metadata ? JSON.stringify(entry.metadata) : "";
      return [
        escape(entry.timestamp),
        escape(entry.level),
        escape(entry.layer),
        escape(entry.component ?? ""),
        escape(entry.message),
        escape(entry.request_id ?? ""),
        escape(entry.session_id ?? ""),
        escape(entry.duration_ms ?? ""),
        escape(metadata),
      ].join(",");
    });

    const header =
      "timestamp,level,layer,component,message,request_id,session_id,duration_ms,metadata";
    const csv = [header, ...rows].join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "logs.csv";
    link.click();
    URL.revokeObjectURL(url);
  }, [logs]);

  return (
    <div
      className={`bg-surface-primary dark:bg-dark-surface-primary rounded-lg border border-border dark:border-dark-border overflow-hidden ${className}`}
    >
      {/* Header */}
      <div className="p-4 border-b border-border dark:border-dark-border flex items-start justify-between gap-3 flex-wrap">
        <div>
          <h2 className="text-lg font-semibold text-content-primary dark:text-dark-content-primary">
            {t("logs.title", "System Logs")}
          </h2>
          <p className="text-sm text-content-secondary dark:text-dark-content-secondary">
            {t("logs.subtitle", "Real-time application logs with filtering")}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            className={`${buttonTokens.size.sm} rounded border border-border dark:border-dark-border text-sm`}
            onClick={() => setCollapsed((prev) => !prev)}
          >
            {collapsed ? t("logs.expand", "Expand") : t("logs.collapse", "Collapse")}
          </button>
          <button
            type="button"
            className={`${buttonTokens.size.sm} rounded border border-border dark:border-dark-border text-sm`}
            onClick={() => setIsStreaming(!isStreaming)}
          >
            {isStreaming ? t("logs.pause", "Pause") : t("logs.resume", "Resume")}
          </button>
          <button
            type="button"
            className={`${buttonTokens.size.sm} rounded border border-border dark:border-dark-border text-sm`}
            onClick={clearLogs}
          >
            {t("logs.clear", "Clear")}
          </button>
          <button
            type="button"
            className={`${buttonTokens.size.sm} rounded border border-border dark:border-dark-border text-sm`}
            onClick={exportJSON}
          >
            {t("logs.exportJson", "Export JSON")}
          </button>
          <button
            type="button"
            className={`${buttonTokens.size.sm} rounded border border-border dark:border-dark-border text-sm`}
            onClick={exportCSV}
          >
            {t("logs.exportCsv", "Export CSV")}
          </button>
        </div>
      </div>

      {collapsed ? (
        <div className="p-4 text-content-secondary dark:text-dark-content-secondary">
          {t("logs.collapsed", "Log viewer collapsed")}
        </div>
      ) : (
        <>
          {/* Stats bar */}
          <div className={`px-4 pt-3 ${spacing.stack.sm}`}>
            <LogStatsBar
              stats={stats}
              isStreaming={isStreaming}
              onToggleStreaming={() => setIsStreaming(!isStreaming)}
            />
          </div>

          {/* Filters */}
          <div className={`px-4 ${spacing.stack.sm}`}>
            <LogFiltersBar
              filters={filters}
              onFilterChange={setFilters}
              onReset={resetFilters}
              availableComponents={availableComponents}
            />
          </div>

          {/* Loading state */}
          {isLoading && (
            <div className="p-4 text-center text-content-secondary dark:text-dark-content-secondary">
              {t("logs.loading", "Loading logs...")}
            </div>
          )}

          {/* Error state */}
          {error && <div className="p-4 text-center text-red-600 dark:text-red-400">{error}</div>}

          {/* Log entries */}
          <div
            ref={logContainerRef}
            className="p-4 overflow-y-auto font-mono text-sm bg-surface-secondary/40 dark:bg-dark-surface-secondary/40 rounded-lg mx-4"
            style={{ maxHeight, minHeight: "320px" }}
            onScroll={handleScroll}
          >
            {logs.length === 0 && !isLoading && (
              <div className="text-center text-content-secondary dark:text-dark-content-secondary py-8">
                {filters.search || filters.levels.length > 0 || filters.layers.length > 0
                  ? t("logs.noMatchingLogs", "No logs match the current filters")
                  : t("logs.noLogs", "No logs yet")}
              </div>
            )}

            {logs.map((entry) => (
              <LogEntryRow
                key={`${entry.timestamp}-${entry.message.substring(0, 20)}`}
                entry={entry}
                expanded={expandedIds.has(entry.timestamp)}
                onToggle={() => toggleExpand(entry.timestamp)}
              />
            ))}
          </div>

          {/* Auto-scroll indicator */}
          {!autoScroll && logs.length > 0 && (
            <div className="p-2 text-center border-t border-border dark:border-dark-border">
              <button
                type="button"
                onClick={() => {
                  setAutoScroll(true);
                  if (logContainerRef.current) {
                    logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight;
                  }
                }}
                className="text-sm text-primary-600 dark:text-primary-400 hover:underline"
              >
                {t("logs.scrollToBottom", "Scroll to latest")}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

export default LogViewerCard;
