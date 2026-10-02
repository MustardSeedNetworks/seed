import type { TestResult } from './healthCheckCardTypes';

/** Sort rank of a result: failures, then warnings, then passes. */
export type ResultRank = 0 | 1 | 2;

export function testResultRank(result: TestResult): ResultRank {
  if (!result.success || result.testStatus === 'error' || result.certStatus === 'error') {
    return 0;
  }
  if (result.testStatus === 'warning' || result.certStatus === 'warning') {
    return 1;
  }
  return 2;
}

/**
 * The items with failures first and passes last. Within a rank the configured
 * order is kept (Array.prototype.sort is stable), so a list that is all green
 * reads exactly as the operator entered it.
 */
export function failuresFirst<T>(items: readonly T[], rank: (item: T) => ResultRank): T[] {
  return [...items].sort((a, b) => rank(a) - rank(b));
}
