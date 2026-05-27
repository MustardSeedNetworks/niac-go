/**
 * i18n init tests — smoke-test the resource loading + language
 * detection wiring so a botched namespace addition or a
 * malformed locale JSON gets caught in CI rather than in production.
 */

import i18n from 'i18next';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { defaultNs, languages, namespaces } from './index';

describe('i18n configuration', () => {
  it('declares the expected set of supported languages', () => {
    const codes = languages.map((l) => l.code);
    expect(codes).toEqual(['en', 'es']);
  });

  it('declares the expected set of namespaces', () => {
    expect([...namespaces]).toEqual([
      'common',
      'devices',
      'errors',
      'help',
      'pages',
      'protocols',
      'settings',
    ]);
  });

  it('uses common as the default namespace', () => {
    expect(defaultNs).toBe('common');
  });

  it('loads EN resources for every declared namespace', () => {
    for (const ns of namespaces) {
      const bundle = i18n.getResourceBundle('en', ns);
      expect(bundle, `EN bundle missing for ${ns}`).toBeTruthy();
      expect(Object.keys(bundle).length, `EN bundle empty for ${ns}`).toBeGreaterThan(0);
    }
  });

  it('loads ES resources for every declared namespace', () => {
    for (const ns of namespaces) {
      const bundle = i18n.getResourceBundle('es', ns);
      expect(bundle, `ES bundle missing for ${ns}`).toBeTruthy();
      expect(Object.keys(bundle).length, `ES bundle empty for ${ns}`).toBeGreaterThan(0);
    }
  });
});

describe('i18n <html lang> sync', () => {
  const originalLanguage = i18n.language;
  const originalDocLang = document.documentElement.lang;

  afterEach(async () => {
    await i18n.changeLanguage(originalLanguage);
    document.documentElement.lang = originalDocLang;
  });

  it('updates document.documentElement.lang when language changes', async () => {
    await i18n.changeLanguage('es');
    expect(document.documentElement.lang).toBe('es');

    await i18n.changeLanguage('en');
    expect(document.documentElement.lang).toBe('en');
  });
});

describe('i18n key resolution', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en');
  });

  it('resolves a top-level common key', () => {
    expect(i18n.t('common:buttons.save')).toBe('Save');
  });

  it('resolves nested namespace.section.key paths', () => {
    expect(i18n.t('settings:tabs.simulation')).toBe('Simulation');
  });

  it('resolves a key with interpolation', () => {
    expect(i18n.t('common:plurals.deviceCount', { count: 5 })).toBe('5 devices');
    expect(i18n.t('common:plurals.deviceCount', { count: 1 })).toBe('1 device');
  });

  it('flips translations when language changes', async () => {
    expect(i18n.t('common:buttons.save')).toBe('Save');
    await i18n.changeLanguage('es');
    expect(i18n.t('common:buttons.save')).toBe('Guardar');
  });
});
