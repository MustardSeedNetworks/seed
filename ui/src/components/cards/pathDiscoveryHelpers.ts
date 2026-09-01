import { isValidNumber } from '../../lib/format';
import { status as statusColor } from '../../styles/theme';
import type { TracerouteHop } from '../../types';

export function formatRtt(ns: number): string {
  // `NaN <= 0` is false, so a NaN used to fall through every branch below and
  // render as "NaNms" (#775). An absent rtt in an API payload arrives as
  // undefined and reaches here as NaN through the arithmetic.
  if (!isValidNumber(ns) || ns <= 0) {
    return '---';
  }
  const ms = ns / 1_000_000;
  if (ms < 1) {
    return '<1ms';
  }
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(1)}s`;
  }
  return `${ms.toFixed(1)}ms`;
}

export function getMaxRtt(hops: TracerouteHop[]): number {
  const max = Math.max(...hops.filter((h) => h.rtt > 0).map((h) => h.rtt));
  return max > 0 ? max : 1;
}

export function getRttBarColor(state: string, rtt: number, maxRtt: number): string {
  if (state === 'error') {
    return statusColor.bg.error;
  }
  if (rtt / maxRtt > 0.7) {
    return statusColor.bg.warning;
  }
  return statusColor.bg.success;
}

export function getSourceColor(source: string): string {
  if (source === 'lldp') {
    return 'text-brand-primary';
  }
  if (source === 'cdp') {
    return statusColor.text.success;
  }
  return 'text-text-muted';
}
