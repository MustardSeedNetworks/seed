/**
 * Alert: an inline message that stays on the page (a Toast is the transient
 * one). Tone is one of the status four; the icon and the text carry it as well
 * as the colour. `error` and `warning` announce at once (`role="alert"`),
 * `success` and `info` politely (`role="status"`).
 *
 * `size="sm"` is the in-card form: seed's dashboard cards hand-roll a padded,
 * tinted `role="alert"` block with 12px text for a failed probe, and this is it,
 * tighter padding and no title. A card whose whole body is an error or a loading
 * state uses Card's own `error` and `loading`; use an Alert for a message that sits
 * beside content.
 */
import { CircleCheck, CircleX, Info, TriangleAlert, X } from 'lucide-react';
import type { ReactNode } from 'react';
import { cx } from './cx';

export type AlertTone = 'success' | 'warning' | 'error' | 'info';

export interface AlertProps {
  tone: AlertTone;
  title?: string;
  children: ReactNode;
  /** `sm` for inside a card. */
  size?: 'md' | 'sm';
  onDismiss?: () => void;
  /** Accessible name of the dismiss button; required with `onDismiss`. */
  dismissLabel?: string;
  className?: string;
  'data-testid'?: string;
}

const tones = {
  success: {
    box: 'border-status-success/30 bg-status-success/10',
    icon: 'text-status-success-strong',
    Icon: CircleCheck,
  },
  warning: {
    box: 'border-status-warning/30 bg-status-warning/10',
    icon: 'text-status-warning-strong',
    Icon: TriangleAlert,
  },
  error: {
    box: 'border-status-error/30 bg-status-error/10',
    icon: 'text-status-error-strong',
    Icon: CircleX,
  },
  info: {
    box: 'border-status-info/30 bg-status-info/10',
    icon: 'text-status-info-strong',
    Icon: Info,
  },
} as const;

export function Alert({
  tone,
  title,
  children,
  size = 'md',
  onDismiss,
  dismissLabel,
  className,
  'data-testid': testId,
}: AlertProps) {
  const { box, icon, Icon } = tones[tone];
  const compact = size === 'sm';
  return (
    <div
      role={tone === 'error' || tone === 'warning' ? 'alert' : 'status'}
      data-tone={tone}
      data-testid={testId}
      className={cx(
        'flex items-start rounded-lg border text-text-primary',
        compact ? 'gap-2 p-2 text-xs' : 'gap-3 p-3 text-sm',
        box,
        className,
      )}
    >
      <Icon className={cx('mt-0.5 h-4 w-4 shrink-0', icon)} aria-hidden="true" />
      <div className="min-w-0 flex-1">
        {title ? <p className="font-semibold">{title}</p> : null}
        <div className="text-text-secondary">{children}</div>
      </div>
      {onDismiss ? (
        <button
          type="button"
          onClick={onDismiss}
          aria-label={dismissLabel}
          className="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-text-muted hover:bg-surface-hover hover:text-text-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-primary"
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </button>
      ) : null}
    </div>
  );
}
