import { useEffect, useState } from 'react';
import { pingDaemon } from '../api/buildVersion';

export type ConnectionState = 'connected' | 'disconnected' | 'checking';

const CHECK_INTERVAL_MS = 15_000;
const TIMEOUT_MS = 5_000;

/**
 * Polls the unauthenticated /__version endpoint to tell whether the daemon
 * is reachable at all, independent of any session or the SSE stream.
 */
export function useConnectionStatus(): ConnectionState {
  const [status, setStatus] = useState<ConnectionState>('checking');
  useEffect(() => {
    let disposed = false;
    let controller = new AbortController();
    let timeout: ReturnType<typeof setTimeout> | undefined;
    const check = async (): Promise<void> => {
      controller = new AbortController();
      timeout = setTimeout(() => controller.abort(), TIMEOUT_MS);
      try {
        await pingDaemon(controller.signal);
        if (!disposed) setStatus('connected');
      } catch {
        if (!disposed) setStatus('disconnected');
      } finally {
        clearTimeout(timeout);
      }
    };
    void check();
    const interval = setInterval(() => void check(), CHECK_INTERVAL_MS);
    return () => {
      disposed = true;
      clearInterval(interval);
      clearTimeout(timeout);
      controller.abort();
    };
  }, []);
  return status;
}
