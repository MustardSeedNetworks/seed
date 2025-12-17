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

/** WebSocket close code categories for better error handling (fixes #676) */
enum CloseCodeCategory {
  Normal = "normal", // 1000-1001: Normal closure
  ProtocolError = "protocol_error", // 1002-1003: Protocol errors
  InvalidData = "invalid_data", // 1007-1008: Invalid data
  PolicyViolation = "policy_violation", // 1008-1009: Policy violations
  TooLarge = "too_large", // 1009: Message too large
  ClientError = "client_error", // 1011-1014: Client-side errors
  ServerError = "server_error", // 1011, 1015: Server-side errors
  Abnormal = "abnormal", // 1006: Abnormal closure (connection lost)
  Unknown = "unknown", // Any other code
}

/**
 * Categorize WebSocket close codes for better error handling and recovery strategies
 */
function categorizeCloseCode(code: number): CloseCodeCategory {
  if (code === 1000 || code === 1001) return CloseCodeCategory.Normal;
  if (code === 1002 || code === 1003) return CloseCodeCategory.ProtocolError;
  if (code === 1007 || code === 1008) return CloseCodeCategory.InvalidData;
  if (code === 1008 || code === 1009) return CloseCodeCategory.PolicyViolation;
  if (code === 1009) return CloseCodeCategory.TooLarge;
  if (code >= 1011 && code <= 1014) return CloseCodeCategory.ClientError;
  if (code === 1011 || code === 1015) return CloseCodeCategory.ServerError;
  if (code === 1006) return CloseCodeCategory.Abnormal;
  return CloseCodeCategory.Unknown;
}

/**
 * Determine if we should attempt reconnection based on close code category
 */
function shouldRetryForCategory(category: CloseCodeCategory): boolean {
  switch (category) {
    case CloseCodeCategory.Normal:
      return false; // Normal closure - no retry needed
    case CloseCodeCategory.Abnormal:
    case CloseCodeCategory.ServerError:
      return true; // Network issues or server problems - retry
    case CloseCodeCategory.ProtocolError:
    case CloseCodeCategory.InvalidData:
    case CloseCodeCategory.PolicyViolation:
    case CloseCodeCategory.ClientError:
      return false; // Client-side issues - retrying won't help
    case CloseCodeCategory.TooLarge:
      return false; // Message too large - client issue
    case CloseCodeCategory.Unknown:
      return true; // Unknown issue - try to reconnect
    default:
      return true; // Default to retry for safety
  }
}

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
  /** JWT access token for WebSocket authentication via protocol header */
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
  token,
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

      // Create WebSocket connection with token via protocol header
      // Backend expects: Sec-WebSocket-Protocol: access_token, <token>
      const protocols = token ? ["access_token", token] : undefined;
      const ws = new WebSocket(baseUrl, protocols);
      wsRef.current = ws;
      connectionIdRef.current += 1;
      const connectionId = connectionIdRef.current;

      // Connection established successfully
      ws.onopen = () => {
        if (connectionId !== connectionIdRef.current) return;
        setStatus("connected");
        reconnectAttempts.current = 0; // Reset reconnection counter on success
      };

      // Connection closed - attempt reconnection with error categorization (fixes #676)
      ws.onclose = (event) => {
        if (connectionId !== connectionIdRef.current) return;

        // Categorize the close code to determine recovery strategy
        const category = categorizeCloseCode(event.code);
        const shouldRetry = shouldRetryForCategory(category);

        setStatus("disconnected");
        logger.warn(LogComponents.WEBSOCKET, "WebSocket closed", {
          code: event.code,
          reason: event.reason,
          category,
          willRetry: shouldRetry && shouldReconnectRef.current,
        });

        // Don't retry if the close code indicates a client-side issue
        if (!shouldRetry) {
          logger.error(LogComponents.WEBSOCKET, "WebSocket closed with non-retryable error", {
            code: event.code,
            category,
          });
          setStatus("error");
          return;
        }

        if (!shouldReconnectRef.current) return;

        // Attempt reconnect with exponential backoff; maxReconnectAttempts gates backoff growth.
        reconnectAttempts.current++;
        const attempt = reconnectAttempts.current;
        const cappedAttempt = Math.max(1, Math.min(attempt, maxReconnectAttempts));

        // Exponential backoff with jitter, capped to keep retries happening after long outages.
        // For server errors, use more aggressive backoff
        const isServerError = category === CloseCodeCategory.ServerError;
        const maxDelayMs = isServerError ? 60000 : 30000; // 60s for server errors, 30s otherwise
        const baseDelayMs = reconnectInterval * 2 ** (cappedAttempt - 1);
        const delayMs = Math.min(baseDelayMs, maxDelayMs) + Math.floor(Math.random() * 500);

        logger.warn(LogComponents.WEBSOCKET, "WebSocket reconnect scheduled", {
          attempt,
          delayMs,
          category,
        });

        clearReconnectTimer();
        reconnectTimeoutRef.current = setTimeout(() => {
          if (!shouldReconnectRef.current) return;
          if (connectionId !== connectionIdRef.current) return;
          connectRef.current();
        }, delayMs);
      };

      // Connection error occurred - enhanced logging (fixes #676)
      ws.onerror = (error) => {
        if (connectionId !== connectionIdRef.current) return;
        setStatus("error");
        logger.error(LogComponents.WEBSOCKET, "WebSocket error", error, {
          url: baseUrl,
          readyState: ws.readyState,
          readyStateText:
            ws.readyState === WebSocket.CONNECTING
              ? "CONNECTING"
              : ws.readyState === WebSocket.OPEN
                ? "OPEN"
                : ws.readyState === WebSocket.CLOSING
                  ? "CLOSING"
                  : "CLOSED",
          reconnectAttempts: reconnectAttempts.current,
        });
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

            // fixes #679 - validate message structure before processing
            if (!message || typeof message !== "object") {
              logger.warn(LogComponents.WEBSOCKET, "Invalid message structure", { payload });
              continue;
            }

            if (typeof message.type !== "string") {
              logger.warn(LogComponents.WEBSOCKET, "Message missing or invalid type", { message });
              continue;
            }

            // Handle card update messages specially with validation
            if (message.type === "card_update") {
              // fixes #679 - validate card update payload structure
              const update = message.payload;
              if (!update || typeof update !== "object") {
                logger.warn(LogComponents.WEBSOCKET, "Invalid card_update payload", {
                  type: typeof update,
                });
                continue;
              }

              const cardUpdate = update as CardUpdate;
              if (!cardUpdate.cardId || typeof cardUpdate.cardId !== "string") {
                logger.warn(LogComponents.WEBSOCKET, "card_update missing valid cardId", {
                  cardId: cardUpdate.cardId,
                });
                continue;
              }

              if (onCardUpdate) {
                onCardUpdate(cardUpdate);
              }
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
      logger.error(LogComponents.WEBSOCKET, "Failed to create WebSocket", error, {
        url,
      });
    }
  }, [
    url,
    token,
    onMessage,
    onCardUpdate,
    reconnectInterval,
    maxReconnectAttempts,
    clearReconnectTimer,
  ]);

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
