import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import '../../i18n';
import { ApiError } from '../../api/errors';
import { installContentBundleWithProgress } from '../../api/library-client';
import { useUIStore } from '../../stores/ui-store';
import { ContentBundleUploader } from './ContentBundleUploader';

vi.mock('../../api/library-client', () => ({
  installContentBundleWithProgress: vi.fn(),
}));

function makeFile(name: string, size: number): File {
  return new File([new Uint8Array(Math.max(size, 0))], name, { type: 'application/gzip' });
}

describe('ContentBundleUploader', () => {
  beforeEach(() => {
    vi.mocked(installContentBundleWithProgress).mockReset();
    useUIStore.setState({ notifications: [] });
  });

  it('rejects a file with the wrong extension', () => {
    render(<ContentBundleUploader />);
    const input = screen.getByTestId('content-bundle-file-input') as HTMLInputElement;

    fireEvent.change(input, { target: { files: [makeFile('bundle.zip', 100)] } });

    expect(screen.getByTestId('content-bundle-error')).toHaveTextContent('Invalid file type');
    expect(screen.getByTestId('content-bundle-install')).toBeDisabled();
  });

  it('uploads a valid bundle, reports progress, and shows a success toast with the manifest summary', async () => {
    vi.mocked(installContentBundleWithProgress).mockImplementation(async (_payload, onProgress) => {
      onProgress(50);
      onProgress(100);
      return {
        success: true,
        files: 3,
        directories: 3,
        bytes: 1234,
        perKind: { networks: 1, walks: 1, pcaps: 1 },
        message: 'Content bundle installed',
      };
    });

    render(<ContentBundleUploader />);
    const input = screen.getByTestId('content-bundle-file-input') as HTMLInputElement;
    fireEvent.change(input, { target: { files: [makeFile('bundle.tar.gz', 1024)] } });

    const installButton = screen.getByTestId('content-bundle-install');
    expect(installButton).not.toBeDisabled();
    fireEvent.click(installButton);

    await waitFor(() => expect(installContentBundleWithProgress).toHaveBeenCalledTimes(1));

    await waitFor(() => {
      const notifications = useUIStore.getState().notifications;
      expect(notifications).toHaveLength(1);
      expect(notifications[0]).toMatchObject({
        type: 'success',
        title: 'Content bundle installed',
      });
      expect(notifications[0].message).toContain('1 networks, 1 walks, 1 pcaps');
    });

    // Selection resets after a successful install.
    expect(screen.queryByText('bundle.tar.gz', { exact: false })).not.toBeInTheDocument();
  });

  it('surfaces a structured error message via toast on failure', async () => {
    vi.mocked(installContentBundleWithProgress).mockRejectedValue(
      new ApiError('bundle is corrupt', 400, 'bundle_invalid'),
    );

    render(<ContentBundleUploader />);
    const input = screen.getByTestId('content-bundle-file-input') as HTMLInputElement;
    fireEvent.change(input, { target: { files: [makeFile('bundle.tar.gz', 1024)] } });
    fireEvent.click(screen.getByTestId('content-bundle-install'));

    await waitFor(() => {
      const notifications = useUIStore.getState().notifications;
      expect(notifications).toHaveLength(1);
      expect(notifications[0].type).toBe('error');
      expect(notifications[0].message).toContain('bundle is corrupt');
    });
  });
});
