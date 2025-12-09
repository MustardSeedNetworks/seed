import { useCallback, useEffect, useRef, useState } from "react";

export type ConnectionStatus =
  | "connecting"
  | "connected"
  | "disconnected"
  | "error";

export interface Message {
  type: string;
  payload: unknown;
}

export interface CardUpdate {
  cardId: string;
  data: unknown;
}

interface UseWebSocketOptions {
  url: string;
  token?: string | null;
  onMessage?: (message: Message) => void;
  onCardUpdate?: (update: CardUpdate) => void;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
}

interface UseWebSocketReturn {
  status: ConnectionStatus;
  send: (message: Message) => void;
  reconnect: () => void;
}

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
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(
    null,
  );
  const connectRef = useRef<(() => void) | null>(null); // New ref for stable connect function

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    // Don't connect if no token (user not logged in)
    if (!token) {
      setStatus("disconnected");
      return;
    }

    setStatus("connecting");

    try {
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      const baseUrl = url.startsWith("ws")
        ? url
        : `${protocol}//${window.location.host}${url}`;
      // Add token as query parameter (browsers can't send custom headers with WebSocket)
      const wsUrl = `${baseUrl}?token=${encodeURIComponent(token)}`;

      wsRef.current = new WebSocket(wsUrl);

      wsRef.current.onopen = () => {
        setStatus("connected");
        reconnectAttempts.current = 0;
        // WebSocket connected successfully
      };

      wsRef.current.onclose = (event) => {
        setStatus("disconnected");
        console.warn("WebSocket closed:", event.code, event.reason);

        // Attempt to reconnect
        if (reconnectAttempts.current < maxReconnectAttempts) {
          reconnectTimeoutRef.current = setTimeout(() => {
            reconnectAttempts.current++;
            console.warn(
              `WebSocket reconnecting... attempt ${reconnectAttempts.current}`,
            );
            connectRef.current?.(); // Call via ref
          }, reconnectInterval);
        }
      };

      wsRef.current.onerror = (error) => {
        setStatus("error");
        console.error("WebSocket error:", error);
      };

      wsRef.current.onmessage = (event) => {
        // Server may coalesce multiple JSON messages in one frame separated by newlines.
        const payloads = String(event.data).split(/\n+/).filter(Boolean);

        for (const payload of payloads) {
          try {
            const message: Message = JSON.parse(payload);

            if (message.type === "card_update" && onCardUpdate) {
              onCardUpdate(message.payload as CardUpdate);
            }

            if (onMessage) {
              onMessage(message);
            }
          } catch (error) {
            console.error(
              "Failed to parse WebSocket message:",
              error,
              "payload=",
              payload,
            );
          }
        }
      };
    } catch (error) {
      setStatus("error");
      console.error("Failed to create WebSocket:", error);
    }
  }, [
    url,
    token,
    onMessage,
    onCardUpdate,
    reconnectInterval,
    maxReconnectAttempts,
  ]);

  // Use useEffect to assign the latest connect function to the ref
  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

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
