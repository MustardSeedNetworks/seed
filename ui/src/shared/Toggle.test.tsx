import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { Toggle } from './Toggle';

afterEach(cleanup);

describe('Toggle', () => {
  it('is a switch named by its label and described by its description', () => {
    render(
      <Toggle
        checked={false}
        onChange={() => {}}
        label="Auto-upgrade"
        description="Off by default"
      />,
    );
    const toggle = screen.getByRole('switch', { name: 'Auto-upgrade' });
    expect(toggle.getAttribute('aria-checked')).toBe('false');
    const describedBy = toggle.getAttribute('aria-describedby') ?? '';
    expect(document.getElementById(describedBy)?.textContent).toBe('Off by default');
  });

  it('asks for the opposite state on click, and on a click of its label', () => {
    const onChange = vi.fn();
    render(<Toggle checked={false} onChange={onChange} label="Dark" />);
    fireEvent.click(screen.getByRole('switch'));
    fireEvent.click(screen.getByText('Dark'));
    expect(onChange).toHaveBeenNthCalledWith(1, true);
    expect(onChange).toHaveBeenNthCalledWith(2, true);
  });

  it('asks to turn off when on', () => {
    const onChange = vi.fn();
    render(<Toggle checked onChange={onChange} label="Dark" />);
    expect(screen.getByRole('switch').getAttribute('aria-checked')).toBe('true');
    fireEvent.click(screen.getByRole('switch'));
    expect(onChange).toHaveBeenCalledWith(false);
  });

  it('draws a visible track in both states: a 3:1 border off, the brand fill on', () => {
    const { rerender } = render(<Toggle checked={false} onChange={() => {}} label="T" />);
    const off = screen.getByRole('switch').className;
    expect(off).toContain('border-surface-border');
    expect(off).toContain('bg-surface-sunken');
    expect(off).not.toContain('bg-brand-primary');

    rerender(<Toggle checked onChange={() => {}} label="T" />);
    const on = screen.getByRole('switch').className;
    expect(on).toContain('bg-brand-primary');
  });

  it('does not fire when disabled', () => {
    const onChange = vi.fn();
    render(<Toggle checked={false} onChange={onChange} label="T" disabled />);
    fireEvent.click(screen.getByRole('switch'));
    expect(onChange).not.toHaveBeenCalled();
  });
});

describe('Toggle, pass-through and target size', () => {
  it('passes id, name, data-testid and aria-* to the switch, and the label follows the id', () => {
    render(
      <Toggle
        checked={false}
        onChange={() => {}}
        label="Updates"
        id="updates"
        name="updates"
        data-testid="updates-switch"
        aria-describedby="elsewhere"
        description="Off by default"
      />,
    );
    const toggle = screen.getByTestId('updates-switch');
    expect(toggle).toBe(screen.getByRole('switch', { name: 'Updates' }));
    expect(toggle.id).toBe('updates');
    expect(toggle.getAttribute('name')).toBe('updates');
    expect(toggle.getAttribute('aria-describedby')).toBe('elsewhere updates-description');
  });

  it('is a 24px-tall track and takes a ReactNode label', () => {
    render(<Toggle checked onChange={() => {}} label={<b data-testid="rich">Rich</b>} />);
    expect(screen.getByRole('switch').className).toContain('h-6');
    expect(screen.getByTestId('rich')).toBeTruthy();
  });
});
