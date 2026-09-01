/**
 * Nothing rendered to an operator may read "NaN" (#775).
 *
 * These formatters take numbers straight from API payloads, where an optional
 * field arrives as undefined and arithmetic on it yields NaN. None of the
 * comparisons they used caught that: `NaN <= 0` is false, so a NaN fell
 * through every branch and came out the other side as text.
 */

import { describe, expect, it } from 'vitest';
import { formatUptime } from './DiscoveryModalDeviceRow';
import { formatRtt } from './pathDiscoveryHelpers';

/** The values an absent or unparseable API number actually arrives as. */
const NOT_A_NUMBER = [
  Number.NaN,
  undefined as unknown as number,
  null as unknown as number,
  Number.POSITIVE_INFINITY,
  Number.NEGATIVE_INFINITY,
];

describe('formatRtt', () => {
  it.each(NOT_A_NUMBER)('renders a placeholder rather than text for %s', (value) => {
    const out = formatRtt(value);

    expect(out).not.toMatch(/NaN|Infinity/);
    expect(out).toBe('---');
  });

  it('still formats real round-trip times', () => {
    expect(formatRtt(500_000)).toBe('<1ms');
    expect(formatRtt(12_300_000)).toBe('12.3ms');
    expect(formatRtt(2_500_000_000)).toBe('2.5s');
    expect(formatRtt(0)).toBe('---');
  });
});

describe('formatUptime', () => {
  it.each(NOT_A_NUMBER)('renders a placeholder rather than text for %s', (value) => {
    const out = formatUptime(value);

    expect(out).not.toMatch(/NaN|Infinity/);
    expect(out).toBe('---');
  });

  it('still formats real uptimes', () => {
    // SNMP sysUpTime is in hundredths of a second.
    expect(formatUptime(0)).toBe('0m');
    expect(formatUptime(360_000)).toBe('1h 0m');
    expect(formatUptime(8_640_000)).toBe('1d 0h 0m');
  });

  it('treats a negative tick count as unusable rather than formatting it', () => {
    expect(formatUptime(-1)).toBe('---');
  });
});
