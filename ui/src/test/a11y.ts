/**
 * a11y.ts — accessibility test helper.
 *
 * Runs axe-core against rendered output and fails the test on any violation.
 * This is the "axe gate" referenced in the PR-A tooltip/a11y plan: pair every
 * component or screen with an a11y test that calls expectNoAxeViolations() on
 * the rendered container. Catches button-name, image-alt, link-name,
 * aria-allowed-attr, label, and other WCAG rules that no static lint detects.
 *
 * Usage:
 *   const { container } = render(<MyComponent />);
 *   await expectNoAxeViolations(container);
 *
 * The harness is intentionally minimal (raw axe-core, no jest-axe/vitest-axe
 * wrapper) so it ages well with axe-core upgrades and matches the seed/stem/
 * niac "each repo owns its own copy" convention — the file shape is identical
 * across repos.
 */
import axe, { type AxeResults, type Result, type RunOptions } from 'axe-core';

/** Default rule set focused on the violations our app cares about. */
const DEFAULT_RULES: RunOptions = {
  rules: {
    // 'region' fires on isolated component renders that lack a landmark. Off
    // for component-scoped tests; re-enable in screen-level harness tests.
    region: { enabled: false },
    // color-contrast cannot run here at all. It samples rendered pixels
    // through a canvas, and jsdom has no canvas implementation — axe was
    // reporting it under `incomplete` and this helper only looked at
    // `violations`, so it was silently skipped on every run while emitting 38
    // "getContext() not implemented" warnings per suite (#2051).
    //
    // Disabled explicitly rather than left to fail quietly: a rule that cannot
    // work should say so in code. Contrast belongs to the Storybook a11y run,
    // which renders in real chromium.
    'color-contrast': { enabled: false },
  },
};

/**
 * Rules allowed to come back `incomplete` — meaning axe could not decide.
 *
 * Everything else being incomplete is treated as a failure, because a rule
 * that silently cannot evaluate is indistinguishable from one that passed.
 * That is exactly how color-contrast went unchecked here.
 */
const ALLOWED_INCOMPLETE = new Set<string>([]);

export async function runAxe(
  container: Element,
  options: RunOptions = DEFAULT_RULES,
): Promise<AxeResults> {
  return axe.run(container, options);
}

export async function expectNoAxeViolations(
  container: Element,
  options?: RunOptions,
): Promise<void> {
  const results = await runAxe(container, options);

  // An unexpected `incomplete` is a blind spot, not a pass: axe ran the rule
  // and could not reach a verdict, which this helper used to discard.
  const unexpected = results.incomplete
    .map((r) => r.id)
    .filter((id) => !ALLOWED_INCOMPLETE.has(id));
  if (unexpected.length > 0) {
    throw new Error(
      `axe could not evaluate ${unexpected.length} rule(s): ${unexpected.join(', ')}. ` +
        'Either the environment cannot support them — disable them in DEFAULT_RULES ' +
        'with a note saying where they are covered instead — or add them to ' +
        'ALLOWED_INCOMPLETE with a reason.',
    );
  }

  if (results.violations.length === 0) return;
  throw new Error(formatViolations(results.violations));
}

function formatViolations(violations: Result[]): string {
  const lines = [`axe found ${violations.length} accessibility violation(s):`];
  for (const v of violations) {
    lines.push(`  [${v.id}] ${v.help}  (${v.nodes.length} node${v.nodes.length === 1 ? '' : 's'})`);
    for (const node of v.nodes.slice(0, 3)) {
      lines.push(`    ${node.html.slice(0, 200)}`);
    }
    lines.push(`    help: ${v.helpUrl}`);
  }
  return lines.join('\n');
}
