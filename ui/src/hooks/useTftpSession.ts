/**
 * Data hook for the TFTP session (P-D4, UI-SEED-44). Every method on the
 * route is admin-only, so the hook reads nothing unless asked to.
 *
 * A running session can stop by itself (idle timeout, lifetime limit), and
 * its transfer count moves, so the state is re-read while one runs.
 */

import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { TftpSession, TftpStartRequest } from '../types/tftp';

const SESSION = '/api/v1/tftp/session';
const RUNNING_REFRESH_MS = 10_000;

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export interface UseTftpSessionResult {
  /** Null until the first read answers. */
  session: TftpSession | null;
  error: string | null;
  busy: boolean;
  start: (request: TftpStartRequest) => Promise<void>;
  stop: () => Promise<void>;
}

export function useTftpSession(enabled: boolean): UseTftpSessionResult {
  const [session, setSession] = useState<TftpSession | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const running = session?.running === true;

  useEffect(() => {
    if (!enabled) {
      return;
    }
    let cancelled = false;
    const read = (): void => {
      api
        .get<TftpSession>(SESSION)
        .then((next) => {
          if (!cancelled) {
            setSession(next);
          }
        })
        .catch((err: unknown) => {
          if (!cancelled) {
            setError(message(err));
          }
        });
    };
    if (!running) {
      read();
      return (): void => {
        cancelled = true;
      };
    }
    const timer = setInterval(read, RUNNING_REFRESH_MS);
    return (): void => {
      cancelled = true;
      clearInterval(timer);
    };
  }, [enabled, running]);

  const mutate = async (request: () => Promise<TftpSession>): Promise<void> => {
    setBusy(true);
    setError(null);
    try {
      setSession(await request());
    } catch (err: unknown) {
      setError(message(err));
    } finally {
      setBusy(false);
    }
  };

  const start = (request: TftpStartRequest): Promise<void> =>
    mutate(() => api.post<TftpSession>(SESSION, request));
  const stop = (): Promise<void> => mutate(() => api.delete<TftpSession>(SESSION));

  return { session, error, busy, start, stop };
}
