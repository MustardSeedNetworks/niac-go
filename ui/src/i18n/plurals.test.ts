/**
 * Plural forms are selected at runtime, not just present in the catalogues.
 *
 * The i18n gate proves `_one` and `_other` exist in both locales; this proves
 * i18next picks the right one for each count. It replaces an E2E assertion
 * that could only see a pluralised count while a simulation happened to be
 * running, which made it a test of daemon state as much as of translation.
 */
import { describe, expect, it } from 'vitest';
import i18n from './index';

const CASES = [{ ns: 'pages', key: 'segments.deviceNoun', en: [/^device$/, /^devices$/] }] as const;

describe('plural selection', () => {
  it('picks singular and plural forms in English', async () => {
    await i18n.changeLanguage('en');

    for (const { ns, key, en } of CASES) {
      expect(i18n.t(key, { ns, count: 1 })).toMatch(en[0]);
      expect(i18n.t(key, { ns, count: 3 })).toMatch(en[1]);
    }
  });

  it('picks singular and plural forms in Spanish', async () => {
    await i18n.changeLanguage('es');

    for (const { ns, key } of CASES) {
      const one = i18n.t(key, { ns, count: 1 });
      const many = i18n.t(key, { ns, count: 3 });

      // Not asserting exact copy — only that both resolve to real strings and
      // that the two counts do not collapse onto the same form.
      expect(one).not.toContain(key);
      expect(many).not.toContain(key);
      expect(one).not.toBe(many);
    }

    await i18n.changeLanguage('en');
  });
});
