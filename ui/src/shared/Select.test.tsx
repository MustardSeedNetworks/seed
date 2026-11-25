import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { Select } from './Select';

afterEach(cleanup);

const options = [
  { value: 'en0', label: 'en0' },
  { value: 'en1', label: 'en1', disabled: true },
];

describe('Select', () => {
  it('is a native select with its options', () => {
    render(<Select aria-label="Interface" options={options} defaultValue="en0" />);
    const select = screen.getByRole('combobox', { name: 'Interface' }) as HTMLSelectElement;
    expect(select.tagName).toBe('SELECT');
    expect(screen.getAllByRole('option')).toHaveLength(2);
    expect((screen.getByRole('option', { name: 'en1' }) as HTMLOptionElement).disabled).toBe(true);
  });

  it('shows a disabled placeholder first when nothing is chosen', () => {
    render(
      <Select aria-label="Interface" options={options} placeholder="Choose" defaultValue="" />,
    );
    const first = screen.getAllByRole('option')[0] as HTMLOptionElement;
    expect(first.textContent).toBe('Choose');
    expect(first.disabled).toBe(true);
  });

  it('reports the chosen value through onValueChange', () => {
    const onChange = vi.fn();
    render(
      <Select
        aria-label="Interface"
        options={options}
        defaultValue=""
        placeholder="Choose"
        onValueChange={onChange}
      />,
    );
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'en0' } });
    expect(onChange).toHaveBeenCalledExactlyOnceWith('en0');
  });

  it('is themed from tokens, not left to the browser', () => {
    render(<Select aria-label="Interface" options={options} />);
    const { className } = screen.getByRole('combobox');
    expect(className).toContain('appearance-none');
    expect(className).toContain('bg-surface-raised');
    expect(className).toContain('border-surface-border');
    expect(className).toContain('min-w-0');
  });
});
