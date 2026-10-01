/**
 * The phone drawer mounts its rail only while open (#2284). Parked off-canvas
 * it kept every nav link, Help, Settings and a second product mark in the
 * tree; `inert` and `invisible` hid them, but the shell still carried a menu
 * nobody could see, which seed's drawer stopped doing under UI-SEED-10.
 */
import { fireEvent, render, screen, within } from '@testing-library/react';
import { Boxes } from 'lucide-react';
import { describe, expect, it } from 'vitest';
import { MemoryDataRouter } from '../test/MemoryDataRouter';
import { SidebarLayout } from './Sidebar';

function renderLayout() {
  render(
    <MemoryDataRouter>
      <SidebarLayout
        groups={[
          { label: 'Simulate', items: [{ path: '/devices', label: 'Devices', icon: Boxes }] },
        ]}
        version="0.0.0"
        onOpenHelp={() => {}}
        onOpenSettings={() => {}}
      >
        <div />
      </SidebarLayout>
    </MemoryDataRouter>,
  );
}

describe('SidebarLayout phone drawer', () => {
  it('holds no control and no product mark while closed', () => {
    renderLayout();
    const drawer = screen.getByTestId('sidebar-mobile');

    expect(within(drawer).queryAllByRole('button')).toHaveLength(0);
    expect(within(drawer).queryByTestId('product-mark')).toBeNull();
  });

  it('mounts the rail on open and drops it again on close', () => {
    renderLayout();
    const drawer = screen.getByTestId('sidebar-mobile');
    const toggle = screen.getByTestId('mobile-menu-toggle');

    fireEvent.click(toggle);
    expect(within(drawer).getByTestId('nav-item-devices')).toBeInTheDocument();
    expect(within(drawer).getByTestId('sidebar-help-button')).toBeInTheDocument();
    expect(within(drawer).getByTestId('product-mark')).toBeInTheDocument();

    fireEvent.click(toggle);
    expect(within(drawer).queryAllByRole('button')).toHaveLength(0);
  });

  it('closes the drawer when a nav item inside it is chosen', () => {
    renderLayout();
    const drawer = screen.getByTestId('sidebar-mobile');

    fireEvent.click(screen.getByTestId('mobile-menu-toggle'));
    fireEvent.click(within(drawer).getByTestId('nav-item-devices'));

    expect(within(drawer).queryByTestId('nav-item-devices')).toBeNull();
  });
});
