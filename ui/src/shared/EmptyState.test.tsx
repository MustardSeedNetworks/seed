import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { EmptyState } from './EmptyState';

afterEach(cleanup);

describe('EmptyState', () => {
  it('renders the title, hint and one action', () => {
    render(
      <EmptyState
        icon={<svg data-testid="icon" />}
        title="No devices yet"
        hint="Add a device to start."
        action={<button type="button">Add device</button>}
      />,
    );
    expect(screen.getByText('No devices yet')).toBeTruthy();
    expect(screen.getByText('Add a device to start.')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Add device' })).toBeTruthy();
  });

  it('hides the decorative icon from assistive technology', () => {
    render(<EmptyState icon={<svg data-testid="icon" />} title="Nothing" />);
    expect(screen.getByTestId('icon').parentElement?.getAttribute('aria-hidden')).toBe('true');
  });

  it('omits the hint, icon and action when not given', () => {
    render(<EmptyState title="Nothing" />);
    expect(screen.getByTestId('empty-state').querySelectorAll('p')).toHaveLength(1);
    expect(screen.queryByRole('button')).toBeNull();
  });

  it('is neutral: no status colour, no frame of its own', () => {
    render(<EmptyState title="Nothing" hint="Why" />);
    const { className } = screen.getByTestId('empty-state');
    expect(className).not.toMatch(/status-|border|bg-/);
    expect(screen.getByTestId('empty-state').innerHTML).not.toMatch(/status-/);
  });
});
