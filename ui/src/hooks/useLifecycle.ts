/**
 * Data hook for the /api/v1/lifecycle surface (P-D9, UI-SEED-47).
 *
 * One read carries every inventoried part and where its product stands.
 * Importing an EoX file replaces the server's table and re-reads, so the page
 * shows the statuses the server assessed rather than guessing them.
 */

import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { LifecycleDevicesResponse } from '../types/generated/lifecycle-devices-response';
import type { LifecycleEoXResponse } from '../types/generated/lifecycle-eox-response';

const DEVICES = '/api/v1/lifecycle/devices';
const EOX = '/api/v1/lifecycle/eox';

/** Thrown for a file that is not JSON, so the server is never asked. */
export class NotJsonError extends Error {
  constructor() {
    super('not JSON');
    this.name = 'NotJsonError';
  }
}

export interface UseLifecycleResult {
  data: LifecycleDevicesResponse | null;
  error: string | null;
  /** Imports a Cisco EoX API response file and resolves to its record count. */
  importEoX: (file: File) => Promise<number>;
}

export function useLifecycle(): UseLifecycleResult {
  const [data, setData] = useState<LifecycleDevicesResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refresh = async (): Promise<void> => {
    setError(null);
    await api
      .get<LifecycleDevicesResponse>(DEVICES)
      .then(setData)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)));
  };

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const importEoX = async (file: File): Promise<number> => {
    // The client sends JSON bodies, so the file is parsed here; the server
    // still decides whether it is an EoX response.
    let body: unknown;
    try {
      body = JSON.parse(await file.text());
    } catch {
      throw new NotJsonError();
    }
    const imported = await api.put<LifecycleEoXResponse>(EOX, body);
    await refresh();
    return imported.records.length;
  };

  return { data, error, importEoX };
}
