import { renderHook } from '@testing-library/react';
import { useTranslation } from 'react-i18next';
import { describe, expect, it } from 'vitest';

import { useNavGroups } from './navGroups';
import { usePages } from './pageRegistry';

// Guards finding H3 (nav/route drift): a page reachable by URL but absent from
// the sidebar is a discoverability bug. These assertions fail the build if the
// route table and the sidebar ever diverge again.
describe('navGroups <-> pageRegistry parity', () => {
  const pages = renderHook(() => usePages()).result.current;
  const navGroups = renderHook(() => useNavGroups()).result.current;

  const navPaths = new Set(navGroups.flatMap((group) => group.items.map((item) => item.path)));
  const routePaths = new Set(pages.map((page) => page.path));

  it('exposes every routable page in the sidebar', () => {
    const missing = pages.map((page) => page.path).filter((path) => !navPaths.has(path));
    expect(missing, `pages missing from navGroups: ${missing.join(', ')}`).toEqual([]);
  });

  it('has no sidebar entries pointing at a non-existent route', () => {
    const orphaned = [...navPaths].filter((path) => !routePaths.has(path));
    expect(orphaned, `navGroups entries without a page: ${orphaned.join(', ')}`).toEqual([]);
  });

  // The grouping is declared once, on the page (pageRegistry's `group`), and
  // the rail must file the route under that same heading. Before #2645 the two
  // disagreed silently: `/network` sat under Live Telemetry in the rail while
  // its page printed an eyebrow of "Diagnostics". The eyebrow is now the
  // group's own label, so this assertion is what keeps the rail honest —
  // without it, moving a route between rail groups would leave its eyebrow
  // (and so its page) claiming the old one.
  it('files every route under the group its page declares', () => {
    const groupLabels = renderHook(() => useTranslation('pages')).result.current.t;
    const mismatched = navGroups.flatMap((group) =>
      group.items
        .map((item) => {
          const page = pages.find((candidate) => candidate.path === item.path);
          if (!page) return undefined;
          const declared = groupLabels(`groups.${page.group}`);
          return group.label === declared
            ? undefined
            : `${item.path}: rail says "${group.label}", page declares "${declared}"`;
        })
        .filter((entry): entry is string => entry !== undefined),
    );

    expect(mismatched, mismatched.join('; ')).toEqual([]);
  });

  // The eyebrow is derived, not authored, so no page may carry one that says
  // something other than its group. A locale regaining a `pages.*.eyebrow`
  // key would be silently ignored by usePages(); this states the intent.
  it('gives every page its group label as the eyebrow', () => {
    const groupLabels = renderHook(() => useTranslation('pages')).result.current.t;
    for (const page of pages) {
      expect(page.eyebrow, `${page.path} eyebrow`).toBe(groupLabels(`groups.${page.group}`));
    }
  });
});
