import { screen, within } from '@testing-library/react';
import { useState } from 'react';
import {
  DataTable,
  type DataTableColumn,
  type DataTableLabels,
  type DataTableProps,
  type DataTableSelection,
} from './DataTable';

export interface Device {
  id: string;
  name: string;
  ports: number;
}

export const devices: Device[] = [
  { id: 'a', name: 'core', ports: 48 },
  { id: 'b', name: 'access', ports: 24 },
  { id: 'c', name: 'edge', ports: 8 },
];

export const columns: DataTableColumn<Device>[] = [
  { key: 'name', header: 'Name', cell: (d) => d.name, sortValue: (d) => d.name },
  {
    key: 'ports',
    header: 'Ports',
    cell: (d) => d.ports,
    sortValue: (d) => d.ports,
    align: 'right',
  },
];

export const labels: DataTableLabels<Device> = {
  selectRow: (d) => `Select ${d.name}`,
  selectAll: 'Select all',
  actions: 'Actions',
  loading: 'Loading devices',
  pagination: 'Pages',
  previousPage: 'Previous page',
  nextPage: 'Next page',
  pageStatus: (page, count, total) => `Page ${page} of ${count} (${total} rows)`,
};

export function names(): string[] {
  return screen
    .getAllByRole('row')
    .slice(1)
    .map((row) => within(row).getAllByRole('cell')[0]?.textContent ?? '');
}

export type Extra = Partial<DataTableProps<Device>>;

export const table = (extra: Extra = {}) => (
  <DataTable
    rows={devices}
    columns={columns}
    getRowKey={(d) => d.id}
    caption="Devices"
    empty={<p>none</p>}
    labels={labels}
    {...extra}
  />
);

export function SelectableTable() {
  const [selectedKeys, setSelected] = useState<ReadonlySet<string>>(new Set());
  return table({
    selection: { selectedKeys, onChange: setSelected },
  });
}

export const ordered = (keys: Iterable<string>) => [...keys].sort((a, b) => a.localeCompare(b));
export const frameOf = () => screen.getByRole('table').parentElement as HTMLElement;
export const many = (count: number): Device[] =>
  Array.from({ length: count }, (_, i) => ({ id: `${i}`, name: `d${i}`, ports: i }));

export function Selectable({
  selection,
  extra = {},
}: {
  selection: (
    keys: ReadonlySet<string>,
    set: (keys: ReadonlySet<string>) => void,
  ) => DataTableSelection;
  extra?: Extra;
}) {
  const [keys, setKeys] = useState<ReadonlySet<string>>(new Set());
  return table({ rows: many(7), pageSize: 3, selection: selection(keys, setKeys), ...extra });
}
