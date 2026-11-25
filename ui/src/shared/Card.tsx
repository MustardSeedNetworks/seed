/**
 * Card: the single container for a block of page content.
 *
 * One radius and one hairline border, both from the shared `.panel` class, and
 * flat: no shadow, no gradient, no blur. A Card does not nest. A Card inside a
 * Card is the audit's "nested versus flat" finding, and it draws a grid of boxes
 * where the eye should read content. Inside a Card use a heading, a CardDivider
 * or spacing. A DataTable and an EmptyState also go inside a Card or stand
 * alone; neither is a Card.
 *
 * One API covers seed's status Card (about 100 dashboard cards: title, subtitle,
 * icon, status badge, rows) and niac's two (the generic `Card` with a variant
 * and padding, and `StatusCard`/`BaseCard`):
 *
 * - Header: `title` (an h2; `headingLevel` 3 or 4 under a section heading),
 *   `subtitle`, a decorative `icon`, `headerAction` (a button or a count), and a
 *   `status` indicator at the right edge. `status` needs its `statusLabel` (the
 *   indicator is an image, named in words); both or neither.
 * - `tone`: a status-coloured bar down the left edge, for a card whose whole
 *   state is a verdict. It is not the only signal: pair it with `status` or
 *   text. Omit it for a neutral card.
 * - Body states: `loading` shows skeleton lines and sets `aria-busy`; `error`
 *   shows the message as an alert; `empty` replaces the body with a neutral
 *   message. Precedence is loading, error, empty, children. Pass `status`
 *   yourself ("loading", "error", "unknown") to match.
 * - `padding`: `md` (default), `sm`, `lg`, or `none` for a body that brings its
 *   own inset (the header keeps its own).
 * - `onClick` makes the whole card a button (Enter and Space too) with a hover
 *   border and focus ring; keep interactive controls out of such a card, or
 *   leave `onClick` off and put a button in `headerAction`.
 * - `liveRegion` announces changes politely, for a card that refreshes in place.
 *
 * CardRow (label and value), CardValue (a prominent figure) and CardDivider
 * fill the body.
 */
import type { KeyboardEvent, ReactNode } from 'react';
import { cx } from './cx';
import { Skeleton } from './Skeleton';
import {
  type IndicatorTone,
  StatusIcon,
  StatusIndicator,
  type StatusTone,
  statusText,
} from './StatusPill';

export type CardStatus = IndicatorTone;

interface CardBaseProps {
  title?: string;
  subtitle?: string;
  /** Decorative; left of the title. */
  icon?: ReactNode;
  /** Right of the title: one button, a count or a link. */
  headerAction?: ReactNode;
  /** The heading's level: 2 on a page, 3 or 4 under a section heading. */
  headingLevel?: 2 | 3 | 4;
  /** A bar down the left edge in the status colour. */
  tone?: StatusTone;
  padding?: 'none' | 'sm' | 'md' | 'lg';
  /** Skeleton lines in place of the body; sets `aria-busy`. */
  loading?: boolean;
  /** The body is this message, as an alert. */
  error?: string;
  /** The body is this neutral message: nothing to show yet. */
  empty?: ReactNode;
  /** Makes the whole card a button. */
  onClick?: () => void;
  /** Announces changes politely, for a card that refreshes in place. */
  liveRegion?: boolean;
  'aria-label'?: string;
  children?: ReactNode;
  id?: string;
  className?: string;
  'data-testid'?: string;
}

type CardStatusProps =
  | { status?: undefined; statusLabel?: undefined }
  | {
      status: CardStatus;
      /** Accessible name of the indicator, already translated: "Success", "Loading". */
      statusLabel: string;
    };

export type CardProps = CardBaseProps & CardStatusProps;

const paddings = {
  none: '',
  sm: 'p-3',
  md: 'p-4 sm:p-5',
  lg: 'p-6',
} as const;

const toneBars: Record<StatusTone, string> = {
  success: 'border-l-4 border-l-status-success',
  warning: 'border-l-4 border-l-status-warning',
  error: 'border-l-4 border-l-status-error',
  info: 'border-l-4 border-l-status-info',
  unknown: '',
};

const headings = { 2: 'h2', 3: 'h3', 4: 'h4' } as const;

export function Card({
  title,
  subtitle,
  icon,
  headerAction,
  headingLevel = 2,
  status,
  statusLabel,
  tone,
  padding = 'md',
  loading = false,
  error,
  empty,
  onClick,
  liveRegion = false,
  'aria-label': ariaLabel,
  children,
  id,
  className,
  'data-testid': testId,
}: CardProps) {
  const Heading = headings[headingLevel];
  const interactive = onClick !== undefined;
  const inset = padding === 'none';
  const hasHeader = title !== undefined || subtitle !== undefined || icon || headerAction || status;

  const onKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    if (
      interactive &&
      event.target === event.currentTarget &&
      (event.key === 'Enter' || event.key === ' ')
    ) {
      event.preventDefault();
      onClick();
    }
  };

  let body: ReactNode = children;
  if (loading) {
    body = <Skeleton lines={3} />;
  } else if (error !== undefined) {
    body = (
      <p role="alert" className="text-sm font-medium text-status-error-strong">
        {error}
      </p>
    );
  } else if (empty !== undefined) {
    body = <div className="text-sm text-text-secondary">{empty}</div>;
  }

  return (
    <section
      id={id}
      data-testid={testId}
      aria-label={ariaLabel}
      aria-busy={loading || undefined}
      aria-live={liveRegion ? 'polite' : undefined}
      aria-atomic={liveRegion ? true : undefined}
      {...(interactive ? { role: 'button', tabIndex: 0, onClick, onKeyDown } : {})}
      className={cx(
        'panel min-w-0',
        paddings[padding],
        tone && toneBars[tone],
        interactive &&
          'panel-hover cursor-pointer focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary',
        className,
      )}
    >
      {hasHeader ? (
        <div
          className={cx(
            'flex flex-wrap items-center justify-between gap-2',
            inset ? 'border-b border-hairline px-4 py-3' : 'mb-3',
          )}
        >
          <div className="flex min-w-0 items-center gap-2">
            {icon ? (
              <span className="shrink-0 text-text-muted [&>svg]:h-5 [&>svg]:w-5" aria-hidden="true">
                {icon}
              </span>
            ) : null}
            <div className="min-w-0">
              {title === undefined ? null : (
                <Heading className="text-base font-semibold leading-snug text-text-primary">
                  {title}
                </Heading>
              )}
              {subtitle ? (
                <p className="text-xs leading-tight text-text-muted">{subtitle}</p>
              ) : null}
            </div>
          </div>
          {headerAction || status ? (
            <div className="flex items-center gap-2">
              {headerAction}
              {status === undefined ? null : (
                <StatusIndicator tone={status} label={statusLabel} size="md" />
              )}
            </div>
          ) : null}
        </div>
      ) : null}
      {body}
    </section>
  );
}

export interface CardRowProps {
  label: string;
  value: ReactNode;
  /** Colours the value and puts the status glyph before it, so colour is not alone. */
  status?: CardStatus;
  /** Wraps a long value instead of truncating it. */
  wrap?: boolean;
  mono?: boolean;
  align?: 'left' | 'right';
}

/** A label and its value on one line; a truncated string value shows in full on hover. */
export function CardRow({
  label,
  value,
  status,
  wrap = false,
  mono = false,
  align = 'right',
}: CardRowProps) {
  const text = typeof value === 'string' || typeof value === 'number' ? String(value) : undefined;
  return (
    <div className={cx('flex justify-between gap-2 py-1', wrap ? 'items-start' : 'items-center')}>
      <span className="shrink-0 text-sm text-text-muted">{label}</span>
      <span
        title={wrap ? undefined : text}
        data-testid="card-row-value"
        className={cx(
          'flex min-w-0 items-center gap-1 text-sm font-medium',
          align === 'right' ? 'justify-end text-right' : 'justify-start text-left',
          wrap ? 'break-all whitespace-pre-wrap' : 'truncate',
          mono && 'figure',
          status ? statusText[status] : 'text-text-primary',
        )}
      >
        {status ? <StatusIcon tone={status} /> : null}
        <span className={wrap ? undefined : 'truncate'}>{value}</span>
      </span>
    </div>
  );
}

export interface CardValueProps {
  label?: string;
  value: string | number;
  unit?: string;
  size?: 'sm' | 'md' | 'lg';
  status?: CardStatus;
  mono?: boolean;
  /** Wraps a long value instead of letting it widen the card. */
  allowWrap?: boolean;
}

const valueSizes = {
  sm: 'text-sm font-medium',
  md: 'text-base font-medium leading-snug',
  lg: 'text-xl font-semibold leading-snug',
} as const;

/** A prominent figure with an optional label above and unit after. */
export function CardValue({
  label,
  value,
  unit,
  size = 'md',
  status,
  mono = false,
  allowWrap = false,
}: CardValueProps) {
  return (
    <div className="min-w-0">
      {label ? <p className="mb-1 text-xs text-text-muted">{label}</p> : null}
      <p
        data-testid="card-value"
        className={cx(
          'flex items-center gap-1.5',
          valueSizes[size],
          status ? statusText[status] : 'text-text-primary',
          mono && 'figure',
          allowWrap && 'break-all whitespace-pre-wrap',
        )}
      >
        {status ? <StatusIcon tone={status} /> : null}
        <span className="flex items-baseline gap-1">
          <span>{value}</span>
          {unit ? <span className="text-sm font-normal text-text-muted">{unit}</span> : null}
        </span>
      </p>
    </div>
  );
}

/** A hairline between groups of rows. */
export function CardDivider({ className }: { className?: string }) {
  return <hr className={cx('my-3 border-hairline', className)} />;
}
