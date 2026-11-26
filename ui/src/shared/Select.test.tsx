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

describe('Select pass-through', () => {
  it('passes id, name, data-testid and aria-* to the <select>, and ref to it', () => {
    let node: HTMLSelectElement | null = null;
    render(
      <Select
        options={[{ value: 'a', label: 'A' }]}
        id="s"
        name="iface"
        data-testid="s-select"
        aria-label="Interface"
        ref={(element) => {
          node = element;
        }}
      />,
    );
    const select = screen.getByTestId('s-select');
    expect(select.id).toBe('s');
    expect(select.getAttribute('name')).toBe('iface');
    expect(node).toBe(select);
  });

  it('also passes the native onChange through, for register(), alongside onValueChange', () => {
    const onChange = vi.fn();
    const onValueChange = vi.fn();
    render(
      <Select
        aria-label="Interface"
        options={[
          { value: 'a', label: 'A' },
          { value: 'b', label: 'B' },
        ]}
        onChange={onChange}
        onValueChange={onValueChange}
      />,
    );
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'b' } });
    expect(onChange).toHaveBeenCalledOnce();
    expect(onChange.mock.calls[0]?.[0].target.value).toBe('b');
    expect(onValueChange).toHaveBeenCalledExactlyOnceWith('b');
  });
});

describe('Select option children', () => {
  it('renders JSX options, with groups, and reports the value', () => {
    const onValueChange = vi.fn();
    render(
      <Select aria-label="Pick" onValueChange={onValueChange} defaultValue="b">
        <optgroup label="Letters">
          <option value="a">A</option>
          <option value="b">B</option>
        </optgroup>
      </Select>,
    );
    const select = screen.getByRole('combobox', { name: 'Pick' }) as HTMLSelectElement;
    expect(screen.getAllByRole('option').map((option) => option.textContent)).toEqual(['A', 'B']);
    expect(select.value).toBe('b');
    fireEvent.change(select, { target: { value: 'a' } });
    expect(onValueChange).toHaveBeenCalledWith('a');
  });

  it('renders the options array first, then the children', () => {
    render(
      <Select aria-label="Pick" options={[{ value: 'x', label: 'From array' }]}>
        {['one', 'two'].map((value) => (
          <option key={value} value={value}>
            {value}
          </option>
        ))}
      </Select>,
    );
    expect(screen.getAllByRole('option').map((option) => option.textContent)).toEqual([
      'From array',
      'one',
      'two',
    ]);
  });

  it('needs neither options nor children to render', () => {
    render(<Select aria-label="Empty" />);
    expect(screen.queryAllByRole('option')).toHaveLength(0);
  });
});
