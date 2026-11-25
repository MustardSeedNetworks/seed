import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { type ReactNode, useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  DataTable,
  type DataTableColumn,
  type DataTableLabels,
  type DataTableProps,
  type DataTableSelection,
} from './DataTable';

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
  selectRow: (d) => `Select ${d.name}`,
  selectAll: 'Select all',
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

type Extra = Partial<DataTableProps<Device>>;

const table = (extra: Extra = {}) => (
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
  return table({
    selection: { selectedKeys, onChange: setSelected },
  });
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

const ordered = (keys: Iterable<string>) => [...keys].sort((a, b) => a.localeCompare(b));
const frameOf = () => screen.getByRole('table').parentElement as HTMLElement;
const many = (count: number): Device[] =>
  Array.from({ length: count }, (_, i) => ({ id: `${i}`, name: `d${i}`, ports: i }));

describe('DataTable containment and keyboard access', () => {
  it('makes the scroll frame a containing block, so sr-only text cannot widen the page', () => {
    render(table({ rowActions: (d) => <button type="button">{`Edit ${d.name}`}</button> }));
    expect(frameOf().className).toContain('relative');
    expect(frameOf().className).toContain('overflow-x-auto');
    const actionsHeader = screen.getByRole('columnheader', { name: 'Actions' });
    expect(actionsHeader.className).toContain('relative');
    expect(actionsHeader.querySelector('.sr-only')).toBeTruthy();
    expect(screen.getByRole('table').querySelector('caption')?.className).toContain('sr-only');
  });

  describe('when the frame scrolls', () => {
    function stubOverflow() {
      vi.spyOn(HTMLElement.prototype, 'scrollWidth', 'get').mockReturnValue(900);
      vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(358);
    }

    it('is a focusable, labelled group', () => {
      stubOverflow();
      render(table());
      const frame = screen.getByRole('group', { name: 'Devices' });
      expect(frame).toBe(frameOf());
      expect(frame.getAttribute('tabindex')).toBe('0');
      expect(frame.className).toContain('focus-visible:outline');
      vi.restoreAllMocks();
    });

    it('is focusable even when it also holds buttons', () => {
      stubOverflow();
      render(table({ rowActions: () => <button type="button">Edit</button> }));
      expect(frameOf().getAttribute('tabindex')).toBe('0');
      vi.restoreAllMocks();
    });
  });

  it('leaves the frame out of the tab order and unlabelled while nothing overflows', () => {
    render(table());
    expect(frameOf().hasAttribute('tabindex')).toBe(false);
    expect(screen.queryByRole('group')).toBeNull();
  });

  it('keeps sort buttons, checkboxes and expand toggles at least 24px', () => {
    render(
      table({
        selection: { selectedKeys: new Set(), onChange: () => {} },
        expandable: {
          render: () => 'x',
          columnLabel: 'Details',
          toggleLabel: (d) => `Show ${d.name}`,
        },
      }),
    );
    for (const button of within(screen.getByRole('columnheader', { name: /Name/ })).getAllByRole(
      'button',
    )) {
      expect(button.className).toContain('min-h-6');
      expect(button.className).toContain('min-w-6');
    }
    for (const box of screen.getAllByRole('checkbox')) {
      expect(box.className).toContain('h-6');
      expect(box.className).toContain('w-6');
    }
    expect(screen.getByRole('button', { name: 'Show core' }).className).toContain('h-6');
  });
});

function Selectable({
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

describe('DataTable select-all scope', () => {
  it('page scope covers only the rows on the current page', () => {
    render(<Selectable selection={(selectedKeys, onChange) => ({ selectedKeys, onChange })} />);
    fireEvent.click(screen.getByRole('checkbox', { name: 'Select all' }));
    expect(screen.getAllByRole('checkbox', { checked: true })).toHaveLength(4);
  });

  it('all scope selects every row across pages, and names itself with the count', () => {
    render(
      <Selectable
        selection={(selectedKeys, onChange) => ({
          selectedKeys,
          onChange,
          scope: 'all',
          selectAllLabel: (total) => `Select all ${total} devices`,
          statusLabel: (selected, total) => `${selected} of ${total} selected`,
        })}
      />,
    );
    const all = screen.getByRole('checkbox', { name: 'Select all 7 devices' }) as HTMLInputElement;
    fireEvent.click(all);
    expect(all.checked).toBe(true);
    expect(screen.getByTestId('data-table-selection-status').textContent).toBe('7 of 7 selected');
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    expect(screen.getAllByRole('checkbox', { checked: true }).length).toBe(4);
    fireEvent.click(all);
    expect(screen.queryByTestId('data-table-selection-status')).toBeNull();
  });

  it('all scope is indeterminate when only some rows are selected, even off this page', () => {
    render(
      <Selectable
        selection={(selectedKeys, onChange) => ({
          selectedKeys,
          onChange,
          scope: 'all',
          selectAllLabel: (total) => `Select all ${total}`,
        })}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    fireEvent.click(screen.getByRole('checkbox', { name: 'Select d3' }));
    fireEvent.click(screen.getByRole('button', { name: 'Previous page' }));
    const all = screen.getByRole('checkbox', { name: 'Select all 7' }) as HTMLInputElement;
    expect(all.indeterminate).toBe(true);
  });

  it('all scope covers only the rows that match the search', () => {
    const onChange = vi.fn();
    render(
      table({
        rows: many(30),
        search: {
          label: 'Search',
          clearLabel: 'Clear',
          text: (d) => d.name,
          noResults: <p>nothing</p>,
        },
        selection: {
          selectedKeys: new Set(),
          onChange,
          scope: 'all',
          selectAllLabel: (total) => `Select all ${total}`,
        },
      }),
    );
    fireEvent.change(screen.getByRole('searchbox', { name: 'Search' }), {
      target: { value: 'd2' },
    });
    fireEvent.click(screen.getByRole('checkbox', { name: 'Select all 11' }));
    const keys = onChange.mock.calls[0]?.[0] as ReadonlySet<string>;
    expect(ordered(keys)).toEqual(
      ordered(['2', '20', '21', '22', '23', '24', '25', '26', '27', '28', '29']),
    );
  });

  it('keeps selecting across a virtual window when the scope is all', () => {
    const onChange = vi.fn();
    render(
      table({
        rows: many(500),
        virtualize: {},
        maxHeight: 440,
        selection: {
          selectedKeys: new Set(),
          onChange,
          scope: 'all',
          selectAllLabel: (total) => `Select all ${total}`,
        },
      }),
    );
    fireEvent.click(screen.getByRole('checkbox', { name: 'Select all 500' }));
    const chosen: ReadonlySet<string> = onChange.mock.calls[0]?.[0] ?? new Set();
    expect(chosen.size).toBe(500);
  });
});

describe('DataTable row hook', () => {
  it('puts a test id, class, style and tone on the row from getRowProps', () => {
    render(
      table({
        getRowProps: (d) => ({
          'data-testid': `row-${d.id}`,
          className: d.id === 'a' ? 'font-bold' : 'italic',
          style: { backgroundColor: d.id === 'a' ? 'rgb(255, 0, 0)' : '' },
          ...(d.id === 'b' ? { tone: 'error' as const } : {}),
        }),
      }),
    );
    const a = screen.getByTestId('row-a');
    expect(a.className).toContain('font-bold');
    expect(a.style.backgroundColor).toBe('rgb(255, 0, 0)');
    const b = screen.getByTestId('row-b');
    expect(b.className).toContain('bg-status-error/10');
    expect(b.className).toContain('italic');
    expect(screen.getByTestId('row-c').className).not.toContain('bg-status');
  });

  it('keeps the base row classes alongside the hook', () => {
    render(table({ getRowProps: () => ({ className: 'x' }) }));
    expect(screen.getAllByRole('row')[1]?.className).toContain('h-11');
  });
});

describe('DataTable sorting options', () => {
  const named = (values: string[]): Device[] =>
    values.map((name, i) => ({ id: `${i}`, name, ports: i }));

  it('compares text in locale order by default and natural order on request', () => {
    const rows = named(['eth10', 'eth2', 'eth1']);
    const { unmount } = render(table({ rows, defaultSort: { key: 'name', direction: 'asc' } }));
    expect(names()).toEqual(['eth1', 'eth10', 'eth2']);
    unmount();
    render(
      table({
        rows,
        columns: [{ ...(columns[0] as DataTableColumn<Device>), collation: 'natural' }],
        defaultSort: { key: 'name', direction: 'asc' },
      }),
    );
    expect(names()).toEqual(['eth1', 'eth2', 'eth10']);
  });

  it('sorts addresses in natural order', () => {
    render(
      table({
        rows: named(['10.0.0.10', '10.0.0.2', '9.0.0.1']),
        columns: [{ ...(columns[0] as DataTableColumn<Device>), collation: 'natural' }],
        defaultSort: { key: 'name', direction: 'desc' },
      }),
    );
    expect(names()).toEqual(['10.0.0.10', '10.0.0.2', '9.0.0.1']);
  });

  it('sorts missing values last in both directions', () => {
    const rows = [
      { id: 'a', name: 'b', ports: 2 },
      { id: 'b', name: 'x', ports: 0 },
      { id: 'c', name: 'a', ports: 1 },
    ];
    const optional: DataTableColumn<Device>[] = [
      {
        key: 'name',
        header: 'Name',
        cell: (d) => d.name,
        sortValue: (d) => (d.name === 'x' ? undefined : d.name),
      },
    ];
    const { unmount } = render(
      table({ rows, columns: optional, defaultSort: { key: 'name', direction: 'asc' } }),
    );
    expect(names()).toEqual(['a', 'b', 'x']);
    unmount();
    render(table({ rows, columns: optional, defaultSort: { key: 'name', direction: 'desc' } }));
    expect(names()).toEqual(['b', 'a', 'x']);
  });

  it('can cycle ascending and descending only', () => {
    render(table({ sortCycle: 'asc-desc' }));
    const header = screen.getByRole('columnheader', { name: /Name/ });
    const button = within(header).getByRole('button');
    fireEvent.click(button);
    fireEvent.click(button);
    expect(header.getAttribute('aria-sort')).toBe('descending');
    fireEvent.click(button);
    expect(header.getAttribute('aria-sort')).toBe('ascending');
    expect(names()).toEqual(['access', 'core', 'edge']);
  });

  it('reports each sort state', () => {
    const onSortChange = vi.fn();
    render(table({ onSortChange }));
    const button = within(screen.getByRole('columnheader', { name: /Name/ })).getByRole('button');
    fireEvent.click(button);
    fireEvent.click(button);
    fireEvent.click(button);
    expect(onSortChange.mock.calls).toEqual([
      [{ key: 'name', direction: 'asc' }],
      [{ key: 'name', direction: 'desc' }],
      [null],
    ]);
  });
});

describe('DataTable search and toolbar', () => {
  const search = {
    label: 'Search devices',
    clearLabel: 'Clear search',
    text: (d: Device) => d.name,
    noResults: <p>No device matches</p>,
  };

  it('filters rows by the text, case-insensitively, before sorting and paging', () => {
    render(table({ search, rows: many(40), pageSize: 5 }));
    expect(screen.getByText('Page 1 of 8 (40 rows)')).toBeTruthy();
    fireEvent.change(screen.getByRole('searchbox', { name: 'Search devices' }), {
      target: { value: ' D3 ' },
    });
    expect(names()).toEqual(['d3', 'd30', 'd31', 'd32', 'd33']);
    expect(screen.getByText('Page 1 of 3 (11 rows)')).toBeTruthy();
  });

  it('goes back to the first page when the query changes', () => {
    render(table({ search, rows: many(40), pageSize: 5 }));
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }));
    fireEvent.change(screen.getByRole('searchbox'), { target: { value: 'd' } });
    expect(screen.getByText(/Page 1 of 8/)).toBeTruthy();
  });

  it('shows noResults in place of the table, keeps the search box, and recovers on clear', () => {
    render(table({ search }));
    fireEvent.change(screen.getByRole('searchbox'), { target: { value: 'zzz' } });
    expect(screen.getByText('No device matches')).toBeTruthy();
    expect(screen.queryByRole('table')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Clear search' }));
    expect(screen.getByRole('table')).toBeTruthy();
  });

  it('renders a toolbar slot with or without search, and none otherwise', () => {
    const { unmount } = render(table({ toolbar: <button type="button">Filter</button> }));
    expect(screen.getByRole('button', { name: 'Filter' })).toBeTruthy();
    expect(screen.queryByRole('searchbox')).toBeNull();
    unmount();
    render(table());
    expect(screen.queryByRole('searchbox')).toBeNull();
  });

  it('shows empty rather than a search box when there are no rows at all', () => {
    render(table({ rows: [], search }));
    expect(screen.getByText('none')).toBeTruthy();
    expect(screen.queryByRole('searchbox')).toBeNull();
  });
});

describe('DataTable row detail', () => {
  const expandable = (extra: Partial<NonNullable<Extra['expandable']>> = {}): Extra => ({
    expandable: {
      render: (d: Device) => <p>{`detail of ${d.name}`}</p>,
      columnLabel: 'Details',
      toggleLabel: (d: Device, open: boolean) => `${open ? 'Hide' : 'Show'} ${d.name}`,
      ...extra,
    },
  });

  it('adds a toggle column and shows the detail full width under an expanded row', () => {
    render(table(expandable()));
    expect(screen.getByRole('columnheader', { name: 'Details' })).toBeTruthy();
    const toggle = screen.getByRole('button', { name: 'Show core' });
    expect(toggle.getAttribute('aria-expanded')).toBe('false');
    expect(screen.queryByText('detail of core')).toBeNull();
    fireEvent.click(toggle);
    expect(screen.getByRole('button', { name: 'Hide core' }).getAttribute('aria-expanded')).toBe(
      'true',
    );
    const detail = screen.getByText('detail of core').closest('td');
    expect(detail?.getAttribute('colspan')).toBe('3');
    expect(detail?.parentElement?.id).toBe(
      screen.getByRole('button', { name: 'Hide core' }).getAttribute('aria-controls'),
    );
    fireEvent.click(screen.getByRole('button', { name: 'Hide core' }));
    expect(screen.queryByText('detail of core')).toBeNull();
  });

  it('does not activate the row when the toggle is clicked', () => {
    const onRowClick = vi.fn();
    render(table({ ...expandable(), onRowClick }));
    fireEvent.click(screen.getByRole('button', { name: 'Show core' }));
    expect(onRowClick).not.toHaveBeenCalled();
  });

  it('starts expanded from defaultExpandedKeys and reports changes', () => {
    const onExpandedChange = vi.fn();
    render(table(expandable({ defaultExpandedKeys: new Set(['b']), onExpandedChange })));
    expect(screen.getByText('detail of access')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Show core' }));
    expect(ordered(onExpandedChange.mock.calls[0]?.[0] as ReadonlySet<string>)).toEqual(['a', 'b']);
  });

  it('is controlled when expandedKeys is passed', () => {
    const onExpandedChange = vi.fn();
    render(table(expandable({ expandedKeys: new Set(['c']), onExpandedChange })));
    expect(screen.getByText('detail of edge')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Show core' }));
    expect(screen.queryByText('detail of core')).toBeNull();
    expect(onExpandedChange).toHaveBeenCalledOnce();
  });

  it('is refused together with virtualize, by type', () => {
    // @ts-expect-error row detail cannot share a table with fixed-height virtual rows
    const both: Extra = { virtualize: {}, ...expandable() };
    expect(both).toBeTruthy();
  });
});

describe('DataTable fits a ReactNode empty state', () => {
  it('renders any node as the empty state', () => {
    const node: ReactNode = <p>nothing here</p>;
    render(table({ rows: [], empty: node }));
    expect(screen.getByText('nothing here')).toBeTruthy();
  });
});
