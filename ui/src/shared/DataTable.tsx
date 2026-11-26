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
import { useEffect, useId, useRef, useState } from 'react';
import { cx } from './cx';
import { TableHead } from './DataTableHead';
import { useTableModel } from './DataTableModel';
import { Pager, SkeletonRows, Spacer, Toolbar } from './DataTableParts';
import { TableRow } from './DataTableRow';
import type { DataTableProps } from './DataTableTypes';
import { useScrollable } from './useScrollable';

export type {
  DataTableColumn,
  DataTableExpandable,
  DataTableLabels,
  DataTableProps,
  DataTableRowProps,
  DataTableSearch,
  DataTableSelection,
  DataTableSort,
  SortDirection,
  SortValue,
} from './DataTableTypes';

const DEFAULT_PAGE_SIZE = 25;

function toggled(keys: ReadonlySet<string>, key: string): ReadonlySet<string> {
  const next = new Set(keys);
  if (!next.delete(key)) {
    next.add(key);
  }
  return next;
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
