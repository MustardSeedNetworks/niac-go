/**
 * FTPSection.test.tsx
 *
 * Data-safety regression test: the per-user FTP password field must render
 * as a masked `type="password"` input, never plaintext `type="text"`. It
 * previously rendered as plaintext, exposing configured FTP credentials to
 * shoulder-surfing and browser autofill/history in the same way a login
 * password field would be.
 */
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { Device } from '../../api/types';
import { FtpSection } from './FTPSection';

const baseDevice: Device = {
  hostname: 'edge-01',
  mac: '00:1A:2B:3C:4D:5E',
  type: 'server',
  ftp: {
    enabled: true,
    systemType: 'UNIX Type: L8',
    users: [{ username: 'anonymous', password: 'hunter2', homeDir: '/' }],
  },
};

describe('FtpSection', () => {
  it('renders the FTP user password field as a masked password input', () => {
    render(
      <FtpSection device={baseDevice} isExpanded={true} onToggle={vi.fn()} onUpdate={vi.fn()} />,
    );

    const passwordInput = screen.getByPlaceholderText('Password') as HTMLInputElement;
    expect(passwordInput.type).toBe('password');
    expect(passwordInput.autocomplete).toBe('new-password');
  });

  it('does not render the password value as plaintext', () => {
    render(
      <FtpSection device={baseDevice} isExpanded={true} onToggle={vi.fn()} onUpdate={vi.fn()} />,
    );

    expect(screen.queryByDisplayValue('hunter2')).not.toBeNull();
    expect(screen.getByPlaceholderText('Password')).toHaveAttribute('type', 'password');
  });
});
