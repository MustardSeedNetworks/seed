/**
 * The archetype's own honest-state guarantee.
 *
 * RecordRow always paints a state bar, so the default matters: with `ok` as the
 * default, a caller that simply forgot to pass `state` would render a green
 * record — a claim of health that nothing measured. That failure is not
 * hypothetical; it is live in the fleet, where every topology node renders
 * green from a hardcoded literal (MustardSeedNetworks/niac-go#1352).
 *
 * Locking the default here rather than in each page is deliberate: the rule
 * belongs to the component, so no page can opt out of it by omission.
 */
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { RecordRow } from './ListDetail';

function barOf(testId: string): Element {
  const row = screen.getByTestId(testId);
  const bar = row.querySelector('[aria-hidden="true"]');
  if (!bar) throw new Error(`no state bar in ${testId}`);
  return bar;
}

describe('RecordRow state default', () => {
  it('renders unknown, not success, when the caller passes no state', () => {
    render(<RecordRow name="core-01" onSelect={vi.fn()} data-testid="row" />);

    const cls = barOf('row').className;
    expect(cls).toContain('bg-text-disabled');
    expect(cls).not.toContain('bg-status-success');
  });

  it('still renders success when a caller genuinely asserts ok', () => {
    render(<RecordRow name="core-01" state="ok" onSelect={vi.fn()} data-testid="row" />);

    expect(barOf('row').className).toContain('bg-status-success');
  });

  it('keeps warn and crit distinct so severity is not flattened', () => {
    render(
      <>
        <RecordRow name="a" state="warn" onSelect={vi.fn()} data-testid="warn" />
        <RecordRow name="b" state="crit" onSelect={vi.fn()} data-testid="crit" />
      </>,
    );

    expect(barOf('warn').className).toContain('bg-status-warning');
    expect(barOf('crit').className).toContain('bg-status-error');
  });

  it('does not colour the value figure as healthy by default either', () => {
    render(<RecordRow name="core-01" value="42" onSelect={vi.fn()} data-testid="row" />);

    const value = screen.getByText('42');
    expect(value.className).toContain('text-text-muted');
    expect(value.className).not.toContain('text-status-success');
  });
});
