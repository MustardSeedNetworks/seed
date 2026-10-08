/**
 * useDashboardLayout reads and saves the signed-in user's own dashboard
 * layout (UI-SEED-22). Every role may arrange their own dashboard; the path
 * names no user, so one user's save never touches another's.
 */
import { useEffect, useState } from 'react';
import { ApiError, api } from '../api';
import type { DashboardLayout } from '../types/generated/dashboard-layout';
import type { DashboardLayoutRequest } from '../types/generated/dashboard-layout-request';

const dashboardEndpoint = '/api/v1/users/me/dashboard';

function reason(err: unknown): string {
  if (err instanceof ApiError) {
    return err.details || err.message;
  }
  return err instanceof Error ? err.message : String(err);
}

export interface UseDashboardLayoutResult {
  /** The stored layout, or null until the read settles. */
  layout: DashboardLayout | null;
  error: string | null;
  /** Replaces the stored layout; resolves to the refusal reason, or null. */
  save: (widgets: string[]) => Promise<string | null>;
}

export function useDashboardLayout(): UseDashboardLayoutResult {
  const [layout, setLayout] = useState<DashboardLayout | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .get<DashboardLayout>(dashboardEndpoint)
      .then((body) => {
        if (!cancelled) {
          setLayout(body);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(reason(err));
        }
      });
    return (): void => {
      cancelled = true;
    };
  }, []);

  const save = async (widgets: string[]): Promise<string | null> => {
    const body: DashboardLayoutRequest = { widgets };
    try {
      setLayout(await api.put<DashboardLayout>(dashboardEndpoint, body));
    } catch (err) {
      return reason(err);
    }
    setError(null);
    return null;
  };

  return { layout, error, save };
}
