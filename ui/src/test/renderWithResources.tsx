import { type RenderOptions, render } from '@testing-library/react';
import type { ReactNode } from 'react';
import { ResourceProvider } from '../contexts/ResourceProvider';

export function renderWithResources(ui: ReactNode, options: RenderOptions = {}) {
  const Wrapper = options.wrapper;
  return render(ui, {
    ...options,
    wrapper: ({ children }) => (
      <ResourceProvider>{Wrapper ? <Wrapper>{children}</Wrapper> : children}</ResourceProvider>
    ),
  });
}
