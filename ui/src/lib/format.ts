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
  return typeof value === "number" && Number.isFinite(value);
}

/**
 * Safely formats a time value in milliseconds.
 * Returns a dash "-" for invalid/zero values.
 *
 * @param ms - Time in milliseconds
 * @param fallback - Value to return if ms is invalid (default: "-")
 */
export function formatTime(ms: number | undefined | null, fallback: string = "-"): string {
  if (!isValidNumber(ms) || ms <= 0) {
    return fallback;
  }
  if (ms < 1) {
    return "<1ms";
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
 * Safely formats a number with fixed decimal places.
 * Returns fallback for invalid values.
 *
 * @param value - Number to format
 * @param decimals - Number of decimal places
 * @param fallback - Value to return if invalid (default: "-")
 */
export function formatFixed(
  value: number | undefined | null,
  decimals: number = 1,
  fallback: string = "-",
): string {
  if (!isValidNumber(value)) {
    return fallback;
  }
  return value.toFixed(decimals);
}

/**
 * Safely formats a percentage value.
 * Returns fallback for invalid values.
 *
 * @param value - Percentage value (0-100)
 * @param decimals - Number of decimal places (default: 0)
 * @param fallback - Value to return if invalid (default: "-")
 */
export function formatPercent(
  value: number | undefined | null,
  decimals: number = 0,
  fallback: string = "-",
): string {
  if (!isValidNumber(value)) {
    return fallback;
  }
  return `${value.toFixed(decimals)}%`;
}

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
  fallback: string = "-",
): string {
  if (!isValidNumber(bytes) || bytes < 0) {
    return fallback;
  }
  if (bytes === 0) {
    return "0 B";
  }

  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const size = i < sizes.length ? sizes[i] : sizes.at(-1);

  return `${Number.parseFloat((bytes / k ** i).toFixed(decimals))} ${size}`;
}

/**
 * Safely formats a bits-per-second value to human-readable format.
 * Returns fallback for invalid values.
 *
 * @param bps - Speed in bits per second
 * @param decimals - Number of decimal places (default: 1)
 * @param fallback - Value to return if invalid (default: "-")
 */
export function formatBps(
  bps: number | undefined | null,
  decimals: number = 1,
  fallback: string = "-",
): string {
  if (!isValidNumber(bps) || bps < 0) {
    return fallback;
  }
  if (bps === 0) {
    return "0 bps";
  }

  const k = 1000;
  const sizes = ["bps", "Kbps", "Mbps", "Gbps", "Tbps"];
  const i = Math.floor(Math.log(bps) / Math.log(k));
  const size = i < sizes.length ? sizes[i] : sizes.at(-1);

  return `${Number.parseFloat((bps / k ** i).toFixed(decimals))} ${size}`;
}

/**
 * Safely formats a number with locale-specific formatting.
 * Returns fallback for invalid values.
 *
 * @param value - Number to format
 * @param options - Intl.NumberFormat options
 * @param fallback - Value to return if invalid (default: "-")
 */
export function formatNumber(
  value: number | undefined | null,
  options?: Intl.NumberFormatOptions,
  fallback: string = "-",
): string {
  if (!isValidNumber(value)) {
    return fallback;
  }
  return value.toLocaleString(undefined, options);
}

/**
 * Safely formats a duration in nanoseconds to human-readable format.
 * Returns fallback for invalid values.
 *
 * @param ns - Duration in nanoseconds
 * @param fallback - Value to return if invalid (default: "-")
 */
export function formatNanoseconds(ns: number | undefined | null, fallback: string = "-"): string {
  if (!isValidNumber(ns) || ns <= 0) {
    return fallback;
  }
  const ms = ns / 1_000_000;
  if (ms < 1) {
    return "<1ms";
  }
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(1)}s`;
  }
  return `${ms.toFixed(1)}ms`;
}

/**
 * Safely formats a signal strength value in dBm.
 * Returns fallback for invalid values.
 *
 * @param dbm - Signal strength in dBm
 * @param fallback - Value to return if invalid (default: "-")
 */
export function formatSignalStrength(
  dbm: number | undefined | null,
  fallback: string = "-",
): string {
  if (!isValidNumber(dbm)) {
    return fallback;
  }
  return `${dbm} dBm`;
}

/**
 * Safe division that returns 0 for invalid divisors.
 * Prevents NaN and Infinity from division by zero or invalid numbers.
 *
 * @param numerator - The numerator
 * @param denominator - The denominator
 * @param fallback - Value to return if division is invalid (default: 0)
 */
export function safeDivide(
  numerator: number | undefined | null,
  denominator: number | undefined | null,
  fallback: number = 0,
): number {
  if (!(isValidNumber(numerator) && isValidNumber(denominator)) || denominator === 0) {
    return fallback;
  }
  const result = numerator / denominator;
  return isValidNumber(result) ? result : fallback;
}
