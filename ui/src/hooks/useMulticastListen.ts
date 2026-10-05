/**
 * useMulticastListen — join one multicast group on one interface for a
 * bounded window and report what arrived (#399).
 *
 * A listen is a `multicast-listen` job (see useBoundedJob). A stopped listen
 * keeps what it heard, so stopping ends in a result over the time actually
 * listened.
 */

import type { ListenRequest } from '../types/generated/multicast-listen-request';
import type { ListenResult } from '../types/generated/multicast-listen-response';
import { type BoundedJobState, type UseBoundedJobReturn, useBoundedJob } from './useBoundedJob';

export type { ListenRequest, ListenResult };
export type ListenState = BoundedJobState<ListenResult>;

function isListenResult(result: unknown): result is ListenResult {
  return (
    typeof result === 'object' &&
    result !== null &&
    typeof (result as { group?: unknown }).group === 'string' &&
    Array.isArray((result as { sources?: unknown }).sources)
  );
}

export function useMulticastListen(): UseBoundedJobReturn<ListenRequest, ListenResult> {
  return useBoundedJob('multicast-listen', isListenResult);
}
