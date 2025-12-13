import { useState, useMemo, useCallback, ReactNode } from "react";
import {
  ChevronUp,
  ChevronDown,
  ArrowUpDown,
  Search,
  X,
  Filter,
} from "lucide-react";

export type SortDirection = "asc" | "desc" | null;

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
}

function SortIcon({
  direction,
  active,
}: {
  direction: SortDirection;
  active: boolean;
}) {
  if (!active || !direction) {
    return <ArrowUpDown className="w-3 h-3 opacity-40" />;
  }
  return direction === "asc" ? (
    <ChevronUp className="w-3 h-3" />
  ) : (
    <ChevronDown className="w-3 h-3" />
  );
}

export function DataTable<T>({
  data,
  columns,
  keyExtractor,
  searchPlaceholder = "Search...",
  searchKeys,
  onRowClick,
  expandedContent: _expandedContent,
  isExpanded,
  emptyMessage = "No data found",
  maxHeight = "max-h-80",
  actions,
  filterOptions,
}: DataTableProps<T>) {
  // Note: expandedContent is available for future row expansion feature
  void _expandedContent;
  const [searchQuery, setSearchQuery] = useState("");
  const [sortKey, setSortKey] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<SortDirection>(null);
  const [activeFilters, setActiveFilters] = useState<Record<string, string>>(
    {},
  );
  const [showFilters, setShowFilters] = useState(false);

  const handleSort = useCallback(
    (key: string) => {
      if (sortKey === key) {
        // Cycle through: asc -> desc -> null
        if (sortDirection === "asc") {
          setSortDirection("desc");
        } else if (sortDirection === "desc") {
          setSortDirection(null);
          setSortKey(null);
        }
      } else {
        setSortKey(key);
        setSortDirection("asc");
      }
    },
    [sortKey, sortDirection],
  );

  const handleFilterChange = useCallback((key: string, value: string) => {
    setActiveFilters((prev) => {
      if (value === "") {
        const { [key]: _, ...rest } = prev;
        return rest;
      }
      return { ...prev, [key]: value };
    });
  }, []);

  const clearFilters = useCallback(() => {
    setActiveFilters({});
    setSearchQuery("");
  }, []);

  const filteredAndSortedData = useMemo(() => {
    let result = [...data];

    // Apply search filter
    if (searchQuery && searchKeys && searchKeys.length > 0) {
      const query = searchQuery.toLowerCase();
      result = result.filter((item) => {
        return searchKeys.some((key) => {
          const column = columns.find((c) => c.key === key);
          if (column) {
            const value = column.accessor(item);
            return value?.toString().toLowerCase().includes(query);
          }
          return false;
        });
      });
    }

    // Apply column filters
    Object.entries(activeFilters).forEach(([key, filterValue]) => {
      if (filterValue) {
        const column = columns.find((c) => c.key === key);
        if (column) {
          result = result.filter((item) => {
            const value = column.accessor(item);
            return value
              ?.toString()
              .toLowerCase()
              .includes(filterValue.toLowerCase());
          });
        }
      }
    });

    // Apply sorting
    if (sortKey && sortDirection) {
      const column = columns.find((c) => c.key === sortKey);
      if (column) {
        result.sort((a, b) => {
          const aVal = column.accessor(a);
          const bVal = column.accessor(b);

          // Handle nulls
          if (aVal == null && bVal == null) return 0;
          if (aVal == null) return sortDirection === "asc" ? 1 : -1;
          if (bVal == null) return sortDirection === "asc" ? -1 : 1;

          // Compare values
          if (typeof aVal === "number" && typeof bVal === "number") {
            return sortDirection === "asc" ? aVal - bVal : bVal - aVal;
          }

          const strA = String(aVal).toLowerCase();
          const strB = String(bVal).toLowerCase();
          const comparison = strA.localeCompare(strB, undefined, {
            numeric: true,
          });
          return sortDirection === "asc" ? comparison : -comparison;
        });
      }
    }

    return result;
  }, [
    data,
    searchQuery,
    searchKeys,
    activeFilters,
    sortKey,
    sortDirection,
    columns,
  ]);

  const hasActiveFilters =
    searchQuery !== "" || Object.keys(activeFilters).length > 0;

  return (
    <div className="space-y-2">
      {/* Search and Filter Bar */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted pointer-events-none" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={searchPlaceholder}
            className="w-full pl-9 pr-8 py-1.5 text-sm bg-surface-base border border-surface-border rounded-lg focus:outline-none focus:ring-1 focus:ring-brand-primary text-text-primary placeholder:text-text-muted"
          />
          {searchQuery && (
            <button
              type="button"
              onClick={() => setSearchQuery("")}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
            >
              <X className="w-4 h-4" />
            </button>
          )}
        </div>
        {filterOptions && filterOptions.length > 0 && (
          <button
            type="button"
            onClick={() => setShowFilters(!showFilters)}
            className={`p-1.5 rounded-lg border transition-colors ${
              showFilters || hasActiveFilters
                ? "bg-brand-primary/20 border-brand-primary text-brand-primary"
                : "border-surface-border text-text-muted hover:text-text-primary hover:border-text-muted"
            }`}
            title="Toggle filters"
          >
            <Filter className="w-4 h-4" />
          </button>
        )}
      </div>

      {/* Filter Dropdowns */}
      {showFilters && filterOptions && (
        <div className="flex items-center gap-2 flex-wrap p-2 bg-surface-hover rounded-lg">
          {filterOptions.map((filter) => (
            <select
              key={filter.key}
              value={activeFilters[filter.key] || ""}
              onChange={(e) => handleFilterChange(filter.key, e.target.value)}
              className="px-2 py-1 text-xs bg-surface-base border border-surface-border rounded focus:outline-none focus:ring-1 focus:ring-brand-primary text-text-primary"
            >
              <option value="">{filter.label}</option>
              {filter.options.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          ))}
          {hasActiveFilters && (
            <button
              type="button"
              onClick={clearFilters}
              className="px-2 py-1 text-xs text-text-muted hover:text-text-primary"
            >
              Clear all
            </button>
          )}
        </div>
      )}

      {/* Results count */}
      {hasActiveFilters && (
        <p className="text-xs text-text-muted">
          Showing {filteredAndSortedData.length} of {data.length} results
        </p>
      )}

      {/* Table */}
      <div className={`overflow-y-auto ${maxHeight}`}>
        <table className="w-full text-sm">
          <thead className="sticky top-0 bg-surface-raised z-10">
            <tr className="border-b border-surface-border">
              {columns.map((column) => (
                <th
                  key={column.key}
                  className={`px-2 py-1.5 text-left text-xs font-medium text-text-muted uppercase tracking-wider ${
                    column.hiddenOnMobile ? "hidden sm:table-cell" : ""
                  } ${column.sortable ? "cursor-pointer hover:text-text-primary select-none" : ""}`}
                  style={column.width ? { width: column.width } : undefined}
                  onClick={
                    column.sortable ? () => handleSort(column.key) : undefined
                  }
                >
                  <span className="flex items-center gap-1">
                    {column.header}
                    {column.sortable && (
                      <SortIcon
                        direction={
                          sortKey === column.key ? sortDirection : null
                        }
                        active={sortKey === column.key}
                      />
                    )}
                  </span>
                </th>
              ))}
              {actions && <th className="px-2 py-1.5 w-16"></th>}
            </tr>
          </thead>
          <tbody>
            {filteredAndSortedData.length === 0 ? (
              <tr>
                <td
                  colSpan={columns.length + (actions ? 1 : 0)}
                  className="px-2 py-4 text-center text-text-muted"
                >
                  {emptyMessage}
                </td>
              </tr>
            ) : (
              filteredAndSortedData.map((item) => {
                const key = keyExtractor(item);
                const expanded = isExpanded?.(item) ?? false;

                return (
                  <tr
                    key={key}
                    className={`border-b border-surface-border/50 ${
                      onRowClick ? "cursor-pointer hover:bg-surface-hover" : ""
                    } ${expanded ? "bg-surface-hover/50" : ""}`}
                    onClick={onRowClick ? () => onRowClick(item) : undefined}
                  >
                    {columns.map((column) => (
                      <td
                        key={`${key}-${column.key}`}
                        className={`px-2 py-2 ${
                          column.hiddenOnMobile ? "hidden sm:table-cell" : ""
                        }`}
                      >
                        {column.render
                          ? column.render(item)
                          : (column.accessor(item) ?? "-")}
                      </td>
                    ))}
                    {actions && (
                      <td className="px-2 py-2 text-right">{actions(item)}</td>
                    )}
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
