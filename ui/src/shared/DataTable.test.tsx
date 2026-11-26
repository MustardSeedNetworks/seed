import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  type Device,
  devices,
  type Extra,
  many,
  names,
  ordered,
  table,
} from './DataTableTestUtils';

afterEach(cleanup);

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

  it('renders a row-actions cell with a screen-reader header', () => {
    render(table({ rowActions: (d) => <button type="button">{`Edit ${d.name}`}</button> }));
    expect(screen.getByRole('columnheader', { name: 'Actions' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Edit core' })).toBeTruthy();
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
      const seven = many(7);
      render(table({ rows: seven, pageSize: 3, loading: true }));
      expect(screen.queryByRole('navigation', { name: 'Pages' })).toBeNull();
    });
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
