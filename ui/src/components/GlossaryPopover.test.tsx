import enHelp from '@locales/en/help.json';
import esHelp from '@locales/es/help.json';
import { cleanup, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it } from 'vitest';
import { getGlossary } from '../data/glossary';
import i18n from '../i18n';
import { GlossaryPopover } from './GlossaryPopover';

const inlineTerms = [
  'bpf',
  'fiveTuple',
  'librarySource',
  'walkProvenance',
  'pcap',
  'snmpWalk',
  'oid',
] as const;

afterEach(async () => {
  cleanup();
  await i18n.changeLanguage('en');
});

describe.each(['en', 'es'] as const)('inline glossary definitions — %s', (language) => {
  it.each(inlineTerms)('%s uses the Help glossary heading and definition', async (term) => {
    await i18n.changeLanguage(language);
    const catalog = language === 'en' ? enHelp : esHelp;
    const definition = catalog.glossary[term];
    const glossaryEntry = getGlossary(i18n.getFixedT(language, 'help')).find(
      (entry) => entry.term === definition.term,
    );
    expect(glossaryEntry).toMatchObject(definition);
    const user = userEvent.setup();
    render(<GlossaryPopover term={term} />);
    await user.tab();
    const trigger = screen.getByRole('button', {
      name: i18n.getFixedT(language, 'common')('jargon.ariaLabel', { term: definition.term }),
    });
    expect(trigger).toHaveFocus();
    await user.keyboard('{Enter}');
    const dialog = screen.getByRole('dialog', { name: definition.term });
    expect(dialog).toHaveAccessibleDescription(definition.definition);
    expect(within(dialog).getByText(definition.definition)).toBeVisible();
    await user.keyboard('{Escape}');
    expect(trigger).toHaveFocus();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});
