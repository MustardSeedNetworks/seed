import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { DataTableColumn } from './DataTable';
import { columns, type Device, names, table } from './test/DataTableTestUtils';

afterEach(cleanup);

describe('DataTable sorting', () => {
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
