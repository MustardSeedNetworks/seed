/**
 * Generate a unique ID for React list keys.
 * Uses crypto.randomUUID if available, falls back to timestamp + random.
 */
export function generateId(): string {
  if (typeof crypto !== "undefined" && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2, 11)}`;
}

/**
 * Ensure an item has an ID. If it doesn't, generate one.
 */
export function ensureId<T extends { id?: string }>(
  item: T,
): T & { id: string } {
  if (item.id) {
    return item as T & { id: string };
  }
  return { ...item, id: generateId() };
}

/**
 * Ensure all items in an array have IDs.
 */
export function ensureIds<T extends { id?: string }>(
  items: T[],
): (T & { id: string })[] {
  return items.map(ensureId);
}
