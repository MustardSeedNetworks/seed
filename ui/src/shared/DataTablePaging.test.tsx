import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { type Device, many, names, table } from './test/DataTableTestUtils';

afterEach(cleanup);

describe('DataTable paging', () => {
  it('paginates, hides the pager for one page, and clamps after a sort', () => {
    const seven = many(7);
    render(table({ rows: seven, pageSize: 3 }));
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
