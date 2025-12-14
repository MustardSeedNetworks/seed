import { useState, useEffect, useCallback } from "react";
import { getAuthHeaders } from "./useAuth";

const API_BASE = import.meta.env.VITE_API_BASE || "";

export type SurveyType = "passive" | "active" | "throughput";
export type SurveyStatus = "created" | "in_progress" | "paused" | "completed";

export interface FloorPlan {
  imageData: string; // Base64-encoded image
  width: number;
  height: number;
  scaleM: number; // Meters per pixel
}

export interface PassiveSample {
  networks: Array<{
    ssid: string;
    bssid: string;
    rssi: number;
    channel: number;
    frequency: number;
  }>;
}

export interface ActiveSample {
  ssid: string;
  bssid: string;
  rssi: number;
  dataRate: number;
  roamingEvent: boolean;
}

export interface ThroughputSample {
  ssid: string;
  bssid: string;
  rssi: number;
  downloadMbps: number;
  uploadMbps: number;
  latency: number;
  jitter: number;
  packetLoss: number;
}

export interface SamplePoint {
  x: number;
  y: number;
  timestamp: string;
  sampleData: PassiveSample | ActiveSample | ThroughputSample;
}

export interface Survey {
  id: string;
  name: string;
  description?: string;
  floorPlan?: FloorPlan;
  surveyType: SurveyType;
  status: SurveyStatus;
  createdAt: string;
  updatedAt: string;
  samples: SamplePoint[];
  interface: string;
  iperfServer?: string;
  testDuration?: number;
}

export interface CreateSurveyRequest {
  name: string;
  description?: string;
  surveyType: SurveyType;
  interface: string;
  iperfServer?: string;
  testDuration?: number;
}

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

  const createSurvey = useCallback(async (request: CreateSurveyRequest): Promise<Survey> => {
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
  }, [listSurveys]);

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

  const deleteSurvey = useCallback(async (id: string) => {
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
  }, [listSurveys]);

  const startSurvey = useCallback(async (id: string) => {
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
  }, [listSurveys]);

  const pauseSurvey = useCallback(async (id: string) => {
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
  }, [listSurveys]);

  const completeSurvey = useCallback(async (id: string) => {
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
  }, [listSurveys]);

  const addSample = useCallback(async (
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
  }, []);

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
