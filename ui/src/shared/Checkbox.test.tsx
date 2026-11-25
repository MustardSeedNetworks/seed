import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { Checkbox } from './Checkbox';

afterEach(cleanup);

describe('Checkbox', () => {
  it('is a native checkbox named by its label and described by its description', () => {
    render(
      <Checkbox checked={false} onChange={() => {}} label="Include logs" description="Adds 2 MB" />,
    );
    const box = screen.getByRole('checkbox', { name: 'Include logs' });
    expect(document.getElementById(box.getAttribute('aria-describedby') ?? '')?.textContent).toBe(
      'Adds 2 MB',
    );
  });

  it('reports the new boolean, from the box and from its label', () => {
    const onChange = vi.fn();
    render(<Checkbox checked={false} onChange={onChange} label="Include logs" />);
    fireEvent.click(screen.getByRole('checkbox'));
    fireEvent.click(screen.getByText('Include logs'));
    expect(onChange).toHaveBeenCalledTimes(2);
    expect(onChange).toHaveBeenCalledWith(true);
  });

  it('shows its state, and is disabled when asked', () => {
    render(<Checkbox checked onChange={() => {}} label="A" disabled />);
    const box = screen.getByRole('checkbox') as HTMLInputElement;
    expect(box.checked).toBe(true);
    expect(box.disabled).toBe(true);
  });
});
