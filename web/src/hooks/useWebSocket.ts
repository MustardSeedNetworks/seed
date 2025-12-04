import { useCallback, useEffect, useRef, useState } from 'react';

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

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
  onMessage,
  onCardUpdate,
  reconnectInterval = 3000,
  maxReconnectAttempts = 10,
}: UseWebSocketOptions): UseWebSocketReturn {
  const [status, setStatus] = useState<ConnectionStatus>('disconnected');
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttempts = useRef(0);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    setStatus('connecting');

    try {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = url.startsWith('ws') ? url : `${protocol}//${window.location.host}${url}`;

      wsRef.current = new WebSocket(wsUrl);

      wsRef.current.onopen = () => {
        setStatus('connected');
        reconnectAttempts.current = 0;
        // WebSocket connected successfully
      };

      wsRef.current.onclose = (event) => {
        setStatus('disconnected');
        console.warn('WebSocket closed:', event.code, event.reason);

        // Attempt to reconnect
        if (reconnectAttempts.current < maxReconnectAttempts) {
          reconnectTimeoutRef.current = setTimeout(() => {
            reconnectAttempts.current++;
            console.warn(`WebSocket reconnecting... attempt ${reconnectAttempts.current}`);
            connect();
          }, reconnectInterval);
        }
      };

      wsRef.current.onerror = (error) => {
        setStatus('error');
        console.error('WebSocket error:', error);
      };

      wsRef.current.onmessage = (event) => {
        try {
          const message: Message = JSON.parse(event.data);

          // Handle card updates specifically
          if (message.type === 'card_update' && onCardUpdate) {
            onCardUpdate(message.payload as CardUpdate);
          }

          // Call general message handler
          if (onMessage) {
            onMessage(message);
          }
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };
    } catch (error) {
      setStatus('error');
      console.error('Failed to create WebSocket:', error);
    }
  }, [url, onMessage, onCardUpdate, reconnectInterval, maxReconnectAttempts]);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    setStatus('disconnected');
  }, []);

  const send = useCallback((message: Message) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
    } else {
      console.warn('WebSocket not connected, cannot send message');
    }
  }, []);

  const reconnect = useCallback(() => {
    reconnectAttempts.current = 0;
    disconnect();
    connect();
  }, [connect, disconnect]);

  useEffect(() => {
    connect();
    return () => disconnect();
  }, [connect, disconnect]);

  return { status, send, reconnect };
}
