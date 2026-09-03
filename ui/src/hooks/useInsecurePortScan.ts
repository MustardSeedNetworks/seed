/**
 * useInsecurePortScan — drives the on-demand insecure-port audit (#347).
 *
 * The scan itself already existed: POST /api/v1/security/discovery/portscan
 * with `profile: "insecure"` runs the legacy/cleartext port list that
 * `config.PortsInsecureTCP` defines, the same list the discovery preset uses.
 * What was missing is a way to run it against an arbitrary target on demand,
 * and somewhere to explain why each open port matters.
 *
 * The route is `minRole: op` and rate-limited, so the caller gates the control.
 */

import { useCallback, useState } from 'react';

import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';

/** One port the scanner reported. Mirrors discovery.ServiceInfo. */
export interface ScannedService {
  port: number;
  state: string;
  service: string;
  banner?: string;
  version?: string;
  protocol?: string;
}

/** Mirrors discovery.PortScanResult. */
export interface PortScanResult {
  ip: string;
  hostname?: string;
  services: ScannedService[];
  scanTime: number;
  error?: string;
}

interface InsecurePortScan {
  result: PortScanResult | null;
  scanning: boolean;
  error: string | null;
  scan: (target: string) => Promise<void>;
  reset: () => void;
}

export function useInsecurePortScan(): InsecurePortScan {
  const [result, setResult] = useState<PortScanResult | null>(null);
  const [scanning, setScanning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const scan = useCallback(async (target: string): Promise<void> => {
    setScanning(true);
    setError(null);
    setResult(null);
    try {
      const scanned = await api.post<PortScanResult>('/api/v1/security/discovery/portscan', {
        target,
        profile: 'insecure',
      });
      setResult(scanned);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Insecure port scan failed';
      setError(message);
      logger.error(LogComponents.DISCOVERY, 'Insecure port scan failed', err);
    } finally {
      setScanning(false);
    }
  }, []);

  const reset = useCallback((): void => {
    setResult(null);
    setError(null);
  }, []);

  return { result, scanning, error, scan, reset };
}
