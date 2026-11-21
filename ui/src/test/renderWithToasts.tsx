import { type RenderOptions, render } from '@testing-library/react';
import type { ReactElement, ReactNode } from 'react';
import { ToastProvider } from '../ui/Toast';

/**
 * render() with the ToastProvider every toast-raising component expects,
 * for tests that mount a page or component outside the App shell.
 */
export function renderWithToasts(ui: ReactElement, options: RenderOptions = {}) {
  const Wrapper = options.wrapper;
  return render(ui, {
    ...options,
    wrapper: ({ children }: { children: ReactNode }) => (
      <ToastProvider>{Wrapper ? <Wrapper>{children}</Wrapper> : children}</ToastProvider>
    ),
  });
}
