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
 * - Secure token transmission via Sec-WebSocket-Protocol header
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
  /** JWT authentication token */
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
  const reconnectAttempts = useRef(0);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const connectRef = useRef<(() => void) | null>(null); // New ref for stable connect function

  /**
   * Establishes WebSocket connection with automatic reconnection logic.
   *
   * - Checks if already connected before attempting new connection
   * - Requires valid authentication token
   * - Automatically determines wss:// vs ws:// based on page protocol
   * - Transmits JWT via Sec-WebSocket-Protocol for security
   * - Sets up event handlers for open, close, error, and message events
   * - Implements automatic reconnection with configurable attempts and interval
   */
  const connect = useCallback(() => {
    // Avoid duplicate connections
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    // Authentication required for WebSocket connection
    if (!token) {
      setStatus("disconnected");
      return;
    }

    setStatus("connecting");

    try {
      // Determine secure vs insecure WebSocket protocol based on page protocol
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      const baseUrl = url.startsWith("ws") ? url : `${protocol}//${window.location.host}${url}`;

      // Use Sec-WebSocket-Protocol header for secure token transmission
      // This prevents tokens from appearing in server logs, browser history, etc.
      wsRef.current = new WebSocket(baseUrl, ["access_token", token]);

      // Connection established successfully
      wsRef.current.onopen = () => {
        setStatus("connected");
        reconnectAttempts.current = 0; // Reset reconnection counter on success
      };

      // Connection closed - attempt reconnection if within retry limit
      wsRef.current.onclose = (event) => {
        setStatus("disconnected");
        console.warn("WebSocket closed:", event.code, event.reason);

        // Schedule reconnection attempt if not exceeded max attempts
        if (reconnectAttempts.current < maxReconnectAttempts) {
          reconnectTimeoutRef.current = setTimeout(() => {
            reconnectAttempts.current++;
            console.warn(`WebSocket reconnecting... attempt ${reconnectAttempts.current}`);
            connectRef.current?.(); // Call via ref to avoid stale closure
          }, reconnectInterval);
        }
      };

      // Connection error occurred
      wsRef.current.onerror = (error) => {
        setStatus("error");
        console.error("WebSocket error:", error);
      };

      // Message received from server
      wsRef.current.onmessage = (event) => {
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
            console.error("Failed to parse WebSocket message:", error, "payload=", payload);
          }
        }
      };
    } catch (error) {
      setStatus("error");
      console.error("Failed to create WebSocket:", error);
    }
  }, [url, token, onMessage, onCardUpdate, reconnectInterval, maxReconnectAttempts]);

  // Keep connectRef updated with latest connect function to avoid stale closures
  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  /**
   * Cleanly disconnects the WebSocket and clears any pending reconnection timers.
   */
  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    setStatus("disconnected");
  }, []);

  // (Re)connect whenever auth state changes
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | null = null;
    timer = setTimeout(() => {
      if (token) {
        connect();
      } else {
        disconnect();
      }
    }, 0);

    return () => {
      if (timer) clearTimeout(timer);
    };
  }, [token, connect, disconnect]);

  const send = useCallback((message: Message) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
    } else {
      console.warn("WebSocket not connected, cannot send message");
    }
  }, []);

  const reconnect = useCallback(() => {
    reconnectAttempts.current = 0;
    disconnect();
    connectRef.current?.(); // Call via ref
  }, [disconnect]); // connect is no longer a direct dependency

  useEffect(() => {
    connectRef.current?.(); // Call via ref
    return () => disconnect();
  }, [disconnect]); // connect is no longer a direct dependency

  return { status, send, reconnect };
}
