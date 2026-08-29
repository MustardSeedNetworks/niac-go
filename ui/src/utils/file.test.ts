/**
 * Tests for the file/clipboard helpers.
 *
 * The clipboard helper has a real fallback path for browsers without the async
 * Clipboard API, and it is the branch that never runs in normal development —
 * so it is the one worth pinning.
 */

import { afterEach, describe, expect, it, vi } from 'vitest';
import { copyToClipboard, fileToBase64, fileToText } from './file';

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe('fileToText', () => {
  it('resolves with the file contents', async () => {
    await expect(fileToText(new File(['hello'], 'a.txt'))).resolves.toBe('hello');
  });

  it('rejects when the read fails', async () => {
    vi.spyOn(FileReader.prototype, 'readAsText').mockImplementation(function (this: FileReader) {
      this.onerror?.(new ProgressEvent('error') as ProgressEvent<FileReader>);
    });

    await expect(fileToText(new File(['x'], 'a.txt'))).rejects.toBeDefined();
  });
});

describe('fileToBase64', () => {
  it('strips the data URL prefix', async () => {
    const encoded = await fileToBase64(new File(['hello'], 'a.txt'));

    expect(encoded).toBe(btoa('hello'));
    expect(encoded).not.toContain(',');
  });

  it('returns the raw result when there is no data URL prefix', async () => {
    vi.spyOn(FileReader.prototype, 'readAsDataURL').mockImplementation(function (this: FileReader) {
      Object.defineProperty(this, 'result', { value: 'no-prefix', configurable: true });
      this.onload?.(new ProgressEvent('load') as ProgressEvent<FileReader>);
    });

    await expect(fileToBase64(new File(['x'], 'a.txt'))).resolves.toBe('no-prefix');
  });

  it('rejects with a descriptive error when the read fails', async () => {
    vi.spyOn(FileReader.prototype, 'readAsDataURL').mockImplementation(function (this: FileReader) {
      this.onerror?.(new ProgressEvent('error') as ProgressEvent<FileReader>);
    });

    await expect(fileToBase64(new File(['x'], 'a.txt'))).rejects.toThrow('Failed to read file');
  });
});

describe('copyToClipboard', () => {
  it('uses the async Clipboard API when available', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal('navigator', { clipboard: { writeText } });

    await copyToClipboard('copied');

    expect(writeText).toHaveBeenCalledWith('copied');
    // The fallback must not also run.
    expect(document.querySelectorAll('textarea')).toHaveLength(0);
  });

  it('falls back to execCommand when the Clipboard API is absent', async () => {
    vi.stubGlobal('navigator', {});
    const execCommand = vi.fn().mockReturnValue(true);
    Object.defineProperty(document, 'execCommand', {
      value: execCommand,
      configurable: true,
      writable: true,
    });

    await copyToClipboard('fallback');

    expect(execCommand).toHaveBeenCalledWith('copy');
    // The scratch textarea is cleaned up.
    expect(document.querySelectorAll('textarea')).toHaveLength(0);
  });
});
