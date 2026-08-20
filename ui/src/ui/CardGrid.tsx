/**
 * CardGrid — the Card grid page archetype: one subject, several independent
 * facets, membership conditional on what the subject can produce.
 *
 * Shared shell pattern, kept visually and behaviourally consistent across
 * seed / stem / niac / trellis by convention; each repo owns this file
 * independently (no master, no sync). All colours reference theme tokens.
 *
 * It is not List + detail. A filtered list of records with one detail pane is
 * a different problem from a set of unrelated cards about the same subject —
 * Link, Wi-Fi, Logs and Reports each show facets of one interface or one
 * system, and the layout carries no selection.
 *
 * The rule this archetype exists to enforce is conditional membership.
 * A card that is not on the page is one of two things, and they are not the
 * same thing to read:
 *
 *   - **The subject cannot produce it.** A wired interface has no channel; a
 *     Free-tier licence has no roam analysis. The grid says so in the card's
 *     place, because a page that quietly omits what it cannot measure looks
 *     identical to one where everything is fine.
 *   - **There is nothing to report.** Cable diagnostics on a healthy link.
 *     Silence about a healthy state is not a missing measurement, and a note
 *     saying "nothing is wrong here" on every load is noise that teaches
 *     people to stop reading the notes that matter.
 *
 * `CardSlot` makes that a required choice rather than a remembered one: there
 * is no way to leave a card out without saying which kind of absence it is.
 *
 * A Card grid page opens with a StatusRollup only when the page itself can be
 * wrong — Link can have no carrier, which is a sentence worth leading with.
 * Pages whose cards each carry their own state do not get a band that only
 * restates them.
 *
 * Usage:
 *   <CardGrid>
 *     <CardSlot present={!isWifi} absence={{ label: 'Wired link', reason: '…' }}>
 *       <LinkCard … />
 *     </CardSlot>
 *     <CardSlot present={linkDown} absence="quiet">
 *       <CableCard … />
 *     </CardSlot>
 *   </CardGrid>
 */
import type { ReactNode } from 'react';
import { cn, layout } from '../styles/theme';

/**
 * Why a card is not on the page.
 *
 * `quiet` is spelled out rather than being the default, so leaving a card out
 * silently is a decision someone made and a reviewer can see.
 */
export type CardAbsence =
  | {
      /** What is missing, in the words the card would have worn. */
      label: string;
      /** Why this subject cannot produce it, and where it does come from. */
      reason: string;
    }
  | 'quiet';

interface CardGridProps {
  children: ReactNode;
  className?: string;
}

export function CardGrid({ children, className }: CardGridProps) {
  return <div className={cn(layout.grid.cards, className)}>{children}</div>;
}

interface CardSlotProps {
  /** Whether the card belongs on the page right now. */
  present: boolean;
  absence: CardAbsence;
  children: ReactNode;
}

export function CardSlot({ present, absence, children }: CardSlotProps) {
  if (present) {
    return <>{children}</>;
  }
  if (absence === 'quiet') {
    return null;
  }
  return <CardAbsent label={absence.label} reason={absence.reason} />;
}

interface CardAbsentProps {
  /** What is missing, in the words the card would have worn. */
  label: string;
  /** Why this subject cannot produce it, and where it does come from. */
  reason: string;
}

/**
 * The card-shaped note that stands in for a card the subject cannot produce.
 *
 * Exported for the cases where the card is withheld by something other than a
 * boolean the page holds — a licence gate's `fallback`, for instance, where
 * there are no children to guard.
 *
 * Dashed rather than solid: it occupies the grid without claiming to be a
 * reading.
 */
export function CardAbsent({ label, reason }: CardAbsentProps) {
  return (
    <section
      data-testid={`card-absent-${slug(label)}`}
      className="rounded-lg border border-dashed border-border-subtle pad"
    >
      <h3 className="text-sm font-semibold text-text-secondary">{label}</h3>
      <p className="mt-tight text-sm text-text-muted">{reason}</p>
    </section>
  );
}

function slug(label: string): string {
  return label
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}
