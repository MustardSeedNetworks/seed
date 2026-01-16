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

import { useCallback, useEffect, useRef, useState } from "react";
import { LogComponents, logger } from "../lib/logger";

/** SSE connection status states */
export type SseConnectionStatus =
  | "connecting" // Attempting to establish connection
  | "connected" // Successfully connected
  | "disconnected" // Not connected (intentional or after failure)
  | "error"; // Connection error occurred

/** Base message structure for SSE communication */
export interface SseMessage {
  type: string; // Message type identifier
  payload: unknown; // Message data (type varies by message type)
}

/** Card update message for real-time UI updates */
export interface SseCardUpdate {
  cardId: string; // ID of the card to update
  data: unknown; // Updated card data
  interface?: string; // Network interface name (e.g., "eth0", "wlan0")
}

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

/**
 * Validates message structure before processing.
 * Defined outside component to be stable across renders.
 */
function isValidMessage(message: unknown): message is SseMessage {
  if (!message || typeof message !== "object") {
    return false;
  }
  const msg = message as Record<string, unknown>;
  return typeof msg.type === "string";
}

/**
 * Validates card update payload structure.
 * Defined outside component to be stable across renders.
 */
function isValidCardUpdate(payload: unknown): payload is SseCardUpdate {
  if (!payload || typeof payload !== "object") {
    return false;
  }
  const update = payload as Record<string, unknown>;
  return typeof update.cardId === "string";
}

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
  const [status, setStatus] = useState<SseConnectionStatus>("disconnected");
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
    // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Complex message routing logic
    (data: string, connectionId: number) => {
      // Ignore messages from stale connections
      if (connectionId !== connectionIdRef.current) {
        return;
      }

      try {
        const message: unknown = JSON.parse(data);

        if (!isValidMessage(message)) {
          logger.warn(LogComponents.SSE, "Invalid message structure", { data });
          return;
        }

        // Handle card update messages specially with validation
        if (message.type === "card_update") {
          if (!isValidCardUpdate(message.payload)) {
            logger.warn(LogComponents.SSE, "Invalid card_update payload", {
              type: typeof message.payload,
            });
            return;
          }

          if (onCardUpdateRef.current) {
            onCardUpdateRef.current(message.payload);
          }
        }

        // Always invoke general message handler
        if (onMessageRef.current) {
          onMessageRef.current(message);
        }
      } catch (error) {
        logger.error(LogComponents.SSE, "Failed to parse SSE message", error, { data });
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
      logger.info(LogComponents.SSE, "Skipping SSE connection - not authenticated");
      setStatus("disconnected");
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

    setStatus("connecting");
    connectionIdRef.current += 1;
    const connectionId = connectionIdRef.current;

    try {
      // Determine the full URL (EventSource doesn't support relative URLs in all browsers)
      const fullUrl = url.startsWith("http")
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
        setStatus("connected");
        logger.info(LogComponents.SSE, "SSE connected", { url: fullUrl });
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
          setStatus("disconnected");
          logger.warn(LogComponents.SSE, "SSE connection closed");
        } else if (eventSource.readyState === EventSource.CONNECTING) {
          setStatus("connecting");
          logger.info(LogComponents.SSE, "SSE reconnecting...");
        } else {
          setStatus("error");
          logger.error(LogComponents.SSE, "SSE error", event);
        }
      };
    } catch (error) {
      setStatus("error");
      logger.error(LogComponents.SSE, "Failed to create EventSource", error, { url });
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
    setStatus("disconnected");
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
