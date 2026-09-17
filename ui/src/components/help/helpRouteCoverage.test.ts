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
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { renderHook } from '@testing-library/react';
import i18next from 'i18next';
import { describe, expect, it } from 'vitest';
import { usePages } from '../../pageRegistry';
import { sectionSearchText } from './helpModel';
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

const pageFiles = {
  '/link': 'LinkPage',
  '/network': 'NetworkPage',
  '/path': 'PathAnalysisPage',
  '/wifi': 'WifiPage',
  '/security': 'SecurityPage',
  '/performance': 'PerformancePage',
  '/reports': 'ReportsPage',
  '/logs': 'LogsPage',
  '/polling-targets': 'PollingTargetsPage',
  '/topology': 'TopologyPage',
  '/alerts': 'AlertsPage',
} as const;
const sourceRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../..');

describe('GUI help — current card coverage', () => {
  it('keeps all 25 sections with translated bodies', () => {
    expect(new Set(helpSections.map((section) => section.id)).size).toBe(25);
    for (const language of ['en', 'es']) {
      const translate = i18next.getFixedT(language, ['help', 'cards', 'pages', 'common'] as const);
      for (const section of helpSections) {
        expect(section.blocks.length, section.id).toBeGreaterThan(0);
        const body = sectionSearchText({ ...section, keywords: [] }, translate);
        expect(body.length, `${language}:${section.id}`).toBeGreaterThan(100);
        expect(body).not.toMatch(/content\.[a-z]+|cards:|common:|pages:/);
      }
    }
  });

  it('checks every routed page when cards are added', () => {
    const { result } = renderHook(() => usePages());
    expect(result.current.map((page) => page.path).sort((a, b) => a.localeCompare(b))).toEqual(
      Object.keys(pageFiles).sort((a, b) => a.localeCompare(b)),
    );
  });

  it.each(Object.entries(pageFiles))(
    '%s explains each card heading from its page source',
    (route, pageFile) => {
      const { result } = renderHook(() => usePages());
      const section = helpSections.find(
        (entry) => entry.id === result.current.find((page) => page.path === route)?.help,
      );
      expect(section).toBeDefined();
      if (!section) throw new Error(`No help for ${route}`);
      const source = readFileSync(resolve(sourceRoot, `pages/${pageFile}.tsx`), 'utf8');
      const imports = [
        ...source.matchAll(/from ['"](\.\.\/components\/(?:cards|wifi)\/[^'"]+)['"]/g),
      ];
      for (const [, componentPath] of imports) {
        if (!componentPath) throw new Error('Missing card import');
        const card = readFileSync(resolve(sourceRoot, 'pages', `${componentPath}.tsx`), 'utf8');
        const namespace = /useTranslation\((?:\[)?['"]([^'"]+)/.exec(card)?.[1];
        const headings = [...card.matchAll(/title=\{\w+\(['"]([^'"]*(?:\.title|Title))['"]\)\}/g)];
        expect(headings.length, componentPath).toBeGreaterThan(0);
        for (const language of ['en', 'es']) {
          const translate = i18next.getFixedT(language, [
            'help',
            'cards',
            'pages',
            'common',
          ] as const);
          const body = sectionSearchText({ ...section, keywords: [] }, translate);
          for (const [, heading] of headings) {
            const label: unknown = i18next.getResource(language, namespace ?? '', heading ?? '');
            if (typeof label !== 'string')
              throw new Error(`Missing card heading: ${namespace}:${heading}`);
            expect(body, `${language}:${route}:${componentPath}:${label}`).toContain(
              label.toLowerCase(),
            );
          }
        }
      }
    },
  );
});
