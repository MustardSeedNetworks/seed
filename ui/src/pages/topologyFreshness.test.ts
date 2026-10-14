import { describe, expect, it } from 'vitest';
import { nodeFreshness } from './topologyFreshness';

const NOW = Date.parse('2026-10-13T12:00:00Z');
const ago = (seconds: number): string => new Date(NOW - seconds * 1000).toISOString();

describe('nodeFreshness', () => {
  it.each([
    ['polled on schedule', ago(60), 300, 'current'],
    ['exactly two intervals old', ago(600), 300, 'current'],
    ['three missed polls', ago(900), 300, 'stale'],
    ['exactly ten intervals old', ago(3000), 300, 'stale'],
    ['an hour old on a ten-minute cadence', ago(3600), 600, 'stale'],
    ['an hour old on a five-minute cadence', ago(3600), 300, 'unknown'],
    ['a week old', ago(7 * 86400), 300, 'unknown'],
    ['no enabled target polls it', ago(10), undefined, 'unknown'],
    ['never dated', '', 300, 'unknown'],
  ] as const)('%s', (_name, lastSeen, interval, want) => {
    expect(nodeFreshness(lastSeen, interval, NOW)).toBe(want);
  });
});
