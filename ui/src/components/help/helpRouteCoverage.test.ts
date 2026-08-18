/**
 * helpRouteCoverage.test.ts — locks GUI help completeness in CI.
 *
 * Every router route MUST declare a HelpDrawer section, and that section MUST
 * exist. The registry entry is the map (seed#1943): the page header's (?)
 * opens the drawer on `page.help`, so a route without one has no help entry
 * point and a route pointing at a missing id opens an empty drawer.
 *
 * Pairs with the en/es locale-parity test so help content stays in sync.
 */
import { renderHook } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { usePages } from '../../pageRegistry';
import { helpSections } from './helpSections';

describe('GUI help — route coverage', () => {
  const { result } = renderHook(() => usePages());
  const pages = result.current;
  const sectionIds = new Set(helpSections.map((s) => s.id));

  it('every route declares a help section', () => {
    const undeclared = pages.filter((p) => !p.help).map((p) => p.path);
    expect(
      undeclared,
      `add a help section id to these pageRegistry entries: ${undeclared.join(', ')}`,
    ).toEqual([]);
  });

  it('every declared help id resolves to a HelpDrawer section', () => {
    const dangling = pages
      .filter((p) => p.help && !sectionIds.has(p.help))
      .map((p) => `${p.path} -> ${p.help}`);
    expect(dangling, `add a HelpDrawer section for: ${dangling.join(', ')}`).toEqual([]);
  });
});
