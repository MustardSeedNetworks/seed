/**
 * Server-Sent Events (SSE) Connection Hook
 *
 * Manages SSE connections to the The Seed backend for real-time updates.
 * SSE provides a simpler, more reliable alternative to WebSockets for
 * server-to-client streaming.
 *
 * Features:
 * - Automatic connection management with authentication (cookies)
 * - Built-in browser reconnection (via EventSource)
 * - Type-safe message handling
 * - Connection status tracking
 *
 * Advantages over WebSocket:
 * - Simpler protocol (standard HTTP)
 * - Automatic reconnection handled by browser
 * - Works through HTTP proxies and load balancers
 * - No special upgrade handshake required
 *
 * Usage:
 * ```typescript
 * const { status, reconnect } = useSse({
 *   url: '/api/events',
 *   onMessage: handleMessage,
 *   onCardUpdate: handleCardUpdate
 * });
 * ```
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import { LogComponents, logger } from '../lib/logger';
import {
  parseSseCardUpdate,
  parseSseMessage,
  type SseCardUpdate as SseCardUpdateSchemaType,
  type SseMessage as SseMessageSchemaType,
} from '../schemas/sse';

/** SSE connection status states */
export type SseConnectionStatus =
  | 'connecting' // Attempting to establish connection
  | 'connected' // Successfully connected
  | 'disconnected' // Not connected (intentional or after failure)
  | 'error'; // Connection error occurred

/**
 * Base message structure for SSE communication. Re-exports the type
 * derived from the valibot schema in `@/schemas/sse` — adding fields
 * means updating the schema and the type falls out automatically.
 */
export type SseMessage = SseMessageSchemaType;

/**
 * Card update message for real-time UI updates. Re-exports the
 * schema-derived type from `@/schemas/sse`.
 */
export type SseCardUpdate = SseCardUpdateSchemaType;

/** Configuration options for useSse hook */
interface UseSseOptions {
  /** SSE endpoint URL */
  url: string;
  /** Whether the user is authenticated (controls connection behavior) */
  isAuthenticated?: boolean;
  /** Callback invoked for general messages */
  onMessage?: (message: SseMessage) => void;
  /** Callback invoked specifically for card update messages */
  onCardUpdate?: (update: SseCardUpdate) => void;
}

/** Return value from useSse hook */
interface UseSseReturn {
  /** Current connection status */
  status: SseConnectionStatus;
  /** Manually trigger reconnection */
  reconnect: () => void;
}

// Frame-level validation lives in @/schemas/sse via valibot safeParse.
// The hand-rolled isValidMessage / isValidCardUpdate guards that used
// to live here drifted independently from the Go side; the schemas now
// keep the runtime check and the static type in one place.

/**
 * Custom hook for managing SSE connections with automatic reconnection.
 *
 * @param options - SSE configuration options
 * @returns Object containing connection status and reconnect function
 */
export function useSse({
  url,
  isAuthenticated = true,
  onMessage,
  onCardUpdate,
}: UseSseOptions): UseSseReturn {
  const [status, setStatus] = useState<SseConnectionStatus>('disconnected');
  const eventSourceRef = useRef<EventSource | null>(null);
  const connectionIdRef = useRef(0);

  // Store callbacks in refs to avoid recreating connect() when callbacks change
  const onMessageRef = useRef(onMessage);
  const onCardUpdateRef = useRef(onCardUpdate);

  // Keep refs up to date with latest callbacks
  useEffect(() => {
    onMessageRef.current = onMessage;
  }, [onMessage]);

  useEffect(() => {
    onCardUpdateRef.current = onCardUpdate;
  }, [onCardUpdate]);

  /**
   * Processes an SSE message and routes it to appropriate handlers.
   */
  const handleSseMessage = useCallback(
    (data: string, connectionId: number) => {
      // Ignore messages from stale connections
      if (connectionId !== connectionIdRef.current) {
        return;
      }

      let raw: unknown;
      try {
        raw = JSON.parse(data);
      } catch (error) {
        logger.error(LogComponents.SSE, 'Failed to parse SSE message', error, { data });
        return;
      }

      const message = parseSseMessage(raw);
      if (!message) {
        logger.warn(LogComponents.SSE, 'Invalid SSE envelope', { data });
        return;
      }

      // Handle card update messages specially with per-type validation.
      // The envelope's payload is `unknown` until we narrow it via the
      // card-update schema; dropping invalid card_update frames is
      // safer than dispatching them and crashing the subscriber.
      if (message.type === 'card_update') {
        const update = parseSseCardUpdate(message.payload);
        if (!update) {
          logger.warn(LogComponents.SSE, 'Invalid card_update payload', {
            payloadType: typeof message.payload,
          });
          return;
        }
        if (onCardUpdateRef.current) {
          onCardUpdateRef.current(update);
        }
      }

      // Always invoke general message handler with the validated envelope.
      if (onMessageRef.current) {
        onMessageRef.current(message);
      }
    },
    // Validation functions are pure and stable - no dependencies needed
    [],
  );

  /**
   * Establishes SSE connection with automatic browser-managed reconnection.
   *
   * EventSource provides built-in reconnection with exponential backoff.
   * Authentication is handled via httpOnly cookies (sent automatically).
   */
  const connect = useCallback(() => {
    // Don't connect if not authenticated
    if (!isAuthenticated) {
      logger.info(LogComponents.SSE, 'Skipping SSE connection - not authenticated');
      setStatus('disconnected');
      return;
    }

    // Avoid duplicate connections
    if (eventSourceRef.current?.readyState === EventSource.OPEN) {
      return;
    }

    // Close any existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    setStatus('connecting');
    connectionIdRef.current += 1;
    const connectionId = connectionIdRef.current;

    try {
      // Determine the full URL (EventSource doesn't support relative URLs in all browsers)
      const fullUrl = url.startsWith('http')
        ? url
        : `${window.location.protocol}//${window.location.host}${url}`;

      // Create EventSource with credentials to send cookies
      const eventSource = new EventSource(fullUrl, { withCredentials: true });
      eventSourceRef.current = eventSource;

      // Connection opened successfully
      eventSource.onopen = (): void => {
        if (connectionId !== connectionIdRef.current) {
          return;
        }
        setStatus('connected');
        logger.info(LogComponents.SSE, 'SSE connected', { url: fullUrl });
      };

      // Handle incoming messages
      eventSource.onmessage = (event: MessageEvent): void => {
        if (connectionId !== connectionIdRef.current) {
          return;
        }
        handleSseMessage(event.data, connectionId);
      };

      // Handle connection errors
      eventSource.onerror = (event: Event): void => {
        if (connectionId !== connectionIdRef.current) {
          return;
        }

        // EventSource reconnects automatically, but we track status
        if (eventSource.readyState === EventSource.CLOSED) {
          setStatus('disconnected');
          logger.warn(LogComponents.SSE, 'SSE connection closed');
        } else if (eventSource.readyState === EventSource.CONNECTING) {
          setStatus('connecting');
          logger.info(LogComponents.SSE, 'SSE reconnecting...');
        } else {
          setStatus('error');
          logger.error(LogComponents.SSE, 'SSE error', event);
        }
      };
    } catch (error) {
      setStatus('error');
      logger.error(LogComponents.SSE, 'Failed to create EventSource', error, { url });
    }
  }, [url, isAuthenticated, handleSseMessage]);

  /**
   * Cleanly disconnects the SSE connection.
   */
  const disconnect = useCallback(() => {
    connectionIdRef.current += 1; // Invalidate handlers
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    setStatus('disconnected');
  }, []);

  /**
   * Manually trigger reconnection.
   */
  const reconnect = useCallback(() => {
    disconnect();
    // Small delay to ensure clean disconnect
    setTimeout(connect, 100);
  }, [connect, disconnect]);

  // Connect on mount, disconnect on unmount
  useEffect(() => {
    connect();
    return () => disconnect();
  }, [connect, disconnect]);

  return { status, reconnect };
}

export default useSse;
