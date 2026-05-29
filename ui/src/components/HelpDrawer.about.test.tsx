/**
 * HelpDrawer.about.test.tsx — asserts the in-app About/version panel exists.
 *
 * The drawer header shows `NIAC v<version>` sourced from /__version via
 * useBuildVersion (which mirrors the seed/stem hook). This test fails if that
 * surface disappears.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { HelpDrawer } from './HelpDrawer';

describe('HelpDrawer — About / version surface', () => {
  it('renders the version badge using /__version payload', async () => {
    const payload = {
      version: '0.88.4',
      commit: 'deadbee',
      buildTime: '2026-05-29T00:00:00Z',
      uiBuildHash: 'abc123',
    };
    vi.spyOn(globalThis, 'fetch').mockImplementation((async () => ({
      ok: true,
      json: async () => payload,
    })) as unknown as typeof fetch);

    render(<HelpDrawer isOpen={true} onClose={() => undefined} />);
    const badge = await screen.findByTestId('help-drawer-version');
    expect(badge).toBeInTheDocument();
    await waitFor(() => {
      expect(badge).toHaveTextContent(/v0\.88\.4/);
    });
    expect(badge).toHaveTextContent(/NIAC/);
  });

  it('renders a fallback version when /__version is unreachable', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation((async () => {
      throw new Error('network');
    }) as unknown as typeof fetch);

    render(<HelpDrawer isOpen={true} onClose={() => undefined} />);
    const badge = await screen.findByTestId('help-drawer-version');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveTextContent(/NIAC v/);
  });
});
