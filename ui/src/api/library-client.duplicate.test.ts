import { beforeEach, describe, expect, it, vi } from 'vitest';
import { duplicateLibraryNetwork } from './library-client';
import { requestJson } from './requestCore';

vi.mock('./requestCore', () => ({
  deduplicatedGet: vi.fn(),
  request: vi.fn(),
  requestJson: vi.fn(),
  requestJsonCamelCase: vi.fn(),
}));

describe('duplicateLibraryNetwork', () => {
  beforeEach(() => {
    vi.mocked(requestJson).mockReset();
  });

  it('posts copyFrom to the networks route so the daemon makes the copy', async () => {
    vi.mocked(requestJson).mockResolvedValue({ success: true, name: 'campus-copy' });

    await expect(duplicateLibraryNetwork({ copyFrom: 'campus' })).resolves.toEqual({
      success: true,
      name: 'campus-copy',
    });
    expect(requestJson).toHaveBeenCalledExactlyOnceWith(
      '/api/v1/library/networks',
      { copyFrom: 'campus' },
      { method: 'POST' },
    );
  });
});
