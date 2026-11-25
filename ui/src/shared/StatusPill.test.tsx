import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { CategoryChip } from './CategoryChip';
import { StatusPill, type StatusTone } from './StatusPill';

afterEach(cleanup);

describe('StatusPill', () => {
  it.each<[StatusTone, string]>([
    ['success', 'status-success'],
    ['warning', 'status-warning'],
    ['error', 'status-error'],
    ['info', 'status-info'],
  ])('%s takes its hue wash and the -strong text token', (tone, hue) => {
    render(<StatusPill tone={tone} label="Label" />);
    const { className } = screen.getByText('Label');
    expect(className).toContain(`bg-${hue}/10`);
    expect(className).toContain(`text-${hue}-strong`);
    expect(className).not.toMatch(new RegExp(`text-${hue}(?!-strong)`));
  });

  it('draws unknown in neutral tokens, not a status hue', () => {
    render(<StatusPill tone="unknown" label="Unknown" />);
    expect(screen.getByText('Unknown').className).not.toContain('status-');
  });

  it('always shows its label, so colour is never the only signal', () => {
    render(<StatusPill tone="error" label="Failed" />);
    expect(screen.getByText('Failed').textContent).toBe('Failed');
  });
});

describe('CategoryChip', () => {
  it('uses neutral tokens only and takes no tone', () => {
    render(<CategoryChip label="Printer" />);
    const { className } = screen.getByText('Printer');
    expect(className).not.toMatch(/status-|brand-/);
    expect(className).toContain('bg-surface-hover');
  });

  it('hides a decorative icon from assistive technology', () => {
    render(<CategoryChip label="Router" icon={<svg data-testid="icon" />} />);
    expect(screen.getByTestId('icon').parentElement?.getAttribute('aria-hidden')).toBe('true');
  });
});
