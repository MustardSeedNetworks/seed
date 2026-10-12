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

import { useState } from 'react';

import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';
import type { PortScanRequest } from '../types/generated/port-scan-request';
import type { PortScanResponse } from '../types/generated/port-scan-response';

interface InsecurePortScan {
  result: PortScanResponse | null;
  scanning: boolean;
  error: string | null;
  scan: (target: string) => Promise<void>;
  reset: () => void;
}

export function useInsecurePortScan(): InsecurePortScan {
  const [result, setResult] = useState<PortScanResponse | null>(null);
  const [scanning, setScanning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const scan = async (target: string): Promise<void> => {
    setScanning(true);
    setError(null);
    setResult(null);
    try {
      const request: PortScanRequest = { target, profile: 'insecure' };
      const scanned = await api.post<PortScanResponse>(
        '/api/v1/security/discovery/portscan',
        request,
      );
      setResult(scanned);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Insecure port scan failed';
      setError(message);
      logger.error(LogComponents.DISCOVERY, 'Insecure port scan failed', err);
    }
    setScanning(false);
  };

  const reset = (): void => {
    setResult(null);
    setError(null);
  };

  return { result, scanning, error, scan, reset };
}
