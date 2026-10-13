/**
 * Read hook for the SNMP credential vault (/api/v1/device-credentials).
 *
 * The list carries names, kinds and v3 metadata only: secrets are write-only
 * on the server and no response type can carry them. Fetched on mount, so a
 * form that opens after a credential was added sees it.
 */

import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type {
  Credentials,
  DeviceCredentialListResponse,
} from '../types/generated/device-credential-list-response';

const ENDPOINT = '/api/v1/device-credentials';

export interface UseDeviceCredentialsResult {
  credentials: Credentials[];
  loading: boolean;
  error: string | null;
}

export function useDeviceCredentials(): UseDeviceCredentialsResult {
  const [credentials, setCredentials] = useState<Credentials[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .get<DeviceCredentialListResponse>(ENDPOINT)
      .then((resp) => {
        if (!cancelled) {
          setCredentials(resp.credentials ?? []);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load credentials');
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return (): void => {
      cancelled = true;
    };
  }, []);

  return { credentials, loading, error };
}
