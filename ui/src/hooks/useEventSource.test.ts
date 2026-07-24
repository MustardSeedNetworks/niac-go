import { beforeEach, describe, expect, it, vi } from 'vitest';
import { clearRuntimeAPIToken, setRuntimeAPIToken } from '../api/requestCore';
import { consumeEventStream } from './useEventSource';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

describe('consumeEventStream', () => {
  beforeEach(() => {
    mockFetch.mockReset();
    clearRuntimeAPIToken();
  });

  it('authenticates the SSE fetch and parses records split across chunks', async () => {
    const encoder = new TextEncoder();
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(encoder.encode('event: connected\r\ndata: {"stream":"logs"}\r\n\r'));
        controller.enqueue(
          encoder.encode('\ndata: {"type":"log","message":"ready"}\n\ndata: plain text\n\n'),
        );
        controller.close();
      },
    });
    mockFetch.mockResolvedValueOnce(
      new Response(body, { status: 200, headers: { 'Content-Type': 'text/event-stream' } }),
    );
    setRuntimeAPIToken('stream-token');
    const messages: unknown[] = [];

    await consumeEventStream('/api/v1/stream/logs', new AbortController().signal, {
      onOpen: vi.fn(),
      onMessage: (message) => messages.push(message),
    });

    const headers = mockFetch.mock.calls[0][1]?.headers as Headers;
    expect(headers.get('Authorization')).toBe('Bearer stream-token');
    expect(headers.get('Accept')).toBe('text/event-stream');
    expect(messages).toEqual([{ type: 'log', message: 'ready' }, 'plain text']);
  });

  it('raises the shared authentication event on a 401', async () => {
    mockFetch.mockResolvedValueOnce(new Response('Unauthorized', { status: 401 }));
    const listener = vi.fn();
    window.addEventListener('niac:authentication-required', listener);

    await expect(
      consumeEventStream('/api/v1/stream/logs', new AbortController().signal, {
        onOpen: vi.fn(),
        onMessage: vi.fn(),
      }),
    ).rejects.toThrow('Unauthorized');

    expect(listener).toHaveBeenCalledOnce();
    window.removeEventListener('niac:authentication-required', listener);
  });
});
