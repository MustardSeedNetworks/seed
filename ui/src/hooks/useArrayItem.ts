// biome-ignore-all lint/style/useNamingConvention: Standard TypeScript generic parameter naming (TSettings, TItem, etc.)
/**
 * useArrayItem Hook
 *
 * A generic hook for managing CRUD operations on array items within a settings object.
 * Consolidates duplicate add/remove/update patterns found across settings components.
 *
 * Features:
 * - Type-safe with proper TypeScript generics
 * - Generates unique IDs for new items using crypto.randomUUID()
 * - Provides memoized add, remove, and update callbacks
 * - Handles optional arrays with nullish coalescing
 *
 * Usage:
 * ```typescript
 * const { add, remove, update } = useArrayItem(
 *   setTestsSettings,
 *   'pingTargets',
 *   () => ({ name: "", host: "", enabled: true, count: 3 })
 * );
 *
 * // Add a new item
 * add();
 *
 * // Remove an item by ID
 * remove("item-id");
 *
 * // Update a specific field
 * update("item-id", "name", "Gateway");
 * ```
 *
 * @module hooks/useArrayItem
 */

import type { Dispatch, SetStateAction } from 'react';
import { useCallback } from 'react';
import { generateId } from '../utils/id';

/**
 * Base type for items that can be managed by this hook.
 * Items must have an optional id field for identification.
 */
interface ItemWithId {
  id?: string;
  [key: string]: unknown;
}

/**
 * Extracts array property keys from a settings object type.
 * Only properties that are arrays of objects with optional id fields are valid.
 */
type ArrayKeys<TSettings> = {
  [K in keyof TSettings]: TSettings[K] extends ItemWithId[] | undefined ? K : never;
}[keyof TSettings];

/**
 * Extracts the item type from an array property.
 */
type ArrayItemType<TSettings, K extends keyof TSettings> = TSettings[K] extends
  | Array<infer T>
  | undefined
  ? T extends ItemWithId
    ? T
    : never
  : never;

/**
 * Return type for the useArrayItem hook.
 *
 * @template TItem - The type of items in the managed array
 */
interface UseArrayItemReturn<TItem extends ItemWithId> {
  /**
   * Adds a new item to the array with a generated unique ID.
   * Uses the createDefault function provided to the hook.
   */
  add: () => void;

  /**
   * Removes an item from the array by its ID.
   *
   * @param id - The unique identifier of the item to remove
   */
  remove: (id: string) => void;

  /**
   * Updates a specific field of an item identified by ID.
   *
   * @param id - The unique identifier of the item to update
   * @param field - The field name to update
   * @param value - The new value for the field
   */
  update: <F extends keyof TItem>(id: string, field: F, value: TItem[F]) => void;
}

/**
 * Generic hook for managing CRUD operations on array items within a settings object.
 *
 * Replaces boilerplate patterns like addPingTarget, removePingTarget, updatePingTarget
 * with a single reusable hook that provides type-safe add, remove, and update callbacks.
 *
 * @template TSettings - The type of the settings object containing the array
 * @template K - The key of the array property in the settings object
 *
 * @param setSettings - The React state setter function for the settings object
 * @param arrayKey - The property key of the array to manage within the settings object
 * @param createDefault - A factory function that returns default values for a new item (without id)
 *
 * @returns An object containing memoized add, remove, and update callbacks
 *
 * @example
 * ```typescript
 * // Managing ping targets in test settings
 * const { add: addPingTarget, remove: removePingTarget, update: updatePingTarget } =
 *   useArrayItem(
 *     setTestsSettings,
 *     'pingTargets',
 *     () => ({ name: "", host: "", enabled: true, count: 3 })
 *   );
 *
 * // Add a new ping target
 * addPingTarget();
 *
 * // Remove a ping target
 * removePingTarget("abc-123");
 *
 * // Update a ping target's host
 * updatePingTarget("abc-123", "host", "192.168.1.1");
 * ```
 *
 * @example
 * ```typescript
 * // Managing optional arrays (handles undefined gracefully)
 * const { add, remove, update } = useArrayItem(
 *   setSettings,
 *   'optionalEndpoints', // May be undefined in the settings
 *   () => ({ url: "https://", enabled: true })
 * );
 * ```
 */
export function useArrayItem<
  TSettings extends Record<string, unknown>,
  K extends ArrayKeys<TSettings>,
>(
  setSettings: Dispatch<SetStateAction<TSettings>>,
  arrayKey: K,
  createDefault: () => Omit<ArrayItemType<TSettings, K>, 'id'>,
): UseArrayItemReturn<ArrayItemType<TSettings, K>> {
  type Item = ArrayItemType<TSettings, K>;

  /**
   * Adds a new item to the array with a generated unique ID.
   */
  const add = useCallback(() => {
    setSettings((prev) => {
      const currentArray = ((prev[arrayKey] as Item[] | undefined) ?? []) as Item[];
      const defaultValues = createDefault();
      const newItem = {
        id: generateId(),
        ...defaultValues,
      } as Item;

      return {
        ...prev,
        [arrayKey]: [...currentArray, newItem],
      };
    });
  }, [setSettings, arrayKey, createDefault]);

  /**
   * Removes an item from the array by its ID.
   */
  const remove = useCallback(
    (id: string) => {
      setSettings((prev) => {
        const currentArray = ((prev[arrayKey] as Item[] | undefined) ?? []) as Item[];
        return {
          ...prev,
          [arrayKey]: currentArray.filter((item) => item.id !== id),
        };
      });
    },
    [setSettings, arrayKey],
  );

  /**
   * Updates a specific field of an item identified by ID.
   */
  const update = useCallback(
    <F extends keyof Item>(id: string, field: F, value: Item[F]) => {
      setSettings((prev) => {
        const currentArray = ((prev[arrayKey] as Item[] | undefined) ?? []) as Item[];
        return {
          ...prev,
          [arrayKey]: currentArray.map((item) =>
            item.id === id ? { ...item, [field]: value } : item,
          ),
        };
      });
    },
    [setSettings, arrayKey],
  );

  return { add, remove, update };
}
