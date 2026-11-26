import { ChevronLeft, ChevronRight } from 'lucide-react';
import type { ReactNode } from 'react';
import { cellPad, pagerButton } from './DataTableStyles';
import type { DataTableSearch } from './DataTableTypes';
import { SearchInput } from './SearchInput';
import { Skeleton } from './Skeleton';

const SKELETON_ROW_KEYS = ['one', 'two', 'three', 'four', 'five'] as const;

interface PagerProps {
  label: string;
  status: string;
  page: number;
  pageCount: number;
  previousLabel: string;
  nextLabel: string;
  onPage: (page: number) => void;
}

export function Pager({
  label,
  status,
  page,
  pageCount,
  previousLabel,
  nextLabel,
  onPage,
}: PagerProps) {
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

export function SkeletonRows({ cellKeys, count }: { cellKeys: readonly string[]; count: number }) {
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

export function Spacer({ height, columnCount }: { height: number; columnCount: number }) {
  return height > 0 ? (
    <tr>
      <td aria-hidden="true" colSpan={columnCount} style={{ height, padding: 0 }} />
    </tr>
  ) : null;
}

interface ToolbarProps<T> {
  search: DataTableSearch<T> | undefined;
  query: string;
  onQuery: (value: string) => void;
  toolbar: ReactNode;
}

export function Toolbar<T>({ search, query, onQuery, toolbar }: ToolbarProps<T>) {
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
