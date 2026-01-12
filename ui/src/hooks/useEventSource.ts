import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Options for configuring the EventSource connection
 */
export interface UseEventSourceOptions {
  /** EventSource URL to connect to */
  url: string;
  /** Callback when a message is received (after JSON parsing) */
  onMessage?: (data: unknown) => void;
  /** Callback when connection is established */
  onConnect?: () => void;
  /** Callback when connection is closed/errored */
  onDisconnect?: () => void;
  /** Callback when an error occurs */
  onError?: (error: Event) => void;
  /** Whether the connection is enabled (default: true) */
  enabled?: boolean;
}

/**
 * Result returned by the useEventSource hook
 */
export interface UseEventSourceResult {
  /** Last received data (JSON parsed) */
  data: unknown;
  /** Whether the EventSource is currently connected */
  connected: boolean;
  /** Last error event, if any */
  error: Event | null;
  /** Manually close the connection */
  close: () => void;
  /** Manually reconnect */
  reconnect: () => void;
}

/**
 * EventSource hook with auto-connect and JSON handling
 *
 * Benefits over WebSocket:
 * - Automatic reconnection built into browser API
 * - Simpler API (server → client only)
 * - Better proxy/CDN compatibility
 * - Works well with HTTP/2 multiplexing
 *
 * @param options - EventSource configuration options
 * @returns EventSource state and control methods
 *
 * @example
 * ```typescript
 * const { data, connected } = useEventSource({
 *   url: '/api/v1/stream/logs',
 *   onMessage: (data) => console.log('Received:', data),
 * });
 * ```
 */
export function useEventSource(options: UseEventSourceOptions): UseEventSourceResult {
  const { url, onMessage, onConnect, onDisconnect, onError, enabled = true } = options;

  const [data, setData] = useState<unknown>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<Event | null>(null);

  const eventSourceRef = useRef<EventSource | null>(null);
  const mountedRef = useRef(true);

  // Store callbacks in refs to avoid re-creating effects
  const onMessageRef = useRef(onMessage);
  const onConnectRef = useRef(onConnect);
  const onDisconnectRef = useRef(onDisconnect);
  const onErrorRef = useRef(onError);

  useEffect(() => {
    onMessageRef.current = onMessage;
  }, [onMessage]);
  useEffect(() => {
    onConnectRef.current = onConnect;
  }, [onConnect]);
  useEffect(() => {
    onDisconnectRef.current = onDisconnect;
  }, [onDisconnect]);
  useEffect(() => {
    onErrorRef.current = onError;
  }, [onError]);

  /**
   * Connect to the EventSource
   */
  const connect = useCallback(() => {
    if (!url || eventSourceRef.current) return;

    try {
      const es = new EventSource(url);
      eventSourceRef.current = es;

      es.onopen = () => {
        if (!mountedRef.current) return;
        setConnected(true);
        setError(null);
        onConnectRef.current?.();
      };

      es.onmessage = (event: MessageEvent) => {
        if (!mountedRef.current) return;

        try {
          const parsedData = JSON.parse(event.data);
          setData(parsedData);
          onMessageRef.current?.(parsedData);
        } catch {
          // If JSON parsing fails, use raw data
          setData(event.data);
          onMessageRef.current?.(event.data);
        }
      };

      es.onerror = (event: Event) => {
        if (!mountedRef.current) return;

        // EventSource automatically reconnects on error
        // We just update state to reflect disconnection
        setError(event);
        setConnected(false);
        onErrorRef.current?.(event);
        onDisconnectRef.current?.();
      };

      // Handle custom 'connected' event from server
      es.addEventListener("connected", () => {
        // Event received - connection confirmed
      });
    } catch {
      setError(new Event("error"));
    }
  }, [url]);

  /**
   * Close the EventSource connection
   */
  const close = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
      setConnected(false);
    }
  }, []);

  /**
   * Manually reconnect
   */
  const reconnect = useCallback(() => {
    close();
    // Small delay before reconnecting
    setTimeout(() => {
      if (mountedRef.current && enabled) {
        connect();
      }
    }, 100);
  }, [close, connect, enabled]);

  // Connect on mount, cleanup on unmount
  useEffect(() => {
    mountedRef.current = true;

    if (enabled && url) {
      connect();
    }

    return () => {
      mountedRef.current = false;
      close();
    };
  }, [enabled, url, connect, close]);

  return {
    data,
    connected,
    error,
    close,
    reconnect,
  };
}

// ============================================================================
// Specialized Stream Hooks
// ============================================================================

/**
 * Get the API base URL for streams
 */
function getStreamBaseUrl(): string {
  return `${window.location.origin}/api/v1/stream`;
}

/**
 * Options for specialized stream hooks
 */
export interface StreamHookOptions {
  /** Callback when a message is received */
  onMessage?: (data: unknown) => void;
  /** Callback when connection is established */
  onConnect?: () => void;
  /** Callback when connection is closed */
  onDisconnect?: () => void;
  /** Callback when an error occurs */
  onError?: (error: Event) => void;
  /** Whether to auto-connect (default: true) */
  enabled?: boolean;
}

/**
 * Packet data structure from /api/v1/stream/packets
 */
export interface PacketData {
  type: "packet";
  timestamp: string;
  data: {
    sourceIP: string;
    destIP: string;
    protocol: string;
    size: number;
    payload?: string;
    [key: string]: unknown;
  };
}

/**
 * Hook for connecting to the packet stream
 */
export function usePacketStream(options: StreamHookOptions = {}): UseEventSourceResult {
  const { enabled = true, ...restOptions } = options;
  const url = enabled ? `${getStreamBaseUrl()}/packets` : "";

  return useEventSource({
    url,
    enabled,
    ...restOptions,
  });
}

/**
 * Log data structure from /api/v1/stream/logs
 */
export interface LogData {
  type: "log";
  timestamp: string;
  level: "debug" | "info" | "warn" | "error";
  message: string;
}

/**
 * Hook for connecting to the log stream
 */
export function useLogStream(options: StreamHookOptions = {}): UseEventSourceResult {
  const { enabled = true, ...restOptions } = options;
  const url = enabled ? `${getStreamBaseUrl()}/logs` : "";

  return useEventSource({
    url,
    enabled,
    ...restOptions,
  });
}

/**
 * Stats data structure from /api/v1/stream/stats
 */
export interface StatsData {
  type: "stats";
  timestamp: string;
  data: {
    packetsPerSecond: number;
    bytesPerSecond: number;
    activeConnections: number;
    cpuUsage?: number;
    memoryUsage?: number;
    [key: string]: unknown;
  };
}

/**
 * Hook for connecting to the stats stream
 */
export function useStatsStream(options: StreamHookOptions = {}): UseEventSourceResult {
  const { enabled = true, ...restOptions } = options;
  const url = enabled ? `${getStreamBaseUrl()}/stats` : "";

  return useEventSource({
    url,
    enabled,
    ...restOptions,
  });
}
