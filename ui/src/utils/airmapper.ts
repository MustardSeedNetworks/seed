/**
 * AirMapper File Parser
 *
 * Purpose: Parse NetAlly AirMapper .amp files for import into surveys.
 * The .amp format is a ZIP archive containing:
 * - .serial - JSON metadata with calibration, settings, and locations
 * - .jpg - Floor plan image
 * - .SurveyResult - Binary survey data (complex protobuf-like format)
 * - .acsx - Site configuration (often empty)
 *
 * Key Features:
 * - Extract floor plan image with proper scale
 * - Import calibration (pixels per foot, propagation)
 * - Import AP and client locations
 * - Extract survey metadata (views, settings)
 *
 * Usage:
 * ```typescript
 * const file = await fetch('survey.amp');
 * const data = await file.arrayBuffer();
 * const result = await parseAirMapperFile(data);
 * if (result.success) {
 *   console.log(result.data.calibration);
 * }
 * ```
 */

// biome-ignore lint/style/useNamingConvention: JSZip is a third-party library name
import JSZip from "jszip";

/** Location point from AirMapper */
export interface AirMapperLocation {
  x: number;
  y: number;
  label: string;
}

/** View configuration from AirMapper */
export interface AirMapperView {
  name: string;
  option: string;
  mode: string;
  limit?: number;
  threshold?: number;
  filters?: Array<{ key: string; value: unknown }>;
}

/** Calibration data extracted from .serial */
export interface AirMapperCalibration {
  scalePpf: number; // Pixels per foot
  scaleM: number; // Meters per pixel (calculated)
  propagation: number; // Propagation radius
  propagationUnit: string; // "ft" or "m"
  propagationM: number; // Propagation in meters (calculated)
  widthPx: number;
  heightPx: number;
}

/** Survey metadata from .serial */
export interface AirMapperMetadata {
  fileName: string;
  surveyName: string;
  surveyMode: string;
  surveyPointCount: number;
  surveyItemsCount: number;
  surveyStartTime: string;
  unitName: string;
  unitType: string;
  unitSerial: string;
  hasActiveData: boolean;
  labels: string[];
}

/** Locations data from .serial */
export interface AirMapperLocations {
  passive: AirMapperLocation[];
  active: AirMapperLocation[];
  oneXone: AirMapperLocation[];
  client: AirMapperLocation[];
  probingClient: AirMapperLocation[];
  bluetooth: AirMapperLocation[];
}

/** Complete parsed AirMapper data */
export interface AirMapperData {
  calibration: AirMapperCalibration;
  metadata: AirMapperMetadata;
  locations: AirMapperLocations;
  views: AirMapperView[];
  floorPlanImage: string; // Base64 data URL
  floorPlanFilename: string;
  rawSerial: unknown; // Full .serial JSON for advanced use
}

/** Parse result */
export interface AirMapperParseResult {
  success: boolean;
  data?: AirMapperData;
  error?: string;
  warnings: string[];
}

/** Raw .serial JSON structure (partial) */
interface SerialJson {
  fileName?: string;
  floorPlanFilename?: string;
  floorPlanScalePpf?: number;
  floorPlanWidthPx?: number;
  floorPlanHeightPx?: number;
  propagation?: number;
  propagationUnit?: string;
  surveyName?: string;
  surveyMode?: string;
  surveyPointCount?: number;
  surveyItemsCount?: number;
  surveyStartTime?: string;
  surveyActive1x1?: boolean;
  unitName?: string;
  unitType?: string;
  unitSerial?: string;
  labels?: string[];
  views?: AirMapperView[];
  locations?: {
    passive?: AirMapperLocation[];
    active?: AirMapperLocation[];
    oneXone?: AirMapperLocation[];
    client?: AirMapperLocation[];
    probingClient?: AirMapperLocation[];
    bluetooth?: AirMapperLocation[];
  };
}

/**
 * Convert pixels per foot to meters per pixel
 */
function ppfToMpp(ppf: number): number {
  // 1 foot = 0.3048 meters
  // ppf = pixels per foot
  // mpp = meters per pixel = 0.3048 / ppf
  return 0.3048 / ppf;
}

/**
 * Convert propagation value to meters
 */
function toMeters(value: number, unit: string): number {
  if (unit.toLowerCase() === "ft" || unit.toLowerCase() === "feet") {
    return value * 0.3048;
  }
  return value; // Assume meters
}

/**
 * Parse an AirMapper .amp file
 *
 * @param data - ArrayBuffer of the .amp file
 * @returns Parse result with extracted data or error
 */
export async function parseAirMapperFile(data: ArrayBuffer): Promise<AirMapperParseResult> {
  const warnings: string[] = [];

  try {
    // Load ZIP archive
    const zip = await JSZip.loadAsync(data);

    // Find files by extension
    let serialFile: JSZip.JSZipObject | null = null;
    let jpgFile: JSZip.JSZipObject | null = null;
    let surveyResultFile: JSZip.JSZipObject | null = null;

    for (const [filename, file] of Object.entries(zip.files)) {
      if (file.dir) continue;

      const lowerName = filename.toLowerCase();
      if (lowerName.endsWith(".serial")) {
        serialFile = file;
      } else if (lowerName.endsWith(".jpg") || lowerName.endsWith(".jpeg")) {
        jpgFile = file;
      } else if (lowerName.endsWith(".surveyresult")) {
        surveyResultFile = file;
      }
    }

    // Validate required files
    if (!serialFile) {
      return {
        success: false,
        error: "Missing .serial metadata file in .amp archive",
        warnings,
      };
    }

    if (!jpgFile) {
      warnings.push("No floor plan image found in archive");
    }

    if (!surveyResultFile) {
      warnings.push("No .SurveyResult file found - survey sample data will not be imported");
    }

    // Parse .serial JSON
    const serialContent = await serialFile.async("text");
    let serialJson: SerialJson;
    try {
      serialJson = JSON.parse(serialContent);
    } catch {
      return {
        success: false,
        error: "Failed to parse .serial JSON metadata",
        warnings,
      };
    }

    // Extract calibration
    const scalePpf = serialJson.floorPlanScalePpf || 10; // Default to 10 ppf
    const propagation = serialJson.propagation || 8;
    const propagationUnit = serialJson.propagationUnit || "ft";

    const calibration: AirMapperCalibration = {
      scalePpf,
      scaleM: ppfToMpp(scalePpf),
      propagation,
      propagationUnit,
      propagationM: toMeters(propagation, propagationUnit),
      widthPx: serialJson.floorPlanWidthPx || 0,
      heightPx: serialJson.floorPlanHeightPx || 0,
    };

    // Extract metadata
    const metadata: AirMapperMetadata = {
      fileName: serialJson.fileName || "Unknown",
      surveyName: serialJson.surveyName || serialJson.fileName || "Imported Survey",
      surveyMode: serialJson.surveyMode || "passive",
      surveyPointCount: serialJson.surveyPointCount || 0,
      surveyItemsCount: serialJson.surveyItemsCount || 0,
      surveyStartTime: serialJson.surveyStartTime || new Date().toISOString(),
      unitName: serialJson.unitName || "Unknown Device",
      unitType: serialJson.unitType || "Unknown",
      unitSerial: serialJson.unitSerial || "",
      hasActiveData: serialJson.surveyActive1x1 || false,
      labels: serialJson.labels || [],
    };

    // Extract locations
    const rawLocations = serialJson.locations || {};
    const locations: AirMapperLocations = {
      passive: rawLocations.passive || [],
      active: rawLocations.active || [],
      oneXone: rawLocations.oneXone || [],
      client: rawLocations.client || [],
      probingClient: rawLocations.probingClient || [],
      bluetooth: rawLocations.bluetooth || [],
    };

    // Extract views
    const views: AirMapperView[] = (serialJson.views || []).map((v) => ({
      name: v.name || "Untitled",
      option: v.option || "",
      mode: v.mode || "passive",
      limit: v.limit,
      threshold: v.threshold,
      filters: v.filters,
    }));

    // Extract floor plan image
    let floorPlanImage = "";
    let floorPlanFilename = serialJson.floorPlanFilename || "floorplan.jpg";

    if (jpgFile) {
      const imageData = await jpgFile.async("base64");
      floorPlanImage = `data:image/jpeg;base64,${imageData}`;
      floorPlanFilename = jpgFile.name.split("/").pop() || floorPlanFilename;
    }

    // Build result
    const result: AirMapperData = {
      calibration,
      metadata,
      locations,
      views,
      floorPlanImage,
      floorPlanFilename,
      rawSerial: serialJson,
    };

    return {
      success: true,
      data: result,
      warnings,
    };
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : "Failed to parse .amp file",
      warnings,
    };
  }
}

/**
 * Convert AirMapper locations to survey sample points format
 * Note: This creates placeholder samples since we can't parse the binary data
 */
export function locationsToSamplePoints(
  locations: AirMapperLocations,
  mode: "passive" | "active" = "passive",
): Array<{ x: number; y: number; label: string }> {
  const points: Array<{ x: number; y: number; label: string }> = [];

  // For passive mode, use AP locations
  if (mode === "passive") {
    for (const loc of locations.passive) {
      points.push({
        x: Math.round(loc.x),
        y: Math.round(loc.y),
        label: loc.label,
      });
    }
  }

  // For active mode, use active/1x1 locations
  if (mode === "active") {
    for (const loc of [...locations.active, ...locations.oneXone]) {
      points.push({
        x: Math.round(loc.x),
        y: Math.round(loc.y),
        label: loc.label,
      });
    }
  }

  return points;
}

const API_BASE = import.meta.env.VITE_API_BASE || "";

/** Result from backend AirMapper import API */
export interface BackendImportResult {
  floorPlanImage: string;
  floorPlanFilename: string;
  calibration: {
    scaleM: number;
    propagationM: number;
  };
  apLocations?: Array<{ bssid: string; x: number; y: number; label?: string }>;
  clientLocations?: Array<{
    mac: string;
    x: number;
    y: number;
    label?: string;
  }>;
  passFailCriteria?: Array<{
    option: string;
    name?: string;
    limit: number;
    suffix: string;
    enabled: boolean;
    mode: string;
    ap?: number;
  }>;
  surveyPointCount: number;
  surveyItemsCount: number;
  warnings?: string[];
}

/**
 * Import AirMapper file using backend API
 * This is faster for large files as parsing happens on the server
 *
 * @param file - The .amp file to import
 * @param authHeaders - Auth headers for the API request
 * @returns Import result from the backend
 */
export async function importAirMapperViaBackend(
  file: File,
  authHeaders: HeadersInit,
): Promise<AirMapperParseResult> {
  const formData = new FormData();
  formData.append("file", file);

  try {
    const response = await fetch(`${API_BASE}/api/canopy/survey/import/airmapper`, {
      method: "POST",
      headers: authHeaders,
      body: formData,
    });

    if (!response.ok) {
      const errorText = await response.text();
      return {
        success: false,
        error: errorText || "Failed to import AirMapper file",
        warnings: [],
      };
    }

    const result: BackendImportResult = await response.json();

    // Convert backend result to AirMapperData format
    const data: AirMapperData = {
      calibration: {
        scalePpf: 0, // Not provided by backend
        scaleM: result.calibration.scaleM,
        propagation: result.calibration.propagationM,
        propagationUnit: "m",
        propagationM: result.calibration.propagationM,
        widthPx: 0, // Not provided by backend
        heightPx: 0,
      },
      metadata: {
        fileName: result.floorPlanFilename,
        surveyName: "Imported Survey",
        surveyMode: "passive",
        surveyPointCount: result.surveyPointCount,
        surveyItemsCount: result.surveyItemsCount,
        surveyStartTime: new Date().toISOString(),
        unitName: "AirMapper",
        unitType: "Unknown",
        unitSerial: "",
        hasActiveData: false,
        labels: [],
      },
      locations: {
        passive: (result.apLocations || []).map((ap) => ({
          x: ap.x,
          y: ap.y,
          label: ap.label || ap.bssid,
        })),
        active: [],
        oneXone: [],
        client: (result.clientLocations || []).map((c) => ({
          x: c.x,
          y: c.y,
          label: c.label || c.mac,
        })),
        probingClient: [],
        bluetooth: [],
      },
      views: [],
      floorPlanImage: result.floorPlanImage,
      floorPlanFilename: result.floorPlanFilename,
      rawSerial: result,
    };

    return {
      success: true,
      data,
      warnings: result.warnings || [],
    };
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : "Failed to import AirMapper file",
      warnings: [],
    };
  }
}

/**
 * Get summary statistics from parsed AirMapper data
 */
export function getAirMapperSummary(data: AirMapperData): {
  surveyName: string;
  deviceInfo: string;
  pointCount: number;
  apCount: number;
  clientCount: number;
  hasBothModes: boolean;
  facilitySize: string;
  propagation: string;
} {
  const apCount = data.locations.passive.length;
  const clientCount = data.locations.client.length + data.locations.probingClient.length;
  const hasBothModes = data.metadata.hasActiveData && data.metadata.surveyMode === "passive";

  // Calculate facility size in meters
  const widthM = data.calibration.widthPx * data.calibration.scaleM;
  const heightM = data.calibration.heightPx * data.calibration.scaleM;
  const areaM2 = widthM * heightM;

  return {
    surveyName: data.metadata.surveyName,
    deviceInfo: `${data.metadata.unitName} (${data.metadata.unitType})`,
    pointCount: data.metadata.surveyPointCount,
    apCount,
    clientCount,
    hasBothModes,
    facilitySize: `${widthM.toFixed(1)} × ${heightM.toFixed(1)} m (${areaM2.toFixed(0)} m²)`,
    propagation: `${data.calibration.propagationM.toFixed(1)} m`,
  };
}
