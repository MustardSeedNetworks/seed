/**
 * MsnMark — the Mustard Seed Networks corporate mark.
 *
 * Shared shell pattern, kept consistent across seed / stem / niac / trellis by
 * convention; each repo owns its own copy (no master, no sync).
 *
 * Every product carries two marks and they do different jobs. The product mark
 * sits in the rail lockup at the top and answers "which tool am I in". This one
 * sits in the rail footer and answers "whose tool is it" — an endorsement line,
 * deliberately quiet, so it never competes with the product it endorses.
 *
 * The glyph is vendored as an asset rather than inlined as markup. Its colours
 * are fixed brand values, not theme tokens: the mark must look the same in
 * light and dark, on any product, and on the marketing site. Inlining it would
 * invite someone to tokenise those colours, at which point the seed stops being
 * mustard.
 */
import type { FC } from 'react';
import msnLogo from '../assets/msn-logo.svg';

interface MsnMarkProps {
  /** Hide the wordmark when the rail is collapsed; the glyph still shows. */
  collapsed?: boolean;
  className?: string;
}

export const MsnMark: FC<MsnMarkProps> = ({ collapsed = false, className = '' }) => (
  <div
    className={`flex items-center gap-2 ${collapsed ? 'justify-center' : ''} ${className}`}
    data-testid="msn-mark"
  >
    <img src={msnLogo} alt="" aria-hidden="true" className="h-4 w-4 shrink-0" />
    {!collapsed ? (
      <span className="text-[10px] font-semibold tracking-wide text-text-muted">
        Mustard Seed Networks
      </span>
    ) : null}
  </div>
);
