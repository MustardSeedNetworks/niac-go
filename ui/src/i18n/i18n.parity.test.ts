/**
 * i18n.parity.test.ts — locks en/es locale parity in CI.
 *
 * Asserts two invariants for every shipped namespace:
 *   1. KEY PARITY  — en and es JSON files have identical key sets at every
 *      depth. Adding or removing a key in one language without the other
 *      fails CI.
 *   2. DNT COMPLIANCE — every industry-standard "Do Not Translate" term
 *      (acronyms, RFC numbers, protocol names, metrics, units) that appears
 *      in an en value must appear in the matching es value. Translating
 *      `throughput` to `rendimiento` or `latency` to `latencia` fails this
 *      gate.
 *
 * Match is case-insensitive (so a term at the start of a sentence still
 * counts) and word-boundary anchored (so `EIR` doesn't match `their`).
 *
 * Mirrors the seed/stem copy — each repo owns its own per the harmonization
 * convention. Product/module names are excluded because they collide with
 * common English vocabulary.
 */

import enCommon from '@locales/en/common.json';
import enDevices from '@locales/en/devices.json';
import enErrors from '@locales/en/errors.json';
import enHelp from '@locales/en/help.json';
import enPages from '@locales/en/pages.json';
import enProtocols from '@locales/en/protocols.json';
import enSettings from '@locales/en/settings.json';
import esCommon from '@locales/es/common.json';
import esDevices from '@locales/es/devices.json';
import esErrors from '@locales/es/errors.json';
import esHelp from '@locales/es/help.json';
import esPages from '@locales/es/pages.json';
import esProtocols from '@locales/es/protocols.json';
import esSettings from '@locales/es/settings.json';
import { describe, expect, it } from 'vitest';

type Json = string | number | boolean | null | Json[] | { [k: string]: Json };

const FIXTURES: { ns: string; en: Json; es: Json }[] = [
  { ns: 'common', en: enCommon as Json, es: esCommon as Json },
  { ns: 'devices', en: enDevices as Json, es: esDevices as Json },
  { ns: 'errors', en: enErrors as Json, es: esErrors as Json },
  { ns: 'help', en: enHelp as Json, es: esHelp as Json },
  { ns: 'pages', en: enPages as Json, es: esPages as Json },
  { ns: 'protocols', en: enProtocols as Json, es: esProtocols as Json },
  { ns: 'settings', en: enSettings as Json, es: esSettings as Json },
];

/** Standard terms that must NEVER be translated. */
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
  'TCP',
  'UDP',
  'ICMP',
  'LLDP',
  'CDP',
  'TLS',
  'SSH',
  // Capture / replay formats & tooling kept verbatim
  'PCAP',
  'PCAPNG',
  'BPF',
  'JSON',
  'CSV',
  // Metrics, abbreviations, units
  'SNR',
  'FLR',
  'FDV',
  'CIR',
  'EIR',
  'Mbps',
  'Gbps',
  'pps',
  'dBm',
  'MTU',
  'MAC',
  'jitter',
  'throughput',
  'latency',
  // Domain nouns kept verbatim app-wide (the Library "Walks" section, the
  // "Walk Analyzer", device-editor "walk file"). es must not render these as
  // "recorrido". Both forms: singular "walk"/"Walk" and plural "walks"/"Walks".
  'walk',
  'Walks',
  // 802.1Q term of art. Most es strings already kept it verbatim ("puertos
  // trunk", "trunk/LAG", "trunk_ports"); two had drifted to "troncal", which
  // reads inconsistently beside the untranslated "VLAN" in the same controls.
  // Matching is case-insensitive, so this one entry covers "Trunk" too.
  'trunk',
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

const DNT_PATTERNS: { term: string; rx: RegExp }[] = DNT_TERMS.map((term) => ({
  term,
  rx: new RegExp(`(?:^|[^\\w])${term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}(?:[^\\w]|$)`, 'i'),
}));

describe('i18n parity — en/es key sets', () => {
  for (const { ns, en, es } of FIXTURES) {
    it(`${ns}: identical key sets in en and es`, () => {
      const enK = new Set(flatKeyPaths(en));
      const esK = new Set(flatKeyPaths(es));
      const enOnly = [...enK]
        .filter((k) => !esK.has(k))
        .sort((left, right) => left.localeCompare(right));
      const esOnly = [...esK]
        .filter((k) => !enK.has(k))
        .sort((left, right) => left.localeCompare(right));
      expect(enOnly, 'keys present in en but missing in es').toEqual([]);
      expect(esOnly, 'keys present in es but missing in en').toEqual([]);
    });
  }
});

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
