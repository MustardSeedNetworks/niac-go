import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { type FC, useRef } from 'react';
import { MemoryRouter, Route, Routes, useLocation, useNavigate } from 'react-router';
import { describe, expect, it } from 'vitest';
import { useFocusOnRouteChange } from './useFocusOnRouteChange';

/**
 * Mirrors the real shell: the heading lives inside a subtree keyed on the
 * pathname (App's PageWithErrorBoundary), so every navigation unmounts the
 * old <h1> and mounts a new one. A hook that captured the node once would
 * pass a naive test and focus a detached element here.
 */
const Shell: FC = () => {
  const titleRef = useRef<HTMLHeadingElement>(null);
  const { pathname } = useLocation();
  const navigate = useNavigate();

  useFocusOnRouteChange(titleRef);

  return (
    <>
      <button type="button" onClick={() => navigate('/devices')}>
        go to devices
      </button>
      <div key={pathname}>
        <h1 ref={titleRef} tabIndex={-1}>
          {pathname}
        </h1>
      </div>
    </>
  );
};

const renderShell = () =>
  render(
    <MemoryRouter initialEntries={['/']}>
      <Routes>
        <Route path="*" element={<Shell />} />
      </Routes>
    </MemoryRouter>,
  );

describe('useFocusOnRouteChange', () => {
  it('leaves focus alone on the first render', () => {
    renderShell();

    expect(document.activeElement).toBe(document.body);
  });

  it('moves focus to the heading of the page navigated to', async () => {
    const user = userEvent.setup();
    renderShell();

    await user.click(screen.getByRole('button', { name: 'go to devices' }));

    const heading = screen.getByRole('heading', { name: '/devices' });
    expect(document.activeElement).toBe(heading);
  });

  it('focuses the freshly mounted heading, not the unmounted one', async () => {
    const user = userEvent.setup();
    renderShell();
    const first = screen.getByRole('heading', { name: '/' });

    await user.click(screen.getByRole('button', { name: 'go to devices' }));

    expect(first.isConnected).toBe(false);
    expect(document.activeElement).not.toBe(first);
    expect(document.activeElement).toBe(screen.getByRole('heading', { name: '/devices' }));
  });
});
