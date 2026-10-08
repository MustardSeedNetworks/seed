/**
 * usePathChecks — the two one-shot path checks on /path as bounded jobs:
 * multi-path egress (#395, `path-multipath`) and path MTU discovery (#435,
 * `path-mtu`). Both take only a destination, and a stopped run ends in
 * `succeeded` with what it had measured.
 */

import type { PathMTURequest } from '../types/generated/path-mtu-request';
import type { PMTUDResult } from '../types/generated/path-mtu-response';
import type { MultiPathRequest } from '../types/generated/path-multipath-request';
import type { MultiPathResult, PathVariant } from '../types/generated/path-multipath-response';
import { type UseBoundedJobReturn, useBoundedJob } from './useBoundedJob';

export type { MultiPathRequest, MultiPathResult, PathMTURequest, PathVariant, PMTUDResult };

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

export function isMultiPathResult(value: unknown): value is MultiPathResult {
  return (
    isObject(value) &&
    typeof value.attempts === 'number' &&
    typeof value.divergesAtTtl === 'number' &&
    Array.isArray(value.paths)
  );
}

export function isPathMTUResult(value: unknown): value is PMTUDResult {
  return (
    isObject(value) &&
    typeof value.status === 'string' &&
    typeof value.pathMtu === 'number' &&
    typeof value.localMtu === 'number'
  );
}

export function useMultiPath(): UseBoundedJobReturn<MultiPathRequest, MultiPathResult> {
  return useBoundedJob('path-multipath', isMultiPathResult);
}

export function usePathMTU(): UseBoundedJobReturn<PathMTURequest, PMTUDResult> {
  return useBoundedJob('path-mtu', isPathMTUResult);
}
