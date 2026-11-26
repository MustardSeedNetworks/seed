import { ChevronRight } from 'lucide-react';
import type { KeyboardEvent, ReactNode } from 'react';
import { checkboxControlClass } from './Checkbox';
import { cx } from './cx';
import { cellPad, focusRing, toneWash } from './DataTableStyles';
import type {
  DataTableColumn,
  DataTableExpandable,
  DataTableRowProps,
  DataTableSelection,
} from './DataTableTypes';

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

export function TableRow<T>({
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
