/**
 * seriesPath: a stretch with no sample, or a sample without the series'
 * value, breaks the line; samples within the gap are joined.
 */
import { describe, expect, it } from 'vitest';
import { seriesPath } from './TrendChart';

interface Sample {
  sampledAt: string;
  v?: number;
}

describe('seriesPath', () => {
  const from = Date.parse('2026-10-08T00:00:00Z');
  const to = from + 600_000;
  const at = (s: number): string => new Date(from + s * 1000).toISOString();
  const value = (p: Sample): number | undefined => p.v;

  it('joins polls within the gap and scales to the peak', () => {
    const d = seriesPath(
      [
        { sampledAt: at(0), v: 0 },
        { sampledAt: at(60), v: 10 },
      ],
      value,
      from,
      to,
      10,
      180_000,
    );
    expect(d).toBe('M0.0 120.0 L60.0 0.0');
  });

  it('breaks the line across a gap and a missing value', () => {
    const d = seriesPath(
      [
        { sampledAt: at(0), v: 5 },
        { sampledAt: at(300), v: 5 },
        { sampledAt: at(360) },
        { sampledAt: at(420), v: 5 },
      ],
      value,
      from,
      to,
      10,
      180_000,
    );
    expect(d).toBe('M0.0 60.0 M300.0 60.0 M420.0 60.0');
  });
});
