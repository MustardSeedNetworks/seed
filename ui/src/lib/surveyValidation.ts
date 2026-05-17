/**
 * Client-side validation for survey API requests (fixes #725).
 *
 * Centralises the rules so every useSurvey callsite enforces the same
 * constraints before sending data over the wire. On failure each helper
 * returns a stable error code (caller maps to a localised string) plus
 * the offending field name where useful, so the UI can surface clear
 * messages instead of generic 4xx responses from the backend.
 */

export const SURVEY_ID_MAX_LEN = 128;
export const SURVEY_NAME_MAX_LEN = 256;

/** PNG/JPEG max pixel dimension that all major browsers/decoders accept. */
export const FLOORPLAN_MAX_DIMENSION_PX = 16_384;

/** Maximum raw image bytes accepted client-side before submission. */
export const FLOORPLAN_MAX_BYTES = 10 * 1024 * 1024; // 10 MB

/** Minimum valid scale (meters per pixel). Below this is almost certainly a bug. */
export const FLOORPLAN_MIN_SCALE_M = 0.0001;
export const FLOORPLAN_MAX_SCALE_M = 100;

/**
 * Result type used by every validator. `ok=true` means safe to send.
 * `error` is a stable string code; callers translate / display it.
 */
export type ValidationResult = { ok: true } | { ok: false; error: string; field?: string };

const VALID_ID_RE = /^[A-Za-z0-9._-]+$/;

export function validateSurveyId(id: string | undefined | null): ValidationResult {
  if (!id) {
    return { ok: false, error: 'survey.id.empty', field: 'id' };
  }
  if (id.length > SURVEY_ID_MAX_LEN) {
    return { ok: false, error: 'survey.id.tooLong', field: 'id' };
  }
  if (!VALID_ID_RE.test(id)) {
    return { ok: false, error: 'survey.id.invalidChars', field: 'id' };
  }
  return { ok: true };
}

export function validateCoordinate(value: number, axis: 'x' | 'y', max?: number): ValidationResult {
  if (!Number.isFinite(value)) {
    return { ok: false, error: 'survey.coord.notFinite', field: axis };
  }
  if (value < 0) {
    return { ok: false, error: 'survey.coord.negative', field: axis };
  }
  if (max !== undefined && value > max) {
    return { ok: false, error: 'survey.coord.outOfBounds', field: axis };
  }
  return { ok: true };
}

/**
 * Validate a sample's (x, y) within an optional floor-plan extent.
 * Pass the floorPlan if available so out-of-bounds is caught before
 * the click ever reaches the backend.
 */
export function validateCoordinates(
  x: number,
  y: number,
  bounds?: { width: number; height: number },
): ValidationResult {
  const xCheck = validateCoordinate(x, 'x', bounds?.width);
  if (!xCheck.ok) {
    return xCheck;
  }
  return validateCoordinate(y, 'y', bounds?.height);
}

interface FloorPlanLike {
  imageData?: string;
  width: number;
  height: number;
  scaleM: number;
}

/**
 * Estimate decoded byte size of a base-64 data URL or raw base-64 string.
 * Avoids decoding the full image just to measure it.
 */
function approximateBase64Bytes(data: string): number {
  const payload = data.startsWith('data:') ? data.slice(data.indexOf(',') + 1) : data;
  // base64 inflates by 4/3; subtract padding chars.
  let padding = 0;
  if (payload.endsWith('==')) {
    padding = 2;
  } else if (payload.endsWith('=')) {
    padding = 1;
  }
  return Math.floor((payload.length * 3) / 4) - padding;
}

export function validateFloorPlan(plan: FloorPlanLike): ValidationResult {
  if (!Number.isFinite(plan.width) || plan.width <= 0) {
    return { ok: false, error: 'survey.floorPlan.widthInvalid', field: 'width' };
  }
  if (!Number.isFinite(plan.height) || plan.height <= 0) {
    return { ok: false, error: 'survey.floorPlan.heightInvalid', field: 'height' };
  }
  if (plan.width > FLOORPLAN_MAX_DIMENSION_PX || plan.height > FLOORPLAN_MAX_DIMENSION_PX) {
    return { ok: false, error: 'survey.floorPlan.tooLarge', field: 'dimensions' };
  }
  if (
    !Number.isFinite(plan.scaleM) ||
    plan.scaleM < FLOORPLAN_MIN_SCALE_M ||
    plan.scaleM > FLOORPLAN_MAX_SCALE_M
  ) {
    return { ok: false, error: 'survey.floorPlan.scaleInvalid', field: 'scaleM' };
  }
  if (plan.imageData && approximateBase64Bytes(plan.imageData) > FLOORPLAN_MAX_BYTES) {
    return { ok: false, error: 'survey.floorPlan.imageTooLarge', field: 'imageData' };
  }
  return { ok: true };
}

interface CreateSurveyLike {
  name?: string;
  surveyType?: string;
  interface?: string;
  iperfServer?: string;
}

export function validateCreateSurvey(req: CreateSurveyLike): ValidationResult {
  if (!req.name?.trim()) {
    return { ok: false, error: 'survey.create.nameRequired', field: 'name' };
  }
  if (req.name.length > SURVEY_NAME_MAX_LEN) {
    return { ok: false, error: 'survey.create.nameTooLong', field: 'name' };
  }
  if (!req.surveyType) {
    return { ok: false, error: 'survey.create.typeRequired', field: 'surveyType' };
  }
  if (!req.interface?.trim()) {
    return { ok: false, error: 'survey.create.interfaceRequired', field: 'interface' };
  }
  if (req.surveyType === 'throughput' && !req.iperfServer?.trim()) {
    return { ok: false, error: 'survey.create.iperfServerRequired', field: 'iperfServer' };
  }
  return { ok: true };
}
