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

import enApi from '@locales/en/api.json';
import enCards from '@locales/en/cards.json';
import enCommon from '@locales/en/common.json';
import enErrors from '@locales/en/errors.json';
import enGlossary from '@locales/en/glossary.json';
import enHelp from '@locales/en/help.json';
import enSettings from '@locales/en/settings.json';
import enSetup from '@locales/en/setup.json';
import enSurvey from '@locales/en/survey.json';
import enValidation from '@locales/en/validation.json';
import esApi from '@locales/es/api.json';
import esCards from '@locales/es/cards.json';
import esCommon from '@locales/es/common.json';
import esErrors from '@locales/es/errors.json';
import esGlossary from '@locales/es/glossary.json';
import esHelp from '@locales/es/help.json';
import esSettings from '@locales/es/settings.json';
import esSetup from '@locales/es/setup.json';
import esSurvey from '@locales/es/survey.json';
import esValidation from '@locales/es/validation.json';
import { describe, expect, it } from 'vitest';

type Json = string | number | boolean | null | Json[] | { [k: string]: Json };

const FIXTURES: { ns: string; en: Json; es: Json }[] = [
  { ns: 'api', en: enApi as Json, es: esApi as Json },
  { ns: 'cards', en: enCards as Json, es: esCards as Json },
  { ns: 'common', en: enCommon as Json, es: esCommon as Json },
  { ns: 'errors', en: enErrors as Json, es: esErrors as Json },
  { ns: 'glossary', en: enGlossary as Json, es: esGlossary as Json },
  { ns: 'help', en: enHelp as Json, es: esHelp as Json },
  { ns: 'settings', en: enSettings as Json, es: esSettings as Json },
  { ns: 'setup', en: enSetup as Json, es: esSetup as Json },
  { ns: 'survey', en: enSurvey as Json, es: esSurvey as Json },
  { ns: 'validation', en: enValidation as Json, es: esValidation as Json },
];

/**
 * Standard terms that must NEVER be translated. Acronyms / RFC numbers /
 * protocol names / metric names / units / product+module names. Keep aligned
 * with the cross-repo memory: feedback_no_translate_standard_terms.
 */
const DNT_TERMS = [
  // Standards
  'RFC 2544',
  'Y.1564',
  'Y.1731',
  'RFC 2889',
  'RFC 6349',
  'MEF',
  'TSN',
  // Protocols & acronyms
  'ARP',
  'DHCP',
  'DNS',
  'BGP',
  'OSPF',
  'SNMP',
  'VLAN',
  'WebSocket',
  // Metrics, abbreviations, units
  'SNR',
  'FLR',
  'FDV',
  'CIR',
  'EIR',
  'Mbps',
  'dBm',
  'jitter',
  'throughput',
  'latency',
  // Product / modules
  'Roots',
  'Canopy',
  'Shell',
  'Sap',
  'Harvest',
  'Seed',
  'Stem',
  'NIAC',
];

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
      const enOnly = [...enK].filter((k) => !esK.has(k)).sort();
      const esOnly = [...esK].filter((k) => !enK.has(k)).sort();
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
