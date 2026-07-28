import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

describe('scenario draft client', () => {
  beforeEach(() => {
    vi.resetModules();
    mockFetch.mockReset();
  });

  it('creates and reads drafts through the protected library boundary', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ token: 'csrf-token' }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: () =>
          Promise.resolve({
            name: 'campus-draft',
            content: 'devices: []\n',
            format: 'yaml',
            revision: 'revision-1',
            modifiedAt: '2026-07-28T12:00:00Z',
            sizeBytes: 12,
          }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () =>
          Promise.resolve({
            name: 'campus-draft',
            content: 'devices: []\n',
            format: 'yaml',
            revision: 'revision-1',
            modifiedAt: '2026-07-28T12:00:00Z',
            sizeBytes: 12,
          }),
      });

    const { createScenarioDraft, fetchScenarioDraft } = await import('./library-client');
    await createScenarioDraft('campus-draft', 'devices: []\n');
    await fetchScenarioDraft('campus-draft');

    expect(mockFetch.mock.calls[1][0]).toContain('/api/v1/library/drafts');
    expect(mockFetch.mock.calls[1][1]).toMatchObject({
      method: 'POST',
      body: JSON.stringify({ name: 'campus-draft', content: 'devices: []\n' }),
    });
    expect(mockFetch.mock.calls[2][0]).toContain('/api/v1/library/drafts/campus-draft');
    expect(mockFetch.mock.calls[2][1]).toMatchObject({ credentials: 'same-origin' });
  });

  it('sends the current revision when replacing and deleting a draft', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ token: 'csrf-token' }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () =>
          Promise.resolve({
            name: 'campus-draft',
            content: 'devices:\n  - name: router-1\n',
            format: 'yaml',
            revision: 'revision-2',
            modifiedAt: '2026-07-28T12:01:00Z',
            sizeBytes: 28,
          }),
      })
      .mockResolvedValueOnce({ ok: true, status: 204 });

    const { deleteScenarioDraft, replaceScenarioDraft } = await import('./library-client');
    await replaceScenarioDraft('campus-draft', 'revision-1', 'devices:\n  - name: router-1\n');
    await deleteScenarioDraft('campus-draft', 'revision-2');

    const replaceHeaders = mockFetch.mock.calls[1][1]?.headers as Headers;
    expect(mockFetch.mock.calls[1][1]).toMatchObject({ method: 'PUT' });
    expect(replaceHeaders.get('If-Match')).toBe('"revision-1"');

    const deleteHeaders = mockFetch.mock.calls[2][1]?.headers as Headers;
    expect(mockFetch.mock.calls[2][1]).toMatchObject({ method: 'DELETE' });
    expect(deleteHeaders.get('If-Match')).toBe('"revision-2"');
  });

  it('creates template drafts without flattening resources in the browser', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ token: 'csrf-token' }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: () =>
          Promise.resolve({
            name: 'switch-draft',
            content: 'devices:\n  - name: switch-1\n',
            format: 'yaml',
            revision: 'revision-1',
            modifiedAt: '2026-07-28T12:00:00Z',
            sizeBytes: 29,
          }),
      });

    const { createScenarioDraftFromTemplate } = await import('./library-client');
    await createScenarioDraftFromTemplate('switch-draft', 'catalyst-9300-48p');

    expect(mockFetch.mock.calls[1][1]).toMatchObject({
      method: 'POST',
      body: JSON.stringify({
        name: 'switch-draft',
        templateName: 'catalyst-9300-48p',
      }),
    });
  });
});
