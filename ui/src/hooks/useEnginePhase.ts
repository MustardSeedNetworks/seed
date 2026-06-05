/**
 * useEnginePhase — track the discovery engine's current scan phase name.
 *
 * The engine-scan job (useEngineScan) carries lifecycle state + a progress
 * fraction, but not the human phase NAME — that rides the engine event bus.
 * This hook subscribes to /api/v1/discovery/engine/events and tracks the
 * latest `scan.progress` phase (S4.2's EventScanProgress), resetting on
 * `scan.started`. The discovery card pairs it with useEngineScan to show
 * "bar + % + phase" without the heavier legacy pipeline progress payloads.
 *
 * Structurally mirrors useJobEvents (named-event EventSource, callback-free
 * since it owns its own state, browser-managed reconnect).
 */

import { useEffect, useState } from 'react';
import { LogComponents, logger } from '../lib/logger';

/** Path of the engine event stream. */
const ENGINE_EVENTS_URL = '/api/v1/discovery/engine/events';

/** Shape of a scan.progress event payload (engine NewScanProgressEvent). */
interface ScanProgressPayload {
  phase?: string;
  fraction?: number;
}

/**
 * useEnginePhase returns the current scan phase name (empty between scans /
 * before the first phase completes). Self-resets on scan.started so a new
 * scan does not briefly show the previous run's last phase.
 */
export function useEnginePhase(): { phase: string } {
  const [phase, setPhase] = useState<string>('');

  useEffect(() => {
    const fullUrl = `${window.location.protocol}//${window.location.host}${ENGINE_EVENTS_URL}`;
    const source = new EventSource(fullUrl, { withCredentials: true });

    source.addEventListener('scan.started', () => setPhase(''));

    source.addEventListener('scan.progress', (event: MessageEvent<string>) => {
      try {
        const data = JSON.parse(event.data) as { payload?: ScanProgressPayload };
        if (data.payload?.phase) {
          setPhase(data.payload.phase);
        }
      } catch (error) {
        logger.error(LogComponents.SSE, 'Failed to parse scan.progress event', error, {
          data: event.data,
        });
      }
    });

    return () => source.close();
  }, []);

  return { phase };
}
