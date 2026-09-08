/**
 * The wire shape of a client log batch is owned by the API, not by this
 * module: `internal/api.ClientLogEntry` decodes with DisallowUnknownFields,
 * so one extra key rejects the whole batch with 400 and the entries go back
 * into the buffer to be retried forever. The generated contract type is the
 * arbiter — these tests compare the body the logger actually sends against
 * its keys rather than against a second hand-written copy of them.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { ClientLogEntry } from '../types/generated/client-log-request';
import { Logger } from './logger';

/**
 * The contract's key set, derived from the generated type rather than
 * retyped: `Required<ClientLogEntry>` makes tsc reject this literal the
 * moment the schema gains or loses a field, so the list cannot drift.
 */
const CONTRACT_SAMPLE: Required<ClientLogEntry> = {
  timestamp: '',
  level: '',
  component: '',
  message: '',
  requestId: '',
  sessionId: '',
  metadata: {},
  stack: '',
};
const CONTRACT_KEYS: string[] = Object.keys(CONTRACT_SAMPLE);

interface SentBatch {
  entries: Record<string, unknown>[];
}

function fetchMockAccepting(): ReturnType<typeof vi.fn> {
  return vi.fn().mockResolvedValue({ ok: true, status: 200 } as Response);
}

function fetchMockReturning(status: number): ReturnType<typeof vi.fn> {
  return vi.fn().mockResolvedValue({ ok: false, status } as Response);
}

function bufferedLogger(): Logger {
  const logger = new Logger({ consoleOutput: false, flushInterval: 1_000_000 });
  logger.setAuthenticated(true);
  logger.info('auth', 'User logged in successfully', { username: 'admin' });
  return logger;
}

describe('Logger flush payload', () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchMock = fetchMockAccepting();
    vi.stubGlobal('fetch', fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  async function flushOneBatch(): Promise<SentBatch> {
    const logger = bufferedLogger();
    await logger.flush();

    expect(fetchMock).toHaveBeenCalledOnce();
    const [call] = fetchMock.mock.calls as [[string, RequestInit]];
    const [url, init] = call;
    expect(url).toBe('/api/v1/reporting/logs/client');
    return JSON.parse(String(init.body)) as SentBatch;
  }

  function onlyEntry(body: SentBatch): Record<string, unknown> {
    const [entry] = body.entries;
    if (entry === undefined) {
      throw new Error('flush sent an empty batch');
    }
    return entry;
  }

  it('sends only keys the API contract declares', async () => {
    const body = await flushOneBatch();

    expect(Object.keys(body)).toEqual(['entries']);
    expect(body.entries).toHaveLength(1);
    const unknown = Object.keys(onlyEntry(body)).filter((key) => !CONTRACT_KEYS.includes(key));
    expect(unknown).toEqual([]);
  });

  it('sends the fields the API needs to store the entry', async () => {
    const entry = onlyEntry(await flushOneBatch());

    expect(entry.level).toBe('INFO');
    expect(entry.component).toBe('auth');
    expect(entry.message).toBe('User logged in successfully');
    expect(entry.sessionId).toEqual(expect.any(String));
    expect(entry.timestamp).toEqual(expect.any(String));
    expect(entry.metadata).toEqual({ username: 'admin' });
  });

  it('drops a batch the server rejected as malformed instead of retrying it', async () => {
    const rejecting = fetchMockReturning(400);
    vi.stubGlobal('fetch', rejecting);
    const logger = bufferedLogger();

    await logger.flush();
    await logger.flush();

    // A 400 is a contract error: the same body will be rejected forever, so
    // re-queuing it turns every later flush into another 400 (observed as a
    // burst of rejected POSTs per page load in the E2E server logs, #2343).
    expect(rejecting).toHaveBeenCalledOnce();
  });

  it.each([503, 429])('keeps a batch the server could not accept yet (%i)', async (status) => {
    const unavailable = fetchMockReturning(status);
    vi.stubGlobal('fetch', unavailable);
    const logger = bufferedLogger();

    await logger.flush();
    await logger.flush();

    expect(unavailable).toHaveBeenCalledTimes(2);
  });
});
