/**
 * useDebouncedAutoSave
 *
 * Debounces a save callback by `delay` ms. Skips the very first invocation
 * (controlled by `isInit`) so opening the drawer doesn't trigger a save
 * before the fetched values have been seeded. Cleans up the timer on
 * re-render and unmount.
 *
 * `enabled` is the role gate. A viewer's drawer loads and normalises the same
 * values an operator's does, and every save behind these is minRole: op, so an
 * unconditional auto-save turns a read into a 403 the user never asked for
 * (#2467). Disabling the controls is not enough — nothing here is a click.
 */

import type React from 'react';
import { useEffect } from 'react';

export function useDebouncedAutoSave(
  saveFn: () => Promise<void> | void,
  isInit: React.MutableRefObject<boolean>,
  timerRef: React.MutableRefObject<ReturnType<typeof setTimeout> | null>,
  enabled: boolean,
  delay = 800,
): void {
  useEffect(() => {
    if (!enabled || isInit.current) {
      return;
    }
    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }
    timerRef.current = setTimeout(() => {
      const result = saveFn();
      if (result && typeof (result as Promise<void>).catch === 'function') {
        (result as Promise<void>).catch(() => undefined);
      }
    }, delay);
    return (): void => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, [saveFn, isInit, timerRef, enabled, delay]);
}
