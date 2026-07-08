import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import '../../i18n';
import { PcapUploader } from './PcapUploader';

function makeFile(name: string, size: number, type = 'application/octet-stream'): File {
  const file = new File([new Uint8Array(Math.max(size, 0))], name, { type });
  return file;
}

describe('PcapUploader', () => {
  it('surfaces a validation error for a rejected file type via file input', () => {
    const onFileSelect = vi.fn();
    const onValidationError = vi.fn();

    const { container } = render(
      <PcapUploader
        onFileSelect={onFileSelect}
        onAnalyze={vi.fn()}
        onValidationError={onValidationError}
        isAnalyzing={false}
        selectedFile={null}
        error={null}
        success={null}
        uploadProgress={null}
      />,
    );

    const input = container.querySelector('input[type="file"]') as HTMLInputElement;
    const badFile = makeFile('not-a-pcap.txt', 100);

    // fireEvent bypasses userEvent's accept-attribute filtering — a real
    // user can still select a non-matching file via "All Files" in the OS
    // picker or by dragging one in, which is exactly what validateFile guards.
    fireEvent.change(input, { target: { files: [badFile] } });

    expect(onValidationError).toHaveBeenCalledWith(expect.stringContaining('Invalid file type'));
    expect(onFileSelect).not.toHaveBeenCalled();
  });

  it('surfaces a validation error for an oversized file', async () => {
    const user = userEvent.setup();
    const onFileSelect = vi.fn();
    const onValidationError = vi.fn();

    const { container } = render(
      <PcapUploader
        onFileSelect={onFileSelect}
        onAnalyze={vi.fn()}
        onValidationError={onValidationError}
        isAnalyzing={false}
        selectedFile={null}
        error={null}
        success={null}
        uploadProgress={null}
      />,
    );

    const input = container.querySelector('input[type="file"]') as HTMLInputElement;
    const bigFile = makeFile('capture.pcap', 100 * 1024 * 1024 + 1);

    await user.upload(input, bigFile);

    expect(onValidationError).toHaveBeenCalledWith(expect.stringContaining('too large'));
    expect(onFileSelect).not.toHaveBeenCalled();
  });

  it('displays the error prop when set', () => {
    render(
      <PcapUploader
        onFileSelect={vi.fn()}
        onAnalyze={vi.fn()}
        onValidationError={vi.fn()}
        isAnalyzing={false}
        selectedFile={null}
        error="Invalid file type. Please select a PCAP file (.pcap, .pcapng, .cap)"
        success={null}
        uploadProgress={null}
      />,
    );

    expect(screen.getByText(/Invalid file type/)).toBeInTheDocument();
  });

  it('renders a determinate progress bar while uploadProgress is set', () => {
    const { rerender } = render(
      <PcapUploader
        onFileSelect={vi.fn()}
        onAnalyze={vi.fn()}
        onValidationError={vi.fn()}
        isAnalyzing={true}
        selectedFile={makeFile('capture.pcap', 1024)}
        error={null}
        success={null}
        uploadProgress={42}
      />,
    );

    expect(screen.getByTestId('pcap-upload-progress')).toBeInTheDocument();
    const bar = screen.getByTestId('pcap-upload-progress-bar');
    expect(bar).toHaveStyle({ width: '42%' });
    expect(screen.getByText(/42%/)).toBeInTheDocument();

    // Upload finished (or hasn't started): no progress bar.
    rerender(
      <PcapUploader
        onFileSelect={vi.fn()}
        onAnalyze={vi.fn()}
        onValidationError={vi.fn()}
        isAnalyzing={false}
        selectedFile={makeFile('capture.pcap', 1024)}
        error={null}
        success={null}
        uploadProgress={null}
      />,
    );

    expect(screen.queryByTestId('pcap-upload-progress')).not.toBeInTheDocument();
  });

  it('displays the friendly over-limit message when the server rejects an oversized upload', () => {
    render(
      <PcapUploader
        onFileSelect={vi.fn()}
        onAnalyze={vi.fn()}
        onValidationError={vi.fn()}
        isAnalyzing={false}
        selectedFile={null}
        error="This capture is too large for the server to accept (limit: 140.0 MB). Please choose a smaller file."
        success={null}
        uploadProgress={null}
      />,
    );

    expect(screen.getByText(/too large for the server to accept/)).toBeInTheDocument();
    expect(screen.getByText(/140\.0 MB/)).toBeInTheDocument();
  });
});
