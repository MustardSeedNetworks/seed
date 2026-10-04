/**
 * useIperfServerSync keeps the iperf3 listener matching the operator's setting.
 *
 * Lifted out of PerformanceCard, where it was the one piece of that card that
 * writes: POST /api/v1/telemetry/iperf/server is registered `minRole: op`, and
 * nothing in the card asks for it — the reconciliation runs on mount. So a
 * viewer's dashboard fired a silent 403 on every load against a listener that
 * was never theirs to change (#1254). The role check lives here, next to the
 * call it guards, rather than in the card that happens to host it.
 *
 * The status reads go through the api client too, so the two unwrapped reads
 * this replaces leave the #2389 baseline.
 */

import { useEffect, useRef } from 'react';
import { api } from '../api';
import { useRole } from '../contexts/RoleContext';
import { LogComponents, logger } from '../lib/logger';

export interface IperfServerStatus {
  running: boolean;
  port: number;
  pid: number;
  error?: string;
}

const STATUS_PATH = '/api/v1/telemetry/iperf/server/status';
const SERVER_PATH = '/api/v1/telemetry/iperf/server';

interface UseIperfServerSyncArgs {
  /** False until the iperf3 binary is known to be present. */
  installed: boolean;
  enableServer: boolean;
  serverPort: number;
  /** Called with the listener's state after each successful start or stop. */
  onStatus: (status: IperfServerStatus) => void;
}

export function useIperfServerSync({
  installed,
  enableServer,
  serverPort,
  onStatus,
}: UseIperfServerSyncArgs): void {
  const { canWrite } = useRole();
  const initialSyncDone = useRef(false);

  const manageServer = async (shouldRun: boolean, port: number): Promise<void> => {
    await api
      .post(SERVER_PATH, { action: shouldRun ? 'start' : 'stop', port })
      .then(() => api.get<IperfServerStatus>(STATUS_PATH))
      .then(onStatus)
      .catch((err: unknown) => {
        logger.error(LogComponents.IPERF, 'Failed to manage iperf server', err);
      });
  };

  useEffect(() => {
    if (!installed || !canWrite) {
      return;
    }

    if (initialSyncDone.current && !enableServer) {
      // The setting was turned off after the initial sync: stop, no read first.
      manageServer(false, serverPort).catch(() => {
        // Already logged.
      });
      return;
    }

    const sync = async (): Promise<void> => {
      try {
        const status = await api.get<IperfServerStatus>(STATUS_PATH);
        if (enableServer !== status.running) {
          await manageServer(enableServer, serverPort);
        }
      } catch {
        // The status read failed, so the listener's state is unknown. Starting
        // is idempotent on the server side; stopping something that may not be
        // running is not worth a blind write.
        if (enableServer) {
          await manageServer(true, serverPort);
        }
      }
      initialSyncDone.current = true;
    };
    sync().catch(() => {
      // Already logged.
    });
  }, [canWrite, installed, enableServer, serverPort, manageServer]);
}
