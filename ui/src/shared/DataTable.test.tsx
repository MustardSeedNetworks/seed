import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { DataTable, type DataTableColumn, type DataTableLabels } from './DataTable';

afterEach(cleanup);

interface Device {
  id: string;
  name: string;
  ports: number;
}

const devices: Device[] = [
  { id: 'a', name: 'core', ports: 48 },
  { id: 'b', name: 'access', ports: 24 },
  { id: 'c', name: 'edge', ports: 8 },
];

const columns: DataTableColumn<Device>[] = [
  { key: 'name', header: 'Name', cell: (d) => d.name, sortValue: (d) => d.name },
  {
    key: 'ports',
    header: 'Ports',
    cell: (d) => d.ports,
    sortValue: (d) => d.ports,
    align: 'right',
  },
];

const labels: DataTableLabels<Device> = {
  selectAll: 'Select all',
  selectRow: (d) => `Select ${d.name}`,
  actions: 'Actions',
  loading: 'Loading devices',
  pagination: 'Pages',
  previousPage: 'Previous page',
  nextPage: 'Next page',
  pageStatus: (page, count, total) => `Page ${page} of ${count} (${total} rows)`,
};

function names(): string[] {
  return screen
    .getAllByRole('row')
    .slice(1)
    .map((row) => within(row).getAllByRole('cell')[0]?.textContent ?? '');
}

const table = (extra: Partial<React.ComponentProps<typeof DataTable<Device>>> = {}) => (
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

function SelectableTable() {
  const [selectedKeys, setSelected] = useState<ReadonlySet<string>>(new Set());
  return table({ selection: { selectedKeys, onChange: setSelected } });
}

describe('DataTable', () => {
  it('renders a named semantic table with column headers and a row per item', () => {
    render(table());
    expect(screen.getByRole('table', { name: 'Devices' })).toBeTruthy();
    expect(screen.getAllByRole('columnheader').map((h) => h.textContent)).toEqual([
      'Name',
      'Ports',
    ]);
    expect(names()).toEqual(['core', 'access', 'edge']);
  });

  it('shows the empty node instead of a table when there are no rows', () => {
    render(table({ rows: [] }));
    expect(screen.getByText('none')).toBeTruthy();
    expect(screen.queryByRole('table')).toBeNull();
  });

  it('cycles a column ascending, descending, then off, with aria-sort tracking it', () => {
    render(table());
    const header = screen.getByRole('columnheader', { name: /Name/ });
    const sortButton = within(header).getByRole('button');
    expect(header.getAttribute('aria-sort')).toBe('none');

    fireEvent.click(sortButton);
    expect(header.getAttribute('aria-sort')).toBe('ascending');
    expect(names()).toEqual(['access', 'core', 'edge']);

    fireEvent.click(sortButton);
    expect(header.getAttribute('aria-sort')).toBe('descending');
    expect(names()).toEqual(['edge', 'core', 'access']);

    fireEvent.click(sortButton);
    expect(header.getAttribute('aria-sort')).toBe('none');
    expect(names()).toEqual(['core', 'access', 'edge']);
  });

  it('sorts numbers numerically and applies a default sort', () => {
    render(table({ defaultSort: { key: 'ports', direction: 'asc' } }));
    expect(names()).toEqual(['edge', 'access', 'core']);
  });

  it('exposes a sortable header as a focusable native button, so Enter and Space sort it', () => {
    render(table());
    const sortButton = within(screen.getByRole('columnheader', { name: /Ports/ })).getByRole(
      'button',
    );
    sortButton.focus();
    expect(document.activeElement).toBe(sortButton);
    expect(sortButton.tagName).toBe('BUTTON');
  });

  it('leaves a column without a sortValue as plain text with no aria-sort', () => {
    render(table({ columns: [{ key: 'name', header: 'Name', cell: (d) => d.name }] }));
    const header = screen.getByRole('columnheader', { name: 'Name' });
    expect(within(header).queryByRole('button')).toBeNull();
    expect(header.getAttribute('aria-sort')).toBeNull();
  });

  it('paginates, hides the pager for one page, and clamps after a sort', () => {
    const many = Array.from({ length: 7 }, (_, i) => ({ id: `${i}`, name: `d${i}`, ports: i }));
    render(table({ rows: many, pageSize: 3 }));
    expect(names()).toEqual(['d0', 'd1', 'd2']);
    expect(screen.getByText('Page 1 of 3 (7 rows)')).toBeTruthy();
    expect(
      (screen.getByRole('button', { name: 'Previous page' }) as HTMLButtonElement).disabled,
    ).toBe(true);

    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    expect(names()).toEqual(['d6']);
    expect((screen.getByRole('button', { name: 'Next page' }) as HTMLButtonElement).disabled).toBe(
      true,
    );

    fireEvent.click(within(screen.getByRole('columnheader', { name: /Name/ })).getByRole('button'));
    expect(screen.getByText('Page 1 of 3 (7 rows)')).toBeTruthy();
  });

  it('shows no pager when everything fits on one page', () => {
    render(table());
    expect(screen.queryByRole('navigation', { name: 'Pages' })).toBeNull();
  });

  it('selects rows and selects all on the page, with an indeterminate header', () => {
    render(<SelectableTable />);
    const all = screen.getByRole('checkbox', { name: 'Select all' }) as HTMLInputElement;

    fireEvent.click(screen.getByRole('checkbox', { name: 'Select core' }));
    expect(all.indeterminate).toBe(true);
    expect(all.checked).toBe(false);

    fireEvent.click(all);
    expect(all.checked).toBe(true);
    expect(all.indeterminate).toBe(false);
    expect(screen.getAllByRole('checkbox', { checked: true })).toHaveLength(4);

    fireEvent.click(all);
    expect(screen.queryAllByRole('checkbox', { checked: true })).toHaveLength(0);
  });

  it('renders a row-actions cell with a screen-reader header', () => {
    render(table({ rowActions: (d) => <button type="button">{`Edit ${d.name}`}</button> }));
    expect(screen.getByRole('columnheader', { name: 'Actions' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Edit core' })).toBeTruthy();
  });

  it('contains horizontal overflow inside the table frame at 390px', () => {
    render(table({ 'data-testid': 'dt' }));
    const frame = screen.getByTestId('dt');
    expect(frame.className).toContain('panel');
    expect(frame.className).toContain('min-w-0');
    expect(frame.className).toContain('overflow-hidden');
    const scroller = screen.getByRole('table').parentElement;
    expect(scroller?.className).toContain('overflow-x-auto');
    expect(scroller?.getAttribute('data-phone-width-exempt')).toBe('table-scroll');
  });

  it('holds a 44px row and a 40px header', () => {
    render(table());
    const [header, firstRow] = screen.getAllByRole('row');
    expect(header?.className).toContain('h-10');
    expect(firstRow?.className).toContain('h-11');
  });

  describe('row click', () => {
    it('activates a row on click and on Enter or Space when the row itself has focus', () => {
      const onRowClick = vi.fn();
      render(table({ onRowClick }));
      const row = screen.getByRole('row', { name: /core/ });
      expect(row.getAttribute('tabindex')).toBe('0');
      fireEvent.click(row);
      fireEvent.keyDown(row, { key: 'Enter' });
      fireEvent.keyDown(row, { key: ' ' });
      fireEvent.keyDown(row, { key: 'a' });
      expect(onRowClick).toHaveBeenCalledTimes(3);
      expect(onRowClick).toHaveBeenCalledWith(devices[0]);
    });

    it('leaves rows out of the tab order without onRowClick', () => {
      render(table());
      expect(screen.getByRole('row', { name: /core/ }).getAttribute('tabindex')).toBeNull();
    });

    it('does not fire from a row action or a checkbox inside the row', () => {
      const onRowClick = vi.fn();
      render(
        table({
          onRowClick,
          selection: { selectedKeys: new Set(), onChange: () => {} },
          rowActions: (d) => <button type="button">{`Edit ${d.name}`}</button>,
        }),
      );
      fireEvent.click(screen.getByRole('button', { name: 'Edit core' }));
      fireEvent.click(screen.getByRole('checkbox', { name: 'Select core' }));
      fireEvent.keyDown(screen.getByRole('button', { name: 'Edit core' }), { key: 'Enter' });
      expect(onRowClick).not.toHaveBeenCalled();
    });

    it('marks only the selected row aria-selected', () => {
      render(table({ onRowClick: () => {}, selectedRowKey: 'b' }));
      const flags = screen
        .getAllByRole('row')
        .slice(1)
        .map((r) => r.getAttribute('aria-selected'));
      expect(flags).toEqual(['false', 'true', 'false']);
    });
  });

  describe('loading', () => {
    it('shows skeleton rows under the real header, busy, with a status line', () => {
      render(table({ loading: true }));
      expect(screen.getAllByTestId('data-table-skeleton-row')).toHaveLength(5);
      expect(screen.getAllByRole('columnheader').map((h) => h.textContent)).toEqual([
        'Name',
        'Ports',
      ]);
      expect(screen.getByRole('table').getAttribute('aria-busy')).toBe('true');
      expect(screen.getByRole('status').textContent).toBe('Loading devices');
      expect(screen.queryByText('core')).toBeNull();
    });

    it('shows skeletons rather than the empty state while loading with no rows yet', () => {
      render(table({ rows: [], loading: true }));
      expect(screen.queryByText('none')).toBeNull();
      expect(screen.getAllByTestId('data-table-skeleton-row').length).toBeGreaterThan(0);
    });

    it('hides the pager while loading', () => {
      const many = Array.from({ length: 7 }, (_, i) => ({ id: `${i}`, name: `d${i}`, ports: i }));
      render(table({ rows: many, pageSize: 3, loading: true }));
      expect(screen.queryByRole('navigation', { name: 'Pages' })).toBeNull();
    });
  });

  describe('sticky header', () => {
    it('pins the header and bounds the scroller when maxHeight is set', () => {
      render(table({ maxHeight: 300 }));
      const scroller = screen.getByRole('table').parentElement;
      expect(scroller?.style.maxHeight).toBe('300px');
      expect(scroller?.className).toContain('overflow-y-auto');
      expect(screen.getAllByRole('rowgroup')[0]?.className).toContain('sticky');
    });

    it('does not pin or bound anything by default', () => {
      render(table());
      expect(screen.getAllByRole('rowgroup')[0]?.className).not.toContain('sticky');
      expect(screen.getByRole('table').parentElement?.style.maxHeight).toBe('');
    });
  });

  describe('virtualization', () => {
    const big = Array.from({ length: 500 }, (_, i) => ({ id: `${i}`, name: `d${i}`, ports: i }));
    const dataRows = () =>
      screen
        .getAllByRole('row')
        .filter((r) => r.hasAttribute('aria-rowindex') && r.getAttribute('aria-rowindex') !== '1');

    it('mounts only a window of rows and no pager', () => {
      render(table({ rows: big, virtualize: {}, maxHeight: 440 }));
      expect(dataRows().length).toBeLessThan(30);
      expect(screen.getByRole('table').getAttribute('aria-rowcount')).toBe('501');
      expect(screen.queryByRole('navigation', { name: 'Pages' })).toBeNull();
      expect(screen.getByText('d0')).toBeTruthy();
      expect(screen.queryByText('d400')).toBeNull();
    });

    it('slides the window with the scroll offset and pads the rest with spacers', () => {
      render(table({ rows: big, virtualize: {}, maxHeight: 440 }));
      const scroller = screen.getByRole('table').parentElement as HTMLElement;
      scroller.scrollTop = 44 * 300;
      fireEvent.scroll(scroller);
      expect(screen.queryByText('d0')).toBeNull();
      expect(screen.getByText('d300')).toBeTruthy();
      const [first] = dataRows();
      expect(first?.getAttribute('aria-rowindex')).toBe(String(295 + 2));
      const spacers = screen.getByRole('table').querySelectorAll('td[aria-hidden="true"]');
      expect(spacers).toHaveLength(2);
    });

    it('stays a plain paged table below the threshold', () => {
      render(table({ virtualize: { threshold: 100 }, pageSize: 2 }));
      expect(screen.getByRole('navigation', { name: 'Pages' })).toBeTruthy();
      expect(screen.getByRole('table').getAttribute('aria-rowcount')).toBeNull();
    });
  });
});
