/**
 * DiffView — a unified configuration diff with its changes highlighted.
 *
 * The server renders the diff (internal/deviceconfig unifiedDiff); this only
 * classifies each line by its prefix. The "---" and "+++" file headers name
 * backup ids the page already shows as dates, so they are dropped.
 */

import type { JSX } from 'react';

export type DiffLineKind = 'added' | 'removed' | 'hunk' | 'context';

export interface DiffLine {
  /** The line's position in the rendered diff, which never reorders. */
  n: number;
  kind: DiffLineKind;
  text: string;
}

export function parseUnifiedDiff(unified: string): DiffLine[] {
  return unified
    .split('\n')
    .filter((line, i, lines) => !(i === lines.length - 1 && line === ''))
    .filter((line) => !line.startsWith('--- ') && !line.startsWith('+++ '))
    .map((text, n) => ({ n, kind: lineKind(text), text }));
}

function lineKind(text: string): DiffLineKind {
  if (text.startsWith('@@')) {
    return 'hunk';
  }
  if (text.startsWith('+')) {
    return 'added';
  }
  if (text.startsWith('-')) {
    return 'removed';
  }
  return 'context';
}

const LINE_CLASS: Record<DiffLineKind, string> = {
  added: 'bg-status-success/10 text-status-success-strong',
  removed: 'bg-status-error/10 text-status-error-strong',
  hunk: 'bg-surface-sunken text-text-muted',
  context: 'text-text-secondary',
};

export function DiffView({ unified }: { unified: string }): JSX.Element {
  return (
    <pre
      data-testid="config-diff"
      className="max-h-[32rem] overflow-auto rounded-lg border border-surface-border bg-surface-base py-2 text-xs leading-5"
    >
      <code className="block min-w-max font-mono">
        {parseUnifiedDiff(unified).map((line) => (
          <span
            key={line.n}
            data-kind={line.kind}
            className={`block px-cell ${LINE_CLASS[line.kind]}`}
          >
            {line.text || ' '}
          </span>
        ))}
      </code>
    </pre>
  );
}
