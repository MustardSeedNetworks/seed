/**
 * StatusPill: a verdict, in the status colours, and nothing else.
 *
 * Status colours mean one of success, warning, error or info, or "cannot say"
 * (`unknown`, neutral). They never label a category: a device type, a protocol
 * or a role is a CategoryChip. niac's device-type pills used the status hues
 * for categories (2026-11-22 audit), which makes a category read as a verdict.
 *
 * The label is required and always shown: colour is never the only signal
 * (WCAG 1.4.1). The text takes the `-strong` token, which clears 4.5:1 on its
 * own wash (UI-FLEET-3); the dot takes the bare hue.
 */
import { cx } from './cx';

export type StatusTone = 'success' | 'warning' | 'error' | 'info' | 'unknown';

export interface StatusPillProps {
  tone: StatusTone;
  /** Already translated. */
  label: string;
  className?: string;
}

const tones: Record<StatusTone, { pill: string; dot: string }> = {
  success: { pill: 'bg-status-success/10 text-status-success-strong', dot: 'bg-status-success' },
  warning: { pill: 'bg-status-warning/10 text-status-warning-strong', dot: 'bg-status-warning' },
  error: { pill: 'bg-status-error/10 text-status-error-strong', dot: 'bg-status-error' },
  info: { pill: 'bg-status-info/10 text-status-info-strong', dot: 'bg-status-info' },
  unknown: { pill: 'bg-surface-hover text-text-secondary', dot: 'bg-text-muted' },
};

export function StatusPill({ tone, label, className }: StatusPillProps) {
  const { pill, dot } = tones[tone];
  return (
    <span
      className={cx(
        'inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium',
        pill,
        className,
      )}
      data-tone={tone}
    >
      <span className={cx('h-1.5 w-1.5 shrink-0 rounded-full', dot)} aria-hidden="true" />
      {label}
    </span>
  );
}
