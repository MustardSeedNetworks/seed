import type { CSSProperties, ReactNode } from 'react';
import type { StatusTone } from './StatusPill';

export type SortDirection = 'asc' | 'desc';
export interface DataTableSort {
  key: string;
  direction: SortDirection;
}
export type SortState = DataTableSort | null;
/** What a sortable value is: nullish sorts last. */
export type SortValue = string | number | null | undefined;

export interface DataTableColumn<T> {
  /** Stable id, also the sort key. */
  key: string;
  /** Already translated. */
  header: string;
  cell: (row: T) => ReactNode;
  /** Makes the column sortable by this value. */
  sortValue?: (row: T) => SortValue;
  /**
   * How strings compare: `text` (default) is locale order; `natural` also reads
   * digit runs as numbers, for names and addresses that end in one.
   */
  collation?: 'text' | 'natural';
  align?: 'left' | 'right';
}

interface SelectionBase {
  selectedKeys: ReadonlySet<string>;
  onChange: (keys: ReadonlySet<string>) => void;
  /** A live line under the table, e.g. "3 of 543 selected". Omit for none. */
  statusLabel?: (selected: number, total: number) => string;
}

export type DataTableSelection =
  | (SelectionBase & {
      /**
       * The header checkbox covers the rows on the current page (or virtual window),
       * and is named by `labels.selectAll`.
       */
      scope?: 'page';
    })
  | (SelectionBase & {
      /** The header checkbox covers every row that passes the search, on every page. */
      scope: 'all';
      /**
       * Accessible name of the header checkbox, in place of `labels.selectAll`
       * (which says "page"); gets the count, e.g. "Select all 543 devices".
       */
      selectAllLabel: (total: number) => string;
    });

export interface DataTableSearch<T> {
  /** Accessible name of the search box, e.g. "Search devices". */
  label: string;
  /** Accessible name of its clear button. */
  clearLabel: string;
  placeholder?: string;
  /** What a row is searched by; matching is a case-insensitive substring. */
  text: (row: T) => string;
  /** Shown instead of the table when no row matches: an EmptyState. */
  noResults: ReactNode;
}

export interface DataTableExpandable<T> {
  /** The detail rendered under an expanded row. */
  render: (row: T) => ReactNode;
  /** Screen-reader header of the toggle column. */
  columnLabel: string;
  /** Accessible name of a row's toggle, e.g. "Show details for core-sw". */
  toggleLabel: (row: T, expanded: boolean) => string;
  defaultExpandedKeys?: ReadonlySet<string>;
  /** Controls the expanded rows; pair with `onExpandedChange`. */
  expandedKeys?: ReadonlySet<string>;
  onExpandedChange?: (keys: ReadonlySet<string>) => void;
}

export interface DataTableRowProps {
  'data-testid'?: string;
  className?: string;
  style?: CSSProperties;
  /** A status wash on the row. Colour only: carry the meaning in text too. */
  tone?: StatusTone;
}

export interface DataTableLabels<T> {
  /** Names the select-all checkbox of a page-scope selection, e.g. "Select all rows on this page". */
  selectAll: string;
  selectRow: (row: T) => string;
  /** Screen-reader header of the row-actions column. */
  actions: string;
  /** Accessible name of the pager landmark. */
  pagination: string;
  previousPage: string;
  nextPage: string;
  /** Status line while loading, e.g. "Loading devices". */
  loading: string;
  /** e.g. "Page 2 of 5 (113 rows)". */
  pageStatus: (page: number, pageCount: number, total: number) => string;
}

interface DataTableBaseProps<T> {
  rows: readonly T[];
  columns: readonly DataTableColumn<T>[];
  getRowKey: (row: T) => string;
  /** Names the table and its scroll region. Already translated. */
  caption: string;
  /** Shown instead of the table when there are no rows: an EmptyState. */
  empty: ReactNode;
  labels: DataTableLabels<T>;
  defaultSort?: DataTableSort;
  /** `asc-desc-off` (default) clears the sort on a third click; `asc-desc` keeps one always set. */
  sortCycle?: 'asc-desc-off' | 'asc-desc';
  onSortChange?: (sort: SortState) => void;
  /** Rows per page. The pager appears only when there is more than one page. */
  pageSize?: number;
  selection?: DataTableSelection;
  search?: DataTableSearch<T>;
  /** Beside the search box: filters, a count, one button. */
  toolbar?: ReactNode;
  /** Makes rows activatable from the mouse and from Enter or Space. */
  onRowClick?: (row: T) => void;
  /** The row marked `aria-selected`, e.g. the one whose detail is open. */
  selectedRowKey?: string;
  /** Per-row test id, class, style and tone. */
  getRowProps?: (row: T) => DataTableRowProps;
  /** Shows skeleton rows instead of `rows` and `empty`. */
  loading?: boolean;
  /** Bounds the table's height in px and pins the header while the body scrolls. */
  maxHeight?: number;
  /** The last column: one to three icon buttons or a menu. */
  rowActions?: (row: T) => ReactNode;
  'data-testid'?: string;
}

export interface VirtualizeOptions {
  /** Rows before it switches on (default 100). */
  threshold?: number;
  /** One row's height in px (default 44, the standard row). */
  rowHeight?: number;
}

type DataTableModeProps<T> =
  | {
      /**
       * Mounts only the rows near the viewport once there are `threshold` rows.
       * Needs rows of one height and shows every row in a scroller bounded by
       * `maxHeight` (default 480) instead of paging.
       */
      virtualize?: VirtualizeOptions;
      expandable?: never;
    }
  | { virtualize?: never; expandable?: DataTableExpandable<T> };

export type DataTableProps<T> = DataTableBaseProps<T> & DataTableModeProps<T>;
