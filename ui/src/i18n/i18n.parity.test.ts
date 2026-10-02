/**
 * i18n.parity.test.ts — locks en/es locale parity in CI.
 *
 * Asserts two invariants for every shipped namespace:
 *   1. KEY PARITY  — en and es JSON files have identical key sets at every
 *      depth. Adding or removing a key in one language without the other
 *      fails CI.
 *   2. DNT COMPLIANCE — every industry-standard "Do Not Translate" term
 *      (acronyms, RFC numbers, protocol names, metrics, units, product/module
 *      names) that appears in an en value must appear in the matching es
 *      value. Translating `throughput` to `rendimiento` or `latency` to
 *      `latencia` fails this gate.
 *
 * Match is case-insensitive so a term at the start of a sentence ("Latency")
 * still counts as the term ("latency"). Substring-based — sufficient for the
 * DNT list which is dominated by acronyms and stable noun forms.
 */

import { describe, expect, it } from 'vitest';

import { DNT_TERMS } from './dnt';
import { namespaces } from './index';

type Json = string | number | boolean | null | Json[] | { [k: string]: Json };

// Read from disk rather than listed by hand: a hand list skipped `pages`, the
// namespace every page title is drawn from (seed#2639). The Go side serves
// `api` and `validation`, which the UI never loads, so the runtime namespace
// list cannot stand in for this one either.
const EN = import.meta.glob<Json>('@locales/en/*.json', { eager: true, import: 'default' });
const ES = import.meta.glob<Json>('@locales/es/*.json', { eager: true, import: 'default' });

function byNamespace(files: Record<string, Json>): Map<string, Json> {
  return new Map(
    Object.entries(files).map(([path, json]) => [path.replace(/^.*\/(.+)\.json$/, '$1'), json]),
  );
}

const enFiles = byNamespace(EN);
const esFiles = byNamespace(ES);

const FIXTURES: { ns: string; en: Json; es: Json }[] = [...enFiles].flatMap(([ns, en]) => {
  const es = esFiles.get(ns);
  return es === undefined ? [] : [{ ns, en, es }];
});

const byName = (a: string, b: string) => a.localeCompare(b);

describe('i18n parity — locale files', () => {
  it('en and es ship the same namespace files', () => {
    expect([...esFiles.keys()].sort(byName)).toEqual([...enFiles.keys()].sort(byName));
  });

  it('every namespace the UI loads is under the parity gate', () => {
    const gated = new Set(FIXTURES.map(({ ns }) => ns));
    expect(namespaces.filter((ns) => !gated.has(ns))).toEqual([]);
  });
});

function flatKeyPaths(node: Json, prefix = ''): string[] {
  if (node === null || typeof node !== 'object') return [prefix];
  if (Array.isArray(node)) {
    return node.flatMap((v, i) => flatKeyPaths(v, `${prefix}[${i}]`));
  }
  return Object.entries(node).flatMap(([k, v]) =>
    flatKeyPaths(v, prefix === '' ? k : `${prefix}.${k}`),
  );
}

function flatStringEntries(node: Json, prefix = ''): [string, string][] {
  if (typeof node === 'string') return [[prefix, node]];
  if (node === null || typeof node !== 'object') return [];
  if (Array.isArray(node)) {
    return node.flatMap((v, i) => flatStringEntries(v, `${prefix}[${i}]`));
  }
  return Object.entries(node).flatMap(([k, v]) =>
    flatStringEntries(v, prefix === '' ? k : `${prefix}.${k}`),
  );
}

describe('i18n parity — en/es key sets', () => {
  for (const { ns, en, es } of FIXTURES) {
    it(`${ns}: identical key sets in en and es`, () => {
      const enK = new Set(flatKeyPaths(en));
      const esK = new Set(flatKeyPaths(es));
      const enOnly = [...enK].filter((k) => !esK.has(k)).sort((a, b) => a.localeCompare(b));
      const esOnly = [...esK].filter((k) => !enK.has(k)).sort((a, b) => a.localeCompare(b));
      expect(enOnly, `keys present in en but missing in es`).toEqual([]);
      expect(esOnly, `keys present in es but missing in en`).toEqual([]);
    });
  }
});

/** Word-boundary regex per DNT term — case-insensitive. Built once per term so
 * the per-string loop stays a constant-time membership check. Word boundaries
 * prevent false positives like "th[eir]" matching "EIR". */
const DNT_PATTERNS: { term: string; rx: RegExp }[] = DNT_TERMS.map((term) => ({
  term,
  // \b doesn't tokenize on dots/spaces, so for terms like "RFC 2544" we
  // anchor with a non-word lookaround instead.
  rx: new RegExp(`(?:^|[^\\w])${term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}(?:[^\\w]|$)`, 'i'),
}));

describe('i18n DNT — standard terms appear verbatim in es', () => {
  for (const { ns, en, es } of FIXTURES) {
    it(`${ns}: DNT terms in en values appear (case-insensitive) in matching es`, () => {
      const enMap = new Map(flatStringEntries(en));
      const esMap = new Map(flatStringEntries(es));
      const violations: string[] = [];
      for (const [path, enVal] of enMap) {
        const esVal = esMap.get(path);
        if (!esVal) continue;
        for (const { term, rx } of DNT_PATTERNS) {
          if (rx.test(enVal) && !rx.test(esVal)) {
            violations.push(`${path}: en has "${term}" but es does not`);
          }
        }
      }
      expect(violations).toEqual([]);
    });
  }
});

/**
 * Canonical spelling of Wi-Fi (seed#2296).
 *
 * The DNT check above cannot do this job. It asserts that a term present in an
 * en value survives into es — so writing "WiFi" in *both* locales removes the
 * term from en and the check has nothing left to enforce. Proved: reverting one
 * en value to "WiFi" left all 18 DNT cases green.
 *
 * Spelling needs its own assertion, and it is what made `Wi-Fi` addable to
 * DNT_TERMS at all: the tree used to carry both spellings, so neither could be
 * called canonical. "Wi-Fi" is the Wi-Fi Alliance's own, and it is what
 * pages.json and common.json already used.
 *
 * Identifiers are not copy: `wifiSettings`, `isWifi` and `WiFiCard` are code and
 * are deliberately out of scope. This reads locale *values* only.
 */
const NON_CANONICAL_WIFI = /(?:^|[^\w-])(WiFi|WIFI|Wifi|wi-fi)(?:[^\w-]|$)/;

describe('i18n spelling — Wi-Fi is spelled one way', () => {
  for (const { ns, en, es } of FIXTURES) {
    it(`${ns}: no locale value spells it any way but "Wi-Fi"`, () => {
      const offenders: string[] = [];
      for (const [locale, tree] of [
        ['en', en],
        ['es', es],
      ] as [string, Json][]) {
        for (const [path, value] of flatStringEntries(tree)) {
          const found = NON_CANONICAL_WIFI.exec(value);
          if (found) {
            offenders.push(`${locale}/${ns} ${path}: "${found[1]}" — write "Wi-Fi"`);
          }
        }
      }
      expect(offenders).toEqual([]);
    });
  }
});
