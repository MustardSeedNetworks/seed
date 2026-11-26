import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { SearchInput } from './SearchInput';

afterEach(cleanup);

describe('SearchInput', () => {
  it('is a search box named by its label', () => {
    render(
      <SearchInput
        value=""
        onValueChange={() => {}}
        label="Search devices"
        clearLabel="Clear search"
      />,
    );
    const box = screen.getByRole('searchbox', { name: 'Search devices' });
    expect(box.getAttribute('type')).toBe('search');
  });

  it('reports typed text through onValueChange', () => {
    const onValueChange = vi.fn();
    render(
      <SearchInput value="" onValueChange={onValueChange} label="Search" clearLabel="Clear" />,
    );
    fireEvent.change(screen.getByRole('searchbox'), { target: { value: 'core' } });
    expect(onValueChange).toHaveBeenCalledExactlyOnceWith('core');
  });

  it('offers a clear button only when there is text, and clears to empty', () => {
    const onValueChange = vi.fn();
    const { rerender } = render(
      <SearchInput value="" onValueChange={onValueChange} label="Search" clearLabel="Clear" />,
    );
    expect(screen.queryByRole('button', { name: 'Clear' })).toBeNull();
    rerender(
      <SearchInput value="core" onValueChange={onValueChange} label="Search" clearLabel="Clear" />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Clear' }));
    expect(onValueChange).toHaveBeenCalledExactlyOnceWith('');
  });

  it('is width-capped and shrinkable', () => {
    render(<SearchInput value="" onValueChange={() => {}} label="Search" clearLabel="Clear" />);
    const frame = screen.getByRole('searchbox').parentElement;
    expect(frame?.className).toContain('max-w-md');
    expect(frame?.className).toContain('w-full');
  });
});
