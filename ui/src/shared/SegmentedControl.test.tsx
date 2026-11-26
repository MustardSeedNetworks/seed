import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { SegmentedControl } from './SegmentedControl';

afterEach(cleanup);

const options = [
  { value: 'mine', label: 'My configs' },
  { value: 'upload', label: 'Upload', icon: <svg data-testid="icon" /> },
];

describe('SegmentedControl', () => {
  it('is a named radio group with the chosen option checked', () => {
    render(<SegmentedControl options={options} value="mine" onChange={() => {}} label="Source" />);
    expect(screen.getByRole('group', { name: 'Source' })).toBeTruthy();
    expect((screen.getByRole('radio', { name: 'My configs' }) as HTMLInputElement).checked).toBe(
      true,
    );
    expect((screen.getByRole('radio', { name: 'Upload' }) as HTMLInputElement).checked).toBe(false);
  });

  it('reports the chosen value', () => {
    const onChange = vi.fn();
    render(<SegmentedControl options={options} value="mine" onChange={onChange} label="Source" />);
    fireEvent.click(screen.getByRole('radio', { name: 'Upload' }));
    expect(onChange).toHaveBeenCalledExactlyOnceWith('upload');
  });

  it('shares one name across the radios so the platform gives arrow-key movement', () => {
    render(<SegmentedControl options={options} value="mine" onChange={() => {}} label="Source" />);
    const names = screen.getAllByRole('radio').map((radio) => radio.getAttribute('name'));
    expect(new Set(names).size).toBe(1);
    expect(names[0]).toBeTruthy();
  });

  it('hides icons and truncates labels so it fits at 390px', () => {
    render(<SegmentedControl options={options} value="mine" onChange={() => {}} label="Source" />);
    expect(screen.getByTestId('icon').parentElement?.getAttribute('aria-hidden')).toBe('true');
    expect(screen.getByText('My configs').className).toContain('truncate');
    expect(screen.getByRole('group').className).toContain('max-w-full');
  });
});
