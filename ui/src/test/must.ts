/**
 * Narrow an indexed lookup that the surrounding assertions already prove exists.
 *
 * `noUncheckedIndexedAccess` types every `arr[0]` and `mock.calls[0]` as
 * possibly undefined, which is correct and which tests hit constantly right
 * after asserting the thing is there. `must` states that expectation as code
 * and fails with a readable message instead of `Cannot read properties of
 * undefined`, which is what a bare `!` would have produced.
 */
export function must<T>(value: T | undefined, what = 'value'): T {
  if (value === undefined) {
    throw new Error(`expected ${what} to be present`);
  }
  return value;
}
