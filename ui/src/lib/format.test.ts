import { describe, expect, it } from 'vitest';
import { formatBitRate, formatPerSecond } from './format';

describe('formatBitRate', () => {
  it.each([
    [0, '0 bps'],
    [950, '950 bps'],
    [1000, '1 kbps'],
    [12_500_000, '12.5 Mbps'],
    [999_000_000, '999 Mbps'],
    [1_240_000_000, '1.2 Gbps'],
    [4e15, '4000 Tbps'],
  ])('formats %d bps as %s', (bps, want) => {
    expect(formatBitRate(bps)).toBe(want);
  });

  it.each([Number.NaN, -1, null, undefined])('falls back for %s', (bps) => {
    expect(formatBitRate(bps, '—')).toBe('—');
  });
});

describe('formatPerSecond', () => {
  it.each([
    [0, '0/s'],
    [0.004, '<0.01/s'],
    [0.25, '0.25/s'],
    [3, '3/s'],
    [1234.567, '1234.57/s'],
  ])('formats %d as %s', (rate, want) => {
    expect(formatPerSecond(rate)).toBe(want);
  });

  it('falls back for a missing rate', () => {
    expect(formatPerSecond(undefined, '—')).toBe('—');
  });
});
