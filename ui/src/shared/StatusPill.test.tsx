import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { CategoryChip } from './CategoryChip';
import {
  type IndicatorTone,
  StatusIndicator,
  StatusPill,
  type StatusTone,
  statusText,
} from './StatusPill';

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

describe('StatusPill icon', () => {
  it('swaps the dot for the tone glyph, hidden from assistive technology, and keeps the label', () => {
    const { container } = render(<StatusPill tone="warning" label="Degraded" icon />);
    const pill = screen.getByText('Degraded');
    expect(pill.querySelector('svg')?.getAttribute('aria-hidden')).toBe('true');
    expect(container.querySelector('.rounded-full.h-1\\.5')).toBeNull();
    expect(pill.className).toContain('text-status-warning-strong');
  });

  it('keeps the dot by default', () => {
    render(<StatusPill tone="success" label="Up" />);
    expect(screen.getByText('Up').querySelector('svg')).toBeNull();
  });
});

describe('StatusIndicator', () => {
  it.each<[IndicatorTone, string]>([
    ['success', 'status-success'],
    ['warning', 'status-warning'],
    ['error', 'status-error'],
    ['info', 'status-info'],
  ])('%s is an image named by its label, on its wash with the -strong text', (tone, hue) => {
    render(<StatusIndicator tone={tone} label={`is ${tone}`} />);
    const badge = screen.getByRole('img', { name: `is ${tone}` });
    expect(badge.className).toContain(`bg-${hue}/10`);
    expect(badge.className).toContain(`text-${hue}-strong`);
    expect(badge.querySelector('svg')?.getAttribute('aria-hidden')).toBe('true');
  });

  it('draws unknown and loading in neutral tokens, and spins loading only when motion is allowed', () => {
    render(
      <>
        <StatusIndicator tone="unknown" label="Unknown" />
        <StatusIndicator tone="loading" label="Loading" />
      </>,
    );
    expect(screen.getByRole('img', { name: 'Unknown' }).className).not.toContain('status-');
    const loading = screen.getByRole('img', { name: 'Loading' });
    expect(loading.className).not.toContain('status-');
    expect(loading.querySelector('svg')?.getAttribute('class')).toContain(
      'motion-safe:animate-spin',
    );
  });

  it('draws a dot variant with the bare hue and no glyph', () => {
    render(<StatusIndicator tone="error" label="Down" variant="dot" size="md" />);
    const dot = screen.getByRole('img', { name: 'Down' });
    expect(dot.className).toContain('bg-status-error');
    expect(dot.className).toContain('h-2.5');
    expect(dot.querySelector('svg')).toBeNull();
  });

  it('has two sizes', () => {
    render(
      <>
        <StatusIndicator tone="success" label="a" size="sm" />
        <StatusIndicator tone="success" label="b" size="md" />
      </>,
    );
    expect(screen.getByRole('img', { name: 'a' }).className).toContain('p-0.5');
    expect(screen.getByRole('img', { name: 'b' }).className).toContain('p-1');
  });
});

describe('statusText', () => {
  it('uses the -strong token for every status hue and neutral for the rest', () => {
    for (const tone of ['success', 'warning', 'error', 'info'] as const) {
      expect(statusText[tone]).toBe(`text-status-${tone}-strong`);
    }
    expect(statusText.unknown).toBe('text-text-secondary');
    expect(statusText.loading).toBe('text-text-secondary');
  });
});
