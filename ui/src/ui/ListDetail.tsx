/**
 * ListDetail — the List + detail page archetype: a filtered record list on
 * the left, the selected record's detail on the right.
 *
 * Shared shell pattern, kept visually and behaviourally consistent across
 * seed / stem / niac / trellis by convention; each repo owns this file
 * independently (no master, no sync). All colours reference theme tokens.
 *
 * The archetype is presentational only — selection, filtering and data live
 * in the page. That keeps one layout serving records as different as polling
 * targets, alerts and reports without growing a configuration language.
 *
 * Two rules from the design system apply here and are enforced by the parts
 * rather than left to callers:
 *   - Figures are monospaced, prose is not. `value` is a figure.
 *   - Only status is saturated: a record's colour comes from its state, never
 *     from its category.
 *
 * Usage:
 *   <ListDetail>
 *     <RecordPane filter={…} chips={…} count={7}>
 *       {targets.map((t) => <RecordRow key={t.id} … />)}
 *     </RecordPane>
 *     <DetailPane eyebrow="Selected target" title="core-01" meta="…">…</DetailPane>
 *   </ListDetail>
 */
import type { ReactNode } from 'react';
import { cn } from '../styles/theme';
import type { RollupState } from './StatusRollup';

/** A record's state uses the same vocabulary as the rollup — one language. */
export type RecordState = RollupState;

const STATE_BAR: Record<RecordState, string> = {
  ok: 'bg-status-success',
  warn: 'bg-status-warning',
  crit: 'bg-status-error',
  // Unknown is not a problem to alarm about, but it is not health either.
  unknown: 'bg-text-disabled',
};

const STATE_VALUE: Record<RecordState, string> = {
  ok: 'text-text-secondary',
  warn: 'text-status-warning',
  crit: 'text-status-error',
  unknown: 'text-text-muted',
};

export function ListDetail({ children }: { children: ReactNode }) {
  return <div className="grid gap-default lg:grid-cols-[340px_1fr] items-start">{children}</div>;
}

interface RecordPaneProps {
  /** The filter control — an input, usually. */
  filter?: ReactNode;
  /** Counted facets over the list. Keep to a handful; this is not a query builder. */
  chips?: ReactNode;
  /** Shown when no record survives the filter, or none exist. */
  empty?: ReactNode;
  children: ReactNode;
}

export function RecordPane({ filter, chips, empty, children }: RecordPaneProps) {
  const rows = Array.isArray(children) ? children.flat() : children;
  const isEmpty = Array.isArray(rows) ? rows.filter(Boolean).length === 0 : !rows;

  return (
    <div className="flex flex-col overflow-hidden rounded-2xl border border-surface-border bg-surface-raised">
      {filter ? <div className="pad-sm border-b border-surface-border">{filter}</div> : null}
      {chips ? (
        <div className="flex items-center gap-tight px-cell py-2 border-b border-surface-border">
          {chips}
        </div>
      ) : null}
      {isEmpty && empty ? (
        <div className="pad-lg text-center body-small text-text-muted">{empty}</div>
      ) : (
        <div className="overflow-y-auto">{rows}</div>
      )}
    </div>
  );
}

interface FilterChipProps {
  label: string;
  /**
   * Omit when the list is filtered server-side and the facet's true size is
   * not known here. A count that is quietly wrong is worse than no count.
   */
  count?: number;
  active?: boolean;
  onClick: () => void;
}

export function FilterChip({ label, count, active = false, onClick }: FilterChipProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={cn(
        'kicker rounded-md px-2 py-1 transition-colors',
        active ? 'bg-surface-sunken text-text-primary' : 'text-text-muted hover:text-text-primary',
      )}
    >
      {label}
      {count === undefined ? null : <span className="figure ml-1">{count}</span>}
    </button>
  );
}

interface RecordRowProps {
  /** The record's identity. */
  name: string;
  /**
   * Hostnames, addresses and ids are figures and set monospaced; a record
   * identified by a sentence — an alert title, a report name — is prose and
   * must not be.
   */
  nameKind?: 'figure' | 'prose';
  /** One line of context. Prose, so not monospaced. */
  meta?: string;
  /** The one figure worth seeing without selecting the record. */
  value?: string;
  state?: RecordState;
  selected?: boolean;
  onSelect: () => void;
  'data-testid'?: string;
}

export function RecordRow({
  name,
  nameKind = 'figure',
  meta,
  value,
  state = 'ok',
  selected = false,
  onSelect,
  'data-testid': testId,
}: RecordRowProps) {
  return (
    <button
      type="button"
      onClick={onSelect}
      aria-current={selected ? 'true' : undefined}
      data-testid={testId}
      className={cn(
        'flex w-full items-center gap-default px-cell py-2 text-left',
        'min-h-[56px] border-b border-surface-border transition-colors',
        selected ? 'bg-surface-sunken' : 'hover:bg-surface-hover',
      )}
    >
      <span
        aria-hidden="true"
        className={cn('h-[30px] w-[3px] shrink-0 rounded-sm', STATE_BAR[state])}
      />
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span
          className={cn(
            'truncate text-text-primary',
            nameKind === 'figure' ? 'figure' : 'text-sm font-medium',
          )}
        >
          {name}
        </span>
        {meta ? <span className="caption truncate">{meta}</span> : null}
      </span>
      {value ? <span className={cn('figure text-xs', STATE_VALUE[state])}>{value}</span> : null}
    </button>
  );
}

interface DetailPaneProps {
  /** Names what the pane is showing, not the record — "Selected target". */
  eyebrow: string;
  title: string;
  meta?: string;
  /** A status readout for the record, shown opposite the title. */
  status?: ReactNode;
  actions?: ReactNode;
  children?: ReactNode;
}

export function DetailPane({ eyebrow, title, meta, status, actions, children }: DetailPaneProps) {
  return (
    <div className="stack-lg rounded-2xl border border-surface-border bg-surface-raised pad-lg">
      <div className="flex flex-wrap items-start gap-default">
        <div className="stack-xs min-w-0">
          <p className="kicker">{eyebrow}</p>
          <h2 className="figure text-lg font-bold text-text-primary">{title}</h2>
          {meta ? <p className="body-small">{meta}</p> : null}
        </div>
        {status ? <div className="ml-auto">{status}</div> : null}
      </div>
      {actions ? <div className="flex flex-wrap gap-compact">{actions}</div> : null}
      {children}
    </div>
  );
}

interface DetailFactsProps {
  /** Label/value pairs. Values are figures unless `prose` says otherwise. */
  items: { label: string; value: ReactNode; prose?: boolean }[];
}

export function DetailFacts({ items }: DetailFactsProps) {
  return (
    <dl className="grid gap-default sm:grid-cols-2">
      {items.map((item) => (
        <div key={item.label} className="stack-xs">
          <dt className="caption">{item.label}</dt>
          <dd className={cn('text-sm text-text-primary', item.prose ? '' : 'figure')}>
            {item.value}
          </dd>
        </div>
      ))}
    </dl>
  );
}

/** Nothing selected is a normal state, not an error — say so plainly. */
export function DetailEmpty({ children }: { children: ReactNode }) {
  return (
    <div className="rounded-2xl border border-dashed border-surface-border pad-xl text-center body-small text-text-muted">
      {children}
    </div>
  );
}
