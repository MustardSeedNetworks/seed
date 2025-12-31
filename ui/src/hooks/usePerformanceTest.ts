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

import { useCallback, useState } from "react";
import { api } from "../lib/api";
import { LogComponents, logger } from "../lib/logger";

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

      const result = await api.post<SpeedtestResult>("/api/sap/speedtest");
      setSpeedtestResult(result);
      return result;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Speedtest failed";
      setSpeedtestError(message);
      logger.error(LogComponents.Speedtest, "Speedtest failed", err, {
        endpoint: "/api/sap/speedtest",
      });
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
      return await api.get<SpeedtestStatus>("/api/sap/speedtest/status");
    } catch (err) {
      logger.error(LogComponents.Speedtest, "Failed to fetch speedtest status", err, {
        endpoint: "/api/sap/speedtest/status",
      });
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

        const result = await api.post<IperfResult>("/api/sap/iperf/client", config);
        setIperfResult(result);
        return result;
      } catch (err) {
        const message = err instanceof Error ? err.message : "iPerf test failed";
        setIperfError(message);
        logger.error(LogComponents.Iperf, "iPerf client test failed", err, {
          endpoint: "/api/sap/iperf/client",
          server: config.server,
          port: config.port,
          protocol: config.protocol,
          duration: config.duration,
          parallel: config.parallel,
          reverse: config.reverse,
        });
        return null;
      } finally {
        setIperfRunning(false);
      }
    },
    [],
  );

  /**
   * Fetches the current iPerf client status.
   */
  const fetchIperfClientStatus = useCallback(async (): Promise<IperfClientStatus | null> => {
    try {
      return await api.get<IperfClientStatus>("/api/sap/iperf/client/status");
    } catch (err) {
      logger.error(LogComponents.Iperf, "Failed to fetch iPerf client status", err, {
        endpoint: "/api/sap/iperf/client/status",
      });
      return null;
    }
  }, []);

  /**
   * Starts the iPerf3 server mode.
   */
  const startIperfServer = useCallback(async (port?: number): Promise<boolean> => {
    try {
      await api.post("/api/sap/iperf/server", { port });
      return true;
    } catch (err) {
      logger.error(LogComponents.Iperf, "Failed to start iPerf server", err, {
        endpoint: "/api/sap/iperf/server",
        port,
      });
      return false;
    }
  }, []);

  /**
   * Fetches the iPerf server status.
   */
  const fetchIperfServerStatus = useCallback(async (): Promise<IperfServerStatus | null> => {
    try {
      return await api.get<IperfServerStatus>("/api/sap/iperf/server/status");
    } catch (err) {
      logger.error(LogComponents.Iperf, "Failed to fetch iPerf server status", err, {
        endpoint: "/api/sap/iperf/server/status",
      });
      return null;
    }
  }, []);

  /**
   * Fetches iPerf server suggestions based on network conditions.
   */
  const fetchIperfSuggestions = useCallback(async (): Promise<IperfServerSuggestion[]> => {
    try {
      const data = await api.get<{ suggestions: IperfServerSuggestion[] }>(
        "/api/sap/iperf/suggestions",
      );
      return data.suggestions || [];
    } catch (err) {
      logger.error(LogComponents.Iperf, "Failed to fetch iPerf suggestions", err, {
        endpoint: "/api/sap/iperf/suggestions",
      });
      return [];
    }
  }, []);

  /**
   * Fetches iPerf3 binary info (version, path).
   */
  const fetchIperfInfo = useCallback(async (): Promise<Record<string, unknown> | null> => {
    try {
      return await api.get<Record<string, unknown>>("/api/sap/iperf/info");
    } catch (err) {
      logger.error(LogComponents.Iperf, "Failed to fetch iPerf info", err, {
        endpoint: "/api/sap/iperf/info",
      });
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
