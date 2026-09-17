/**
 * useIsPhone — true below Tailwind's `sm` breakpoint (640px).
 *
 * A media query rather than a CSS class because the caller needs to choose a
 * mount POINT, not a style: the run control lives in the page header on a
 * phone and on the fixed layer on a desktop, and no stylesheet can move an
 * element between two parents. Rendering both and hiding one with `sm:hidden`
 * was the obvious alternative and is worse — it puts two buttons with the same
 * accessible name in the tree, so `getByRole('button', { name: ... })` matches
 * twice and every spec that touches the control has to disambiguate by
 * viewport.
 *
 * Mirrors the matchMedia use in `useTheme.ts`; SSR-safe by defaulting to false
 * when `window` has no matchMedia (jsdom without the test-setup shim).
 */
import { useEffect, useState } from 'react';

/** Tailwind's `sm` breakpoint. A phone is anything narrower. */
const PHONE_QUERY = '(max-width: 639px)';

export function useIsPhone(): boolean {
  const [isPhone, setIsPhone] = useState<boolean>(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
      return false;
    }
    return window.matchMedia(PHONE_QUERY).matches;
  });

  useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
      return;
    }
    const query = window.matchMedia(PHONE_QUERY);
    const onChange = (event: MediaQueryListEvent): void => setIsPhone(event.matches);

    // Sync once on mount: the viewport can have changed between the initial
    // state and the effect, and a rotated phone must not keep the old answer.
    setIsPhone(query.matches);
    query.addEventListener('change', onChange);
    return () => query.removeEventListener('change', onChange);
  }, []);

  return isPhone;
}
