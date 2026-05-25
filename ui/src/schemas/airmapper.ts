/**
 * Valibot schemas for the AirMapper .amp file import path.
 *
 * The .amp archive contains a `.serial` JSON file with survey metadata
 * and floor-plan calibration. That JSON comes from a third-party tool
 * (AirMapper) which we can't audit; treating its shape as untrusted is
 * the correct posture even if the tool is benign in practice.
 *
 * Before this layer, `ui/src/utils/airmapper.ts` did `JSON.parse(...) as
 * SerialJson` — a hard cast with no runtime check. A malformed file
 * could surface as undefined-property access deep inside survey logic.
 * The schemas here let `safeParse` surface a structured, user-facing
 * error instead.
 */
import * as v from 'valibot';

export const AirMapperLocationSchema = v.object({
  x: v.number(),
  y: v.number(),
  label: v.string(),
});

export const AirMapperViewSchema = v.object({
  name: v.string(),
  option: v.string(),
  mode: v.string(),
  limit: v.optional(v.number()),
  threshold: v.optional(v.number()),
  filters: v.optional(
    v.array(
      v.object({
        key: v.string(),
        value: v.unknown(),
      }),
    ),
  ),
});

/**
 * Schema for the parsed `.serial` JSON.
 *
 * Every field is optional — AirMapper writes very sparse files in
 * practice, and the parser tolerates missing fields by applying
 * defaults. This schema enforces *shape* (types when present) rather
 * than completeness. Missing-required errors live in the consumer.
 */
export const SerialJsonSchema = v.object({
  fileName: v.optional(v.string()),
  floorPlanFilename: v.optional(v.string()),
  floorPlanScalePpf: v.optional(v.number()),
  floorPlanWidthPx: v.optional(v.number()),
  floorPlanHeightPx: v.optional(v.number()),
  propagation: v.optional(v.number()),
  propagationUnit: v.optional(v.string()),
  surveyName: v.optional(v.string()),
  surveyMode: v.optional(v.string()),
  surveyPointCount: v.optional(v.number()),
  surveyItemsCount: v.optional(v.number()),
  surveyStartTime: v.optional(v.string()),
  surveyActive1x1: v.optional(v.boolean()),
  unitName: v.optional(v.string()),
  unitType: v.optional(v.string()),
  unitSerial: v.optional(v.string()),
  labels: v.optional(v.array(v.string())),
  views: v.optional(v.array(AirMapperViewSchema)),
  locations: v.optional(
    v.object({
      passive: v.optional(v.array(AirMapperLocationSchema)),
      active: v.optional(v.array(AirMapperLocationSchema)),
      oneXone: v.optional(v.array(AirMapperLocationSchema)),
      client: v.optional(v.array(AirMapperLocationSchema)),
      probingClient: v.optional(v.array(AirMapperLocationSchema)),
      bluetooth: v.optional(v.array(AirMapperLocationSchema)),
    }),
  ),
});

export type SerialJson = v.InferOutput<typeof SerialJsonSchema>;

/**
 * Parse the raw text contents of a `.serial` file into a typed
 * SerialJson. Returns either a populated value or a list of
 * human-readable failure reasons suitable for surfacing to the user.
 *
 * Uses safeParse so a malformed file produces a structured error
 * instead of an exception.
 */
export function parseSerialJson(
  text: string,
): { ok: true; value: SerialJson } | { ok: false; reason: string; issues: string[] } {
  let raw: unknown;
  try {
    raw = JSON.parse(text);
  } catch (cause) {
    return {
      ok: false,
      reason: 'Could not parse .serial as JSON.',
      issues: [cause instanceof Error ? cause.message : String(cause)],
    };
  }

  const result = v.safeParse(SerialJsonSchema, raw);
  if (result.success) {
    return { ok: true, value: result.output };
  }

  // valibot returns one issue per failing field; render path + message.
  const issues = result.issues.map((iss) => {
    const path = iss.path?.map((p) => String(p.key)).join('.') ?? '';
    return path ? `${path}: ${iss.message}` : iss.message;
  });

  return {
    ok: false,
    reason: 'The .serial file did not match the expected shape.',
    issues,
  };
}
