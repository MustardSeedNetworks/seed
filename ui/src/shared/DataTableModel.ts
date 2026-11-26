import { useState } from 'react';
import { nextSort, sortRows } from './DataTableSort';
import type {
  DataTableColumn,
  DataTableSearch,
  DataTableSelection,
  DataTableSort,
  SortState,
  VirtualizeOptions,
} from './DataTableTypes';
import { useVirtualWindow } from './useVirtualWindow';

const DEFAULT_VIRTUAL_THRESHOLD = 100;
const DEFAULT_ROW_HEIGHT = 44;
const DEFAULT_VIRTUAL_HEIGHT = 480;

export interface ModelArgs<T> {
  rows: readonly T[];
  columns: readonly DataTableColumn<T>[];
  getRowKey: (row: T) => string;
  defaultSort: DataTableSort | undefined;
  sortCycle: 'asc-desc-off' | 'asc-desc';
  onSortChange: ((sort: SortState) => void) | undefined;
  pageSize: number;
  selection: DataTableSelection | undefined;
  search: DataTableSearch<T> | undefined;
  virtualize: VirtualizeOptions | undefined;
  maxHeight: number | undefined;
}

/** Search, sort, page and selection coverage over `rows`, all client-side. */
export function useTableModel<T>({
  rows,
  columns,
  getRowKey,
  defaultSort,
  sortCycle,
  onSortChange,
  pageSize,
  selection,
  search,
  virtualize,
  maxHeight,
}: ModelArgs<T>) {
  const [sort, setSort] = useState<SortState>(defaultSort ?? null);
  const [requestedPage, setRequestedPage] = useState(1);
  const [query, setQuery] = useState('');

  const needle = query.trim().toLowerCase();
  const filteredRows =
    search && needle ? rows.filter((row) => search.text(row).toLowerCase().includes(needle)) : rows;
  const sortColumn = sort ? columns.find((column) => column.key === sort.key) : undefined;
  const sortedRows =
    sort && sortColumn ? sortRows(filteredRows, sortColumn, sort.direction) : filteredRows;

  const pageCount = Math.max(1, Math.ceil(sortedRows.length / pageSize));
  const page = Math.min(requestedPage, pageCount);
  const virtual =
    virtualize !== undefined &&
    filteredRows.length >= (virtualize.threshold ?? DEFAULT_VIRTUAL_THRESHOLD);
  const rowHeight = virtualize?.rowHeight ?? DEFAULT_ROW_HEIGHT;
  const viewportHeight = maxHeight ?? DEFAULT_VIRTUAL_HEIGHT;
  const rowWindow = useVirtualWindow({ count: sortedRows.length, rowHeight, viewportHeight });
  const pageRows = virtual
    ? sortedRows.slice(rowWindow.start, rowWindow.end)
    : sortedRows.slice((page - 1) * pageSize, page * pageSize);

  const coveredKeys = (selection?.scope === 'all' ? sortedRows : pageRows).map(getRowKey);
  const selectedCovered = selection
    ? coveredKeys.filter((key) => selection.selectedKeys.has(key)).length
    : 0;

  return {
    sort,
    toggleSort: (key: string) => {
      const next = nextSort(sort, key, sortCycle);
      setRequestedPage(1);
      setSort(next);
      onSortChange?.(next);
    },
    query,
    changeQuery: (value: string) => {
      setQuery(value);
      setRequestedPage(1);
    },
    page,
    pageCount,
    setPage: setRequestedPage,
    sortedRows,
    pageRows,
    virtual,
    rowHeight,
    viewportHeight,
    rowWindow,
    coveredKeys,
    selectedCovered,
    allSelected: coveredKeys.length > 0 && selectedCovered === coveredKeys.length,
  };
}
