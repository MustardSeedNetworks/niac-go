import enHelp from '@locales/en/help.json';
import esHelp from '@locales/es/help.json';
import { act, cleanup, render, screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { GlossarySection } from '../components/help-drawer/GlossarySection';
import i18n from '../i18n';
import { getGlossary, glossaryCategories } from './glossary';

afterEach(async () => {
  cleanup();
  await i18n.changeLanguage('en');
});

describe.each(['en', 'es'] as const)('shared glossary — %s', (language) => {
  const catalog = language === 'en' ? enHelp : esHelp;
  it('has no missing or orphaned definitions', () => {
    const entries = getGlossary(i18n.getFixedT(language, 'help'));
    expect(Object.keys(catalog.glossary).sort((a, b) => a.localeCompare(b))).toEqual(
      Object.keys(glossaryCategories).sort((a, b) => a.localeCompare(b)),
    );
    expect(new Set(entries.map((entry) => entry.term)).size).toBe(entries.length);
    for (const entry of entries) {
      expect(entry.term.trim()).not.toBe('');
      expect(entry.definition.trim()).not.toBe('');
    }
    for (const key of ['scenarioPack', 'draft', 'session', 'fleet'] as const) {
      expect(entries).toContainEqual({ ...catalog.glossary[key], category: 'niac' });
    }
  });

  it('renders the same definition that inline explanations use', async () => {
    await Promise.resolve(
      act(async () => {
        await i18n.changeLanguage(language);
      }),
    );
    render(<GlossarySection searchQuery="BPF" />);
    const term = screen.getByText('BPF');
    expect(term.parentElement).not.toBeNull();
    expect(
      within(term.parentElement as HTMLElement).getByText(catalog.glossary.bpf.definition),
    ).toBeVisible();
    expect(screen.getByText(catalog.glossaryCategories.concept)).toBeVisible();
  });

  it('searches translated definitions and terms', async () => {
    await Promise.resolve(
      act(async () => {
        await i18n.changeLanguage(language);
      }),
    );
    render(<GlossarySection searchQuery={catalog.glossary.draft.term} />);
    expect(screen.getByText(catalog.glossary.draft.term)).toBeVisible();
    expect(screen.getByText(catalog.glossary.draft.definition)).toBeVisible();
    expect(screen.queryByText(catalog.glossary.arp.definition)).not.toBeInTheDocument();
  });
});
