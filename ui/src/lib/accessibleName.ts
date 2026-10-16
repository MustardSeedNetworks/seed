/**
 * Accessible names that still contain the visible label.
 *
 * axe-core 4.14 enables `label-content-name-mismatch` (WCAG 2.5.3 Label in
 * Name) by default. The visible words must occur, in order and without
 * other words between them, inside the accessible name. A longer aria-label
 * that replaces the on-screen label fails the rule. Appending the visible
 * label keeps the extra description and satisfies the check.
 */

const WORD_GAP = /[^\p{L}\p{N}]+/gu;

/** Words the way the axe check tokenizes them: parentheses dropped, then split. */
function words(value: string): string[] {
  let text = value.normalize('NFKD');
  let previous: string;
  do {
    previous = text;
    text = text.replace(/\([^()]*\)/g, ' ');
  } while (text !== previous);
  return text.toLowerCase().split(WORD_GAP).filter(Boolean);
}

function containsVisibleWords(name: string, visible: string): boolean {
  const needle = words(visible);
  if (needle.length === 0) return true;
  const haystack = words(name);
  for (let i = 0; i + needle.length <= haystack.length; i++) {
    if (needle.every((word, index) => word === haystack[i + index])) return true;
  }
  return false;
}

/**
 * `description` is the extra context a screen reader should hear.
 * `visible` is the text drawn on the control. The result contains `visible`
 * as a run of words, so label-in-name passes.
 */
export function accessibleNameIncluding(description: string, visible: string): string {
  const shown = visible.trim();
  if (!shown) return description;
  if (containsVisibleWords(description, shown)) return description;
  return `${description} ${shown}`;
}
