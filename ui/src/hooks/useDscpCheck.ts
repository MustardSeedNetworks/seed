/**
 * useDscpCheck — run one DSCP preservation check (#400) as a bounded job.
 *
 * Across two hosts, `qos-listen` holds a port open on the far side and reads
 * the marking every probe arrives with, and `qos-send` puts the marked probes
 * on the wire. On one host, `qos-single-host` sends out of one interface and
 * reads them back on the other. The card runs one check at a time and locks
 * the mode while it runs, so one hook serves all three kinds.
 */

import type { ListenRequest } from '../types/generated/qos-listen-request';
import type { ClassResult, ListenResult } from '../types/generated/qos-listen-response';
import type { SendRequest } from '../types/generated/qos-send-request';
import type { SendResult } from '../types/generated/qos-send-response';
import type { SingleHostRequest } from '../types/generated/qos-single-host-request';
import type { SingleHostResult } from '../types/generated/qos-single-host-response';
import { type BoundedJobState, type UseBoundedJobReturn, useBoundedJob } from './useBoundedJob';

export type { ClassResult, ListenResult, SendResult, SingleHostResult };

export type DscpMode = 'listen' | 'send' | 'single-host';
export type DscpRequest = ListenRequest | SendRequest | SingleHostRequest;
export type DscpResult = ListenResult | SendResult | SingleHostResult;
export type DscpState = BoundedJobState<DscpResult>;

const KINDS: Record<DscpMode, string> = {
  listen: 'qos-listen',
  send: 'qos-send',
  'single-host': 'qos-single-host',
};

export function isListenResult(result: DscpResult): result is ListenResult {
  return 'runs' in result;
}

export function isSingleHostResult(result: DscpResult): result is SingleHostResult {
  return 'sendInterface' in result;
}

/** Whether result is the shape mode's job kind answers with. */
export function isModeResult(mode: DscpMode, result: unknown): result is DscpResult {
  if (typeof result !== 'object' || result === null) {
    return false;
  }
  const fields = result as Record<string, unknown>;
  switch (mode) {
    case 'listen':
      return Array.isArray(fields.runs);
    case 'send':
      return typeof fields.marked === 'boolean' && Array.isArray(fields.classes);
    case 'single-host':
      return typeof fields.sendInterface === 'string' && Array.isArray(fields.classes);
  }
}

export function useDscpCheck(mode: DscpMode): UseBoundedJobReturn<DscpRequest, DscpResult> {
  return useBoundedJob(KINDS[mode], (result: unknown): result is DscpResult =>
    isModeResult(mode, result),
  );
}
