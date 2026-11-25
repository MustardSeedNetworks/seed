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

describe('Checkbox, target size and pass-through', () => {
  it('is a 24px box, the WCAG 2.5.8 minimum, and its label is part of the target', () => {
    render(<Checkbox checked={false} onChange={() => {}} label="A" />);
    const box = screen.getByRole('checkbox');
    expect(box.className).toContain('h-6');
    expect(box.className).toContain('w-6');
    expect(box.className).not.toMatch(/(^|\s)(h|w)-4(\s|$)/);
    expect(screen.getByText('A').className).toContain('min-h-6');
  });

  it('passes id, name, data-testid and aria-* to the input, and the label follows the id', () => {
    render(
      <Checkbox
        checked={false}
        onChange={() => {}}
        label="Include logs"
        id="logs"
        name="logs"
        data-testid="logs-box"
        aria-describedby="elsewhere"
        description="Adds 2 MB"
      />,
    );
    const box = screen.getByTestId('logs-box');
    expect(box).toBe(screen.getByLabelText('Include logs'));
    expect(box.id).toBe('logs');
    expect(box.getAttribute('name')).toBe('logs');
    expect(box.getAttribute('aria-describedby')).toBe('elsewhere logs-description');
  });

  it('takes a ReactNode label and a ref to the input', () => {
    let node: HTMLInputElement | null = null;
    render(
      <Checkbox
        checked
        onChange={() => {}}
        label={<span data-testid="rich">Rich</span>}
        ref={(element) => {
          node = element;
        }}
      />,
    );
    expect(screen.getByTestId('rich')).toBeTruthy();
    expect((node as HTMLInputElement | null)?.type).toBe('checkbox');
  });
});
