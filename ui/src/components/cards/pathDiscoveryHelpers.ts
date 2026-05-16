import type { TracerouteHop } from '../../types';

export function formatRtt(ns: number): string {
  if (ns <= 0) {
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
    return 'bg-status-error';
  }
  if (rtt / maxRtt > 0.7) {
    return 'bg-status-warning';
  }
  return 'bg-status-success';
}

export function getSourceColor(source: string): string {
  if (source === 'lldp') {
    return 'text-brand-primary';
  }
  if (source === 'cdp') {
    return 'text-status-success';
  }
  return 'text-text-muted';
}
