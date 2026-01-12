/**
 * @deprecated This module is deprecated. Use useEventSource.ts instead.
 *
 * SSE (Server-Sent Events) is preferred over WebSocket for NIAC because:
 * - Automatic reconnection built into browser EventSource API
 * - Simpler API (server → client only, which is all we need)
 * - Better proxy/CDN compatibility
 * - Works well with HTTP/2 multiplexing
 *
 * WebSocket endpoints (/ws/*) are still available for backwards compatibility,
 * but new code should use SSE endpoints (/api/v1/stream/*).
 */

import { useCallback, useEffect, useRef, useState } from 'react';

/**
 * @deprecated Use UseEventSourceOptions from useEventSource.ts instead.
 * Options for configuring the WebSocket connection
 */
export interface UseWebSocketOptions {
  /** WebSocket URL to connect to */
  url: string;
  /** Callback when a message is received (after JSON parsing) */
  onMessage?: (data: unknown) => void;
  /** Callback when connection is established */
  onConnect?: () => void;
  /** Callback when connection is closed */
  onDisconnect?: () => void;
  /** Callback when an error occurs */
  onError?: (error: Event) => void;
  /** Reconnection interval in milliseconds (default: 2000) */
  reconnectInterval?: number;
  /** Maximum number of reconnection attempts (default: 10) */
  maxReconnectAttempts?: number;
}

/**
 * Result returned by the useWebSocket hook
 */
export interface UseWebSocketResult {
  /** Last received data (JSON parsed) */
  data: unknown;
  /** Whether the WebSocket is currently connected */
  connected: boolean;
  /** Last error event, if any */
  error: Event | null;
  /** Whether the hook is currently attempting to reconnect */
  reconnecting: boolean;
  /** Current reconnection attempt number (0 if not reconnecting) */
  reconnectAttempt: number;
  /** Send data through the WebSocket (will be JSON stringified) */
  send: (data: unknown) => void;
  /** Manually disconnect the WebSocket */
  disconnect: () => void;
  /** Manually reconnect the WebSocket */
  reconnect: () => void;
}

/**
 * WebSocket hook with auto-connect, auto-reconnect, and JSON handling
 *
 * Features:
 * - Auto-connect on mount
 * - Auto-reconnect on disconnect with exponential backoff
 * - JSON parse incoming messages
 * - Connection state tracking
 * - Error handling
 * - Send method for bidirectional communication
 * - Cleanup on unmount
 *
 * @param options - WebSocket configuration options
 * @returns WebSocket state and control methods
 *
 * @example
 * ```typescript
 * const { data, connected, send } = useWebSocket({
 *   url: 'ws://localhost:8080/ws/packets',
 *   onMessage: (data) => console.log('Received:', data),
 *   onConnect: () => console.log('Connected!'),
 * });
 * ```
 */
export function useWebSocket(options: UseWebSocketOptions): UseWebSocketResult {
  const {
    url,
    onMessage,
    onConnect,
    onDisconnect,
    onError,
    reconnectInterval = 2000,
    maxReconnectAttempts = 10,
  } = options;

  const [data, setData] = useState<unknown>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<Event | null>(null);
  const [reconnecting, setReconnecting] = useState(false);
  const [reconnectAttempt, setReconnectAttempt] = useState(0);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const shouldReconnectRef = useRef(true);
  const mountedRef = useRef(true);

  // Store callbacks in refs to avoid re-creating effects
  const onMessageRef = useRef(onMessage);
  const onConnectRef = useRef(onConnect);
  const onDisconnectRef = useRef(onDisconnect);
  const onErrorRef = useRef(onError);

  // Update callback refs when they change
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
   * Clear any pending reconnect timeout
   */
  const clearReconnectTimeout = useCallback(() => {
    if (reconnectTimeoutRef.current !== null) {
      window.clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
  }, []);

  /**
   * Calculate reconnect delay with exponential backoff
   * Caps at 30 seconds maximum
   */
  const getReconnectDelay = useCallback(
    (attempt: number): number => {
      const exponentialDelay = reconnectInterval * Math.pow(2, attempt);
      return Math.min(exponentialDelay, 30000); // Cap at 30 seconds
    },
    [reconnectInterval]
  );

  /**
   * Connect to the WebSocket server
   */
  const connect = useCallback(() => {
    // Don't connect if already connected or connecting
    if (wsRef.current?.readyState === WebSocket.OPEN || wsRef.current?.readyState === WebSocket.CONNECTING) {
      return;
    }

    // Clear any pending reconnect
    clearReconnectTimeout();

    try {
      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        if (!mountedRef.current) return;

        setConnected(true);
        setError(null);
        setReconnecting(false);
        setReconnectAttempt(0);
        onConnectRef.current?.();
      };

      ws.onmessage = (event: MessageEvent) => {
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

      ws.onerror = (event: Event) => {
        if (!mountedRef.current) return;

        setError(event);
        onErrorRef.current?.(event);
      };

      ws.onclose = () => {
        if (!mountedRef.current) return;

        const wasConnected = connected;
        setConnected(false);

        if (wasConnected) {
          onDisconnectRef.current?.();
        }

        // Attempt reconnection if we should
        if (shouldReconnectRef.current && reconnectAttempt < maxReconnectAttempts) {
          setReconnecting(true);
          const nextAttempt = reconnectAttempt + 1;
          setReconnectAttempt(nextAttempt);

          const delay = getReconnectDelay(reconnectAttempt);
          reconnectTimeoutRef.current = window.setTimeout(() => {
            if (mountedRef.current && shouldReconnectRef.current) {
              connect();
            }
          }, delay);
        } else if (reconnectAttempt >= maxReconnectAttempts) {
          setReconnecting(false);
        }
      };
    } catch (err) {
      if (!mountedRef.current) return;

      // Handle connection errors (e.g., invalid URL)
      const errorEvent = new Event('error');
      setError(errorEvent);
      onErrorRef.current?.(errorEvent);
    }
  }, [url, connected, reconnectAttempt, maxReconnectAttempts, getReconnectDelay, clearReconnectTimeout]);

  /**
   * Send data through the WebSocket
   */
  const send = useCallback((sendData: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      const message = typeof sendData === 'string' ? sendData : JSON.stringify(sendData);
      wsRef.current.send(message);
    }
    // Silently ignore sends when not connected
  }, []);

  /**
   * Disconnect from the WebSocket server
   */
  const disconnect = useCallback(() => {
    shouldReconnectRef.current = false;
    clearReconnectTimeout();
    setReconnecting(false);
    setReconnectAttempt(0);

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, [clearReconnectTimeout]);

  /**
   * Manually trigger a reconnection
   */
  const reconnect = useCallback(() => {
    shouldReconnectRef.current = true;
    setReconnectAttempt(0);
    disconnect();
    // Use setTimeout to ensure disconnect completes
    window.setTimeout(() => {
      shouldReconnectRef.current = true;
      connect();
    }, 0);
  }, [connect, disconnect]);

  // Connect on mount, cleanup on unmount
  useEffect(() => {
    mountedRef.current = true;
    shouldReconnectRef.current = true;
    connect();

    return () => {
      mountedRef.current = false;
      shouldReconnectRef.current = false;
      clearReconnectTimeout();

      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [connect, clearReconnectTimeout]);

  return {
    data,
    connected,
    error,
    reconnecting,
    reconnectAttempt,
    send,
    disconnect,
    reconnect,
  } as const;
}

// ============================================================================
// Specialized Stream Hooks
// ============================================================================

/**
 * Get the WebSocket base URL from the current location
 */
function getWebSocketBaseUrl(): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}`;
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
 * Packet data structure from /ws/packets
 */
export interface PacketData {
  timestamp: string;
  sourceIP: string;
  destIP: string;
  protocol: string;
  size: number;
  payload?: string;
  [key: string]: unknown;
}

/**
 * Hook for connecting to the packet stream WebSocket
 *
 * @param options - Stream configuration options
 * @returns WebSocket state and control methods
 *
 * @example
 * ```typescript
 * const { data, connected } = usePacketStream();
 *
 * useEffect(() => {
 *   if (data) {
 *     setPackets(prev => [...prev, data].slice(-100));
 *   }
 * }, [data]);
 * ```
 */
export function usePacketStream(options: StreamHookOptions = {}): UseWebSocketResult {
  const { enabled = true, ...wsOptions } = options;
  const url = enabled ? `${getWebSocketBaseUrl()}/ws/packets` : '';

  return useWebSocket({
    url,
    ...wsOptions,
  });
}

/**
 * Log data structure from /ws/logs
 */
export interface LogData {
  timestamp: string;
  level: 'debug' | 'info' | 'warn' | 'error';
  message: string;
  source?: string;
  [key: string]: unknown;
}

/**
 * Hook for connecting to the log stream WebSocket
 *
 * @param options - Stream configuration options
 * @returns WebSocket state and control methods
 *
 * @example
 * ```typescript
 * const { data, connected } = useLogStream({
 *   onMessage: (log) => console.log(`[${log.level}] ${log.message}`),
 * });
 * ```
 */
export function useLogStream(options: StreamHookOptions = {}): UseWebSocketResult {
  const { enabled = true, ...wsOptions } = options;
  const url = enabled ? `${getWebSocketBaseUrl()}/ws/logs` : '';

  return useWebSocket({
    url,
    ...wsOptions,
  });
}

/**
 * Stats data structure from /ws/stats
 */
export interface StatsData {
  timestamp: string;
  packetsPerSecond: number;
  bytesPerSecond: number;
  activeConnections: number;
  cpuUsage?: number;
  memoryUsage?: number;
  [key: string]: unknown;
}

/**
 * Hook for connecting to the stats stream WebSocket
 *
 * @param options - Stream configuration options
 * @returns WebSocket state and control methods
 *
 * @example
 * ```typescript
 * const { data: stats, connected } = useStatsStream();
 *
 * return (
 *   <div>
 *     {connected && stats && (
 *       <span>Packets/sec: {stats.packetsPerSecond}</span>
 *     )}
 *   </div>
 * );
 * ```
 */
export function useStatsStream(options: StreamHookOptions = {}): UseWebSocketResult {
  const { enabled = true, ...wsOptions } = options;
  const url = enabled ? `${getWebSocketBaseUrl()}/ws/stats` : '';

  return useWebSocket({
    url,
    ...wsOptions,
  });
}
