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

import { useState, useEffect, useCallback } from "react";
import { getAuthHeaders } from "./useAuth";

// API base URL for survey endpoints
const API_BASE = import.meta.env.VITE_API_BASE || "";

/** Survey data collection mode */
export type SurveyType = "passive" | "active" | "throughput";

/** Survey lifecycle status */
export type SurveyStatus = "created" | "in_progress" | "paused" | "completed";

/** Floor plan image and scale information for coordinate mapping */
export interface FloorPlan {
  imageData: string; // Base64-encoded image data
  width: number; // Image width in pixels
  height: number; // Image height in pixels
  scaleM: number; // Scale factor: meters per pixel
}

/** Passive survey sample data (multiple networks scanned) */
export interface PassiveSample {
  networks: Array<{
    ssid: string; // Network name
    bssid: string; // Access point MAC address
    rssi: number; // Received signal strength indicator (dBm)
    channel: number; // WiFi channel number
    frequency: number; // Frequency in MHz
  }>;
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

/** Complete survey object */
export interface Survey {
  id: string; // Unique survey identifier
  name: string; // Survey name
  description?: string; // Optional description
  floorPlan?: FloorPlan; // Optional floor plan image
  surveyType: SurveyType; // Type of survey (passive/active/throughput)
  status: SurveyStatus; // Current lifecycle status
  createdAt: string; // Creation timestamp (ISO 8601)
  updatedAt: string; // Last update timestamp (ISO 8601)
  samples: SamplePoint[]; // Collected sample points
  interface: string; // WiFi interface used for survey
  iperfServer?: string; // iperf3 server for throughput tests
  testDuration?: number; // Duration of throughput tests (seconds)
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
      const res = await fetch(`${API_BASE}/api/survey/list`, {
        headers: getAuthHeaders(),
      });
      if (res.ok) {
        const data = await res.json();
        setSurveys(data.surveys || []);
      } else {
        throw new Error("Failed to load surveys");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load surveys");
    } finally {
      setLoading(false);
    }
  }, []);

  const createSurvey = useCallback(
    async (request: CreateSurveyRequest): Promise<Survey> => {
      setLoading(true);
      setError(null);
      try {
        const res = await fetch(`${API_BASE}/api/survey/create`, {
          method: "POST",
          headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
          body: JSON.stringify(request),
        });
        if (!res.ok) throw new Error("Failed to create survey");
        const survey = await res.json();
        await listSurveys(); // Refresh list
        return survey;
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : "Failed to create survey";
        setError(errorMsg);
        throw new Error(errorMsg);
      } finally {
        setLoading(false);
      }
    },
    [listSurveys]
  );

  const getSurvey = useCallback(async (id: string): Promise<Survey> => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${API_BASE}/api/survey?id=${id}`, {
        headers: getAuthHeaders(),
      });
      if (!res.ok) throw new Error("Failed to get survey");
      return await res.json();
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : "Failed to get survey";
      setError(errorMsg);
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
        const res = await fetch(`${API_BASE}/api/survey/delete?id=${id}`, {
          method: "DELETE",
          headers: getAuthHeaders(),
        });
        if (!res.ok) throw new Error("Failed to delete survey");
        await listSurveys(); // Refresh list
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to delete survey");
      } finally {
        setLoading(false);
      }
    },
    [listSurveys]
  );

  const startSurvey = useCallback(
    async (id: string) => {
      setLoading(true);
      setError(null);
      try {
        const res = await fetch(`${API_BASE}/api/survey/start?id=${id}`, {
          method: "POST",
          headers: getAuthHeaders(),
        });
        if (!res.ok) throw new Error("Failed to start survey");
        await listSurveys(); // Refresh list
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to start survey");
      } finally {
        setLoading(false);
      }
    },
    [listSurveys]
  );

  const pauseSurvey = useCallback(
    async (id: string) => {
      setLoading(true);
      setError(null);
      try {
        const res = await fetch(`${API_BASE}/api/survey/pause?id=${id}`, {
          method: "POST",
          headers: getAuthHeaders(),
        });
        if (!res.ok) throw new Error("Failed to pause survey");
        await listSurveys(); // Refresh list
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to pause survey");
      } finally {
        setLoading(false);
      }
    },
    [listSurveys]
  );

  const completeSurvey = useCallback(
    async (id: string) => {
      setLoading(true);
      setError(null);
      try {
        const res = await fetch(`${API_BASE}/api/survey/complete?id=${id}`, {
          method: "POST",
          headers: getAuthHeaders(),
        });
        if (!res.ok) throw new Error("Failed to complete survey");
        await listSurveys(); // Refresh list
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to complete survey");
      } finally {
        setLoading(false);
      }
    },
    [listSurveys]
  );

  const addSample = useCallback(
    async (
      id: string,
      x: number,
      y: number,
      sampleData: PassiveSample | ActiveSample | ThroughputSample
    ) => {
      setLoading(true);
      setError(null);
      try {
        const res = await fetch(`${API_BASE}/api/survey/sample?id=${id}`, {
          method: "POST",
          headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
          body: JSON.stringify({ x, y, sampleData }),
        });
        if (!res.ok) throw new Error("Failed to add sample");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to add sample");
      } finally {
        setLoading(false);
      }
    },
    []
  );

  const updateFloorPlan = useCallback(async (id: string, floorPlan: FloorPlan) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${API_BASE}/api/survey/floorplan?id=${id}`, {
        method: "POST",
        headers: { ...getAuthHeaders(), "Content-Type": "application/json" },
        body: JSON.stringify(floorPlan),
      });
      if (!res.ok) throw new Error("Failed to update floor plan");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update floor plan");
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
