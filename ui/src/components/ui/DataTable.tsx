// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Complex component
/**
 * DataTable Component
 *
 * Purpose: Flexible, feature-rich table component with sorting, searching, filtering,
 * and expandable rows. Used throughout the app for displaying lists of data.
 *
 * Key Features:
 * - Sorting: click column headers to sort ascending/descending (with visual indicators)
 * - Search: global text search across specified columns
 * - Filtering: dropdown filters for specific columns (e.g., status, protocol)
 * - Expandable rows: optional expanded content per row
 * - Row actions: custom action buttons per row
 * - Mobile responsive: hiddenOnMobile flag hides columns on small screens
 * - Custom rendering: render prop for cells to customize display
 * - Click handlers: onRowClick callback for row selection
 * - Empty state: customizable message when no data
 * - Customizable width: per-column width control
 *
 * Generic types: <T> for data items, Column<T> for column definitions
 *
 * Usage:
 * ```typescript
 * <DataTable<DeviceInfo>
 *   data={devices}
 *   columns={[
 *     { key: 'ip', header: 'IP Address', accessor: (d) => d.ip },
 *     { key: 'host', header: 'Hostname', accessor: (d) => d.hostname }
 *   ]}
 *   keyExtractor={(d) => d.ip}
 *   onRowClick={(d) => selectDevice(d)}
 *   searchKeys={['hostname', 'ip']}
 * />
 * ```
 *
 * Dependencies: React hooks, theme utilities, Lucide icons
 * State: Manages sort column/direction, search term, filter values, expanded rows
 */

import { ArrowUpDown, ChevronDown, ChevronUp, Filter, Search, X } from 'lucide-react';
import type React from 'react';
import { type ReactNode, useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { logger } from '../../lib/logger';
import { border, cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';

export type SortDirection = 'asc' | 'desc' | null;

export interface Column<T> {
  key: string;
  header: string;
  accessor: (item: T) => string | number | null | undefined;
  sortable?: boolean;
  hiddenOnMobile?: boolean;
  width?: string;
  render?: (item: T) => ReactNode;
}

export interface DataTableProps<T> {
  data: T[];
  columns: Column<T>[];
  keyExtractor: (item: T) => string;
  searchPlaceholder?: string;
  searchKeys?: string[];
  onRowClick?: (item: T) => void;
  expandedContent?: (item: T) => ReactNode;
  isExpanded?: (item: T) => boolean;
  emptyMessage?: string;
  maxHeight?: string;
  actions?: (item: T) => ReactNode;
  filterOptions?: {
    key: string;
    label: string;
    options: { value: string; label: string }[];
  }[];
  /** Error message to display when rendering fails (fixes #680) */
  error?: string;
  /** Loading state (fixes #680) */
  loading?: boolean;
}

function _sortIcon({
  direction,
  active,
}: {
  direction: SortDirection;
  active: boolean;
}): React.JSX.Element {
  if (!(active && direction)) {
    return <ArrowUpDown class={cn(iconTokens.size.xs, 'opacity-40')} />;
  }
  return direction === 'asc' ? (
    <ChevronUp class={iconTokens.size.xs} />
  ) : (
    <ChevronDown class={iconTokens.size.xs} />
  );
}

/**
 * Generic table component with search, sorting, and expandable row support.
 * Fixes #680: Added error handling and null checks for safe rendering
 */
export function DataTable<T>({
  data,
  columns,
  keyExtractor,
  searchPlaceholder = 'Search...',
  searchKeys,
  onRowClick,
  expandedContent: EXPANDED_CONTENT,
  isExpanded,
  emptyMessage = 'No data found',
  maxHeight = 'max-h-80',
  actions,
  filterOptions,
  error,
  loading = false,
}: DataTableProps<T>): React.JSX.Element {
  const { t } = useTranslation('common');
  // Note: _expandedContent is available for future row expansion feature (prefixed to suppress unused warning)
  const _expandedContent = EXPANDED_CONTENT;
  const [searchQuery, setSearchQuery] = useState('');
  const [sortKey, setSortKey] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<SortDirection>(null);
  const [activeFilters, setActiveFilters] = useState<Record<string, string>>({});
  const [showFilters, setShowFilters] = useState(false);
  const [renderError, setRenderError] = useState<Error | null>(null);

  const handleSort = useCallback(
    (key: string) => {
      if (sortKey === key) {
        // Cycle through: asc -> desc -> null
        if (sortDirection === 'asc') {
          setSortDirection('desc');
        } else if (sortDirection === 'desc') {
          setSortDirection(null);
          setSortKey(null);
        }
      } else {
        setSortKey(key);
        setSortDirection('asc');
      }
    },
    [sortKey, sortDirection],
  );

  const handleFilterChange = useCallback((key: string, value: string) => {
    setActiveFilters((prev) => {
      if (value === '') {
        const { [key]: _, ...rest } = prev;
        return rest;
      }
      return { ...prev, [key]: value };
    });
  }, []);

  const clearFilters = useCallback(() => {
    setActiveFilters({});
    setSearchQuery('');
  }, []);

  const filteredAndSortedData = useMemo(() => {
    // Fixes #680: Add null checks and error handling
    try {
      if (!Array.isArray(data)) {
        logger.error('ui', 'DataTable: data prop is not an array', undefined, { data });
        return [];
      }

      let result = [...data];

      // Apply search filter
      if (searchQuery && searchKeys && searchKeys.length > 0) {
        const query = searchQuery.toLowerCase();
        result = result.filter((item) => {
          if (!item) {
            return false; // Null check
          }
          return searchKeys.some((key) => {
            const column = columns.find((c) => c.key === key);
            if (column) {
              try {
                const value = column.accessor(item);
                return value?.toString().toLowerCase().includes(query);
              } catch (err) {
                logger.error('ui', 'DataTable: Error accessing column value', err);
                return false;
              }
            }
            return false;
          });
        });
      }

      // Apply column filters
      for (const [key, filterValue] of Object.entries(activeFilters)) {
        if (filterValue) {
          const column = columns.find((c) => c.key === key);
          if (column) {
            result = result.filter((item) => {
              if (!item) {
                return false; // Null check
              }
              try {
                const value = column.accessor(item);
                return value?.toString().toLowerCase().includes(filterValue.toLowerCase());
              } catch (err) {
                logger.error('ui', 'DataTable: Error filtering column value', err);
                return false;
              }
            });
          }
        }
      }

      // Apply sorting
      if (sortKey && sortDirection) {
        const column = columns.find((c) => c.key === sortKey);
        if (column) {
          result.sort((a, b) => {
            if (!(a && b)) {
              return 0; // Null checks
            }
            try {
              const aVal = column.accessor(a);
              const bVal = column.accessor(b);

              // Handle nulls
              if (aVal === null && bVal === null) {
                return 0;
              }
              if (aVal === null) {
                return sortDirection === 'asc' ? 1 : -1;
              }
              if (bVal === null) {
                return sortDirection === 'asc' ? -1 : 1;
              }

              // Compare values
              if (typeof aVal === 'number' && typeof bVal === 'number') {
                return sortDirection === 'asc' ? aVal - bVal : bVal - aVal;
              }

              const strA = String(aVal).toLowerCase();
              const strB = String(bVal).toLowerCase();
              const comparison = strA.localeCompare(strB, undefined, {
                numeric: true,
              });
              return sortDirection === 'asc' ? comparison : -comparison;
            } catch (err) {
              logger.error('ui', 'DataTable: Error sorting data', err);
              return 0;
            }
          });
        }
      }

      return result;
    } catch (err) {
      logger.error('ui', 'DataTable: Error in filteredAndSortedData', err);
      setRenderError(err instanceof Error ? err : new Error(String(err)));
      return [];
    }
  }, [data, searchQuery, searchKeys, activeFilters, sortKey, sortDirection, columns]);

  const hasActiveFilters = searchQuery !== '' || Object.keys(activeFilters).length > 0;

  // Fixes #680: Show error state if error prop is provided or render error occurred
  const displayError = error || renderError?.message;

  return (
    <div class="stack-sm">
      {/* Error state (fixes #680) */}
      {displayError ? (
        <div
          class={cn(
            spacing.pad.sm,
            radius.md,
            'bg-status-error/10 border border-status-error/20 text-status-error body-small',
            layout.inline.tight,
          )}
          role="alert"
        >
          <span class={iconTokens.size.sm}>⚠</span>
          <span>{displayError}</span>
        </div>
      ) : null}

      {/* Loading state (fixes #680) */}
      {loading ? (
        // biome-ignore lint/a11y/useSemanticElements: Status role is semantically correct for loading indicator
        <div
          class={cn(
            spacing.pad.sm,
            radius.md,
            'bg-surface-hover text-text-muted body-small text-center',
          )}
          role="status"
          aria-label="Loading data"
        >
          <span class="inline-block animate-spin mr-2">◐</span>
          Loading data...
        </div>
      ) : null}
      {/* Search and Filter Bar */}
      <div class={layout.inline.default}>
        <div class="relative flex-1">
          <Search
            class={cn(
              iconTokens.size.sm,
              'absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none',
            )}
          />
          <input
            type="text"
            value={searchQuery}
            onChange={(e: React.ChangeEvent<HTMLInputElement>): void =>
              setSearchQuery(e.target.value)
            }
            placeholder={searchPlaceholder}
            class={cn(
              'w-full pl-9 pr-8',
              spacing.compact.pyMd,
              'body-small bg-surface-base text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-1 focus:ring-brand-primary',
              border.card,
              radius.lg,
            )}
          />
          {searchQuery ? (
            <button
              type="button"
              onClick={(): void => setSearchQuery('')}
              class="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
              aria-label="Clear search"
            >
              <X class={iconTokens.size.sm} />
            </button>
          ) : null}
        </div>
        {filterOptions && filterOptions.length > 0 ? (
          <button
            type="button"
            onClick={(): void => setShowFilters(!showFilters)}
            class={cn(
              spacing.iconBtn.md,
              'transition-colors',
              radius.lg,
              border.width.default,
              showFilters || hasActiveFilters
                ? 'bg-brand-primary/20 border-brand-primary text-brand-primary'
                : 'border-surface-border text-text-muted hover:text-text-primary hover:border-text-muted',
            )}
            title="Toggle filters"
          >
            <Filter class={iconTokens.size.sm} />
          </button>
        ) : null}
      </div>

      {/* Filter Dropdowns */}
      {showFilters && filterOptions ? (
        <div class={cn(layout.inline.wrap, spacing.pad.xs, 'bg-surface-hover', radius.lg)}>
          {filterOptions.map((filter) => (
            <select
              key={filter.key}
              value={activeFilters[filter.key] || ''}
              onChange={(e: React.ChangeEvent<HTMLSelectElement>): void =>
                handleFilterChange(filter.key, e.target.value)
              }
              class={cn(
                spacing.cell.px,
                spacing.compact.py,
                'caption bg-surface-base text-text-primary focus:outline-none focus:ring-1 focus:ring-brand-primary',
                border.card,
                radius.default,
              )}
              aria-label={filter.label}
            >
              <option value="">{filter.label}</option>
              {filter.options.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          ))}
          {hasActiveFilters ? (
            <button
              type="button"
              onClick={clearFilters}
              class={cn(
                spacing.cell.px,
                spacing.compact.py,
                'caption text-text-muted hover:text-text-primary',
              )}
            >
              {t('dataTable.clearAll')}
            </button>
          ) : null}
        </div>
      ) : null}

      {/* Results count */}
      {hasActiveFilters ? (
        <p class="caption">
          {t('dataTable.showingResults', {
            shown: filteredAndSortedData.length,
            total: data.length,
          })}
        </p>
      ) : null}

      {/* Table */}
      <div class={cn('overflow-y-auto', maxHeight)}>
        <table class="w-full body-small">
          <thead class="sticky top-0 bg-surface-raised z-10">
            <tr class={border.divider}>
              {columns.map((column) => (
                <th
                  key={column.key}
                  class={cn(
                    spacing.cell.px,
                    spacing.compact.pyMd,
                    'text-left section-title',
                    column.hiddenOnMobile && 'hidden sm:table-cell',
                    column.sortable && 'cursor-pointer hover:text-text-primary select-none',
                    column.width ? `w-[${column.width}]` : '',
                  )}
                  onClick={column.sortable ? (): void => handleSort(column.key) : undefined}
                >
                  <span class={layout.inline.tight}>
                    {column.header}
                    {column.sortable ? (
                      <sortIcon
                        direction={sortKey === column.key ? sortDirection : null}
                        active={sortKey === column.key}
                      />
                    ) : null}
                  </span>
                </th>
              ))}
              {actions ? <th class={cn(spacing.cell.px, spacing.compact.pyMd, 'w-16')} /> : null}
            </tr>
          </thead>
          <tbody>
            {filteredAndSortedData.length === 0 ? (
              <tr>
                <td
                  colSpan={columns.length + (actions ? 1 : 0)}
                  class={cn(spacing.tableCell.empty, 'text-center text-text-muted')}
                >
                  {emptyMessage}
                </td>
              </tr>
            ) : (
              filteredAndSortedData.map((item) => {
                // Fixes #680: Add null checks for safe rendering
                if (!item) {
                  logger.warn('ui', 'DataTable: Encountered null/undefined item in data');
                  return null;
                }

                try {
                  const key = keyExtractor(item);
                  const expanded = isExpanded?.(item) ?? false;

                  return (
                    <tr
                      key={key}
                      class={cn(
                        'border-b border-surface-border/50',
                        onRowClick && 'cursor-pointer hover:bg-surface-hover',
                        expanded && 'bg-surface-hover/50',
                      )}
                      onClick={onRowClick ? (): void => onRowClick(item) : undefined}
                    >
                      {columns.map((column) => {
                        try {
                          return (
                            <td
                              key={`${key}-${column.key}`}
                              class={cn(
                                spacing.cell.px,
                                spacing.row.py,
                                column.hiddenOnMobile && 'hidden sm:table-cell',
                              )}
                            >
                              {column.render ? column.render(item) : (column.accessor(item) ?? '-')}
                            </td>
                          );
                        } catch (err) {
                          logger.error('ui', 'DataTable: Error rendering column', err, {
                            columnKey: column.key,
                          });
                          return (
                            <td
                              key={`${key}-${column.key}`}
                              class={cn(
                                spacing.cell.px,
                                spacing.row.py,
                                column.hiddenOnMobile && 'hidden sm:table-cell',
                              )}
                            >
                              <span class="text-status-error">{t('status.error')}</span>
                            </td>
                          );
                        }
                      })}
                      {actions ? (
                        <td class={cn(spacing.cell.px, spacing.row.py, 'text-right')}>
                          {((): React.ReactNode => {
                            try {
                              return actions(item);
                            } catch (err) {
                              logger.error('ui', 'DataTable: Error rendering actions', err);
                              return null;
                            }
                          })()}
                        </td>
                      ) : null}
                    </tr>
                  );
                } catch (err) {
                  logger.error('ui', 'DataTable: Error rendering row', err);
                  return null;
                }
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
