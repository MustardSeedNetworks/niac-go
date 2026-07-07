/**
 * ActionsMenu.test.tsx — the topology "Export ▾" dropdown.
 *
 * Guards the server-side DOT + GraphML export options (wired to
 * GET /api/v1/topology/export) alongside the existing PNG + JSON.
 */
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ActionsMenu } from './ActionsMenu';

describe('ActionsMenu — topology export', () => {
  const noop = () => undefined;

  const renderMenu = (overrides: Partial<Parameters<typeof ActionsMenu>[0]> = {}) => {
    const onExportDOT = vi.fn();
    const onExportGraphML = vi.fn();
    render(
      <ActionsMenu
        onExportPNG={noop}
        onExportJSON={noop}
        onExportDOT={onExportDOT}
        onExportGraphML={onExportGraphML}
        {...overrides}
      />,
    );
    return { onExportDOT, onExportGraphML };
  };

  const openMenu = () => fireEvent.click(screen.getByRole('button', { name: /export/i }));

  it('offers four export options: PNG, JSON, DOT, GraphML', () => {
    renderMenu();
    openMenu();
    expect(screen.getAllByRole('menuitem')).toHaveLength(4);
  });

  it('invokes the DOT export handler', () => {
    const { onExportDOT } = renderMenu();
    openMenu();
    // Order matches the menu: [0]=PNG, [1]=JSON, [2]=DOT, [3]=GraphML.
    fireEvent.click(screen.getAllByRole('menuitem')[2]);
    expect(onExportDOT).toHaveBeenCalledTimes(1);
  });

  it('invokes the GraphML export handler', () => {
    const { onExportGraphML } = renderMenu();
    openMenu();
    fireEvent.click(screen.getAllByRole('menuitem')[3]);
    expect(onExportGraphML).toHaveBeenCalledTimes(1);
  });

  it('does not render the menu when disabled and unopened', () => {
    renderMenu({ disabled: true });
    expect(screen.queryByRole('menuitem')).toBeNull();
  });
});
