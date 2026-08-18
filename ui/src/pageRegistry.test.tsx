/**
 * Guards the registry's locale contract: every route resolves real copy
 * in both locales, and the eyebrow slot stays opt-in — a page has one
 * only when its locale namespace declares it.
 *
 * src/test/setup.ts mocks react-i18next globally with a fixed lookup
 * table, which would make every key resolve to itself. This file opts
 * out so the assertions run against the real i18next instance and the
 * real pages.json, which is the whole point of the test.
 */
import { renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

vi.unmock('react-i18next');

const { default: i18n } = await import('./i18n');
const { usePages } = await import('./pageRegistry');
const { useNavGroups } = await import('./navGroups');

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

  it('gives an eyebrow only to pages whose locale declares one', () => {
    const { result } = renderHook(() => usePages());
    const withEyebrow = result.current.filter((page) => page.eyebrow !== undefined);

    expect(withEyebrow.map((page) => page.path)).toEqual(['/network']);
    expect(withEyebrow[0]?.eyebrow).toBe('Diagnostics');
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
