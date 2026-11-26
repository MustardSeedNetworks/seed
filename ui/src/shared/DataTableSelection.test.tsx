import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { many, ordered, Selectable, SelectableTable, table } from './DataTableTestUtils';

afterEach(cleanup);

describe('DataTable selection', () => {
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
});

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
