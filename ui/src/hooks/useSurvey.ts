/**
 * WiFi Survey Management Hook
 *
 * Manages WiFi site surveys for signal strength mapping and coverage analysis.
 *
 * Features:
 * - Multiple survey types: passive scanning, active connection monitoring, throughput testing
 * - Floor plan integration with coordinate mapping
 * - Sample collection and storage
 * - Survey lifecycle management (create, start, pause, resume, complete)
 * - Real-time sampling with location tracking
 *
 * Survey Types:
 * - **Passive**: Scans for all available WiFi networks without connecting
 * - **Active**: Monitors connected WiFi network performance (RSSI, data rate, roaming)
 * - **Throughput**: Performs iperf3 speed tests at each sample point
 *
 * Usage:
 * ```typescript
 * const { surveys, createSurvey, addSample, loading } = useSurvey();
 *
 * // Create a new survey
 * await createSurvey({
 *   name: 'Office Coverage',
 *   surveyType: 'passive',
 *   interface: 'wlan0'
 * });
 *
 * // Add sample point at coordinates
 * await addSample(surveyId, { x: 100, y: 200 });
 * ```
 */

import { useCallback, useEffect, useState } from "react";
import { api } from "../lib/api";
import { LogComponents, logger } from "../lib/logger";

/** Survey data collection mode */
export type SurveyType = "passive" | "active" | "throughput";

/** Survey lifecycle status */
export type SurveyStatus = "created" | "in_progress" | "paused" | "completed";

/** How the floor plan scale was determined */
export type ScaleSource = "auto" | "dimensions" | "calibration" | "imported" | "default";

/** AP location marker for manual or imported AP placements */
export interface ApLocation {
  id: string; // Unique identifier
  x: number; // X coordinate on floor plan (pixels)
  y: number; // Y coordinate on floor plan (pixels)
  label: string; // Display name (e.g., "AP-101")
  bssid?: string; // MAC address if known
  ssids?: string[]; // Advertised SSIDs
  band?: WiFiBand; // Primary band
  channel?: number; // Primary channel
  model?: string; // AP model (e.g., "Cisco 9130")
  source?: "manual" | "imported" | "detected"; // How this AP was added
}

/** Heatmap visualization metric types */
export type HeatmapMetric =
  | "rssi"
  | "throughput"
  | "latency"
  | "snr"
  | "noise"
  | "cochannel"
  | "adjacent"
  | "channelUtil"
  | "apDensity"
  | "ssidCount"
  | null;

/** Survey view mode */
export type SurveyViewMode = "passive" | "active" | "client" | "probingClient" | "all";

/** Heatmap filter configuration */
export interface HeatmapFilter {
  ssid?: string; // Filter to specific SSID
  bssid?: string; // Filter to specific BSSID/AP
  band?: WiFiBand; // Filter to specific band
  channel?: number; // Filter to specific channel
  apId?: string; // Filter to specific AP location
  minRssi?: number; // Minimum RSSI threshold
  channelWidth?: ChannelWidth; // Filter by channel width
  phyType?: PhyType; // Filter by 802.11 standard
  security?: SecurityType; // Filter by security type
  vendor?: string; // Filter by AP vendor
  viewMode?: SurveyViewMode; // Which survey data to show
}

/** Floor plan image and scale information for coordinate mapping */
export interface FloorPlan {
  imageData: string; // Base64-encoded image data
  width: number; // Image width in pixels
  height: number; // Image height in pixels
  scaleM: number; // Scale factor: meters per pixel
  scaleSource?: ScaleSource; // How scale was determined
  propagationM?: number; // Signal propagation radius in meters (for survey planning)
  originalFile?: string; // Original filename for reference
}

/** 802.11 PHY type identifier */
export type PhyType = "a" | "b" | "g" | "n" | "ac" | "ax" | "be" | "unknown";

/** WiFi security type */
export type SecurityType =
  | "open"
  | "wep"
  | "wpa"
  | "wpa2"
  | "wpa3"
  | "wpa2-enterprise"
  | "wpa3-enterprise"
  | "unknown";

/** Channel width in MHz */
export type ChannelWidth = 20 | 40 | 80 | 160 | 320;

/** Network information from passive scan */
export interface ScannedNetwork {
  ssid: string; // Network name
  bssid: string; // Access point MAC address
  rssi: number; // Received signal strength indicator (dBm)
  channel: number; // Primary WiFi channel number
  frequency: number; // Frequency in MHz
  channelWidth?: ChannelWidth; // Channel width (20/40/80/160/320 MHz)
  phyType?: PhyType; // 802.11 standard (a/b/g/n/ac/ax/be)
  security?: SecurityType; // Security type
  noiseFloor?: number; // Noise floor in dBm (if available)
  snr?: number; // Signal-to-noise ratio (if available)
  txRate?: number; // Transmit rate in Mbps
  rxRate?: number; // Receive rate in Mbps
  vendor?: string; // AP vendor (from OUI lookup)
  isHidden?: boolean; // Hidden SSID
}

/** Passive survey sample data (multiple networks scanned) */
export interface PassiveSample {
  networks: ScannedNetwork[];
  noiseFloor?: number; // Ambient noise floor at this location
  timestamp?: string; // Sample timestamp
}

/** Active survey sample data (connected network only) */
export interface ActiveSample {
  ssid: string; // Connected network name
  bssid: string; // Connected access point MAC
  rssi: number; // Signal strength (dBm)
  dataRate: number; // Current connection speed (Mbps)
  roamingEvent: boolean; // True if this sample includes a roaming event
}

/** Throughput survey sample data (includes speed test results) */
export interface ThroughputSample {
  ssid: string; // Network name
  bssid: string; // Access point MAC
  rssi: number; // Signal strength (dBm)
  downloadMbps: number; // Download speed from iperf3 test
  uploadMbps: number; // Upload speed from iperf3 test
  latency: number; // Average latency (ms)
  jitter: number; // Jitter/variance (ms)
  packetLoss: number; // Packet loss percentage
}

/** Sample point with location and measurement data */
export interface SamplePoint {
  x: number; // X coordinate on floor plan (pixels)
  y: number; // Y coordinate on floor plan (pixels)
  timestamp: string; // ISO 8601 timestamp of sample
  sampleData: PassiveSample | ActiveSample | ThroughputSample; // Measurement data
}

/** WiFi band identifier */
export type WiFiBand = "2.4" | "5" | "6";

/** Configuration for a single WiFi adapter in a survey */
export interface AdapterConfig {
  interface: string; // Interface name (e.g., "wlan0", "en0")
  mode: SurveyType; // Survey mode for this adapter
  bands: WiFiBand[]; // Which bands this adapter scans
  dwellTimeMs?: number; // Per-channel dwell time (optional)
}

/** Survey scan configuration for bands, channels, and adapters */
export interface SurveyConfig {
  // Band configuration
  bands: WiFiBand[]; // Enabled bands ["2.4", "5", "6"]
  channels24Ghz?: number[]; // Specific 2.4GHz channels or empty for all
  channels5Ghz?: number[]; // Specific 5GHz channels or empty for all
  channels6Ghz?: number[]; // Specific 6GHz channels or empty for all

  // Multi-adapter configuration (optional)
  adapters?: AdapterConfig[]; // Per-adapter settings

  // Passive survey filters
  ssidFilter?: string; // SSID filter pattern (include/exclude)
  minRssi?: number; // Minimum RSSI threshold to record

  // Active survey settings
  targetSsid?: string; // Target SSID for active survey
  roamingSensitivity?: "low" | "medium" | "high";
}

/** Default channels for each band */
export const DEFAULT_CHANNELS = {
  "2.4": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11],
  "5": [
    36, 40, 44, 48, 52, 56, 60, 64, 100, 104, 108, 112, 116, 120, 124, 128, 132, 136, 140, 144, 149,
    153, 157, 161, 165,
  ],
  "6": [
    1, 5, 9, 13, 17, 21, 25, 29, 33, 37, 41, 45, 49, 53, 57, 61, 65, 69, 73, 77, 81, 85, 89, 93,
  ],
};

/** Floor in a multi-floor survey */
export interface Floor {
  id: string; // Unique floor identifier
  name: string; // Floor display name (e.g., "Floor 1", "Basement")
  level: number; // Numeric level (-1, 0, 1, 2...)
  floorPlan?: FloorPlan; // Optional floor plan image for this floor
  samples: SamplePoint[]; // Samples collected on this floor
  createdAt: string; // Creation timestamp (ISO 8601)
  updatedAt: string; // Last update timestamp (ISO 8601)
}

/** Complete survey object */
export interface Survey {
  id: string; // Unique survey identifier
  name: string; // Survey name
  description?: string; // Optional description
  floorPlan?: FloorPlan; // Legacy: floor plan (deprecated, use floors)
  surveyType: SurveyType; // Type of survey (passive/active/throughput)
  status: SurveyStatus; // Current lifecycle status
  createdAt: string; // Creation timestamp (ISO 8601)
  updatedAt: string; // Last update timestamp (ISO 8601)
  samples?: SamplePoint[]; // Legacy: samples (deprecated, use floors)
  interface: string; // WiFi interface used for survey
  iperfServer?: string; // iperf3 server for throughput tests
  testDuration?: number; // Duration of throughput tests (seconds)
  config?: SurveyConfig; // Advanced scan configuration (bands, channels, adapters)
  apLocations?: ApLocation[]; // AP markers placed on floor plan
  passFailCriteria?: PassFailCriterion[]; // Custom pass/fail criteria
  lastValidation?: SurveyValidation; // Most recent validation results
  // Multi-floor support (#654)
  floors?: Floor[]; // Multiple floors in the building
  activeFloorId?: string; // Currently active floor for data collection
}

// ============================================================================
// Pass/Fail Criteria System Types
// ============================================================================

/** Comparison operator for pass/fail threshold */
export type ComparisonOperator = "gte" | "lte";

/** A single pass/fail criterion with configurable threshold */
export interface PassFailCriterion {
  id: string; // Unique identifier
  name: string; // Internal name (e.g., "primarySignal")
  displayKey: string; // i18n translation key for display
  metric: HeatmapMetric; // Which metric this criterion tests
  comparison: ComparisonOperator; // >= or <= threshold
  threshold: number; // Threshold value
  suffix: string; // Unit suffix (dBm, dB, %, Mbps, APs, ms)
  enabled: boolean; // Whether this criterion is active
  mode: "passive" | "active" | "throughput" | "all"; // Survey type this applies to
  apIndex?: number; // For "nth strongest AP" tests (0=strongest, 1=second, etc.)
  description?: string; // Optional description of what this criterion measures
}

/** Result of validating a single criterion against survey data */
export interface PassFailResult {
  criterionId: string; // ID of the criterion tested
  criterionName: string; // Name for display
  passed: boolean; // Overall pass/fail status
  averageValue: number; // Average measured value across all samples
  worstValue: number; // Worst measured value
  bestValue: number; // Best measured value
  threshold: number; // Threshold it was tested against
  comparison: ComparisonOperator; // Comparison operator used
  suffix: string; // Unit suffix
  failedSampleCount: number; // How many samples failed this criterion
  totalSampleCount: number; // Total samples tested
  failedLocations: Array<{ x: number; y: number; value: number }>; // Failed sample locations
  percentage: number; // Pass percentage (0-100)
}

/** Complete validation results for a survey */
export interface SurveyValidation {
  overallPass: boolean; // True if all enabled criteria pass
  overallPercentage: number; // Overall pass percentage
  results: PassFailResult[]; // Individual criterion results
  timestamp: string; // When validation was performed
  criteria: PassFailCriterion[]; // Criteria used for this validation
  passedCount: number; // Number of criteria that passed
  failedCount: number; // Number of criteria that failed
  surveyId: string; // Survey that was validated
}

/** Default pass/fail criteria for passive surveys */
export const DEFAULT_PASSIVE_CRITERIA: PassFailCriterion[] = [
  {
    id: "primary-signal",
    name: "primarySignal",
    displayKey: "criteria.primarySignal",
    metric: "rssi",
    comparison: "gte",
    threshold: -65,
    suffix: "dBm",
    enabled: true,
    mode: "passive",
    apIndex: 0,
    description: "Strongest AP signal at each location",
  },
  {
    id: "secondary-signal",
    name: "secondarySignal",
    displayKey: "criteria.secondarySignal",
    metric: "rssi",
    comparison: "gte",
    threshold: -70,
    suffix: "dBm",
    enabled: true,
    mode: "passive",
    apIndex: 1,
    description: "Second strongest AP for roaming redundancy",
  },
  {
    id: "snr",
    name: "snr",
    displayKey: "criteria.snr",
    metric: "snr",
    comparison: "gte",
    threshold: 25,
    suffix: "dB",
    enabled: true,
    mode: "passive",
    description: "Signal-to-noise ratio for reliable connections",
  },
  {
    id: "cochannel",
    name: "coChannel",
    displayKey: "criteria.coChannel",
    metric: "cochannel",
    comparison: "lte",
    threshold: 4,
    suffix: "APs",
    enabled: true,
    mode: "passive",
    description: "Co-channel interference from APs on same channel",
  },
  {
    id: "adjacent",
    name: "adjChannel",
    displayKey: "criteria.adjChannel",
    metric: "adjacent",
    comparison: "lte",
    threshold: 1,
    suffix: "APs",
    enabled: true,
    mode: "passive",
    description: "Adjacent channel interference",
  },
];

/** Default pass/fail criteria for active surveys */
export const DEFAULT_ACTIVE_CRITERIA: PassFailCriterion[] = [
  {
    id: "active-signal",
    name: "activeSignal",
    displayKey: "criteria.activeSignal",
    metric: "rssi",
    comparison: "gte",
    threshold: -65,
    suffix: "dBm",
    enabled: true,
    mode: "active",
    description: "Connected AP signal strength",
  },
  {
    id: "active-snr",
    name: "activeSnr",
    displayKey: "criteria.activeSnr",
    metric: "snr",
    comparison: "gte",
    threshold: 25,
    suffix: "dB",
    enabled: true,
    mode: "active",
    description: "Signal-to-noise ratio while connected",
  },
  {
    id: "active-link-rate",
    name: "activeLinkRate",
    displayKey: "criteria.linkRate",
    metric: "throughput",
    comparison: "gte",
    threshold: 200,
    suffix: "Mbps",
    enabled: true,
    mode: "active",
    description: "Minimum PHY data rate",
  },
];

/** Default pass/fail criteria for throughput surveys */
export const DEFAULT_THROUGHPUT_CRITERIA: PassFailCriterion[] = [
  {
    id: "bandwidth",
    name: "bandwidth",
    displayKey: "criteria.bandwidth",
    metric: "throughput",
    comparison: "gte",
    threshold: 100,
    suffix: "Mbps",
    enabled: true,
    mode: "throughput",
    description: "Minimum measured bandwidth (iperf3)",
  },
  {
    id: "latency",
    name: "latency",
    displayKey: "criteria.latency",
    metric: "latency",
    comparison: "lte",
    threshold: 50,
    suffix: "ms",
    enabled: true,
    mode: "throughput",
    description: "Maximum acceptable latency",
  },
  {
    id: "jitter",
    name: "jitter",
    displayKey: "criteria.jitter",
    metric: "latency",
    comparison: "lte",
    threshold: 10,
    suffix: "ms",
    enabled: true,
    mode: "throughput",
    description: "Maximum acceptable jitter",
  },
];

/**
 * Get default criteria for a survey type
 */
export function getDefaultCriteria(surveyType: SurveyType): PassFailCriterion[] {
  switch (surveyType) {
    case "passive":
      return [...DEFAULT_PASSIVE_CRITERIA];
    case "active":
      return [...DEFAULT_ACTIVE_CRITERIA];
    case "throughput":
      return [...DEFAULT_THROUGHPUT_CRITERIA];
    default:
      return [...DEFAULT_PASSIVE_CRITERIA];
  }
}

/**
 * Get all criteria (combined) for comprehensive surveys
 */
export function getAllCriteria(): PassFailCriterion[] {
  return [...DEFAULT_PASSIVE_CRITERIA, ...DEFAULT_ACTIVE_CRITERIA, ...DEFAULT_THROUGHPUT_CRITERIA];
}

/** Request payload for creating a new survey */
export interface CreateSurveyRequest {
  name: string; // Survey name (required)
  description?: string; // Optional description
  surveyType: SurveyType; // Type of survey to create
  interface: string; // WiFi interface to use
  iperfServer?: string; // Required for throughput surveys
  testDuration?: number; // Test duration for throughput surveys
}

/**
 * Custom hook for managing WiFi site surveys.
 *
 * Provides functions for survey lifecycle management and data collection.
 *
 * @returns Survey state and control functions
 */
export function useSurvey() {
  const [surveys, setSurveys] = useState<Survey[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const listSurveys = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.get<Survey[] | { surveys?: Survey[] }>("/api/survey/list");
      setSurveys(Array.isArray(data) ? data : (data.surveys ?? []));
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : "Failed to load surveys";
      setError(errorMsg);
      // fixes #678 - added structured error logging
      logger.error(LogComponents.Survey, "Failed to list surveys", err, {
        endpoint: "/api/survey/list",
      });
    } finally {
      setLoading(false);
    }
  }, []);

  const createSurvey = useCallback(
    async (request: CreateSurveyRequest): Promise<Survey> => {
      setLoading(true);
      setError(null);
      try {
        const survey = await api.post<Survey>("/api/survey/create", request);
        await listSurveys(); // Refresh list
        logger.info(LogComponents.Survey, "Survey created successfully", {
          surveyId: survey.id,
          name: request.name,
          type: request.surveyType,
        });
        return survey;
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : "Failed to create survey";
        setError(errorMsg);
        // fixes #678 - added structured error logging
        logger.error(LogComponents.Survey, "Failed to create survey", err, {
          endpoint: "/api/survey/create",
          request,
        });
        throw new Error(errorMsg);
      } finally {
        setLoading(false);
      }
    },
    [listSurveys],
  );

  const getSurvey = useCallback(async (id: string): Promise<Survey> => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({ id });
      return await api.get<Survey>(`/api/survey?${params.toString()}`);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : "Failed to get survey";
      setError(errorMsg);
      // fixes #678 - added structured error logging
      logger.error(LogComponents.Survey, "Failed to get survey", err, {
        endpoint: "/api/survey",
        surveyId: id,
      });
      throw new Error(errorMsg);
    } finally {
      setLoading(false);
    }
  }, []);

  const deleteSurvey = useCallback(
    async (id: string) => {
      setLoading(true);
      setError(null);
      try {
        const params = new URLSearchParams({ id });
        await api.delete(`/api/survey/delete?${params.toString()}`);
        await listSurveys(); // Refresh list
        logger.info(LogComponents.Survey, "Survey deleted successfully", {
          surveyId: id,
        });
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : "Failed to delete survey";
        setError(errorMsg);
        // fixes #678 - added structured error logging
        logger.error(LogComponents.Survey, "Failed to delete survey", err, {
          endpoint: "/api/survey/delete",
          surveyId: id,
        });
      } finally {
        setLoading(false);
      }
    },
    [listSurveys],
  );

  const startSurvey = useCallback(
    async (id: string) => {
      setLoading(true);
      setError(null);
      try {
        const params = new URLSearchParams({ id });
        await api.post(`/api/survey/start?${params.toString()}`);
        await listSurveys(); // Refresh list
        logger.info(LogComponents.Survey, "Survey started", { surveyId: id });
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : "Failed to start survey";
        setError(errorMsg);
        // fixes #678 - added structured error logging
        logger.error(LogComponents.Survey, "Failed to start survey", err, {
          endpoint: "/api/survey/start",
          surveyId: id,
        });
      } finally {
        setLoading(false);
      }
    },
    [listSurveys],
  );

  const pauseSurvey = useCallback(
    async (id: string) => {
      setLoading(true);
      setError(null);
      try {
        const params = new URLSearchParams({ id });
        await api.post(`/api/survey/pause?${params.toString()}`);
        await listSurveys(); // Refresh list
        logger.info(LogComponents.Survey, "Survey paused", { surveyId: id });
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : "Failed to pause survey";
        setError(errorMsg);
        // fixes #678 - added structured error logging
        logger.error(LogComponents.Survey, "Failed to pause survey", err, {
          endpoint: "/api/survey/pause",
          surveyId: id,
        });
      } finally {
        setLoading(false);
      }
    },
    [listSurveys],
  );

  const completeSurvey = useCallback(
    async (id: string) => {
      setLoading(true);
      setError(null);
      try {
        const params = new URLSearchParams({ id });
        await api.post(`/api/survey/complete?${params.toString()}`);
        await listSurveys(); // Refresh list
        logger.info(LogComponents.Survey, "Survey completed", { surveyId: id });
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : "Failed to complete survey";
        setError(errorMsg);
        // fixes #678 - added structured error logging
        logger.error(LogComponents.Survey, "Failed to complete survey", err, {
          endpoint: "/api/survey/complete",
          surveyId: id,
        });
      } finally {
        setLoading(false);
      }
    },
    [listSurveys],
  );

  const addSample = useCallback(
    async (
      id: string,
      x: number,
      y: number,
      sampleData: PassiveSample | ActiveSample | ThroughputSample,
    ) => {
      setLoading(true);
      setError(null);
      try {
        const params = new URLSearchParams({ id });
        await api.post(`/api/survey/sample?${params.toString()}`, {
          x,
          y,
          sampleData,
        });
        logger.debug(LogComponents.Survey, "Sample added to survey", {
          surveyId: id,
          x,
          y,
        });
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : "Failed to add sample";
        setError(errorMsg);
        // fixes #678 - added structured error logging
        logger.error(LogComponents.Survey, "Failed to add sample", err, {
          endpoint: "/api/survey/sample",
          surveyId: id,
          coordinates: { x, y },
        });
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  const updateFloorPlan = useCallback(async (id: string, floorPlan: FloorPlan) => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({ id });
      await api.post(`/api/survey/floorplan?${params.toString()}`, floorPlan);
      logger.info(LogComponents.Survey, "Floor plan updated", {
        surveyId: id,
        dimensions: { width: floorPlan.width, height: floorPlan.height },
        scale: floorPlan.scaleM,
      });
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : "Failed to update floor plan";
      setError(errorMsg);
      // fixes #678 - added structured error logging
      logger.error(LogComponents.Survey, "Failed to update floor plan", err, {
        endpoint: "/api/survey/floorplan",
        surveyId: id,
      });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    listSurveys();
  }, [listSurveys]);

  return {
    surveys,
    loading,
    error,
    listSurveys,
    createSurvey,
    getSurvey,
    deleteSurvey,
    startSurvey,
    pauseSurvey,
    completeSurvey,
    addSample,
    updateFloorPlan,
  };
}
