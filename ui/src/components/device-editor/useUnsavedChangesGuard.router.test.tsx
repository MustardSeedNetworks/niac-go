import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { Search } from 'lucide-react';
import { useState } from 'react';
import { createMemoryRouter, Link, RouterProvider, useNavigate } from 'react-router';
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest';
import { CommandPalette } from '../../ui/CommandPalette';
import { useUnsavedChangesGuard } from './useUnsavedChangesGuard';
import '../../i18n';

function Editor() {
  const [dirty, setDirty] = useState(true);
  const [selected, setSelected] = useState('A');
  const [paletteOpen, setPaletteOpen] = useState(false);
  const navigate = useNavigate();
  const guard = useUnsavedChangesGuard(dirty);
  return (
    <>
      <CommandPalette
        open={paletteOpen}
        onOpenChange={setPaletteOpen}
        groups={[
          { label: 'Navigation', items: [{ path: '/next', label: 'Next page', icon: Search }] },
        ]}
      />
      <span data-testid="selection">{selected}</span>
      <Link to="/next">Sidebar</Link>
      <button type="button" onClick={() => navigate('/next')}>
        Command
      </button>
      <button type="button" onClick={() => guard.requestAction(() => setSelected('B'))}>
        Select B
      </button>
      <button type="button" onClick={() => setDirty(false)}>
        Save
      </button>
      {guard.pending && (
        <div role="dialog">
          <button type="button" onClick={guard.confirmNavigate}>
            Discard
          </button>
          <button type="button" onClick={guard.cancelNavigate}>
            Cancel
          </button>
        </div>
      )}
    </>
  );
}

function setup(index = 1) {
  const router = createMemoryRouter(
    [
      { path: '/editor', element: <Editor /> },
      { path: '/previous', element: <p>Previous</p> },
      { path: '/next', element: <p>Next</p> },
    ],
    { initialEntries: ['/previous', '/editor', '/next'], initialIndex: index },
  );
  render(<RouterProvider router={router} />);
  return router;
}

describe('unsaved changes with the application router', () => {
  const scrollIntoView = Object.getOwnPropertyDescriptor(Element.prototype, 'scrollIntoView');
  beforeAll(() => {
    Object.defineProperty(Element.prototype, 'scrollIntoView', {
      configurable: true,
      value: vi.fn(),
    });
  });
  afterAll(() => {
    if (scrollIntoView) Object.defineProperty(Element.prototype, 'scrollIntoView', scrollIntoView);
    else Reflect.deleteProperty(Element.prototype, 'scrollIntoView');
  });
  it('guards the real command palette selection', async () => {
    const router = setup();
    fireEvent.keyDown(document, { key: 'k', ctrlKey: true });
    fireEvent.click(await screen.findByRole('option', { name: /Next page/ }));
    expect(await screen.findByRole('dialog')).toHaveTextContent('Discard');
    expect(router.state.location.pathname).toBe('/editor');
    fireEvent.click(screen.getByText('Cancel'));
    expect(router.state.location.pathname).toBe('/editor');
  });
  it.each([-1, 1])('blocks history navigation by %i until discard', async (delta) => {
    const router = setup();
    void router.navigate(delta);
    expect(await screen.findByRole('dialog')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/editor');
    fireEvent.click(screen.getByText('Cancel'));
    expect(router.state.location.pathname).toBe('/editor');
    void router.navigate(delta);
    await screen.findByRole('dialog');
    fireEvent.click(screen.getByText('Discard'));
    await waitFor(() =>
      expect(router.state.location.pathname).toBe(delta === -1 ? '/previous' : '/next'),
    );
  });
  it.each(['Sidebar', 'Command'])(
    'guards %s navigation without click interception',
    async (label) => {
      const router = setup();
      fireEvent.click(screen.getByText(label));
      expect(await screen.findByRole('dialog')).toBeInTheDocument();
      fireEvent.click(screen.getByText('Discard'));
      await waitFor(() => expect(router.state.location.pathname).toBe('/next'));
    },
  );
});
