import type { DataTableColumn, SortDirection, SortState } from './DataTableTypes';

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

export function sortRows<T>(
  rows: readonly T[],
  column: DataTableColumn<T>,
  direction: SortDirection,
) {
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

export function nextSort(
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
