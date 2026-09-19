/**
 * ColoringRulesPanel.i18n.test.tsx / Sidebar — niac#2178: the icon-button
 * tooltips and the sidebar's collapse/menu controls were English literals in
 * files that already localize everything else, so a Spanish operator saw
 * "Move up" next to "Subir". The parity gate compares key sets, and the
 * hardcoded-JSX check reads text nodes, so neither could see an attribute.
 * These render both components under `es` and read the accessible names.
 */
import { render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import i18n from '../i18n';
import { MemoryDataRouter } from '../test/MemoryDataRouter';
import { SidebarLayout } from '../ui/Sidebar';
import { ColoringRulesPanel } from './ColoringRulesPanel';

const rules = [
  {
    id: 'rule-1',
    name: 'Regla',
    filter: 'tcp',
    foreground: '#ffffff',
    background: '#374151',
    enabled: true,
  },
];

const noop = () => {};

afterEach(async () => {
  await i18n.changeLanguage('en');
});

describe('ColoringRulesPanel in es', () => {
  it('localizes every icon-button control and both placeholders', async () => {
    await i18n.changeLanguage('es');
    render(<ColoringRulesPanel rules={rules} onRulesChange={noop} onReset={noop} onClose={noop} />);

    for (const label of ['Color del texto', 'Color de fondo', 'Subir', 'Bajar', 'Eliminar regla']) {
      expect(screen.getByLabelText(label), label).toBeInTheDocument();
    }
    expect(screen.getByPlaceholderText('Nombre')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Expresión de filtro')).toBeInTheDocument();
  });
});

describe('SidebarLayout in es', () => {
  it('localizes the collapse and mobile-menu controls', async () => {
    await i18n.changeLanguage('es');
    render(
      <MemoryDataRouter>
        <SidebarLayout groups={[]} version="0.0.0">
          <div />
        </SidebarLayout>
      </MemoryDataRouter>,
    );

    // The layout renders the desktop rail and the mobile drawer together, so
    // the collapse control appears in both trees.
    expect(screen.getAllByLabelText('Contraer barra lateral').length).toBeGreaterThan(0);
    expect(screen.getAllByLabelText('Abrir menú').length).toBeGreaterThan(0);
  });
});
