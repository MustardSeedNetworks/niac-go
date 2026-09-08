/**
 * ScopeContext tests — pin the canWrite/isAdmin matrix and the
 * fail-closed loading/error state so the #762 UI gate can't silently
 * regress. Also exercise the wrapper components in their viewer +
 * operator paths.
 */

import { act, render, renderHook, waitFor } from '@testing-library/react';
import { useEffect } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ReadOnlyView } from '../components/ui/ReadOnlyView';
import { RequireScope } from '../components/ui/RequireScope';
import { type Scope, ScopeProvider, useScope } from './ScopeContext';

const mockGet = vi.fn<(path: string) => Promise<unknown>>();
vi.mock('../api/requestCore', () => ({
  deduplicatedGet: (path: string): Promise<unknown> => mockGet(path),
}));

const wrapper = ({ children }: { children: React.ReactNode }): React.ReactElement => (
  <ScopeProvider>{children}</ScopeProvider>
);

const respond = (scope: Scope): { scope: Scope } => ({ scope });

describe('ScopeContext / useScope', () => {
  beforeEach(() => {
    mockGet.mockReset();
  });

  it('fetches /api/v1/auth/scope on mount and exposes scope + helpers', async () => {
    mockGet.mockResolvedValueOnce(respond('read-write'));
    const { result } = renderHook(() => useScope(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(mockGet).toHaveBeenCalledWith('/api/v1/auth/scope');
    expect(result.current.scope).toBe('read-write');
    expect(result.current.canWrite).toBe(true);
    expect(result.current.isAdmin).toBe(false);
  });

  it('read-only token cannot write and is not admin', async () => {
    mockGet.mockResolvedValueOnce(respond('read-only'));
    const { result } = renderHook(() => useScope(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.canWrite).toBe(false);
    expect(result.current.isAdmin).toBe(false);
  });

  it('admin token can write and is admin', async () => {
    mockGet.mockResolvedValueOnce(respond('admin'));
    const { result } = renderHook(() => useScope(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.canWrite).toBe(true);
    expect(result.current.isAdmin).toBe(true);
  });

  it('fails closed on fetch error: canWrite=false, error surfaced', async () => {
    mockGet.mockRejectedValueOnce(new Error('boom'));
    const { result } = renderHook(() => useScope(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.scope).toBeNull();
    expect(result.current.canWrite).toBe(false);
    expect(result.current.isAdmin).toBe(false);
    expect(result.current.error).toBe('boom');
  });

  it('refresh() re-fetches /auth/scope and updates state', async () => {
    mockGet.mockResolvedValueOnce(respond('read-only'));
    const { result } = renderHook(() => useScope(), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.canWrite).toBe(false);

    mockGet.mockResolvedValueOnce(respond('admin'));
    act(() => {
      void result.current.refresh();
    });
    await waitFor(() => expect(result.current.isAdmin).toBe(true));
  });

  it('throws when used outside <ScopeProvider>', () => {
    const orig = console.error;
    console.error = (): void => {};
    try {
      expect(() => renderHook(() => useScope())).toThrow(/inside <ScopeProvider>/);
    } finally {
      console.error = orig;
    }
  });
});

describe('<RequireScope>', () => {
  beforeEach(() => {
    mockGet.mockReset();
  });

  it('renders children when scope meets minimum', async () => {
    mockGet.mockResolvedValueOnce(respond('admin'));
    const { findByText } = render(
      <ScopeProvider>
        <RequireScope min="admin" fallback={<span>nope</span>}>
          <span>admin-only</span>
        </RequireScope>
      </ScopeProvider>,
    );
    await findByText('admin-only');
  });

  it('renders fallback when scope is below minimum', async () => {
    mockGet.mockResolvedValueOnce(respond('read-write'));
    const { findByText } = render(
      <ScopeProvider>
        <RequireScope min="admin" fallback={<span>nope</span>}>
          <span>admin-only</span>
        </RequireScope>
      </ScopeProvider>,
    );
    await findByText('nope');
  });
});

describe('<ReadOnlyView>', () => {
  beforeEach(() => {
    mockGet.mockReset();
  });

  it('leaves the page unbannered and usable for a read-write token', async () => {
    mockGet.mockResolvedValueOnce(respond('read-write'));
    const { container, findByTestId, queryByRole } = render(
      <ScopeProvider>
        <ReadOnlyView>
          <input data-testid="probe" />
        </ReadOnlyView>
      </ScopeProvider>,
    );
    const probe = await findByTestId('probe');
    await waitFor(() => expect(probe).not.toBeDisabled());
    expect(queryByRole('status')).toBeNull();
    // The fieldset stays in the tree on purpose (#1941) -- it is what makes
    // the two scopes the same shape -- so what matters is that it is not
    // disabled and the wrapper is laid out as if it were not there.
    expect(container.querySelector('fieldset')?.hasAttribute('disabled')).toBe(false);
    expect(container.firstElementChild?.className).toContain('contents');
  });

  it('does not remount the page when the scope arrives (#1941)', async () => {
    let grantWrite: (() => void) | undefined;
    mockGet.mockReturnValueOnce(
      new Promise((resolve) => {
        grantWrite = () => resolve(respond('admin'));
      }),
    );

    let mounts = 0;
    function Page(): React.ReactElement {
      useEffect(() => {
        mounts += 1;
      }, []);
      return (
        <button data-testid="probe" type="button">
          start
        </button>
      );
    }

    const { findByTestId } = render(
      <ScopeProvider>
        <ReadOnlyView>
          <Page />
        </ReadOnlyView>
      </ScopeProvider>,
    );
    const probe = await findByTestId('probe');
    expect(mounts).toBe(1);

    // Every session is read-only for as long as /auth/scope takes to answer.
    // Swapping the tree shape when it answered remounted the page below and
    // discarded the file, interface or wizard step already chosen. The control
    // going live is the scope arriving; the banner is not, because it is
    // withheld while the fetch is in flight.
    act(() => grantWrite?.());
    await waitFor(() => expect(probe).not.toBeDisabled());

    expect(mounts).toBe(1);
  });

  it('does not claim read-only before the scope has been read', async () => {
    mockGet.mockReturnValueOnce(new Promise(() => {}));
    const { findByTestId, queryByRole } = render(
      <ScopeProvider>
        <ReadOnlyView>
          <input data-testid="probe" />
        </ReadOnlyView>
      </ScopeProvider>,
    );
    // Fail-closed still holds: the control is disabled. What the app cannot
    // yet say is why.
    expect(await findByTestId('probe')).toBeDisabled();
    expect(queryByRole('status')).toBeNull();
  });

  it('wraps children in a disabled fieldset and banners for read-only', async () => {
    mockGet.mockResolvedValueOnce(respond('read-only'));
    const { findByTestId, findByRole, container } = render(
      <ScopeProvider>
        <ReadOnlyView>
          <input data-testid="probe-input" />
          <button data-testid="probe-button" type="button">
            Save
          </button>
        </ReadOnlyView>
      </ScopeProvider>,
    );
    await findByRole('status');
    const fieldset = container.querySelector('fieldset');
    expect(fieldset).not.toBeNull();
    expect(fieldset?.hasAttribute('disabled')).toBe(true);
    const input = await findByTestId('probe-input');
    const button = await findByTestId('probe-button');
    expect(fieldset?.contains(input)).toBe(true);
    expect(fieldset?.contains(button)).toBe(true);
  });
});
