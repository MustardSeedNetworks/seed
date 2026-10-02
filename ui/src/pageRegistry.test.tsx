/**
 * Guards the registry's locale contract: every route resolves real copy
 * in both locales, and the eyebrow slot stays opt-in — a page has one
 * only when its locale namespace declares it.
 */
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import i18n from './i18n';
import { useNavGroups } from './navGroups';
import { usePages } from './pageRegistry';

describe('page registry translations', () => {
  afterEach(async () => {
    await i18n.changeLanguage('en');
  });

  it.each(['en', 'es'])('resolves page metadata in %s', async (language) => {
    await i18n.changeLanguage(language);
    const { result, unmount } = renderHook(() => usePages());

    for (const page of result.current) {
      // An unresolved key falls back to the key itself, e.g. "network.title".
      expect(page.label, `${page.path} label`).not.toContain('.label');
      expect(page.title, `${page.path} title`).not.toContain('.title');
      expect(page.description, `${page.path} description`).not.toContain('.description');
    }
    unmount();
  });

  // This used to assert that `/network` was the one page with an eyebrow and
  // that the eyebrow read "Diagnostics" — which is precisely the defect #2645
  // reports, since the rail files `/network` under Live Telemetry. The eyebrow
  // is no longer authored per page: it is the label of the group the page
  // declares, so every page has one and none can contradict the rail.
  it('gives every page its declared group label as the eyebrow', () => {
    const { result } = renderHook(() => usePages());

    expect(result.current.every((page) => page.eyebrow.length > 0)).toBe(true);
    const network = result.current.find((page) => page.path === '/network');
    expect(network?.group).toBe('liveTelemetry');
    expect(network?.eyebrow).toBe('Live Telemetry');
  });
});

describe('rail <-> header label agreement', () => {
  afterEach(async () => {
    await i18n.changeLanguage('en');
  });

  it.each(['en', 'es'])(
    'labels a route the same in the rail and the registry in %s',
    async (language) => {
      await i18n.changeLanguage(language);
      const pages = renderHook(() => usePages()).result.current;
      const rail = renderHook(() => useNavGroups()).result.current;

      const railLabel = new Map(
        rail.flatMap((group) => group.items.map((item) => [item.path, item.label] as const)),
      );
      for (const page of pages) {
        expect(railLabel.get(page.path), `${page.path} rail label`).toBe(page.label);
      }
    },
  );

  it('translates the rail into the active locale', async () => {
    await i18n.changeLanguage('es');
    const rail = renderHook(() => useNavGroups()).result.current;
    const byPath = new Map(
      rail.flatMap((group) => group.items.map((item) => [item.path, item.label] as const)),
    );

    expect(byPath.get('/logs')).toBe('Registros');
    expect(byPath.get('/topology')).toBe('Topología');
    // Wi-Fi is a standard term and reads the same in every locale.
    expect(byPath.get('/wifi')).toBe('Wi-Fi');
    expect(rail[0]?.label).toBe('Telemetría en vivo');
  });
});

/**
 * CI's phone-width job visits a hand-listed set of routes: the reusable
 * workflow cannot read this registry. A page added here but not there would
 * ship without ever being checked at 390px, with the job still green.
 */
describe('pageRegistry <-> phone-width routes', () => {
  it('checks every registered page at phone width', () => {
    const ci = readFileSync(resolve(import.meta.dirname, '../../.github/workflows/ci.yml'), 'utf8');
    const job = ci.slice(ci.indexOf('\n  phone-width:\n'));
    const routes = /^ {6}routes: '(.+)'$/m.exec(job)?.[1];
    expect(routes, 'phone-width job has no routes input').toBeDefined();

    const byPath = (a: string, b: string) => a.localeCompare(b);
    const { result: pages } = renderHook(() => usePages());
    expect((JSON.parse(routes ?? '[]') as string[]).sort(byPath)).toEqual(
      pages.current.map((page) => page.path).sort(byPath),
    );
  });
});
