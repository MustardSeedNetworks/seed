/**
 * SummaryTable — a captioned, fixed-layout table of a job's tallies, for the
 * cards that summarise a bounded run (packet capture, multicast listen).
 * Renders nothing when there are no rows.
 */

import type { JSX } from 'react';

import { cn, radius } from '../../styles/theme';

export interface SummaryColumn {
  label: string;
  width: string;
  mono?: boolean;
}

export function SummaryTable({
  testId,
  caption,
  columns,
  rows,
}: {
  testId: string;
  caption: string;
  columns: SummaryColumn[];
  rows: { key: string; cells: string[] }[];
}): JSX.Element | null {
  if (rows.length === 0) {
    return null;
  }
  return (
    // See NeighbourCacheCard: table-fixed, not overflow-x-auto (#2708).
    <div className={cn(radius.default, 'border border-surface-border')}>
      <table className="w-full table-fixed body-small" data-testid={testId}>
        <caption className="caption text-text-muted text-left px-cell pt-row">{caption}</caption>
        <thead>
          <tr className="border-b border-surface-border text-text-muted">
            {columns.map((column) => (
              <th
                key={column.label}
                scope="col"
                className={cn('px-cell py-row text-left truncate', column.width)}
                title={column.label}
              >
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.key} className="border-b border-surface-border last:border-0">
              {row.cells.map((cell, i) => (
                <td
                  // Cells are positional; the column label is unique per table.
                  key={columns[i]?.label ?? i}
                  className={cn('px-cell py-row truncate', columns[i]?.mono && 'font-mono')}
                  title={cell}
                >
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
