/**
 * usePacketCapture — run one packet capture through the jobs spine (#326).
 *
 * A capture is a `packet-capture` job (see useBoundedJob). A stopped capture
 * keeps what it recorded, so stopping ends in a result whose stopReason is
 * "stopped". The result names the file GET /api/v1/captures/{id} downloads,
 * and carries its summary (#239).
 */

import type { Request as CaptureRequest } from '../types/generated/packet-capture-request';
import type { Result } from '../types/generated/packet-capture-response';
import { type BoundedJobState, type UseBoundedJobReturn, useBoundedJob } from './useBoundedJob';

export type { CaptureRequest };
export type CaptureResult = Result;
export type CaptureState = BoundedJobState<CaptureResult>;

function isCaptureResult(result: unknown): result is CaptureResult {
  return (
    typeof result === 'object' &&
    result !== null &&
    typeof (result as { id?: unknown }).id === 'string' &&
    typeof (result as { summary?: unknown }).summary === 'object'
  );
}

export function usePacketCapture(): UseBoundedJobReturn<CaptureRequest, CaptureResult> {
  return useBoundedJob('packet-capture', isCaptureResult);
}
