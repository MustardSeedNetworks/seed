/**
 * LogViewerModal - Full-screen modal for viewing system logs.
 *
 * Opens as a large modal overlay (similar to Help dialog) for better readability.
 * Fixes GitHub issues #721, #386 - cramped UI, hard to read, no obvious close.
 *
 * Features:
 * - Full-screen modal with backdrop
 * - Large, readable fonts
 * - Clear close button in header
 * - Keyboard support (Escape to close)
 * - Export buttons prominently displayed
 * - All existing filter/search functionality
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  formatLogTimestamp,
  LOG_LEVEL_COLORS,
  type LogEntry,
  type LogFilters,
  type LogLayer,
  type LogLevel,
  useLogs,
} from "../../hooks/useLogs";
import { button, cn, icon as iconTokens, layout, modal, radius, spacing } from "../../styles/theme";

interface LogViewerModalProps {
  isOpen: boolean;
  onClose: () => void;
}

// Filter badge component
interface FilterBadgeProps {
  label: string;
  active: boolean;
  onClick: () => void;
  color?: string;
}

function FilterBadge({ label, active, onClick, color }: FilterBadgeProps) {
  return (
    <button
      type="button"
      className={cn(
        "px-3 py-1.5",
        radius.md,
        "text-sm font-medium cursor-pointer transition-all",
        active
          ? color || "bg-brand-primary text-text-inverse"
          : "bg-surface-base text-text-secondary hover:bg-surface-hover",
      )}
      onClick={onClick}
    >
      {label}
    </button>
  );
}

// Log entry row - larger and more readable than card version
interface LogEntryRowProps {
  entry: LogEntry;
  expanded: boolean;
  onToggle: () => void;
  onClose: () => void;
}

function LogEntryRow({ entry, expanded, onToggle, onClose }: LogEntryRowProps) {
  const colors = LOG_LEVEL_COLORS[entry.level];

  return (
    <button
      type="button"
      className={cn(
        colors.bg,
        colors.border,
        "px-4 py-3", // Larger padding than card version
        "mb-2",
        radius.lg,
        "cursor-pointer transition-colors hover:brightness-95",
        "w-full text-left", // Button needs explicit width and text alignment
      )}
      onClick={onToggle}
    >
      <div className={cn(layout.inline.default, "flex-wrap items-center")}>
        {/* Level badge - larger */}
        <span
          className={cn(
            colors.badge,
            "px-3 py-1",
            radius.default,
            "font-mono font-bold min-w-15 text-center text-sm",
          )}
        >
          {entry.level}
        </span>

        {/* Timestamp - larger */}
        <span className="text-sm text-text-muted font-mono">
          {formatLogTimestamp(entry.timestamp)}
        </span>

        {/* Layer badge */}
        <span className={cn("px-3 py-1", radius.default, "bg-surface-base text-sm")}>
          {entry.layer}
        </span>

        {/* Component badge */}
        {entry.component && (
          <span
            className={cn(
              "px-3 py-1",
              radius.default,
              "bg-purple-500/20 text-purple-600 dark:text-purple-400 text-sm",
            )}
          >
            {entry.component}
          </span>
        )}

        {/* Request ID badge */}
        {entry.request_id && (
          <span
            className={cn(
              "px-3 py-1",
              radius.default,
              "bg-status-info/20 text-status-info text-sm font-mono",
            )}
          >
            {entry.request_id.substring(0, 8)}
          </span>
        )}

        {/* Message - larger, don't truncate as aggressively */}
        <span className={cn(colors.text, "flex-1 text-base")} title={entry.message}>
          {entry.message}
        </span>

        {/* Duration badge */}
        {entry.duration_ms !== undefined && entry.duration_ms > 0 && (
          <span
            className={cn(
              "px-3 py-1",
              radius.default,
              "bg-status-success/20 text-status-success text-sm",
            )}
          >
            {entry.duration_ms}ms
          </span>
        )}

        {/* Expand indicator and close button for expanded entries */}
        <div className="flex items-center gap-2">
          <span className="text-sm text-text-muted">{expanded ? "▼" : "▶"}</span>
          {expanded && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onClose();
              }}
              className={cn(
                "p-1 rounded-full",
                "text-text-muted hover:text-text-primary hover:bg-surface-hover",
                "transition-colors",
              )}
              aria-label="Collapse entry"
            >
              <svg className="w-4 h-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                <path
                  fillRule="evenodd"
                  d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                  clipRule="evenodd"
                />
              </svg>
            </button>
          )}
        </div>
      </div>

      {/* Expanded content - more spacious */}
      {expanded && (
        <div className={cn("mt-4 space-y-3")}>
          {/* Full message */}
          <div className="text-base text-text-primary wrap-break-word whitespace-pre-wrap">
            {entry.message}
          </div>

          {/* Metadata - larger font, better formatting */}
          {entry.metadata && Object.keys(entry.metadata).length > 0 && (
            <pre
              className={cn(
                "p-4",
                radius.lg,
                "text-sm bg-surface-sunken overflow-x-auto font-mono whitespace-pre-wrap wrap-break-word",
              )}
            >
              {JSON.stringify(entry.metadata, null, 2)}
            </pre>
          )}

          {/* Stack trace */}
          {entry.stack && (
            <pre
              className={cn(
                "p-4",
                radius.lg,
                "text-sm text-status-error bg-status-error/10 overflow-x-auto font-mono whitespace-pre-wrap",
              )}
            >
              {entry.stack}
            </pre>
          )}

          {/* Full details - larger grid */}
          <div
            className={cn(
              "grid grid-cols-2 md:grid-cols-4",
              spacing.gap.comfortable,
              "text-sm text-text-secondary",
              "p-3 bg-surface-base rounded-lg",
            )}
          >
            <div>
              <strong className="text-text-primary">Timestamp:</strong>{" "}
              {new Date(entry.timestamp).toISOString()}
            </div>
            {entry.session_id && (
              <div>
                <strong className="text-text-primary">Session:</strong> {entry.session_id}
              </div>
            )}
            {entry.request_id && (
              <div>
                <strong className="text-text-primary">Request ID:</strong> {entry.request_id}
              </div>
            )}
            {entry.duration_ms !== undefined && (
              <div>
                <strong className="text-text-primary">Duration:</strong> {entry.duration_ms}ms
              </div>
            )}
          </div>
        </div>
      )}
    </button>
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
    <div className={cn("space-y-3", "p-4", "bg-surface-base", radius.lg)}>
      {/* Search bar - larger */}
      <div className={cn(layout.inline.default)}>
        <input
          type="text"
          placeholder={t("logs.searchPlaceholder", "Search logs...")}
          value={filters.search}
          onChange={(e) => onFilterChange({ search: e.target.value })}
          className={cn(
            "flex-1",
            "px-4 py-2.5 text-base",
            radius.lg,
            "border border-surface-border bg-surface-raised text-text-primary",
            "focus:outline-none focus:ring-2 focus:ring-brand-primary",
          )}
        />
        {hasActiveFilters && (
          <button
            type="button"
            onClick={onReset}
            className={cn("px-4 py-2", "text-base text-text-secondary hover:text-text-primary")}
          >
            {t("logs.clearFilters", "Clear All")}
          </button>
        )}
      </div>

      {/* Level filters */}
      <div className={cn(layout.inline.default, "flex-wrap")}>
        <span className="text-sm text-text-secondary font-medium min-w-20">
          {t("logs.level", "Level")}:
        </span>
        {levels.map((level) => {
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
        <span className="text-sm text-text-secondary font-medium min-w-20">
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

      {/* Component filters */}
      {availableComponents.length > 0 && (
        <div className={cn(layout.inline.default, "flex-wrap")}>
          <span className="text-sm text-text-secondary font-medium min-w-20">
            {t("logs.component", "Component")}:
          </span>
          {availableComponents.slice(0, 12).map((component) => (
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

/**
 * LogViewerModal - Full-screen modal for log viewing.
 */
export function LogViewerModal({ isOpen, onClose }: LogViewerModalProps) {
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
    clearLogs,
  } = useLogs({ maxLogs: 1000 });

  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const [autoScroll, setAutoScroll] = useState(true);
  const logContainerRef = useRef<HTMLDivElement>(null);

  // Get unique components from logs
  const availableComponents = Array.from(
    new Set(allLogs.map((log) => log.component).filter(Boolean) as string[]),
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

  // Close expanded entry
  const closeExpanded = useCallback((timestamp: string) => {
    setExpandedIds((prev) => {
      const newSet = new Set(prev);
      newSet.delete(timestamp);
      return newSet;
    });
  }, []);

  // Auto-scroll to bottom when new logs arrive
  const logsLength = logs.length;
  useEffect(() => {
    if (autoScroll && logContainerRef.current && isStreaming && logsLength > 0) {
      logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight;
    }
  }, [logsLength, autoScroll, isStreaming]);

  // Handle scroll to detect if user scrolled up
  const handleScroll = useCallback(() => {
    if (!logContainerRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = logContainerRef.current;
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 50;
    setAutoScroll(isAtBottom);
  }, []);

  // Keyboard handler for Escape
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onClose]);

  // Export functions
  const exportJson = useCallback(() => {
    const blob = new Blob([JSON.stringify(logs, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `logs-${new Date().toISOString().split("T")[0]}.json`;
    link.click();
    URL.revokeObjectURL(url);
  }, [logs]);

  const exportCsv = useCallback(() => {
    const escapeCsv = (val: unknown) => {
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
        escapeCsv(entry.timestamp),
        escapeCsv(entry.level),
        escapeCsv(entry.layer),
        escapeCsv(entry.component ?? ""),
        escapeCsv(entry.message),
        escapeCsv(entry.request_id ?? ""),
        escapeCsv(entry.session_id ?? ""),
        escapeCsv(entry.duration_ms ?? ""),
        escapeCsv(metadata),
      ].join(",");
    });

    const header =
      "timestamp,level,layer,component,message,request_id,session_id,duration_ms,metadata";
    const csv = [header, ...rows].join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `logs-${new Date().toISOString().split("T")[0]}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  }, [logs]);

  if (!isOpen) return null;

  return (
    <div className={modal.overlay}>
      {/* Backdrop */}
      <div className={modal.backdrop} onClick={onClose} aria-hidden="true" />

      {/* Modal - use xl size for logs */}
      <div
        className={cn(
          "relative",
          modal.content,
          modal.size.xl,
          radius.lg,
          "flex",
          "flex-col",
          "h-[90vh]", // 90% viewport height
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby="log-viewer-modal-title"
      >
        {/* Header */}
        <div
          className={cn(
            layout.flex.between,
            "px-6 py-4",
            "border-b",
            "border-surface-border",
            "bg-surface-raised",
            "shrink-0",
          )}
        >
          <div>
            <h2 id="log-viewer-modal-title" className="heading-2">
              {t("logs.title", "System Logs")}
            </h2>
            <p className="body-small text-text-secondary mt-1">
              {t("logs.subtitle", "Real-time application logs with filtering")}
              {stats && (
                <span className="ml-4">
                  <strong>{stats.total_count}</strong> {t("logs.totalLogs", "logs")}
                  {stats.errors_last_hour > 0 && (
                    <span className="text-status-error ml-2">
                      ({stats.errors_last_hour} {t("logs.errorsLastHour", "errors last hour")})
                    </span>
                  )}
                </span>
              )}
            </p>
          </div>

          <div className={cn("flex items-center", spacing.gap.comfortable)}>
            {/* Streaming toggle */}
            <button
              type="button"
              onClick={() => setIsStreaming(!isStreaming)}
              className={cn(
                button.size.md,
                radius.lg,
                "font-medium transition-colors",
                isStreaming
                  ? "bg-status-success text-text-inverse hover:brightness-90"
                  : "bg-surface-base text-text-primary hover:bg-surface-hover border border-surface-border",
              )}
            >
              {isStreaming ? t("logs.streaming", "● Live") : t("logs.paused", "○ Paused")}
            </button>

            {/* Clear logs */}
            <button
              type="button"
              className={cn(
                button.size.md,
                radius.lg,
                "border border-surface-border hover:bg-surface-hover",
              )}
              onClick={clearLogs}
            >
              {t("logs.clear", "Clear")}
            </button>

            {/* Export JSON */}
            <button
              type="button"
              className={cn(
                button.size.md,
                radius.lg,
                "border border-surface-border hover:bg-surface-hover",
                "flex items-center gap-2",
              )}
              onClick={exportJson}
            >
              <svg
                className="w-4 h-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                />
              </svg>
              JSON
            </button>

            {/* Export CSV */}
            <button
              type="button"
              className={cn(
                button.size.md,
                radius.lg,
                "border border-surface-border hover:bg-surface-hover",
                "flex items-center gap-2",
              )}
              onClick={exportCsv}
            >
              <svg
                className="w-4 h-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                />
              </svg>
              CSV
            </button>

            {/* Close button */}
            <button
              type="button"
              onClick={onClose}
              className={cn(
                "p-2",
                "text-text-muted",
                "hover:text-text-primary",
                "transition-colors",
                radius.lg,
                "hover:bg-surface-base",
              )}
              aria-label={t("logs.close", "Close log viewer")}
            >
              <svg
                className={iconTokens.size.lg}
                viewBox="0 0 20 20"
                fill="currentColor"
                aria-hidden="true"
              >
                <path
                  fillRule="evenodd"
                  d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                  clipRule="evenodd"
                />
              </svg>
            </button>
          </div>
        </div>

        {/* Filters */}
        <div className="px-6 py-4 bg-surface-raised border-b border-surface-border shrink-0">
          <LogFiltersBar
            filters={filters}
            onFilterChange={setFilters}
            onReset={resetFilters}
            availableComponents={availableComponents}
          />
        </div>

        {/* Log entries - scrollable area */}
        <div
          ref={logContainerRef}
          className={cn("flex-1 overflow-y-auto p-6 bg-surface-base/40")}
          onScroll={handleScroll}
        >
          {/* Loading state */}
          {isLoading && (
            <div className={cn("text-center text-text-secondary py-8")}>
              {t("logs.loading", "Loading logs...")}
            </div>
          )}

          {/* Error state */}
          {error && <div className={cn("text-center text-status-error py-8")}>{error}</div>}

          {/* Empty state */}
          {logs.length === 0 && !isLoading && (
            <div className={cn("text-center text-text-secondary py-12")}>
              {filters.search || filters.levels.length > 0 || filters.layers.length > 0
                ? t("logs.noMatchingLogs", "No logs match the current filters")
                : t("logs.noLogs", "No logs yet")}
            </div>
          )}

          {/* Log entries */}
          {logs.map((entry) => (
            <LogEntryRow
              key={`${entry.timestamp}-${entry.message.substring(0, 20)}`}
              entry={entry}
              expanded={expandedIds.has(entry.timestamp)}
              onToggle={() => toggleExpand(entry.timestamp)}
              onClose={() => closeExpanded(entry.timestamp)}
            />
          ))}
        </div>

        {/* Footer with scroll-to-bottom */}
        {!autoScroll && logs.length > 0 && (
          <div
            className={cn(
              "px-6 py-3",
              "text-center border-t border-surface-border",
              "bg-surface-raised shrink-0",
            )}
          >
            <button
              type="button"
              onClick={() => {
                setAutoScroll(true);
                if (logContainerRef.current) {
                  logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight;
                }
              }}
              className="text-base text-brand-primary hover:underline"
            >
              ↓ {t("logs.scrollToBottom", "Scroll to latest")}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

export default LogViewerModal;
