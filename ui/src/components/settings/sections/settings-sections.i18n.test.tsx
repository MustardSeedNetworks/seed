/**
 * settings-sections.i18n.test.tsx — every settings section renders real locale
 * copy, in both locales.
 *
 * S1-14c (#1942). The two page slices before this one hand-listed each
 * surface's English strings; at 36 section files that does not scale and it
 * misses whatever the author did not think to list. So this suite asserts the
 * property instead of the strings: render a section under `es`, then fail on
 * any en value whose es translation differs. A hardcoded label and a key whose
 * es value was never translated both surface as the same failure, and a section
 * added later inherits the assertion the moment it joins the fixture table.
 *
 * The English half is not decoration: without it a section that renders nothing
 * at all — a broken mock, a licence gate — would pass the Spanish half
 * vacuously.
 */

import enCommon from '@locales/en/common.json';
import enErrors from '@locales/en/errors.json';
import enHelp from '@locales/en/help.json';
import enSettings from '@locales/en/settings.json';
import esCommon from '@locales/es/common.json';
import esErrors from '@locales/es/errors.json';
import esHelp from '@locales/es/help.json';
import esSettings from '@locales/es/settings.json';
import { render, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { type CurrentUser, RoleProvider } from '../../../contexts/RoleContext';
import i18n from '../../../i18n';
import { DNT_TERMS } from '../../../i18n/dnt';
import {
  COPY_ONLY_SECTIONS,
  MIXED_SECTIONS,
  RAW_GET_BODIES,
  SECTIONS,
} from './settings-sections.fixtures';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../../../api/client', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (): Promise<unknown> => Promise.resolve({}),
    put: (): Promise<unknown> => Promise.resolve({}),
    delete: (): Promise<unknown> => Promise.resolve({}),
  },
}));
vi.mock('../../../api', () => ({
  api: {
    get: (path: string): Promise<unknown> => mockGet(path),
    post: (): Promise<unknown> => Promise.resolve({}),
    put: (): Promise<unknown> => Promise.resolve({}),
    delete: (): Promise<unknown> => Promise.resolve({}),
  },
}));
vi.mock('../../../contexts/LicenseContext', () => ({
  useLicense: (): { status: { features: string[] } } => ({
    status: { features: ['sso', 'multi_interface'] },
  }),
}));
vi.mock('../../../contexts/profileContext', () => ({
  useProfileContext: (): Record<string, () => unknown> => ({
    getAllEthernetInterfaces: () => [{ name: 'eth0' }, { name: 'eth1' }],
    getAllWifiInterfaces: () => [{ name: 'wlan0' }],
    getEthernetInterface: () => ({ name: 'eth0' }),
    getWifiInterface: () => ({ name: 'wlan0' }),
    addEthernetInterface: () => undefined,
    addWifiInterface: () => undefined,
    removeEthernetInterface: () => undefined,
    removeWifiInterface: () => undefined,
    setActiveEthernetInterface: () => undefined,
    setActiveWifiInterface: () => undefined,
  }),
}));

type Json = { [k: string]: string | Json };

function flatten(node: Json, prefix = ''): [string, string][] {
  return Object.entries(node).flatMap(([key, value]) =>
    typeof value === 'string'
      ? ([[prefix + key, value]] as [string, string][])
      : flatten(value, `${prefix}${key}.`),
  );
}

/**
 * Every English string a settings section can render that Spanish spells
 * differently. A value both locales share is a standard term (SNMP, MTU, dBm)
 * and proves nothing when it shows up under `es`, so it is excluded here rather
 * than maintained as a second exception list.
 *
 * Interpolated values are dropped: `Connected to {{ssid}}` never appears
 * literally in the DOM, and its `{{` would not survive the word-boundary match
 * anyway.
 */
const ENGLISH_ONLY: { key: string; english: string }[] = (
  [
    ['settings', enSettings, esSettings],
    ['common', enCommon, esCommon],
    ['errors', enErrors, esErrors],
    ['help', enHelp, esHelp],
  ] as [string, Json, Json][]
).flatMap(([ns, en, es]) => {
  const spanish = new Map(flatten(es));

  return flatten(en)
    .filter(([key, value]) => {
      const other = spanish.get(key);

      return (
        other !== undefined &&
        other !== value &&
        value.length >= 4 &&
        !value.includes('{{') &&
        // A three-letter word inside a longer Spanish one is a coincidence, not
        // a leak; the boundary match below cannot tell them apart.
        /[a-z]{3}/.test(value)
      );
    })
    .map(([key, value]) => ({ key: `${ns}:${key}`, english: value }));
});

/**
 * The mirror image: a key whose es value was never translated. It renders
 * English under `es` and the check above cannot see it, because en === es is
 * exactly the shape that check treats as a shared standard term. S1-14b found
 * nine of these across the pages, so the sections get the assertion too.
 *
 * Only values that actually reach the DOM count — the tree carries hundreds of
 * deliberately identical strings (SNMP, MTU, dBm, Ping) and listing them here
 * would be a second glossary to maintain.
 */
/**
 * Keys whose es value is the en value on purpose, and which the DNT list does
 * not cover because they are not fleet-wide standard terms: product and vendor
 * names, the language names themselves, and words Spanish spells identically.
 * Each one was read before it was added — the list is the audit, not an
 * exemption to reach for.
 */
const SHARED_BY_DESIGN = new Set([
  // Vendor and product names.
  'settings:sso.providers.google',
  'settings:sso.providers.microsoft',
  'settings:users.providers.google',
  'settings:users.providers.microsoft',
  'settings:vulnerability.databaseNVD',
  'settings:performance.speedtest',
  'help:portServices.postgresql',
  'settings:performance.iperf',
  // Spanish spells this the same way.
  'settings:health.endpoints',
  // Standard networking terms the glossary keeps verbatim but DNT_TERMS, which
  // is the fleet's list, does not carry.
  'common:interface.ethernet',
  'settings:interfaces.ethernet',
  'common:labels.autoNeg',
  'settings:discovery.traceroute',
  'settings:common.host',
  // Spanish spells these the same way.
  'settings:users.providers.local',
  // The language picker names each language in its own language.
  'settings:appearance.languageEn',
  'settings:appearance.languageEs',
]);

const UNTRANSLATED: { key: string; english: string }[] = (
  [
    ['settings', enSettings, esSettings],
    ['common', enCommon, esCommon],
    ['errors', enErrors, esErrors],
    ['help', enHelp, esHelp],
  ] as [string, Json, Json][]
).flatMap(([ns, en, es]) => {
  const spanish = new Map(flatten(es));

  return flatten(en)
    .filter(([key, value]) => {
      const other = spanish.get(key);

      return (
        other === value &&
        value.length >= 4 &&
        !value.includes('{{') &&
        // A value both locales share on purpose is a standard term. Anything
        // with a lowercase run of three letters is a word, not an acronym.
        /[a-z]{3}/.test(value) &&
        // The DNT exemption applies to term-like values only. A sentence is
        // untranslated however many acronyms it quotes — matching DNT against
        // prose exempted every paragraph in the tree, first because `IP` is a
        // substring of "equipment" and then, once that was fixed, because `ms`
        // is a whole word in "<20ms".
        (value.split(' ').length > 3 ||
          !DNT_TERMS.some((term) =>
            new RegExp(
              `(^|[^\\p{L}])${term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}($|[^\\p{L}])`,
              'iu',
            ).test(value),
          ))
      );
    })
    .map(([key, value]) => ({ key: `${ns}:${key}`, english: value }))
    .filter(({ key }) => !SHARED_BY_DESIGN.has(key));
});

/**
 * A single English word inside a Spanish sentence is usually a coincidence, not
 * a leak: `Active` appears inside "Active Directory" and `Gateway` inside the
 * translated "Ping de Gateway". So a one-word value only counts when it is the
 * whole of some text node — the way a label or a button actually renders. A
 * multi-word phrase is unambiguous and is matched anywhere in the section.
 */
function leakedEnglish(section: HTMLElement): string[] {
  const text = section.textContent ?? '';
  const nodes = new Set<string>();
  const walker = document.createTreeWalker(section, NodeFilter.SHOW_TEXT);
  for (let node = walker.nextNode(); node !== null; node = walker.nextNode()) {
    nodes.add((node.textContent ?? '').trim());
  }

  return ENGLISH_ONLY.filter(({ english }) =>
    english.includes(' ') ? text.includes(english) : nodes.has(english),
  ).map(({ key, english }) => `${key}: ${english}`);
}

/** Keys this section renders whose es value is still the English one. */
function untranslated(section: HTMLElement): string[] {
  const text = section.textContent ?? '';

  return UNTRANSLATED.filter(({ english }) => text.includes(english)).map(
    ({ key, english }) => `${key}: ${english}`,
  );
}

function asUser(role: CurrentUser['role']): void {
  mockGet.mockImplementation((path: string) => {
    if (path.includes('/users/me')) {
      return Promise.resolve({ username: 'u', role, isActive: true });
    }
    if (path.includes('/sso/settings')) {
      return Promise.resolve({ providers: [] });
    }

    return Promise.resolve({});
  });
}

/**
 * The fixture headers are English regexes, so the role-gate suite's
 * `findByRole({ name: header })` cannot open a section under `es` — which is
 * the whole point of this suite. Each section renders exactly one
 * `CollapsibleSection` and its header button is the first button in the tree,
 * so opening by position works in either language.
 */
async function openSection(container: HTMLElement): Promise<HTMLElement> {
  const button = container.querySelector('button');
  expect(button).not.toBeNull();
  await userEvent.click(button as HTMLButtonElement);

  return (button as HTMLButtonElement).closest('section') as HTMLElement;
}

beforeEach(() => {
  mockGet.mockReset();
  asUser('operator');
  vi.stubGlobal('fetch', (input: RequestInfo | URL) => {
    const url = String(input);
    const key = Object.keys(RAW_GET_BODIES).find((path) => url.includes(path));

    return Promise.resolve({
      ok: key !== undefined,
      status: key === undefined ? 404 : 200,
      json: () => Promise.resolve(key === undefined ? {} : RAW_GET_BODIES[key]),
    } as Response);
  });
});

afterEach(async () => {
  vi.unstubAllGlobals();
  vi.clearAllMocks();
  await i18n.changeLanguage('en');
});

const ALL_SECTIONS = [...SECTIONS, ...MIXED_SECTIONS, ...COPY_ONLY_SECTIONS];

describe.each(ALL_SECTIONS)('$name — real locale copy', ({ header, render: renderSection }) => {
  it('renders its English header', async () => {
    await i18n.changeLanguage('en');
    const { container } = render(
      <RoleProvider isAuthenticated={true}>{renderSection()}</RoleProvider>,
    );

    const section = await openSection(container);
    await waitFor(() => {
      expect(within(section).getByRole('button', { name: header })).toBeVisible();
    });

    expect(untranslated(section)).toEqual([]);
  });

  it('renders Spanish under es, with no English left behind', async () => {
    await i18n.changeLanguage('es');
    const { container } = render(
      <RoleProvider isAuthenticated={true}>{renderSection()}</RoleProvider>,
    );

    const section = await openSection(container);
    await waitFor(() => {
      expect(section.textContent?.trim().length ?? 0).toBeGreaterThan(0);
    });

    expect(leakedEnglish(section)).toEqual([]);
  });
});
