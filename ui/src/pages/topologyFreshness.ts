import type { RecordState } from '../ui/ListDetail';

/**
 * A node's row state from how old its last poll is (seed#2525).
 *
 * Any dated node used to render `ok`, so a switch that stopped answering a
 * week ago stayed green. Age is measured in the node's own poll intervals:
 * within two it is current, within ten it has missed polls and reads `stale`,
 * and past that the reading says nothing about the device now. A node no
 * enabled target polls has no cadence to be late against, so it is `unknown`
 * however recent its last reading.
 */
export function nodeFreshness(
  lastSeen: string,
  pollIntervalSeconds: number | undefined,
  now: number,
): RecordState {
  const seen = Date.parse(lastSeen);
  if (!pollIntervalSeconds || Number.isNaN(seen)) return 'unknown';
  const intervals = (now - seen) / (pollIntervalSeconds * 1000);
  if (intervals <= 2) return 'ok';
  if (intervals <= 10) return 'stale';
  return 'unknown';
}
