/**
 * Tests for the SSE stream consumer behind useEventSource.
 *
 * The frame parser is the interesting part: it has to split on all three
 * line-ending conventions, join multi-line `data:` fields, ignore named events,
 * and cope with a record arriving split across two network chunks. Every one of
 * those is a silent-data-loss path — a dropped event looks like "the daemon
 * didn't send anything", not like a bug.
 */

import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { consumeEventStream, useEventSource } from './useEventSource';

/** Serves `chunks` as an SSE body; each chunk is delivered as its own read. */
function mockStream(chunks: string[], init: ResponseInit = {}): void {
  vi.spyOn(globalThis, 'fetch').mockImplementation(() => {
    const encoder = new TextEncoder();
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        for (const chunk of chunks) {
          controller.enqueue(encoder.encode(chunk));
        }
        controller.close();
      },
    });
    return Promise.resolve(new Response(body, { status: 200, ...init }));
  });
}

/** Collects every message consumeEventStream emits for the given chunks. */
async function collect(chunks: string[]): Promise<unknown[]> {
  mockStream(chunks);
  const messages: unknown[] = [];
  await consumeEventStream('/api/v1/events', new AbortController().signal, {
    onOpen: () => {},
    onMessage: (m) => messages.push(m),
  });
  return messages;
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe('frame parsing', () => {
  it('parses a JSON data payload', async () => {
    expect(await collect(['data: {"a":1}\n\n'])).toEqual([{ a: 1 }]);
  });

  it('falls back to the raw string when the payload is not JSON', async () => {
    expect(await collect(['data: hello\n\n'])).toEqual(['hello']);
  });

  it('joins multi-line data fields with newlines', async () => {
    expect(await collect(['data: one\ndata: two\n\n'])).toEqual(['one\ntwo']);
  });

  it('splits records on LF, CRLF and CR boundaries alike', async () => {
    expect(await collect(['data: a\n\ndata: b\r\n\r\ndata: c\r\r'])).toEqual(['a', 'b', 'c']);
  });

  it('reassembles a record split across two chunks', async () => {
    // The boundary is the point at which a naive per-chunk parser loses data.
    expect(await collect(['data: {"a":', '1}\n\n'])).toEqual([{ a: 1 }]);
  });

  it('emits several records from one chunk', async () => {
    expect(await collect(['data: 1\n\ndata: 2\n\ndata: 3\n\n'])).toEqual([1, 2, 3]);
  });

  it('ignores a named event that is not "message"', async () => {
    expect(await collect(['event: ping\ndata: 1\n\n'])).toEqual([]);
  });

  it('accepts an explicit "message" event', async () => {
    expect(await collect(['event: message\ndata: 1\n\n'])).toEqual([1]);
  });

  it('ignores comment-only and empty records', async () => {
    expect(await collect([': keep-alive\n\n', 'data:\n\n'])).toEqual([]);
  });

  it('strips all leading whitespace after the colon, not just one space', async () => {
    // The SSE grammar strips a single leading U+0020; parseEventRecord uses
    // trimStart(), so 'data:  a' yields 'a' rather than ' a'. Only observable
    // for payloads that are deliberately space-indented, which ours are not.
    expect(await collect(['data:  a\n\n'])).toEqual(['a']);
  });

  it('drops a trailing partial record rather than emitting half of it', async () => {
    expect(await collect(['data: complete\n\ndata: partial'])).toEqual(['complete']);
  });
});

describe('stream failures', () => {
  it('throws when the response is not ok', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response('nope', { status: 500, statusText: 'Server Error' }),
    );

    await expect(
      consumeEventStream('/api/v1/events', new AbortController().signal, {
        onOpen: () => {},
        onMessage: () => {},
      }),
    ).rejects.toBeDefined();
  });

  it('throws when the response carries no body', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 204 }));

    await expect(
      consumeEventStream('/api/v1/events', new AbortController().signal, {
        onOpen: () => {},
        onMessage: () => {},
      }),
    ).rejects.toThrow(/did not include a body/);
  });

  it('does not signal open when the request fails', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('x', { status: 403 }));
    const onOpen = vi.fn();

    await expect(
      consumeEventStream('/api/v1/events', new AbortController().signal, {
        onOpen,
        onMessage: () => {},
      }),
    ).rejects.toBeDefined();
    expect(onOpen).not.toHaveBeenCalled();
  });
});

describe('useEventSource', () => {
  it('connects, reports data and invokes the callbacks', async () => {
    mockStream(['data: {"n":1}\n\n']);
    const onConnect = vi.fn();
    const onMessage = vi.fn();

    const { result } = renderHook(() =>
      useEventSource({ url: '/api/v1/events', onConnect, onMessage }),
    );

    await waitFor(() => expect(result.current.data).toEqual({ n: 1 }));
    expect(onConnect).toHaveBeenCalled();
    expect(onMessage).toHaveBeenCalledWith({ n: 1 });
  });

  it('does not connect when disabled', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch');

    renderHook(() => useEventSource({ url: '/api/v1/events', enabled: false }));

    await Promise.resolve();
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it('does not connect without a url', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch');

    renderHook(() => useEventSource({ url: '' }));

    await Promise.resolve();
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it('reports an error when the stream fails', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('x', { status: 500 }));
    const onError = vi.fn();

    const { result } = renderHook(() => useEventSource({ url: '/api/v1/events', onError }));

    await waitFor(() => expect(result.current.error).not.toBeNull());
    expect(onError).toHaveBeenCalled();
    expect(result.current.connected).toBe(false);
  });

  it('close() marks the stream disconnected', async () => {
    // A stream that stays open: a closing stream would flip `connected` back to
    // false on its own and the assertion would pass without close() doing it.
    vi.spyOn(globalThis, 'fetch').mockImplementation(() => {
      const encoder = new TextEncoder();
      const body = new ReadableStream<Uint8Array>({
        start(controller) {
          controller.enqueue(encoder.encode('data: 1\n\n'));
          // deliberately never closed
        },
      });
      return Promise.resolve(new Response(body, { status: 200 }));
    });
    const { result } = renderHook(() => useEventSource({ url: '/api/v1/events' }));

    await waitFor(() => expect(result.current.connected).toBe(true));
    act(() => {
      result.current.close();
    });

    expect(result.current.connected).toBe(false);
  });
});
