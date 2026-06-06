// Maps anomaly severities (info < warning < critical) to design-token classes.
// Token-only — no raw colours — so the design-token CI gate stays green and the
// badges are theme-aware.

export interface SeverityStyle {
  /** Tailwind classes for a severity pill (token colours only). */
  badge: string;
  /** Sort rank, most urgent first. */
  rank: number;
}

const styles: Record<string, SeverityStyle> = {
  critical: { badge: 'bg-status-error/10 text-status-error', rank: 3 },
  warning: { badge: 'bg-status-warning/10 text-status-warning', rank: 2 },
  info: { badge: 'bg-status-info/10 text-status-info', rank: 1 },
};

const fallback: SeverityStyle = { badge: 'bg-surface-sunken text-text-secondary', rank: 0 };

/** severityStyle returns the token classes + rank for a severity string. */
export function severityStyle(severity: string): SeverityStyle {
  return styles[severity] ?? fallback;
}
