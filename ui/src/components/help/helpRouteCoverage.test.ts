/**
 * helpRouteCoverage.test.ts — locks GUI help completeness in CI.
 *
 * Every router route MUST have a corresponding HelpDrawer section. If a new
 * page is added to pageRegistry without a help section, this test fails. Pairs
 * with the en/es locale-parity test (PR-D) so help content stays in sync.
 *
 * The route→section map is explicit (not a heuristic) so renames and additions
 * are visible in PR review rather than buried in a regex.
 */
import { describe, expect, it } from 'vitest';
import { pages } from '../../pageRegistry';
import { helpSections } from './helpDrawerContent';

const ROUTE_TO_HELP: Record<string, string> = {
  '/link': 'link',
  '/network': 'network',
  '/path': 'path',
  '/wifi': 'wifi',
  '/security': 'security',
  '/performance': 'performance',
  '/reports': 'reports',
  '/logs': 'logs',
};

describe('GUI help — route coverage', () => {
  const sectionIds = new Set(helpSections.map((s) => s.id));

  it('every route in pageRegistry has an explicit help section mapping', () => {
    const unmapped = pages.filter((p) => !(p.path in ROUTE_TO_HELP));
    expect(
      unmapped,
      `add the route to ROUTE_TO_HELP and create a HelpDrawer section: ${unmapped
        .map((p) => p.path)
        .join(', ')}`,
    ).toEqual([]);
  });

  it('every mapped help id resolves to a HelpDrawer section', () => {
    const missing = Object.entries(ROUTE_TO_HELP)
      .filter(([, helpId]) => !sectionIds.has(helpId))
      .map(([route, helpId]) => `${route} -> ${helpId}`);
    expect(missing, `add a HelpDrawer section for: ${missing.join(', ')}`).toEqual([]);
  });

  it('every route has a HelpDrawer section (composed assertion)', () => {
    const broken = pages
      .map((p) => ({ path: p.path, helpId: ROUTE_TO_HELP[p.path] }))
      .filter((e) => !e.helpId || !sectionIds.has(e.helpId));
    expect(broken).toEqual([]);
  });
});
