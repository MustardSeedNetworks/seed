import { ChevronDown, ChevronsUpDown, ChevronUp } from 'lucide-react';
import type { RefObject } from 'react';
import { checkboxControlClass } from './Checkbox';
import { cx } from './cx';
import { cellPad, focusRing } from './DataTableStyles';
import type {
  DataTableColumn,
  DataTableSelection,
  SortDirection,
  SortState,
} from './DataTableTypes';

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

export function TableHead<T>({
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
