/**
 * DataTable: the one table. A semantic <table> in a hairline frame, with a
 * sticky-free header, 44px rows, sortable columns, optional selection, search,
 * row detail and row actions, and a pager.
 *
 * Based on niac's DataTable (typed columns that own their cells, a sort that
 * cycles, select-all, virtual scrolling, row click), with pagination,
 * `aria-sort`, skeleton loading and the contained scroll added. seed's table,
 * stem's list and niac's CSS-grid device list all give way to this.
 *
 * - Sorting: a column with `sortValue` sorts. `collation: 'natural'` compares
 *   strings numeric-aware ("eth2" before "eth10", "10.0.0.2" before
 *   "10.0.0.10"); numbers always compare as numbers, and a null or undefined
 *   value sorts last in either direction. `sortCycle` is `asc-desc-off`
 *   (default: a third click clears the sort) or `asc-desc`. `onSortChange`
 *   reports the state.
 * - Row click: `onRowClick` makes rows focusable; Enter or Space on a focused
 *   row activates it. `selectedRowKey` marks the open row with `aria-selected`.
 * - Row hook: `getRowProps(row)` returns a `data-testid`, a `className`, a
 *   `style` and a status `tone` wash for that row (niac's packet colouring rules,
 *   a failed run). A tone is colour only: say the same in a cell.
 * - Selection: `selection` adds a checkbox column. Its scope is `page` (the
 *   rows on this page or window) or `all` (every row that passes the search,
 *   across pages and past the virtual window). Either way the header checkbox is
 *   named in `selectAllLabel`, which for `all` receives the count, so the label
 *   says what it will do; `statusLabel` adds a "3 of 543 selected" line.
 * - Search: `search` renders a SearchInput above the table that filters rows by
 *   `text(row)` (case-insensitive substring) before they are sorted, paged and
 *   selected; `noResults` shows when nothing matches. `toolbar` is a slot beside
 *   it for filters. Both sit in the same frame as the table.
 * - Row detail: `expandable` adds a toggle column and renders `render(row)`
 *   under an expanded row, full width. Not with `virtualize`, which needs rows of
 *   one height (the types refuse the combination).
 * - Virtualization: `virtualize` mounts only the rows near the viewport for
 *   very long lists (niac's device list runs to 543). It shows every row in a
 *   bounded scroller instead of paging, and rows must stay one height.
 * - Sticky header: `maxHeight` bounds the table vertically and pins the header.
 * - Loading: `loading` shows skeleton rows under the real header, with
 *   `aria-busy` and a status line, never a bare "Loading...".
 *
 * At 390px the table scrolls sideways inside its own frame, never the page,
 * and says so with `data-phone-width-exempt`, which the fleet's phone-width
 * gate needs to tell a deliberate scroller from content leaving the viewport.
 * The frame is `relative`, so the screen-reader-only caption and header text,
 * which are absolutely positioned, are contained by it and clipped by its
 * overflow instead of widening the page (they added 765px of document scroll at
 * 390px when the frame was not a containing block). While the content scrolls,
 * the frame itself takes `tabIndex={0}` and the table's name (a labelled group),
 * so a keyboard user can pan it (axe `scrollable-region-focusable`) even when no
 * focusable control is inside.
 *
 * Pagination, sorting, search and selection are client-side over `rows`. A list
 * that is paged by the server is not this component; do not fake it by passing
 * one page of rows and a `pageSize` that hides the pager.
 */
import { ChevronDown, ChevronLeft, ChevronRight, ChevronsUpDown, ChevronUp } from 'lucide-react';
import {
  type CSSProperties,
  type KeyboardEvent,
  type ReactNode,
  type RefObject,
  useEffect,
  useId,
  useRef,
  useState,
} from 'react';
import { checkboxControlClass } from './Checkbox';
import { cx } from './cx';
import { SearchInput } from './SearchInput';
import { Skeleton } from './Skeleton';
import type { StatusTone } from './StatusPill';
import { useScrollable } from './useScrollable';
import { useVirtualWindow } from './useVirtualWindow';

export type SortDirection = 'asc' | 'desc';
export interface DataTableSort {
  key: string;
  direction: SortDirection;
}
type SortState = DataTableSort | null;
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

interface VirtualizeOptions {
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

const DEFAULT_PAGE_SIZE = 25;
const SKELETON_ROW_KEYS = ['one', 'two', 'three', 'four', 'five'] as const;
const DEFAULT_VIRTUAL_THRESHOLD = 100;
const DEFAULT_ROW_HEIGHT = 44;
const DEFAULT_VIRTUAL_HEIGHT = 480;

const toneWash: Record<StatusTone, string> = {
  success: 'bg-status-success/10',
  warning: 'bg-status-warning/10',
  error: 'bg-status-error/10',
  info: 'bg-status-info/10',
  unknown: '',
};

function compareValues(a: string | number, b: string | number, natural: boolean): number {
  if (typeof a === 'number' && typeof b === 'number') {
    return a - b;
  }
  return String(a).localeCompare(
    String(b),
    undefined,
    natural ? { numeric: true, sensitivity: 'base' } : undefined,
  );
}

function sortRows<T>(rows: readonly T[], column: DataTableColumn<T>, direction: SortDirection) {
  const { sortValue } = column;
  if (!sortValue) {
    return rows;
  }
  const sign = direction === 'asc' ? 1 : -1;
  const natural = column.collation === 'natural';
  return rows
    .map((row) => ({ row, value: sortValue(row) }))
    .sort((a, b) => {
      const aMissing = a.value === null || a.value === undefined;
      const bMissing = b.value === null || b.value === undefined;
      if (aMissing || bMissing) {
        return Number(aMissing) - Number(bMissing);
      }
      return sign * compareValues(a.value as string | number, b.value as string | number, natural);
    })
    .map(({ row }) => row);
}

function SortIcon({ direction }: { direction: SortDirection | null }) {
  const Icon = { asc: ChevronUp, desc: ChevronDown, none: ChevronsUpDown }[direction ?? 'none'];
  return (
    <Icon
      className={cx('h-3.5 w-3.5 shrink-0', direction ? 'text-text-primary' : 'text-text-muted')}
      aria-hidden="true"
    />
  );
}

function ariaSort(direction: SortDirection | null): 'ascending' | 'descending' | 'none' {
  if (direction === null) {
    return 'none';
  }
  return direction === 'asc' ? 'ascending' : 'descending';
}

const cellPad = 'px-4 py-2';
const focusRing =
  'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary';
const pagerButton =
  'inline-flex h-8 w-8 items-center justify-center rounded-lg text-text-primary hover:bg-surface-hover focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary disabled:cursor-not-allowed disabled:text-text-disabled disabled:hover:bg-transparent';

interface HeadProps<T> {
  columns: readonly DataTableColumn<T>[];
  sort: SortState;
  onSort: (key: string) => void;
  selection: DataTableSelection | undefined;
  selectAllRef: RefObject<HTMLInputElement | null>;
  selectAllLabel: string;
  allSelected: boolean;
  onToggleAll: () => void;
  loading: boolean;
  expandableLabel: string | undefined;
  actionsLabel: string | undefined;
  sticky: boolean;
}

function TableHead<T>({
  columns,
  sort,
  onSort,
  selection,
  selectAllRef,
  selectAllLabel,
  allSelected,
  onToggleAll,
  loading,
  expandableLabel,
  actionsLabel,
  sticky,
}: HeadProps<T>) {
  return (
    <thead
      className={cx(
        'bg-surface-sunken text-xs uppercase tracking-wide text-text-muted',
        sticky && 'sticky top-0 z-10',
      )}
    >
      <tr className="h-10">
        {expandableLabel === undefined ? null : (
          <th scope="col" className={cx(cellPad, 'relative w-10')}>
            <span className="sr-only">{expandableLabel}</span>
          </th>
        )}
        {selection ? (
          <th scope="col" className={cx(cellPad, 'w-10 text-left')}>
            <input
              ref={selectAllRef}
              type="checkbox"
              checked={allSelected}
              onChange={onToggleAll}
              disabled={loading}
              aria-label={selectAllLabel}
              className={checkboxControlClass}
            />
          </th>
        ) : null}
        {columns.map((column) => {
          const direction = sort?.key === column.key ? sort.direction : null;
          return (
            <th
              key={column.key}
              scope="col"
              aria-sort={column.sortValue ? ariaSort(direction) : undefined}
              className={cx(
                cellPad,
                'font-semibold',
                column.align === 'right' ? 'text-right' : 'text-left',
              )}
            >
              {column.sortValue ? (
                <button
                  type="button"
                  onClick={() => onSort(column.key)}
                  className={cx(
                    '-mx-1 inline-flex min-h-6 min-w-6 items-center gap-1 rounded px-1 uppercase tracking-wide hover:text-text-primary',
                    focusRing,
                    direction && 'text-text-primary',
                  )}
                >
                  {column.header}
                  <SortIcon direction={direction} />
                </button>
              ) : (
                column.header
              )}
            </th>
          );
        })}
        {actionsLabel === undefined ? null : (
          <th scope="col" className={cx(cellPad, 'relative')}>
            <span className="sr-only">{actionsLabel}</span>
          </th>
        )}
      </tr>
    </thead>
  );
}

interface RowProps<T> {
  row: T;
  rowKey: string;
  columns: readonly DataTableColumn<T>[];
  columnCount: number;
  selection: DataTableSelection | undefined;
  selectRowLabel: string;
  current: boolean;
  checked: boolean;
  onToggleRow: (key: string) => void;
  onRowClick: ((row: T) => void) | undefined;
  extra: DataTableRowProps | undefined;
  rowActions: ((row: T) => ReactNode) | undefined;
  expandable: DataTableExpandable<T> | undefined;
  expanded: boolean;
  detailId: string;
  onToggleExpanded: (key: string) => void;
  ariaRowIndex: number | undefined;
  tracksSelection: boolean;
}

function TableRow<T>({
  row,
  rowKey,
  columns,
  columnCount,
  selection,
  selectRowLabel,
  current,
  checked,
  onToggleRow,
  onRowClick,
  extra,
  rowActions,
  expandable,
  expanded,
  detailId,
  onToggleExpanded,
  ariaRowIndex,
  tracksSelection,
}: RowProps<T>) {
  const selected = checked || current;
  const activate = (event: KeyboardEvent<HTMLTableRowElement>) => {
    if (event.target === event.currentTarget && (event.key === 'Enter' || event.key === ' ')) {
      event.preventDefault();
      onRowClick?.(row);
    }
  };
  return (
    <>
      <tr
        aria-selected={tracksSelection ? selected : undefined}
        aria-rowindex={ariaRowIndex}
        tabIndex={onRowClick ? 0 : undefined}
        onClick={onRowClick ? () => onRowClick(row) : undefined}
        onKeyDown={onRowClick ? activate : undefined}
        data-testid={extra?.['data-testid']}
        style={extra?.style}
        className={cx(
          'h-11 hover:bg-surface-hover',
          selected ? 'bg-surface-hover' : extra?.tone && toneWash[extra.tone],
          onRowClick &&
            'cursor-pointer focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-brand-primary',
          extra?.className,
        )}
      >
        {expandable ? (
          <td className={cellPad}>
            <button
              type="button"
              aria-expanded={expanded}
              aria-controls={expanded ? detailId : undefined}
              aria-label={expandable.toggleLabel(row, expanded)}
              onClick={(event) => {
                event.stopPropagation();
                onToggleExpanded(rowKey);
              }}
              className={cx(
                'inline-flex h-6 w-6 items-center justify-center rounded text-text-muted hover:text-text-primary',
                focusRing,
              )}
            >
              <ChevronRight
                className={cx('h-4 w-4 transition-transform', expanded && 'rotate-90')}
                aria-hidden="true"
              />
            </button>
          </td>
        ) : null}
        {selection ? (
          <td className={cellPad}>
            <input
              type="checkbox"
              checked={checked}
              onChange={() => onToggleRow(rowKey)}
              onClick={(event) => event.stopPropagation()}
              aria-label={selectRowLabel}
              className={checkboxControlClass}
            />
          </td>
        ) : null}
        {columns.map((column) => (
          <td key={column.key} className={cx(cellPad, column.align === 'right' && 'text-right')}>
            {column.cell(row)}
          </td>
        ))}
        {rowActions ? (
          <td
            className={cx(cellPad, 'text-right')}
            onClick={(event) => event.stopPropagation()}
            onKeyDown={(event) => event.stopPropagation()}
          >
            <div className="inline-flex items-center gap-1">{rowActions(row)}</div>
          </td>
        ) : null}
      </tr>
      {expandable && expanded ? (
        <tr id={detailId} data-testid="data-table-detail-row">
          <td colSpan={columnCount} className="bg-surface-base px-4 py-3">
            {expandable.render(row)}
          </td>
        </tr>
      ) : null}
    </>
  );
}

interface PagerProps {
  label: string;
  status: string;
  page: number;
  pageCount: number;
  previousLabel: string;
  nextLabel: string;
  onPage: (page: number) => void;
}

function Pager({ label, status, page, pageCount, previousLabel, nextLabel, onPage }: PagerProps) {
  return (
    <nav
      aria-label={label}
      className="flex flex-wrap items-center justify-between gap-2 border-t border-hairline px-4 py-2 text-xs text-text-muted"
    >
      <span aria-live="polite">{status}</span>
      <div className="flex items-center gap-1">
        <button
          type="button"
          onClick={() => onPage(page - 1)}
          disabled={page === 1}
          aria-label={previousLabel}
          className={pagerButton}
        >
          <ChevronLeft className="h-4 w-4" aria-hidden="true" />
        </button>
        <button
          type="button"
          onClick={() => onPage(page + 1)}
          disabled={page === pageCount}
          aria-label={nextLabel}
          className={pagerButton}
        >
          <ChevronRight className="h-4 w-4" aria-hidden="true" />
        </button>
      </div>
    </nav>
  );
}

function SkeletonRows({ cellKeys, count }: { cellKeys: readonly string[]; count: number }) {
  return SKELETON_ROW_KEYS.slice(0, count).map((rowKey) => (
    <tr key={rowKey} className="h-11" data-testid="data-table-skeleton-row">
      {cellKeys.map((cellKey, position) => (
        <td key={cellKey} className={cellPad}>
          <Skeleton className={position === 0 ? 'w-3/4' : 'w-1/2'} />
        </td>
      ))}
    </tr>
  ));
}

function Spacer({ height, columnCount }: { height: number; columnCount: number }) {
  return height > 0 ? (
    <tr>
      <td aria-hidden="true" colSpan={columnCount} style={{ height, padding: 0 }} />
    </tr>
  ) : null;
}

function toggled(keys: ReadonlySet<string>, key: string): ReadonlySet<string> {
  const next = new Set(keys);
  if (!next.delete(key)) {
    next.add(key);
  }
  return next;
}

interface ModelArgs<T> {
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

function nextSort(
  previous: SortState,
  key: string,
  sortCycle: 'asc-desc-off' | 'asc-desc',
): SortState {
  if (previous?.key !== key) {
    return { key, direction: 'asc' };
  }
  if (previous.direction === 'asc') {
    return { key, direction: 'desc' };
  }
  return sortCycle === 'asc-desc' ? { key, direction: 'asc' } : null;
}

/** Search, sort, page and selection coverage over `rows`, all client-side. */
function useTableModel<T>({
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

interface ToolbarProps<T> {
  search: DataTableSearch<T> | undefined;
  query: string;
  onQuery: (value: string) => void;
  toolbar: ReactNode;
}

function Toolbar<T>({ search, query, onQuery, toolbar }: ToolbarProps<T>) {
  return (
    <div className="flex flex-wrap items-center gap-2 border-b border-hairline px-4 py-3">
      {search ? (
        <SearchInput
          value={query}
          onValueChange={onQuery}
          label={search.label}
          clearLabel={search.clearLabel}
          {...(search.placeholder === undefined ? {} : { placeholder: search.placeholder })}
          className="min-w-0 flex-1"
        />
      ) : null}
      {toolbar}
    </div>
  );
}

export function DataTable<T>({
  rows,
  columns,
  getRowKey,
  caption,
  empty,
  labels,
  defaultSort,
  sortCycle = 'asc-desc-off',
  onSortChange,
  pageSize = DEFAULT_PAGE_SIZE,
  selection,
  search,
  toolbar,
  onRowClick,
  selectedRowKey,
  getRowProps,
  loading = false,
  maxHeight,
  virtualize,
  expandable,
  rowActions,
  'data-testid': testId,
}: DataTableProps<T>) {
  const detailPrefix = useId();
  const [openRows, setOpenRows] = useState<ReadonlySet<string>>(
    expandable?.defaultExpandedKeys ?? new Set(),
  );
  const selectAllRef = useRef<HTMLInputElement>(null);
  const frame = useScrollable<HTMLDivElement>();
  const model = useTableModel({
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
  });
  const { allSelected, selectedCovered, rowWindow, virtual, rowHeight } = model;
  const expandedKeys = expandable?.expandedKeys ?? openRows;

  useEffect(() => {
    if (selectAllRef.current) {
      selectAllRef.current.indeterminate = selectedCovered > 0 && !allSelected;
    }
  }, [selectedCovered, allSelected]);

  if (rows.length === 0 && !loading) {
    return <div data-testid={testId}>{empty}</div>;
  }

  const toggleAll = () => {
    if (!selection) {
      return;
    }
    const next = new Set(selection.selectedKeys);
    for (const key of model.coveredKeys) {
      if (allSelected) {
        next.delete(key);
      } else {
        next.add(key);
      }
    }
    selection.onChange(next);
  };

  const toggleExpanded = (key: string) => {
    const next = toggled(expandedKeys, key);
    setOpenRows(next);
    expandable?.onExpandedChange?.(next);
  };

  const bounded = maxHeight !== undefined || virtual;
  const cellKeys = [
    ...(expandable ? ['expand'] : []),
    ...(selection ? ['select'] : []),
    ...columns.map((column) => column.key),
    ...(rowActions ? ['actions'] : []),
  ];
  const columnCount = cellKeys.length;
  const filteredOut = search !== undefined && model.sortedRows.length === 0 && !loading;
  const selectedCount = selection?.selectedKeys.size ?? 0;
  const frameAttributes = frame.scrolls
    ? { tabIndex: 0, role: 'group', 'aria-label': caption }
    : {};

  return (
    <div className="panel min-w-0 overflow-hidden" data-testid={testId}>
      {search || toolbar ? (
        <Toolbar
          search={search}
          query={model.query}
          onQuery={model.changeQuery}
          toolbar={toolbar}
        />
      ) : null}
      {filteredOut ? (
        search.noResults
      ) : (
        <div
          ref={frame.ref}
          {...frameAttributes}
          className={cx(
            'relative overflow-x-auto',
            bounded && 'overflow-y-auto',
            frame.scrolls &&
              'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-brand-primary',
          )}
          style={bounded ? { maxHeight: model.viewportHeight } : undefined}
          onScroll={virtual ? rowWindow.onScroll : undefined}
          data-phone-width-exempt="table-scroll"
        >
          <table
            className="w-full min-w-max text-sm"
            aria-busy={loading || undefined}
            aria-rowcount={virtual ? model.sortedRows.length + 1 : undefined}
          >
            <caption className="sr-only">{caption}</caption>
            <TableHead
              columns={columns}
              sort={model.sort}
              onSort={model.toggleSort}
              selection={selection}
              selectAllRef={selectAllRef}
              selectAllLabel={
                selection?.scope === 'all'
                  ? selection.selectAllLabel(model.coveredKeys.length)
                  : labels.selectAll
              }
              allSelected={allSelected}
              onToggleAll={toggleAll}
              loading={loading}
              expandableLabel={expandable?.columnLabel}
              actionsLabel={rowActions ? labels.actions : undefined}
              sticky={bounded}
            />
            <tbody className="divide-y divide-hairline text-text-secondary">
              {loading ? (
                <SkeletonRows cellKeys={cellKeys} count={pageSize} />
              ) : (
                <>
                  <Spacer
                    height={virtual ? rowWindow.start * rowHeight : 0}
                    columnCount={columnCount}
                  />
                  {model.pageRows.map((row, offset) => {
                    const key = getRowKey(row);
                    return (
                      <TableRow
                        key={key}
                        row={row}
                        rowKey={key}
                        columns={columns}
                        columnCount={columnCount}
                        selection={selection}
                        selectRowLabel={selection ? labels.selectRow(row) : ''}
                        current={key === selectedRowKey}
                        checked={selection?.selectedKeys.has(key) ?? false}
                        onToggleRow={(rowKeyToToggle) =>
                          selection?.onChange(toggled(selection.selectedKeys, rowKeyToToggle))
                        }
                        onRowClick={onRowClick}
                        extra={getRowProps?.(row)}
                        rowActions={rowActions}
                        expandable={expandable}
                        expanded={expandable !== undefined && expandedKeys.has(key)}
                        detailId={`${detailPrefix}-${key}`}
                        onToggleExpanded={toggleExpanded}
                        ariaRowIndex={virtual ? rowWindow.start + offset + 2 : undefined}
                        tracksSelection={selection !== undefined || selectedRowKey !== undefined}
                      />
                    );
                  })}
                  <Spacer
                    height={virtual ? (model.sortedRows.length - rowWindow.end) * rowHeight : 0}
                    columnCount={columnCount}
                  />
                </>
              )}
            </tbody>
          </table>
        </div>
      )}
      {loading ? (
        <p role="status" className="sr-only">
          {labels.loading}
        </p>
      ) : null}
      {selection?.statusLabel && selectedCount > 0 ? (
        <p
          role="status"
          className="border-t border-hairline px-4 py-2 text-xs text-text-muted"
          data-testid="data-table-selection-status"
        >
          {selection.statusLabel(selectedCount, rows.length)}
        </p>
      ) : null}
      {model.pageCount > 1 && !virtual && !loading ? (
        <Pager
          label={labels.pagination}
          status={labels.pageStatus(model.page, model.pageCount, model.sortedRows.length)}
          page={model.page}
          pageCount={model.pageCount}
          previousLabel={labels.previousPage}
          nextLabel={labels.nextPage}
          onPage={model.setPage}
        />
      ) : null}
    </div>
  );
}
