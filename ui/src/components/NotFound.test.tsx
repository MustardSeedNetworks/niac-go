import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router';
import { describe, expect, it } from 'vitest';
import '../i18n';
import { NotFound } from './NotFound';

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/" element={<p>dashboard</p>} />
        <Route path="*" element={<NotFound />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('NotFound', () => {
  it('names NIAC, says the page was not found, and links home', () => {
    renderAt('/no/such/page');

    expect(screen.getByRole('heading', { name: 'Page not found' })).toBeVisible();
    expect(screen.getByTestId('not-found')).toHaveTextContent('NIAC has no page at this address');
    expect(screen.getByRole('link', { name: 'Back to the dashboard' })).toHaveAttribute(
      'href',
      '/',
    );
  });

  it('leaves the unknown URL in place instead of redirecting', () => {
    renderAt('/no/such/page');

    expect(screen.queryByText('dashboard')).not.toBeInTheDocument();
  });
});
