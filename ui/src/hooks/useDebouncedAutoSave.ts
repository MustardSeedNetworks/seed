/**
 * useDebouncedAutoSave
 *
 * Debounces a save callback by `delay` ms. Skips the very first invocation
 * (controlled by `isInit`) so opening the drawer doesn't trigger a save
 * before the fetched values have been seeded. Cleans up the timer on
 * re-render and unmount.
 */

import type React from 'react';
import { useEffect } from 'react';

export function useDebouncedAutoSave(
  saveFn: () => Promise<void> | void,
  isInit: React.MutableRefObject<boolean>,
  timerRef: React.MutableRefObject<ReturnType<typeof setTimeout> | null>,
  delay = 800,
): void {
  useEffect(() => {
    if (isInit.current) {
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
  }, [saveFn, isInit, timerRef, delay]);
}
