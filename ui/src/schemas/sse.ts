/**
 * Valibot schemas for the seed SSE channel.
 *
 * The SSE stream pushes JSON frames from the backend; each frame has a
 * `type` discriminator and a per-type `payload`. Before this layer,
 * `ui/src/hooks/useSse.ts` carried two inline guards (`isValidMessage`,
 * `isValidCardUpdate`) that drifted independently from the Go side.
 * Centralising the shapes here:
 *
 *   - keeps one source of truth (the schema) for both the runtime
 *     check and the static type (via v.InferOutput),
 *   - lets the hook drop invalid frames cleanly (safeParse → log →
 *     skip) instead of crashing the page,
 *   - prepares the ground for additional frame types (telemetry,
 *     alerts, etc.) — each adds one schema and one branch.
 */
import * as v from 'valibot';

/**
 * SseMessage — the outer envelope. Every frame has a string `type` and
 * an opaque `payload`. The hook narrows `payload` via per-type schemas
 * (e.g., SseCardUpdateSchema) once it knows the `type`.
 */
export const SseMessageSchema = v.object({
  type: v.string(),
  payload: v.unknown(),
});

export type SseMessage = v.InferOutput<typeof SseMessageSchema>;

/**
 * SseCardUpdate — payload for `type: "card_update"` frames. The hook
 * dispatches these to a separate callback so card components don't
 * have to re-parse the envelope.
 */
export const SseCardUpdateSchema = v.object({
  cardId: v.string(),
  data: v.unknown(),
  interface: v.optional(v.string()),
});

export type SseCardUpdate = v.InferOutput<typeof SseCardUpdateSchema>;

/**
 * parseSseMessage — runs the envelope schema on a raw `unknown` (the
 * result of `JSON.parse(event.data)`). Returns the typed message on
 * success or `null` so the consumer can drop + log without branching
 * on a structured error.
 */
export function parseSseMessage(value: unknown): SseMessage | null {
  const result = v.safeParse(SseMessageSchema, value);
  return result.success ? result.output : null;
}

/**
 * parseSseCardUpdate — runs the per-type schema on a card_update
 * payload. Returns the parsed payload or null on shape mismatch.
 */
export function parseSseCardUpdate(value: unknown): SseCardUpdate | null {
  const result = v.safeParse(SseCardUpdateSchema, value);
  return result.success ? result.output : null;
}
