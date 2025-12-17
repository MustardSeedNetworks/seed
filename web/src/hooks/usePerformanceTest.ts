/**
 * Performance Test Hook
 *
 * Manages network performance testing including speedtest and iPerf3 operations.
 *
 * Features:
 * - Run speedtest to public servers
 * - Run iPerf3 client tests to custom servers
 * - Manage iPerf3 server mode
 * - Track test progress and results
 * - Get server suggestions for iPerf tests
 *
 * Usage:
 * ```typescript
 * const { runSpeedtest, runIperfClient, speedtestResult } = usePerformanceTest();
 *
 * // Run a speedtest
 * await runSpeedtest();
 *
 * // Run iPerf to a server
 * await runIperfClient({ server: "192.168.1.100", duration: 10 });
 * ```
 */

import { useState, useCallback } from "react";
import { api } from "../lib/api";

/** Speedtest result from public servers */
export interface SpeedtestResult {
  download: number; // Mbps
  upload: number; // Mbps
  latency: number; // ms
  jitter?: number; // ms
  server: {
    name: string;
    location: string;
    host: string;
  };
  timestamp: string;
}

/** iPerf3 test result */
export interface IperfResult {
  bitrateSend: number; // bits/sec
  bitrateReceive: number; // bits/sec
  jitter?: number; // ms
  lostPackets?: number;
  totalPackets?: number;
  duration: number; // seconds
  protocol: "tcp" | "udp";
  server: string;
  timestamp: string;
}

/** iPerf3 client configuration */
export interface IperfClientConfig {
  server: string;
  port?: number;
  duration?: number;
  parallel?: number;
  reverse?: boolean;
  protocol?: "tcp" | "udp";
  bandwidth?: string;
}

/** iPerf3 server status */
export interface IperfServerStatus {
  running: boolean;
  port: number;
  connectedClients: number;
}

/** Speedtest status */
export interface SpeedtestStatus {
  running: boolean;
  phase: "idle" | "download" | "upload" | "complete";
  progress: number;
}

/** iPerf client status */
export interface IperfClientStatus {
  running: boolean;
  server: string;
  progress: number;
  elapsed: number;
}

/** Server suggestion for iPerf tests */
export interface IperfServerSuggestion {
  address: string;
  name: string;
  location: string;
  latency: number;
}

/**
 * Custom hook for managing performance testing operations.
 *
 * Provides functions to run speedtest and iPerf tests, manage iPerf server,
 * and track test progress.
 *
 * @returns Performance test state and control functions
 */
export function usePerformanceTest() {
  const [speedtestRunning, setSpeedtestRunning] = useState(false);
  const [speedtestResult, setSpeedtestResult] = useState<SpeedtestResult | null>(null);
  const [speedtestError, setSpeedtestError] = useState<string | null>(null);

  const [iperfRunning, setIperfRunning] = useState(false);
  const [iperfResult, setIperfResult] = useState<IperfResult | null>(null);
  const [iperfError, setIperfError] = useState<string | null>(null);

  /**
   * Runs a speedtest to public servers.
   */
  const runSpeedtest = useCallback(async (): Promise<SpeedtestResult | null> => {
    try {
      setSpeedtestError(null);
      setSpeedtestRunning(true);
      setSpeedtestResult(null);

      const result = await api.post<SpeedtestResult>("/api/speedtest");
      setSpeedtestResult(result);
      return result;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Speedtest failed";
      setSpeedtestError(message);
      console.error("Speedtest failed:", err);
      return null;
    } finally {
      setSpeedtestRunning(false);
    }
  }, []);

  /**
   * Fetches the current speedtest status.
   */
  const fetchSpeedtestStatus = useCallback(async (): Promise<SpeedtestStatus | null> => {
    try {
      return await api.get<SpeedtestStatus>("/api/speedtest/status");
    } catch (err) {
      console.error("Failed to fetch speedtest status:", err);
      return null;
    }
  }, []);

  /**
   * Runs an iPerf3 client test to the specified server.
   */
  const runIperfClient = useCallback(
    async (config: IperfClientConfig): Promise<IperfResult | null> => {
      try {
        setIperfError(null);
        setIperfRunning(true);
        setIperfResult(null);

        const result = await api.post<IperfResult>("/api/iperf/client", config);
        setIperfResult(result);
        return result;
      } catch (err) {
        const message = err instanceof Error ? err.message : "iPerf test failed";
        setIperfError(message);
        console.error("iPerf client test failed:", err);
        return null;
      } finally {
        setIperfRunning(false);
      }
    },
    []
  );

  /**
   * Fetches the current iPerf client status.
   */
  const fetchIperfClientStatus = useCallback(async (): Promise<IperfClientStatus | null> => {
    try {
      return await api.get<IperfClientStatus>("/api/iperf/client/status");
    } catch (err) {
      console.error("Failed to fetch iPerf client status:", err);
      return null;
    }
  }, []);

  /**
   * Starts the iPerf3 server mode.
   */
  const startIperfServer = useCallback(async (port?: number): Promise<boolean> => {
    try {
      await api.post("/api/iperf/server", { port });
      return true;
    } catch (err) {
      console.error("Failed to start iPerf server:", err);
      return false;
    }
  }, []);

  /**
   * Fetches the iPerf server status.
   */
  const fetchIperfServerStatus = useCallback(async (): Promise<IperfServerStatus | null> => {
    try {
      return await api.get<IperfServerStatus>("/api/iperf/server/status");
    } catch (err) {
      console.error("Failed to fetch iPerf server status:", err);
      return null;
    }
  }, []);

  /**
   * Fetches iPerf server suggestions based on network conditions.
   */
  const fetchIperfSuggestions = useCallback(async (): Promise<IperfServerSuggestion[]> => {
    try {
      const data = await api.get<{ suggestions: IperfServerSuggestion[] }>(
        "/api/iperf/suggestions"
      );
      return data.suggestions || [];
    } catch (err) {
      console.error("Failed to fetch iPerf suggestions:", err);
      return [];
    }
  }, []);

  /**
   * Fetches iPerf3 binary info (version, path).
   */
  const fetchIperfInfo = useCallback(async (): Promise<Record<string, unknown> | null> => {
    try {
      return await api.get<Record<string, unknown>>("/api/iperf/info");
    } catch (err) {
      console.error("Failed to fetch iPerf info:", err);
      return null;
    }
  }, []);

  return {
    // Speedtest state
    speedtestRunning,
    speedtestResult,
    speedtestError,

    // iPerf state
    iperfRunning,
    iperfResult,
    iperfError,

    // Speedtest operations
    runSpeedtest,
    fetchSpeedtestStatus,

    // iPerf client operations
    runIperfClient,
    fetchIperfClientStatus,

    // iPerf server operations
    startIperfServer,
    fetchIperfServerStatus,

    // Utility operations
    fetchIperfSuggestions,
    fetchIperfInfo,
  };
}
