import { Modal } from '../../shared/Modal';
import { SearchInput } from '../../shared/SearchInput';
import { Tooltip } from '../../shared/Tooltip';
/** Full-screen log filtering, streaming and export. */

import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  formatLogTimestamp,
  LOG_LEVEL_COLORS,
  type LogEntry,
  type LogFilters,
  type LogLayer,
  type LogLevel,
  useLogs,
} from '../../hooks/useLogs';
import { button, cn, layout, radius, spacing } from '../../styles/theme';

interface LogViewerModalProps {
  isOpen: boolean;
  onClose: () => void;
}

/**
 * Compute the hover tooltip for the live/paused toggle button.
 * Hoisted to keep `LogViewerModal` under the cognitive-complexity limit.
 */
function getStreamingToggleHint(
  streaming: boolean,
  t: (key: string, fallback?: string) => string,
): string {
  return streaming ? t('logs.streamingHint') : t('logs.pausedHint');
}

interface FilterBadgeProps {
  label: string;
  active: boolean;
  onClick: () => void;
  color?: string;
}

function FilterBadge({ label, active, onClick, color }: FilterBadgeProps): React.JSX.Element {
  return (
    <button
      type="button"
      className={cn(
        'px-3 py-compact-md',
        radius.md,
        'text-sm font-medium cursor-pointer transition-all',
        active
          ? color || 'bg-brand-primary text-on-brand'
          : 'bg-surface-base text-text-secondary hover:bg-surface-hover',
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

function LogEntryRow({ entry, expanded, onToggle, onClose }: LogEntryRowProps): React.JSX.Element {
  const { t: tCards } = useTranslation('cards');
  const colors = LOG_LEVEL_COLORS[entry.level];

  return (
    <button
      type="button"
      className={cn(
        colors.bg,
        colors.border,
        'px-4 py-row-lg', // Larger padding than card version
        'mb-2',
        radius.lg,
        'cursor-pointer transition-colors hover:brightness-95',
        'w-full text-left', // Button needs explicit width and text alignment
      )}
      onClick={onToggle}
    >
      <div className={cn(layout.inline.default, 'flex-wrap items-center')}>
        {/* Level badge - larger */}
        <span
          className={cn(
            colors.badge,
            'px-3 py-compact',
            radius.default,
            'font-mono font-bold min-w-15 text-center text-sm',
          )}
        >
          {entry.level}
        </span>

        {/* Timestamp - larger */}
        <span className="text-sm text-text-muted font-mono">
          {formatLogTimestamp(entry.timestamp)}
        </span>

        {/* Layer badge */}
        <span className={cn('px-3 py-compact', radius.default, 'bg-surface-base text-sm')}>
          {entry.layer}
        </span>

        {/* Component badge */}
        {entry.component ? (
          <span
            className={cn(
              'px-3 py-compact',
              radius.default,
              'bg-cat-6/15 text-cat-6-strong text-sm',
            )}
          >
            {entry.component}
          </span>
        ) : null}

        {/* Request ID badge */}
        {entry.requestId ? (
          <span
            className={cn(
              'px-3 py-compact',
              radius.default,
              'bg-status-info/15 text-status-info-strong text-sm font-mono',
            )}
          >
            {entry.requestId.substring(0, 8)}
          </span>
        ) : null}

        {/* Message - larger, don't truncate as aggressively */}
        <Tooltip text={entry.message}>
          <span className={cn(colors.text, 'flex-1 text-base')}>{entry.message}</span>
        </Tooltip>

        {/* Duration badge */}
        {entry.durationMs !== undefined && entry.durationMs > 0 ? (
          <span
            className={cn(
              'px-3 py-compact',
              radius.default,
              'bg-status-success/15 text-status-success-strong text-sm',
            )}
          >
            {entry.durationMs}ms
          </span>
        ) : null}

        {/* Expand indicator and close button for expanded entries */}
        <div className="flex items-center gap-compact">
          <span className="text-sm text-text-muted">{expanded ? '▼' : '▶'}</span>
          {expanded ? (
            <button
              type="button"
              onClick={(e: React.MouseEvent): void => {
                e.stopPropagation();
                onClose();
              }}
              className={cn(
                'p-1 rounded-full',
                'text-text-muted hover:text-text-primary hover:bg-surface-hover',
                'transition-colors',
              )}
              aria-label={tCards('logViewer.collapseEntry')}
            >
              <svg className="w-4 h-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                <path
                  fillRule="evenodd"
                  d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                  clipRule="evenodd"
                />
              </svg>
            </button>
          ) : null}
        </div>
      </div>
      {/* Expanded content - more spacious */}
      {expanded ? (
        <div className={cn('mt-content stack')}>
          {/* Full message */}
          <div className="text-base text-text-primary wrap-break-word whitespace-pre-wrap">
            {entry.message}
          </div>

          {/* Metadata - larger font, better formatting */}
          {entry.metadata && Object.keys(entry.metadata).length > 0 ? (
            <pre
              className={cn(
                'pad',
                radius.lg,
                'text-sm bg-surface-sunken overflow-x-auto font-mono whitespace-pre-wrap wrap-break-word',
              )}
            >
              {JSON.stringify(entry.metadata, null, 2)}
            </pre>
          ) : null}

          {/* Stack trace */}
          {entry.stack ? (
            <pre
              className={cn(
                'pad',
                radius.lg,
                'text-sm text-status-error-strong bg-status-error/10 overflow-x-auto font-mono whitespace-pre-wrap',
              )}
            >
              {entry.stack}
            </pre>
          ) : null}

          {/* Full details - larger grid */}
          <div
            className={cn(
              'grid grid-cols-2 md:grid-cols-4',
              spacing.gap.comfortable,
              'text-sm text-text-secondary',
              'pad-sm bg-surface-base rounded-lg',
            )}
          >
            <div>
              <strong className="text-text-primary">{tCards('logViewer.timestamp')}</strong>{' '}
              {new Date(entry.timestamp).toISOString()}
            </div>
            {entry.sessionId ? (
              <div>
                <strong className="text-text-primary">{tCards('logViewer.session')}</strong>{' '}
                {entry.sessionId}
              </div>
            ) : null}
            {entry.requestId ? (
              <div>
                <strong className="text-text-primary">{tCards('logViewer.requestId')}</strong>{' '}
                {entry.requestId}
              </div>
            ) : null}
            {entry.durationMs !== undefined ? (
              <div>
                <strong className="text-text-primary">{tCards('logViewer.duration')}</strong>{' '}
                {entry.durationMs}ms
              </div>
            ) : null}
          </div>
        </div>
      ) : null}
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
}: LogFiltersBarProps): React.JSX.Element {
  const { t } = useTranslation('common');
  const levels: LogLevel[] = ['ERROR', 'WARN', 'INFO', 'DEBUG'];
  const layers: LogLayer[] = ['backend', 'api', 'frontend'];

  const toggleLevel = (level: LogLevel): void => {
    const newLevels = filters.levels.includes(level)
      ? filters.levels.filter((l) => l !== level)
      : [...filters.levels, level];
    onFilterChange({ levels: newLevels });
  };

  const toggleLayer = (layer: LogLayer): void => {
    const newLayers = filters.layers.includes(layer)
      ? filters.layers.filter((l) => l !== layer)
      : [...filters.layers, layer];
    onFilterChange({ layers: newLayers });
  };

  const toggleComponent = (component: string): void => {
    const newComponents = filters.components.includes(component)
      ? filters.components.filter((c) => c !== component)
      : [...filters.components, component];
    onFilterChange({ components: newComponents });
  };

  const hasActiveFilters =
    filters.levels.length > 0 ||
    filters.layers.length > 0 ||
    filters.components.length > 0 ||
    filters.search !== '';

  return (
    <div className={cn('stack', 'pad', 'bg-surface-base', radius.lg)}>
      {/* Search bar - larger */}
      <div className={cn(layout.inline.default)}>
        <div className="flex-1">
          <SearchInput
            value={filters.search}
            onValueChange={(search): void => onFilterChange({ search })}
            label={t('logs.searchPlaceholder')}
            clearLabel={t('logs.clearSearch')}
            placeholder={t('logs.searchPlaceholder')}
          />
        </div>
        {hasActiveFilters ? (
          <button
            type="button"
            onClick={onReset}
            className={cn('px-4 py-row', 'text-base text-text-secondary hover:text-text-primary')}
          >
            {t('logs.clearFilters')}
          </button>
        ) : null}
      </div>
      {/* Level filters */}
      <div className={cn(layout.inline.default, 'flex-wrap')}>
        <span className="text-sm text-text-secondary font-medium min-w-20">{t('logs.level')}:</span>
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
      <div className={cn(layout.inline.default, 'flex-wrap')}>
        <span className="text-sm text-text-secondary font-medium min-w-20">{t('logs.layer')}:</span>
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
      {availableComponents.length > 0 ? (
        <div className={cn(layout.inline.default, 'flex-wrap')}>
          <span className="text-sm text-text-secondary font-medium min-w-20">
            {t('logs.component')}:
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
      ) : null}
    </div>
  );
}

/**
 * UTF-16 code-unit order, the same order as `Array#sort()` with no comparator.
 * `localeCompare` would reorder mixed-case component names.
 */
function compareUtf16(left: string, right: string): number {
  if (left < right) {
    return -1;
  }
  if (left > right) {
    return 1;
  }
  return 0;
}

/**
 * LogViewerModal - Full-screen modal for log viewing.
 */
export function LogViewerModal({ isOpen, onClose }: LogViewerModalProps): React.JSX.Element | null {
  const { t } = useTranslation('common');
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

  // Get unique components from logs
  const availableComponents = Array.from(
    new Set(allLogs.map((log) => log.component).filter(Boolean) as string[]),
  ).sort(compareUtf16);

  // Toggle log expansion
  const toggleExpand = (timestamp: string): void => {
    setExpandedIds((prev: Set<string>): Set<string> => {
      const newSet = new Set(prev);
      if (newSet.has(timestamp)) {
        newSet.delete(timestamp);
      } else {
        newSet.add(timestamp);
      }
      return newSet;
    });
  };

  // Close expanded entry
  const closeExpanded = (timestamp: string): void => {
    setExpandedIds((prev: Set<string>): Set<string> => {
      const newSet = new Set(prev);
      newSet.delete(timestamp);
      return newSet;
    });
  };

  // Export functions
  const exportJson = (): void => {
    const blob = new Blob([JSON.stringify(logs, null, 2)], {
      type: 'application/json',
    });
    const url: string = URL.createObjectURL(blob);
    const link: HTMLAnchorElement = document.createElement('a');
    link.href = url;
    link.download = `logs-${new Date().toISOString().split('T')[0]}.json`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const exportCsv = (): void => {
    const escapeCsv = (val: unknown): string => {
      if (val === null || val === undefined) {
        return '';
      }
      const str: string = String(val);
      if (/[",\n]/.test(str)) {
        return `"${str.replace(/"/g, '""')}"`;
      }
      return str;
    };

    const rows: string[] = logs.map((entry: LogEntry): string => {
      const metadata: string = entry.metadata ? JSON.stringify(entry.metadata) : '';
      return [
        escapeCsv(entry.timestamp),
        escapeCsv(entry.level),
        escapeCsv(entry.layer),
        escapeCsv(entry.component ?? ''),
        escapeCsv(entry.message),
        escapeCsv(entry.requestId ?? ''),
        escapeCsv(entry.sessionId ?? ''),
        escapeCsv(entry.durationMs ?? ''),
        escapeCsv(metadata),
      ].join(',');
    });

    const header: string =
      'timestamp,level,layer,component,message,requestId,sessionId,durationMs,metadata';
    const csv: string = [header, ...rows].join('\n');
    const blob = new Blob([csv], { type: 'text/csv' });
    const url: string = URL.createObjectURL(blob);
    const link: HTMLAnchorElement = document.createElement('a');
    link.href = url;
    link.download = `logs-${new Date().toISOString().split('T')[0]}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  };

  if (!isOpen) {
    return null;
  }

  return (
    <Modal
      open={isOpen}
      onClose={onClose}
      title={t('logs.title')}
      closeLabel={t('buttons.close')}
      size="full"
      data-testid="log-viewer-modal"
      header={
        <div className={cn('flex items-center', spacing.gap.comfortable)}>
          {/* Streaming toggle */}
          <Tooltip
            text={getStreamingToggleHint(
              isStreaming,
              t as unknown as (k: string, f?: string) => string,
            )}
          >
            <button
              type="button"
              onClick={() => setIsStreaming(!isStreaming)}
              className={cn(
                button.size.md,
                radius.lg,
                'font-medium transition-colors',
                isStreaming
                  ? 'bg-status-success text-text-inverse hover:brightness-90'
                  : 'bg-surface-base text-text-primary hover:bg-surface-hover border border-surface-border',
              )}
            >
              {isStreaming ? t('logs.streaming') : t('logs.paused')}
            </button>
          </Tooltip>

          {/* Clear logs */}
          <Tooltip text={t('logs.clearHint')}>
            <button
              type="button"
              className={cn(
                button.size.md,
                radius.lg,
                'border border-surface-border hover:bg-surface-hover',
              )}
              onClick={clearLogs}
            >
              {t('logs.clear')}
            </button>
          </Tooltip>

          {/* Export JSON */}
          <Tooltip text={t('logs.exportJsonHint')}>
            <button
              type="button"
              className={cn(
                button.size.md,
                radius.lg,
                'border border-surface-border hover:bg-surface-hover',
                'flex items-center gap-compact',
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
          </Tooltip>

          {/* Export CSV */}
          <Tooltip text={t('logs.exportCsvHint')}>
            <button
              type="button"
              className={cn(
                button.size.md,
                radius.lg,
                'border border-surface-border hover:bg-surface-hover',
                'flex items-center gap-compact',
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
          </Tooltip>
        </div>
      }
    >
      <p className="body-small text-text-secondary mb-content">
        {t('logs.subtitle')}
        {stats ? (
          <span className="ml-content">
            <strong>{stats.totalCount}</strong> {t('logs.totalLogs')}
            {stats.errorsLastHour > 0 ? (
              <span className="text-status-error ml-inline">
                ({stats.errorsLastHour} {t('logs.errorsLastHour')})
              </span>
            ) : null}
          </span>
        ) : null}
      </p>

      {/* Filters */}
      <div className="mb-content">
        <LogFiltersBar
          filters={filters}
          onFilterChange={setFilters}
          onReset={resetFilters}
          availableComponents={availableComponents}
        />
      </div>

      {/* Log entries - scrollable area */}
      <div
        data-testid="log-viewer-entries"
        className={cn('max-h-[60vh] overflow-y-auto pad-lg bg-surface-base/40')}
      >
        {/* Loading state */}
        {isLoading ? (
          <div className={cn('text-center text-text-secondary py-8')}>{t('logs.loading')}</div>
        ) : null}

        {/* Error state */}
        {error ? <div className={cn('text-center text-status-error py-8')}>{error}</div> : null}

        {/* Empty state */}
        {logs.length === 0 && !isLoading ? (
          <div className={cn('text-center text-text-secondary py-centered')}>
            {filters.search || filters.levels.length > 0 || filters.layers.length > 0
              ? t('logs.noMatchingLogs')
              : t('logs.noLogs')}
          </div>
        ) : null}

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
    </Modal>
  );
}

export default LogViewerModal;
