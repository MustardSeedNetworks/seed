/**
 * locale-integrity.i18n.test.ts — the locale files themselves, asserted.
 *
 * #1942: wrecking both locale trees fails only a handful of tests, because
 * the suite asserts on testids and on English hardcoded in components. The
 * component tests beside this one fix that surface by surface; this covers the
 * failure the component tests cannot see, which is copy that never made it
 * into `es` at all.
 *
 * A key that is missing from `es` does not throw — i18next falls back to `en`
 * and the screen renders an English word in a Spanish sentence. That is
 * invisible to a single-locale assertion and to a testid one.
 *
 * The list is user-facing copy, not the whole tree: this is a floor that
 * grows as surfaces get covered, not a claim about every string.
 */
import { describe, expect, it } from 'vitest';

import { DNT_TERMS } from './dnt';
import i18n, { languages, namespaces } from './index';

/**
 * A value made only of do-not-translate terms is correctly identical in both
 * locales. The list is shared with i18n.parity.test.ts rather than restated,
 * so the two suites cannot disagree about what counts as a standard term.
 */
function isAllStandardTerms(value: string): boolean {
  const words = value.split(/[^\w.]+/).filter(Boolean);

  return (
    words.length > 0 &&
    words.every((w) => DNT_TERMS.some((t) => t.toLowerCase() === w.toLowerCase()))
  );
}

/** Copy a user reads on the main surfaces. */
const TRANSLATED = [
  'cards:network.title',
  'cards:network.mac',
  'cards:network.dns',
  'cards:network.ipv4',
  'cards:network.ipv6',
  'cards:dns.title',
  'cards:wifi.bssid',
  'cards:network.mode',
  'cards:network.address',
  'cards:network.gateway',
  'cards:network.lease',
  'cards:gateway.title',
  'cards:gateway.packets',
  'cards:gateway.packetLoss',
  'cards:gateway.reachable',
  'cards:gateway.unreachable',
  'cards:dns.servers',
  'cards:dns.dnsServers',
  'cards:dns.responseTime',
  'cards:dns.resolving',
  'cards:performance.latency',
  'settings:discovery.targetNetworks',
  'settings:network.title',
];

function resolve(lng: string, key: string): string {
  const separator = key.indexOf(':');
  const ns = key.slice(0, separator);
  const path = key.slice(separator + 1);

  // The typed getFixedT wants a Namespace literal and a key from the generated
  // union. These are data, resolved at run time, so both are reached through
  // the untyped signature on purpose -- that is the point of the test.
  const getFixedT = i18n.getFixedT as unknown as (l: string, n: string) => (k: string) => string;

  return getFixedT(lng, ns)(path);
}

/** Every string leaf under a locale bundle, with its dotted path. */
function walk(node: unknown, path: string, visit: (path: string, value: string) => void): void {
  if (typeof node === 'string') {
    visit(path, node);

    return;
  }
  if (node !== null && typeof node === 'object') {
    for (const [key, child] of Object.entries(node)) {
      walk(child, `${path}.${key}`, visit);
    }
  }
}

describe('locale integrity', () => {
  it('resolves every listed key in both locales', () => {
    for (const key of TRANSLATED) {
      for (const lng of ['en', 'es']) {
        const value = resolve(lng, key);
        expect(value, `${key} did not resolve under ${lng}`).not.toBe(
          key.slice(key.indexOf(':') + 1),
        );
        expect(value.trim(), `${key} is empty under ${lng}`).not.toBe('');
      }
    }
  });

  it('leaves no English behind in Spanish on the covered surfaces', () => {
    const untranslated = TRANSLATED.filter((key) => {
      const en = resolve('en', key);

      return en === resolve('es', key) && !isAllStandardTerms(en);
    });

    expect(untranslated, 'these keys render English on a Spanish screen').toEqual([]);
  });

  /**
   * seed#2642: three gate sentences carried `&lt;KEY&gt;` in the JSON so that
   * `<Trans>` would not read `<KEY>` as a tag. It does not need the help — it
   * renders an unmatched element as its text — and React prints the entity
   * verbatim, so the copy-pasteable command on Reports, Path Analysis and the
   * API-token panel read `seed license activate -k &lt;KEY&gt;` on screen.
   *
   * The three page suites assert their own sentence whole; this asserts the
   * shape across every namespace in both locales, because the next one will
   * not be on a page that has a suite.
   */
  it('escapes no HTML entity in any locale value', () => {
    const escaped: string[] = [];

    for (const lng of languages) {
      for (const ns of namespaces) {
        const bundle = i18n.getResourceBundle(lng.code, ns) as unknown;
        walk(bundle, `${lng.code}:${ns}`, (path, value) => {
          if (/&(lt|gt|amp|quot|#\d+);/.test(value)) {
            escaped.push(`${path} = ${value}`);
          }
        });
      }
    }

    expect(escaped, 'these values print an HTML entity as text').toEqual([]);
  });
});
