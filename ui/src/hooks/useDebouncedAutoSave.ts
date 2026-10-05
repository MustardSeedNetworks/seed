/**
 * useDebouncedAutoSave
 *
 * Wraps a settings group's state setter for the controls that edit it: every
 * call through the returned setter schedules `saveFn` after `delay` ms, and a
 * later edit restarts the wait. The loaders keep the raw setter, so seeding
 * the fetched values never saves. An edit is told apart from a load by who
 * calls the setter, not by when: the drawer used to ignore every change for
 * 500 ms after opening, which dropped an edit made inside that window and
 * saved the loaded values when a fetch landed after it (#2994).
 *
 * The save runs the `saveFn` of the render the edit produced, so it sends the
 * value the user typed even if a slow load replaces the shown one meanwhile.
 *
 * `enabled` is the role gate. Every save behind these is minRole: op, so a
 * viewer must never issue one (#2467). It is read through a ref at fire time:
 * RoleProvider clears the user whenever /users/me fails and sets it again on
 * the next success, so canWrite flips false -> true with nothing the operator
 * did, and that alone must not save.
 */

import type React from 'react';
import { useEffect, useRef, useState } from 'react';

export function useDebouncedAutoSave<T>(
  setValue: React.Dispatch<React.SetStateAction<T>>,
  saveFn: () => Promise<void> | void,
  enabled: boolean,
  delay = 800,
): React.Dispatch<React.SetStateAction<T>> {
  const enabledRef = useRef(enabled);
  enabledRef.current = enabled;
  const [edits, setEdits] = useState(0);

  // Deliberately keyed on `edits` alone: `saveFn` changes on every load too,
  // and a load is not an edit.
  useEffect(() => {
    if (edits === 0) {
      return;
    }
    const timer = setTimeout(() => {
      if (!enabledRef.current) {
        return;
      }
      const result = saveFn();
      if (result && typeof (result as Promise<void>).catch === 'function') {
        (result as Promise<void>).catch(() => undefined);
      }
    }, delay);
    return (): void => clearTimeout(timer);
  }, [edits, delay]);

  return (update: React.SetStateAction<T>): void => {
    setValue(update);
    setEdits((n) => n + 1);
  };
}
