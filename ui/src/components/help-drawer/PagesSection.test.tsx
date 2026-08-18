/**
 * The page help moved out of the registry's JSX and into data, so these
 * assert the two things that move could have broken: the drawer opens on the
 * page the (?) was clicked on, and the inline markers the prose relies on
 * (`code`, **strong**, links) still render as markup rather than literals.
 */
import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import '../../i18n';
import { pageHelp } from '../../data/page-help';
import { HelpDrawer } from '../HelpDrawer';
import { PageHelpBody } from './PageHelpBody';

describe('page help in the drawer', () => {
  it('opens on the page the header asked for', () => {
    render(<HelpDrawer isOpen={true} onClose={() => {}} section="/runtime" />);

    const drawer = within(screen.getByTestId('help-drawer'));
    expect(drawer.getByText(/Start, stop, and inspect the simulation/)).toBeTruthy();
  });

  it('re-applies the same page after the reader browses away and reopens', async () => {
    const { rerender } = render(<HelpDrawer isOpen={true} onClose={() => {}} section="/runtime" />);
    rerender(<HelpDrawer isOpen={false} onClose={() => {}} />);
    rerender(<HelpDrawer isOpen={true} onClose={() => {}} section="/runtime" />);

    expect(screen.getByText(/Start, stop, and inspect the simulation/)).toBeTruthy();
  });

  it('renders inline markers as markup, not as literal text', () => {
    render(
      <PageHelpBody
        blocks={[
          {
            kind: 'paragraph',
            text: 'Use `lo0` for **safe** testing — see [SECURITY.md](https://example.test/s).',
          },
        ]}
      />,
    );

    expect(screen.getByText('lo0').tagName).toBe('CODE');
    expect(screen.getByText('safe').tagName).toBe('STRONG');
    const link = screen.getByRole('link', { name: 'SECURITY.md' });
    expect(link.getAttribute('href')).toBe('https://example.test/s');
    expect(screen.queryByText(/\*\*/)).toBeNull();
  });

  it('carries no leftover markup from the JSX it was converted from', () => {
    const text = JSON.stringify(pageHelp);
    expect(text).not.toMatch(/<\/?(p|h4|ul|li|strong|code|em|a)\b/);
  });
});
