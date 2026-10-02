import { describe, expect, it } from 'vitest';
import type { TestResult } from './healthCheckCardTypes';
import { failuresFirst, testResultRank } from './healthCheckResultOrder';

const result = (name: string, fields: Partial<TestResult> = {}): TestResult => ({
  name,
  success: true,
  latency: 1,
  ...fields,
});

describe('testResultRank', () => {
  it.each([
    ['a failed check', result('a', { success: false }), 0],
    ['a threshold error', result('a', { testStatus: 'error' }), 0],
    ['an expiring certificate past its error threshold', result('a', { certStatus: 'error' }), 0],
    ['a threshold warning', result('a', { testStatus: 'warning' }), 1],
    ['a certificate warning', result('a', { certStatus: 'warning' }), 1],
    ['a pass', result('a', { testStatus: 'success', certStatus: 'success' }), 2],
    ['a pass with no status fields', result('a'), 2],
  ])('ranks %s as %i', (_label, r, want) => {
    expect(testResultRank(r)).toBe(want);
  });
});

describe('failuresFirst', () => {
  it('moves failures ahead of warnings ahead of passes', () => {
    const got = failuresFirst(
      [
        result('pass'),
        result('warn', { testStatus: 'warning' }),
        result('fail', { success: false }),
      ],
      testResultRank,
    );
    expect(got.map((r) => r.name)).toEqual(['fail', 'warn', 'pass']);
  });

  it('keeps the configured order within a rank', () => {
    const got = failuresFirst(
      [
        result('pass-1'),
        result('fail-1', { success: false }),
        result('pass-2'),
        result('fail-2', { success: false }),
      ],
      testResultRank,
    );
    expect(got.map((r) => r.name)).toEqual(['fail-1', 'fail-2', 'pass-1', 'pass-2']);
  });

  it('does not reorder the caller’s array', () => {
    const input = [result('pass'), result('fail', { success: false })];
    failuresFirst(input, testResultRank);
    expect(input.map((r) => r.name)).toEqual(['pass', 'fail']);
  });
});
