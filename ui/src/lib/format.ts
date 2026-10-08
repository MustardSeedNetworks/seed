// biome-ignore-all lint/style/noInferrableTypes: useExplicitType requires types on default params
/**
 * Number and Time Formatting Utilities
 *
 * Provides safe formatting functions that guard against NaN, Infinity,
 * and other invalid numeric values to prevent "NaN" from displaying in the UI.
 *
 * #775: Centralized formatting with NaN guards to prevent display issues.
 */

/**
 * Checks if a value is a valid, finite number (not NaN, Infinity, or non-numeric).
 */
export function isValidNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value);
}

/**
 * Safely formats a time value in milliseconds.
 * Returns a dash "-" for invalid/zero values.
 *
 * @param ms - Time in milliseconds
 * @param fallback - Value to return if ms is invalid (default: "-")
 */
export function formatTime(ms: number | undefined | null, fallback: string = '-'): string {
  if (!isValidNumber(ms) || ms <= 0) {
    return fallback;
  }
  if (ms < 1) {
    return '<1ms';
  }
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(1)}s`;
  }
  return `${Math.round(ms * 10) / 10}ms`;
}

/**
 * Safely formats a latency value (alias for formatTime).
 */
export const formatLatency: typeof formatTime = formatTime;

/**
 * Safely formats a byte size value to human-readable format.
 * Returns fallback for invalid values.
 *
 * @param bytes - Size in bytes
 * @param decimals - Number of decimal places (default: 1)
 * @param fallback - Value to return if invalid (default: "-")
 */
export function formatBytes(
  bytes: number | undefined | null,
  decimals: number = 1,
  fallback: string = '-',
): string {
  if (!isValidNumber(bytes) || bytes < 0) {
    return fallback;
  }
  if (bytes === 0) {
    return '0 B';
  }

  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const size = i < sizes.length ? sizes[i] : sizes.at(-1);

  return `${Number.parseFloat((bytes / k ** i).toFixed(decimals))} ${size}`;
}

/**
 * Formats a bit rate with an SI prefix: 950 bps, 12.5 Mbps, 1.2 Gbps.
 * Returns fallback for invalid or negative values.
 */
export function formatBitRate(bps: number | undefined | null, fallback: string = '-'): string {
  if (!isValidNumber(bps) || bps < 0) {
    return fallback;
  }
  const units = ['bps', 'kbps', 'Mbps', 'Gbps', 'Tbps'];
  let value = bps;
  let unit = 0;
  while (value >= 1000 && unit < units.length - 1) {
    value /= 1000;
    unit++;
  }
  const digits = unit === 0 || value >= 100 ? 0 : 1;
  return `${Number.parseFloat(value.toFixed(digits))} ${units[unit]}`;
}

/**
 * Formats an events-per-second rate such as interface errors. A rate too small
 * to show at two decimals still reads as non-zero.
 */
export function formatPerSecond(rate: number | undefined | null, fallback: string = '-'): string {
  if (!isValidNumber(rate) || rate < 0) {
    return fallback;
  }
  if (rate === 0) {
    return '0/s';
  }
  if (rate < 0.01) {
    return '<0.01/s';
  }
  return `${Number.parseFloat(rate.toFixed(2))}/s`;
}
