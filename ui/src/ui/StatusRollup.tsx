/**
 * StatusRollup — the one band that answers "is anything wrong".
 *
 * Shared shell pattern, kept visually and behaviourally consistent across
 * seed / stem / niac / trellis by convention; each repo owns this file
 * independently (no master, no sync). All colours reference theme tokens.
 *
 * Any page that can be *wrong* opens with this rather than with stat cards.
 *
 * The `unknown` state exists because the alternative is worse than useless. A
 * rollup that renders zeros when its source failed says "nothing is wrong" at
 * the exact moment nobody can tell — so `unknown` refuses to print the figures
 * at all and says why. That is enforced here rather than left to each caller
 * to remember: pass figures with `state="unknown"` and they render as em
 * dashes, because a caller under pressure should not be able to produce a
 * reassuring screen by accident.
 *
 * Usage:
 *   <StatusRollup
 *     state="warn"
 *     headline="Reflector dropped 2.1% of frames on the last run"
 *     body="Detail is in the run log; the runbook covers loss above 1%."
 *     figures={[{ label: 'Frames', value: '1.2M' }, { label: 'Loss', value: '2.1%' }]}
 *   />
 */
import type { FC, ReactNode } from 'react';

export type RollupState = 'ok' | 'warn' | 'crit' | 'unknown';

export interface RollupFigure {
  label: string;
  /** Rendered in the figure face. Ignored when the rollup state is unknown. */
  value: string;
}

interface StatusRollupProps {
  state: RollupState;
  /** A sentence that names the problem, not a category. */
  headline: string;
  /** One paragraph of what to do about it. */
  body?: string;
  /** Up to four. More than four stops being a summary. */
  figures?: RollupFigure[];
  actions?: ReactNode;
  className?: string;
}

/**
 * `ok` is deliberately calm: a status edge and a dot, no green banner. Nothing
 * being wrong is the normal case and should not compete for attention with the
 * case that is.
 */
const STATE_STYLES: Record<
  RollupState,
  { edge: string; dot: string; kicker: string; wash: string }
> = {
  ok: {
    edge: 'bg-status-success',
    dot: 'bg-status-success',
    kicker: 'text-status-success',
    wash: '',
  },
  warn: {
    edge: 'bg-status-warning',
    dot: 'bg-status-warning',
    kicker: 'text-status-warning',
    wash: 'bg-status-warning/5',
  },
  crit: {
    edge: 'bg-status-error',
    dot: 'bg-status-error',
    kicker: 'text-status-error',
    wash: 'bg-status-error/5',
  },
  unknown: {
    edge: 'bg-text-muted',
    dot: 'bg-text-muted',
    kicker: 'text-text-muted',
    wash: '',
  },
};

const STATE_LABELS: Record<RollupState, string> = {
  ok: 'All clear',
  warn: 'Degraded',
  crit: 'Critical',
  unknown: 'No data',
};

export const StatusRollup: FC<StatusRollupProps> = ({
  state,
  headline,
  body,
  figures = [],
  actions,
  className = '',
}) => {
  const styles = STATE_STYLES[state];
  const shown = figures.slice(0, 4);

  return (
    <section
      data-state={state}
      aria-live="polite"
      className={`relative overflow-hidden rounded-[18px] border border-hairline bg-surface-raised ${styles.wash} ${className}`}
    >
      {/* 3px status edge down the left. */}
      <span aria-hidden="true" className={`absolute inset-y-0 left-0 w-[3px] ${styles.edge}`} />

      <div className="flex flex-col gap-6 p-6 pl-7 lg:flex-row lg:items-start lg:justify-between">
        <div className="min-w-0">
          <p className="kicker flex items-center gap-2">
            {/* motion-safe: the pulse is decoration, and a reader who has asked
                for less motion should not have to watch it to read the state. */}
            <span
              aria-hidden="true"
              className={`h-2 w-2 rounded-full motion-safe:animate-pulse ${styles.dot}`}
            />
            <span className={styles.kicker}>{STATE_LABELS[state]}</span>
          </p>
          <h2 className="mt-2 text-xl font-extrabold tracking-[-0.02em] text-text-primary">
            {headline}
          </h2>
          {body ? <p className="mt-2 max-w-prose text-sm text-text-secondary">{body}</p> : null}
          {actions ? <div className="mt-4 flex flex-wrap gap-2">{actions}</div> : null}
        </div>

        {shown.length > 0 ? (
          <dl className="flex shrink-0 flex-wrap gap-x-10 gap-y-4">
            {shown.map((figure) => (
              <div key={figure.label}>
                <dd className="figure text-2xl font-bold leading-none text-text-primary">
                  {state === 'unknown' ? '—' : figure.value}
                </dd>
                <dt className="kicker mt-2">{figure.label}</dt>
              </div>
            ))}
          </dl>
        ) : null}
      </div>
    </section>
  );
};
