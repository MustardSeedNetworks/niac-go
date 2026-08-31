import { useCallback, useEffect, useRef, useState } from 'react';
import {
  buildRequestHeaders,
  buildUrl,
  notifyIfAuthenticationFailed,
  parseApiError,
} from '../api/requestCore';

export interface UseEventSourceOptions {
  url: string;
  onMessage?: (data: unknown) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
  enabled?: boolean;
}

export interface UseEventSourceResult {
  data: unknown;
  connected: boolean;
  error: Event | null;
  close: () => void;
  reconnect: () => void;
}

interface StreamCallbacks {
  onOpen: () => void;
  onMessage: (data: unknown) => void;
}

function parseEventRecord(record: string): unknown | undefined {
  const lines = record.split(/\r\n|\r|\n/);
  const event = lines
    .find((line) => line.startsWith('event:'))
    ?.slice(6)
    .trim();
  if (event && event !== 'message') return undefined;

  const data = lines
    .filter((line) => line.startsWith('data:'))
    // The grammar strips a single leading U+0020 after the colon, not all
    // leading whitespace: a second space, or a tab, belongs to the payload.
    .map((line) => line.slice(5).replace(/^ /, ''))
    .join('\n');
  if (!data) return undefined;

  try {
    return JSON.parse(data);
  } catch {
    return data;
  }
}

export async function consumeEventStream(
  url: string,
  signal: AbortSignal,
  callbacks: StreamCallbacks,
): Promise<void> {
  const headers = await buildRequestHeaders(url, { method: 'GET' });
  headers.set('Accept', 'text/event-stream');
  const response = await fetch(buildUrl(url), {
    headers,
    credentials: 'same-origin',
    signal,
  });
  if (!response.ok) {
    notifyIfAuthenticationFailed(response.status);
    throw parseApiError(await response.text(), response.status, response.statusText);
  }
  if (!response.body) {
    throw new Error('Event stream response did not include a body');
  }

  callbacks.onOpen();
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  while (!signal.aborted) {
    const { done, value } = await reader.read();
    buffer += decoder.decode(value, { stream: !done });

    let boundary = buffer.search(/\r\n\r\n|\r\r|\n\n/);
    while (boundary >= 0) {
      const delimiter = buffer.slice(boundary).match(/^(\r\n\r\n|\r\r|\n\n)/)?.[0];
      if (!delimiter) break;
      const parsed = parseEventRecord(buffer.slice(0, boundary));
      if (parsed !== undefined) callbacks.onMessage(parsed);
      buffer = buffer.slice(boundary + delimiter.length);
      boundary = buffer.search(/\r\n\r\n|\r\r|\n\n/);
    }
    if (done) break;
  }
}

export function useEventSource(options: UseEventSourceOptions): UseEventSourceResult {
  const { url, onMessage, onConnect, onDisconnect, onError, enabled = true } = options;
  const [data, setData] = useState<unknown>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<Event | null>(null);
  const controllerRef = useRef<AbortController | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const mountedRef = useRef(true);
  const callbacksRef = useRef({ onMessage, onConnect, onDisconnect, onError });

  useEffect(() => {
    callbacksRef.current = { onMessage, onConnect, onDisconnect, onError };
  }, [onMessage, onConnect, onDisconnect, onError]);

  const close = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    controllerRef.current?.abort();
    controllerRef.current = null;
    setConnected(false);
  }, []);

  const connect = useCallback(() => {
    if (!url || controllerRef.current) return;

    const controller = new AbortController();
    controllerRef.current = controller;
    void consumeEventStream(url, controller.signal, {
      onOpen: () => {
        if (!mountedRef.current || controller.signal.aborted) return;
        setConnected(true);
        setError(null);
        callbacksRef.current.onConnect?.();
      },
      onMessage: (message) => {
        if (!mountedRef.current || controller.signal.aborted) return;
        setData(message);
        callbacksRef.current.onMessage?.(message);
      },
    })
      .catch(() => {
        if (!mountedRef.current || controller.signal.aborted) return;
        const event = new Event('error');
        setError(event);
        callbacksRef.current.onError?.(event);
      })
      .finally(() => {
        if (controllerRef.current !== controller) return;
        controllerRef.current = null;
        if (!mountedRef.current || controller.signal.aborted) return;
        setConnected(false);
        callbacksRef.current.onDisconnect?.();
        reconnectTimerRef.current = setTimeout(connect, 1_000);
      });
  }, [url]);

  const reconnect = useCallback(() => {
    close();
    reconnectTimerRef.current = setTimeout(() => {
      if (mountedRef.current && enabled) connect();
    }, 100);
  }, [close, connect, enabled]);

  useEffect(() => {
    mountedRef.current = true;
    if (enabled && url) connect();
    return () => {
      mountedRef.current = false;
      close();
    };
  }, [enabled, url, connect, close]);

  return { data, connected, error, close, reconnect };
}

function getStreamBaseUrl(): string {
  return `${window.location.origin}/api/v1/stream`;
}

export interface StreamHookOptions {
  onMessage?: (data: unknown) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
  enabled?: boolean;
  sessionId?: string;
}

export interface PacketStreamEvent {
  type: 'packet';
  timestamp: string;
  data: Record<string, unknown>;
}

export function isPacketStreamEvent(value: unknown): value is PacketStreamEvent {
  if (typeof value !== 'object' || value === null) return false;
  const event = value as Record<string, unknown>;
  return (
    event.type === 'packet' &&
    typeof event.timestamp === 'string' &&
    typeof event.data === 'object' &&
    event.data !== null
  );
}

export function usePacketStream(options: StreamHookOptions = {}): UseEventSourceResult {
  const { enabled = true, sessionId, ...restOptions } = options;
  const query = sessionId ? `?sessionId=${encodeURIComponent(sessionId)}` : '';
  return useEventSource({
    url: enabled ? `${getStreamBaseUrl()}/packets${query}` : '',
    enabled,
    ...restOptions,
  });
}

export interface LogData {
  type: 'log';
  timestamp: string;
  level: 'debug' | 'info' | 'warn' | 'error';
  message: string;
}

export function useLogStream(options: StreamHookOptions = {}): UseEventSourceResult {
  const { enabled = true, ...restOptions } = options;
  return useEventSource({
    url: enabled ? `${getStreamBaseUrl()}/logs` : '',
    enabled,
    ...restOptions,
  });
}
