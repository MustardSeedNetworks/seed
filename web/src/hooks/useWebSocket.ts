/**
 * WebSocket Connection Hook
 *
 * Manages WebSocket connections to the The Seed backend for real-time updates.
 *
 * Features:
 * - Automatic connection management with authentication
 * - Automatic reconnection with exponential backoff
 * - Type-safe message handling
 * - Connection status tracking
 * - Cookie-based authentication (httpOnly cookies)
 *
 * The hook automatically handles:
 * - Initial connection establishment
 * - Authentication via JWT token
 * - Connection loss recovery
 * - Message parsing and routing
 *
 * Usage:
 * ```typescript
 * const { status, send, reconnect } = useWebSocket({
 *   url: '/api/ws',
 *   token: authToken,
 *   onMessage: handleMessage,
 *   onCardUpdate: handleCardUpdate
 * });
 * ```
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { logger, LogComponents } from "../lib/logger";

/** WebSocket connection status states */
export type ConnectionStatus =
  | "connecting" // Attempting to establish connection
  | "connected" // Successfully connected
  | "disconnected" // Not connected (intentional or after failure)
  | "error"; // Connection error occurred

/** Base message structure for WebSocket communication */
export interface Message {
  type: string; // Message type identifier
  payload: unknown; // Message data (type varies by message type)
}

/** Card update message for real-time UI updates */
export interface CardUpdate {
  cardId: string; // ID of the card to update
  data: unknown; // Updated card data
}

/** Configuration options for useWebSocket hook */
interface UseWebSocketOptions {
  /** WebSocket endpoint URL */
  url: string;
  /** JWT authentication token (deprecated - using cookie auth instead) */
  token?: string | null;
  /** Callback invoked for general messages */
  onMessage?: (message: Message) => void;
  /** Callback invoked specifically for card update messages */
  onCardUpdate?: (update: CardUpdate) => void;
  /** Milliseconds to wait between reconnection attempts (default: 3000) */
  reconnectInterval?: number;
  /** Maximum number of reconnection attempts (default: 10) */
  maxReconnectAttempts?: number;
}

/** Return value from useWebSocket hook */
interface UseWebSocketReturn {
  /** Current connection status */
  status: ConnectionStatus;
  /** Send a message to the server */
  send: (message: Message) => void;
  /** Manually trigger reconnection */
  reconnect: () => void;
}

/**
 * Custom hook for managing WebSocket connections with automatic reconnection.
 *
 * @param options - WebSocket configuration options
 * @returns Object containing connection status, send function, and reconnect function
 */
export function useWebSocket({
  url,
  onMessage,
  onCardUpdate,
  reconnectInterval = 3000,
  maxReconnectAttempts = 10,
}: UseWebSocketOptions): UseWebSocketReturn {
  const [status, setStatus] = useState<ConnectionStatus>("disconnected");
  const wsRef = useRef<WebSocket | null>(null);
  const connectRef = useRef<() => void>(() => {});
  const reconnectAttempts = useRef(0);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const shouldReconnectRef = useRef(true);
  const connectionIdRef = useRef(0);

  const clearReconnectTimer = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
  }, []);

  /**
   * Establishes WebSocket connection with automatic reconnection logic.
   *
   * - Checks if already connected before attempting new connection
   * - Automatically determines wss:// vs ws:// based on page protocol
   * - Uses cookie-based authentication (httpOnly cookies sent automatically)
   * - Sets up event handlers for open, close, error, and message events
   * - Implements automatic reconnection with configurable attempts and interval
   */
  const connect = useCallback(() => {
    // Avoid duplicate connections
    if (
      wsRef.current?.readyState === WebSocket.OPEN ||
      wsRef.current?.readyState === WebSocket.CONNECTING
    ) {
      return;
    }

    clearReconnectTimer();
    shouldReconnectRef.current = true;

    setStatus("connecting");

    try {
      // Determine secure vs insecure WebSocket protocol based on page protocol
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      const baseUrl = url.startsWith("ws") ? url : `${protocol}//${window.location.host}${url}`;

      // Create WebSocket connection without token in protocol header
      // Authentication is handled via httpOnly cookies sent automatically by the browser
      const ws = new WebSocket(baseUrl);
      wsRef.current = ws;
      connectionIdRef.current += 1;
      const connectionId = connectionIdRef.current;

      // Connection established successfully
      ws.onopen = () => {
        if (connectionId !== connectionIdRef.current) return;
        setStatus("connected");
        reconnectAttempts.current = 0; // Reset reconnection counter on success
      };

      // Connection closed - attempt reconnection if within retry limit
      ws.onclose = (event) => {
        if (connectionId !== connectionIdRef.current) return;
        setStatus("disconnected");
        logger.warn(LogComponents.WEBSOCKET, "WebSocket closed", {
          code: event.code,
          reason: event.reason,
        });

        if (!shouldReconnectRef.current) return;

        // Attempt reconnect forever; maxReconnectAttempts gates exponential backoff growth.
        reconnectAttempts.current++;
        const attempt = reconnectAttempts.current;
        const cappedAttempt = Math.max(1, Math.min(attempt, maxReconnectAttempts));

        // Exponential backoff with jitter, capped to keep retries happening after long outages.
        const maxDelayMs = Math.max(reconnectInterval, 30000);
        const baseDelayMs = reconnectInterval * 2 ** (cappedAttempt - 1);
        const delayMs = Math.min(baseDelayMs, maxDelayMs) + Math.floor(Math.random() * 250);

        logger.warn(LogComponents.WEBSOCKET, "WebSocket reconnect scheduled", {
          attempt,
          delayMs,
        });

        clearReconnectTimer();
        reconnectTimeoutRef.current = setTimeout(() => {
          if (!shouldReconnectRef.current) return;
          if (connectionId !== connectionIdRef.current) return;
          connectRef.current();
        }, delayMs);
      };

      // Connection error occurred
      ws.onerror = (error) => {
        if (connectionId !== connectionIdRef.current) return;
        setStatus("error");
        logger.error(LogComponents.WEBSOCKET, "WebSocket error", error);
      };

      // Message received from server
      ws.onmessage = (event) => {
        if (connectionId !== connectionIdRef.current) return;
        // Server may coalesce multiple JSON messages in one frame separated by newlines
        const payloads = String(event.data).split(/\n+/).filter(Boolean);

        // Process each message payload
        for (const payload of payloads) {
          try {
            const message: Message = JSON.parse(payload);

            // Handle card update messages specially
            if (message.type === "card_update" && onCardUpdate) {
              onCardUpdate(message.payload as CardUpdate);
            }

            // Always invoke general message handler
            if (onMessage) {
              onMessage(message);
            }
          } catch (error) {
            logger.error(LogComponents.WEBSOCKET, "Failed to parse WebSocket message", error, {
              payload,
            });
          }
        }
      };
    } catch (error) {
      setStatus("error");
      logger.error(LogComponents.WEBSOCKET, "Failed to create WebSocket", error);
    }
  }, [url, onMessage, onCardUpdate, reconnectInterval, maxReconnectAttempts, clearReconnectTimer]);

  /**
   * Cleanly disconnects the WebSocket and clears any pending reconnection timers.
   */
  const disconnect = useCallback(() => {
    shouldReconnectRef.current = false;
    connectionIdRef.current += 1; // Invalidate handlers/timers from any in-flight socket.
    clearReconnectTimer();

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    setStatus("disconnected");
  }, [clearReconnectTimer]);

  const send = useCallback((message: Message) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
    } else {
      logger.warn(LogComponents.WEBSOCKET, "WebSocket not connected, cannot send message");
    }
  }, []);

  const reconnect = useCallback(() => {
    reconnectAttempts.current = 0;
    disconnect();
    shouldReconnectRef.current = true;
    connect();
  }, [connect, disconnect]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- Establish WebSocket connection on mount/reconnect
    connect();
    return () => disconnect();
  }, [connect, disconnect]);

  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  return { status, send, reconnect };
}
