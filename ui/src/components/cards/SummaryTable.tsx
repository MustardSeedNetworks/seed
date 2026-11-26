/**
 * SummaryTable — a captioned table of a job's tallies, for the cards that
 * summarise a bounded run (packet capture, multicast listen). Renders nothing
 * when there are no rows.
 */

import type { JSX } from 'react';

import { useTableLabels } from '../../hooks/useTableLabels';
import { DataTable, type DataTableColumn } from '../../shared/DataTable';

export interface SummaryColumn {
  label: string;
  mono?: boolean;
}

interface SummaryRow {
  key: string;
  cells: string[];
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
  rows: SummaryRow[];
}): JSX.Element | null {
  const labels = useTableLabels<SummaryRow>({ loading: caption });
  if (rows.length === 0) {
    return null;
  }
  // Cells are positional and the cell text is already formatted ("12 KB"), so
  // the columns keep the job's order rather than sorting it.
  const tableColumns: DataTableColumn<SummaryRow>[] = columns.map((column, index) => ({
    key: column.label,
    header: column.label,
    cell: (row) => (
      <span className={column.mono ? 'font-mono' : undefined} title={row.cells[index]}>
        {row.cells[index]}
      </span>
    ),
  }));
  return (
    <DataTable
      rows={rows}
      columns={tableColumns}
      getRowKey={(row) => row.key}
      caption={caption}
      empty={null}
      labels={labels}
      pageSize={10}
      data-testid={testId}
    />
  );
}
