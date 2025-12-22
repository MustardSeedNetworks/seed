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
import { cn, spacing, button, radius, layout, input } from "../../styles/theme";

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
  return (
    <button
      type="button"
      className={cn(
        spacing.chip.sm,
        radius.md,
        "text-xs font-medium cursor-pointer transition-all",
        active
          ? color || "bg-brand-primary text-text-inverse"
          : "bg-surface-base text-text-secondary hover:bg-surface-hover"
      )}
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
      className={cn(
        colors.bg,
        colors.border,
        spacing.pad.xs,
        spacing.margin.bottom.tight,
        radius.default,
        "cursor-pointer transition-colors hover:brightness-95"
      )}
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
      <div className={cn(layout.inline.default, "flex-wrap")}>
        {/* Level badge */}
        <span
          className={cn(
            colors.badge,
            spacing.chip.sm,
            radius.default,
            "font-mono font-bold min-w-[50px] text-center text-xs"
          )}
        >
          {entry.level}
        </span>

        {/* Timestamp */}
        <span className="text-xs text-text-muted font-mono">
          {formatLogTimestamp(entry.timestamp)}
        </span>

        {/* Layer badge */}
        <span
          className={cn(
            spacing.chip.sm,
            radius.default,
            "bg-surface-base text-xs"
          )}
        >
          {entry.layer}
        </span>

        {/* Component badge */}
        {entry.component && (
          <span
            className={cn(
              spacing.chip.sm,
              radius.default,
              "bg-purple-500/20 text-purple-600 dark:text-purple-400 text-xs"
            )}
          >
            {entry.component}
          </span>
        )}

        {/* Request ID badge */}
        {entry.request_id && (
          <span
            className={cn(
              spacing.chip.sm,
              radius.default,
              "bg-status-info/20 text-status-info text-xs font-mono"
            )}
          >
            {entry.request_id.substring(0, 8)}
          </span>
        )}

        {/* Message */}
        <span
          className={cn(colors.text, "flex-1 truncate text-sm")}
          title={entry.message}
        >
          {entry.message}
        </span>

        {/* Duration badge */}
        {entry.duration_ms !== undefined && entry.duration_ms > 0 && (
          <span
            className={cn(
              spacing.chip.sm,
              radius.default,
              "bg-status-success/20 text-status-success text-xs"
            )}
          >
            {entry.duration_ms}ms
          </span>
        )}

        {/* Expand indicator */}
        <span className="text-xs text-text-muted">{expanded ? "▼" : "▶"}</span>
      </div>

      {/* Expanded content */}
      {expanded && (
        <div className={cn(spacing.margin.top.inline, "space-y-2")}>
          {/* Full message */}
          <div className="text-sm text-text-primary break-words whitespace-pre-wrap">
            {entry.message}
          </div>

          {/* Metadata */}
          {entry.metadata && Object.keys(entry.metadata).length > 0 && (
            <pre
              className={cn(
                spacing.pad.xs,
                radius.default,
                "text-xs bg-surface-sunken overflow-x-auto font-mono whitespace-pre-wrap break-words"
              )}
            >
              {JSON.stringify(entry.metadata, null, 2)}
            </pre>
          )}

          {/* Stack trace */}
          {entry.stack && (
            <pre
              className={cn(
                spacing.pad.xs,
                radius.default,
                "text-xs text-status-error bg-status-error/10 overflow-x-auto font-mono whitespace-pre-wrap"
              )}
            >
              {entry.stack}
            </pre>
          )}

          {/* Full details */}
          <div
            className={cn(
              "grid grid-cols-2",
              spacing.gap.compact,
              "text-xs text-text-secondary"
            )}
          >
            <div>
              <strong>Timestamp:</strong>{" "}
              {new Date(entry.timestamp).toISOString()}
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
    <div
      className={cn("space-y-2", spacing.pad.sm, "bg-surface-base", radius.lg)}
    >
      {/* Search bar */}
      <div className={cn(layout.inline.default)}>
        <input
          type="text"
          placeholder={t("logs.searchPlaceholder", "Search logs...")}
          value={filters.search}
          onChange={(e) => onFilterChange({ search: e.target.value })}
          className={cn(
            "flex-1",
            input.size.sm,
            radius.md,
            "border border-surface-border bg-surface-raised text-text-primary",
            "focus:outline-none focus:ring-2 focus:ring-brand-primary"
          )}
        />
        {hasActiveFilters && (
          <button
            type="button"
            onClick={onReset}
            className={cn(
              spacing.chip.sm,
              "text-sm text-text-secondary hover:text-text-primary"
            )}
          >
            {t("logs.clearFilters", "Clear")}
          </button>
        )}
      </div>

      {/* Level filters */}
      <div className={cn(layout.inline.default, "flex-wrap")}>
        <span className="text-xs text-text-secondary font-medium">
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
      <div className={cn(layout.inline.default, "flex-wrap")}>
        <span className="text-xs text-text-secondary font-medium">
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
        <div className={cn(layout.inline.default, "flex-wrap")}>
          <span className="text-xs text-text-secondary font-medium">
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
function LogStatsBar({
  stats,
  isStreaming,
  onToggleStreaming,
}: LogStatsBarProps) {
  const { t } = useTranslation("common");

  return (
    <div
      className={cn(
        layout.flex.between,
        spacing.pad.xs,
        "bg-surface-hover",
        radius.lg,
        "text-sm"
      )}
    >
      <div className={cn(layout.inline.comfortable)}>
        {stats && (
          <>
            <span>
              <strong>{stats.total_count}</strong> {t("logs.totalLogs", "logs")}
            </span>
            <span className="text-status-error">
              <strong>
                {stats.by_level && "ERROR" in stats.by_level
                  ? stats.by_level.ERROR
                  : 0}
              </strong>{" "}
              {t("logs.errors", "errors")}
            </span>
            <span className="text-status-warning">
              <strong>
                {stats.by_level && "WARN" in stats.by_level
                  ? stats.by_level.WARN
                  : 0}
              </strong>{" "}
              {t("logs.warnings", "warnings")}
            </span>
            {stats.errors_last_hour > 0 && (
              <span className="text-status-error text-xs">
                ({stats.errors_last_hour} {t("logs.lastHour", "last hour")})
              </span>
            )}
          </>
        )}
      </div>

      <button
        type="button"
        onClick={onToggleStreaming}
        className={cn(
          spacing.chip.sm,
          radius.md,
          "text-xs font-medium transition-colors",
          isStreaming
            ? "bg-status-success text-text-inverse hover:brightness-90"
            : "bg-surface-base text-text-primary hover:bg-surface-hover"
        )}
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
export function LogViewerCard({
  maxHeight = "500px",
  className = "",
}: LogViewerCardProps) {
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
    const blob = new Blob([JSON.stringify(logs, null, 2)], {
      type: "application/json",
    });
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
      className={cn(
        "bg-surface-raised",
        radius.lg,
        "border border-surface-border overflow-hidden",
        className
      )}
    >
      {/* Header */}
      <div
        className={cn(
          spacing.pad.md,
          "border-b border-surface-border",
          layout.flex.between,
          "items-start flex-wrap",
          spacing.gap.default
        )}
      >
        <div>
          <h2 className="heading-3 text-text-primary">
            {t("logs.title", "System Logs")}
          </h2>
          <p className="body-small text-text-secondary">
            {t("logs.subtitle", "Real-time application logs with filtering")}
          </p>
        </div>
        <div className={cn(layout.inline.default, "flex-wrap")}>
          <button
            type="button"
            className={cn(
              button.size.sm,
              radius.default,
              "border border-surface-border text-sm hover:bg-surface-hover"
            )}
            onClick={() => setCollapsed((prev) => !prev)}
          >
            {collapsed
              ? t("logs.expand", "Expand")
              : t("logs.collapse", "Collapse")}
          </button>
          <button
            type="button"
            className={cn(
              button.size.sm,
              radius.default,
              "border border-surface-border text-sm hover:bg-surface-hover"
            )}
            onClick={() => setIsStreaming(!isStreaming)}
          >
            {isStreaming
              ? t("logs.pause", "Pause")
              : t("logs.resume", "Resume")}
          </button>
          <button
            type="button"
            className={cn(
              button.size.sm,
              radius.default,
              "border border-surface-border text-sm hover:bg-surface-hover"
            )}
            onClick={clearLogs}
          >
            {t("logs.clear", "Clear")}
          </button>
          <button
            type="button"
            className={cn(
              button.size.sm,
              radius.default,
              "border border-surface-border text-sm hover:bg-surface-hover"
            )}
            onClick={exportJSON}
          >
            {t("logs.exportJson", "Export JSON")}
          </button>
          <button
            type="button"
            className={cn(
              button.size.sm,
              radius.default,
              "border border-surface-border text-sm hover:bg-surface-hover"
            )}
            onClick={exportCSV}
          >
            {t("logs.exportCsv", "Export CSV")}
          </button>
        </div>
      </div>

      {collapsed ? (
        <div className={cn(spacing.pad.md, "text-text-secondary")}>
          {t("logs.collapsed", "Log viewer collapsed")}
        </div>
      ) : (
        <>
          {/* Stats bar */}
          <div className={cn("px-4 pt-3", spacing.stack.sm)}>
            <LogStatsBar
              stats={stats}
              isStreaming={isStreaming}
              onToggleStreaming={() => setIsStreaming(!isStreaming)}
            />
          </div>

          {/* Filters */}
          <div className={cn("px-4", spacing.stack.sm)}>
            <LogFiltersBar
              filters={filters}
              onFilterChange={setFilters}
              onReset={resetFilters}
              availableComponents={availableComponents}
            />
          </div>

          {/* Loading state */}
          {isLoading && (
            <div
              className={cn(spacing.pad.md, "text-center text-text-secondary")}
            >
              {t("logs.loading", "Loading logs...")}
            </div>
          )}

          {/* Error state */}
          {error && (
            <div
              className={cn(spacing.pad.md, "text-center text-status-error")}
            >
              {error}
            </div>
          )}

          {/* Log entries */}
          <div
            ref={logContainerRef}
            className={cn(
              spacing.pad.md,
              "overflow-y-auto font-mono text-sm bg-surface-base/40",
              radius.lg,
              "mx-4"
            )}
            style={{ maxHeight, minHeight: "320px" }}
            onScroll={handleScroll}
          >
            {logs.length === 0 && !isLoading && (
              <div
                className={cn(
                  "text-center text-text-secondary",
                  spacing.pad.lg
                )}
              >
                {filters.search ||
                filters.levels.length > 0 ||
                filters.layers.length > 0
                  ? t(
                      "logs.noMatchingLogs",
                      "No logs match the current filters"
                    )
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
            <div
              className={cn(
                spacing.pad.xs,
                "text-center border-t border-surface-border"
              )}
            >
              <button
                type="button"
                onClick={() => {
                  setAutoScroll(true);
                  if (logContainerRef.current) {
                    logContainerRef.current.scrollTop =
                      logContainerRef.current.scrollHeight;
                  }
                }}
                className="text-sm text-brand-primary hover:underline"
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
